package noop

import (
	log "github.com/sirupsen/logrus"
	"github.com/sky-uk/etcd-bootstrap/cloud"
)

// RegistrationProvider that performs a noop Update.
type RegistrationProvider struct{}

// Update is a noop.
func (n RegistrationProvider) Update(instances []cloud.Instance) error {
	log.Info("Registration provider set to noop")
	return nil
}
