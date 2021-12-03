BINARY := metal-core
MAINMODULE := github.com/metal-stack/metal-core
COMMONDIR := $(or ${COMMONDIR},../builder)
CGO_ENABLED := 1

in-docker: gofmt test all;

include $(COMMONDIR)/Makefile.inc

.PHONY: all
all::
	go mod tidy

release:: gofmt test all;

.PHONY: spec-in-docker
spec-in-docker:
	docker run -it --rm -v ${PWD}:/work -w /work metalstack/builder make spec

.PHONY: spec
spec: all
	@$(info spec=$$(bin/metal-core spec | jq -S 'walk(if type == "array" then sort_by(strings) else . end)' 2>/dev/null) && echo "$${spec}" > spec/metal-core.json)
	@spec=`bin/metal-core spec | jq -S 'walk(if type == "array" then sort_by(strings) else . end)' 2>/dev/null` && echo "$${spec}" > spec/metal-core.json || { echo "jq >=1.6 required"; exit 1; }

.PHONY: test-switcher
test-switcher:
	cd ./switcher && ./validate.sh && cd -

.PHONY: redoc
redoc:
	docker run -it --rm -v $(PWD):/work -w /work letsdeal/redoc-cli bundle -o generate/index.html /work/spec/metal-core.json
	xdg-open generate/index.html
