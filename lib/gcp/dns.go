package gcp

import (
	"context"
	"fmt"
	"reflect"

	log "github.com/Sirupsen/logrus"
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
func newGDNS() (GDNS, error) {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	c, err := newClientDNS(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to create GCP compute DNS API client: %v", err)
	}

	return &impl{gdns: c}, nil
}

func newClientDNS(ctx context.Context) (*dns.Service, error) {
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
	mz, err := r.gdns.ManagedZones.Get(project, managedZone).Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve the managed zone %q details: %v", managedZone, err)
	}

	fqdn := name + "." + mz.DnsName
	resp, err := r.gdns.ResourceRecordSets.List(project, managedZone).Name(fqdn).Do()
	if err != nil {
		return fmt.Errorf("unable to retrive the recordsets for zone %q: %v", managedZone, err)
	}

	recordSetUdpate := &dns.ResourceRecordSet{
		Name:    fqdn,
		Type:    "A",
		Ttl:     300,
		Rrdatas: values,
	}

	var deletions []*dns.ResourceRecordSet
	if len(resp.Rrsets) > 0 && !reflect.DeepEqual(recordSetUdpate, resp.Rrsets[0]) {
		deletions = resp.Rrsets
	}
	var dnsChanges *dns.Change
	if len(resp.Rrsets) == 0 || len(deletions) > 0 {
		// Upserts are additions + deletions
		dnsChanges = &dns.Change{
			Additions: []*dns.ResourceRecordSet{recordSetUdpate},
			Deletions: deletions,
		}
		_, err = r.gdns.Changes.Create(project, managedZone, dnsChanges).Context(context.Background()).Do()
		if err != nil {
			return fmt.Errorf("unable to update record : %v", err)
		}
		log.Infof("Successfully set %s to %v", fqdn, values)
	}
	return nil
}
