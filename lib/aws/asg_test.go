package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/sky-uk/etcd-bootstrap/lib/cloud"
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

type mockR53 struct {
	mock.Mock
}

func (m *mockR53) UpdateARecords(zoneID, name string, values []string) error {
	args := m.Called(zoneID, name, values)
	return args.Error(0)
}

func TestAddsInstancesToRoute53Entry(t *testing.T) {
	// given
	mockedR53 := new(mockR53)
	testASG := &localASG{
		ec2metadata.EC2InstanceIdentityDocument{},
		[]cloud.Instance{
			{InstanceID: "e1", PrivateIP: "10.50.99.1"},
			{InstanceID: "e2", PrivateIP: "10.50.199.1"},
			{InstanceID: "e3", PrivateIP: "10.50.155.1"}},
		mockedR53,
		"test-zone-id",
	}

	// when
	mockedR53.
		On("UpdateARecords", "test-zone-id", "my-etcd-name", []string{"10.50.99.1", "10.50.199.1", "10.50.155.1"}).
		Return(nil)
	testASG.UpdateDNS("my-etcd-name")

	// then
	mockedR53.AssertExpectations(t)
}
