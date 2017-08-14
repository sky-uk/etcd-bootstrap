package bootstrap

import (
	"testing"

	"github.com/sky-uk/etcd-bootstrap/lib/cloud"
)

func TestAddsInstancesToRoute53Entry(t *testing.T) {
	// given
	testASG := &testASG{}
	testASG.instances = []cloud.Instance{
		{InstanceID: "e1", PrivateIP: "10.50.99.1"},
		{InstanceID: "e2", PrivateIP: "10.50.199.1"},
		{InstanceID: "e3", PrivateIP: "10.50.155.1"}}
	testASG.local = cloud.Instance{InstanceID: "e2", PrivateIP: "10.50.199.1"}
	etcdCluster := &testCluster{}
	mockedR53 := new(mockR53)
	bootstrapper := New(testASG, etcdCluster, mockedR53)

	// when
	mockedR53.
		On("UpdateARecords", "my-zone-id", "my-etcd-name", []string{"10.50.99.1", "10.50.199.1", "10.50.155.1"}).
		Return(nil)
	bootstrapper.BootstrapRoute53("my-zone-id", "my-etcd-name")

	// then
	mockedR53.AssertExpectations(t)
}
