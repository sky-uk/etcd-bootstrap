#!/bin/sh -e

/etcd-bootstrap -o /bootstrap/etcd-bootstrap.conf $@
cp /etcd.sh /bootstrap
