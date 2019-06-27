package bootstrap

import (
	"fmt"
	"github.com/sky-uk/etcd-bootstrap/bootstrap/etcd"
	"github.com/sky-uk/etcd-bootstrap/mock"
	"github.com/sky-uk/etcd-bootstrap/provider"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	localPrivateIP  = "10.0.0.1"
	localInstanceID = "test-new-local-instance"
)

var (
	testInstances = []testInstance{
		{
			Name:      "test-instance-id-1",
			IPAddress: "192.168.0.1",
		},
		{
			Name:      "test-instance-id-2",
			IPAddress: "192.168.0.2",
		},
		{
			Name:      "test-instance-id-3",
			IPAddress: "192.168.0.3",
		},
	}
)

type testInstance struct {
	Name      string
	IPAddress string
}

func TestBoostrap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Boostrap")
}

var _ = Describe("Bootstrap", func() {
	var cloudProvider mock.CloudProvider
	var etcdCluster mock.EtcdCluster
	var bootstrapperClient bootstrapper
	var testEtcdMemberList string

	testInstances = append(testInstances, testInstance{Name: localInstanceID, IPAddress: localPrivateIP})

	BeforeEach(func() {
		testEtcdMemberList = ""
		var testProviderInstances []provider.Instance
		var testEtcdMemberInstances []etcd.Member
		for _, testInstance := range testInstances {
			peerURL := fmt.Sprintf("http://%v:2380", testInstance.IPAddress)
			testProviderInstances = append(testProviderInstances, provider.Instance{
				InstanceID: testInstance.Name,
				PrivateIP:  testInstance.IPAddress,
			})
			testEtcdMemberInstances = append(testEtcdMemberInstances, etcd.Member{
				Name:    testInstance.Name,
				PeerURL: peerURL,
			})
			if testEtcdMemberList != "" {
				testEtcdMemberList += ","
			}
			testEtcdMemberList += fmt.Sprintf("%v=%v", testInstance.Name, peerURL)
		}
		cloudProvider = mock.CloudProvider{
			MockGetInstances: mock.MockGetInstances{
				GetInstancesOutput: testProviderInstances,
			},
			MockGetLocalInstance: mock.MockGetLocalInstance{
				GetLocalInstance: provider.Instance{
					InstanceID: localInstanceID,
					PrivateIP:  localPrivateIP,
				},
			},
		}
		etcdCluster = mock.EtcdCluster{
			MockMembers: mock.MockMembers{
				MembersOutput: testEtcdMemberInstances,
			},
			MockAddMember: mock.MockAddMember{
				ExpectedInput: fmt.Sprintf("http://%v:2380", localPrivateIP),
			},
		}
		bootstrapperClient = bootstrapper{
			provider: cloudProvider,
			cluster:  etcdCluster,
		}
	})

	It("new cluster", func() {
		etcdCluster.MockMembers.MembersOutput = []etcd.Member{}
		bootstrapperClient.cluster = etcdCluster
		etcdFlags, err := bootstrapperClient.GenerateEtcdFlags()
		flags := strings.Split(etcdFlags, "\n")
		Expect(err).To(BeNil())
		Expect(flags).To(ContainElement("ETCD_INITIAL_CLUSTER_STATE=new"))
		Expect(flags).To(ContainElement("ETCD_INITIAL_CLUSTER=" + testEtcdMemberList))
		Expect(flags).To(ContainElement("ETCD_NAME=" + localInstanceID))
		Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_INITIAL_ADVERTISE_PEER_URLS=http://%v:2380", localPrivateIP)))
		Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_LISTEN_PEER_URLS=http://%v:2380", localPrivateIP)))
		Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=http://%v:2379,http://127.0.0.1:2379", localPrivateIP)))
		Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_ADVERTISE_CLIENT_URLS=http://%v:2379", localPrivateIP)))
	})

	It("existing cluster", func() {
		etcdFlags, err := bootstrapperClient.GenerateEtcdFlags()
		print(etcdFlags)
		flags := strings.Split(etcdFlags, "\n")
		Expect(err).To(BeNil())
		Expect(flags).To(ContainElement("ETCD_INITIAL_CLUSTER_STATE=new"))
		Expect(flags).To(ContainElement("ETCD_INITIAL_CLUSTER=" + testEtcdMemberList))
		Expect(flags).To(ContainElement("ETCD_NAME=" + localInstanceID))
		Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_INITIAL_ADVERTISE_PEER_URLS=http://%v:2380", localPrivateIP)))
		Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_LISTEN_PEER_URLS=http://%v:2380", localPrivateIP)))
		Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=http://%v:2379,http://127.0.0.1:2379", localPrivateIP)))
		Expect(flags).To(ContainElement(fmt.Sprintf("ETCD_ADVERTISE_CLIENT_URLS=http://%v:2379", localPrivateIP)))
	})
})
