BINARY := metal-core
MAINMODULE := github.com/metal-stack/metal-core
COMMONDIR := $(or ${COMMONDIR},../builder)
CGO_ENABLED := 1

include $(COMMONDIR)/Makefile.inc

release:: generate-client tidy gofmt all;

.PHONY: spec
spec: release
	@$(info spec=$$(bin/metal-core spec | jq -S 'walk(if type == "array" then sort_by(strings) else . end)' 2>/dev/null) && echo "$${spec}" > spec/metal-core.json)
	@spec=`bin/metal-core spec | jq -S 'walk(if type == "array" then sort_by(strings) else . end)' 2>/dev/null` && echo "$${spec}" > spec/metal-core.json || echo "jq >1.6 required"

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
