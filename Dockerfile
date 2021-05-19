FROM alpine:3.12

RUN apk add --no-cache curl

COPY etcd-bootstrap /
COPY scripts/* /

ENTRYPOINT ["/etcd-bootstrap"]
