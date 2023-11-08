FROM golang:1.21-alpine3.18 as builder
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

FROM alpine:3.18

RUN apk add \
    libpcap \
    ca-certificates
COPY --from=builder /work/bin/metal-core /

ENTRYPOINT ["/metal-core"]
