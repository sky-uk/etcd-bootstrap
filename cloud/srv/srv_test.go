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
	receivedTXTName                              string
	sentTXT                                      map[string][]string
	sentTXTerr                                   error
}

func (r *stubResolver) LookupSRV(ctx context.Context, service, proto, name string) (cname string, addrs []*net.SRV, err error) {
	r.receivedService = service
	r.receivedProto = proto
	r.receivedName = name
	return r.sentCname, r.sentAddrs, r.sentErr
}

func (r *stubResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	r.receivedTXTName = name
	return r.sentTXT[name], r.sentTXTerr
}

var _ = Describe("SRV Instances", func() {
	var (
		addrs    []*net.SRV
		sentTXT  map[string][]string
		resolver *stubResolver
		conf     *Config
	)

	BeforeEach(func() {
		addrs = []*net.SRV{
			&net.SRV{
				Target: "etcd-1",
			},
			&net.SRV{
				Target: "etcd-2",
			},
			&net.SRV{
				Target: "etcd-3",
			},
		}
		sentTXT = make(map[string][]string)
		sentTXT["etcd-1"] = []string{"bogus", "boz=woz", "name=i-abc1", "gbg=rrr"}
		sentTXT["etcd-2"] = []string{"name=i-abc2"}
		sentTXT["etcd-3"] = []string{"name=i-abc3"}
	})

	JustBeforeEach(func() {
		resolver = &stubResolver{
			sentAddrs: addrs,
			sentTXT:   sentTXT,
		}
		conf = &Config{
			DomainName: "my-etcd-cluster.example.com",
			Resolver:   resolver,
			Service:    "etcd-server-ssl",
		}
	})

	It("should request the correct SRV record", func() {
		_, err := New(conf).GetInstances()
		Expect(err).To(Succeed())

		Expect(resolver.receivedService).To(Equal("etcd-server-ssl"))
		Expect(resolver.receivedProto).To(Equal("tcp"))
		Expect(resolver.receivedName).To(Equal("my-etcd-cluster.example.com"))
	})

	It("should return the instances in the SRV record", func() {
		srv := New(conf)
		instances, err := srv.GetInstances()
		Expect(err).To(Succeed())

		Expect(instances).To(HaveLen(3))
		Expect(instances[0].PrivateIP).To(Equal("etcd-1"))
		Expect(instances[1].PrivateIP).To(Equal("etcd-2"))
		Expect(instances[2].PrivateIP).To(Equal("etcd-3"))
	})

	It("should return unique instance IDs", func() {
		srv := New(conf)
		instances, err := srv.GetInstances()
		Expect(err).To(Succeed())

		Expect(instances).To(HaveLen(3))
		Expect(instances[0].InstanceID).To(Equal("i-abc1"))
		Expect(instances[1].InstanceID).To(Equal("i-abc2"))
		Expect(instances[2].InstanceID).To(Equal("i-abc3"))
	})
})
