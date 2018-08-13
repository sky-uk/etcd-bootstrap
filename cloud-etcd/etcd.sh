#!/bin/sh -e

BOOTSTRAP_DIR=${BOOTSTRAP_DIR:="/bootstrap"}

set -a
source $BOOTSTRAP_DIR/etcd-bootstrap.conf
set +a
exec /usr/local/bin/etcd $@
