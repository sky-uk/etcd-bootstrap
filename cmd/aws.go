package cmd

import (
	"fmt"

	"github.com/sky-uk/etcd-bootstrap/bootstrap"
	"github.com/sky-uk/etcd-bootstrap/cloud"

	log "github.com/sirupsen/logrus"
	aws_cloud "github.com/sky-uk/etcd-bootstrap/cloud/aws"
	"github.com/sky-uk/etcd-bootstrap/cloud/noop"
	"github.com/sky-uk/etcd-bootstrap/cloud/srv"
	"github.com/spf13/cobra"
)

// awsCmd represents the generate config command for AWS etcd clusters
var awsCmd = &cobra.Command{
	Use:   "aws",
	Short: "Generates config for an AWS etcd cluster",
	Run:   aws,
}

var (
	awsRegistrationProvider string
	route53ZoneID           string
	dnsHostname             string
	lbTargetGroupName       string
	instanceLookupMethod    string
	srvDomainName           string
	srvService              string
)

func init() {
	RootCmd.AddCommand(awsCmd)

	awsCmd.Flags().StringVarP(&awsRegistrationProvider, "registration-provider", "r", "noop", fmt.Sprintf(
		"automatic registration provider to use, options are: noop, lb, route53"))
	awsCmd.Flags().StringVar(&route53ZoneID, "r53-zone-id", "",
		"zone id for automatic registration for registration-provider=route53")
	awsCmd.Flags().StringVar(&dnsHostname, "dns-hostname", "",
		"hostname to set to the etcd cluster when registration-provider=route53")
	awsCmd.Flags().StringVar(&lbTargetGroupName, "lb-target-group-name", "",
		"loadbalancer target group name to use when --registration-provider=lb")
	awsCmd.Flags().StringVar(&instanceLookupMethod, "instance-lookup-method", "asg",
		"method for looking up instances in the cluster, options are: asg, srv")
	awsCmd.Flags().StringVar(&srvDomainName, "srv-domain-name", "",
		"domain name to use for instance-lookup-method=srv")
	awsCmd.Flags().StringVar(&srvService, "srv-service", "etcd-bootstrap",
		"service to use for instance-lookup-method=srv")
}

type registrationProvider interface {
	Update([]cloud.Instance) error
}

func aws(cmd *cobra.Command, args []string) {
	registrator := initialiseAWSRegistrationProvider()

	aws, err := aws_cloud.NewAWS()
	if err != nil {
		log.Fatalf("Failed to create AWS provider: %v", err)
	}

	cloudInstances := cloudInstances(aws)
	bootstrapper, err := bootstrap.New(cloudInstances, aws)
	if err != nil {
		log.Fatalf("Failed to create etcd bootstrapper: %v", err)
	}

	if err := bootstrapper.GenerateEtcdFlagsFile(outputFilename); err != nil {
		log.Fatalf("Failed to generate etcd flags file: %v", err)
	}

	instances, err := aws.GetInstances()
	if err != nil {
		log.Fatalf("Failed to retrieve instances: %v", err)
	}
	if err := registrator.Update(instances); err != nil {
		log.Fatalf("Failed to register etcd cluster data with cloud registration provider: %v", err)
	}
}

func cloudInstances(asg *aws_cloud.Members) bootstrap.CloudInstances {
	switch instanceLookupMethod {
	case "asg":
		log.Info("Using ASG for looking up cluster instances")
		return asg
	case "srv":
		log.Info("Using SRV record for looking up cluster instances")
		if srvDomainName == "" {
			log.Fatalf("srv-domain-name must be provided")
		}
		// Instead of using ASG, use SRV for looking up the instances.
		cloudInstances := srv.New(&srv.Config{
			DomainName: srvDomainName,
			Service:    srvService,
		})
		return cloudInstances
	default:
		log.Fatalf("Unsupported cluster lookup method %q", instanceLookupMethod)
		return nil
	}
}

func initialiseAWSRegistrationProvider() registrationProvider {
	switch awsRegistrationProvider {
	case "noop":
		log.Info("Using noop cloud registration provider")
		return noop.RegistrationProvider{}
	case "route53":
		checkRequiredFlag(route53ZoneID, "--r53-zone-id")
		checkRequiredFlag(dnsHostname, "--dns-hostname")

		registrator, err := aws_cloud.NewRoute53RegistrationProvider(&aws_cloud.Route53RegistrationProviderConfig{
			ZoneID:   route53ZoneID,
			Hostname: dnsHostname,
		})
		if err != nil {
			log.Fatalf("Failed to create route53 registration client: %v", err)
		}
		log.Info("Using route53 cloud registration provider")
		return registrator
	case "lb":
		checkRequiredFlag(lbTargetGroupName, "--lb-target-group-name")

		registrator, err := aws_cloud.NewLBTargetGroupRegistrationProvider(&aws_cloud.LBTargetGroupRegistrationProviderConfig{
			TargetGroupName: lbTargetGroupName,
		})
		if err != nil {
			log.Fatalf("Failed to create loadbalancer registration client: %v", err)
		}

		log.Info("Using loadbalancer target group cloud registration provider")
		return registrator
	default:
		log.Fatalf("Unsupported registration type: %v", awsRegistrationProvider)
		return nil
	}
}
