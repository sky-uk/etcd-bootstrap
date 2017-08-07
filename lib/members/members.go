package members

// Members represents the members of an etcd cluster.
type Members interface {
	// GetInstances returns all the non-terminated instances that will be part of the etcd cluster.
	GetInstances() []Instance
	// GetLocalInstance returns the local machine instance.
	GetLocalInstance() Instance
}

// Instance represents an instance inside of the auto scaling group.
type Instance struct {
	InstanceID string
	PrivateIP  string
}
