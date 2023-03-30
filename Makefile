TEST?=$$(go list ./... | grep -v 'vendor')
HOSTNAME=github.com
NAMESPACE=actions
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
GOFMT_FILES?=$$(find . -name '*.go')

default: install

checks:
	@go fmt ./...
	@go vet ./...

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

.PHONY: testacc
testacc: 
	TF_ACC=1 $(GO) test $(TEST) -v $(TESTARGS) -timeout 120m

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
lint: get-lint-deps ## running GoLint
	@ $(GOBIN)/golangci-lint run ./...

.PHONY: get-lint-deps
get-lint-deps:
	@if [ ! -x $(GOBIN)/golangci-lint ]; then\
		curl -sfL \
		https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) v1.31.0 ;\
	fi

.PHONY: sec
sec: get-gosec-deps
	@$(GOSEC) ./...

.PHONY: get-gosec-deps
get-gosec-deps:
	@ cd $(GOPATH); \
		$(GO) get -u github.com/securego/gosec/cmd/gosec

generate-changelog:
	@echo "==> Generating changelog..."
	@sh -c "'$(CURDIR)/scripts/generate-changelog.sh'"

.PHONY: clean
clean:
	rm -rf ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}/${BINARY}
	rm -rf ./examples/.terraform*
	rm -rf ./examples/resource/.terraform*
	find ./examples/ -name ".terraform*" -exec rm {} \;
	find ./examples/ -name *.lock.hcl -exec rm {} \;
	find ./examples/ -name "*tfstate*" -exec rm {} \;