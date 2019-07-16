FROM alpine:3.9

RUN apk add --no-cache curl

COPY etcd-bootstrap /
COPY scripts/* /

ENTRYPOINT ["/etcd-bootstrap"]
