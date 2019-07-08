package cmd

import (
	"fmt"

	"github.com/sky-uk/etcd-bootstrap/bootstrap"
	"github.com/sky-uk/etcd-bootstrap/provider"

	log "github.com/sirupsen/logrus"
	aws_provider "github.com/sky-uk/etcd-bootstrap/provider/aws"
	"github.com/spf13/cobra"
)

const (
	noop registrationProvider = iota
	route53
	lb
)

var (
	awsRegistrationProviders = []string{
		"noop",
		"route53",
		"lb",
	}
)

const defaultAWSRegistrationProvider = "noop"

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
)

func init() {
	RootCmd.AddCommand(awsCmd)

	awsCmd.Flags().StringVarP(&awsRegistrationProvider, "registration-provider", "r", defaultAWSRegistrationProvider, fmt.Sprintf(
		"automatic registration provider to use, options are: %v", awsRegistrationProviders))
	awsCmd.Flags().StringVar(&route53ZoneID, "r53-zone-id", "",
		"route53 zone id for automatic registration (required when --registration-provider=route53)")
	awsCmd.Flags().StringVar(&dnsHostname, "dns-hostname", "",
		"hostname to set when registering the etcd cluster with route53 (required when --registration-provider=route53)")
	awsCmd.Flags().StringVar(&lbTargetGroupName, "lb-target-group-name", "",
		"loadbalancer target group name to use when registering the etcd cluster (required when: --registration-provider=lb)")
}

func aws(cmd *cobra.Command, args []string) {
	registrationProvider := initialiseAWSRegistrationProvider()

	awsProvider, err := aws_provider.NewAWS()
	if err != nil {
		log.Fatalf("Failed to create AWS provider: %v", err)
	}

	bootstrapper, err := bootstrap.New(awsProvider)
	if err != nil {
		log.Fatalf("Failed to create etcd bootstrapper: %v", err)
	}

	if err := bootstrapper.GenerateEtcdFlagsFile(outputFileName); err != nil {
		log.Fatalf("Failed to generate etcd flags file: %v", err)
	}

	if err := registrationProvider.Update(awsProvider.GetInstances()); err != nil {
		log.Fatalf("Failed to register etcd cluster data with cloud registration provider: %v", err)
	}
}

func initialiseAWSRegistrationProvider() provider.RegistrationProvider {
	// default to noop registration provider
	registrationProvider := provider.NewNoopRegistrationProvider()
	var err error

	switch awsRegistrationProvider {
	case noop.String(awsRegistrationProviders):
		log.Info("Using noop cloud registration provider")
	case route53.String(awsRegistrationProviders):
		checkRequired(route53ZoneID, "--r53-zone-id")
		checkRequired(dnsHostname, "--dns-hostname")

		registrationProvider, err = aws_provider.NewRoute53RegistrationProvider(&aws_provider.Route53RegistrationProviderConfig{
			ZoneID:   route53ZoneID,
			Hostname: dnsHostname,
		})
		if err != nil {
			log.Fatalf("Failed to create route53 registration client: %v", err)
		}

		log.Info("Using route53 cloud registration provider")
	case lb.String(awsRegistrationProviders):
		checkRequired(lbTargetGroupName, "--lb-target-group-name")

		registrationProvider, err = aws_provider.NewLBTargetGroupRegistrationProvider(&aws_provider.LBTargetGroupRegistrationProviderConfig{
			TargetGroupName: lbTargetGroupName,
		})
		if err != nil {
			log.Fatalf("Failed to create loadbalancer registration client: %v", err)
		}

		log.Info("Using loadbalancer target group cloud registration provider")
	default:
		log.Fatalf("Unsupported registration type: %v", awsRegistrationProvider)
	}

	log.Debugf("Registration provider created for: %v", awsRegistrationProvider)
	return registrationProvider
}
