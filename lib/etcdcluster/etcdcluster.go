package etcdcluster

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/etcd/client"
	"github.com/sky-uk/etcd-bootstrap/lib/asg"
	"golang.org/x/net/context"
)

// Cluster represents an etcd cluster.
type Cluster interface {
	// Members returns the members by name of the cluster.
	Members() []string
}

type cluster struct {
	client client.Client
}

// New returns a cluster object representing the etcd cluster in the local auto scaling group.
func New(asg asg.ASG) Cluster {
	instances := asg.GetInstances()

	var endpoints []string
	for _, instance := range instances {
		endpoints = append(endpoints, fmt.Sprintf("http://%s:2379", instance.PrivateIP))
	}

	c, err := client.New(client.Config{Endpoints: endpoints})
	if err != nil {
		log.Fatal("Unable to create etcd client: ", err)
	}
	return &cluster{c}
}

func (e *cluster) Members() []string {
	membersAPI := client.NewMembersAPI(e.client)
	members, err := membersAPI.List(context.Background())

	if err != nil {
		log.Info("Detected cluster errors, this is normal when bootstrapping a new cluster: ", err)
	}

	var memberNames []string
	for _, member := range members {
		memberNames = append(memberNames, member.Name)
	}

	return memberNames
}
