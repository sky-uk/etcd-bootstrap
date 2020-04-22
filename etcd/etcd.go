package etcd

import (
	"fmt"

	"github.com/coreos/etcd/client"
	log "github.com/sirupsen/logrus"
	"github.com/sky-uk/etcd-bootstrap/cloud"
	"golang.org/x/net/context"
)

type etcdMembersAPI interface {
	List(ctx context.Context) ([]client.Member, error)
	Add(ctx context.Context, peerURL string) (*client.Member, error)
	Remove(ctx context.Context, mID string) error
}

// Cluster represents an etcd cluster.
type Cluster struct {
	membersAPIClient etcdMembersAPI
}

// Provider of cloud node instance information.
type Provider interface {
	// GetInstances returns all the non-terminated instances that will be part of the etcd cluster.
	GetInstances() ([]cloud.Instance, error)
}

// Member represents a node in the etcd cluster.
type Member struct {
	Name    string
	PeerURL string
}

// New returns a cluster object for interacting with the etcd cluster API.
func New(provider Provider) (*Cluster, error) {
	instances, err := provider.GetInstances()
	if err != nil {
		return nil, err
	}

	var endpoints []string
	for _, instance := range instances {
		endpoints = append(endpoints, fmt.Sprintf("http://%s:2379", instance.Endpoint))
	}

	c, err := client.New(client.Config{Endpoints: endpoints})
	if err != nil {
		return nil, err
	}

	return &Cluster{client.NewMembersAPI(c)}, nil
}

// Members returns the cluster members.
func (e *Cluster) Members() ([]Member, error) {
	etcdMembers, err := e.membersAPIClient.List(context.Background())
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

// AddMember adds a new member to the cluster by its peer URL.
func (e *Cluster) AddMember(peerURL string) error {
	_, err := e.membersAPIClient.Add(context.Background(), peerURL)
	return err
}

// RemoveMember removes a member of the cluster by its peer URL.
func (e *Cluster) RemoveMember(peerURL string) error {
	members, err := e.membersAPIClient.List(context.Background())
	if err != nil {
		return err
	}

	for _, member := range members {
		if err := assertSinglePeerURL(member); err != nil {
			return err
		}
		if member.PeerURLs[0] == peerURL {
			return e.membersAPIClient.Remove(context.Background(), member.ID)
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
