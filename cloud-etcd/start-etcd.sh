#!/bin/sh -e

/etcd-bootstrap -o /etcd-bootstrap.conf $ETCD_BOOTSTRAP_FLAGS
set -a
source /etcd-bootstrap.conf
set +a
exec /etcd $@
