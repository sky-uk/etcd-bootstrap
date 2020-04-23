package bootstrap

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sky-uk/etcd-bootstrap/cloud"
	"github.com/sky-uk/etcd-bootstrap/etcd"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestBootstrap to register the test suite
func TestBootstrap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bootstrap")
}

var _ = Describe("Bootstrap", func() {
	var (
		cloudAPIMock            *CloudAPIMock
		etcdAPIMock             *EtcdAPIMock
		bootstrapper            *Bootstrapper
		localEndpoint           string
		localIP                 string
		localInstanceID         string
		localListenPeerURL      string
		localListenClientURL    string
		localAdvertisePeerURL   string
		localAdvertiseClientURL string
	)

	BeforeEach(func() {
		localEndpoint = "test-local-endpoint"
		localIP = "192.168.100.1"
		localInstanceID = "test-local-instance"
		localAdvertisePeerURL = fmt.Sprintf("http://%v:2380", localEndpoint)
		localAdvertiseClientURL = fmt.Sprintf("http://%v:2379", localEndpoint)
		localListenPeerURL = fmt.Sprintf("http://%v:2380", localIP)
		localListenClientURL = fmt.Sprintf("http://%v:2379", localIP)
	})

	JustBeforeEach(func() {
		cloudAPIMock = &CloudAPIMock{
			GetInstancesMock: &GetInstances{},
			GetLocalInstanceMock: &GetLocalInstance{
				GetLocalInstance: cloud.Instance{
					Name:     localInstanceID,
					Endpoint: localEndpoint,
				},
			},
			GetLocalIPMock: &GetLocalIP{
				LocalIP: localIP,
			},
		}
		etcdAPIMock = &EtcdAPIMock{
			MembersMock:      &Members{},
			AddMemberMock:    &AddMember{},
			RemoveMemberMock: &RemoveMember{},
		}
		bootstrapper = &Bootstrapper{
			cloudAPI: cloudAPIMock,
			etcdAPI:  etcdAPIMock,
			protocol: "http",
		}
	})

	It("helper functions work", func() {
		Expect(bootstrapper.peerURL(localIP)).To(Equal(localListenPeerURL))
		Expect(bootstrapper.clientURL(localIP)).To(Equal(localListenClientURL))
	})

	It("fails when it cannot get etcd members", func() {
		cloudAPIMock.GetInstancesMock.GetInstancesOutput = []cloud.Instance{
			{
				Name:     localInstanceID,
				Endpoint: localEndpoint,
			},
		}
		etcdAPIMock.MembersMock.Err = fmt.Errorf("failed to get etcd members")
		_, err := bootstrapper.GenerateEtcdFlags()
		Expect(err).To(Not(Succeed()))
	})

	It("does not fail when it cannot remove member", func() {
		cloudAPIMock.GetInstancesMock.GetInstancesOutput = []cloud.Instance{
			{
				Name:     localInstanceID,
				Endpoint: localEndpoint,
			},
		}
		etcdAPIMock.MembersMock.MembersOutput = []etcd.Member{
			{
				Name:    localInstanceID,
				PeerURL: localAdvertisePeerURL,
			},
			{
				Name:    "test-remove-instance-id-1",
				PeerURL: "http://etcd-remove-me:2380",
			},
		}

		By("Expecting to receive a call to remove a test instance")
		nameToRemove := "test-remove-instance-id-1"
		etcdAPIMock.RemoveMemberMock.ExpectedInput = &nameToRemove

		By("Returning an error when trying to remove an etcd member")
		etcdAPIMock.RemoveMemberMock.Err = fmt.Errorf("failed to remove etcd members")
		_, err := bootstrapper.GenerateEtcdFlags()

		By("Do not fail as it may be down to an etcd quorum issue")
		Expect(err).To(BeNil())
	})

	It("fails when it cannot add the local instance as a new etcd member", func() {
		By("The cloudAPI returns instances including the local instance")
		cloudAPIMock.GetInstancesMock.GetInstancesOutput = []cloud.Instance{
			{
				Name:     localInstanceID,
				Endpoint: localEndpoint,
			},
			{
				Name:     "test-add-instance-id-1",
				Endpoint: "endpoint-1",
			},
		}

		By("The etcd API only returns one member, missing the local instance")
		etcdAPIMock.MembersMock.MembersOutput = []etcd.Member{
			{
				Name:    "test-add-instance-id-1",
				PeerURL: "http://endpoint-1:2380",
			},
		}

		By("Returning an error when attempting to add the local instance to the etcd API")
		etcdAPIMock.AddMemberMock.ExpectedInput = &localAdvertisePeerURL
		etcdAPIMock.AddMemberMock.Err = fmt.Errorf("failed to add etcd member")
		_, err := bootstrapper.GenerateEtcdFlags()
		Expect(err).ToNot(Succeed())
	})

	Describe("new cluster", func() {
		JustBeforeEach(func() {
			By("Return a cluster of instances including the local instance")
			cloudAPIMock.GetInstancesMock.GetInstancesOutput = []cloud.Instance{
				{
					Name:     localInstanceID,
					Endpoint: localEndpoint,
				},
				{
					Name:     "test-new-cluster-instance-id-1",
					Endpoint: "test-new-cluster-endpoint-1",
				},
				{
					Name:     "test-new-cluster-instance-id-2",
					Endpoint: "test-new-cluster-endpoint-2",
				},
			}

			By("And return an empty list of etcd members")
			etcdAPIMock.MembersMock.MembersOutput = []etcd.Member{}
		})

		It("should create etcd flags for initializing a new cluster", func() {
			etcdFlags, err := bootstrapper.GenerateEtcdFlags()
			flags := strings.Split(etcdFlags, "\n")
			Expect(err).To(BeNil())
			Expect(flags).To(ContainElement("ETCD_INITIAL_CLUSTER_STATE=new"))
			Expect(flags).To(ContainElement(
				fmt.Sprintf("ETCD_INITIAL_CLUSTER=%s=%s,%s=%s,%s=%s",
					localInstanceID, localAdvertisePeerURL,
					"test-new-cluster-instance-id-1", "http://test-new-cluster-endpoint-1:2380",
					"test-new-cluster-instance-id-2", "http://test-new-cluster-endpoint-2:2380")))
			Expect(flags).To(ContainElement("ETCD_NAME=" + localInstanceID))
			Expect(flags).To(ContainElement("ETCD_INITIAL_ADVERTISE_PEER_URLS=" + localAdvertisePeerURL))
			Expect(flags).To(ContainElement("ETCD_ADVERTISE_CLIENT_URLS=" + localAdvertiseClientURL))
			Expect(flags).To(ContainElement("ETCD_LISTEN_PEER_URLS=" + localListenPeerURL))
			Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=%v,%v",
				localListenClientURL, bootstrapper.clientURL("127.0.0.1"))))
		})
	})

	Describe("an existing cluster", func() {
		JustBeforeEach(func() {
			By("Returning some instances including the local instance")
			cloudAPIMock.GetInstancesMock.GetInstancesOutput = []cloud.Instance{
				{
					Name:     localInstanceID,
					Endpoint: localEndpoint,
				},
				{
					Name:     "test-existing-cluster-instance-id-1",
					Endpoint: "endpoint-1",
				},
				{
					Name:     "test-existing-cluster-instance-id-2",
					Endpoint: "endpoint-2",
				},
			}

			By("And returning a list of etcd members that includes all of the instances")
			etcdAPIMock.MembersMock.MembersOutput = []etcd.Member{
				{
					Name:    localInstanceID,
					PeerURL: localListenPeerURL,
				},
				{
					Name:    "test-existing-cluster-instance-id-1",
					PeerURL: "http://test-existing-cluster-endpoint-1:2380",
				},
				{
					Name:    "test-existing-cluster-instance-id-2",
					PeerURL: "http://test-existing-cluster-endpoint-2:2380",
				},
			}
		})

		It("should create etcd flags for joining an existing cluster", func() {
			etcdFlags, err := bootstrapper.GenerateEtcdFlags()
			flags := strings.Split(etcdFlags, "\n")
			Expect(err).To(BeNil())
			Expect(flags).To(ContainElement("ETCD_INITIAL_CLUSTER_STATE=new"))
			Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_INITIAL_CLUSTER=%s=%s,%s=%s,%s=%s",
				localInstanceID, localAdvertisePeerURL,
				"test-existing-cluster-instance-id-1", "http://endpoint-1:2380",
				"test-existing-cluster-instance-id-2", "http://endpoint-2:2380")))
			Expect(flags).To(ContainElement("ETCD_NAME=" + localInstanceID))
			Expect(flags).To(ContainElement("ETCD_INITIAL_ADVERTISE_PEER_URLS=" + localAdvertisePeerURL))
			Expect(flags).To(ContainElement("ETCD_ADVERTISE_CLIENT_URLS=" + localAdvertiseClientURL))
			Expect(flags).To(ContainElement("ETCD_LISTEN_PEER_URLS=" + localListenPeerURL))
			Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=%v,%v",
				localListenClientURL, bootstrapper.clientURL("127.0.0.1"))))
		})
	})

	Describe("an existing cluster where a node needs replacing", func() {
		JustBeforeEach(func() {
			By("Returning some instances including the local instance")
			cloudAPIMock.GetInstancesMock.GetInstancesOutput = []cloud.Instance{
				{
					Name:     localInstanceID,
					Endpoint: localEndpoint,
				},
				{
					Name:     "test-existing-cluster-instance-id-2",
					Endpoint: "endpoint-2",
				},
				{
					Name:     "test-existing-cluster-instance-id-3",
					Endpoint: "endpoint-3",
				},
			}

			By("Returning a list of etcd members that contains too many members but does not include the local instance")
			etcdAPIMock.MembersMock.MembersOutput = []etcd.Member{
				{
					Name:    "test-existing-cluster-old-instance-id-1",
					PeerURL: "http://endpoint-1:2380",
				},
				{
					Name:    "test-existing-cluster-instance-id-2",
					PeerURL: "http://endpoint-2:2380",
				},
				{
					Name:    "test-existing-cluster-instance-id-3",
					PeerURL: "http://endpoint-3:2380",
				},
			}
		})

		It("should remove the prior node and add the local node", func() {
			oldInstanceID := "test-existing-cluster-old-instance-id-1"
			etcdAPIMock.RemoveMemberMock.ExpectedInput = &oldInstanceID
			etcdAPIMock.AddMemberMock.ExpectedInput = &localAdvertisePeerURL
			etcdFlags, err := bootstrapper.GenerateEtcdFlags()
			Expect(err).To(BeNil())
			Expect(etcdAPIMock.RemoveMemberMock.Called).To(BeTrue())
			Expect(etcdAPIMock.AddMemberMock.Called).To(BeTrue())
			flags := strings.Split(etcdFlags, "\n")
			Expect(flags).To(ContainElement("ETCD_INITIAL_CLUSTER_STATE=existing"))
			Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_INITIAL_CLUSTER=%s=%s,%s=%s",
				"test-existing-cluster-instance-id-2", "http://endpoint-2:2380",
				"test-existing-cluster-instance-id-3", "http://endpoint-3:2380")))
			Expect(flags).To(ContainElement("ETCD_NAME=" + localInstanceID))
			Expect(flags).To(ContainElement("ETCD_INITIAL_ADVERTISE_PEER_URLS=" + localAdvertisePeerURL))
			Expect(flags).To(ContainElement("ETCD_ADVERTISE_CLIENT_URLS=" + localAdvertiseClientURL))
			Expect(flags).To(ContainElement("ETCD_LISTEN_PEER_URLS=" + localListenPeerURL))
			Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=%v,%v", localListenClientURL, bootstrapper.clientURL("127.0.0.1"))))
		})
	})

	Describe("an existing cluster that is partially initialised", func() {
		JustBeforeEach(func() {
			By("Returning some instances including the local instance")
			cloudAPIMock.GetInstancesMock.GetInstancesOutput = []cloud.Instance{
				{
					Name:     localInstanceID,
					Endpoint: localEndpoint,
				},
				{
					Name:     "test-existing-cluster-partially-initialised-instance-id-1",
					Endpoint: "endpoint-1",
				},
				{
					Name:     "test-existing-cluster-partially-initialised-instance-id-2",
					Endpoint: "endpoint-2",
				},
			}

			By("Returning a list of etcd members that contains the local instance as partially initialised")
			etcdAPIMock.MembersMock.MembersOutput = []etcd.Member{
				{
					// Name will be blank after adding the peerURL until the instance registers itself.
					Name:    "",
					PeerURL: localAdvertisePeerURL,
				},
				{
					Name:    "test-existing-cluster-partially-initialised-instance-id-1",
					PeerURL: "http://endpoint-1:2380",
				},
				{
					Name:    "test-existing-cluster-partially-initialised-instance-id-2",
					PeerURL: "http://endpoint-2:2380",
				},
			}
		})

		It("should add the local instance and generate etcd flags including the local instance", func() {
			etcdFlags, err := bootstrapper.GenerateEtcdFlags()
			Expect(err).To(BeNil())

			flags := strings.Split(etcdFlags, "\n")
			Expect(flags).To(ContainElement("ETCD_INITIAL_CLUSTER_STATE=existing"))
			Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_INITIAL_CLUSTER=%s=%s,%s=%s",
				"test-existing-cluster-partially-initialised-instance-id-1", "http://endpoint-1:2380",
				"test-existing-cluster-partially-initialised-instance-id-2", "http://endpoint-2:2380")))
			Expect(flags).To(ContainElement("ETCD_NAME=" + localInstanceID))
			Expect(flags).To(ContainElement("ETCD_INITIAL_ADVERTISE_PEER_URLS=" + localAdvertisePeerURL))
			Expect(flags).To(ContainElement("ETCD_ADVERTISE_CLIENT_URLS=" + localAdvertiseClientURL))
			Expect(flags).To(ContainElement("ETCD_LISTEN_PEER_URLS=" + localListenPeerURL))
			Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=%v,%v", localListenClientURL, bootstrapper.clientURL("127.0.0.1"))))
		})
	})

	Describe("TLS new cluster", func() {
		var (
			serverCA   = "server-ca.pem"
			serverCert = "server.pem"
			serverKey  = "server-key.pem"
			peerCA     = "peer-ca.pem"
			peerCert   = "peer.pem"
			peerKey    = "peer.key"
		)

		BeforeEach(func() {
			localAdvertisePeerURL = fmt.Sprintf("https://%v:2380", localEndpoint)
			localAdvertiseClientURL = fmt.Sprintf("https://%v:2379", localEndpoint)
			localListenPeerURL = fmt.Sprintf("https://%v:2380", localIP)
			localListenClientURL = fmt.Sprintf("https://%v:2379", localIP)
		})

		JustBeforeEach(func() {
			WithTLS(serverCA, serverCert, serverKey, peerCA, peerCert, peerKey)(bootstrapper)
			cloudAPIMock.GetInstancesMock.GetInstancesOutput = []cloud.Instance{
				{
					Name:     localInstanceID,
					Endpoint: localEndpoint,
				},
				{
					Name:     "test-new-cluster-instance-id-1",
					Endpoint: "endpoint-1",
				},
				{
					Name:     "test-new-cluster-instance-id-2",
					Endpoint: "endpoint-2",
				},
			}
			etcdAPIMock.MembersMock.MembersOutput = []etcd.Member{}
		})

		It("adds the required TLS flags", func() {
			etcdFlags, err := bootstrapper.GenerateEtcdFlags()
			Expect(err).To(BeNil())
			flags := strings.Split(etcdFlags, "\n")
			Expect(flags).To(ContainElement("ETCD_INITIAL_CLUSTER_STATE=new"))
			Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_INITIAL_CLUSTER=%s=%s,%s=%s,%s=%s",
				localInstanceID, localAdvertisePeerURL,
				"test-new-cluster-instance-id-1", "https://endpoint-1:2380",
				"test-new-cluster-instance-id-2", "https://endpoint-2:2380")))
			Expect(flags).To(ContainElement("ETCD_NAME=" + localInstanceID))
			Expect(flags).To(ContainElement("ETCD_INITIAL_ADVERTISE_PEER_URLS=" + localAdvertisePeerURL))
			Expect(flags).To(ContainElement("ETCD_ADVERTISE_CLIENT_URLS=" + localAdvertiseClientURL))
			Expect(flags).To(ContainElement("ETCD_LISTEN_PEER_URLS=" + localListenPeerURL))
			Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=%s,%s",
				localListenClientURL, bootstrapper.clientURL("127.0.0.1"))))
			// TLS flags
			Expect(flags).To(ContainElement("ETCD_CLIENT_CERT_AUTH=true"))
			Expect(flags).To(ContainElement("ETCD_TRUSTED_CA_FILE=" + serverCA))
			Expect(flags).To(ContainElement("ETCD_CERT_FILE=" + serverCert))
			Expect(flags).To(ContainElement("ETCD_KEY_FILE=" + serverKey))
			Expect(flags).To(ContainElement("ETCD_PEER_CLIENT_CERT_AUTH=true"))
			Expect(flags).To(ContainElement("ETCD_PEER_TRUSTED_CA_FILE=" + peerCA))
			Expect(flags).To(ContainElement("ETCD_PEER_CERT_FILE=" + peerCert))
			Expect(flags).To(ContainElement("ETCD_PEER_KEY_FILE=" + peerKey))
		})
	})
})

// EtcdAPIMock for mocking calls to the etcd cluster package client
type EtcdAPIMock struct {
	MembersMock      *Members
	RemoveMemberMock *RemoveMember
	AddMemberMock    *AddMember
}

// Members sets the expected output for Members() on EtcdCluster
type Members struct {
	MembersOutput []etcd.Member
	Err           error
}

// Members mocks the etcd cluster package client
func (t EtcdAPIMock) Members() ([]etcd.Member, error) {
	return t.MembersMock.MembersOutput, t.MembersMock.Err
}

// RemoveMember sets the expected input for RemoveMember() on EtcdCluster
type RemoveMember struct {
	Called        bool
	ExpectedInput *string
	Err           error
}

// RemoveMemberByName mocks the etcd cluster package client
func (t EtcdAPIMock) RemoveMemberByName(name string) error {
	t.RemoveMemberMock.Called = true
	Expect(t.RemoveMemberMock.ExpectedInput).To(Not(BeNil()), "unexpected RemoveMember call with %q", name)
	Expect(*t.RemoveMemberMock.ExpectedInput).To(Equal(name), "unexpected RemoveMember call")
	return t.RemoveMemberMock.Err
}

// AddMember sets the expected input for AddMember() on EtcdCluster
type AddMember struct {
	Called        bool
	ExpectedInput *string
	Err           error
}

// AddMemberByPeerURL mocks the etcd cluster package client
func (t EtcdAPIMock) AddMemberByPeerURL(peerURL string) error {
	t.AddMemberMock.Called = true
	Expect(t.AddMemberMock.ExpectedInput).To(Not(BeNil()), "unexpected AddMember call with %q", peerURL)
	Expect(*t.AddMemberMock.ExpectedInput).To(Equal(peerURL), "unexpected AddMember call")
	return t.AddMemberMock.Err
}

// CloudAPIMock for mocking calls to an etcd-bootstrap cloud provider
type CloudAPIMock struct {
	GetInstancesMock     *GetInstances
	GetLocalInstanceMock *GetLocalInstance
	GetLocalIPMock       *GetLocalIP
}

// GetInstances sets the expected output for GetInstances() on CloudProvider
type GetInstances struct {
	GetInstancesOutput []cloud.Instance
	Error              error
}

// GetInstances mocks the etcd-bootstrap cloud provider
func (t CloudAPIMock) GetInstances() ([]cloud.Instance, error) {
	return t.GetInstancesMock.GetInstancesOutput, t.GetInstancesMock.Error
}

// GetLocalInstance sets the expected output for GetLocalInstance() on CloudProvider
type GetLocalInstance struct {
	GetLocalInstance cloud.Instance
	Error            error
}

// GetLocalInstance mocks the etcd-bootstrap cloud provider
func (t CloudAPIMock) GetLocalInstance() (cloud.Instance, error) {
	return t.GetLocalInstanceMock.GetLocalInstance, t.GetLocalInstanceMock.Error
}

type GetLocalIP struct {
	LocalIP string
	Error   error
}

func (t CloudAPIMock) GetLocalIP() (string, error) {
	return t.GetLocalIPMock.LocalIP, t.GetLocalIPMock.Error
}
