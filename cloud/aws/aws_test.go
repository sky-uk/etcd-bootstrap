package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sky-uk/etcd-bootstrap/cloud"
	"github.com/sky-uk/etcd-bootstrap/mock"
)

// TestAWSProvider to register the test suite
func TestAWSProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AWS Provider")
}

const (
	localPrivateIP       = "127.0.0.1"
	localInstanceID      = "test-local-instance-id"
	autoscalingGroupName = "test-autoscaling-group"
)

var (
	testInstances = []cloud.Instance{
		{
			InstanceID: "test-instance-id-1",
			PrivateIP:  "192.168.0.1",
		},
		{
			InstanceID: "test-instance-id-2",
			PrivateIP:  "192.168.0.2",
		},
		{
			InstanceID: "test-instance-id-3",
			PrivateIP:  "192.168.0.3",
		},
	}
)

var _ = Describe("AWS Provider", func() {
	var identityDoc *ec2metadata.EC2InstanceIdentityDocument

	BeforeEach(func() {
		identityDoc = &ec2metadata.EC2InstanceIdentityDocument{
			PrivateIP:  localPrivateIP,
			InstanceID: localInstanceID,
		}
	})

	Context("interface functions", func() {
		var awsProvider *Members

		BeforeEach(func() {
			awsProvider = &Members{
				identityDocument: identityDoc,
				instances:        testInstances,
			}
		})

		It("runs GetInstances successfully", func() {
			Expect(awsProvider.GetInstances()).To(Equal(testInstances))
		})

		It("run GetLocalInstance successfully", func() {
			Expect(awsProvider.GetLocalInstance()).To(Equal(cloud.Instance{
				InstanceID: localInstanceID,
				PrivateIP:  localPrivateIP,
			}))
		})
	})

	Context("AWS clients", func() {
		var awsASGClient mock.AWSASGClient
		var awsEC2Client mock.AWSEC2Client

		BeforeEach(func() {
			By("Generating instance arrays based on the test data")
			var nonTerminatedStates = []string{"pending", "running", "shutting-down", "stopped", "stopping"}
			var autoscalingInstances []*autoscaling.Instance
			var autoscalingInstanceIDs []string
			var ec2Instances []*ec2.Instance
			for _, testInstance := range testInstances {
				autoscalingInstances = append(autoscalingInstances, &autoscaling.Instance{
					InstanceId: aws.String(testInstance.InstanceID),
				})
				autoscalingInstanceIDs = append(autoscalingInstanceIDs, testInstance.InstanceID)
				ec2Instances = append(ec2Instances, &ec2.Instance{
					InstanceId:       aws.String(testInstance.InstanceID),
					PrivateIpAddress: aws.String(testInstance.PrivateIP),
				})
			}

			By("Creating dummy client responses")
			awsASGClient = mock.AWSASGClient{
				MockDescribeAutoScalingInstances: mock.DescribeAutoScalingInstances{
					ExpectedInput: &autoscaling.DescribeAutoScalingInstancesInput{
						InstanceIds: aws.StringSlice([]string{localInstanceID}),
					},
					DescribeAutoScalingInstancesOutput: &autoscaling.DescribeAutoScalingInstancesOutput{
						AutoScalingInstances: []*autoscaling.InstanceDetails{{
							AutoScalingGroupName: aws.String(autoscalingGroupName),
						}},
					},
				},
				MockDescribeAutoScalingGroups: mock.DescribeAutoScalingGroups{
					ExpectedInput: &autoscaling.DescribeAutoScalingGroupsInput{
						AutoScalingGroupNames: aws.StringSlice([]string{autoscalingGroupName}),
					},
					DescribeAutoScalingGroupsOutput: &autoscaling.DescribeAutoScalingGroupsOutput{
						AutoScalingGroups: []*autoscaling.Group{{
							Instances: autoscalingInstances,
						}},
					},
				},
			}
			awsEC2Client = mock.AWSEC2Client{
				MockDescribeInstances: mock.DescribeInstances{
					ExpectedInput: &ec2.DescribeInstancesInput{
						InstanceIds: aws.StringSlice(autoscalingInstanceIDs),
						Filters: []*ec2.Filter{
							{
								Name:   aws.String("instance-state-name"),
								Values: aws.StringSlice(nonTerminatedStates),
							},
						},
					},
					DescribeInstancesOutput: &ec2.DescribeInstancesOutput{
						Reservations: []*ec2.Reservation{{
							Instances: ec2Instances,
						}},
					},
				},
			}
		})

		It("queryInstances fails when getASGName errors", func() {
			awsASGClient.MockDescribeAutoScalingInstances.Err = fmt.Errorf("failed to describe autoscaling instances")
			_, err := queryInstances(identityDoc, awsASGClient, awsEC2Client)
			Expect(err).ToNot(BeNil())
		})

		It("queryInstances fails when getASGInstanceIDs errors", func() {
			awsASGClient.MockDescribeAutoScalingGroups.Err = fmt.Errorf("failed to describe autoscaling groups")
			_, err := queryInstances(identityDoc, awsASGClient, awsEC2Client)
			Expect(err).ToNot(BeNil())
		})

		It("queryInstances fails when DescribeInstances errors", func() {
			awsEC2Client.MockDescribeInstances.Err = fmt.Errorf("failed to describe instances")
			_, err := queryInstances(identityDoc, awsASGClient, awsEC2Client)
			Expect(err).ToNot(BeNil())
		})

		It("queryInstances returns correct instance array", func() {
			instances, err := queryInstances(identityDoc, awsASGClient, awsEC2Client)
			Expect(err).To(BeNil())
			Expect(instances).To(Equal(testInstances))
		})

		It("getASGName fails when there are more than 1 autoscaling groups returned for an instance", func() {
			awsASGClient.MockDescribeAutoScalingInstances.DescribeAutoScalingInstancesOutput.AutoScalingInstances = []*autoscaling.InstanceDetails{{}, {}}
			_, err := getASGName(localInstanceID, awsASGClient)
			Expect(err).ToNot(BeNil())
		})

		It("getASGName fails when there are 0 autoscaling groups returned for an instance", func() {
			awsASGClient.MockDescribeAutoScalingInstances.DescribeAutoScalingInstancesOutput.AutoScalingInstances = []*autoscaling.InstanceDetails{}
			_, err := getASGName(localInstanceID, awsASGClient)
			Expect(err).ToNot(BeNil())
		})

		It("getASGInstanceIDs fails when there are more than 1 autoscaling groups returned", func() {
			awsASGClient.MockDescribeAutoScalingGroups.DescribeAutoScalingGroupsOutput.AutoScalingGroups = []*autoscaling.Group{{}, {}}
			_, err := getASGInstanceIDs(autoscalingGroupName, awsASGClient)
			Expect(err).ToNot(BeNil())
		})

		It("getASGInstanceIDs fails when there are 0 autoscaling groups returned", func() {
			awsASGClient.MockDescribeAutoScalingGroups.DescribeAutoScalingGroupsOutput.AutoScalingGroups = []*autoscaling.Group{}
			_, err := getASGInstanceIDs(autoscalingGroupName, awsASGClient)
			Expect(err).ToNot(BeNil())
		})
	})
})
