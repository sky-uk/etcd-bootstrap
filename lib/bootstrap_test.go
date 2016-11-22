package bootstrap

import (
	"testing"

	"strings"

	"github.com/sky-uk/aws-etcd/lib/asg"
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
	members []string
}

func (e *testCluster) Members() []string {
	return e.members
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
