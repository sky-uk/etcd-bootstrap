package main

import (
	"flag"

	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	"github.com/sky-uk/etcd-bootstrap/lib"
	"github.com/sky-uk/etcd-bootstrap/lib/members"
)

var vmwareUsername string
var vmwarePassword string
var vmwareIP string
var vmwarePort string
var vmwareInsecure bool
var vmwareAttempts uint
var vmwareVMName string
var vmwareEnv string
var vmwareRole string
var outputFilename string
var zoneID string
var domainName string

const (
	defaultVmwarePort     = "443"
	defaultVmwareInsecure = true
	defaultVmwareAttempts = 3
)

func init() {
	const defaultOutputFilename = "/var/run/etcd-bootstrap.conf"

	flag.StringVar(&vmwareUsername, "vmware-username", "",
		"username for vSphere API")
	flag.StringVar(&vmwarePassword, "vmware-password", "",
		"plaintext password for vSphere API")
	flag.StringVar(&vmwareIP, "vmware-ip", "",
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

	flag.StringVar(&outputFilename, "o", defaultOutputFilename,
		"location to write environment variables for etcd to use")
	flag.StringVar(&zoneID, "route53-zone-id", "",
		"route53 zone ID to update with the IP addresses of the etcd auto scaling group")
	flag.StringVar(&domainName, "route53-domain-name", "",
		"domain name to update inside the Route53 zone")
}

func main() {
	flag.Parse()

	out := "# created by etcd-bootstrap\n"
	validateArguments()

	var bootstrapper bootstrap.Bootstrapper
	var err error
	if vmwareIP != "" {
		config := &members.VmwareConfig{
			User:              vmwareUsername,
			Password:          vmwarePassword,
			VCenterIP:         vmwareIP,
			VCenterPort:       vmwarePort,
			InsecureFlag:      vmwareInsecure,
			RoundTripperCount: vmwareAttempts,
			VMName:            vmwareVMName,
		}

		bootstrapper, err = bootstrap.LocalVMWare(config, vmwareEnv, vmwareRole)
	} else {
		bootstrapper, err = bootstrap.LocalASG()
	}
	if err != nil {
		log.Fatalf("Unable to initialise bootstrapper: %v", err)
	}

	etcdOut, err := bootstrapper.BootstrapEtcdFlags()
	if err != nil {
		log.Fatalf("Unable to bootstrap etcd flags: %v", err)
	}

	out += etcdOut

	log.Infof("Writing environment variables to %s", outputFilename)
	if err := ioutil.WriteFile(outputFilename, []byte(out), 0644); err != nil {
		log.Fatalf("Unable to write to %s: %v", outputFilename, err)
	}

	if zoneID != "" && domainName != "" {
		log.Infof("Adding etcd IPs to %q in route53 zone %q", domainName, zoneID)
		if err := bootstrapper.BootstrapRoute53(zoneID, domainName); err != nil {
			log.Fatalf("Unable to bootstrap route53: %v", err)
		}
	}
}

func validateArguments() {
	if vmwareIP != "" && (zoneID != "" || domainName != "") {
		log.Warn("Route53 zone setup cannot be run on VMWare")
	}
}
