BINARY := metal-core
MAINMODULE := git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core
COMMONDIR := $(or ${COMMONDIR},../../common)

include $(COMMONDIR)/Makefile.inc

export GATEWAY := `docker inspect -f "{{ .NetworkSettings.Networks.metal.Gateway }}" metal-core`

.PHONY: localbuild
localbuild: bin/$(BINARY)
	docker build -t registry.fi-ts.io/metal/metal-core -f Dockerfile.dev .

.PHONY: spec
spec:
	METAL_CORE_IP=1.2.3.4 METAL_CORE_SITE_ID=dummy METAL_CORE_RACK_ID=rack1 APIJSON=True bin/metal-core  2> spec/metal-core.json

.PHONY: generate-client
generate-client:
	rm -rf client/*
	GO111MODULE=off swagger generate client -f domain/metal-api.json --skip-validation
