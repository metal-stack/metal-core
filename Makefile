BINARY := metal-core
COMMONDIR := $(or ${COMMONDIR},../common)
MAINMODULE := git.f-i-ts.de/cloud-native/metal/metal-core
CGO_ENABLED := 1

in-docker: generate-client gofmt check all;

include $(COMMONDIR)/Makefile.inc

release:: generate-client gofmt check all;

.PHONY: all
all::

release:: all ;

.PHONY: spec
spec: generate-client gofmt all
	bin/metal-core spec | python -c "$$PYTHON_DEEP_SORT" > spec/metal-core.json

.PHONY: test-switcher
test-switcher:
	cd ./switcher && ./validate.sh && cd -

.PHONY: generate-client
generate-client:
	rm -rf models client
	GO111MODULE=off swagger generate client -f metal-api.json --skip-validation
