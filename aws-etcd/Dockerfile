FROM alpine:3.4

ENV VERSION v2.3.7
COPY checksums /

RUN apk add --no-cache curl

RUN curl -OLv https://github.com/coreos/etcd/releases/download/$VERSION/etcd-$VERSION-linux-amd64.tar.gz \
    && sha256sum -c checksums \
    && tar xzvf etcd-$VERSION-linux-amd64.tar.gz \
    && cp etcd-$VERSION-linux-amd64/etcd / \
    && cp etcd-$VERSION-linux-amd64/etcdctl / \
    && rm -rf etcd-$VERSION-linux-amd64 \
    && rm -f *.tar.gz

COPY etcd-bootstrap /
COPY start-etcd.sh /

ENTRYPOINT ["/start-etcd.sh"]
