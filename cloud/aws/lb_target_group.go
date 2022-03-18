package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/sky-uk/etcd-bootstrap/cloud"
)

// LBTargetGroupRegistrationProviderConfig contains configuration when creating a default LBTargetGroupRegistrationProvider
type LBTargetGroupRegistrationProviderConfig struct {
	TargetGroupName string
}

// elb interface to abstract away from AWS commands
type elb interface {
	// DescribeTargetGroups returns information about an aws elb target group
	DescribeTargetGroups(e *elbv2.DescribeTargetGroupsInput) (*elbv2.DescribeTargetGroupsOutput, error)
	// RegisterTargets registers instance or ip targets with an aws elb target group
	RegisterTargets(e *elbv2.RegisterTargetsInput) (*elbv2.RegisterTargetsOutput, error)
	// DescribeTargetHealth returns information about the health of an aws elb target
	DescribeTargetHealth(e *elbv2.DescribeTargetHealthInput) (*elbv2.DescribeTargetHealthOutput, error)
	// DeregisterTargets deregisters instance or ip targets from an aws elb target group
	DeregisterTargets(e *elbv2.DeregisterTargetsInput) (*elbv2.DeregisterTargetsOutput, error)
}

// LBTargetGroupRegistrationProvider contains an aws elb client and a target group name used for registering etcd
// cluster information with an aws elb target group
type LBTargetGroupRegistrationProvider struct {
	targetGroupName string
	elb             elb
}

// NewLBTargetGroupRegistrationProvider returns a default LBTargetGroupRegistrationProvider and initiates a new aws elb
// client
func NewLBTargetGroupRegistrationProvider(c *LBTargetGroupRegistrationProviderConfig) (*LBTargetGroupRegistrationProvider, error) {
	awsSession, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create new AWS session: %v", err)
	}

	meta := ec2metadata.New(awsSession)
	identityDoc, err := meta.GetInstanceIdentityDocument()
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS local instance data: %v", err)
	}
	config := &aws.Config{Region: aws.String(identityDoc.Region)}
	elbClient := elbv2.New(awsSession, config)

	return &LBTargetGroupRegistrationProvider{
		targetGroupName: c.TargetGroupName,
		elb:             elbClient,
	}, nil
}

// Update will update the aws lb target group with the discovered etcd instances
func (l LBTargetGroupRegistrationProvider) Update(instances []cloud.Instance) error {
	targetGroups, err := l.elb.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{
		Names: []*string{
			aws.String(l.targetGroupName),
		},
	})
	if err != nil {
		return fmt.Errorf("unable to describe loadbalancer target groups: %v", err)
	}

	targetGroupARN, err := getTargetGroupARN(targetGroups)
	if err != nil {
		return fmt.Errorf("target group validation failed: %v", err)
	}

	var targets []*elbv2.TargetDescription
	for _, instance := range instances {
		targets = append(targets, &elbv2.TargetDescription{
			Id: aws.String(instance.Endpoint),
		})
	}

	registerEtcdInstances := &elbv2.RegisterTargetsInput{
		TargetGroupArn: targetGroupARN,
		Targets:        targets,
	}

	if _, err := l.elb.RegisterTargets(registerEtcdInstances); err != nil {
		return fmt.Errorf("unable to register etcd instances with loadbalancer target group: %v", err)
	}

	targetsToRemove := []*elbv2.TargetDescription{}

	existingTargets, err := l.getExistingLBTargets(targetGroupARN)
	if err != nil {
		return err
	}

	for _, target := range existingTargets {
		if !contains(instances, target) {
			targetsToRemove = append(targetsToRemove, target)
		}
	}

	if len(targetsToRemove) > 0 {
		deregisterEtcdInstances := &elbv2.DeregisterTargetsInput{
			TargetGroupArn: targetGroupARN,
			Targets:        targetsToRemove,
		}

		_, err = l.elb.DeregisterTargets(deregisterEtcdInstances)
		if err != nil {
			return fmt.Errorf("unable to deregister etcd instances from loadbalancer target group: %v", err)
		}
	}

	return nil
}

func (l LBTargetGroupRegistrationProvider) getExistingLBTargets(targetGroupARN *string) ([]*elbv2.TargetDescription, error) {
	existingTargets, err := l.elb.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
		TargetGroupArn: targetGroupARN,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to describe loadbalancer target health: %v", err)
	}

	targetDescriptions := []*elbv2.TargetDescription{}
	for _, target := range existingTargets.TargetHealthDescriptions {
		targetDescriptions = append(targetDescriptions, target.Target)
	}

	return targetDescriptions, nil
}

func getTargetGroupARN(targetGroups *elbv2.DescribeTargetGroupsOutput) (*string, error) {
	var targetGroupARN *string

	if totalTargetGroups := len(targetGroups.TargetGroups); totalTargetGroups != 1 {
		return targetGroupARN, fmt.Errorf("unexpected number of target groups found: expected: 1, received: %v", totalTargetGroups)
	}

	return targetGroups.TargetGroups[0].TargetGroupArn, nil
}

func contains(instances []cloud.Instance, targetDescription *elbv2.TargetDescription) bool {
	for _, i := range instances {
		if i.Endpoint == *targetDescription.Id {
			return true
		}
	}
	return false
}
