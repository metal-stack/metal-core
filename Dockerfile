FROM metalstack/builder:latest as builder

FROM alpine:3.10

RUN apk update \
 && apk add \
    ca-certificates \
    ipmitool \
    libpcap-dev

COPY --from=builder /work/bin/metal-core /

ENTRYPOINT ["/metal-core"]
