package aws

import (
	log "github.com/Sirupsen/logrus"

	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

// R53 abstracts interactions with AWS Route53.
type R53 interface {
	// UpdateARecords updates the A record for the given name with the provided IPs.
	UpdateARecords(zoneID, name string, values []string) error
}

type impl struct {
	r53 *route53.Route53
}

// UpdateARecords will update the specified hostname in the route53 zone with discovered etcd ip addresses
func (r *impl) UpdateARecords(zoneID, name string, values []string) error {
	zoneInput := &route53.GetHostedZoneInput{Id: aws.String(zoneID)}
	zone, err := r.r53.GetHostedZone(zoneInput)
	if err != nil {
		return fmt.Errorf("unable to retrieve hosted zone - are you sure it exists?: %v", err)
	}

	fqdn := name + "." + *zone.HostedZone.Name

	var resourceRecords []*route53.ResourceRecord
	for _, value := range values {
		resourceRecords = append(resourceRecords,
			&route53.ResourceRecord{Value: aws.String(value)})
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

	log.Infof("Successfully set %q to %v", fqdn, values)

	return nil
}

func newR53() (R53, error) {
	awsSession, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	meta := ec2metadata.New(awsSession)
	identityDoc, err := meta.GetInstanceIdentityDocument()
	if err != nil {
		return nil, err
	}
	config := &aws.Config{Region: aws.String(identityDoc.Region)}

	awsRoute53 := route53.New(awsSession, config)
	return &impl{r53: awsRoute53}, nil
}
