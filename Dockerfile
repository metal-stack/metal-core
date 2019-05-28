FROM registry.fi-ts.io/cloud-native/go-builder:latest as builder

FROM alpine:3.9
LABEL maintainer FI-TS Devops <devops@f-i-ts.de>

RUN apk update \
 && apk add \
    ca-certificates \
    ipmitool \
    libpcap-dev

COPY --from=builder /work/bin/metal-core /

ENTRYPOINT ["/metal-core"]
