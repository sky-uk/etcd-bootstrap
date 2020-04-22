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

var _ = Describe("SRV Instances", func() {
	var (
		domainName, service string
		addrs               []*net.SRV
		sentTXTs            map[string][]string
		sentHostAddrs       map[string][]string
		resolver            *stubResolver
		localResolver       *stubLocalResolver
		srv                 *SRV
	)

	BeforeEach(func() {
		domainName = "my-etcd-cluster.example.com"
		service = "etcd-server-ssl"
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
		sentTXTs = make(map[string][]string)
		sentTXTs["etcd-1"] = []string{"bogus", "boz=woz", "name=i-abc1", "gbg=rrr"}
		sentTXTs["etcd-2"] = []string{"name=i-abc2"}
		sentTXTs["etcd-3"] = []string{"name=i-abc3"}
		sentHostAddrs = make(map[string][]string)
		sentHostAddrs["etcd-1"] = []string{"10.10.10.1"}
		sentHostAddrs["etcd-2"] = []string{"172.17.1.2", "10.10.10.2"}
		sentHostAddrs["etcd-3"] = []string{"10.10.10.3"}
	})

	JustBeforeEach(func() {
		resolver = &stubResolver{
			sentAddrs:     addrs,
			sentTXTs:      sentTXTs,
			sentHostAddrs: sentHostAddrs,
		}
		localResolver = &stubLocalResolver{
			sentIP: net.ParseIP("10.10.10.2"),
		}
		srv = New(domainName, service, localResolver)
		srv.resolver = resolver
	})

	It("should request the correct SRV record", func() {
		_, err := srv.GetInstances()
		Expect(err).To(Succeed())
		Expect(resolver.receivedService).To(Equal("etcd-server-ssl"))
		Expect(resolver.receivedProto).To(Equal("tcp"))
		Expect(resolver.receivedName).To(Equal("my-etcd-cluster.example.com"))
	})

	It("should return the instances in the SRV record", func() {
		instances, err := srv.GetInstances()
		Expect(err).To(Succeed())
		Expect(instances).To(HaveLen(3))
		Expect(instances[0].Endpoint).To(Equal("etcd-1"))
		Expect(instances[1].Endpoint).To(Equal("etcd-2"))
		Expect(instances[2].Endpoint).To(Equal("etcd-3"))
	})

	It("should return unique instance IDs", func() {
		instances, err := srv.GetInstances()
		Expect(err).To(Succeed())
		Expect(instances).To(HaveLen(3))
		Expect(instances[0].Name).To(Equal("i-abc1"))
		Expect(instances[1].Name).To(Equal("i-abc2"))
		Expect(instances[2].Name).To(Equal("i-abc3"))
	})

	It("should discover its local instance information via the SRV record", func() {
		local, err := srv.GetLocalInstance()
		Expect(err).To(Succeed())
		Expect(local.Name).To(Equal("i-abc2"))
		Expect(local.Endpoint).To(Equal("etcd-2"))
	})
})

type stubResolver struct {
	receivedService, receivedProto, receivedName string
	sentCname                                    string
	sentAddrs                                    []*net.SRV
	sentErr                                      error
	sentTXTs                                     map[string][]string
	sentTXTerr                                   error
	sentHostAddrs                                map[string][]string
	sentHostErr                                  error
}

func (r *stubResolver) LookupSRV(ctx context.Context, service, proto, name string) (cname string, addrs []*net.SRV, err error) {
	r.receivedService = service
	r.receivedProto = proto
	r.receivedName = name
	return r.sentCname, r.sentAddrs, r.sentErr
}

func (r *stubResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return r.sentTXTs[name], r.sentTXTerr
}

func (r *stubResolver) LookupHost(ctx context.Context, host string) (addrs []string, err error) {
	return r.sentHostAddrs[host], r.sentHostErr
}

type stubLocalResolver struct {
	sentIP  net.IP
	sentErr error
}

func (r *stubLocalResolver) LookupLocalIP() (net.IP, error) {
	return r.sentIP, r.sentErr
}
