package bootstrap

import (
	"github.com/sky-uk/etcd-bootstrap/lib/dns"
	"github.com/sky-uk/etcd-bootstrap/lib/etcdcluster"
	"github.com/sky-uk/etcd-bootstrap/lib/members"
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
	members members.Members
	cluster etcdcluster.Cluster
	r53     dns.R53
}

// LocalASG creates a bootstrapper wired to the local ASG.
func LocalASG() (Bootstrapper, error) {
	asg, err := members.NewAws()
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
func LocalVMWare(configLocation, env, role string) (Bootstrapper, error) {
	vmware, err := members.NewVMware(configLocation, env, role)
	if err != nil {
		return nil, err
	}
	return newBootstrapper(vmware, nil)
}

func newBootstrapper(members members.Members, r53 dns.R53) (Bootstrapper, error) {
	etcd, err := etcdcluster.New(members)
	if err != nil {
		return nil, err
	}
	return New(members, etcd, r53), nil
}

// New creates a new bootstrapper.
func New(members members.Members, cluster etcdcluster.Cluster, awsR53 dns.R53) Bootstrapper {
	return &bootstrapper{members, cluster, awsR53}
}
