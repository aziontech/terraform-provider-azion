TEST?=$$(go list ./... | grep -v 'vendor')
HOSTNAME=hashicorp.com
NAMESPACE=dev
NAME=azion
BINARY=terraform-provider-${NAME}
VERSION=0.0.1
OS_ARCH=${GOHOSTOS}_${GOHOSTARCH}
CURDIR=$(shell pwd)

GO := $(shell which go)
ifeq (, $(GO))
$(error "No go binary found in $(PATH), please install go 1.19 or higher before continue")
endif

GOBIN ?= $(shell $(GO) env | grep GOBIN)
GOHOSTOS ?= $(shell $(GO) env GOHOSTOS || echo unknown)
GOHOSTARCH ?= $(shell $(GO) env GOHOSTARCH || echo unknown)
GOSEC ?= $(GOBIN)/gosec
GORELEASER ?= $(shell which goreleaser)
GOLINT ?= $(shell which golint)
GOFMT ?= $(shell which gofmt)
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)

default: install

.PHONY: build
build:
	$(GO) build -gcflags="all=-N -l" -o ${BINARY}

.PHONY: release
release: tools
	$(GORELEASER) release --rm-dist --snapshot --skip-publish  --skip-sign

.PHONY: clean-release
clean-release:
	rm -Rf dist/*

.PHONY: install
install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

.PHONY: test
test: 
	$(GO) test -i $(TEST) || exit 1
	echo $(TEST) | xargs -t -n4 $(GO) test -v $(TESTARGS) -timeout=30s -parallel=4

.PHONY: testacc
testacc: 
	TF_ACC=1 $(GO) test $(TEST) -v $(TESTARGS) -timeout 120m

tools:
	$(GO) get github.com/kisielk/errcheck
	$(GO) get golang.org/x/lint
	$(GO) install github.com/securego/gosec/v2/cmd/gosec@latest
	$(GO) install github.com/goreleaser/goreleaser@latest

.PHONY: vet
vet:
	@$(GO) vet $(TEST) ; if [ $$? -eq 1 ]; then \
		echo "\nVet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

.PHONY: fmt
fmt:
	@$(GOFMT) -w $(GOFMT_FILES)

.PHONY: lint
lint: tools
	@$(GOLINT) ./...

.PHONY: sec
sec: tools
	@$(GOSEC) ./...

.PHONY: clean
clean:
	rm -rf ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}/${BINARY}
	rm -rf ./examples/.terraform*
	rm -f ./examples/terraform.tfstate.backup
	rm -f ./examples/terraform.tfstate
	rm -rf ./examples/resource/.terraform*
	rm -f ./examples/resource/terraform.tfstate.backup
	rm -f ./examples/resource/terraform.tfstate