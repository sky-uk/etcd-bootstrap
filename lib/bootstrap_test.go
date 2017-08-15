// This file contains test stubs and mocks.
package bootstrap

import (
	"errors"

	"github.com/sky-uk/etcd-bootstrap/lib/cloud"
	"github.com/sky-uk/etcd-bootstrap/lib/etcdcluster"
)

type testASG struct {
	instances []cloud.Instance
	local     cloud.Instance
}

func (a *testASG) GetInstances() []cloud.Instance {
	return a.instances
}

func (a *testASG) GetLocalInstance() cloud.Instance {
	return a.local
}

func (a *testASG) UpdateDNS(name string) error {
	return nil
}

type testCluster struct {
	members       []etcdcluster.Member
	removedMember []string
	addedMember   []string
}

func (e *testCluster) Members() ([]etcdcluster.Member, error) {
	return e.members, nil
}

func (e *testCluster) RemoveMember(peerURL string) error {
	var idx int
	for i, m := range e.members {
		if m.PeerURL == peerURL {
			idx = i
		}
	}
	e.members = append(e.members[:idx], e.members[idx+1:]...)

	e.removedMember = append(e.removedMember, peerURL)
	return errors.New("Test that remove member error is ignored")
}

func (e *testCluster) AddMember(peerURL string) error {
	e.members = append(e.members, etcdcluster.Member{PeerURL: peerURL})

	e.addedMember = append(e.addedMember, peerURL)
	return nil
}
