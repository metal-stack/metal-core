BINARY := metal-core
MAINMODULE := git.f-i-ts.de/cloud-native/metal/metal-core/cmd/metal-core
COMMONDIR := $(or ${COMMONDIR},../common)

include $(COMMONDIR)/Makefile.inc

export GATEWAY := `docker inspect -f "{{ .NetworkSettings.Networks.metal.Gateway }}" metal-core`

.PHONY: all
all::
	@bin/metal-core spec spec/metal-core.json
	go mod tidy

release:: all ;

.PHONY: localbuild
localbuild: bin/$(BINARY)
	docker build -t registry.fi-ts.io/metal/metal-core -f Dockerfile.dev .

.PHONY: spec
spec:
	bin/metal-core spec spec/metal-core.json

.PHONY: test-switcher
test-switcher:
	cd ./switcher && ./validate.sh && cd -

.PHONY: swagger-prepare
swagger-prepare:
	rm -rf client/*
	cp ../metal-api/spec/metal-api.json domain/metal-api.json

# 'swaggergenerate' generates swagger client with SWAGGERSPEC="swagger.json" SWAGGERTARET="./".
.PHONY: swagger
generate-client: SWAGGERSPEC="domain/metal-api.json"
generate-client: swagger-prepare swaggergenerate
