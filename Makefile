BINARY := metal-core
MAINMODULE := git.f-i-ts.de/cloud-native/maas/metal-core
COMMONDIR := $(or ${COMMONDIR},../../common)

include $(COMMONDIR)/Makefile.inc

export GATEWAY := `docker inspect -f "{{ .NetworkSettings.Networks.metal.Gateway }}" metal-core`

.PHONY: localbuild
localbuild:
	docker build -t registry.fi-ts.io/metal/metal-core -f Dockerfile.dev .

.PHONY: spec
spec:
	curl -s http://$(GATEWAY):4242/apidocs.json > spec/metal-core.json

.PHONY: generate-client
generate-client:
	rm -rf client/*
	GO111MODULE=off swagger generate client -f internal/domain/metal-api.json --skip-validation
