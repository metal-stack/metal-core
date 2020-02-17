BINARY := metal-core
COMMONDIR := $(or ${COMMONDIR},../builder)
MAINMODULE := github.com/metal-stack/metal-core
CGO_ENABLED := 1

include $(COMMONDIR)/Makefile.inc

release:: generate-client tidy gofmt check all;

.PHONY: spec
spec: release
	bin/metal-core spec | python -c "$$PYTHON_DEEP_SORT" > spec/metal-core.json

.PHONY: test-switcher
test-switcher:
	cd ./switcher && ./validate.sh && cd -

.PHONY: generate-client
generate-client:
	rm -rf models client
	GO111MODULE=off swagger generate client -f metal-api.json --skip-validation
