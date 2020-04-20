package srv

import (
	"context"
	"net"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSRV(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SRV Suite")
}

type stubResolver struct {
	receivedService, receivedProto, receivedName string
	sentCname                                    string
	sentAddrs                                    []*net.SRV
	sentErr                                      error
}

func (r *stubResolver) LookupSRV(ctx context.Context, service, proto, name string) (cname string, addrs []*net.SRV, err error) {
	r.receivedService = service
	r.receivedProto = proto
	r.receivedName = name
	return r.sentCname, r.sentAddrs, r.sentErr
}

var _ = Describe("SRV Instances", func() {
	var (
		addrs    []*net.SRV
		resolver *stubResolver
		conf     *Config
	)

	BeforeEach(func() {
		addrs = []*net.SRV{
			&net.SRV{
				Target: "10.10.10.1",
			},
			&net.SRV{
				Target: "10.10.10.2",
			},
			&net.SRV{
				Target: "10.10.10.3",
			},
		}
	})

	JustBeforeEach(func() {
		resolver = &stubResolver{
			sentAddrs: addrs,
		}
		conf = &Config{
			DomainName: "my-etcd-cluster.example.com",
			Resolver:   resolver,
			Service:    "etcd-server-ssl",
		}
	})

	It("should request the correct SRV record", func() {
		New(conf).GetInstances()

		Expect(resolver.receivedService).To(Equal("etcd-server-ssl"))
		Expect(resolver.receivedProto).To(Equal("tcp"))
		Expect(resolver.receivedName).To(Equal("my-etcd-cluster.example.com"))
	})

	It("should return the instances in the SRV record", func() {
		srv := New(conf)
		instances, _ := srv.GetInstances()

		Expect(instances).To(HaveLen(3))
		Expect(instances[0].PrivateIP).To(Equal("10.10.10.1"))
		Expect(instances[1].PrivateIP).To(Equal("10.10.10.2"))
		Expect(instances[2].PrivateIP).To(Equal("10.10.10.3"))
	})

	It("should return unique instance IDs", func() {
		srv := New(conf)
		instances, _ := srv.GetInstances()

		Expect(instances).To(HaveLen(3))
		Expect(instances[0].InstanceID).To(Equal("etcd-10.10.10.1"))
		Expect(instances[1].InstanceID).To(Equal("etcd-10.10.10.2"))
		Expect(instances[2].InstanceID).To(Equal("etcd-10.10.10.3"))
	})
})
