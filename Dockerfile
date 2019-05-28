FROM registry.fi-ts.io/cloud-native/go-builder:latest as builder

ENV CGO_ENABLED=1 \
    COMMONDIR=/common

COPY --from=builder /common /common

WORKDIR /work

COPY . .
RUN go mod download \
 && make clean bin/metal-core test

FROM alpine:3.9
LABEL maintainer FI-TS Devops <devops@f-i-ts.de>

RUN apk update \
 && apk add \
    ca-certificates \
    ipmitool \
    libpcap-dev

COPY --from=builder /work/bin/metal-core /

ENTRYPOINT ["/metal-core"]
