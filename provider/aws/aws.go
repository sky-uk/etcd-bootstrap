package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/sky-uk/etcd-bootstrap/provider"
)

// awsASG interface to abstract away from AWS commands
type awsASG interface {
	DescribeAutoScalingInstances(a *autoscaling.DescribeAutoScalingInstancesInput) (*autoscaling.DescribeAutoScalingInstancesOutput, error)
	DescribeAutoScalingGroups(a *autoscaling.DescribeAutoScalingGroupsInput) (*autoscaling.DescribeAutoScalingGroupsOutput, error)
}

// awsEC2 interface to abstract away from AWS commands
type awsEC2 interface {
	DescribeInstances(e *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error)
}

type awsMembers struct {
	identityDocument ec2metadata.EC2InstanceIdentityDocument
	instances        []provider.Instance
}

// GetInstances will return the aws etcd instances
func (a *awsMembers) GetInstances() []provider.Instance {
	return a.instances
}

// GetLocalInstance will get the aws instance etcd bootstrap is running on
func (a *awsMembers) GetLocalInstance() provider.Instance {
	return provider.Instance{
		InstanceID: a.identityDocument.InstanceID,
		PrivateIP:  a.identityDocument.PrivateIP,
	}
}

// NewAWS returns the Members this local instance belongs to.
func NewAWS() (provider.Provider, error) {
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
	awsASGClient := autoscaling.New(awsSession, config)
	awsEC2Client := ec2.New(awsSession, config)

	instances, err := queryInstances(identityDoc, awsASGClient, awsEC2Client)
	if err != nil {
		return nil, err
	}

	return &awsMembers{
		identityDocument: identityDoc,
		instances:        instances,
	}, nil
}

func queryInstances(identity ec2metadata.EC2InstanceIdentityDocument, awsASGClient awsASG, awsEC2Client awsEC2) ([]provider.Instance, error) {
	instanceID := identity.InstanceID
	asgName, err := getASGName(instanceID, awsASGClient)
	if err != nil {
		return nil, err
	}

	instanceIDs, err := getASGInstanceIDs(asgName, awsASGClient)
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

	out, err := awsEC2Client.DescribeInstances(req)
	if err != nil {
		return nil, err
	}

	var instances []provider.Instance
	for _, reservation := range out.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, provider.Instance{
				InstanceID: *instance.InstanceId,
				PrivateIP:  *instance.PrivateIpAddress,
			})
		}
	}

	return instances, nil
}

func getASGName(instanceID string, a awsASG) (string, error) {
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

func getASGInstanceIDs(asgName string, awsASG awsASG) ([]string, error) {
	req := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: aws.StringSlice([]string{asgName}),
	}
	out, err := awsASG.DescribeAutoScalingGroups(req)

	if err != nil {
		return nil, err
	}
	if len(out.AutoScalingGroups) != 1 {
		return nil, fmt.Errorf("expected a single autoscaling group for %s, but found %d", asgName,
			len(out.AutoScalingGroups))
	}

	var instanceIDs []string
	for _, instance := range out.AutoScalingGroups[0].Instances {
		instanceIDs = append(instanceIDs, *instance.InstanceId)
	}
	return instanceIDs, nil
}
