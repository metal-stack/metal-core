FROM golang:1.24-alpine3.21 AS builder
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

FROM alpine:3.21

RUN apk add \
    libpcap \
    ca-certificates
COPY --from=builder /work/bin/metal-core /

ENTRYPOINT ["/metal-core"]
