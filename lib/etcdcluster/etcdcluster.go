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
	// Members returns the cluster members by peer URL.
	MemberURLs() []string
	// RemoveMember removes a member of the cluster by its peer URL.
	RemoveMember(peerURL string) error
	// AddMember adds a new member to the cluster by its peer URL.
	AddMember(peerURL string) error
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

func (e *cluster) MemberURLs() []string {
	membersAPI := client.NewMembersAPI(e.client)
	members, err := membersAPI.List(context.Background())

	if err != nil {
		log.Info("Detected cluster errors, this is normal when bootstrapping a new cluster: ", err)
	}

	var memberURLs []string
	for _, member := range members {
		assertSinglePeerURL(member)
		memberURLs = append(memberURLs, member.PeerURLs[0])
	}

	return memberURLs
}

func assertSinglePeerURL(member client.Member) {
	if len(member.PeerURLs) != 1 {
		log.Fatalf("Expected a single peer URL, but found %v for %s", member.PeerURLs, member.ID)
	}
}

func (e *cluster) RemoveMember(peerURL string) error {
	membersAPI := client.NewMembersAPI(e.client)
	members := ensureGetMembers(membersAPI)

	for _, member := range members {
		assertSinglePeerURL(member)
		if member.PeerURLs[0] == peerURL {
			return membersAPI.Remove(context.Background(), member.ID)
		}
	}

	log.Infof("%s has already been removed", peerURL)
	return nil
}

func ensureGetMembers(api client.MembersAPI) []client.Member {
	members, err := api.List(context.Background())

	if err != nil {
		log.Fatal("Unexpected error when querying members API on existing cluster: ", err)
	}

	return members
}

func (e *cluster) AddMember(peerURL string) error {
	membersAPI := client.NewMembersAPI(e.client)
	_, err := membersAPI.Add(context.Background(), peerURL)
	return err
}
