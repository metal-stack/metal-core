FROM golang:1.27-alpine3.24 AS builder
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

FROM alpine:3.24

RUN apk add \
    libpcap \
    ca-certificates
COPY --from=builder /work/bin/metal-core /

ENTRYPOINT ["/metal-core"]
