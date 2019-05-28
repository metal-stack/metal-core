BINARY := metal-core
COMMONDIR := $(or ${COMMONDIR},../common)
CGO_ENABLED := 1

in-docker: generate-client fmt test all;

include $(COMMONDIR)/Makefile.inc

release:: generate-client fmt test all;

.PHONY: all
all::
	go mod tidy
	@bin/metal-core spec spec/metal-core.json

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

.PHONY: swagger-prepare
swagger-prepare:
	rm -rf client/*
	cp ../metal-api/spec/metal-api.json domain/metal-api.json

.PHONY: generate-client
generate-client: SWAGGERSPEC="domain/metal-api.json"
generate-client: swagger-prepare swaggergenerate
	GO111MODULE=off swagger generate client -f domain/metal-api.json --skip-validation
