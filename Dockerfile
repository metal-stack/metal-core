FROM golang:latest as builder

COPY ./ /

WORKDIR /

ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOOS=linux

RUN go mod download \
 && go get github.com/magefile/mage \
 && mage build

FROM alpine:latest

COPY --from=builder /bin/metalcore /

ENTRYPOINT ["/metalcore"]
