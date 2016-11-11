# etcd-bootstrap

Bootstrap etcd nodes in the cloud. `etcd-bootstrap` takes care of setting up etcd
to automatically join existing clusters and create new ones if necessary.

It's intended for use with AWS Auto Scaling groups and etcd2.

# Usage

Create instances inside of an ASG. Tag the ASG with `etcd-bootstrap/clusterName` 
and then run:

    ./etcd-bootstrap -o /var/run/bootstrap.conf
    source /var/run/bootstrap.conf
    ./etcd $ETCD_BOOTSTRAP_FLAGS

This will:

1. Query the ASG for all the instance IPs.
2. Determine if joining an existing cluster or not, by querying all instances
   to see if etcd is available.
3. Setup etcd arguments to correctly bootstrap with all the other instances.
