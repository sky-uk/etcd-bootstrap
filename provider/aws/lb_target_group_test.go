package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/sky-uk/etcd-bootstrap/mock"
	"github.com/sky-uk/etcd-bootstrap/provider"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	targetGroupName = "test-target-group-name"
	targetGroupARN  = "test-target-group-arn"
)

func TestLBTargetGroupRegistrationProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Loadbalancer Target Group Registration Provider")
}

var _ = Describe("Loadbalancer Target Group Registration Provider", func() {
	var elbClient mock.AWSELBClient
	var registrationProvider LBTargetGroupRegistrationProvider

	BeforeEach(func() {
		// generate instance arrays based on the test data
		var elbInstances []*elbv2.TargetDescription
		for _, testInstance := range testInstances {
			elbInstances = append(elbInstances, &elbv2.TargetDescription{
				Id: aws.String(testInstance.InstanceID),
			})
		}
		// create dummy client responses
		elbClient = mock.AWSELBClient{
			MockDescribeTargetGroups: mock.MockDescribeTargetGroups{
				ExpectedInput: &elbv2.DescribeTargetGroupsInput{
					Names: []*string{
						aws.String(targetGroupName),
					},
				},
				DescribeTargetGroupsOutput: &elbv2.DescribeTargetGroupsOutput{
					TargetGroups: []*elbv2.TargetGroup{{
						TargetGroupArn: aws.String(targetGroupARN),
					}},
				},
			},
			MockRegisterTargets: mock.MockRegisterTargets{
				ExpectedInput: &elbv2.RegisterTargetsInput{
					TargetGroupArn: aws.String(targetGroupARN),
					Targets:        elbInstances,
				},
			},
		}
		registrationProvider = LBTargetGroupRegistrationProvider{
			targetGroupName: targetGroupName,
			elb:             elbClient,
		}
	})

	Context("Update()", func() {
		It("passes when DescribeTargetGroups and RegisterTargets return expected values with instances", func() {
			Expect(registrationProvider.Update(testInstances)).To(BeNil())
		})

		It("passes when DescribeTargetGroups and RegisterTargets return expected values with no instances", func() {
			elbClient.MockRegisterTargets.ExpectedInput.Targets = nil
			registrationProvider.elb = elbClient
			Expect(registrationProvider.Update([]provider.Instance{})).To(BeNil())
		})

		It("fails when DescribeTargetGroups errors", func() {
			elbClient.MockDescribeTargetGroups.Err = fmt.Errorf("failed to describe target group")
			registrationProvider.elb = elbClient
			Expect(registrationProvider.Update(testInstances)).ToNot(BeNil())
		})

		It("fails when there are more than 1 target groups returned", func() {
			elbClient.MockDescribeTargetGroups.DescribeTargetGroupsOutput = &elbv2.DescribeTargetGroupsOutput{
				TargetGroups: []*elbv2.TargetGroup{{}, {}},
			}
			registrationProvider.elb = elbClient
			Expect(registrationProvider.Update(testInstances)).ToNot(BeNil())
		})

		It("fails when there are 0 target groups returned", func() {
			elbClient.MockDescribeTargetGroups.DescribeTargetGroupsOutput = &elbv2.DescribeTargetGroupsOutput{
				TargetGroups: []*elbv2.TargetGroup{},
			}
			registrationProvider.elb = elbClient
			Expect(registrationProvider.Update(testInstances)).ToNot(BeNil())
		})

		It("fails when RegisterTargets errors", func() {
			elbClient.MockRegisterTargets.Err = fmt.Errorf("failed to register targets")
			registrationProvider.elb = elbClient
			Expect(registrationProvider.Update(testInstances)).ToNot(BeNil())
		})
	})
})
