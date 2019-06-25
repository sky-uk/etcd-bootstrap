package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

// LBTargetGroup abstracts interactions with AWS elbv2.
type LBTargetGroup interface {
	UpdateTargetGroup(targetGroupName string, instances []string) error
}

type lbTargetGroup struct {
	elb *elbv2.ELBV2
}

// UpdateTargetGroup will update the aws lb target group with the discovered etcd instances
func (l *lbTargetGroup) UpdateTargetGroup(targetGroupName string, targetIDs []string) error {
	targetGroups, err := l.elb.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{
		Names: []*string{
			aws.String(targetGroupName),
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
	for _, targetID := range targetIDs {
		targets = append(targets, &elbv2.TargetDescription{
			Id: aws.String(targetID),
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

func newLBTargetGroup() (LBTargetGroup, error) {
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

	awsELBv2 := elbv2.New(awsSession, config)
	return &lbTargetGroup{elb: awsELBv2}, nil
}
