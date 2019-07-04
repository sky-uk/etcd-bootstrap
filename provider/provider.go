package provider

import log "github.com/sirupsen/logrus"

// Provider represents a cloud provider which can discover etcd cluster configuration.
type Provider interface {
	// GetInstances returns all the non-terminated instances that will be part of the etcd cluster.
	GetInstances() []Instance
	// GetLocalInstance returns the local machine instance.
	GetLocalInstance() Instance
}

// RegistrationProvider represents a cloud registration provider which can update an external service with information
// about the bootstrapped etcd cluster (such as dns or loadbalancer pools)
type RegistrationProvider interface {
	// Update will update a registration provider with information about the etcd cluster using the discovered instances
	Update(instances []Instance) error
}

// Instance represents an instance which will form an etcd cluster.
type Instance struct {
	InstanceID string
	PrivateIP  string
}

// NewNoopRegistrationProvider can be used by cloud providers when they want to support optionally registering with
// external registration providers
func NewNoopRegistrationProvider() RegistrationProvider {
	return noopRegistrationProvider{}
}

type noopRegistrationProvider struct{}

func (n noopRegistrationProvider) Update(instances []Instance) error {
	log.Debugf("Registration provider set to noop")
	return nil
}
