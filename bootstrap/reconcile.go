package bootstrap

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// reconcileMembers uses the etcd API to remove any non-existing members and add new ones that
// need to join. This is used primarily to handle node replacement.
func (b *Bootstrapper) reconcileMembers() error {
	if err := b.removeOldEtcdMembers(); err != nil {
		return err
	}
	return b.addLocalInstanceToEtcd()
}

// removeOldEtcdMembers removes any etcd members that are no longer part of the instances
// returned by the cloud API. We assume if it's not part of the cloud instances then the actual
// node VM has been removed.
func (b *Bootstrapper) removeOldEtcdMembers() error {
	members, err := b.etcdAPI.Members()
	if err != nil {
		return err
	}
	instances, err := b.cloudAPI.GetInstances()
	if err != nil {
		return err
	}
	var instanceNames []string
	var instanceURLs []string
	for _, instance := range instances {
		instanceNames = append(instanceNames, instance.Name)
		instanceURLs = append(instanceURLs, b.peerURL(instance.Endpoint))
	}

	for _, member := range members {
		if !contains(instanceNames, member.Name) {
			// The etcd member name doesn't exist in the list of cloud instances.
			if member.Name == "" && contains(instanceURLs, member.PeerURL) {
				// A special case is when member.Name == "". This means the member is still initialising, so don't remove it.
				// Unless the peerURL doesn't exist in the instance list either, in which case this node is no longer around.
				continue
			}
			log.Infof("Removing %s (%s) from etcd member list, not found in cloud provider", member.Name, member.PeerURL)
			if err := b.etcdAPI.RemoveMemberByName(member.Name); err != nil {
				log.Warnf("Unable to remove old member. This may be due to temporary lack of quorum,"+
					" will ignore: %v", err)
			}
		}
	}

	return nil
}

// addLocalInstanceToEtcd ensures the advertise peerURL is added to the existing cluster. This is required by
// etcd prior to a node joining a cluster, as described in https://etcd.io/docs/v3.4.0/op-guide/runtime-configuration/.
//
// After the peerURL is added, the member will show up with a blank name when listing members from the etcd API.
// Once it successfully joins the name will be set.
func (b *Bootstrapper) addLocalInstanceToEtcd() error {
	members, err := b.etcdAPI.Members()
	if err != nil {
		return err
	}
	var memberNames []string
	var memberURLs []string
	for _, member := range members {
		memberNames = append(memberNames, member.Name)
		memberURLs = append(memberURLs, member.PeerURL)
	}
	localInstance, err := b.cloudAPI.GetLocalInstance()
	if err != nil {
		return err
	}

	if !contains(memberNames, localInstance.Name) &&
		!contains(memberURLs, b.peerURL(localInstance.Endpoint)) {
		// Don't add if the member name already exists - the local instance is already part of the cluster.
		// Also don't re-add if the local instance's peerURL has already been added. This could happen
		// if the node crashed or restarted before it registered.

		log.Infof("Adding local instance %v to the etcd member list", localInstance)
		localInstanceURL := b.peerURL(localInstance.Endpoint)
		if err := b.etcdAPI.AddMemberByPeerURL(localInstanceURL); err != nil {
			return fmt.Errorf("unexpected error when adding new member URL %s: %v", localInstanceURL, err)
		}
	}

	return nil
}
