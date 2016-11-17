package asg

import (
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// ASG represents the local auto scaling group.
type ASG interface {
	// GetInstances returns all the non-terminated instances in the local ASG.
	GetInstances() []Instance
	// GetLocalInstance returns the local machine instance.
	GetLocalInstance() Instance
}

// Instance represents an instance inside of the auto scaling group.
type Instance struct {
	InstanceID string
	PrivateIP  string
}

type localASG struct {
	identityDocument ec2metadata.EC2InstanceIdentityDocument
	asg              *autoscaling.AutoScaling
	ec2              *ec2.EC2
	cachedInstances  []Instance
}

// New returns an ASG representing the ASG the local instance belongs to.
func New() ASG {
	session, err := session.NewSession()
	if err != nil {
		log.Fatal("Unable to create AWS session: ", err)
	}

	identityDoc := getIdentityDocument(session)
	config := &aws.Config{Region: aws.String(identityDoc.Region)}
	asg := autoscaling.New(session, config)
	ec2 := ec2.New(session, config)
	return &localASG{
		identityDocument: identityDoc,
		asg:              asg,
		ec2:              ec2}
}

func getIdentityDocument(session *session.Session) ec2metadata.EC2InstanceIdentityDocument {
	meta := ec2metadata.New(session)
	identityDoc, err := meta.GetInstanceIdentityDocument()
	if err != nil {
		log.Fatal("Unable to retrieve local instance identity document from metadata: ", err)
	}
	return identityDoc
}

func (a *localASG) GetInstances() []Instance {
	if a.cachedInstances != nil {
		return a.cachedInstances
	}

	instanceID := a.GetLocalInstance().InstanceID
	asgName := a.getASGName(instanceID)
	instanceIDs := a.getASGInstanceIDs(asgName)

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

	out, err := a.ec2.DescribeInstances(req)
	if err != nil {
		log.Fatal("Unable to describe instances: ", err)
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

	a.cachedInstances = instances
	return instances
}

func (a *localASG) GetLocalInstance() Instance {
	return Instance{
		InstanceID: a.identityDocument.InstanceID,
		PrivateIP:  a.identityDocument.PrivateIP,
	}
}

func (a *localASG) getASGName(instanceID string) string {
	req := &autoscaling.DescribeAutoScalingInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceID}),
	}
	out, err := a.asg.DescribeAutoScalingInstances(req)
	if err != nil {
		log.Fatal("Unable to describe autoscaling instances: ", err)
	}
	if len(out.AutoScalingInstances) != 1 {
		log.Fatal("This instance doesn't appear to be part of an autoscaling group")
	}
	return *out.AutoScalingInstances[0].AutoScalingGroupName
}

func (a *localASG) getASGInstanceIDs(asgName string) []string {
	req := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: aws.StringSlice([]string{asgName}),
	}
	out, err := a.asg.DescribeAutoScalingGroups(req)

	if err != nil {
		log.Fatalf("Unable to describe auto scaling group %s: %v", asgName, err)
	}
	if len(out.AutoScalingGroups) != 1 {
		log.Fatalf("Expected a single autoscaling group for %s, but found %d", asgName,
			len(out.AutoScalingGroups))
	}

	var instanceIDs []string
	for _, instance := range out.AutoScalingGroups[0].Instances {
		instanceIDs = append(instanceIDs, *instance.InstanceId)
	}
	return instanceIDs
}
