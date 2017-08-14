// This file contains test stubs and mocks.
package bootstrap

import (
	"errors"

	"github.com/sky-uk/etcd-bootstrap/lib/cloud"
	"github.com/sky-uk/etcd-bootstrap/lib/etcdcluster"
	"github.com/stretchr/testify/mock"
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

type mockR53 struct {
	mock.Mock
}

func (m *mockR53) UpdateARecords(zoneID, name string, values []string) error {
	args := m.Called(zoneID, name, values)
	return args.Error(0)
}

type emptyR53 struct{}

func (e *emptyR53) UpdateARecords(string, string, []string) error {
	return errors.New("shouldn't call me")
}
