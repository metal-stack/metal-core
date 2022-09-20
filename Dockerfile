FROM golang:1.19-alpine3.16 as builder
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

FROM alpine:3.16

RUN apk add \
    libpcap \
    ca-certificates
COPY --from=builder /work/bin/metal-core /

ENTRYPOINT ["/metal-core"]
