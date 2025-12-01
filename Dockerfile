FROM golang:1.25-alpine3.22 AS builder
WORKDIR /work
COPY . .
RUN apk add \
    make \
    binutils \
    coreutils \
    git \
    gcc \
    libpcap-dev \
    musl-dev \
    dbus-libs
RUN make

FROM alpine:3.22

RUN apk add \
    libpcap \
    ca-certificates \
    lldpd
COPY --from=builder /work/bin/metal-core /

ENTRYPOINT ["/metal-core"]
