package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/sky-uk/etcd-bootstrap/lib/cloud"
	"github.com/stretchr/testify/mock"
)

const (
	defaultR53ZoneID         = "test-zone-id"
	defaultLBTargetGroupName = "test-lb-target-group"
)

type mockR53 struct {
	mock.Mock
}

func (m *mockR53) UpdateARecords(zoneID, name string, values []string) error {
	args := m.Called(zoneID, name, values)
	return args.Error(0)
}

type mockLBTargetGroup struct {
	mock.Mock
}

func (m *mockLBTargetGroup) UpdateTargetGroup(targetGroupName string, instances []string) error {
	args := m.Called(targetGroupName, instances)
	return args.Error(0)
}

func TestAddsInstancesToRoute53Entry(t *testing.T) {
	// given
	etcdClusterName := "my-etcd-name"
	mockedR53, _, testAWS := setupTest()

	// when
	mockedR53.
		On("UpdateARecords", defaultR53ZoneID, etcdClusterName, []string{"10.50.99.1", "10.50.199.1", "10.50.155.1"}).
		Return(nil)
	testAWS.UpdateDNS(etcdClusterName)

	// then
	mockedR53.AssertExpectations(t)
}

func TestAddsInstancesToLBTargetGroup(t *testing.T) {
	// given
	_, mockedLBTargetGroup, testAWS := setupTest()

	// when
	mockedLBTargetGroup.
		On("UpdateTargetGroup", defaultLBTargetGroupName, []string{"e1", "e2", "e3"}).
		Return(nil)
	testAWS.UpdateLB()

	// then
	mockedLBTargetGroup.AssertExpectations(t)
}

func setupTest() (*mockR53, *mockLBTargetGroup, *awsMembers) {
	mockedR53 := new(mockR53)
	mockedLBTargetGroup := new(mockLBTargetGroup)
	return mockedR53, mockedLBTargetGroup, &awsMembers{
		identityDocument: ec2metadata.EC2InstanceIdentityDocument{},
		instances: []cloud.Instance{
			{InstanceID: "e1", PrivateIP: "10.50.99.1"},
			{InstanceID: "e2", PrivateIP: "10.50.199.1"},
			{InstanceID: "e3", PrivateIP: "10.50.155.1"},
		},
		r53:               mockedR53,
		r53ZoneID:         defaultR53ZoneID,
		lbTargetGroup:     mockedLBTargetGroup,
		lbTargetGroupName: defaultLBTargetGroupName,
	}
}
