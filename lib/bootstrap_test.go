package bootstrap

import (
	"testing"

	"strings"

	"errors"

	"github.com/sky-uk/etcd-bootstrap/lib/asg"
	"github.com/stretchr/testify/assert"
)

type testASG struct {
	instances []asg.Instance
	local     asg.Instance
}

func (a *testASG) GetInstances() []asg.Instance {
	return a.instances
}

func (a *testASG) GetLocalInstance() asg.Instance {
	return a.local
}

type testCluster struct {
	memberURLs   []string
	removeMember []string
	addMember    []string
}

func (e *testCluster) MemberURLs() []string {
	return e.memberURLs
}

func (e *testCluster) RemoveMember(peerURL string) error {
	e.removeMember = append(e.removeMember, peerURL)
	return errors.New("Test that remove member error is ignored")
}

func (e *testCluster) AddMember(peerURL string) error {
	e.addMember = append(e.addMember, peerURL)
	return nil
}

func TestCreateNewCluster(t *testing.T) {
	assert := assert.New(t)

	testASG := &testASG{}
	testASG.instances = []asg.Instance{
		{InstanceID: "e1", PrivateIP: "10.50.99.1"},
		{InstanceID: "e2", PrivateIP: "10.50.199.1"},
		{InstanceID: "e3", PrivateIP: "10.50.155.1"}}
	testASG.local = asg.Instance{InstanceID: "e2", PrivateIP: "10.50.199.1"}

	etcdCluster := &testCluster{}

	bootstrapper := New(testASG, etcdCluster)
	vars := strings.Split(bootstrapper.Bootstrap(), "\n")

	assert.Contains(vars, "ETCD_INITIAL_CLUSTER_STATE=new")
	assert.Contains(vars, "ETCD_INITIAL_CLUSTER=e1=http://10.50.99.1:2380,"+
		"e2=http://10.50.199.1:2380,e3=http://10.50.155.1:2380")
	assert.Contains(vars, "ETCD_INITIAL_ADVERTISE_PEER_URLS=http://10.50.199.1:2380")
	assert.Contains(vars, "ETCD_NAME=e2")
	assert.Contains(vars, "ETCD_LISTEN_PEER_URLS=http://10.50.199.1:2380")
	assert.Contains(vars, "ETCD_LISTEN_CLIENT_URLS=http://10.50.199.1:2379,http://127.0.0.1:2379")
	assert.Contains(vars, "ETCD_ADVERTISE_CLIENT_URLS=http://10.50.199.1:2379")
}

func TestExistingCluster(t *testing.T) {
	assert := assert.New(t)

	testASG := &testASG{}
	testASG.instances = []asg.Instance{
		{InstanceID: "e1", PrivateIP: "10.50.99.1"},
		{InstanceID: "e2", PrivateIP: "10.50.199.1"},
		{InstanceID: "e3", PrivateIP: "10.50.155.1"}}
	testASG.local = asg.Instance{InstanceID: "e2", PrivateIP: "10.50.199.1"}

	etcdCluster := &testCluster{}
	etcdCluster.memberURLs = []string{
		"http://10.50.99.1:2380",
		"http://10.50.199.1:2380",
		"http://10.50.155.1:2380"}

	bootstrapper := New(testASG, etcdCluster)
	vars := strings.Split(bootstrapper.Bootstrap(), "\n")

	assert.Contains(vars, "ETCD_INITIAL_CLUSTER_STATE=new")
	assert.Contains(vars, "ETCD_INITIAL_CLUSTER=e1=http://10.50.99.1:2380,"+
		"e2=http://10.50.199.1:2380,e3=http://10.50.155.1:2380")
	assert.Contains(vars, "ETCD_INITIAL_ADVERTISE_PEER_URLS=http://10.50.199.1:2380")
	assert.Contains(vars, "ETCD_NAME=e2")
	assert.Contains(vars, "ETCD_LISTEN_PEER_URLS=http://10.50.199.1:2380")
	assert.Contains(vars, "ETCD_LISTEN_CLIENT_URLS=http://10.50.199.1:2379,http://127.0.0.1:2379")
	assert.Contains(vars, "ETCD_ADVERTISE_CLIENT_URLS=http://10.50.199.1:2379")
}

func TestJoinAnExistingCluster(t *testing.T) {
	assert := assert.New(t)

	testASG := &testASG{}
	testASG.instances = []asg.Instance{
		{InstanceID: "e1", PrivateIP: "10.50.99.1"},
		{InstanceID: "e2", PrivateIP: "10.50.199.1"},
		{InstanceID: "e3", PrivateIP: "10.50.155.1"}}
	testASG.local = asg.Instance{InstanceID: "e2", PrivateIP: "10.50.199.1"}

	etcdCluster := &testCluster{}
	etcdCluster.memberURLs = []string{
		"http://10.50.99.1:2380",
		"http://10.50.65.2:2380",
		"http://10.50.44.44:2380",
	}

	bootstrapper := New(testASG, etcdCluster)
	vars := strings.Split(bootstrapper.Bootstrap(), "\n")

	assert.Contains(etcdCluster.addMember, "http://10.50.199.1:2380")
	assert.NotContains(etcdCluster.addMember, "http://10.50.155.1:2380",
		"Should only add itself, to prevent stuck quorum errors")
	assert.Contains(etcdCluster.removeMember, "http://10.50.65.2:2380")
	assert.Contains(etcdCluster.removeMember, "http://10.50.44.44:2380")
	assert.Len(etcdCluster.addMember, 1)
	assert.Len(etcdCluster.removeMember, 2)

	assert.Contains(vars, "ETCD_INITIAL_CLUSTER=e1=http://10.50.99.1:2380,e2=http://10.50.199.1:2380",
		"Initial cluster should only include the new local node and existing node, otherwise we'll get a "+
			"'member count unequal' error at etcd startup.")
}
