package mock

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/route53"
	. "github.com/onsi/gomega"
	"github.com/sky-uk/etcd-bootstrap/bootstrap/etcd"
	"github.com/sky-uk/etcd-bootstrap/provider"
)

// Mock AWS ASG client
type AWSASGClient struct {
	MockDescribeAutoScalingInstances DescribeAutoScalingInstances
	MockDescribeAutoScalingGroups    DescribeAutoScalingGroups
}

type DescribeAutoScalingInstances struct {
	ExpectedInput                      *autoscaling.DescribeAutoScalingInstancesInput
	DescribeAutoScalingInstancesOutput *autoscaling.DescribeAutoScalingInstancesOutput
	Err                                error
}

type DescribeAutoScalingGroups struct {
	ExpectedInput                   *autoscaling.DescribeAutoScalingGroupsInput
	DescribeAutoScalingGroupsOutput *autoscaling.DescribeAutoScalingGroupsOutput
	Err                             error
}

func (t AWSASGClient) DescribeAutoScalingInstances(a *autoscaling.DescribeAutoScalingInstancesInput) (*autoscaling.DescribeAutoScalingInstancesOutput, error) {
	Expect(a).To(Equal(t.MockDescribeAutoScalingInstances.ExpectedInput))
	return t.MockDescribeAutoScalingInstances.DescribeAutoScalingInstancesOutput, t.MockDescribeAutoScalingInstances.Err
}

func (t AWSASGClient) DescribeAutoScalingGroups(a *autoscaling.DescribeAutoScalingGroupsInput) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	Expect(a).To(Equal(t.MockDescribeAutoScalingGroups.ExpectedInput))
	return t.MockDescribeAutoScalingGroups.DescribeAutoScalingGroupsOutput, t.MockDescribeAutoScalingGroups.Err
}

// Mock AWS EC2 client
type AWSEC2Client struct {
	MockDescribeInstances DescribeInstances
}

type DescribeInstances struct {
	ExpectedInput           *ec2.DescribeInstancesInput
	DescribeInstancesOutput *ec2.DescribeInstancesOutput
	Err                     error
}

func (t AWSEC2Client) DescribeInstances(e *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	Expect(e).To(Equal(t.MockDescribeInstances.ExpectedInput))
	return t.MockDescribeInstances.DescribeInstancesOutput, t.MockDescribeInstances.Err
}

// Mock AWS ELB client
type AWSELBClient struct {
	MockDescribeTargetGroups MockDescribeTargetGroups
	MockRegisterTargets      MockRegisterTargets
}

type MockDescribeTargetGroups struct {
	ExpectedInput              *elbv2.DescribeTargetGroupsInput
	DescribeTargetGroupsOutput *elbv2.DescribeTargetGroupsOutput
	Err                        error
}

func (t AWSELBClient) DescribeTargetGroups(e *elbv2.DescribeTargetGroupsInput) (*elbv2.DescribeTargetGroupsOutput, error) {
	Expect(e).To(Equal(t.MockDescribeTargetGroups.ExpectedInput))
	return t.MockDescribeTargetGroups.DescribeTargetGroupsOutput, t.MockDescribeTargetGroups.Err
}

type MockRegisterTargets struct {
	ExpectedInput         *elbv2.RegisterTargetsInput
	RegisterTargetsOutput *elbv2.RegisterTargetsOutput
	Err                   error
}

func (t AWSELBClient) RegisterTargets(e *elbv2.RegisterTargetsInput) (*elbv2.RegisterTargetsOutput, error) {
	Expect(e).To(Equal(t.MockRegisterTargets.ExpectedInput))
	return t.MockRegisterTargets.RegisterTargetsOutput, t.MockRegisterTargets.Err
}

// Mock AWS Route53 client
type AWSR53Client struct {
	MockGetHostedZone            MockGetHostedZone
	MockChangeResourceRecordSets MockChangeResourceRecordSets
}

type MockGetHostedZone struct {
	ExpectedInput       *route53.GetHostedZoneInput
	GetHostedZoneOutput *route53.GetHostedZoneOutput
	Err                 error
}

func (t AWSR53Client) GetHostedZone(r *route53.GetHostedZoneInput) (*route53.GetHostedZoneOutput, error) {
	Expect(r).To(Equal(t.MockGetHostedZone.ExpectedInput))
	return t.MockGetHostedZone.GetHostedZoneOutput, t.MockGetHostedZone.Err
}

type MockChangeResourceRecordSets struct {
	ExpectedInput                  *route53.ChangeResourceRecordSetsInput
	ChangeResourceRecordSetsOutput *route53.ChangeResourceRecordSetsOutput
	Err                            error
}

func (t AWSR53Client) ChangeResourceRecordSets(r *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error) {
	Expect(r).To(Equal(t.MockChangeResourceRecordSets.ExpectedInput))
	return t.MockChangeResourceRecordSets.ChangeResourceRecordSetsOutput, t.MockChangeResourceRecordSets.Err
}

// Mock etcd cluster client
type EtcdCluster struct {
	MockMembers      MockMembers
	MockRemoveMember MockRemoveMember
	MockAddMember    MockAddMember
}

type MockMembers struct {
	MembersOutput []etcd.Member
	Err           error
}

func (t EtcdCluster) Members() ([]etcd.Member, error) {
	return t.MockMembers.MembersOutput, t.MockMembers.Err
}

type MockRemoveMember struct {
	ExpectedInput string
	Err           error
}

func (t EtcdCluster) RemoveMember(peerURL string) error {
	Expect(peerURL).To(Equal(t.MockRemoveMember.ExpectedInput))
	return t.MockRemoveMember.Err
}

type MockAddMember struct {
	ExpectedInput string
	Err           error
}

func (t EtcdCluster) AddMember(peerURL string) error {
	Expect(peerURL).To(Equal(t.MockAddMember.ExpectedInput))
	return t.MockAddMember.Err
}

// Mock cloud provider
type CloudProvider struct {
	MockGetInstances     MockGetInstances
	MockGetLocalInstance MockGetLocalInstance
}

type MockGetInstances struct {
	GetInstancesOutput []provider.Instance
}

func (t CloudProvider) GetInstances() []provider.Instance {
	return t.MockGetInstances.GetInstancesOutput
}

type MockGetLocalInstance struct {
	GetLocalInstance provider.Instance
}

func (t CloudProvider) GetLocalInstance() provider.Instance {
	return t.MockGetLocalInstance.GetLocalInstance
}
