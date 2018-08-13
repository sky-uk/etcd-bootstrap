#!/bin/sh -e

if [ -z $BOOTSTRAP_DIR ] ; then
    BOOTSTRAP_DIR=/bootstrap
fi

/etcd-bootstrap -o $BOOTSTRAP_DIR/etcd-bootstrap.conf $@

# Copy etcd.sh startup to the output directory so that it may be used to load the config before starting etcd
cp /etcd.sh $BOOTSTRAP_DIR
