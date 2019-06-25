package etcd

import (
	"fmt"

	"github.com/coreos/etcd/client"
	log "github.com/sirupsen/logrus"
	"github.com/sky-uk/etcd-bootstrap/provider"
	"golang.org/x/net/context"
)

// Cluster represents an etcd cluster.
type Cluster interface {
	// Members returns the cluster members.
	Members() ([]Member, error)
	// RemoveMember removes a member of the cluster by its peer URL.
	RemoveMember(peerURL string) error
	// AddMember adds a new member to the cluster by its peer URL.
	AddMember(peerURL string) error
}

type cluster struct {
	client client.Client
}

// Member represents a node in the etcd cluster.
type Member struct {
	Name    string
	PeerURL string
}

// New returns a cluster object representing the etcd cluster in the cloud provider.
func New(provider provider.Provider) (Cluster, error) {
	instances := provider.GetInstances()

	var endpoints []string
	for _, instance := range instances {
		endpoints = append(endpoints, fmt.Sprintf("http://%s:2379", instance.PrivateIP))
	}

	c, err := client.New(client.Config{Endpoints: endpoints})
	if err != nil {
		return nil, err
	}
	return &cluster{c}, nil
}

func (e *cluster) Members() ([]Member, error) {
	membersAPI := client.NewMembersAPI(e.client)
	etcdMembers, err := membersAPI.List(context.Background())
	if err != nil {
		log.Infof("Detected cluster errors, this is normal when bootstrapping a new cluster: %v", err)
	}

	var members []Member
	for _, etcdMember := range etcdMembers {
		if err := assertSinglePeerURL(etcdMember); err != nil {
			return nil, err
		}

		members = append(members, Member{
			Name:    etcdMember.Name,
			PeerURL: etcdMember.PeerURLs[0],
		})
	}

	return members, nil
}

func (e *cluster) AddMember(peerURL string) error {
	membersAPI := client.NewMembersAPI(e.client)
	_, err := membersAPI.Add(context.Background(), peerURL)
	return err
}

func (e *cluster) RemoveMember(peerURL string) error {
	membersAPI := client.NewMembersAPI(e.client)
	members, err := membersAPI.List(context.Background())
	if err != nil {
		return err
	}

	for _, member := range members {
		if err := assertSinglePeerURL(member); err != nil {
			return err
		}
		if member.PeerURLs[0] == peerURL {
			return membersAPI.Remove(context.Background(), member.ID)
		}
	}

	log.Infof("%s has already been removed", peerURL)
	return nil
}

func assertSinglePeerURL(member client.Member) error {
	if len(member.PeerURLs) != 1 {
		return fmt.Errorf("expected a single peer URL, but found %v for %s", member.PeerURLs, member.ID)
	}
	return nil
}
