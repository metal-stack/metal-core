FROM golang:latest

WORKDIR $GOPATH/src/git.f-i-ts.de/cloud-native/maas/metalcore
COPY main.go Gopkg.toml ./

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh \
 && dep ensure

ENTRYPOINT ["go", "run", "main.go"]
