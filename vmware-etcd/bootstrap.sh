#!/bin/sh -e

set -e
if [ -n "$VMWARE_CREDENTIALS" ]; then
    if [ ! -f "$VMWARE_CREDENTIALS" ]; then
        echo "ERROR: specified credentials file not found"
        exit 1
    else
        source $VMWARE_CREDENTIALS
        if [ -z "$VMWARE_USERNAME" ]; then
            echo '$VMWARE_USERNAME is missing'
            exit 1
        fi
        if [ -z "$VMWARE_PASSWORD" ]; then
            echo '$VMWARE_USERNAME is missing'
            exit 1
        fi
        ETCD_BOOTSTRAP_FLAGS="$ETCD_BOOTSTRAP_FLAGS -vmware-username $VMWARE_USERNAME -vmware-password $VMWARE_PASSWORD"
    fi
fi

if [ -z $BOOTSTRAP_DIR ] ; then
    BOOTSTRAP_DIR=/bootstrap
fi

/etcd-bootstrap -o $BOOTSTRAP_DIR/etcd-bootstrap.conf $@

# Copy etcd.sh startup to the output directory so that it may be used to load the config before starting etcd
cp /etcd.sh $BOOTSTRAP_DIR
