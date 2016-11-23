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
	members := b.cluster.Members()
	return len(members) == 0
}

func (b *bootstrapper) bootstrapNew() string {
	return b.createEtcdConfig(newCluster)
}

func (b *bootstrapper) bootstrapExisting() string {
	log.Warn("Joining an existing cluster is not implemented yet - will treat this as an existing node")
	return b.createEtcdConfig(existingCluster)
}

type clusterState string

const (
	newCluster      clusterState = "new"
	existingCluster clusterState = "existing"
)

func (b *bootstrapper) createEtcdConfig(state clusterState) string {
	var envs []string

	envs = append(envs, "ETCD_INITIAL_CLUSTER_STATE="+string(state))

	instances := b.asg.GetInstances()
	var initialCluster []string
	for _, instance := range instances {
		initialCluster = append(initialCluster,
			fmt.Sprintf("%s=http://%s:2380", instance.InstanceID, instance.PrivateIP))
	}
	envs = append(envs, fmt.Sprintf("ETCD_INITIAL_CLUSTER=%s", strings.Join(initialCluster, ",")))

	local := b.asg.GetLocalInstance()
	envs = append(envs, fmt.Sprintf("ETCD_NAME=%s", local.InstanceID))
	envs = append(envs, fmt.Sprintf("ETCD_INITIAL_ADVERTISE_PEER_URLS=http://%s:2380", local.PrivateIP))
	envs = append(envs, fmt.Sprintf("ETCD_LISTEN_PEER_URLS=http://%s:2380", local.PrivateIP))
	envs = append(envs, fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=http://%s:2379,http://127.0.0.1:2379", local.PrivateIP))
	envs = append(envs, fmt.Sprintf("ETCD_ADVERTISE_CLIENT_URLS=http://%s:2379", local.PrivateIP))

	return strings.Join(envs, "\n") + "\n"
}
