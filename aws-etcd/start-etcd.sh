#!/bin/sh -e

if [ -e /etc/credentials/vmware_ro_credentials.txt ]; then
    source /etc/credentials/vmware_ro_credentials.txt
fi

/etcd-bootstrap -o /etcd-bootstrap.conf $(eval echo ${ETCD_BOOTSTRAP_FLAGS})
set -a
source /etcd-bootstrap.conf
set +a
exec /etcd $@
