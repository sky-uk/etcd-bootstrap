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

	if b.clusterDoesNotExist() {
		log.Info("No cluster found - treating as an initial node in the new cluster")
		return b.bootstrapNewCluster()
	}

	if b.nodeExistsInCluster() {
		// We treat it as a new cluster in case the cluster hasn't fully bootstrapped yet.
		// etcd will ignore the INITIAL_* flags otherwise, so this should be safe.
		log.Info("Node peer URL already exists - treating as an existing node in a new cluster")
		return b.bootstrapNewCluster()
	}

	log.Info("Node does not exist yet in cluster - joining as a new node")
	return b.bootstrapNewNode()
}

func (b *bootstrapper) clusterDoesNotExist() bool {
	m := b.cluster.Members()
	return len(m) == 0
}

func (b *bootstrapper) nodeExistsInCluster() bool {
	members := b.cluster.Members()
	localInstanceURL := peerURL(b.asg.GetLocalInstance().PrivateIP)

	for _, member := range members {
		if member.PeerURL == localInstanceURL && len(member.Name) > 0 {
			return true
		}
	}

	return false
}

func (b *bootstrapper) bootstrapNewCluster() string {
	instanceURLs := b.getInstancePeerURLs()
	return b.createEtcdConfig(newCluster, instanceURLs)
}

func (b *bootstrapper) getInstancePeerURLs() []string {
	var peerURLs []string
	for _, i := range b.asg.GetInstances() {
		peerURLs = append(peerURLs, peerURL(i.PrivateIP))
	}
	return peerURLs
}

func (b *bootstrapper) bootstrapNewNode() string {
	b.reconcileMembers()
	clusterURLs := b.etcdMemberPeerURLs()
	return b.createEtcdConfig(existingCluster, clusterURLs)
}

func (b *bootstrapper) reconcileMembers() {
	b.removeOldEtcdMembers()
	b.addLocalInstanceToEtcd()
}

func (b *bootstrapper) removeOldEtcdMembers() {
	memberURLs := b.etcdMemberPeerURLs()
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
}

func (b *bootstrapper) addLocalInstanceToEtcd() {
	memberURLs := b.etcdMemberPeerURLs()
	localInstanceURL := peerURL(b.asg.GetLocalInstance().PrivateIP)

	if !contains(memberURLs, localInstanceURL) {
		log.Infof("Adding local instance %s to the etcd member list", localInstanceURL)
		if err := b.cluster.AddMember(localInstanceURL); err != nil {
			log.Fatalf("Unexpected error when adding new member URL %s: %v", localInstanceURL, err)
		}
	}
}

func (b *bootstrapper) etcdMemberPeerURLs() []string {
	members := b.cluster.Members()
	var memberURLs []string
	for _, member := range members {
		memberURLs = append(memberURLs, member.PeerURL)
	}
	return memberURLs
}

type clusterState string

const (
	// If node already exists, cluster state should be set to new
	newCluster clusterState = "new"
	// If node is joining an existing cluster, cluster state should be set to existing
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
