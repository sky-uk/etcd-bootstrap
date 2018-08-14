#!/bin/sh -e

BOOTSTRAP_DIR=/bootstrap
mkdir -p $BOOTSTRAP_DIR

/bootstrap.sh $ETCD_BOOTSTRAP_FLAGS
/etcd.sh
