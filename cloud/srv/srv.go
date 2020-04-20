package srv

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/sky-uk/etcd-bootstrap/cloud"
)

const (
	service = "etcd-bootstrap"
	proto   = "tcp"
	timeout = 5 * time.Second
)

// Members returns the instance information for an etcd cluster.
type Members struct {
	instances []cloud.Instance
}

// Resolver for looking up SRV records.
type Resolver interface {
	// LookupSRV is from net.LookupSRV.
	LookupSRV(ctx context.Context, service, proto, name string) (cname string, addrs []*net.SRV, err error)
}

// Config is the configuration for an SRV lookup.
type Config struct {
	// DomainName is the domain name to use.
	DomainName string
	// Service is the SRV service to use.
	Service string
	// Resolver is the underlying SRV resolver to use.
	// Leave nil to use net.Resolver.
	Resolver Resolver
}

// Lookup uses the SRV record to return the members in an etcd cluster.
func Lookup(conf *Config) (*Members, error) {
	r := conf.Resolver
	if r == nil {
		r = &net.Resolver{}
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	_, addrs, err := r.LookupSRV(ctx, service, proto, conf.DomainName)
	if err != nil {
		return nil, fmt.Errorf("unable to lookup SRV for _%s._%s.%s: %w", service, proto, conf.DomainName, err)
	}
	var instances []cloud.Instance
	for _, addr := range addrs {
		instances = append(instances, cloud.Instance{
			PrivateIP:  addr.Target,
			InstanceID: fmt.Sprintf("etcd-%s", addr.Target),
		})
	}
	return &Members{
		instances: instances,
	}, err
}

// GetInstances returns the instances inside of the SRV record.
func (s *Members) GetInstances() []cloud.Instance {
	return s.instances
}
