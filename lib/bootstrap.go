package bootstrap

import (
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/sky-uk/etcd-bootstrap/lib/asg"
	"github.com/sky-uk/etcd-bootstrap/lib/etcdcluster"
)

// Bootstrapper bootstraps etcd config.
type Bootstrapper interface {
	// Bootstrap creates a file with etcd flags based on the ASG status.
	// It's intended to be sourced in startup scripts.
	Bootstrap() string
}

type bootstrapper struct {
	asg     asg.ASG
	cluster etcdcluster.Cluster
}

// Default creates a default bootstrapper wired to the local ASG.
func Default() Bootstrapper {
	asg := asg.New()
	return New(asg, etcdcluster.New(asg))
}

// New creates a new bootstrapper.
func New(localASG asg.ASG, cluster etcdcluster.Cluster) Bootstrapper {
	return &bootstrapper{localASG, cluster}
}

func (b *bootstrapper) Bootstrap() string {
	log.Info("Bootstrapping etcd flags")
	if b.newCluster() {
		log.Info("No cluster found, bootstrapping a new cluster")
		return b.bootstrapNew()
	}
	log.Info("Detected existing cluster")
	return b.bootstrapExisting()
}

func (b *bootstrapper) newCluster() bool {
	m := b.cluster.MemberURLs()
	return len(m) == 0
}

func (b *bootstrapper) bootstrapNew() string {
	availURLs := b.getInstancePeerURLs()
	return b.createEtcdConfig(newCluster, availURLs)
}

func (b *bootstrapper) getInstancePeerURLs() []string {
	var peerURLs []string
	for _, i := range b.asg.GetInstances() {
		peerURLs = append(peerURLs, peerURL(i.PrivateIP))
	}
	return peerURLs
}

func (b *bootstrapper) bootstrapExisting() string {
	availURLs := b.reconcileMembers()
	return b.createEtcdConfig(existingCluster, availURLs)
}

func (b *bootstrapper) reconcileMembers() []string {
	memberURLs := b.cluster.MemberURLs()
	instanceURLs := b.getInstancePeerURLs()

	for _, memberURL := range memberURLs {
		if !contains(instanceURLs, memberURL) {
			log.Infof("Removing %s from etcd member list, not found in ASG", memberURL)
			if err := b.cluster.RemoveMember(memberURL); err != nil {
				log.Warn("Unable to remove old member. This may be due to temporary lack of quorum,"+
					" will ignore: ", err)
			}
		}
	}

	localInstanceURL := peerURL(b.asg.GetLocalInstance().PrivateIP)
	if !contains(memberURLs, localInstanceURL) {
		log.Infof("Adding local instance %s to the etcd member list", localInstanceURL)
		if err := b.cluster.AddMember(localInstanceURL); err != nil {
			log.Fatalf("Unexpected error when adding new member URL %s: %v", localInstanceURL, err)
		}
		memberURLs = append(memberURLs, localInstanceURL)
	}

	return memberURLs
}

type clusterState string

const (
	newCluster      clusterState = "new"
	existingCluster clusterState = "existing"
)

func (b *bootstrapper) createEtcdConfig(state clusterState, availablePeerURLs []string) string {
	var envs []string

	envs = append(envs, "ETCD_INITIAL_CLUSTER_STATE="+string(state))
	envs = append(envs, fmt.Sprintf("ETCD_INITIAL_CLUSTER=%s", b.constructInitialCluster(availablePeerURLs)))

	local := b.asg.GetLocalInstance()
	envs = append(envs, fmt.Sprintf("ETCD_NAME=%s", local.InstanceID))
	envs = append(envs, fmt.Sprintf("ETCD_INITIAL_ADVERTISE_PEER_URLS=%s", peerURL(local.PrivateIP)))
	envs = append(envs, fmt.Sprintf("ETCD_LISTEN_PEER_URLS=%s", peerURL(local.PrivateIP)))
	envs = append(envs, fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=%s,%s", clientURL(local.PrivateIP), clientURL("127.0.0.1")))
	envs = append(envs, fmt.Sprintf("ETCD_ADVERTISE_CLIENT_URLS=%s", clientURL(local.PrivateIP)))

	return strings.Join(envs, "\n") + "\n"
}

func (b *bootstrapper) constructInitialCluster(availablePeerURLs []string) string {
	instances := b.asg.GetInstances()

	var initialCluster []string
	for _, instance := range instances {
		instancePeerURL := peerURL(instance.PrivateIP)
		if contains(availablePeerURLs, instancePeerURL) {
			initialCluster = append(initialCluster,
				fmt.Sprintf("%s=%s", instance.InstanceID, instancePeerURL))
		}
	}

	return strings.Join(initialCluster, ",")
}

func peerURL(ip string) string {
	return fmt.Sprintf("http://%s:2380", ip)
}

func clientURL(ip string) string {
	return fmt.Sprintf("http://%s:2379", ip)
}

func contains(strings []string, value string) bool {
	for _, s := range strings {
		if value == s {
			return true
		}
	}
	return false
}
