BINARY := metal-core
MAINMODULE := github.com/metal-stack/metal-core
CGO_ENABLED := 1

SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
# gnu date format iso-8601 is parsable with Go RFC3339
BUILDDATE := $(shell date --iso-8601=seconds)
VERSION := $(or ${VERSION},$(shell git describe --tags --exact-match 2> /dev/null || git symbolic-ref -q --short HEAD || git rev-parse --short HEAD))

in-docker: gofmt test all;

release:: gofmt test all;

LINKMODE := -linkmode external -extldflags '-s -w' \
		 -X 'github.com/metal-stack/v.Version=$(VERSION)' \
		 -X 'github.com/metal-stack/v.Revision=$(GITVERSION)' \
		 -X 'github.com/metal-stack/v.GitSHA1=$(SHA)' \
		 -X 'github.com/metal-stack/v.BuildDate=$(BUILDDATE)'

.PHONY: all
all:: bin/$(BINARY);

bin/$(BINARY): $(GOSRC)
	$(info CGO_ENABLED="$(CGO_ENABLED)")
	go build \
		-tags netgo,client \
		-ldflags \
		"$(LINKMODE)" \
		-o bin/$(BINARY) \
		$(MAINMODULE)

.PHONY: test
test:
	CGO_ENABLED=1 go test -tags client -cover ./...

.PHONY: lint
lint:
	golangci-lint run --build-tags client -p bugs -p unused

.PHONY: gofmt
gofmt:
	go fmt ./...

.PHONY: test-switcher
test-switcher:
	cd ./switcher/templates && ./validate.sh && cd -
