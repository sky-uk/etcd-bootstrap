package main

import (
	"flag"

	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	"github.com/sky-uk/etcd-bootstrap/lib"
)

var vmwareConfigLocation string
var vmwareEnv string
var vmwareRole string
var outputFilename string
var zoneID string
var domainName string

func init() {
	const defaultOutputFilename = "/var/run/etcd-bootstrap.conf"

	flag.StringVar(&vmwareConfigLocation, "vmware-config-location", "",
		"location of the vSphere cloud provider configuration file")
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
	if vmwareConfigLocation != "" {
		bootstrapper, err = bootstrap.LocalVMWare(vmwareConfigLocation, vmwareEnv, vmwareRole)
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
	if vmwareConfigLocation != "" && (zoneID != "" || domainName != "") {
		log.Warn("Route53 zone setup cannot be run on VMWare")
	}
}