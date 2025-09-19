.PHONY: build install test clean dev

# Provider details for OpenTofu
PROVIDER_NAME=tofukit
PROVIDER_VERSION=0.1.0
# Binary name uses terraform- prefix for compatibility
PROVIDER_BINARY=terraform-provider-$(PROVIDER_NAME)_v$(PROVIDER_VERSION)

# Paths
GOPATH=$(shell go env GOPATH)
PROVIDER_PATH=registry.terraform.io/tofukit/tofukit/$(PROVIDER_VERSION)
OS_ARCH=$(shell go env GOOS)_$(shell go env GOARCH)

# OpenTofu development override path
OPENTOFU_OVERRIDE_PATH=~/.local/share/opentofu/plugins/$(PROVIDER_PATH)/$(OS_ARCH)
DEV_OVERRIDE_PATH=$(shell pwd)/../.terraform.d/plugins/$(PROVIDER_PATH)/$(OS_ARCH)

default: build

build:
	go build -o $(PROVIDER_BINARY)

install: build
	mkdir -p $(OPENTOFU_OVERRIDE_PATH)
	cp $(PROVIDER_BINARY) $(OPENTOFU_OVERRIDE_PATH)/

dev: build
	mkdir -p $(DEV_OVERRIDE_PATH)
	cp $(PROVIDER_BINARY) $(DEV_OVERRIDE_PATH)/
	@echo "Provider installed for development at $(DEV_OVERRIDE_PATH)"

test:
	go test ./... -v

clean:
	rm -f $(PROVIDER_BINARY)
	rm -rf $(DEV_OVERRIDE_PATH)

fmt:
	go fmt ./...
	tofu fmt -recursive ./examples/ 2>/dev/null || echo "OpenTofu not installed - skipping format check"

docs:
	go generate ./...
