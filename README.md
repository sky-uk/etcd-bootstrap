![travis](https://travis-ci.org/sky-uk/etcd-bootstrap.svg?branch=master)

# etcd-bootstrap

Bootstrap etcd nodes in the cloud. `etcd-bootstrap` takes care of setting up etcd
to automatically join existing clusters and create new ones if necessary.

It's intended for use with AWS Auto Scaling groups and etcd2.

## aws-etcd container

etcd2 container that bootstraps in AWS. Run it the same as the etcd container:

    docker run skycirrus/aws-etcd-v2.3.7 -h

This will take care of setting all the required flags to create or join an existing
etcd cluster, based on the ASG the local instance is on.

## Command usage

Create instances inside of an ASG. Then run: 

    ./etcd-bootstrap -o /var/run/bootstrap.conf
    source /var/run/bootstrap.conf
    ./etcd

This will:

1. Query the ASG for all the instance IPs.
2. Determine if joining an existing cluster or not, by querying all instances
   to see if etcd is available.
3. Create `ETCD_*` environment variables to correctly bootstrap with all the
   other instances.

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
      ],
      "Resource": "*"
    }
  ]
}

```
