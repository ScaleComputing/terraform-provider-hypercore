# Terraform Provider Scale
## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22
- [Golangci-lint](https://golangci-lint.run/welcome/install/#local-installation) v1.62.2

### Installation
#### 1 Terraform
```shell
wget -O - https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install terraform

# check installation
terraform --version
```

#### 2 Go
```shell
# donwload latest version
curl -LO https://go.dev/dl/go1.23.4.linux-amd64.tar.gz

# remove old version and install the new one
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz

# remove donwloaded tar because it's not needed anymore
rm go1.23.4.linux-amd64.tar.gz

# better to add this in your ~/.bashrc
export PATH=$PATH:/usr/local/go/bin

# refresh your shell if needed
source ~/.bashrc

# check installation
go version
```

#### 3 Golangci-lint
```shell
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2

# check installation
golangci-lint --version
```

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

Fill this in for each provider

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

To install the provider locally to test it out, run `make local_provider`.

*Note:* Acceptance tests create real resources, and often cost money to run (this is for aws providers and such).

```shell
make testacc
```

### Using the GNUmakefile
```shell
❯ make
help                           prints help for targets with comments
all                            format, lint and install (build) the binary and generate docs 
build                          build provider
install                        build and install (into $GOPATH)
local_provider                 install provider locally for testing
lint                           run linter
generate                       generate documentation
fmt                            format the code
test                           run unit tests
testacc                        acceptance tests
```

### Try out the provider with a local installation
```shell
# install the provider locally
make install local_provider

# use the main.tf scrit in ./local
cd local

# init all the providers in main.tf
terraform init

# (optional) check how the resources will be generated
terraform plan

# apply the plan
terraform apply

# you can check the results in terraform.tfstate
cat terraform.tfstate | less

# destroy all resources
terraform destroy
```
