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

.PHONY: generate-client
generate-client: SWAGGERSPEC="domain/metal-api.json"
	GO111MODULE=off swagger generate client -f domain/metal-api.json --skip-validation
