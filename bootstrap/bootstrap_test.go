package bootstrap

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sky-uk/etcd-bootstrap/bootstrap/etcd"
	"github.com/sky-uk/etcd-bootstrap/mock"
	"github.com/sky-uk/etcd-bootstrap/provider"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	localPrivateIP  = "192.168.0.100"
	localInstanceID = "test-local-instance"
)

var (
	localPeerURL   = fmt.Sprintf("http://%v:2380", localPrivateIP)
	localClientURL = fmt.Sprintf("http://%v:2379", localPrivateIP)
)

// TestBoostrap to register the test suite
func TestBoostrap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bootstrap")
}

var _ = Describe("Bootstrap", func() {
	var (
		cloudProvider mock.CloudProvider
		etcdCluster   mock.EtcdCluster
	)

	BeforeEach(func() {
		By("Returning the constant instance values")
		cloudProvider = mock.CloudProvider{
			MockGetLocalInstance: mock.GetLocalInstance{
				GetLocalInstance: provider.Instance{
					InstanceID: localInstanceID,
					PrivateIP:  localPrivateIP,
				},
			},
		}

		By("Only calling AddMember() from the local instance using the constant values")
		etcdCluster = mock.EtcdCluster{
			MockAddMember: mock.AddMember{
				ExpectedInput: localPeerURL,
			},
		}
	})

	It("helper functions work", func() {
		Expect(peerURL(localPrivateIP)).To(Equal(localPeerURL))
		Expect(clientURL(localPrivateIP)).To(Equal(localClientURL))
	})

	It("fails when it cannot get etcd members", func() {
		By("Returning some instances including the local instance")
		cloudProvider.MockGetInstances.GetInstancesOutput = []provider.Instance{
			{
				InstanceID: localInstanceID,
				PrivateIP:  localPrivateIP,
			},
		}

		By("Returning an error when getting the list of etcd members")
		etcdCluster.MockMembers.Err = fmt.Errorf("failed to get etcd members")
		bootstrapperClient := bootstrapper{
			provider: cloudProvider,
			cluster:  etcdCluster,
		}

		_, err := bootstrapperClient.GenerateEtcdFlags()
		Expect(err).ToNot(BeNil())
	})

	It("does not fail when it cannot remove member", func() {
		By("Returning some instances including the local instance")
		cloudProvider.MockGetInstances.GetInstancesOutput = []provider.Instance{
			{
				InstanceID: localInstanceID,
				PrivateIP:  localPrivateIP,
			},
		}

		By("Returning an etcd member list requiring an update")
		etcdCluster.MockMembers.MembersOutput = []etcd.Member{
			{
				Name:    "test-remove-instance-id-1",
				PeerURL: "http://192.168.0.1:2380",
			},
		}

		By("Expecting to receive a call to remove a test instance")
		etcdCluster.MockRemoveMember.ExpectedInputs = []string{"http://192.168.0.1:2380"}

		By("Returning an error when trying to remove an etcd member")
		etcdCluster.MockRemoveMember.Err = fmt.Errorf("failed to remove etcd members")
		bootstrapperClient := bootstrapper{
			provider: cloudProvider,
			cluster:  etcdCluster,
		}

		_, err := bootstrapperClient.GenerateEtcdFlags()

		By("Do not fail as it may be down to an etcd quorum issue")
		Expect(err).To(BeNil())
	})

	It("fails when it cannot add etcd member", func() {
		By("Returning some instances including the local instance")
		cloudProvider.MockGetInstances.GetInstancesOutput = []provider.Instance{
			{
				InstanceID: localInstanceID,
				PrivateIP:  localPrivateIP,
			},
			{
				InstanceID: "test-add-instance-id-1",
				PrivateIP:  "192.168.0.1",
			},
		}

		By("Returning an etcd member list requiring an update")
		etcdCluster.MockMembers.MembersOutput = []etcd.Member{
			{
				Name:    "test-add-instance-id-1",
				PeerURL: "http://192.168.0.1:2380",
			},
		}

		By("Returning an error when attempting to list all etcd members")
		etcdCluster.MockAddMember.Err = fmt.Errorf("failed to add etcd member")
		bootstrapperClient := bootstrapper{
			provider: cloudProvider,
			cluster:  etcdCluster,
		}

		_, err := bootstrapperClient.GenerateEtcdFlags()

		By("Do not fail as it may be down to an etcd quorum issue")
		Expect(err).ToNot(BeNil())
	})

	It("new cluster", func() {
		By("Returning some instances including the local instance")
		cloudProvider.MockGetInstances.GetInstancesOutput = []provider.Instance{
			{
				InstanceID: localInstanceID,
				PrivateIP:  localPrivateIP,
			},
			{
				InstanceID: "test-new-cluster-instance-id-1",
				PrivateIP:  "192.168.0.1",
			},
			{
				InstanceID: "test-new-cluster-instance-id-2",
				PrivateIP:  "192.168.0.2",
			},
		}

		By("Returning a list of etcd members that is empty")
		etcdCluster.MockMembers.MembersOutput = []etcd.Member{}
		bootstrapperClient := bootstrapper{
			provider: cloudProvider,
			cluster:  etcdCluster,
		}

		etcdFlags, err := bootstrapperClient.GenerateEtcdFlags()
		flags := strings.Split(etcdFlags, "\n")
		Expect(err).To(BeNil())
		Expect(flags).To(ContainElement("ETCD_INITIAL_CLUSTER_STATE=new"))
		Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_INITIAL_CLUSTER=%v=%v,"+
			"test-new-cluster-instance-id-1=http://192.168.0.1:2380,"+
			"test-new-cluster-instance-id-2=http://192.168.0.2:2380", localInstanceID, localPeerURL)))
		Expect(flags).To(ContainElement("ETCD_NAME=" + localInstanceID))
		Expect(flags).To(ContainElement("ETCD_INITIAL_ADVERTISE_PEER_URLS=" + localPeerURL))
		Expect(flags).To(ContainElement("ETCD_LISTEN_PEER_URLS=" + localPeerURL))
		Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=%v,%v", localClientURL, clientURL("127.0.0.1"))))
		Expect(flags).To(ContainElement("ETCD_ADVERTISE_CLIENT_URLS=" + localClientURL))
	})

	It("an existing cluster", func() {
		By("Returning some instances including the local instance")
		cloudProvider.MockGetInstances.GetInstancesOutput = []provider.Instance{
			{
				InstanceID: localInstanceID,
				PrivateIP:  localPrivateIP,
			},
			{
				InstanceID: "test-existing-cluster-instance-id-1",
				PrivateIP:  "192.168.0.1",
			},
			{
				InstanceID: "test-existing-cluster-instance-id-2",
				PrivateIP:  "192.168.0.2",
			},
		}

		By("Returning a list of etcd members that contains too many members but does not include the local instance")
		etcdCluster.MockMembers.MembersOutput = []etcd.Member{
			{
				Name:    localInstanceID,
				PeerURL: localPeerURL,
			},
			{
				Name:    "test-existing-cluster-instance-id-1",
				PeerURL: "http://192.168.0.1:2380",
			},
			{
				Name:    "test-existing-cluster-instance-id-2",
				PeerURL: "http://192.168.0.2:2380",
			},
		}
		bootstrapperClient := bootstrapper{
			provider: cloudProvider,
			cluster:  etcdCluster,
		}

		etcdFlags, err := bootstrapperClient.GenerateEtcdFlags()
		flags := strings.Split(etcdFlags, "\n")
		Expect(err).To(BeNil())
		Expect(flags).To(ContainElement("ETCD_INITIAL_CLUSTER_STATE=new"))
		Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_INITIAL_CLUSTER=%v=%v,"+
			"test-existing-cluster-instance-id-1=http://192.168.0.1:2380,"+
			"test-existing-cluster-instance-id-2=http://192.168.0.2:2380", localInstanceID, localPeerURL)))
		Expect(flags).To(ContainElement("ETCD_NAME=" + localInstanceID))
		Expect(flags).To(ContainElement("ETCD_INITIAL_ADVERTISE_PEER_URLS=" + localPeerURL))
		Expect(flags).To(ContainElement("ETCD_LISTEN_PEER_URLS=" + localPeerURL))
		Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=%v,%v", localClientURL, clientURL("127.0.0.1"))))
		Expect(flags).To(ContainElement("ETCD_ADVERTISE_CLIENT_URLS=" + localClientURL))
	})

	It("an existing cluster where a node needs replacing", func() {
		By("Returning some instances including the local instance")
		cloudProvider.MockGetInstances.GetInstancesOutput = []provider.Instance{
			{
				InstanceID: localInstanceID,
				PrivateIP:  localPrivateIP,
			},
			{
				InstanceID: "test-existing-cluster-instance-id-2",
				PrivateIP:  "192.168.0.2",
			},
			{
				InstanceID: "test-existing-cluster-instance-id-3",
				PrivateIP:  "192.168.0.3",
			},
		}

		By("Returning a list of etcd members that contains too many members but does not include the local instance")
		etcdCluster.MockMembers.MembersOutput = []etcd.Member{
			{
				Name:    "test-existing-cluster-old-instance-id-1",
				PeerURL: "http://192.168.0.1:2380",
			},
			{
				Name:    "test-existing-cluster-instance-id-2",
				PeerURL: "http://192.168.0.2:2380",
			},
			{
				Name:    "test-existing-cluster-instance-id-3",
				PeerURL: "http://192.168.0.3:2380",
			},
		}

		By("Expecting a RemoveMember() call to be made with the old instance PeerURL")
		etcdCluster.MockRemoveMember.ExpectedInputs = []string{"http://192.168.0.1:2380"}
		bootstrapperClient := bootstrapper{
			provider: cloudProvider,
			cluster:  etcdCluster,
		}

		etcdFlags, err := bootstrapperClient.GenerateEtcdFlags()
		flags := strings.Split(etcdFlags, "\n")
		Expect(err).To(BeNil())
		Expect(flags).To(ContainElement("ETCD_INITIAL_CLUSTER_STATE=existing"))
		Expect(flags).To(ContainElement("ETCD_INITIAL_CLUSTER=test-existing-cluster-instance-id-2=http://192.168.0.2:2380," +
			"test-existing-cluster-instance-id-3=http://192.168.0.3:2380"))
		Expect(flags).To(ContainElement("ETCD_NAME=" + localInstanceID))
		Expect(flags).To(ContainElement("ETCD_INITIAL_ADVERTISE_PEER_URLS=" + localPeerURL))
		Expect(flags).To(ContainElement("ETCD_LISTEN_PEER_URLS=" + localPeerURL))
		Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=%v,%v", localClientURL, clientURL("127.0.0.1"))))
		Expect(flags).To(ContainElement("ETCD_ADVERTISE_CLIENT_URLS=" + localClientURL))
	})

	It("an existing cluster when partially initialised", func() {
		By("Returning some instances including the local instance")
		cloudProvider.MockGetInstances.GetInstancesOutput = []provider.Instance{
			{
				InstanceID: localInstanceID,
				PrivateIP:  localPrivateIP,
			},
			{
				InstanceID: "test-existing-cluster-partially-initialised-instance-id-1",
				PrivateIP:  "192.168.0.1",
			},
			{
				InstanceID: "test-existing-cluster-partially-initialised-instance-id-2",
				PrivateIP:  "192.168.0.2",
			},
		}

		By("Returning a list of etcd members that contains too many members but does not include the local instance")
		etcdCluster.MockMembers.MembersOutput = []etcd.Member{
			{
				Name:    "",
				PeerURL: localPeerURL,
			},
			{
				Name:    "test-existing-cluster-partially-initialised-instance-id-1",
				PeerURL: "http://192.168.0.1:2380",
			},
			{
				Name:    "test-existing-cluster-partially-initialised-instance-id-2",
				PeerURL: "http://192.168.0.2:2380",
			},
		}
		bootstrapperClient := bootstrapper{
			provider: cloudProvider,
			cluster:  etcdCluster,
		}

		etcdFlags, err := bootstrapperClient.GenerateEtcdFlags()
		flags := strings.Split(etcdFlags, "\n")
		Expect(err).To(BeNil())

		By("Joining the existing cluster as the node has not initialised fully yet")
		Expect(flags).To(ContainElement("ETCD_INITIAL_CLUSTER_STATE=existing"))
		Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_INITIAL_CLUSTER=%v=%v,"+
			"test-existing-cluster-partially-initialised-instance-id-1=http://192.168.0.1:2380,"+
			"test-existing-cluster-partially-initialised-instance-id-2=http://192.168.0.2:2380", localInstanceID, localPeerURL)))
		Expect(flags).To(ContainElement("ETCD_NAME=" + localInstanceID))
		Expect(flags).To(ContainElement("ETCD_INITIAL_ADVERTISE_PEER_URLS=" + localPeerURL))
		Expect(flags).To(ContainElement("ETCD_LISTEN_PEER_URLS=" + localPeerURL))
		Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=%v,%v", localClientURL, clientURL("127.0.0.1"))))
		Expect(flags).To(ContainElement("ETCD_ADVERTISE_CLIENT_URLS=" + localClientURL))
	})
})
