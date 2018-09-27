FROM golang:latest as builder
ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOOS=linux
WORKDIR /
COPY go.mod go.sum /
RUN go mod download \
 && go get github.com/magefile/mage
COPY ./ /
RUN mage build:binary

FROM alpine:latest
LABEL maintainer FI-TS Devops <devops@f-i-ts.de>
COPY --from=builder /bin/metalcore /
ENTRYPOINT ["/metalcore"]
