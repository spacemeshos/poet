ORG ?= spacemeshos
IMAGE ?= poet
BINARY := poet
PROJECT := poet

BUF_VERSION := 1.8.0
PROTOC_VERSION = 21.8
PROTOC_GEN_GO_VERSION = v1.5.2
PROTOC_GEN_GRPC_GATEWAY_VERSION = v2.12.0

# The directories to store protoc builds
# must be in sync with contents of buf.gen.yaml
PROTOC_GO_BUILD_DIR := ./release/proto/go
PROTOC_SWAGGER_BUILD_DIR := ./release/proto/swagger
PROTOC_BUILD_DIRS := $(PROTOC_GO_BUILD_DIR) $(PROTOC_SWAGGER_BUILD_DIR)

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

PROTOC_DIR := $(CACHE_BIN)/protoc/$(PROTOC_VERSION)
PROTOC := $(PROTOC_DIR)/bin/protoc
$(PROTOC):
	@mkdir -p $(dir $@)
	curl -sSL \
		https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-${UNAME_OS}-${UNAME_ARCH}.zip -o $(PROTOC_DIR)/protoc.zip
	@unzip $(PROTOC_DIR)/protoc.zip -d $(PROTOC_DIR)
	@rm $(PROTOC_DIR)/protoc.zip
	@chmod +x "$@"

protoc-plugins:
	@go install github.com/golang/protobuf/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@$(PROTOC_GEN_GRPC_GATEWAY_VERSION)
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
