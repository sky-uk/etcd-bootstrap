#!/bin/sh -e

BOOTSTRAP_DIR=${BOOTSTRAP_DIR:="/bootstrap"}
ETCD_BIN_PATH=/usr/local/bin/etcd

set -a
source ${BOOTSTRAP_DIR}/etcd-bootstrap.conf
set +a
exec ${ETCD_BIN_PATH} $@