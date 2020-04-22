package bootstrap

import (
	"fmt"
	"io/ioutil"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/sky-uk/etcd-bootstrap/cloud"
	"github.com/sky-uk/etcd-bootstrap/etcd"
)

// Bootstrapper bootstraps an etcd process by generating a set of Etcd flags for discovery.
type Bootstrapper struct {
	cloudAPI        CloudAPI
	etcdAPI         EtcdAPI
	protocol        string
	additionalFlags []string
}

type clusterState string

const (
	// If the node already exists in the cluster state (i.e. other nodes know about it) then the cluster state should
	// retain it's original new status
	newCluster clusterState = "new"
	// If the node is joining a cluster that doesn't know about the node then the cluster state should be existing
	existingCluster clusterState = "existing"
)

// CloudAPI returns instance information for the etcd cluster from cloud APIs.
type CloudAPI interface {
	// GetInstances returns all the non-terminated instances that will be part of the etcd cluster.
	GetInstances() ([]cloud.Instance, error)
	// GetLocalInstance returns the local machine instance.
	GetLocalInstance() (cloud.Instance, error)
	// GetLocalIP returns the IP of a local interface to listen on. This should be an externally accessible IP,
	// and may be the same as the endpoint returned in GetLocalInstance but this is not required.
	GetLocalIP() (string, error)
}

// EtcdAPI returns information from the etcd cluster API.
type EtcdAPI interface {
	Members() ([]etcd.Member, error)
	AddMemberByPeerURL(string) error
	RemoveMemberByName(string) error
}

// Option for configuring the bootstrapper.
type Option func(*Bootstrapper) error

// WithTLS enables TLS for peer and client endpoints.
func WithTLS(serverCA, serverCert, serverKey, peerCA, peerCert, peerKey string) Option {
	return func(b *Bootstrapper) error {
		vals := []struct {
			name, val string
		}{
			{"serverCA", serverCA},
			{"serverCert", serverCert},
			{"serverKey", serverKey},
			{"peerCA", peerCA},
			{"peerCert", peerCert},
			{"peerKey", peerKey},
		}
		for _, val := range vals {
			if val.val == "" {
				return fmt.Errorf("%s must be provided, but was empty", val.name)
			}
		}

		flags := []string{
			"ETCD_CLIENT_CERT_AUTH=true",
			"ETCD_TRUSTED_CA_FILE=" + serverCA,
			"ETCD_CERT_FILE=" + serverCert,
			"ETCD_KEY_FILE=" + serverKey,
			"ETCD_PEER_CLIENT_CERT_AUTH=true",
			"ETCD_PEER_TRUSTED_CA_FILE=" + peerCA,
			"ETCD_PEER_CERT_FILE=" + peerCert,
			"ETCD_PEER_KEY_FILE=" + peerKey,
		}
		b.additionalFlags = append(b.additionalFlags, flags...)
		b.protocol = "https"
		return nil
	}
}

// New creates a new bootstrapper.
func New(cloudAPI CloudAPI, etcdAPI EtcdAPI, opts ...Option) (*Bootstrapper, error) {
	bootstrapper := &Bootstrapper{
		cloudAPI: cloudAPI,
		etcdAPI:  etcdAPI,
		protocol: "http",
	}
	for _, opt := range opts {
		if err := opt(bootstrapper); err != nil {
			return nil, err
		}
	}
	return bootstrapper, nil
}

// GenerateEtcdFlagsFile writes etcd flag data to a file.
// It's intended to be sourced in startup scripts.
func (b *Bootstrapper) GenerateEtcdFlagsFile(outputFilename string) error {
	log.Infof("Writing environment variables to %s", outputFilename)
	etcdFlags, err := b.GenerateEtcdFlags()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(outputFilename, []byte(etcdFlags), 0644)
}

// GenerateEtcdFlags returns a string containing the generated etcd flags.
func (b *Bootstrapper) GenerateEtcdFlags() (string, error) {
	log.Infof("Generating etcd cluster flags")

	clusterExists, err := b.clusterExists()
	if err != nil {
		return "", err
	}
	if !clusterExists {
		log.Info("No cluster found - treating as an initial node in the new cluster")
		return b.createEtcdConfigForNewCluster()
	}

	nodeExistsInCluster, err := b.nodeExistsInCluster()
	if err != nil {
		return "", err
	}
	if nodeExistsInCluster {
		// We treat it as a new cluster in case the cluster hasn't fully bootstrapped yet.
		// etcd will ignore the INITIAL_* flags otherwise, so this should be safe.
		log.Info("Node already exists in cluster - treating as an existing node in a new cluster")
		return b.createEtcdConfigForNewCluster()
	}

	log.Info("Node does not exist yet in cluster - joining as a new node")
	if err := b.reconcileMembers(); err != nil {
		return "", err
	}
	return b.createEtcdConfigForExistingCluster()
}

func (b *Bootstrapper) clusterExists() (bool, error) {
	m, err := b.etcdAPI.Members()
	if err != nil {
		return false, err
	}
	return len(m) > 0, nil
}

func (b *Bootstrapper) nodeExistsInCluster() (bool, error) {
	members, err := b.etcdAPI.Members()
	if err != nil {
		return false, err
	}
	localInstance, err := b.cloudAPI.GetLocalInstance()
	if err != nil {
		return false, err
	}
	for _, member := range members {
		if member.Name == localInstance.Name {
			return true, nil
		}
	}
	return false, nil
}

func (b *Bootstrapper) createEtcdConfigForNewCluster() (string, error) {
	instances, err := b.cloudAPI.GetInstances()
	if err != nil {
		return "", err
	}
	var initialClusterURLs []string
	for _, instance := range instances {
		initialClusterURLs = append(initialClusterURLs, b.peerURL(instance.Endpoint))
	}
	return b.createEtcdConfig(newCluster, initialClusterURLs)
}

func (b *Bootstrapper) createEtcdConfigForExistingCluster() (string, error) {
	members, err := b.etcdAPI.Members()
	if err != nil {
		return "", err
	}
	var initialClusterURLs []string
	for _, member := range members {
		if member.Name != "" {
			// Don't include joining instances whose peerURL will be added but the name will be blank.
			initialClusterURLs = append(initialClusterURLs, member.PeerURL)
		}
	}
	return b.createEtcdConfig(existingCluster, initialClusterURLs)
}

func (b *Bootstrapper) createEtcdConfig(state clusterState, initialPeerURLs []string) (string, error) {
	envs := []string{"ETCD_INITIAL_CLUSTER_STATE=" + string(state)}

	initialClusterValue, err := b.initialClusterFlagValue(initialPeerURLs)
	if err != nil {
		return "", err
	}
	envs = append(envs, fmt.Sprintf("ETCD_INITIAL_CLUSTER=%s", initialClusterValue))

	local, err := b.cloudAPI.GetLocalInstance()
	if err != nil {
		return "", err
	}
	envs = append(envs, fmt.Sprintf("ETCD_NAME=%s", local.Name))

	// Advertise using the local endpoint which may be a domain name.
	envs = append(envs, fmt.Sprintf("ETCD_INITIAL_ADVERTISE_PEER_URLS=%s", b.peerURL(local.Endpoint)))
	envs = append(envs, fmt.Sprintf("ETCD_ADVERTISE_CLIENT_URLS=%s", b.clientURL(local.Endpoint)))

	// Listen on the local IP.
	localIP, err := b.cloudAPI.GetLocalIP()
	if err != nil {
		return "", err
	}
	envs = append(envs, fmt.Sprintf("ETCD_LISTEN_PEER_URLS=%s", b.peerURL(localIP)))
	envs = append(envs, fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=%s,%s", b.clientURL(localIP), b.clientURL("127.0.0.1")))

	for _, flag := range b.additionalFlags {
		envs = append(envs, flag)
	}
	return strings.Join(envs, "\n") + "\n", nil
}

func (b *Bootstrapper) initialClusterFlagValue(initialPeerURLs []string) (string, error) {
	instances, err := b.cloudAPI.GetInstances()
	if err != nil {
		return "", err
	}
	var initialCluster []string
	for _, instance := range instances {
		// Only include instances that are part of the initial/existing cluster.
		instancePeerURL := b.peerURL(instance.Endpoint)
		if contains(initialPeerURLs, instancePeerURL) {
			initialCluster = append(initialCluster, fmt.Sprintf("%s=%s", instance.Name, instancePeerURL))
		}
	}

	return strings.Join(initialCluster, ","), nil
}

func (b *Bootstrapper) peerURL(host string) string {
	return fmt.Sprintf("%s://%s:2380", b.protocol, host)
}

func (b *Bootstrapper) clientURL(host string) string {
	return fmt.Sprintf("%s://%s:2379", b.protocol, host)
}

func contains(strings []string, value string) bool {
	for _, s := range strings {
		if value == s {
			return true
		}
	}
	return false
}
