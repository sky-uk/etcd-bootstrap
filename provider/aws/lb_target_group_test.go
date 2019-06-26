package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/sky-uk/etcd-bootstrap/provider"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	targetGroupName             = "test-target-group-name"
	targetGroupARN              = "test-target-group-arn"
	lbTargetGroupTestInstanceID = "test-instance-id"
)

var (
	lbTargetGroupInstances = []provider.Instance{{
		InstanceID: lbTargetGroupTestInstanceID,
	}}
)

func TestLBTargetGroupRegistrationProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Loadbalancer Target Group Registration Provider")
}

func (t testELBClient) DescribeTargetGroups(e *elbv2.DescribeTargetGroupsInput) (*elbv2.DescribeTargetGroupsOutput, error) {
	Expect(e).To(Equal(t.mockDescribeTargetGroups.expectedInput))
	return t.mockDescribeTargetGroups.describeTargetGroupsOutput, t.mockDescribeTargetGroups.err
}

func (t testELBClient) RegisterTargets(e *elbv2.RegisterTargetsInput) (*elbv2.RegisterTargetsOutput, error) {
	Expect(e).To(Equal(t.mockRegisterTargets.expectedInput))
	return t.mockRegisterTargets.registerTargetsOutput, t.mockRegisterTargets.err
}

type testELBClient struct {
	mockDescribeTargetGroups mockDescribeTargetGroups
	mockRegisterTargets      mockRegisterTargets
}

type mockDescribeTargetGroups struct {
	expectedInput              *elbv2.DescribeTargetGroupsInput
	describeTargetGroupsOutput *elbv2.DescribeTargetGroupsOutput
	err                        error
}

type mockRegisterTargets struct {
	expectedInput         *elbv2.RegisterTargetsInput
	registerTargetsOutput *elbv2.RegisterTargetsOutput
	err                   error
}

var _ = Describe("Loadbalancer Target Group Registration Provider", func() {
	var elbClient testELBClient
	var registrationProvider LBTargetGroupRegistrationProvider

	BeforeEach(func() {
		elbClient = testELBClient{
			mockDescribeTargetGroups: mockDescribeTargetGroups{
				expectedInput: &elbv2.DescribeTargetGroupsInput{
					Names: []*string{
						aws.String(targetGroupName),
					},
				},
				describeTargetGroupsOutput: &elbv2.DescribeTargetGroupsOutput{
					TargetGroups: []*elbv2.TargetGroup{{
						TargetGroupArn: aws.String(targetGroupARN),
					}},
				},
			},
			mockRegisterTargets: mockRegisterTargets{
				expectedInput: &elbv2.RegisterTargetsInput{
					TargetGroupArn: aws.String(targetGroupARN),
					Targets: []*elbv2.TargetDescription{
						{
							Id: aws.String(lbTargetGroupTestInstanceID),
						},
					},
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
			Expect(registrationProvider.Update(lbTargetGroupInstances)).To(BeNil())
		})

		It("passes when DescribeTargetGroups and RegisterTargets return expected values with no instances", func() {
			elbClient.mockRegisterTargets.expectedInput.Targets = nil
			registrationProvider.elb = elbClient
			Expect(registrationProvider.Update([]provider.Instance{})).To(BeNil())
		})

		It("fails when DescribeTargetGroups errors", func() {
			elbClient.mockDescribeTargetGroups.err = fmt.Errorf("failed to describe target group")
			registrationProvider.elb = elbClient
			Expect(registrationProvider.Update(lbTargetGroupInstances)).ToNot(BeNil())
		})

		It("fails when there are more than 1 target groups returned", func() {
			elbClient.mockDescribeTargetGroups.describeTargetGroupsOutput = &elbv2.DescribeTargetGroupsOutput{
				TargetGroups: []*elbv2.TargetGroup{{}, {}},
			}
			registrationProvider.elb = elbClient
			Expect(registrationProvider.Update(lbTargetGroupInstances)).ToNot(BeNil())
		})

		It("fails when there are 0 target groups returned", func() {
			elbClient.mockDescribeTargetGroups.describeTargetGroupsOutput = &elbv2.DescribeTargetGroupsOutput{
				TargetGroups: []*elbv2.TargetGroup{},
			}
			registrationProvider.elb = elbClient
			Expect(registrationProvider.Update(lbTargetGroupInstances)).ToNot(BeNil())
		})

		It("fails when RegisterTargets errors", func() {
			elbClient.mockRegisterTargets.err = fmt.Errorf("failed to register targets")
			registrationProvider.elb = elbClient
			Expect(registrationProvider.Update(lbTargetGroupInstances)).ToNot(BeNil())
		})
	})
})
