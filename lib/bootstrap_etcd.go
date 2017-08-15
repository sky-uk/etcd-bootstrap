package bootstrap

import (
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
)

func (b *bootstrapper) BootstrapEtcdFlags() (string, error) {
	log.Info("Bootstrapping etcd flags")

	clusterDoesNotExist, err := b.clusterDoesNotExist()
	if err != nil {
		return "", err
	}
	if clusterDoesNotExist {
		log.Info("No cluster found - treating as an initial node in the new cluster")
		return b.bootstrapNewCluster(), nil
	}

	nodeExistsInCluster, err := b.nodeExistsInCluster()
	if err != nil {
		return "", err
	}
	if nodeExistsInCluster {
		// We treat it as a new cluster in case the cluster hasn't fully bootstrapped yet.
		// etcd will ignore the INITIAL_* flags otherwise, so this should be safe.
		log.Info("Node peer URL already exists - treating as an existing node in a new cluster")
		return b.bootstrapNewCluster(), nil
	}

	log.Info("Node does not exist yet in cluster - joining as a new node")
	return b.bootstrapNewNode()
}

func (b *bootstrapper) clusterDoesNotExist() (bool, error) {
	m, err := b.cluster.Members()
	if err != nil {
		return false, err
	}
	return len(m) == 0, nil
}

func (b *bootstrapper) nodeExistsInCluster() (bool, error) {
	members, err := b.cluster.Members()
	if err != nil {
		return false, err
	}
	localInstanceURL := peerURL(b.cloud.GetLocalInstance().PrivateIP)

	for _, member := range members {
		if member.PeerURL == localInstanceURL && len(member.Name) > 0 {
			return true, nil
		}
	}

	return false, nil
}

func (b *bootstrapper) bootstrapNewCluster() string {
	instanceURLs := b.getInstancePeerURLs()
	return b.createEtcdConfig(newCluster, instanceURLs)
}

func (b *bootstrapper) getInstancePeerURLs() []string {
	instances := b.cloud.GetInstances()

	var peerURLs []string
	for _, i := range instances {
		peerURLs = append(peerURLs, peerURL(i.PrivateIP))
	}

	return peerURLs
}

func (b *bootstrapper) bootstrapNewNode() (string, error) {
	err := b.reconcileMembers()
	if err != nil {
		return "", err
	}
	clusterURLs, err := b.etcdMemberPeerURLs()
	if err != nil {
		return "", err
	}
	return b.createEtcdConfig(existingCluster, clusterURLs), nil
}

func (b *bootstrapper) reconcileMembers() error {
	if err := b.removeOldEtcdMembers(); err != nil {
		return err
	}
	if err := b.addLocalInstanceToEtcd(); err != nil {
		return err
	}
	return nil
}

func (b *bootstrapper) removeOldEtcdMembers() error {
	memberURLs, err := b.etcdMemberPeerURLs()
	if err != nil {
		return err
	}
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

	return nil
}

func (b *bootstrapper) addLocalInstanceToEtcd() error {
	memberURLs, err := b.etcdMemberPeerURLs()
	if err != nil {
		return err
	}
	localInstanceURL := peerURL(b.cloud.GetLocalInstance().PrivateIP)

	if !contains(memberURLs, localInstanceURL) {
		log.Infof("Adding local instance %s to the etcd member list", localInstanceURL)
		if err := b.cluster.AddMember(localInstanceURL); err != nil {
			return fmt.Errorf("unexpected error when adding new member URL %s: %v", localInstanceURL, err)
		}
	}

	return nil
}

func (b *bootstrapper) etcdMemberPeerURLs() ([]string, error) {
	members, err := b.cluster.Members()
	if err != nil {
		return nil, err
	}
	var memberURLs []string
	for _, member := range members {
		memberURLs = append(memberURLs, member.PeerURL)
	}
	return memberURLs, nil
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
	initialCluster := b.constructInitialCluster(availablePeerURLs)
	envs = append(envs, fmt.Sprintf("ETCD_INITIAL_CLUSTER=%s", initialCluster))

	local := b.cloud.GetLocalInstance()
	envs = append(envs, fmt.Sprintf("ETCD_NAME=%s", local.InstanceID))
	envs = append(envs, fmt.Sprintf("ETCD_INITIAL_ADVERTISE_PEER_URLS=%s", peerURL(local.PrivateIP)))
	envs = append(envs, fmt.Sprintf("ETCD_LISTEN_PEER_URLS=%s", peerURL(local.PrivateIP)))
	envs = append(envs, fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=%s,%s", clientURL(local.PrivateIP), clientURL("127.0.0.1")))
	envs = append(envs, fmt.Sprintf("ETCD_ADVERTISE_CLIENT_URLS=%s", clientURL(local.PrivateIP)))

	return strings.Join(envs, "\n") + "\n"
}

func (b *bootstrapper) constructInitialCluster(availablePeerURLs []string) string {
	var initialCluster []string
	for _, instance := range b.cloud.GetInstances() {
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
