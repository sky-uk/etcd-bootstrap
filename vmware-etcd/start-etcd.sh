#!/bin/sh -e
set -e
if [ -n "$VMWARE_CREDENTIALS" ]; then
    if [ ! -f "$VMWARE_CREDENTIALS" ]; then
        echo "ERROR: specified credentials file not found"
        exit 1
    else
        source $VMWARE_CREDENTIALS
    fi
fi

/etcd-bootstrap -o /etcd-bootstrap.conf $(eval echo ${ETCD_BOOTSTRAP_FLAGS})
set -a
source /etcd-bootstrap.conf
set +a
exec /etcd $@