![travis](https://travis-ci.org/sky-uk/etcd-bootstrap.svg?branch=master)

# etcd-bootstrap

Bootstrap etcd nodes for cloud and vmware. `etcd-bootstrap` takes care of setting up etcd to automatically generate
configuration and optionally register the cluster with a provider. It is intended to be used as an initialisation tool
which is run before starting an etcd instance (e.g. kubernetes init-containers).

It currently supports use with etcd and one of:
  * An AWS Auto Scaling group; or
  * A vSphere server; or
  * A GCP Managed Instance group

The provider type used is determined by the parameter passed after `etcd-bootstrap` and the options can be listed by
running `./etcd-boostrap -h`. Once you have selected a provider to use, you can list the various flags supported by
running `./etcd-bootsrap <provider> -h`.

### AWS

When using the AWS provider, by default etcd-bootstrap will get information about the instance it is running on (must
be running on an AWS EC2 instance and be part of one AWS autoscaling group). It will discover the autoscaling group the
node is part of and then discover all other instances within that group. From this information, configuration will be
generated to either join or create a new etcd cluster based on the etcd cluster state.

#### Provider Flags:

| Flag | Default | Comment |
| ---- | -------- | ------- |
| `--registration-provider` | `noop` | select the registration provider to use (either: dns, lb or noop) |
| `--r53-zone-id` | `n/a` | the zone to use when using the dns registration provider |
| `--dns-hostname` | `n/a` | the dns hostname to use when using the dns registration provider |
| `--lb-target-group-name` | `n/a` | the aws loadbalancer target group name when using the lb registration provider |

#### Registration Providers

##### Route53

If running etcd bootstrap with `-registration-type=dns` this will create a route53 record containing all etcd instance
ip addresses as A records. It will create it in the zone supplied using `-route53-zone-id` and the domain supplied by 
`-domain-name` (both flags are required when using this registration type).

Optionally etcd-bootstrap can also register all the IPs in the autoscaling group with a domain name.

    ./etcd-bootstrap -o /var/run/bootstrap.conf -cloud aws -registration-type dns -route53-zone-id MYZONEID -domain-name etcd

If zone `MYZONEID` has domain name `example.com`, this will update the domain name `etcd.example.com` with all
of the IPs. This lets clients use round robin DNS for connecting to the cluster.

##### AWS Loadbalancer Target Group

If running etcd bootsrap with `-registration-type=lb` this will attempt to register all etcd instances with an AWS
loadbalancer target group with the name supplied by `-aws-lb-target-group` (flag is required when using this
registration type).

##### Example Kubernetes Pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: etcd
spec:
  initContainers:
  - name: etcd-bootstrap
    image: skycirrus/etcd-boostrap:v2.0.0
    args:
    - --registration-provider=lb
    - --lb-target-group-name=my-aws-target-group
  containers:
  - name: etcd
    image: quay.io/coreos/etcd:v3.1.12
    args:
    - --data-dir {{ etcd_cluster_data }}
    - --heartbeat-interval 200
    - --election-timeout 2000
    livenessProbe:
      tcpSocket:
        port: clientport
      initialDelaySeconds: 15
      timeoutSeconds: 15
    readinessProbe:
      httpGet:
        path: /health
        port: clientport
        scheme: HTTP
      initialDelaySeconds: 15
      timeoutSeconds: 15
    ports:
    - containerPort: 2380
      hostPort: 2380
      name: peerport
      protocol: TCP
    - containerPort: 2379
      hostPort: 2379
      name: clientport
      protocol: TCP
```

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

### GCP

#### Provider Flags:

| Flag | Default | Comment |
| ---- | -------- | ------- |
| `--project-id` | `n/a` | the name of the project to query |
| `--environment` | `n/a` | the name of the environment to filter |
| `--role` | `n/a` | the role to filter |

#### Notes

GCP nodes must be created in an MIG of which the node on which the container runs must be a part. In order for the role
filters to work, the VMs must have been provisioned with extra configuration labels named "environment" and "role" set
to the values provided on the command line.

In case a node has multiple Network Interfaces, the GCP bootstrapper will take the
private ip of the first available one.

### VMWare

#### Provider Flags:

| Flag | Default | Comment |
| ---- | -------- | ------- |
| `--vsphere-username` | `n/a` | username for vSphere API |
| `--vsphere-host` | `n/a` | host address for vSphere API |
| `--vsphere-port` | `443` | port for vSphere API |
| `--insecure-skip-verify` | `false` | skip SSL verification when communicating with the vSphere host |
| `--max-api-attempts` | `3` | number of attempts to make against the vSphere SOAP API (in case of temporary failure) |
| `--vm-name` | `n/a` | node name in vSphere of this VM |
| `--environment` | `n/a` | value of the 'tags_environment' extra configuration option in vSphere to filter nodes by |
| `--role` | `n/a` | value of the 'tags_role' extra configuration option in vSphere to filter nodes by |

#### Provider Environment Variables:

| ENV | Default | Comment |
| ---- | -------- | ------- |
| `VSPHERE_PASSWORD` | `n/a` | password for vSphere API |

#### Notes

The VMWare mode requires configuring with connectivity information to the vSphere VCenter API.  See usage help for
required arguments. In order for the environment and role filters to work, the VMs must have been provisioned with extra
configuration parameters named "tags_environment" and "tags_role" set to the values provided on the command line.
