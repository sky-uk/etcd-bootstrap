![travis](https://travis-ci.org/sky-uk/etcd-bootstrap.svg?branch=master)

# etcd-bootstrap

Bootstrap etcd nodes in the cloud. `etcd-bootstrap` takes care of setting up etcd
to automatically join existing clusters and create new ones if necessary.

It's intended for use with etcd2 and one of:
  * An AWS Auto Scaling group; or
  * A vSphere server; or
  * A GCP Managed Instance group

The cloud type used is determined by the `-cloud (aws|vmware|gcp)` command line argument

## cloud-etcd container

etcd2 container that bootstraps in the cloud. Run it the same as the etcd container:

    docker run skycirrus/cloud-etcd-v2.3.8:1.2.0 -h # lists all the etcd options

You can pass in any flag that etcd takes normally.


Alternatively, cloud-etcd may be used to generate a startup script for etcd only, which may then be used with this
image, or v3+ images.

    docker run -v BOOTSTRAP_DIR:/bootstrap --entrypoint=/bootstrap.sh skycirrus/cloud-etcd-v2.3.8:1.2.0 -h # lists all the etcd-bootstrap options

And then to run etcd v2 using this image:

    docker run -v BOOTSTRAP_DIR:/bootstrap --entrypoint=/bootstrap/etcd.sh skycirrus/cloud-etcd-v2.3.8:1.2.0 -h # lists all the etcd options
    
Or for etcd v3+ using the coreos image:

    docker run -v BOOTSTRAP_DIR:/bootstrap --entrypoint=/bootstrap/etcd.sh quay.io/coreos/etcd:v3.2 -h # lists all the etcd options 

The startup script is generated and placed into the volume mounted bootstrap directory. This must be mounted into the
container which will run etcd, and the entrypoint must be changed to the generated script. The location of this
bootstrap directory may be customised by using `-e BOOTSTRAP_DIR=[NEW LOCATION]`.

### AWS

The wrapper will take care of setting all the required flags to create or join an existing
etcd cluster, based on the ASG the local instance is on.

To pass flags to etcd-bootstrap, set the `ETCD_BOOTSTRAP_FLAGS` environment variable.

    docker run -e ETCD_BOOTSTRAP_FLAGS='-cloud aws -registration-type dns -route53-zone-id MY_ZONE_ID -domain-name etcd' skycirrus/cloud-etcd-v2.3.8:1.1.0

### GCP

The wrapper will take care of setting all the required flags to create or join an existing
etcd cluster, based on the flags provided.

To pass flags to etcd-bootstrap, set the `ETCD_BOOTSTRAP_FLAGS` environment variable.

    docker run -e ETCD_BOOTSTRAP_FLAGS='-cloud gcp -gcp-project-id MY_PROJECT_ID -gcp-environment MY_ENV -gcp-role ETCD_ROLE etcd' skycirrus/cloud-etcd-v2.3.8:1.1.0

## vmware-etcd container

On top of the functionality provided by the cloud-etcd container, the vmware-etcd container has support to read
credentials from a user specified file.

To use this feature, set the `VMWARE_CREDENTIALS` environment variable with reference to the location of the file that
contains your credentials. e.g.

```bash
VMWARE_USERNAME=myusername
VMWARE_PASSWORD=supersecret
```

If the `VMWARE_CREDENTIALS` environment variable is then set, the wrapper script will then source this file and then
append the appropriate flags to `ETCD_BOOTSTRAP_FLAGS`.

    docker run -e VMWARE_CREDENTIALS='/etc/vmware-credentials' \
               -e ETCD_BOOTSTRAP_FLAGS='-cloud vmware ...' \
               skycirrus/vmware-etcd-v2.3.8:1.2.0

## Command usage

Create instances inside of an ASG, vSphere, or GCP. Then run:

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

    ./etcd-bootstrap -o /var/run/bootstrap.conf -cloud aws -registration-type dns -route53-zone-id MYZONEID -domain-name etcd

If zone `MYZONEID` has domain name `example.com`, this will update the domain name `etcd.example.com` with all
of the IPs. This lets clients use round robin DNS for connecting to the cluster.

`-route53-zone-id` is **not supported** when using `-cloud (vmware|gcp)`.

## Cloud-specific notes

### AWS

AWS nodes must be created in an ASG of which the node on which the container runs must be a part.

### Registration types

AWS supports both `dns` and `lb` registration types.

##### dns

If running etcd bootstrap with `-registration-type=dns` this will create a route53 record containing all etcd instance
ip addresses as A records. It will create it in the zone supplied using `-route53-zone-id` and the domain supplied by 
`-domain-name` (both flags are required when using this registration type).

##### lb

If running etcd bootsrap with `-registration-type=lb` this will attempt to register all etcd instances with an AWS
loadbalancer target group with the name supplied by `-aws-lb-target-group` (flag is required when using this
registration type).

#### IAM role

Instances must have one of the following IAM policy rules:

##### Registration type: none 

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances",
        "autoscaling:DescribeAutoScaling*",
      ],
      "Resource": "*"
    }
  ]
}

```

##### Registration type: dns 

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

##### Registration type: lb 

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances",
        "autoscaling:DescribeAutoScaling*",
        "elasticloadbalancing:RegisterTargets",
        "elasticloadbalancing:DescribeTargetGroups"
      ],
      "Resource": "*"
    }
  ]
}

```

### VMWare

The VMWare mode requires configuring with connectivity information to the vSphere VCenter API.  See usage help for
required arguments.  In addition to connectivity, the VMWare mode requires two further options:

 * `vmware-environment` - the environment to filter
 * `vmware-role` - the role to filter

In order for the environment and role filters to work, the VMs must have been provisioned with extra configuration
parameters named "tags_environment" and "tags_role" set to the values provided on the command line.

### GCP

GCP nodes must be created in an MIG of which the node on which the container runs must be a part.
The GCP cloud mode requires additional options:

 * `gcp-project-id` - the name of the project to query
 * `gcp-environment` - the name of the environment to filter
 * `gcp-role` - the role to filter

In order for the role filters to work, the VMs must have been provisioned with extra configuration
labels named "environment" and "role" set to the values provided on the command line.

In case a node has multiple Network Interfaces, the GCP bootstrapper will take the
private ip of the first available one.