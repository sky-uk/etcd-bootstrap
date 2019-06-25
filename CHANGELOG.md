# v1.3.0

If you are using the DNS record created by etcd bootstrap this is a breaking change to the command-line arguments
required by the application:

* Add `-registration-type` flag (default: `none`) which sets how etcd bootstrap will register the cluster. Options are:
    - none: don't attempt to register the cluster externally, this will simply create the etcd config for the cluster
    - dns: after generating the cluster config, create a DNS record containing multiple A records if supported by the cloud
    provider (requires `-route53-zone-id` and `-domain-name` to be set)
    - lb: after generating the cluster config, add the etcd instances to a loadbalancer target group if supported by the
    cloud provider (requires `-aws-lb-target-group` to be set)

# v1.2.1

* Update alpine image to fix CVE-2019-5021.
* Update to go version 1.11

# v1.2.0

* Split `start-etcd.sh` into `bootstrap.sh` and `etcd.sh`, to allow etcd-bootstrap to be used with coreos etcd v3+ images.

# v1.1.1

* Fix vmware docker image.

# v1.1.0

* Add support for discovering cluster members using the GCP API.
* Renames aws image to cloud.
* Updates etcd version to 2.3.8.

# v1.0.0

Add support for discovering cluster members using the VMWare vSphere API.
 Changes: https://github.com/sky-uk/etcd-bootstrap/pull/11

This is a breaking change to the command-line arguments required by the application:

* There is a new mandatory command-line argument, "cloud", which takes values "aws" or "vmware" to select the
  appropriate cloud provider to use as a backend.
* Command-line argument "route53-domain-name" has been renamed to "domain-name" so as to be applicable to multiple cloud
  providers.

# v0.1.1

First official release.
