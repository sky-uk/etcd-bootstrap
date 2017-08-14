package bootstrap

import (
	"github.com/sky-uk/etcd-bootstrap/lib/aws"
	"github.com/sky-uk/etcd-bootstrap/lib/cloud"
	"github.com/sky-uk/etcd-bootstrap/lib/etcdcluster"
	"github.com/sky-uk/etcd-bootstrap/lib/vmware"
)

// Bootstrapper bootstraps etcd.
type Bootstrapper interface {
	// BootstrapEtcdFlags creates a file with etcd flags based on the ASG status.
	// It's intended to be sourced in startup scripts.
	BootstrapEtcdFlags() (string, error)
	// BootstrapDNS adds the etcd instances to the DNS provider for the given cloud.
	BootstrapDNS(name string) error
}

type bootstrapper struct {
	cloud   cloud.Cloud
	cluster etcdcluster.Cluster
}

func (b *bootstrapper) BootstrapDNS(name string) error {
	return b.cloud.UpdateDNS(name)
}

// LocalASG creates a bootstrapper wired to the local ASG.
func LocalASG(zoneID string) (Bootstrapper, error) {
	asg, err := aws.NewAws(zoneID)
	if err != nil {
		return nil, err
	}
	return newBootstrapper(asg)
}

// LocalVMWare creates a bootstrapper wired to the VMWare vSphere controller
func LocalVMWare(config *vmware.Config) (Bootstrapper, error) {
	vmware, err := vmware.NewVMware(config)
	if err != nil {
		return nil, err
	}
	return newBootstrapper(vmware)
}

func newBootstrapper(members cloud.Cloud) (Bootstrapper, error) {
	etcd, err := etcdcluster.New(members)
	if err != nil {
		return nil, err
	}
	return New(members, etcd), nil
}

// New creates a new bootstrapper.
func New(members cloud.Cloud, cluster etcdcluster.Cluster) Bootstrapper {
	return &bootstrapper{members, cluster}
}
