#!/bin/sh -e

/etcd-bootstrap -o /etcd-bootstrap.conf
cat /etcd-bootstrap.conf
set -a
source /etcd-bootstrap.conf
set +a
exec /etcd $@
