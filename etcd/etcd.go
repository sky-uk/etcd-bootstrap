package etcd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/coreos/etcd/client"
	log "github.com/sirupsen/logrus"
	"github.com/sky-uk/etcd-bootstrap/cloud"
	"golang.org/x/net/context"
)

const timeout = 5 * time.Second

type etcdMembersAPI interface {
	List(ctx context.Context) ([]client.Member, error)
	Add(ctx context.Context, peerURL string) (*client.Member, error)
	Remove(ctx context.Context, mID string) error
}

// ClusterAPI represents an etcd cluster API.
type ClusterAPI struct {
	instances Instances
	protocol  string
	transport client.CancelableTransport
	// membersAPIClient is the cached API client. Don't use it directly, use list/add/remove instead.
	membersAPIClient etcdMembersAPI
}

// Instances returns the instances in the cluster.
type Instances interface {
	// GetInstances returns all the non-terminated instances that will be part of the etcd cluster.
	GetInstances() ([]cloud.Instance, error)
}

// Member represents a node in the etcd cluster.
type Member struct {
	Name    string
	PeerURL string
}

// Option for New.
type Option func(c *ClusterAPI) error

// WithTLS enables TLS for talking to the etcd cluster. It requires the locations of the peer certificates.
func WithTLS(peerCA, peerCert, peerKey string) Option {
	return func(c *ClusterAPI) error {
		vals := []struct {
			name, val string
		}{
			{"peerCA", peerCA},
			{"peerCert", peerCert},
			{"peerKey", peerKey},
		}
		for _, val := range vals {
			if _, err := os.Stat(val.val); err != nil {
				return fmt.Errorf("%s is inaccessible: %w", val.val, err)
			}
		}

		// Set up client certificates.
		cert, err := tls.LoadX509KeyPair(peerCert, peerKey)
		if err != nil {
			return fmt.Errorf("unable to load peer certs: %w", err)
		}
		caCert, err := ioutil.ReadFile(peerCA)
		if err != nil {
			return fmt.Errorf("unable to load peer ca: %w", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		}
		tlsConfig.BuildNameToCertificate()

		// Base settings are copied from client.DefaultTransport.
		c.transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig:     tlsConfig,
		}
		c.protocol = "https"
		return nil
	}
}

// New returns a cluster object for interacting with the etcd cluster API.
func New(instances Instances, opts ...Option) (*ClusterAPI, error) {
	c := &ClusterAPI{
		instances: instances,
		protocol:  "http",
		transport: client.DefaultTransport,
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *ClusterAPI) membersAPI() (etcdMembersAPI, error) {
	if c.membersAPIClient == nil {
		instances, err := c.instances.GetInstances()
		if err != nil {
			return nil, err
		}

		var endpoints []string
		for _, instance := range instances {
			endpoints = append(endpoints, fmt.Sprintf("%s://%s:2379", c.protocol, instance.Endpoint))
		}

		cl, err := client.New(client.Config{Endpoints: endpoints})
		if err != nil {
			return nil, err
		}

		c.membersAPIClient = client.NewMembersAPI(cl)
	}
	return c.membersAPIClient, nil
}

func (c *ClusterAPI) list(ctx context.Context) ([]client.Member, error) {
	api, err := c.membersAPI()
	if err != nil {
		return nil, err
	}
	return api.List(ctx)
}

func (c *ClusterAPI) add(ctx context.Context, peerURL string) (*client.Member, error) {
	api, err := c.membersAPI()
	if err != nil {
		return nil, err
	}
	return api.Add(ctx, peerURL)
}

func (c *ClusterAPI) remove(ctx context.Context, mID string) error {
	api, err := c.membersAPI()
	if err != nil {
		return err
	}
	return api.Remove(ctx, mID)
}

// Members returns the cluster members.
func (c *ClusterAPI) Members() ([]Member, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	etcdMembers, err := c.list(ctx)
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
func (c *ClusterAPI) AddMember(peerURL string) error {
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	_, err := c.add(ctx, peerURL)
	return err
}

// RemoveMember removes a member of the cluster by its peer URL.
func (c *ClusterAPI) RemoveMember(peerURL string) error {
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	members, err := c.list(ctx)
	if err != nil {
		return err
	}

	for _, member := range members {
		if err := assertSinglePeerURL(member); err != nil {
			return err
		}
		if member.PeerURLs[0] == peerURL {
			return c.remove(ctx, member.ID)
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
