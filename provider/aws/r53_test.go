package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/sky-uk/etcd-bootstrap/provider"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	hostedZoneID      = "test-hosted-zone-id"
	hostedZoneName    = "test.hosted.zone."
	hostname          = "my-test-etcd-cluster"
)

func TestRoute53RegistrationProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Loadbalancer Target Group Registration Provider")
}

func (t testR53Client) GetHostedZone(r *route53.GetHostedZoneInput) (*route53.GetHostedZoneOutput, error) {
	Expect(r).To(Equal(t.mockGetHostedZone.expectedInput))
	return t.mockGetHostedZone.getHostedZoneOutput, t.mockGetHostedZone.err
}

func (t testR53Client) ChangeResourceRecordSets(r *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error) {
	Expect(r).To(Equal(t.mockChangeResourceRecordSets.expectedInput))
	return t.mockChangeResourceRecordSets.changeResourceRecordSetsOutput, t.mockChangeResourceRecordSets.err
}

type testR53Client struct {
	mockGetHostedZone            mockGetHostedZone
	mockChangeResourceRecordSets mockChangeResourceRecordSets
}

type mockGetHostedZone struct {
	expectedInput       *route53.GetHostedZoneInput
	getHostedZoneOutput *route53.GetHostedZoneOutput
	err                 error
}

type mockChangeResourceRecordSets struct {
	expectedInput                  *route53.ChangeResourceRecordSetsInput
	changeResourceRecordSetsOutput *route53.ChangeResourceRecordSetsOutput
	err                            error
}

var _ = Describe("Route53 Registration Provider", func() {
	var r53Client testR53Client
	var registrationProvider Route53RegistrationProvider

	BeforeEach(func() {
		// generate instance arrays based on the test data
		var route53Instances []*route53.ResourceRecord
		for _, testInstance := range testInstances {
			route53Instances = append(route53Instances, &route53.ResourceRecord{
				Value: aws.String(testInstance.PrivateIP),
			})
		}
		// create dummy client responses
		r53Client = testR53Client{
			mockGetHostedZone: mockGetHostedZone{
				expectedInput: &route53.GetHostedZoneInput{
					Id: aws.String(hostedZoneID),
				},
				getHostedZoneOutput: &route53.GetHostedZoneOutput{
					HostedZone: &route53.HostedZone{
						Id:   aws.String(hostedZoneID),
						Name: aws.String(hostedZoneName),
					},
				},
			},
			mockChangeResourceRecordSets: mockChangeResourceRecordSets{
				expectedInput: &route53.ChangeResourceRecordSetsInput{
					ChangeBatch: &route53.ChangeBatch{
						Changes: []*route53.Change{
							{
								Action: aws.String(route53.ChangeActionUpsert),
								ResourceRecordSet: &route53.ResourceRecordSet{
									Name: aws.String(fmt.Sprintf("%v.%v", hostname, hostedZoneName)),
									Type: aws.String(route53.RRTypeA),
									TTL:  aws.Int64(300),
									ResourceRecords: route53Instances,
								},
							},
						},
					},
					HostedZoneId: aws.String(hostedZoneID),
				},
			},
		}
		registrationProvider = Route53RegistrationProvider{
			zoneID:   hostedZoneID,
			hostname: hostname,
			r53:      r53Client,
		}
	})

	Context("Update()", func() {
		It("passes when DescribeTargetGroups and RegisterTargets return expected values with instances", func() {
			Expect(registrationProvider.Update(testInstances)).To(BeNil())
		})

		It("passes when GetHostedZone and ChangeResourceRecordSets return expected values with no instances", func() {
			r53Client.mockChangeResourceRecordSets.expectedInput.ChangeBatch.Changes = []*route53.Change{
				{
					Action: aws.String(route53.ChangeActionUpsert),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(fmt.Sprintf("%v.%v", hostname, hostedZoneName)),
						Type: aws.String(route53.RRTypeA),
						TTL: aws.Int64(300),
					},
				},
			}
			Expect(registrationProvider.Update([]provider.Instance{})).To(BeNil())
		})

		It("fails when GetHostedZone returns an error", func() {
			r53Client.mockGetHostedZone.err = fmt.Errorf("failed to get hosted zones")
			registrationProvider.r53 = r53Client
			Expect(registrationProvider.Update(testInstances))
		})

		It("fails when ChangeResourceRecordSets returns an error", func() {
			r53Client.mockChangeResourceRecordSets.err = fmt.Errorf("failed to get change resource record set")
			registrationProvider.r53 = r53Client
			Expect(registrationProvider.Update(testInstances))
		})
	})
})
