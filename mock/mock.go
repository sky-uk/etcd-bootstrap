package mock

import (
	"errors"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/route53"

	"github.com/onsi/gomega"
)

// AWSASGClient for mocking calls to the aws autoscaling client
type AWSASGClient struct {
	MockDescribeAutoScalingInstances DescribeAutoScalingInstances
	MockDescribeAutoScalingGroups    DescribeAutoScalingGroups
}

// DescribeAutoScalingInstances sets the expected input and output for DescribeAutoScalingInstances() on AWSASGClient
type DescribeAutoScalingInstances struct {
	ExpectedInput                      *autoscaling.DescribeAutoScalingInstancesInput
	DescribeAutoScalingInstancesOutput *autoscaling.DescribeAutoScalingInstancesOutput
	Err                                error
}

// DescribeAutoScalingGroups sets the expected input and output for DescribeAutoScalingGroups() on AWSASGClient
type DescribeAutoScalingGroups struct {
	ExpectedInput                   *autoscaling.DescribeAutoScalingGroupsInput
	DescribeAutoScalingGroupsOutput *autoscaling.DescribeAutoScalingGroupsOutput
	Err                             error
}

// DescribeAutoScalingInstances mocks the aws autoscaling group client
func (t AWSASGClient) DescribeAutoScalingInstances(a *autoscaling.DescribeAutoScalingInstancesInput) (*autoscaling.DescribeAutoScalingInstancesOutput, error) {
	gomega.Expect(a).To(gomega.Equal(t.MockDescribeAutoScalingInstances.ExpectedInput))
	return t.MockDescribeAutoScalingInstances.DescribeAutoScalingInstancesOutput, t.MockDescribeAutoScalingInstances.Err
}

// DescribeAutoScalingGroups mocks the aws autoscaling group client
func (t AWSASGClient) DescribeAutoScalingGroups(a *autoscaling.DescribeAutoScalingGroupsInput) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	gomega.Expect(a).To(gomega.Equal(t.MockDescribeAutoScalingGroups.ExpectedInput))
	return t.MockDescribeAutoScalingGroups.DescribeAutoScalingGroupsOutput, t.MockDescribeAutoScalingGroups.Err
}

// AWSEC2Client for mocking calls to the aws ec2 client
type AWSEC2Client struct {
	MockDescribeInstances DescribeInstances
}

// DescribeInstances sets the expected input and output for DescribeInstances() on AWSEC2Client
type DescribeInstances struct {
	ExpectedInput           *ec2.DescribeInstancesInput
	DescribeInstancesOutput *ec2.DescribeInstancesOutput
	Err                     error
}

// DescribeInstances mocks the aws ec2 client
func (t AWSEC2Client) DescribeInstances(e *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	gomega.Expect(e).To(gomega.Equal(t.MockDescribeInstances.ExpectedInput))
	return t.MockDescribeInstances.DescribeInstancesOutput, t.MockDescribeInstances.Err
}

// AWSELBClient for mocking calls to the aws elb client
type AWSELBClient struct {
	MockDescribeTargetGroups DescribeTargetGroups
	MockRegisterTargets      RegisterTargets
	MockDescribeTargetHealth DescribeTargetHealth
	MockDeregisterTargets    DeregisterTargets
}

// DescribeTargetGroups sets the expected input and output for DescribeTargetGroups() on AWSELBClient
type DescribeTargetGroups struct {
	ExpectedInput              *elbv2.DescribeTargetGroupsInput
	DescribeTargetGroupsOutput *elbv2.DescribeTargetGroupsOutput
	Err                        error
}

// DescribeTargetGroups mocks the aws elb client
func (t AWSELBClient) DescribeTargetGroups(e *elbv2.DescribeTargetGroupsInput) (*elbv2.DescribeTargetGroupsOutput, error) {
	gomega.Expect(e).To(gomega.Equal(t.MockDescribeTargetGroups.ExpectedInput))
	return t.MockDescribeTargetGroups.DescribeTargetGroupsOutput, t.MockDescribeTargetGroups.Err
}

// RegisterTargets sets the expected input and output for RegisterTargets() on AWSELBClient
type RegisterTargets struct {
	ExpectedInput         *elbv2.RegisterTargetsInput
	RegisterTargetsOutput *elbv2.RegisterTargetsOutput
	Err                   error
}

// RegisterTargets mocks the aws elb client
func (t AWSELBClient) RegisterTargets(e *elbv2.RegisterTargetsInput) (*elbv2.RegisterTargetsOutput, error) {
	gomega.Expect(e).To(gomega.Equal(t.MockRegisterTargets.ExpectedInput))
	return t.MockRegisterTargets.RegisterTargetsOutput, t.MockRegisterTargets.Err
}

// DescribeTargetHealth sets the expected input and output for DescribeTargetHealth() on AWSELBClient
type DescribeTargetHealth struct {
	ExpectedInput              *elbv2.DescribeTargetHealthInput
	DescribeTargetHealthOutput *elbv2.DescribeTargetHealthOutput
	Err                        error
}

// DescribeTargetHealth mocks the aws elb client
func (t AWSELBClient) DescribeTargetHealth(e *elbv2.DescribeTargetHealthInput) (*elbv2.DescribeTargetHealthOutput, error) {
	gomega.Expect(e).To(gomega.Equal(t.MockDescribeTargetHealth.ExpectedInput))
	return t.MockDescribeTargetHealth.DescribeTargetHealthOutput, t.MockDescribeTargetHealth.Err
}

// DeregisterTargets sets the expected input and output for DeregisterTargets() on AWSELBClient
type DeregisterTargets struct {
	ExpectedInput           *elbv2.DeregisterTargetsInput
	DeregisterTargetsOutput *elbv2.DeregisterTargetsOutput
	Err                     error
}

// DeregisterTargets mocks the aws elb client
func (t AWSELBClient) DeregisterTargets(e *elbv2.DeregisterTargetsInput) (*elbv2.DeregisterTargetsOutput, error) {
	if len(e.Targets) == 0 {
		return nil, errors.New("ValidationError: Targets must be specified")
	}
	gomega.Expect(e).To(gomega.Equal(t.MockDeregisterTargets.ExpectedInput))
	return t.MockDeregisterTargets.DeregisterTargetsOutput, t.MockDeregisterTargets.Err
}

// AWSR53Client for mocking calls to the aws route53 client
type AWSR53Client struct {
	MockGetHostedZone            GetHostedZone
	MockChangeResourceRecordSets ChangeResourceRecordSets
}

// GetHostedZone sets the expected input and output for GetHostedZone() on AWSR53Client
type GetHostedZone struct {
	ExpectedInput       *route53.GetHostedZoneInput
	GetHostedZoneOutput *route53.GetHostedZoneOutput
	Err                 error
}

// GetHostedZone mocks the aws route53 client
func (t AWSR53Client) GetHostedZone(r *route53.GetHostedZoneInput) (*route53.GetHostedZoneOutput, error) {
	gomega.Expect(r).To(gomega.Equal(t.MockGetHostedZone.ExpectedInput))
	return t.MockGetHostedZone.GetHostedZoneOutput, t.MockGetHostedZone.Err
}

// ChangeResourceRecordSets sets the expected input and output for ChangeResourceRecordSets() on AWSR53Client
type ChangeResourceRecordSets struct {
	ExpectedInput                  *route53.ChangeResourceRecordSetsInput
	ChangeResourceRecordSetsOutput *route53.ChangeResourceRecordSetsOutput
	Err                            error
}

// ChangeResourceRecordSets mocks the aws route53 client
func (t AWSR53Client) ChangeResourceRecordSets(r *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error) {
	gomega.Expect(r).To(gomega.Equal(t.MockChangeResourceRecordSets.ExpectedInput))
	return t.MockChangeResourceRecordSets.ChangeResourceRecordSetsOutput, t.MockChangeResourceRecordSets.Err
}
