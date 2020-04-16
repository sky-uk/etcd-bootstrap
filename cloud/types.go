package cloud

// Instance represents a cloud instance which is intended to be part of an etcd cluster.
type Instance struct {
	InstanceID string
	PrivateIP  string
}
