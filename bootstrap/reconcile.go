package bootstrap

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

func (b *Bootstrapper) reconcileMembers() error {
	if err := b.removeOldEtcdMembers(); err != nil {
		return err
	}
	return b.addLocalInstanceToEtcd()
}

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
			if member.Name == "" && contains(instanceURLs, member.PeerURL) {
				// This member is still initialising, don't remove it.
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
		// Don't re-add if the local instance's peerURL has already been added.
		!contains(memberURLs, b.peerURL(localInstance.Endpoint)) {

		log.Infof("Adding local instance %v to the etcd member list", localInstance)
		localInstanceURL := b.peerURL(localInstance.Endpoint)
		if err := b.etcdAPI.AddMemberByPeerURL(localInstanceURL); err != nil {
			return fmt.Errorf("unexpected error when adding new member URL %s: %v", localInstanceURL, err)
		}
	}

	return nil
}
