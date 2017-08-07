package members

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type localASG struct {
	identityDocument ec2metadata.EC2InstanceIdentityDocument
	instances        []Instance
}

func (a *localASG) GetInstances() []Instance {
	return a.instances
}

func (a *localASG) GetLocalInstance() Instance {
	return Instance{
		InstanceID: a.identityDocument.InstanceID,
		PrivateIP:  a.identityDocument.PrivateIP,
	}
}

// NewAws returns the Members this local instance belongs to.
func NewAws() (Members, error) {
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

	return &localASG{
		identityDocument: identityDoc,
		instances:        instances}, nil
}

func queryInstances(identity ec2metadata.EC2InstanceIdentityDocument, awsASG *autoscaling.AutoScaling, awsEC2 *ec2.EC2) ([]Instance, error) {
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

	var instances []Instance
	for _, reservation := range out.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, Instance{
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
		return "", errors.New("this instance doesn't appear to be part of an autoscaling group")
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
