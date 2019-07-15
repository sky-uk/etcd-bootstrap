FROM alpine:3.9

COPY etcd-bootstrap /

ENTRYPOINT ["/etcd-bootstrap"]
