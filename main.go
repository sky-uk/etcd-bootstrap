package main

import (
	"flag"

	"io/ioutil"

	"fmt"

	log "github.com/Sirupsen/logrus"
	bootstrap "github.com/sky-uk/etcd-bootstrap/lib"
	"github.com/sky-uk/etcd-bootstrap/lib/aws"
	"github.com/sky-uk/etcd-bootstrap/lib/gcp"
	"github.com/sky-uk/etcd-bootstrap/lib/vmware"
)

var (
	cloudProvider    string
	registrationType string
	r53ZoneID        string
	domainName       string
	outputFilename   string
	awsLBTargetGroup string
	vmwareUsername   string
	vmwarePassword   string
	vmwareHost       string
	vmwarePort       string
	vmwareInsecure   bool
	vmwareAttempts   uint
	vmwareVMName     string
	vmwareEnv        string
	vmwareRole       string
	gcpProjectID     string
	gcpEnv           string
	gcpRole          string
)

const (
	defaultRegistrationType = "none"
	defaultOutputFilename   = "/var/run/etcd-bootstrap.conf"
	defaultVmwarePort       = "443"
	defaultVmwareInsecure   = true
	defaultVmwareAttempts   = 3
)

func init() {
	// default flags
	flag.StringVar(&cloudProvider, "cloud", "",
		"cloud provider to use.  Required, and must be one of 'aws' or 'vmware'")
	flag.StringVar(&registrationType, "registration-type", defaultRegistrationType, "set etcd "+
		"registration type (options: none, dns, lb)")
	flag.StringVar(&r53ZoneID, "route53-zone-id", "",
		"route53 zone ID to update with the IP addresses of the etcd auto scaling group")
	flag.StringVar(&domainName, "domain-name", "",
		"domain name to update inside the DNS provider, eg. 'etcd' (only required for -registration-type=dns)")
	flag.StringVar(&outputFilename, "o", defaultOutputFilename,
		"location to write environment variables for etcd to use")

	// aws flags
	flag.StringVar(&awsLBTargetGroup, "aws-lb-target-group", "",
		"aws loadbalancer target group name to update with the etcd instances")

	// vmware flags
	flag.StringVar(&vmwareUsername, "vmware-username", "",
		"username for vSphere API")
	flag.StringVar(&vmwarePassword, "vmware-password", "",
		"plaintext password for vSphere API")
	flag.StringVar(&vmwareHost, "vmware-host", "",
		"host address for vSphere API")
	flag.StringVar(&vmwarePort, "vmware-port", defaultVmwarePort,
		"port for vSphere API.  Defaults to 443")
	flag.BoolVar(&vmwareInsecure, "vmware-insecure", defaultVmwareInsecure,
		"flag to indicate if vSphere API uses a self-signed certificate")
	flag.UintVar(&vmwareAttempts, "vmware-attempts", defaultVmwareAttempts,
		"number of attempts to make against the vSphere SOAP API (in case of temporary failure)")
	flag.StringVar(&vmwareVMName, "vmware-vm-name", "",
		"node name in vSphere of this VM")
	flag.StringVar(&vmwareEnv, "vmware-environment", "",
		"value of the 'tags_environment' extra configuration option in vSphere to filter nodes by")
	flag.StringVar(&vmwareRole, "vmware-role", "",
		"value of the 'tags_role' extra configuration option in vSphere to filter nodes by")

	// gcp flags
	flag.StringVar(&gcpProjectID, "gcp-project-id", "",
		"value of the GCP 'project id' to query")
	flag.StringVar(&gcpEnv, "gcp-environment", "",
		"value of the 'environment' label in GCP nodes to filter them by")
	flag.StringVar(&gcpRole, "gcp-role", "",
		"value of the 'role' label in GCP nodes to filter them by")
}

func main() {
	flag.Parse()

	bootstrapper, err := createBootstrapper()
	if err != nil {
		log.Fatalf("Unable to initialise bootstrapper: %v", err)
	}

	etcdOut, err := bootstrapper.BootstrapEtcdFlags()
	if err != nil {
		log.Fatalf("Unable to bootstrap etcd flags: %v", err)
	}

	out := "# created by etcd-bootstrap\n"
	out += etcdOut

	log.Infof("Writing environment variables to %s", outputFilename)
	if err := ioutil.WriteFile(outputFilename, []byte(out), 0644); err != nil {
		log.Fatalf("Unable to write to %s: %v", outputFilename, err)
	}

	switch registrationType {
	case "dns":
		log.Infof("Adding etcd IPs to %q in DNS", domainName)
		if domainName == "" {
			log.Fatalf("-domain-name must be set when using dns registration type")
		}
		if err := bootstrapper.BootstrapDNS(domainName); err != nil {
			log.Fatalf("Unable to bootstrap DNS: %v", err)
		}
	case "lb":
		log.Infof("Registering etcd targets with loadbalancer")
		if err := bootstrapper.BootstrapLB(); err != nil {
			log.Fatalf("Unable to bootstrap LB: %v", err)
		}
	case "none":
		log.Infof("Skipping etcd bootstrap registration")
	}
}

func createBootstrapper() (bootstrap.Bootstrapper, error) {
	var bootstrapper bootstrap.Bootstrapper
	var err error
	switch cloudProvider {
	case "vmware":
		config := &vmware.Config{
			User:              vmwareUsername,
			Password:          vmwarePassword,
			VCenterHost:       vmwareHost,
			VCenterPort:       vmwarePort,
			InsecureFlag:      vmwareInsecure,
			RoundTripperCount: vmwareAttempts,
			VMName:            vmwareVMName,
			Environment:       vmwareEnv,
			Role:              vmwareRole,
		}

		bootstrapper, err = bootstrap.LocalVMWare(config)
	case "aws":
		config := &aws.Config{
			R53ZoneID:         r53ZoneID,
			LBTargetGroupName: awsLBTargetGroup,
		}
		bootstrapper, err = bootstrap.LocalAWS(config)
	case "gcp":
		config := &gcp.Config{
			ProjectID:   gcpProjectID,
			Environment: gcpEnv,
			Role:        gcpRole,
		}
		bootstrapper, err = bootstrap.GCP(config)
	default:
		err = fmt.Errorf("unknown cloud provider '%s': must be 'aws', 'vmware' or 'gcp'", cloudProvider)
	}
	return bootstrapper, err
}
