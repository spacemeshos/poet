ORG ?= spacemeshos
IMAGE ?= poet
BINARY := poet
PROJECT := poet

BUF_VERSION := 1.8.0
PROTOC_VERSION = 21.8
PROTOC_GEN_GO_VERSION = v1.28
PROTOC_GEN_GRPC_VERSION = v1.2
PROTOC_GEN_GRPC_GATEWAY_VERSION = v1.16.0
PROTOC_GEN_OPENAPIV2_VERSION = v2.12.0

# `go install` will put binaries in $(GOBIN), avoiding
# messing up with global environment.
export GOBIN := $(PWD)/bin
export PATH := $(GOBIN):$(PATH)

# The directories to store protoc builds
# must be in sync with contents of buf.gen.yaml
PROTOC_GO_BUILD_DIR := ./release/proto/go
PROTOC_OPENAPI_BUILD_DIR := ./release/proto/openapiv2
PROTOC_BUILD_DIRS := $(PROTOC_GO_BUILD_DIR) $(PROTOC_OPENAPI_BUILD_DIR)

# Everything below this line is meant to be static, i.e. only adjust the above variables. ###

UNAME_OS := $(shell uname -s)
UNAME_ARCH := $(shell uname -m)
# Buf will be cached to ~/.cache/buf-example.
CACHE_BASE := $(HOME)/.cache/$(PROJECT)
# This allows switching between i.e a Docker container and your local setup without overwriting.
CACHE := $(CACHE_BASE)/$(UNAME_OS)/$(UNAME_ARCH)
# The location where buf will be installed.
CACHE_BIN := $(CACHE)/bin

# If BUF_VERSION is changed, the binary will be re-downloaded.
BUF := $(CACHE_BIN)/buf/$(BUF_VERSION)
$(BUF):
	@mkdir -p $(dir $@)
	curl -sSL \
		"https://github.com/bufbuild/buf/releases/download/v$(BUF_VERSION)/buf-$(UNAME_OS)-$(UNAME_ARCH)" -o $@
	@chmod +x "$@"

# If PROTOC_VERSION is changed, the binary will be re-downloaded.
PROTOC_DIR := $(CACHE_BIN)/protoc/$(PROTOC_VERSION)
PROTOC := $(PROTOC_DIR)/bin/protoc
export PATH := $(dir $(PROTOC)):$(PATH)
$(PROTOC):
	@mkdir -p $(dir $@)
	curl -sSL \
		https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-${UNAME_OS}-${UNAME_ARCH}.zip -o $(PROTOC_DIR)/protoc.zip
	@unzip $(PROTOC_DIR)/protoc.zip -d $(PROTOC_DIR)
	@rm $(PROTOC_DIR)/protoc.zip
	@chmod +x "$@"

# Download protoc plugins
protoc-plugins:
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GRPC_VERSION)
	@go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@$(PROTOC_GEN_GRPC_GATEWAY_VERSION)
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@$(PROTOC_GEN_OPENAPIV2_VERSION)
.PHONY: protoc-plugins

ifdef TRAVIS_BRANCH
        BRANCH := $(TRAVIS_BRANCH)
else
        BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
endif

all: build
.PHONY: all

build:
	go build -o $(BINARY)
.PHONY: build

docker:
	@DOCKER_BUILDKIT=1 docker build -t $(ORG)/$(IMAGE):$(BRANCH) .
.PHONY: docker


push:
	docker push $(ORG)/$(IMAGE):$(BRANCH)
.PHONY: push

# Lint .proto files
lint-protos: $(BUF) $(PROTOC)
	$(BUF) lint
.PHONY: lint-protos

# Rebuild .proto files
generate: $(BUF) $(PROTOC)
	$(BUF) generate
.PHONY: generate

# Verify if files built from .proto are up to date.
check: generate
	@git add -N $(PROTOC_BUILD_DIRS)
	@git diff --name-only --diff-filter=AM --exit-code $(PROTOC_BUILD_DIRS) \
	  || { echo "\nPlease rerun 'make generate' and commit changes.\n"; exit 1; }
.PHONY: check
