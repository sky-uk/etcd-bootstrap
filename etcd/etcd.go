package etcd

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
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
	cloudAPI  CloudAPI
	protocol  string
	transport client.CancelableTransport
	// membersAPIClient is the cached API client. Don't use it directly, use list/add/remove instead.
	membersAPIClient etcdMembersAPI
}

// CloudAPI returns the cloud instances in the cluster.
type CloudAPI interface {
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

		// Set up CA certificate.
		caCerts, err := ioutil.ReadFile(peerCA)
		if err != nil {
			return fmt.Errorf("unable to load peer ca: %w", err)
		}
		caCertPool := x509.NewCertPool()
		for len(caCerts) > 0 {
			var block *pem.Block
			block, caCerts = pem.Decode(caCerts)
			if block == nil {
				break
			}
			if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
				continue
			}

			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return fmt.Errorf("unable to parse CA certificate %s: %w", peerCA, err)
			}

			caCertPool.AddCert(cert)
		}

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
func New(cloudAPI CloudAPI, opts ...Option) (*ClusterAPI, error) {
	c := &ClusterAPI{
		cloudAPI:  cloudAPI,
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

func (c *ClusterAPI) createEtcdClientConfig() (client.Config, error) {
	instances, err := c.cloudAPI.GetInstances()
	if err != nil {
		return client.Config{}, err
	}

	var endpoints []string
	for _, instance := range instances {
		endpoints = append(endpoints, fmt.Sprintf("%s://%s:2379", c.protocol, instance.Endpoint))
	}

	return client.Config{
		Endpoints: endpoints,
		Transport: c.transport,
	}, nil
}

func (c *ClusterAPI) membersAPI() (etcdMembersAPI, error) {
	if c.membersAPIClient == nil {
		conf, err := c.createEtcdClientConfig()
		if err != nil {
			return nil, err
		}
		cl, err := client.New(conf)
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

func isTLSError(err error) bool {
	if cerr, ok := err.(*client.ClusterError); ok {
		for _, clusterErr := range cerr.Errors {
			if _, ok := clusterErr.(x509.CertificateInvalidError); ok {
				return true
			}
			if _, ok := clusterErr.(x509.UnknownAuthorityError); ok {
				return true
			}
			if _, ok := clusterErr.(x509.HostnameError); ok {
				return true
			}
		}
	}
	return false
}

// Members returns the cluster members.
func (c *ClusterAPI) Members() ([]Member, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	etcdMembers, err := c.list(ctx)
	if err != nil {
		if isTLSError(err) {
			// TLS errors are unexpected, so fail.
			return nil, fmt.Errorf("there is an error with the TLS certificates: %w", err)
		}
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

// AddMemberByPeerURL adds a new member to the cluster by its peer URL.
// etcd bootstraps by requiring the peer URL to be first added. Then the new node informs etcd of its name.
func (c *ClusterAPI) AddMemberByPeerURL(peerURL string) error {
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	_, err := c.add(ctx, peerURL)
	return err
}

// RemoveMemberByName removes a member of the cluster by its name.
func (c *ClusterAPI) RemoveMemberByName(name string) error {
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	members, err := c.list(ctx)
	if err != nil {
		return err
	}

	for _, member := range members {
		if member.Name == name {
			return c.remove(ctx, member.ID)
		}
	}

	log.Infof("%s has already been removed", name)
	return nil
}

func assertSinglePeerURL(member client.Member) error {
	if len(member.PeerURLs) != 1 {
		return fmt.Errorf("expected a single peer URL, but found %v for %s", member.PeerURLs, member.ID)
	}
	return nil
}
