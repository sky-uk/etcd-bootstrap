#!/bin/sh -e

BOOTSTRAP_DIR=${BOOTSTRAP_DIR:="/bootstrap"}

/etcd-bootstrap -o ${BOOTSTRAP_DIR}/etcd-bootstrap.conf $@

# Copy etcd.sh startup to the output directory so that it may be used to load the config before starting etcd
cp /etcd.sh ${BOOTSTRAP_DIR}
