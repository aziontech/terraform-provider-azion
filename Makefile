TEST?=$$(go list ./...)
HOSTNAME=github.com
NAMESPACE=aziontech
NAME=azion
VERSION=$(shell git describe --tags --always)
DEV_VERSION=0.1.0
OS_ARCH=${GOHOSTOS}_${GOHOSTARCH}
CURDIR=$(shell pwd)

GO := $(shell which go)
ifeq (, $(GO))
$(error "No go binary found in $(PATH), please install go 1.19 or higher before continue")
endif

GOPATH ?= $(shell $(GO) env GOPATH)
GOBIN ?= $(GOPATH)/bin
GOHOSTOS ?= $(shell $(GO) env GOHOSTOS || echo unknown)
GOHOSTARCH ?= $(shell $(GO) env GOHOSTARCH || echo unknown)
GOSEC ?= $(GOBIN)/gosec
GORELEASER ?= $(shell which goreleaser)
GOFMT ?= $(shell which gofmt)
GOFMT_FILES?=$$(find . -name '*.go')

default: install

install:
	go mod tidy
	go install .

fmt:
	go fmt ./...

vet:
	go vet ./...

generate:
	go generate ./...

.PHONY: release
release: tools
	$(GORELEASER) release --rm-dist --snapshot --skip-publish  --skip-sign

.PHONY: clean-release
clean-release:
	rm -Rf dist/*

clean-dev:
	@echo "==> Removing development version ($(DEV_VERSION))"
	@rm -rf ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${DEV_VERSION}/${OS_ARCH}/terraform-provider-azion_$(DEV_VERSION)
	@if [ -d ./terraformScripts ]; then \
  		echo "==> Removing terraform Files "; \
		rm -rf ./terraformScripts/.terraform*; \
		rm -rf ./terraformScripts/resource/.terraform*; \
		find ./terraformScripts/ -name ".terraform*" -exec rm {} \; ; \
		find ./terraformScripts/ -name *.lock.hcl -exec rm {} \; ; \
		find ./terraformScripts/ -name "*tfstate*" -exec rm {} \; ; \
	fi

install-dev:
	@echo "==> Building development version ($(DEV_VERSION))"
	go build -gcflags="all=-N -l" -o terraform-provider-azion_$(DEV_VERSION)
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${DEV_VERSION}/${OS_ARCH}
	mv terraform-provider-azion_$(DEV_VERSION) ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${DEV_VERSION}/${OS_ARCH}

.PHONY: testacc
testacc: 
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 120m -parallel 1

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
	@ $(GOBIN)/golangci-lint run ./... --config .golintci.yml

.PHONY: get-lint-deps
get-lint-deps:
	@if [ ! -x $(GOBIN)/golangci-lint ]; then\
		curl -sfL \
		https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) v1.52.2 ;\
	fi

.PHONY: sec
sec: get-gosec-deps
	@ -$(GOSEC) ./...

.PHONY: get-gosec-deps
get-gosec-deps:
	@if [ ! -x $(GOBIN)/gosec ]; then\
		curl -sfL \
		https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b $(GOBIN) v2.15.0 ;\
	fi

func-init:
	@rm -rf func-tests/.terraform.lock.hcl
	@rm -rf func-tests/.terraform
	@rm -rf func-tests/terraform.log
	@rm -rf func-tests/terraform.tfstate
	@cd func-tests && terraform init

func-plan:
	@cd func-tests && TF_LOG=TRACE TF_LOG_PATH=./terraform.log terraform plan

func-apply:
	@cd func-tests && TF_LOG=TRACE TF_LOG_PATH=./terraform.log terraform apply -auto-approve -lock=false

func-destroy:
	@cd func-tests && terraform destroy -auto-approve

debug: 
	@go build -o terraform-provider-azion
	@dlv exec terraform-provider-azion -- -debug

dev: 
	@go run main.go -debug
