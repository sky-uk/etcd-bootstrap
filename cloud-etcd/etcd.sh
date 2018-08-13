#!/bin/sh -e

if [ -z $BOOTSTRAP_DIR ] ; then
    BOOTSTRAP_DIR=/bootstrap
fi

set -a
source $BOOTSTRAP_DIR/etcd-bootstrap.conf
set +a
exec /usr/local/bin/etcd $@
