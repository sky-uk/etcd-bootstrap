package aws

import (
	"fmt"
	"github.com/sky-uk/etcd-bootstrap/provider"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAWSProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AWS Provider")
}

const (
	localPrivateIP       = "127.0.0.1"
	localInstanceID      = "test-local-instance-id"
	autoscalingGroupName = "test-autoscaling-group"
	autoscalingGroupID   = "test-autoscaling-group-id"
)

var (
	testInstances = []provider.Instance{
		{
			InstanceID: "test-instance-id",
			PrivateIP:  "192.168.0.1",
		},
	}
)

func (t testAWSASGClient) DescribeAutoScalingInstances(a *autoscaling.DescribeAutoScalingInstancesInput) (*autoscaling.DescribeAutoScalingInstancesOutput, error) {
	Expect(a).To(Equal(t.mockDescribeAutoScalingInstances.expectedInput))
	return t.mockDescribeAutoScalingInstances.describeAutoScalingInstancesOutput, t.mockDescribeAutoScalingInstances.err
}

func (t testAWSASGClient) DescribeAutoScalingGroups(a *autoscaling.DescribeAutoScalingGroupsInput) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	Expect(a).To(Equal(t.mockDescribeAutoScalingGroups.expectedInput))
	return t.mockDescribeAutoScalingGroups.describeAutoScalingGroupsOutput, t.mockDescribeAutoScalingGroups.err
}

func (t testAWSEC2Client) DescribeInstances(e *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	Expect(e).To(Equal(t.mockDescribeInstances.expectedInput))
	return t.mockDescribeInstances.describeInstancesOutput, t.mockDescribeInstances.err
}

type testAWSASGClient struct {
	mockDescribeAutoScalingInstances mockDescribeAutoScalingInstances
	mockDescribeAutoScalingGroups    mockDescribeAutoScalingGroups
}

type mockDescribeAutoScalingInstances struct {
	expectedInput                      *autoscaling.DescribeAutoScalingInstancesInput
	describeAutoScalingInstancesOutput *autoscaling.DescribeAutoScalingInstancesOutput
	err                                error
}

type mockDescribeAutoScalingGroups struct {
	expectedInput                   *autoscaling.DescribeAutoScalingGroupsInput
	describeAutoScalingGroupsOutput *autoscaling.DescribeAutoScalingGroupsOutput
	err                             error
}

type testAWSEC2Client struct {
	mockDescribeInstances mockDescribeInstances
}

type mockDescribeInstances struct {
	expectedInput           *ec2.DescribeInstancesInput
	describeInstancesOutput *ec2.DescribeInstancesOutput
	err                     error
}

var _ = Describe("AWS Provider", func() {
	var identityDoc ec2metadata.EC2InstanceIdentityDocument

	BeforeEach(func() {
		identityDoc = ec2metadata.EC2InstanceIdentityDocument{
			PrivateIP:  localPrivateIP,
			InstanceID: localInstanceID,
		}
	})

	Context("interface functions", func() {
		var awsProvider provider.Provider

		BeforeEach(func() {
			awsProvider = &awsMembers{
				identityDocument: identityDoc,
				instances:        testInstances,
			}
		})

		It("runs GetInstances successfully", func() {
			Expect(awsProvider.GetInstances()).To(Equal(testInstances))
		})

		It("run GetLocalInstance successfully", func() {
			Expect(awsProvider.GetLocalInstance()).To(Equal(provider.Instance{
				InstanceID: localInstanceID,
				PrivateIP:  localPrivateIP,
			}))
		})
	})

	Context("AWS clients", func() {
		var awsASGClient testAWSASGClient
		var awsEC2Client testAWSEC2Client

		BeforeEach(func() {
			// generate instance arrays based on the test data
			var nonTerminatedStates = []string{"pending", "running", "shutting-down", "stopped", "stopping"}
			var autoscalingInstances []*autoscaling.Instance
			var autoscalingInstanceIDs []string
			var ec2Instances []*ec2.Instance
			for _, testInstance := range testInstances {
				autoscalingInstances = append(autoscalingInstances, &autoscaling.Instance{
					InstanceId:       aws.String(testInstance.InstanceID),
				})
				autoscalingInstanceIDs = append(autoscalingInstanceIDs, testInstance.InstanceID)
				ec2Instances = append(ec2Instances, &ec2.Instance{
					InstanceId:       aws.String(testInstance.InstanceID),
					PrivateIpAddress: aws.String(testInstance.PrivateIP),
				})
			}
			// create dummy passing client responses
			awsASGClient = testAWSASGClient{
				mockDescribeAutoScalingInstances: mockDescribeAutoScalingInstances{
					expectedInput: &autoscaling.DescribeAutoScalingInstancesInput{
						InstanceIds: aws.StringSlice([]string{localInstanceID}),
					},
					describeAutoScalingInstancesOutput: &autoscaling.DescribeAutoScalingInstancesOutput{
						AutoScalingInstances: []*autoscaling.InstanceDetails{{
							AutoScalingGroupName: aws.String(autoscalingGroupName),
						}},
					},
				},
				mockDescribeAutoScalingGroups: mockDescribeAutoScalingGroups{
					expectedInput: &autoscaling.DescribeAutoScalingGroupsInput{
						AutoScalingGroupNames: aws.StringSlice([]string{autoscalingGroupName}),
					},
					describeAutoScalingGroupsOutput: &autoscaling.DescribeAutoScalingGroupsOutput{
						AutoScalingGroups: []*autoscaling.Group{{
							Instances: autoscalingInstances,
						}},
					},
				},
			}
			awsEC2Client = testAWSEC2Client{
				mockDescribeInstances: mockDescribeInstances{
					expectedInput: &ec2.DescribeInstancesInput{
						InstanceIds: aws.StringSlice(autoscalingInstanceIDs),
						Filters: []*ec2.Filter{
							{
								Name:   aws.String("instance-state-name"),
								Values: aws.StringSlice(nonTerminatedStates),
							},
						},
					},
					describeInstancesOutput: &ec2.DescribeInstancesOutput{
						Reservations: []*ec2.Reservation{{
							Instances: ec2Instances,
						}},
					},
				},
			}
		})

		It("queryInstances fails when getASGName errors", func() {
			awsASGClient.mockDescribeAutoScalingInstances.err = fmt.Errorf("failed to describe autoscaling instances")
			_, err := queryInstances(identityDoc, awsASGClient, awsEC2Client)
			Expect(err).ToNot(BeNil())
		})

		It("queryInstances fails when getASGInstanceIDs errors", func() {
			awsASGClient.mockDescribeAutoScalingGroups.err = fmt.Errorf("failed to describe autoscaling groups")
			_, err := queryInstances(identityDoc, awsASGClient, awsEC2Client)
			Expect(err).ToNot(BeNil())
		})

		It("queryInstances fails when DescribeInstances errors", func() {
			awsEC2Client.mockDescribeInstances.err = fmt.Errorf("failed to describe instances")
			_, err := queryInstances(identityDoc, awsASGClient, awsEC2Client)
			Expect(err).ToNot(BeNil())
		})

		It("queryInstances returns correct instance array", func() {
			instances, err := queryInstances(identityDoc, awsASGClient, awsEC2Client)
			Expect(err).To(BeNil())
			Expect(instances).To(Equal(testInstances))
		})

		It("getASGName fails when there are more than 1 autoscaling groups returned for an instance", func() {
			awsASGClient.mockDescribeAutoScalingInstances.describeAutoScalingInstancesOutput.AutoScalingInstances = []*autoscaling.InstanceDetails{{}, {}}
			_, err := getASGName(localInstanceID, awsASGClient)
			Expect(err).ToNot(BeNil())
		})

		It("getASGName fails when there are 0 autoscaling groups returned for an instance", func() {
			awsASGClient.mockDescribeAutoScalingInstances.describeAutoScalingInstancesOutput.AutoScalingInstances = []*autoscaling.InstanceDetails{}
			_, err := getASGName(localInstanceID, awsASGClient)
			Expect(err).ToNot(BeNil())
		})

		It("getASGInstanceIDs fails when there are more than 1 autoscaling groups returned", func() {
			awsASGClient.mockDescribeAutoScalingGroups.describeAutoScalingGroupsOutput.AutoScalingGroups = []*autoscaling.Group{{}, {}}
			_, err := getASGInstanceIDs(autoscalingGroupName, awsASGClient)
			Expect(err).ToNot(BeNil())
		})

		It("getASGInstanceIDs fails when there are 0 autoscaling groups returned", func() {
			awsASGClient.mockDescribeAutoScalingGroups.describeAutoScalingGroupsOutput.AutoScalingGroups = []*autoscaling.Group{}
			_, err := getASGInstanceIDs(autoscalingGroupName, awsASGClient)
			Expect(err).ToNot(BeNil())
		})
	})
})
