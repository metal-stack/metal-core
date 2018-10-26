FROM golang:1.11-stretch as builder

WORKDIR /build
ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOOS=linux \
    GOPROXY=https://gomods.fi-ts.io

# Install dependencies and Mage
COPY go.mod ./
RUN go mod download \
 && go get github.com/magefile/mage

# Copy source code
COPY . .

# Test and build metal-core
RUN mage build:model test:unit test:int build:bin

FROM alpine:3.8
LABEL maintainer FI-TS Devops <devops@f-i-ts.de>

RUN apk update \
 && apk add \
    ca-certificates \
    ipmitool

COPY --from=builder /build/bin/metal-core /

ENTRYPOINT ["/metal-core"]
