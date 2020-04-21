package srv

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sky-uk/etcd-bootstrap/cloud"
)

const (
	proto   = "tcp"
	timeout = 5 * time.Second
)

// Members returns the instance information for an etcd cluster.
type Members struct {
	Config
	instances []cloud.Instance
}

// Resolver for looking up SRV records and their associated TXT record.
type Resolver interface {
	// LookupSRV is from net.LookupSRV.
	LookupSRV(ctx context.Context, service, proto, name string) (cname string, addrs []*net.SRV, err error)
	// LookupTXT is from net.LookupTXT.
	LookupTXT(ctx context.Context, name string) ([]string, error)
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

// New returns members that will use the SRV record to look themselves up.
func New(conf *Config) *Members {
	m := &Members{Config: *conf}
	if m.Resolver == nil {
		m.Resolver = &net.Resolver{}
	}
	return m
}

// GetInstances returns the instances inside of the SRV record.
func (m *Members) GetInstances() ([]cloud.Instance, error) {
	if m.instances == nil {
		ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
		defer cancelFn()
		_, addrs, err := m.Resolver.LookupSRV(ctx, m.Service, proto, m.DomainName)
		if err != nil {
			return nil, fmt.Errorf("unable to lookup SRV for _%s._%s.%s: %w", m.Service, proto, m.DomainName, err)
		}
		var instances []cloud.Instance
		for _, addr := range addrs {
			name, err := m.lookupTXTName(addr.Target)
			if err != nil {
				return nil, fmt.Errorf("unable to lookup instance name for SRV target %s: %w", addr.Target, err)
			}
			instances = append(instances, cloud.Instance{
				PrivateIP:  addr.Target,
				InstanceID: name,
			})
		}
		m.instances = instances
	}
	return m.instances, nil
}

// lookupTXTName looks for the name associated with the target, using RFC1464 conventions.
func (m *Members) lookupTXTName(target string) (string, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	records, err := m.Resolver.LookupTXT(ctx, target)
	if err != nil {
		return "", err
	}
	for _, record := range records {
		split := strings.SplitN(record, "=", 2)
		if len(split) != 2 {
			// No '=' so skip.
			continue
		}
		if split[0] == "name" {
			return split[1], nil
		}
	}
	return "", fmt.Errorf("no TXT record with `name=` attribute found for %s", target)
}
