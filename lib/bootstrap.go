package bootstrap

import (
	"github.com/sky-uk/etcd-bootstrap/lib/cloud"
	"github.com/sky-uk/etcd-bootstrap/lib/dns"
	"github.com/sky-uk/etcd-bootstrap/lib/etcdcluster"
)

// Bootstrapper bootstraps etcd.
type Bootstrapper interface {
	// BootstrapEtcdFlags creates a file with etcd flags based on the ASG status.
	// It's intended to be sourced in startup scripts.
	BootstrapEtcdFlags() (string, error)
	// BootstrapRoute53 adds the etcd instances to a route53 entry for external access.
	BootstrapRoute53(zoneID, name string) error
}

type bootstrapper struct {
	cloud   cloud.Cloud
	cluster etcdcluster.Cluster
	r53     dns.R53
}

// LocalASG creates a bootstrapper wired to the local ASG.
func LocalASG() (Bootstrapper, error) {
	asg, err := cloud.NewAws()
	if err != nil {
		return nil, err
	}
	r53, err := dns.New()
	if err != nil {
		return nil, err
	}
	return newBootstrapper(asg, r53)
}

// LocalVMWare creates a bootstrapper wired to the VMWare vSphere controller
func LocalVMWare(config *cloud.VmwareConfig) (Bootstrapper, error) {
	vmware, err := cloud.NewVMware(config)
	if err != nil {
		return nil, err
	}
	return newBootstrapper(vmware, nil)
}

func newBootstrapper(members cloud.Cloud, r53 dns.R53) (Bootstrapper, error) {
	etcd, err := etcdcluster.New(members)
	if err != nil {
		return nil, err
	}
	return New(members, etcd, r53), nil
}

// New creates a new bootstrapper.
func New(members cloud.Cloud, cluster etcdcluster.Cluster, awsR53 dns.R53) Bootstrapper {
	return &bootstrapper{members, cluster, awsR53}
}
