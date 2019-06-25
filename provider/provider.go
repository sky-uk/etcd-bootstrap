package provider

import log "github.com/sirupsen/logrus"

// Provider represents a cloud provider which can discover etcd cluster configuration.
type Provider interface {
	// GetInstances returns all the non-terminated instances that will be part of the etcd cluster.
	GetInstances() []Instance
	// GetLocalInstance returns the local machine instance.
	GetLocalInstance() Instance
}

type RegistrationProvider interface {
	Update(instances []Instance) error
}

// Instance represents an instance which will form an etcd cluster.
type Instance struct {
	InstanceID string
	PrivateIP  string
}

func NewNoopRegistrationProvider() RegistrationProvider {
	return noopRegistrationProvider{}
}

type noopRegistrationProvider struct{}

func (n noopRegistrationProvider) Update(instances []Instance) error {
	log.Debugf("Registration provider set to noop")
	return nil
}
