package cloud

// Instance represents a cloud instance which is intended to be part of an etcd cluster.
type Instance struct {
	// Name is the unique name to identify this instance in an etcd cluster.
	Name string

	// Endpoint is the address to reach this instance from an etcd client.
	// It is used to construct the peer and client URLs.
	// It should be of the form `hostname` or `x.x.x.x`.
	Endpoint string
}
