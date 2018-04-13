package gcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/sky-uk/etcd-bootstrap/lib/cloud"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
)

// GDNS abstracts interactions with GCP dns.
type GDNS interface {
	// UpdateARecords updates the A record for the given name with the provided IPs.
	UpdateARecords(project string, zoneID, name string, values []string) error
}

type impl struct {
	gdns *dns.Service
}

// newGDNS create a new GDNS
func newGDNS(cfg *Config) (GDNS, error) {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	c, err := newClientDNS(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create GCP compute DNS API client: %v", err)
	}

	return &impl{gdns: c}, nil
}

func newClientDNS(ctx context.Context, cfg *Config) (*dns.Service, error) {
	client, err := google.DefaultClient(ctx, dns.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	dnsService, err := dns.New(client)
	if err != nil {
		return nil, err
	}
	return dnsService, err
}

func (r *impl) UpdateARecords(project string, managedZone string, name string, values []string) error {

	fqdn := name + "." + managedZone

	recordSetUdpate := &ResourceRecordSet{
		Name:    fqdn,
		Type:    "A",
		Ttl:     300,
		Rrdatas: values,
	}

	dnsChanges := &dns.Change{
		Additions: recordSetUdpate,
	}

	resp, err := dns.Changes.Create(project, managedZone, dnsChanges).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("unable to update record : %v", err)
	}

	log.Infof("Successfully set %q to %v", fqdn, values)

	return nil
}
