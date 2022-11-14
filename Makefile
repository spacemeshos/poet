ORG ?= spacemeshos
IMAGE ?= poet
BINARY := poet
PROJECT := poet
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

GOLANGCI_LINT_VERSION := v1.50.0
STATICCHECK_VERSION := v0.3.3
GOTESTSUM_VERSION := v1.8.2
GOSCALE_VERSION := v1.1.1

BUF_VERSION := 1.8.0
PROTOC_VERSION = 21.8
PROTOC_GEN_GO_VERSION := v1.28
PROTOC_GEN_GRPC_VERSION := v1.2
PROTOC_GEN_GRPC_GATEWAY_VERSION := v1.16.0
PROTOC_GEN_OPENAPIV2_VERSION := v2.12.0

# The directories to store protoc builds
# must be in sync with contents of buf.gen.yaml
PROTOC_GO_BUILD_DIR := ./release/proto/go
PROTOC_OPENAPI_BUILD_DIR := ./release/proto/openapiv2
PROTOC_BUILD_DIRS := $(PROTOC_GO_BUILD_DIR) $(PROTOC_OPENAPI_BUILD_DIR)

# Everything below this line is meant to be static, i.e. only adjust the above variables. ###

ifeq ($(OS),Windows_NT)
	UNAME_OS := windows
	ifeq ($(PROCESSOR_ARCHITECTURE),AMD64)
		UNAME_ARCH := x86_64
	endif
	ifeq ($(PROCESSOR_ARCHITECTURE),ARM64)
		UNAME_ARCH := aarch64
	endif
	PROTOC_BUILD := win64

	BIN_DIR := $(abspath .)/bin
	export PATH := $(BIN_DIR);$(PATH)
	TMP_PROTOC := $(TEMP)/protoc-$(RANDOM)
else
	UNAME_OS := $(shell uname -s)
	UNAME_ARCH := $(shell uname -m)
	PROTOC_BUILD := $(shell echo ${UNAME_OS}-${UNAME_ARCH} | tr '[:upper:]' '[:lower:]' | sed 's/darwin/osx/' | sed 's/aarch64/aarch_64/')

 	BIN_DIR := $(abspath .)/bin
 	export PATH := $(BIN_DIR):$(PATH)
 	TMP_PROTOC := $(shell mktemp -d)
endif

# `go install` will put binaries in $(GOBIN), avoiding
# messing up with global environment.
export GOBIN := $(BIN_DIR)
GOTESTSUM := $(GOBIN)/gotestsum

BUILD_DEPS := $(BIN_DIR)/api.h
export CGO_LDFLAGS := -L$(BIN_DIR)  -Wl,-rpath,$(BIN_DIR)
export CPATH := $(BIN_DIR)

# We depend on post submodule Makefile.Inc fetching
# libgpu-setup.so so there is no must
# to keep it in sync with Go's dependency on post package.
post/Makefile.Inc:
	git submodule update --init -- post

$(BIN_DIR)/api.h: post/Makefile.Inc
	BIN_DIR=$(abspath $(dir $@))/ make -f post/Makefile.Inc get-gpu-setup

$(BIN_DIR)/mockgen:
	go install github.com/golang/mock/mockgen@v1.6.0

install-buf:
	@mkdir -p $(BIN_DIR)
	curl -sSL "https://github.com/bufbuild/buf/releases/download/v$(BUF_VERSION)/buf-$(UNAME_OS)-$(UNAME_ARCH)" -o $(BIN_DIR)/buf
	@chmod +x $(BIN_DIR)/buf
.PHONY: install-buf

install-protoc: protoc-plugins
	@mkdir -p $(BIN_DIR)
ifeq ($(OS),Windows_NT)
	@mkdir -p $(TMP_PROTOC)
endif
	curl -sSL https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-${PROTOC_BUILD}.zip -o $(TMP_PROTOC)/protoc.zip
	@unzip $(TMP_PROTOC)/protoc.zip -d $(TMP_PROTOC)
	@cp -f $(TMP_PROTOC)/bin/protoc $(BIN_DIR)/protoc
	@chmod +x $(BIN_DIR)/protoc
.PHONY: install-protoc

# Download protoc plugins
protoc-plugins:
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GRPC_VERSION)
	@go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@$(PROTOC_GEN_GRPC_GATEWAY_VERSION)
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@$(PROTOC_GEN_OPENAPIV2_VERSION)
.PHONY: protoc-plugins

all: build
.PHONY: all

test: $(BUILD_DEPS)
	$(GOTESTSUM) -- -timeout 5m -p 1 ./...
.PHONY: test

install: install-buf install-protoc
	@go mod download
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(GOLANGCI_LINT_VERSION)
	@go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	@go install gotest.tools/gotestsum@$(GOTESTSUM_VERSION)
	@go install github.com/spacemeshos/go-scale/scalegen@$(GOSCALE_VERSION)
.PHONY: install

tidy:
	go mod tidy
.PHONY: tidy

test-tidy:
	# Working directory must be clean, or this test would be destructive
	@git diff --quiet || (echo "\033[0;31mWorking directory not clean!\033[0m" && git --no-pager diff && exit 1)
	# We expect `go mod tidy` not to change anything, the test should fail otherwise
	@make tidy
	@git diff --exit-code || (git --no-pager diff && git checkout . && exit 1)
.PHONY: test-tidy

test-fmt:
	@git diff --quiet || (echo "\033[0;31mWorking directory not clean!\033[0m" && git --no-pager diff && exit 1)
	# We expect `go fmt` not to change anything, the test should fail otherwise
	@go fmt ./...
	@git diff --exit-code || (git --no-pager diff && git checkout . && exit 1)
.PHONY: test-fmt

clear-test-cache:
	go clean -testcache
.PHONY: clear-test-cache

lint:
	go vet ./...
	golangci-lint run --config .golangci.yml
.PHONY: lint

# Auto-fixes golangci-lint issues where possible.
lint-fix:
	golangci-lint run --config .golangci.yml --fix
.PHONY: lint-fix

lint-github-action:
	go vet ./...
	golangci-lint run --config .golangci.yml --out-format=github-actions
.PHONY: lint-github-action

# Lint .proto files
lint-protos:
	buf lint
.PHONY: lint-protos

cover: $(BUILD_DEPS)
	go test -coverprofile=cover.out -timeout 0 -p 1 ./...
.PHONY: cover

staticcheck:
	staticcheck ./...
.PHONY: staticcheck

build: $(BUILD_DEPS)
	go build -o $(BINARY)
.PHONY: build

docker:
	@DOCKER_BUILDKIT=1 docker build -t $(ORG)/$(IMAGE):$(BRANCH) .
.PHONY: docker

push:
	docker push $(ORG)/$(IMAGE):$(BRANCH)
.PHONY: push

# Rebuild .proto files
generate: $(BIN_DIR)/mockgen $(BUILD_DEPS)
	go generate ./...
	buf generate
.PHONY: generate

# Verify if files built from .proto are up to date.
test-generate:
	@git diff --quiet || (echo "\033[0;31mWorking directory not clean!\033[0m" && git --no-pager diff && exit 1)
	@make generate
	@git diff --name-only --diff-filter=AM --exit-code . || { echo "\nPlease rerun 'make generate' and commit changes.\n"; exit 1; }
.PHONY: check
