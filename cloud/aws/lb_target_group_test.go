package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/sky-uk/etcd-bootstrap/cloud"
	"github.com/sky-uk/etcd-bootstrap/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	targetGroupName = "test-target-group-name"
	targetGroupARN  = "test-target-group-arn"
)

// TestLBTargetGroupRegistrationProvider to register the test suite
func TestLBTargetGroupRegistrationProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Loadbalancer Target Group Registration Provider")
}

var _ = Describe("Loadbalancer Target Group Registration Provider", func() {
	var elbClient mock.AWSELBClient
	var registrationProvider LBTargetGroupRegistrationProvider

	BeforeEach(func() {
		By("Generating instance arrays based on the test data")
		var elbInstances []*elbv2.TargetDescription
		for _, testInstance := range testInstances {
			elbInstances = append(elbInstances, &elbv2.TargetDescription{
				Id: aws.String(testInstance.Endpoint),
			})
		}

		By("Creating dummy client responses")
		elbClient = mock.AWSELBClient{
			MockDescribeTargetGroups: mock.DescribeTargetGroups{
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
			MockRegisterTargets: mock.RegisterTargets{
				ExpectedInput: &elbv2.RegisterTargetsInput{
					TargetGroupArn: aws.String(targetGroupARN),
					Targets:        elbInstances,
				},
			},
			MockDescribeTargetHealth: mock.DescribeTargetHealth{
				ExpectedInput: &elbv2.DescribeTargetHealthInput{
					TargetGroupArn: aws.String(targetGroupARN),
				},
				DescribeTargetHealthOutput: &elbv2.DescribeTargetHealthOutput{
					TargetHealthDescriptions: []*elbv2.TargetHealthDescription{},
				},
			},
			MockDeregisterTargets: mock.DeregisterTargets{
				ExpectedInput: &elbv2.DeregisterTargetsInput{
					TargetGroupArn: aws.String(targetGroupARN),
					Targets:        []*elbv2.TargetDescription{},
				},
			},
		}
		registrationProvider = LBTargetGroupRegistrationProvider{
			targetGroupName: targetGroupName,
			elb:             elbClient,
		}
	})

	Context("Update()", func() {
		It("passes when DescribeTargetGroups, DescribeTargetHealth, RegisterTargets and DeRegisterTargets return expected values with instances", func() {
			Expect(registrationProvider.Update(testInstances)).To(BeNil())
		})

		It("passes when DescribeTargetGroups, DescribeTargetHealth, RegisterTargets and DeRegisterTargets return expected values with no instances", func() {
			elbClient.MockRegisterTargets.ExpectedInput.Targets = nil
			registrationProvider.elb = elbClient
			Expect(registrationProvider.Update([]cloud.Instance{})).To(BeNil())
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

		It("passes when there are targets to deregister", func() {
			var targetHealthDescriptions []*elbv2.TargetHealthDescription

			for _, testInstance := range testInstances {
				targetHealthDescriptions = append(targetHealthDescriptions, &elbv2.TargetHealthDescription{
					Target: &elbv2.TargetDescription{
						Id: aws.String(testInstance.Endpoint),
					},
				})
			}

			staleTarget := &elbv2.TargetDescription{
				Id: aws.String("10.0.0.254"),
			}

			// add a 'stale' target that will get deregistered
			targetHealthDescriptions = append(targetHealthDescriptions, &elbv2.TargetHealthDescription{
				Target: staleTarget,
			})

			elbClient.MockDescribeTargetHealth.DescribeTargetHealthOutput = &elbv2.DescribeTargetHealthOutput{
				TargetHealthDescriptions: targetHealthDescriptions,
			}

			elbClient.MockDeregisterTargets.ExpectedInput.Targets = []*elbv2.TargetDescription{staleTarget}

			registrationProvider.elb = elbClient
			Expect(registrationProvider.Update(testInstances)).To(BeNil())
		})
	})
})
