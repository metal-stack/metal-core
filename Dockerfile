FROM golang:1.23-alpine3.20 AS builder
WORKDIR /work
COPY . .
RUN apk add \
    make \
    binutils \
    git \
    gcc \
    libpcap-dev \
    musl-dev \
    dbus-libs
RUN make

FROM alpine:3.20

RUN apk add \
    libpcap \
    ca-certificates
COPY --from=builder /work/bin/metal-core /

ENTRYPOINT ["/metal-core"]
