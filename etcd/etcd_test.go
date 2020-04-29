package etcd

import (
	"crypto/x509"
	"fmt"
	"net/http"
	"testing"

	"github.com/coreos/etcd/client"
	"github.com/sky-uk/etcd-bootstrap/cloud"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
)

// TestEtcd to register the test suite
func TestEtcd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Etcd client")
}

var _ = Describe("Etcd client", func() {
	var membersAPIClient MockMembersAPI

	BeforeEach(func() {
		By("Creating dummy client responses")
		membersAPIClient = MockMembersAPI{
			MockList: List{
				ListOutput: []client.Member{
					{
						ID:         "test-good-response-id-1",
						Name:       "test-good-response-name-1",
						PeerURLs:   []string{"http://192.168.0.1:2380"},
						ClientURLs: []string{"http://192.168.0.1:2379"},
					},
					{
						ID:         "test-good-response-id-2",
						Name:       "test-good-response-name-2",
						PeerURLs:   []string{"http://192.168.0.2:2380"},
						ClientURLs: []string{"http://192.168.0.2:2379"},
					},
				},
			},
			MockAdd:    Add{},
			MockRemove: Remove{},
		}
	})

	Context("Members()", func() {
		It("can list when the etcd members api client responds with expected results", func() {
			By("Returning all expected responses")
			etcdCluster := &ClusterAPI{membersAPIClient: membersAPIClient}
			memberList, err := etcdCluster.Members()
			Expect(err).To(BeNil())
			Expect(memberList).To(Equal([]Member{
				{
					Name:    "test-good-response-name-1",
					PeerURL: "http://192.168.0.1:2380",
				},
				{
					Name:    "test-good-response-name-2",
					PeerURL: "http://192.168.0.2:2380",
				},
			}))
		})

		It("continues even if the etcd members api client errors on List()", func() {
			membersAPIClient.MockList.Err = fmt.Errorf("failed to list members")

			By("Return a client that isn't able to list etcd members")
			etcdCluster := &ClusterAPI{membersAPIClient: membersAPIClient}
			_, err := etcdCluster.Members()
			Expect(err).To(BeNil())
		})

		It("fails if a TLS error occurred", func() {
			certErrors := []error{x509.CertificateInvalidError{}, x509.UnknownAuthorityError{}, x509.HostnameError{}}
			for _, certErr := range certErrors {
				clusterErr := client.ClusterError{
					Errors: []error{certErr},
				}
				membersAPIClient.MockList.Err = &clusterErr

				etcdCluster := &ClusterAPI{membersAPIClient: membersAPIClient}
				_, err := etcdCluster.Members()
				Expect(err).To(Not(Succeed()), "should fail on %v", certErr)
			}
		})

		It("fails when the etcd members api response contains a member with more than one peer url", func() {
			membersAPIClient.MockList.ListOutput = []client.Member{
				{
					ID:   "test-complex-response-id-1",
					Name: "test-complex-response-id-1",
					PeerURLs: []string{
						"http://192.168.0.1:2380",
						"http://172.16.0.1:2380",
					},
					ClientURLs: []string{
						"http://192.168.0.1:2379",
						"http://172.16.0.1:2379",
					},
				},
			}

			By("Returning an etcd client that returns complex members")
			etcdCluster := &ClusterAPI{membersAPIClient: membersAPIClient}
			_, err := etcdCluster.Members()
			Expect(err).ToNot(BeNil())
		})
	})

	Context("AddMemberByPeerURL()", func() {
		It("can add a member when the client doesn't error", func() {
			membersAPIClient.MockAdd.ExpectedPeerURL = "http://192.168.0.100"

			By("Returning all expected responses")
			etcdCluster := &ClusterAPI{membersAPIClient: membersAPIClient}
			Expect(etcdCluster.AddMemberByPeerURL("http://192.168.0.100")).To(BeNil())
		})
	})

	Context("RemoveMemberByName()", func() {
		It("can use the etcd members api client to remove a member", func() {
			membersAPIClient.MockList.ListOutput = []client.Member{
				{
					ID:         "test-remove-instance-id",
					Name:       "test-remove-instance-name",
					PeerURLs:   []string{"http://192.168.0.1:2379"},
					ClientURLs: []string{"http://192.168.0.1:2380"},
				},
			}
			membersAPIClient.MockRemove.ExpectedMID = "test-remove-instance-id"

			By("Returning all expected responses")
			etcdCluster := &ClusterAPI{membersAPIClient: membersAPIClient}
			Expect(etcdCluster.RemoveMemberByName("test-remove-instance-name")).To(BeNil())
		})

		It("fails if it is unable to list members using the etcd members api client", func() {
			membersAPIClient.MockList.Err = fmt.Errorf("failed to list members")

			By("Returning a client that isn't able to list etcd members")
			etcdCluster := &ClusterAPI{membersAPIClient: membersAPIClient}
			Expect(etcdCluster.RemoveMemberByName("test-remove-instance-name")).ToNot(BeNil())
		})

		It("does nothing if the member has already been removed", func() {
			By("Expecting the Remove() call not to be made as we allow only blank PeerURL's")
			membersAPIClient.MockList.ListOutput = []client.Member{
				{
					ID:         "test-other-instance-id",
					Name:       "test-other-instance-name",
					PeerURLs:   []string{"http://192.168.0.1:2379"},
					ClientURLs: []string{"http://192.168.0.1:2380"},
				},
			}

			By("Returning an etcd member list containing irrelevant members")
			etcdCluster := &ClusterAPI{membersAPIClient: membersAPIClient}
			Expect(etcdCluster.RemoveMemberByName("test-remove-instance-name")).To(BeNil())
		})
	})

	Describe("WithTLS()", func() {
		var (
			// Created with:
			//   CAROOT=. mkcert -install && mv rootCA.pem root
			//   CAROOT=. mkcert -client -cert-file test.pem -key-file test-key.pem test-client-certs
			peerCA   = "test-ca.pem"
			peerCert = "test.pem"
			peerKey  = "test-key.pem"
		)

		It("can load the certificates successfully", func() {
			cluster := &ClusterAPI{}
			Expect(WithTLS(peerCA, peerCert, peerKey)(cluster)).To(Succeed())
			Expect(cluster.protocol).To(Equal("https"))
			transport := (cluster.transport).(*http.Transport)
			Expect(transport.TLSClientConfig).To(Not(BeNil()), "tls client config should be set")
		})
	})

	Describe("createEtcdClientConfig()", func() {
		var cloudAPI CloudAPI

		BeforeEach(func() {
			cloudAPI = &mockCloudAPI{
				instances: []cloud.Instance{
					{
						Name:     "i-123",
						Endpoint: "etcd-1",
					},
				},
			}
		})

		It("adds the correct endponts", func() {
			cluster := &ClusterAPI{cloudAPI: cloudAPI, protocol: "pigeon"}
			conf, err := cluster.createEtcdClientConfig()
			Expect(err).To(BeNil())
			Expect(conf.Endpoints).To(ContainElement("pigeon://etcd-1:2379"))
		})

		It("sets the configured transport", func() {
			transport := client.DefaultTransport
			cluster := &ClusterAPI{cloudAPI: cloudAPI, transport: transport}
			conf, err := cluster.createEtcdClientConfig()
			Expect(err).To(BeNil())
			Expect(conf.Transport).To(Equal(transport))
		})
	})
})

// EtcdMembersAPI for mocking calls to the coreos etcd client
type MockMembersAPI struct {
	MockList   List
	MockAdd    Add
	MockRemove Remove
}

// List sets the expected input and output for List() on EtcdMembersAPI
type List struct {
	ListOutput []client.Member
	Err        error
}

func expectContextToHaveDeadline(ctx context.Context) {
	_, ok := ctx.Deadline()
	Expect(ok).To(BeTrue(), "context should have a deadline")
}

// List mocks the coreos etcd client
func (t MockMembersAPI) List(ctx context.Context) ([]client.Member, error) {
	expectContextToHaveDeadline(ctx)
	return t.MockList.ListOutput, t.MockList.Err
}

// Add sets the expected input and output for Add() on EtcdMembersAPI
type Add struct {
	ExpectedPeerURL string
	AddOutput       *client.Member
	Err             error
}

// Add mocks the coreos etcd client
func (t MockMembersAPI) Add(ctx context.Context, peerURL string) (*client.Member, error) {
	expectContextToHaveDeadline(ctx)
	Expect(peerURL).To(Equal(t.MockAdd.ExpectedPeerURL))
	return t.MockAdd.AddOutput, t.MockAdd.Err
}

// Remove sets the expected input and output for Remove() on EtcdMembersAPI
type Remove struct {
	ExpectedMID string
	Err         error
}

// Remove mocks the coreos etcd client
func (t MockMembersAPI) Remove(ctx context.Context, mID string) error {
	expectContextToHaveDeadline(ctx)
	Expect(mID).To(Equal(t.MockRemove.ExpectedMID))
	return t.MockRemove.Err
}

type mockCloudAPI struct {
	instances []cloud.Instance
}

func (m *mockCloudAPI) GetInstances() ([]cloud.Instance, error) {
	return m.instances, nil
}
