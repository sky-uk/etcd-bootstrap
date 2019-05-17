FROM alpine:3.9

ENV VERSION v2.3.8
COPY checksums /

RUN apk add --no-cache curl

RUN curl -OLv https://github.com/coreos/etcd/releases/download/$VERSION/etcd-$VERSION-linux-amd64.tar.gz \
    && sha256sum -c checksums \
    && tar xzvf etcd-$VERSION-linux-amd64.tar.gz \
    && cp etcd-$VERSION-linux-amd64/etcd /usr/local/bin \
    && cp etcd-$VERSION-linux-amd64/etcdctl /usr/local/bin \
    && rm -rf etcd-$VERSION-linux-amd64 \
    && rm -f *.tar.gz

COPY etcd-bootstrap /
COPY bootstrap.sh /
COPY etcd.sh /
COPY start-etcd.sh /

ENTRYPOINT ["/start-etcd.sh"]
