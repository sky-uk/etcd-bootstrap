#!/bin/sh -e
set -e
if [ -n "$VMWARE_CREDENTIALS" ]; then
    if [ ! -f "$VMWARE_CREDENTIALS" ]; then
        echo "ERROR: specified credentials file not found"
        exit 1
    else
        source $VMWARE_CREDENTIALS
        ETCD_BOOTSTRAP_FLAGS+=" -vmware-username $VMWARE_USERNAME -vmware-password $VMWARE_PASSWORD"
    fi
fi

/etcd-bootstrap -o /etcd-bootstrap.conf $ETCD_BOOTSTRAP_FLAGS
set -a
source /etcd-bootstrap.conf
set +a
exec /etcd $@
