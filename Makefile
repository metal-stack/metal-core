BINARY := metal-core
MAINMODULE := github.com/metal-stack/metal-core
COMMONDIR := $(or ${COMMONDIR},../builder)
CGO_ENABLED := 1

include $(COMMONDIR)/Makefile.inc

release:: generate-client tidy gofmt all;

.PHONY: spec
spec: release
	bin/metal-core spec > spec/metal-core.json

.PHONY: test-switcher
test-switcher:
	cd ./switcher && ./validate.sh && cd -

.PHONY: generate-client
generate-client:
	rm -rf models client
	GO111MODULE=off swagger generate client -f metal-api.json --skip-validation

.PHONY: redoc
redoc:
	docker run -it --rm -v $(PWD):/work -w /work letsdeal/redoc-cli bundle -o generate/index.html /work/spec/metal-core.json
	xdg-open generate/index.html
