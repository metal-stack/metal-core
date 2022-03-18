BINARY := metal-core
MAINMODULE := github.com/metal-stack/metal-core
CGO_ENABLED := 1

in-docker: gofmt test all;

release:: gofmt test all;

LINKMODE := -linkmode external -extldflags '-static -s -w' \
		 -X 'github.com/metal-stack/v.Version=$(VERSION)' \
		 -X 'github.com/metal-stack/v.Revision=$(GITVERSION)' \
		 -X 'github.com/metal-stack/v.GitSHA1=$(SHA)' \
		 -X 'github.com/metal-stack/v.BuildDate=$(BUILDDATE)'

.PHONY: all
all:: bin/$(BINARY);

bin/$(BINARY): test $(GOSRC)
	$(info CGO_ENABLED="$(CGO_ENABLED)")
	go build \
		-tags netgo,client \
		-ldflags \
		"$(LINKMODE)" \
		-o bin/$(BINARY) \
		$(MAINMODULE)

.PHONY: test
test:
	CGO_ENABLED=1 go test -tags client -cover ./...


.PHONY: gofmt
gofmt:
	go fmt ./...

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
