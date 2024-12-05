PROVIDER_NAME ?= scale
PROVIDER_VERSION ?= 0.1.0
ORGANIZATION ?= xlab
ARCHITECTURE ?= linux_amd64


default: help

help: ## prints help for targets with comments
	@cat $(MAKEFILE_LIST) | grep -E '^[a-zA-Z_-]+:.*?## .*$$' | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

all: fmt lint install generate  ## format, lint and install (build) the binary and generate docs 

build:  ## build provider
	go build -o ./bin -v ./...

install: build  ## build and install (into $GOPATH)
	go install -v ./...

local_provider:  ## install provider locally for testing
	@mkdir -p ~/.terraform.d/plugins/local/$(ORGANIZATION)/$(PROVIDER_NAME)/$(PROVIDER_VERSION)/$(ARCHITECTURE)
	@if [ -f "$(GOPATH)bin/terraform-provider-$(PROVIDER_NAME)" ]; then \
		cp $(GOPATH)bin/terraform-provider-$(PROVIDER_NAME) ~/.terraform.d/plugins/local/$(ORGANIZATION)/$(PROVIDER_NAME)/$(PROVIDER_VERSION)/$(ARCHITECTURE)/; \
	else \
		echo "Error: terraform-provider-$(PROVIDER_NAME) not found in $(GOPATH)bin"; \
		exit 1; \
	fi
	@echo "Local provider ready"

lint:  ## run linter
	golangci-lint run

generate:  ## generate documentation
	cd tools; go generate ./...

fmt:  ## format the code
	gofmt -s -w -e .

test:  ## run unit tests
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:  ## acceptance tests
	TF_ACC=1 go test -v -cover -timeout 120m ./...

.PHONY: help all local_provider fmt lint test testacc build install generate
