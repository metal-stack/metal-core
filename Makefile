BINARY := metal-core
COMMONDIR := $(or ${COMMONDIR},../common)
MAINMODULE := git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core
CGO_ENABLED := 1

in-docker: generate-client fmt test all;

include $(COMMONDIR)/Makefile.inc

release:: generate-client fmt test all;

.PHONY: all
all::

release:: all ;

.PHONY: localbuild
localbuild: bin/$(BINARY)
	docker build -t registry.fi-ts.io/metal/metal-core -f Dockerfile.dev .

.PHONY: spec
spec: all
	bin/metal-core spec spec/metal-core.json

.PHONY: localbuild
localbuild: bin/$(BINARY)

.PHONY: test-switcher
test-switcher:
	cd ./switcher && ./validate.sh && cd -

.PHONY: fmt
fmt:
	GO111MODULE=off go fmt ./...

.PHONY: generate-client
generate-client:
	GO111MODULE=off swagger generate client -f domain/metal-api.json --skip-validation
