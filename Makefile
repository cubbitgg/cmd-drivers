VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

MODULE := $(shell go list -m)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

## Location to install binaries to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

LDFLAGS := -ldflags "-X $(MODULE)/version.Version=$(VERSION)"

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (install from https://golangci-lint.run).
	golangci-lint run ./...

.PHONY: test
test: fmt vet ## Run tests with coverage report.
	go test ./... -coverprofile cover.out
	go tool cover -func=cover.out

.PHONY: run-scan
run-scan: fmt vet ## Run driver-scan locally (pass args via ARGS, e.g. make run-scan ARGS="-dir /mnt -fs ext4").
	go run ./cmd/scan/ $(ARGS)

##@ Build

.PHONY: build
build: build-scan build-mounter build-init ## Build all binaries.

.PHONY: build-scan
build-scan: fmt vet | $(LOCALBIN) ## Build driver-scan binary.
	go build $(LDFLAGS) -o $(LOCALBIN)/driver-scan ./cmd/scan/

.PHONY: build-mounter
build-mounter: fmt vet | $(LOCALBIN) ## Build driver-mounter binary.
	go build $(LDFLAGS) -o $(LOCALBIN)/driver-mounter ./cmd/mounter/

.PHONY: build-init
build-init: fmt vet | $(LOCALBIN) ## Build driver-init binary.
	go build $(LDFLAGS) -o $(LOCALBIN)/driver-init ./cmd/init/

.PHONY: clean
clean: ## Remove build artifacts.
	rm -rf $(LOCALBIN)
	rm -f cover.out

##@ Docker

IMG_BASE ?= cubbit/cmd-drivers

.PHONY: docker-build
docker-build: ## Build docker image (IMG_BASE and VERSION can be overridden).
	docker build -t $(IMG_BASE):$(VERSION) -t $(IMG_BASE):latest .

.PHONY: docker-push
docker-push: ## Push docker image.
	docker push $(IMG_BASE):$(VERSION)
	docker push $(IMG_BASE):latest
