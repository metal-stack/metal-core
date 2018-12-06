BINARY := metal-core
MAINMODULE := git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core
COMMONDIR := $(or ${COMMONDIR},../common)

include $(COMMONDIR)/Makefile.inc

export GATEWAY := `docker inspect -f "{{ .NetworkSettings.Networks.metal.Gateway }}" metal-core`

.PHONY: all
all::
	@bin/metal-core spec spec/metal-core.json
	go mod tidy

.PHONY: localbuild
localbuild: bin/$(BINARY)
	docker build -t registry.fi-ts.io/metal/metal-core -f Dockerfile.dev .

.PHONY: spec
spec:
	bin/metal-core spec spec/metal-core.json

.PHONY: generate-client
generate-client:
	rm -rf client/*
	cp ../metal-api/spec/metal-api.json domain/metal-api.json
	GO111MODULE=off swagger generate client -f domain/metal-api.json --skip-validation
