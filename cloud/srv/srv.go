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

// SRV returns the instance information for an etcd cluster using an SRV record.
type SRV struct {
	domainName    string
	service       string
	localResolver LocalResolver
	resolver      resolver
	instances     []cloud.Instance
	localInstance *cloud.Instance
}

// LocalResolver finds the IP address associated with the local instance.
type LocalResolver interface {
	LookupLocalIP() (net.IP, error)
}

type resolver interface {
	// LookupSRV is from net.LookupSRV.
	LookupSRV(ctx context.Context, service, proto, name string) (cname string, addrs []*net.SRV, err error)
	// LookupTXT is from net.LookupTXT.
	LookupTXT(ctx context.Context, name string) ([]string, error)
	// LookupHost is from net.LookupHost.
	LookupHost(ctx context.Context, host string) (addrs []string, err error)
}

// New returns a struct that will use the SRV record to look up etcd instances.
func New(domainName, service string, localResolver LocalResolver) *SRV {
	return &SRV{
		domainName:    domainName,
		service:       service,
		localResolver: localResolver,
		resolver:      &net.Resolver{},
	}
}

// GetInstances returns the instances inside of the SRV record.
func (s *SRV) GetInstances() ([]cloud.Instance, error) {
	if s.instances == nil {
		ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
		defer cancelFn()
		_, addrs, err := s.resolver.LookupSRV(ctx, s.service, proto, s.domainName)
		if err != nil {
			return nil, fmt.Errorf("unable to lookup SRV for _%s._%s.%s: %w", s.service, proto, s.domainName, err)
		}
		var instances []cloud.Instance
		for _, addr := range addrs {
			name, err := s.lookupTXTName(addr.Target)
			if err != nil {
				return nil, fmt.Errorf("unable to lookup instance name for SRV target %s: %w", addr.Target, err)
			}
			instances = append(instances, cloud.Instance{
				Endpoint: addr.Target,
				Name:     name,
			})
		}
		s.instances = instances
	}
	return s.instances, nil
}

// lookupTXTName looks for the name associated with the target, using RFC1464 conventions.
func (s *SRV) lookupTXTName(target string) (string, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
	defer cancelFn()
	records, err := s.resolver.LookupTXT(ctx, target)
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

func (s *SRV) lookupInstanceAddresses(instances []cloud.Instance) (map[cloud.Instance][]string, error) {
	instanceAddrs := make(map[cloud.Instance][]string)
	for _, instance := range instances {
		ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
		defer cancelFn()
		addrs, err := s.resolver.LookupHost(ctx, instance.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve target %s: %w", instance, err)
		}
		instanceAddrs[instance] = addrs
	}
	return instanceAddrs, nil
}

func (s *SRV) findLocalInstance() (cloud.Instance, error) {
	localIP, err := s.localResolver.LookupLocalIP()
	if err != nil {
		return cloud.Instance{}, fmt.Errorf("unable to lookup local IP: %w", err)
	}
	instances, err := s.GetInstances()
	if err != nil {
		return cloud.Instance{}, err
	}
	instanceAddrs, err := s.lookupInstanceAddresses(instances)
	if err != nil {
		return cloud.Instance{}, fmt.Errorf("unable to lookup SRV targets: %w", err)
	}

	for instance, addrs := range instanceAddrs {
		for _, addr := range addrs {
			if localIP.Equal(net.ParseIP(addr)) {
				return instance, nil
			}
		}
	}
	return cloud.Instance{}, fmt.Errorf("none of the SRV targets %v resolve to local IP %v", instanceAddrs, localIP)
}

// GetLocalInstance will return the unique name and endpoint of the local instance using the SRV record.
func (s *SRV) GetLocalInstance() (cloud.Instance, error) {
	if s.localInstance == nil {
		instance, err := s.findLocalInstance()
		if err != nil {
			return cloud.Instance{}, err
		}
		s.localInstance = &instance
	}
	return *s.localInstance, nil
}
