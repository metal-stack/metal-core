.ONESHELL:
SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
BUILDDATE := $(shell date -Iseconds)
VERSION := $(or ${VERSION},devel)

BINARY := metal-core
MODULE := git.f-i-ts.de/cloud-native/maas/metal-core
GOSRC = main.go $(shell find internal/ -type f -name '*.go')

export GOPROXY := https://gomods.fi-ts.io
export GO111MODULE := on
export CGO_ENABLED := 0

.PHONY: all clean up restart vendor generate-client

all: bin/$(BINARY);

bin/$(BINARY): $(GOSRC)
	go build -tags netgo -ldflags \
		"-X 'main.version=$(VERSION)' \
		 -X 'main.revision=$(GITVERSION)' \
		 -X 'main.gitsha1=$(SHA)' \
		 -X 'main.builddate=$(BUILDDATE)'" \
		-o bin/$(BINARY)

clean:
	rm -rf bin/$(BINARY)

up:
	docker-compose up

vendor:
	go mod vendor

restart:
	docker build -t registry.fi-ts.io/metal/metal-core -f Dockerfile.dev .

generate-client:
	swagger generate client -f internal/domain/metal-api.json --skip-validation
