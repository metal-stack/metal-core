FROM golang:latest as builder

WORKDIR $GOPATH/src/git.f-i-ts.de/cloud-native/maas/metalcore
COPY main.go Gopkg.toml ./

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh \
 && dep ensure

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /metalcore .

FROM alpine:latest

COPY --from=builder /metalcore /

ENTRYPOINT ["/metalcore"]
