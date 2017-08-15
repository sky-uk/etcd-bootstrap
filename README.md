![travis](https://travis-ci.org/sky-uk/etcd-bootstrap.svg?branch=master)

# etcd-bootstrap

Bootstrap etcd nodes in the cloud. `etcd-bootstrap` takes care of setting up etcd
to automatically join existing clusters and create new ones if necessary.

It's intended for use with etcd2 and one of:
  * An AWS Auto Scaling group; or
  * A vSphere server

The cloud type used is determined by the `-cloud (aws|vmware)` command line argument

## aws-etcd container

etcd2 container that bootstraps in the cloud. Run it the same as the etcd container:

    docker run skycirrus/aws-etcd-v2.3.7:1.0.0 -h # lists all the etcd options

You can pass in any flag that etcd takes normally.

The wrapper will take care of setting all the required flags to create or join an existing
etcd cluster, based on the ASG the local instance is on.

To pass flags to etcd-bootstrap, set the `ETCD_BOOTSTRAP_FLAGS` environment variable.

    docker run -e ETCD_BOOTSTRAP_FLAGS='-cloud aws -route53-zone-id MYZONEID -domain-name etcd' skycirrus/aws-etcd-v2.3.7:1.0.0

## Command usage

Create instances inside of an ASG or in vSphere. Then run:

    ./etcd-bootstrap -o /var/run/bootstrap.conf
    source /var/run/bootstrap.conf
    ./etcd

This will:

1. Query the relevant API for all the instance IPs.
2. Determine if joining an existing cluster or not, by querying all instances
   to see if etcd is available.
3. Create `ETCD_*` environment variables to correctly bootstrap with all the
   other instances.

### Updating a Route53 domain name for the etcd cluster members

Optionally etcd-bootstrap can also register all the IPs in the autoscaling group with a domain name.

    ./etcd-bootstrap -o /var/run/bootstrap.conf -cloud aws -route53-zone-id MYZONEID -domain-name etcd

If zone `MYZONEID` has domain name `example.com`, this will update the domain name `etcd.example.com` with all
of the IPs. This lets clients use round robin DNS for connecting to the cluster.

## IAM role

Instances must have the following IAM policy rules:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances",
        "autoscaling:DescribeAutoScaling*",
        "route53:ChangeResourceRecordSets",
        "route53:GetHostedZone"
      ],
      "Resource": "*"
    }
  ]
}

```

## Cloud-specific notes

### AWS

AWS nodes must be created in an ASG of which the node on which the container runs must be a part.

### VMWare

The VMWare mode requires configuring with connectivity information to the vSphere VCenter API.  See usage help for
required arguments.  In addition to connectivity, the VMWare mode requires two further options:

 * `vmware-environment` - the environment to filter
 * `vmware-role` - the role to filter

In order for the environment and role filters to work, the VMs must have been provisioned with extra configuration
parameters named "tags_environment" and "tags_role" set to the values provided on the command line.