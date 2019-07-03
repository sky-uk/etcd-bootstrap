package etcd

import (
	"fmt"
	"testing"

	"github.com/coreos/etcd/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
)

// Mock etcd members API client
type EtcdMembersAPI struct {
	MockList   MockList
	MockAdd    MockAdd
	MockRemove MockRemove
}

type MockList struct {
	ExpectedInput context.Context
	ListOutput    []client.Member
	Err           error
}

func (t EtcdMembersAPI) List(ctx context.Context) ([]client.Member, error) {
	Expect(ctx).To(Equal(t.MockList.ExpectedInput))
	return t.MockList.ListOutput, t.MockList.Err
}

type MockAdd struct {
	ExpectedContext context.Context
	ExpectedPeerURL string
	AddOutput       *client.Member
	Err             error
}

func (t EtcdMembersAPI) Add(ctx context.Context, peerURL string) (*client.Member, error) {
	Expect(ctx).To(Equal(t.MockAdd.ExpectedContext))
	Expect(peerURL).To(Equal(t.MockAdd.ExpectedPeerURL))
	return t.MockAdd.AddOutput, t.MockAdd.Err
}

type MockRemove struct {
	ExpectedContext context.Context
	ExpectedMID     string
	Err             error
}

func (t EtcdMembersAPI) Remove(ctx context.Context, mID string) error {
	Expect(ctx).To(Equal(t.MockRemove.ExpectedContext))
	Expect(mID).To(Equal(t.MockRemove.ExpectedMID))
	return t.MockRemove.Err
}

func TestEtcd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Etcd client")
}

var _ = Describe("Etcd client", func() {
	var membersAPIClient EtcdMembersAPI

	BeforeEach(func() {
		// create dummy client with expected data
		// this data can be manipulated within tests
		membersAPIClient = EtcdMembersAPI{
			MockList: MockList{
				ExpectedInput: context.Background(),
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
			MockAdd: MockAdd{
				ExpectedContext: context.Background(),
			},
			MockRemove: MockRemove{
				ExpectedContext: context.Background(),
			},
		}
	})

	Context("Members()", func() {
		It("can list when the etcd members api client responds with expected results", func() {
			// create etcdCluster with a client which responds with expected values
			etcdCluster := &cluster{membersAPIClient: membersAPIClient}
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
			// create an etcdCluster which isn't able to list members
			etcdCluster := &cluster{membersAPIClient: membersAPIClient}
			_, err := etcdCluster.Members()
			Expect(err).To(BeNil())
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
			// create an etcdCluster which lists complex members
			etcdCluster := &cluster{membersAPIClient: membersAPIClient}
			_, err := etcdCluster.Members()
			Expect(err).ToNot(BeNil())
		})
	})

	Context("AddMember()", func() {
		It("can add a member when the client doesn't error", func() {
			membersAPIClient.MockAdd.ExpectedPeerURL = "http://192.168.0.100"
			// create etcdCluster with a client which responds with expected values
			etcdCluster := &cluster{membersAPIClient: membersAPIClient}
			Expect(etcdCluster.AddMember("http://192.168.0.100")).To(BeNil())
		})
	})

	Context("Remove()", func() {
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
			// create etcdCluster with a client which responds with expected values
			etcdCluster := &cluster{membersAPIClient: membersAPIClient}
			Expect(etcdCluster.RemoveMember("http://192.168.0.1:2379")).To(BeNil())
		})

		It("fails if it is unable to list members using the etcd members api client", func() {
			membersAPIClient.MockList.Err = fmt.Errorf("failed to list members")
			// create etcdCluster with a client which is unable to list members
			etcdCluster := &cluster{membersAPIClient: membersAPIClient}
			Expect(etcdCluster.RemoveMember("http://192.168.0.1:2379")).ToNot(BeNil())
		})

		It("fails if the member list contains complex peer urls", func() {
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
			// create an etcdCluster which lists complex members
			etcdCluster := &cluster{membersAPIClient: membersAPIClient}
			Expect(etcdCluster.RemoveMember("http://192.168.0.1:2379")).ToNot(BeNil())
		})

		It("does nothing if the member has already been removed", func() {
			// we can assume this has done nothin as we are accepting a blank string when Remove() is called
			membersAPIClient.MockList.ListOutput = []client.Member{
				{
					ID:         "test-other-instance-id",
					Name:       "test-other-instance-name",
					PeerURLs:   []string{"http://192.168.0.1:2379"},
					ClientURLs: []string{"http://192.168.0.1:2380"},
				},
			}
			// create an etcdCluster which lists members which I won't interact with
			etcdCluster := &cluster{membersAPIClient: membersAPIClient}
			Expect(etcdCluster.RemoveMember("http://172.16.0.1:2379")).To(BeNil())
		})
	})
})
