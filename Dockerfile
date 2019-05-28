FROM golang:1.12-stretch as builder

ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOOS=linux \
    GOPROXY=https://gomods.fi-ts.io

WORKDIR /build/metal-core

COPY . .
RUN go mod download \
 && CGO_ENABLED=1 make clean bin/metal-core test

FROM alpine:3.9
LABEL maintainer FI-TS Devops <devops@f-i-ts.de>

RUN apk update \
 && apk add \
    ca-certificates \
    ipmitool

COPY --from=builder /build/metal-core/bin/metal-core /

ENTRYPOINT ["/metal-core"]
