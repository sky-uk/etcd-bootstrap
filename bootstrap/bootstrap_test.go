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
		cloudProvider = mock.CloudProvider{
			// GetLocalInstance() will always return the constant values
			MockGetLocalInstance: mock.GetLocalInstance{
				GetLocalInstance: provider.Instance{
					InstanceID: localInstanceID,
					PrivateIP:  localPrivateIP,
				},
			},
		}
		etcdCluster = mock.EtcdCluster{
			// AddMember() is only ever called using the local instance values which are constants
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
		// return some instances including the local instance
		cloudProvider.MockGetInstances.GetInstancesOutput = []provider.Instance{
			{
				InstanceID: localInstanceID,
				PrivateIP:  localPrivateIP,
			},
		}
		// error when trying to get etcd members
		etcdCluster.MockMembers.Err = fmt.Errorf("failed to get etcd members")
		bootstrapperClient := bootstrapper{
			provider: cloudProvider,
			cluster:  etcdCluster,
		}

		_, err := bootstrapperClient.GenerateEtcdFlags()
		Expect(err).ToNot(BeNil())
	})

	It("does not fail when it cannot remove member", func() {
		// return some instances including the local instance
		cloudProvider.MockGetInstances.GetInstancesOutput = []provider.Instance{
			{
				InstanceID: localInstanceID,
				PrivateIP:  localPrivateIP,
			},
		}
		// return a list of etcd members that needs to be updated
		etcdCluster.MockMembers.MembersOutput = []etcd.Member{
			{
				Name:    "test-remove-instance-id-1",
				PeerURL: "http://192.168.0.1:2380",
			},
		}
		// expect to get the remove member call
		etcdCluster.MockRemoveMember.ExpectedInputs = []string{"http://192.168.0.1:2380"}
		// error when trying to remove an etcd member
		etcdCluster.MockRemoveMember.Err = fmt.Errorf("failed to remove etcd members")
		bootstrapperClient := bootstrapper{
			provider: cloudProvider,
			cluster:  etcdCluster,
		}

		_, err := bootstrapperClient.GenerateEtcdFlags()
		// do not fail as it may be to do with etcd quorum
		Expect(err).To(BeNil())
	})

	It("fails when it cannot add etcd member", func() {
		// return some instances including the local instance
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
		// return a list of etcd members that needs to be updated
		etcdCluster.MockMembers.MembersOutput = []etcd.Member{
			{
				Name:    "test-add-instance-id-1",
				PeerURL: "http://192.168.0.1:2380",
			},
		}
		// error when trying to get etcd members
		etcdCluster.MockAddMember.Err = fmt.Errorf("failed to add etcd member")
		bootstrapperClient := bootstrapper{
			provider: cloudProvider,
			cluster:  etcdCluster,
		}

		_, err := bootstrapperClient.GenerateEtcdFlags()
		Expect(err).ToNot(BeNil())
	})

	It("new cluster", func() {
		// return some instances including the local instance
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
		// return an empty etcd cluster member list
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
		// return some instances including the local instance
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
		// return a cluster will one too many instance and lacking the local instance
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
		// return some instances including the local instance
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
		// return a cluster will one too many instance and lacking the local instance
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
		// expect RemoveMember() to be called with the old instance peerURL
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
		// return some instances including the local instance
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
		// return a cluster will one too many instance and lacking the local instance
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
		// Should join existing cluster as it hasn't initialised yet (name is blank), despite having its peerURL already added.
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
