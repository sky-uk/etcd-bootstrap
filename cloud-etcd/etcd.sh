#!/bin/sh -e

set -a
source /bootstrap/etcd-bootstrap.conf
set +a
exec /usr/local/bin/etcd $@
