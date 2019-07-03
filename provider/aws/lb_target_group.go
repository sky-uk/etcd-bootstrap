package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/sky-uk/etcd-bootstrap/provider"
)

type LBTargetGroupRegistrationProviderConfig struct {
	TargetGroupName string
}

// elb interface to abstract away from AWS commands
type elb interface {
	DescribeTargetGroups(e *elbv2.DescribeTargetGroupsInput) (*elbv2.DescribeTargetGroupsOutput, error)
	RegisterTargets(e *elbv2.RegisterTargetsInput) (*elbv2.RegisterTargetsOutput, error)
}

type LBTargetGroupRegistrationProvider struct {
	targetGroupName string
	elb             elb
}

func NewLBTargetGroupRegistrationProvider(c *LBTargetGroupRegistrationProviderConfig) (provider.RegistrationProvider, error) {
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
	elbClient := elbv2.New(awsSession, config)

	return LBTargetGroupRegistrationProvider{
		targetGroupName: c.TargetGroupName,
		elb:             elbClient,
	}, nil
}

// Update will update the aws lb target group with the discovered etcd instances
func (l LBTargetGroupRegistrationProvider) Update(instances []provider.Instance) error {
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
			Id: aws.String(instance.InstanceID),
		})
	}

	registerEtcdInstances := &elbv2.RegisterTargetsInput{
		TargetGroupArn: targetGroupARN,
		Targets:        targets,
	}

	if _, err := l.elb.RegisterTargets(registerEtcdInstances); err != nil {
		return fmt.Errorf("unable to register etcd instances with loadbalancer target group: %v", err)
	}

	return nil
}

func getTargetGroupARN(targetGroups *elbv2.DescribeTargetGroupsOutput) (*string, error) {
	var targetGroupARN *string

	if totalTargetGroups := len(targetGroups.TargetGroups); totalTargetGroups != 1 {
		return targetGroupARN, fmt.Errorf("unexpected number of target groups found (%v)", totalTargetGroups)
	}

	return targetGroups.TargetGroups[0].TargetGroupArn, nil
}
