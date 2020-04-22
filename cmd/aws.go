package cmd

import (
	"fmt"
	"net"

	"github.com/sky-uk/etcd-bootstrap/bootstrap"
	"github.com/sky-uk/etcd-bootstrap/cloud"
	"github.com/sky-uk/etcd-bootstrap/etcd"

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
	enableTLS               bool
	clientCA                string
	clientCert              string
	clientKey               string
	peerCA                  string
	peerCert                string
	peerKey                 string
)

func init() {
	RootCmd.AddCommand(awsCmd)
	f := awsCmd.Flags()
	f.StringVarP(&awsRegistrationProvider, "registration-provider", "r", "noop", fmt.Sprintf(
		"automatic registration provider to use, options are: noop, lb, route53"))
	f.StringVar(&route53ZoneID, "r53-zone-id", "",
		"zone id for automatic registration for registration-provider=route53")
	f.StringVar(&dnsHostname, "dns-hostname", "",
		"hostname to set to the etcd cluster when registration-provider=route53")
	f.StringVar(&lbTargetGroupName, "lb-target-group-name", "",
		"loadbalancer target group name to use when --registration-provider=lb")
	f.StringVar(&instanceLookupMethod, "instance-lookup-method", "asg",
		"method for looking up instances in the cluster, options are: asg, srv")
	f.StringVar(&srvDomainName, "srv-domain-name", "", "domain name to use for instance-lookup-method=srv")
	f.StringVar(&srvService, "srv-service", "etcd-bootstrap", "service to use for instance-lookup-method=srv")
	f.BoolVar(&enableTLS, "enable-tls", false, "enable TLS")
	f.StringVar(&clientCA, "tls-client-ca", "", "path to client CA")
	f.StringVar(&clientCert, "tls-client-cert", "", "path to client certificate")
	f.StringVar(&clientKey, "tls-client-key", "", "path to client key")
	f.StringVar(&peerCA, "tls-peer-ca", "", "path to peer CA")
	f.StringVar(&peerCert, "tls-peer-cert", "", "path to peer certificate")
	f.StringVar(&peerKey, "tls-peer-key", "", "path to peer key")
}

func aws(cmd *cobra.Command, args []string) {
	aws, err := aws_cloud.NewAWS()
	if err != nil {
		log.Fatalf("Failed to create AWS provider: %v", err)
	}

	cloudInstances := createCloudInstances(aws)
	etcdCluster := createEtcdClusterAPI(cloudInstances)

	var opts []bootstrap.Option
	if enableTLS {
		opts = []bootstrap.Option{bootstrap.WithTLS(clientCA, clientCert, clientKey, peerCA, peerCert, peerKey)}
	}
	bootstrapper, err := bootstrap.New(cloudInstances, aws, etcdCluster, opts...)
	if err != nil {
		log.Fatalf("Failed to create etcd bootstrapper: %v", err)
	}

	if err := bootstrapper.GenerateEtcdFlagsFile(outputFilename); err != nil {
		log.Fatalf("Failed to generate etcd flags file: %v", err)
	}

	registerInstances(cloudInstances)
}

type localIPResolver struct {
	aws *aws_cloud.AWS
}

func (l *localIPResolver) LookupLocalIP() (net.IP, error) {
	localInstance, err := l.aws.GetLocalInstance()
	if err != nil {
		return net.IP{}, err
	}
	// Assumes the aws endpoint is always in IP format.
	ip := net.ParseIP(localInstance.Endpoint)
	if ip == nil {
		panic("expected aws.GetLocalInstance() to return an IP endpoint, but was " + localInstance.Endpoint)
	}
	return ip, nil
}

func createCloudInstances(aws *aws_cloud.AWS) bootstrap.CloudInstances {
	switch instanceLookupMethod {
	case "asg":
		log.Info("Using ASG for looking up cluster instances")
		return aws
	case "srv":
		log.Info("Using SRV record for looking up cluster instances")
		if srvDomainName == "" {
			log.Fatalf("srv-domain-name must be provided")
		}
		if srvService == "" {
			log.Fatalf("srv-service must be provided")
		}
		return srv.New(srvDomainName, srvService, &localIPResolver{aws: aws})
	default:
		log.Fatalf("Unsupported cluster lookup method %q", instanceLookupMethod)
		return nil
	}
}

func registerInstances(cloudInstances bootstrap.CloudInstances) {
	instances, err := cloudInstances.GetInstances()
	if err != nil {
		log.Fatalf("Failed to retrieve instances: %v", err)
	}
	registrator := initialiseAWSRegistrationProvider()
	if err := registrator.Update(instances); err != nil {
		log.Fatalf("Failed to register etcd cluster data with cloud registration provider: %v", err)
	}
}

type registrationProvider interface {
	Update([]cloud.Instance) error
}

func createEtcdClusterAPI(instances etcd.Instances) *etcd.ClusterAPI {
	var etcdOpts []etcd.Option
	if enableTLS {
		etcdOpts = []etcd.Option{etcd.WithTLS(peerCA, peerCert, peerKey)}
	}
	etcdCluster, err := etcd.New(instances, etcdOpts...)
	if err != nil {
		log.Fatalf("Failed to create etcd cluster API: %v", err)
	}
	return etcdCluster
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
