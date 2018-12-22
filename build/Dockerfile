FROM alpine:3.6

RUN apk add --no-cache ca-certificates

USER nobody

ADD build/_output/bin/gcp-cloud-compute-operator /usr/local/bin/gcp-cloud-compute-operator
