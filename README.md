![travis](https://travis-ci.org/sky-uk/etcd-bootstrap.svg?branch=master)

# etcd-bootstrap

Bootstrap etcd nodes for cloud and vmware. `etcd-bootstrap` takes care of setting up etcd to automatically generate
configuration and optionally register the cluster with a provider. It is intended to be used as an initialisation tool
which is run before starting an etcd instance (e.g. kubernetes init-containers).

It currently supports use with etcd and one of:
  * An AWS Auto Scaling group or SRV record; or
  * A vSphere server; or
  * A GCP Managed Instance group

The provider type used is determined by the parameter passed after `etcd-bootstrap` and the options can be listed by
running `./etcd-bootstrap -h`. Once you have selected a provider to use, you can list the various flags supported by
running `./etcd-bootsrap <provider> -h`.

## AWS

When using the AWS provider, by default etcd-bootstrap will get information about the instance it is running on (must
be running on an AWS EC2 instance and be part of one AWS autoscaling group). It has two modes of operation for
discovering the instances of the cluster:

- ASG mode which uses the local auto scaling group the node is a part of.
- SRV mode which uses an SRV record to discover all the nodes in the cluster.

### Provider Flags:

| Flag | Default | Comment |
| ---- | -------- | ------- |
| `--instance-lookup-method` | `asg` | the method for looking up instances (either: asg or srv) |
| `--srv-domain-name` | `n/a` | SRV record to use when using SRV lookup |
| `--srv-service` | `etcd-bootstrap` | SRV service to use when using SRV lookup |
| `--registration-provider` | `noop` | select the registration provider to use (either: dns, lb or noop) |
| `--r53-zone-id` | `n/a` | the zone to use when using the dns registration provider |
| `--dns-hostname` | `n/a` | the dns hostname to use when using the dns registration provider |
| `--lb-target-group-name` | `n/a` | the aws loadbalancer target group name when using the lb registration provider |
| `--enable-tls` | `n/a` | enable client/server/peer TLS |
| `--tls-ca` | `n/a` | path to client/server CA |
| `--tls-cert` | `n/a` | path to server certificate |
| `--tls-key` | `n/a` | path to server key |
| `--tls-peer-ca` | `n/a` | path to peer CA |
| `--tls-peer-cert` | `n/a` | path to peer cert |
| `--tls-peer-key` | `n/a` | path to peer key |

### Instance Lookup Method

#### Auto scaling group (ASG)

When this method is used, `etcd-bootstrap` will query the local ASG for instance information. All that is required is the
instance is part of an ASG.

#### SRV records

When this method is used, `etcd-bootstrap` will lookup an SRV record to find the associated instances. To set this up,
create an SRV record and then a TXT record for each instance with its name:

*SRV*
``` text
_etcd-bootstrap._tcp.etcd.example.com. 300 IN SRV 0 0 2379 etcd-0.etcd.example.com
_etcd-bootstrap._tcp.etcd.example.com. 300 IN SRV 0 0 2379 etcd-1.etcd.example.com
_etcd-bootstrap._tcp.etcd.example.com. 300 IN SRV 0 0 2379 etcd-2.etcd.example.com
```

*TXT*

``` text
etcd-0.etcd.example.com. 300 IN TXT "name=etcd-0"
etcd-1.etcd.example.com. 300 IN TXT "name=etcd-1"
etcd-2.etcd.example.com. 300 IN TXT "name=etcd-2"
```

Then inform `etcd-bootstrap` to use the SRV record:

``` sh
etcd-bootstrap --instance-lookup-method=srv --srv-domain-name=etcd.example.com ...
```

### Registration Providers

#### dns: Route53

If running etcd bootstrap with `--registration-provider=dns` this will create a route53 record containing all etcd instance
ip addresses as A records. It will create it in the zone supplied using `--r53-zone-id=` and the domain supplied by 
`--dns-hostname` (both flags are required when using this registration type).

Optionally etcd-bootstrap can also register all the IPs in the autoscaling group with a domain name.

    ./etcd-bootstrap -o=/var/run/bootstrap.conf aws --registration-provider=dns --r53-zone-id=MYZONEID --dns-hostname=etcd

If zone `MYZONEID` has domain name `example.com`, this will update the domain name `etcd.example.com` with all
of the IPs. This lets clients use round robin DNS for connecting to the cluster.

#### lb: AWS Loadbalancer Target Group

If running etcd bootstrap with `--registration-provider=lb` this will attempt to register all etcd instances with an AWS
loadbalancer target group with the name supplied by `--lb-target-group-name` (flag is required when using this
registration type).

#### Example Kubernetes Pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: etcd
spec:
  initContainers:
  - name: etcd-bootstrap
    image: skycirrus/etcd-bootstrap:v2.0.0
    command:
    - /bootstrap.sh
    args:
    - aws
    - --registration-provider=lb
    - --lb-target-group-name=my-aws-target-group
    volumeMounts:
     # required to be able to share the etcd-bootstrap ENV variables with the etcd container
    - mountPath: /bootstrap
      name: bootstrap
  containers:
  - name: etcd
    image: quay.io/coreos/etcd:v3.1.12
    args:
    - --data-dir {{ etcd_cluster_data }}
    - --heartbeat-interval 200
    - --election-timeout 2000
    volumeMounts:
    # required to be able to source the etcd-bootstrap ENV variables
    - mountPath: /bootstrap
      name: bootstrap
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

### IAM role

Instances must have one of the following IAM policy rules based on registration type.

If use the `SRV` instance lookup method, then `autoscaling:DescribeAutoScaling*` can be removed.

#### Registration type: none 

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

#### Registration type: dns 

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

#### Registration type: lb 

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

### TLS

TLS can be enabled for client to server and peer communication. etcd treats client and peer communication separately, so
certificates must be provided for each. The flags follow what [etcd itself uses](https://github.com/etcd-io/etcd/blob/master/Documentation/op-guide/security.md), with the exception that
`--enable-tls` enables both client and peer TLS.

## GCP

### Provider Flags:

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

## VMWare

### Provider Flags:

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

### Provider Environment Variables:

| ENV | Default | Comment |
| ---- | -------- | ------- |
| `VSPHERE_PASSWORD` | `n/a` | password for vSphere API |

### Notes

The VMWare mode requires configuring with connectivity information to the vSphere VCenter API.  See usage help for
required arguments. In order for the environment and role filters to work, the VMs must have been provisioned with extra
configuration parameters named "tags_environment" and "tags_role" set to the values provided on the command line.
