PROVIDER_NAME ?= hypercore
PROVIDER_VERSION ?= 0.1.0
ORGANIZATION ?= xlab
# ARCHITECTURE is linux_amd64, darwin_arm64
ARCHITECTURE := $(shell go version | awk '{print $$4}' | sed 's|/|_|' )

default: help

help: ## prints help for targets with comments
	@cat $(MAKEFILE_LIST) | grep -E '^[a-zA-Z_-]+:.*?## .*$$' | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

all: fmt lint install generate  ## format, lint and install (build) the binary and generate docs 

build:  ## build provider
	@echo ARCHITECTURE=$(ARCHITECTURE)
	mkdir -p bin
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

force_reinit_local:  ## removes autogenerated terraform files from ./local/ and runs 'terraform init'
	cd local; rm -rf .terraform .terraform.lock.hcl terraform.tfstate; terraform init

apply_with_logs:  ## runs TF_LOG=DEBUG terraform apply -auto-approve in ./local/
	cd local; terraform apply -auto-approve

local_refresh:  ## runs a Terraform plan with a refresh
	cd local; terraform plan -refresh-only

run: fmt build install local_provider force_reinit_local apply_with_logs local_refresh ## run all the targets for a fresh run or re-run (this also removes all terraform cache to ensure a fresh run every time)

lint:  ## run linter
	golangci-lint run

generate:  ## generate documentation
	cd tools; go generate ./...

fmt:  ## format the code
	gofmt -s -w -e .
	cd local; terraform fmt

test:  ## run unit tests
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:  ## acceptance tests
	TF_ACC=1 go test -v -cover -count=1 -timeout 120m ./...

.PHONY: help all local_provider fmt lint test testacc build install generate force_reinit_local apply_with_logs run
