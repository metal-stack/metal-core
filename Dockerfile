FROM metalstack/builder:latest as builder

FROM alpine:3.10
LABEL maintainer FI-TS Devops <devops@f-i-ts.de>

RUN apk update \
 && apk add \
    ca-certificates \
    ipmitool \
    libpcap-dev

COPY --from=builder /work/bin/metal-core /

ENTRYPOINT ["/metal-core"]
