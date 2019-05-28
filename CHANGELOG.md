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
