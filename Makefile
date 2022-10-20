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

all: install build
.PHONY: all

test:
	gotestsum -- -timeout 5m -p 1 ./...
.PHONY: test

install:
	go mod download
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.50.0
	go install github.com/spacemeshos/go-scale/scalegen@v1.0.0
	go install gotest.tools/gotestsum@v1.8.2
	go install honnef.co/go/tools/cmd/staticcheck@latest
.PHONY: install

tidy:
	go mod tidy
.PHONY: tidy

test-tidy:
	# Working directory must be clean, or this test would be destructive
	git diff --quiet || (echo "\033[0;31mWorking directory not clean!\033[0m" && git --no-pager diff && exit 1)
	# We expect `go mod tidy` not to change anything, the test should fail otherwise
	make tidy
	git diff --exit-code || (git --no-pager diff && git checkout . && exit 1)
.PHONY: test-tidy

test-fmt:
	if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then echo "Code needs reformatting. Please run 'make format'" && exit 1; fi
.PHONY: test-fmt

clear-test-cache:
	go clean -testcache
.PHONY: clear-test-cache

lint:
	go vet ./...
	./bin/golangci-lint run --config .golangci.yml
.PHONY: lint

# Auto-fixes golangci-lint issues where possible.
lint-fix:
	./bin/golangci-lint run --config .golangci.yml --fix
.PHONY: lint-fix

lint-github-action:
	go vet ./...
	./bin/golangci-lint run --config .golangci.yml --out-format=github-actions
.PHONY: lint-github-action

cover:
	go test -coverprofile=cover.out -timeout 0 -p 1 ./...
.PHONY: cover

staticcheck:
	staticcheck ./...
.PHONY: staticcheck

build:
	go build -o $(BINARY)
.PHONY: build

docker:
	@DOCKER_BUILDKIT=1 docker build -t $(ORG)/$(IMAGE):$(BRANCH) .
.PHONY: docker

push:
	docker push $(ORG)/$(IMAGE):$(BRANCH)
.PHONY: push

# Rebuild .proto files
generate: $(BUF) $(PROTOC)
	go generate ./...
	$(BUF) generate
.PHONY: generate

# Lint .proto files
lint-protos: $(BUF) $(PROTOC)
	$(BUF) lint
.PHONY: lint-protos

# Verify if files built from .proto are up to date.
check: generate
	@git add -N $(PROTOC_BUILD_DIRS)
	@git diff --name-only --diff-filter=AM --exit-code $(PROTOC_BUILD_DIRS) \
	  || { echo "\nPlease rerun 'make generate' and commit changes.\n"; exit 1; }
.PHONY: check
