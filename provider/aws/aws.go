package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/sky-uk/etcd-bootstrap/lib/cloud"
)

// Config is the configuration required to talk to the AWS API.
type Config struct {
	R53ZoneID         string
	LBTargetGroupName string
}

type awsMembers struct {
	identityDocument  ec2metadata.EC2InstanceIdentityDocument
	instances         []cloud.Instance
	r53               R53
	r53ZoneID         string
	lbTargetGroup     LBTargetGroup
	lbTargetGroupName string
}

// GetInstances will return the aws etcd instances
func (a *awsMembers) GetInstances() []cloud.Instance {
	return a.instances
}

// GetLocalInstance will get the aws instance etcd bootstrap is running on
func (a *awsMembers) GetLocalInstance() cloud.Instance {
	return cloud.Instance{
		InstanceID: a.identityDocument.InstanceID,
		PrivateIP:  a.identityDocument.PrivateIP,
	}
}

// UpdateDNS will update the specified route53 zone with the configured domain if the dns registration type is enabled
func (a *awsMembers) UpdateDNS(name string) error {
	var ips []string
	for _, instance := range a.GetInstances() {
		ips = append(ips, instance.PrivateIP)
	}

	if a.r53ZoneID == "" {
		return fmt.Errorf("unable to register DNS, route53 zone id not set")
	}
	return a.r53.UpdateARecords(a.r53ZoneID, name, ips)
}

// UpdateLB will update the specified loadbalancer target group with the aws etcd instances if the lb registration type
// is enabled
func (a *awsMembers) UpdateLB() error {
	var instanceIDs []string
	for _, instance := range a.GetInstances() {
		instanceIDs = append(instanceIDs, instance.InstanceID)
	}
	if a.lbTargetGroupName == "" {
		return fmt.Errorf("unable to register loadbalancer, loadbalancer target group name not set")
	}
	return a.lbTargetGroup.UpdateTargetGroup(a.lbTargetGroupName, instanceIDs)
}

// NewAws returns the Members this local instance belongs to.
func NewAws(cfg *Config) (cloud.Cloud, error) {
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
	awsASG := autoscaling.New(awsSession, config)
	awsEC2 := ec2.New(awsSession, config)

	instances, err := queryInstances(identityDoc, awsASG, awsEC2)
	if err != nil {
		return nil, fmt.Errorf("unable to query instances: %v", err)
	}

	dns, err := newR53()
	lb, err := newLBTargetGroup()

	return &awsMembers{
		identityDocument:  identityDoc,
		instances:         instances,
		r53:               dns,
		r53ZoneID:         cfg.R53ZoneID,
		lbTargetGroup:     lb,
		lbTargetGroupName: cfg.LBTargetGroupName,
	}, nil
}

func queryInstances(identity ec2metadata.EC2InstanceIdentityDocument, awsASG *autoscaling.AutoScaling, awsEC2 *ec2.EC2) ([]cloud.Instance, error) {
	instanceID := identity.InstanceID
	asgName, err := getASGName(instanceID, awsASG)
	if err != nil {
		return nil, err
	}

	instanceIDs, err := getASGInstanceIDs(asgName, awsASG)
	if err != nil {
		return nil, err
	}

	var nonTerminatedStates = []string{"pending", "running", "shutting-down", "stopped", "stopping"}
	req := &ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice(instanceIDs),
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: aws.StringSlice(nonTerminatedStates),
			},
		},
	}

	out, err := awsEC2.DescribeInstances(req)
	if err != nil {
		return nil, err
	}

	var instances []cloud.Instance
	for _, reservation := range out.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, cloud.Instance{
				InstanceID: *instance.InstanceId,
				PrivateIP:  *instance.PrivateIpAddress,
			})
		}
	}

	return instances, nil
}

func getASGName(instanceID string, a *autoscaling.AutoScaling) (string, error) {
	req := &autoscaling.DescribeAutoScalingInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceID}),
	}
	out, err := a.DescribeAutoScalingInstances(req)
	if err != nil {
		return "", err
	}
	if len(out.AutoScalingInstances) != 1 {
		return "", fmt.Errorf("this instance doesn't appear to be part of an autoscaling group")
	}
	return *out.AutoScalingInstances[0].AutoScalingGroupName, nil
}

func getASGInstanceIDs(asgName string, awsASG *autoscaling.AutoScaling) ([]string, error) {
	req := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: aws.StringSlice([]string{asgName}),
	}
	out, err := awsASG.DescribeAutoScalingGroups(req)

	if err != nil {
		return nil, err
	}
	if len(out.AutoScalingGroups) != 1 {
		return nil, fmt.Errorf("Expected a single autoscaling group for %s, but found %d", asgName,
			len(out.AutoScalingGroups))
	}

	var instanceIDs []string
	for _, instance := range out.AutoScalingGroups[0].Instances {
		instanceIDs = append(instanceIDs, *instance.InstanceId)
	}
	return instanceIDs, nil
}
