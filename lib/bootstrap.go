package bootstrap

import (
	"github.com/sky-uk/etcd-bootstrap/lib/asg"
	"github.com/sky-uk/etcd-bootstrap/lib/etcdcluster"
	"github.com/sky-uk/etcd-bootstrap/lib/r53"
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
	asg     asg.ASG
	cluster etcdcluster.Cluster
	r53     r53.R53
}

// LocalASG creates a bootstrapper wired to the local ASG.
func LocalASG() (Bootstrapper, error) {
	asg, err := asg.New()
	if err != nil {
		return nil, err
	}
	r53, err := r53.New()
	if err != nil {
		return nil, err
	}
	etcd, err := etcdcluster.New(asg)
	if err != nil {
		return nil, err
	}
	return New(asg, etcd, r53), nil
}

// New creates a new bootstrapper.
func New(awsASG asg.ASG, cluster etcdcluster.Cluster, awsR53 r53.R53) Bootstrapper {
	return &bootstrapper{awsASG, cluster, awsR53}
}
