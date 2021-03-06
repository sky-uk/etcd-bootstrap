package aws

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/sky-uk/etcd-bootstrap/cloud"
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

// AWS returns the instances in the local auto scaling group.
type AWS struct {
	awsSession       *session.Session
	identityDocument *ec2metadata.EC2InstanceIdentityDocument
	instances        []cloud.Instance
}

// GetInstances will return the aws etcd instances
func (m *AWS) GetInstances() ([]cloud.Instance, error) {
	if m.instances == nil {
		identityDoc, err := m.getIdentityDoc()
		if err != nil {
			return nil, fmt.Errorf("unable to get local instance information: %w", err)
		}
		config := &aws.Config{Region: aws.String(identityDoc.Region)}
		awsASGClient := autoscaling.New(m.awsSession, config)
		awsEC2Client := ec2.New(m.awsSession, config)
		instances, err := queryInstances(identityDoc, awsASGClient, awsEC2Client)
		if err != nil {
			return nil, fmt.Errorf("unable to query ASG: %w", err)
		}
		m.instances = instances
	}

	return m.instances, nil
}

// GetLocalInstance will get the aws instance etcd bootstrap is running on
func (m *AWS) GetLocalInstance() (cloud.Instance, error) {
	identityDoc, err := m.getIdentityDoc()
	if err != nil {
		return cloud.Instance{}, err
	}
	return cloud.Instance{
		Name:     identityDoc.InstanceID,
		Endpoint: identityDoc.PrivateIP,
	}, nil
}

// GetLocalIP returns the local instance's PrivateIP.
func (m *AWS) GetLocalIP() (string, error) {
	localInstance, err := m.GetLocalInstance()
	if err != nil {
		return "", err
	}
	return localInstance.Endpoint, nil
}

func (m *AWS) getIdentityDoc() (*ec2metadata.EC2InstanceIdentityDocument, error) {
	if m.identityDocument == nil {
		meta := ec2metadata.New(m.awsSession)
		identityDoc, err := meta.GetInstanceIdentityDocument()
		if err != nil {
			return nil, fmt.Errorf("failed to get AWS local instance data: %v", err)
		}
		m.identityDocument = &identityDoc
	}
	return m.identityDocument, nil
}

// NewAWS returns the Members this local instance belongs to.
func NewAWS() (*AWS, error) {
	awsSession, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create new AWS session: %v", err)
	}
	return &AWS{
		awsSession: awsSession,
	}, nil
}

func queryInstances(identity *ec2metadata.EC2InstanceIdentityDocument, awsASGClient awsASG, awsEC2Client awsEC2) ([]cloud.Instance, error) {
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

	var instances []cloud.Instance
	for _, reservation := range out.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, cloud.Instance{
				Name:     *instance.InstanceId,
				Endpoint: *instance.PrivateIpAddress,
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
		return "", fmt.Errorf("failed to describe AWS ASG instances: %v", err)
	}
	if len(out.AutoScalingInstances) != 1 {
		return "", errors.New("this instance doesn't appear to be part of an autoscaling group")
	}
	return *out.AutoScalingInstances[0].AutoScalingGroupName, nil
}

func getASGInstanceIDs(asgName string, awsASG awsASG) ([]string, error) {
	req := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: aws.StringSlice([]string{asgName}),
	}
	out, err := awsASG.DescribeAutoScalingGroups(req)

	if err != nil {
		return nil, fmt.Errorf("failed to describe AWS ASG groups: %v", err)
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
