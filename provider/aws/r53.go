package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	log "github.com/sirupsen/logrus"
	"github.com/sky-uk/etcd-bootstrap/provider"
)

// Route53RegistrationProviderConfig contains configuration when creating a default Route53RegistrationProvider
type Route53RegistrationProviderConfig struct {
	ZoneID   string
	Hostname string
}

// r53 interface to abstract away from AWS commands
type r53 interface {
	// GetHostedZone gets information about a given hosted zone from the aws route53 client
	GetHostedZone(r *route53.GetHostedZoneInput) (*route53.GetHostedZoneOutput, error)
	// ChangeResourceRecordSets will update a given hosted zone using the aws route53 client
	ChangeResourceRecordSets(r *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error)
}

// Route53RegistrationProvider contains an aws route53 client and information about the desired hosted zone the user
// wants to update
type Route53RegistrationProvider struct {
	zoneID   string
	hostname string
	r53      r53
}

// NewRoute53RegistrationProvider returns a default Route53RegistrationProvider and initiates an new aws route53 client
func NewRoute53RegistrationProvider(c *Route53RegistrationProviderConfig) (provider.RegistrationProvider, error) {
	awsSession, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create new AWS session: %v", err)
	}

	meta := ec2metadata.New(awsSession)
	identityDoc, err := meta.GetInstanceIdentityDocument()
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS local instance data: %v", err)
	}
	config := &aws.Config{Region: aws.String(identityDoc.Region)}
	r53Client := route53.New(awsSession, config)

	return Route53RegistrationProvider{
		zoneID:   c.ZoneID,
		hostname: c.Hostname,
		r53:      r53Client,
	}, nil
}

// Update will update the specified hostname in the route53 zone with discovered etcd ip addresses
func (r Route53RegistrationProvider) Update(instances []provider.Instance) error {
	zoneInput := &route53.GetHostedZoneInput{Id: aws.String(r.zoneID)}
	zone, err := r.r53.GetHostedZone(zoneInput)
	if err != nil {
		return fmt.Errorf("unable to retrieve hosted zone - are you sure it exists?: %v", err)
	}

	fqdn := r.hostname + "." + *zone.HostedZone.Name

	var resourceRecords []*route53.ResourceRecord
	for _, instance := range instances {
		resourceRecords = append(resourceRecords, &route53.ResourceRecord{Value: aws.String(instance.PrivateIP)})
	}

	change := &route53.Change{
		Action: aws.String(route53.ChangeActionUpsert),
		ResourceRecordSet: &route53.ResourceRecordSet{
			Name: aws.String(fqdn),
			Type: aws.String(route53.RRTypeA),
			// Completely arbitrary amount that is not too long or too short.
			TTL:             aws.Int64(300),
			ResourceRecords: resourceRecords,
		},
	}

	changeInput := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: zone.HostedZone.Id,
		ChangeBatch:  &route53.ChangeBatch{Changes: []*route53.Change{change}},
	}

	if _, err := r.r53.ChangeResourceRecordSets(changeInput); err != nil {
		return fmt.Errorf("unable to change resource record set: %v", err)
	}

	log.Infof("Successfully set %q to %v", fqdn, resourceRecords)

	return nil
}
