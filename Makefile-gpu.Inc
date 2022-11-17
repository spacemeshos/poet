PROJ_DIR := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
BUILD_DIR ?= $(PROJ_DIR)build/

export CGO_LDFLAGS := -L$(BUILD_DIR)
export CGO_CFLAGS := -I$(BUILD_DIR)
export GOOS
export GOARCH
export GOARM
export BUILD_DIR

CGO_TEST_LDFLAGS = $(CGO_LDFLAGS) -Wl,-rpath,$(BUILD_DIR)

ifeq ($(OS),Windows_NT)
  HOST_OS := windows
else
  HOST_OS := $(shell uname | tr [A-Z] [a-z])
endif

ifeq ($(GOOS),)
  GOOS := $(HOST_OS)
endif

ifeq ($(GOARCH),)
  GOARCH := $(shell go env GOARCH)
endif

ifeq ($(GOOS),windows)
  platform := windows
  export PATH := $(PATH):$(BUILD_DIR)
  EXE := .exe
else
  TEMP := /tmp
  ifeq ($(GOOS),darwin)
    ifeq ($(GOARCH),arm64)
    	platform := macos-m1
    else
    	platform := macos
    endif
    CGO_LDFLAGS := $(CGO_LDFLAGS) -Wl,-rpath,@loader_path
    ULIMIT := ulimit -n 9999;
  else
    ifeq ($(GOARCH),arm64)
    	platform := linux-arm64
    else
    	platform := linux
    endif
    CGO_LDFLAGS := $(CGO_LDFLAGS) -Wl,-rpath,$$ORIGIN
  endif
endif

ifneq ($(VERBOSE),)
  $(info "OS: $(OS), HOST_OS: $(HOST_OS), GOOS: $(GOOS), GOARCH: $(GOARCH), BUILD_DIR: $(BUILD_DIR), platform: $(platform)")
endif

GPU_SETUP_REV = 0.1.26
GPU_SETUP_ZIP = libgpu-setup-$(platform)-$(GPU_SETUP_REV).zip
GPU_SETUP_URL_ZIP = https://github.com/spacemeshos/gpu-post/releases/download/v$(GPU_SETUP_REV)/$(GPU_SETUP_ZIP)
ifeq ($(platform), windows)
  GPU_SETUP_LIBS = gpu-setup.dll gpu-setup.lib libgpu-setup.a api.h
else
  ifeq ($(platform), $(filter $(platform), macos macos-m1))
    GPU_SETUP_LIBS = libgpu-setup.dylib libMoltenVK.dylib libvulkan.1.dylib MoltenVK_icd.json api.h
  else
    GPU_SETUP_LIBS = libgpu-setup.so api.h
  endif
endif

BINDIR_GPU_SETUP_LIBS = $(foreach X,$(GPU_SETUP_LIBS),$(BUILD_DIR)$(X))
$(BINDIR_GPU_SETUP_LIBS): $(PROJ_DIR)$(GPU_SETUP_ZIP)
	mkdir -p $(dir $@)
	unzip -o $(PROJ_DIR)$(GPU_SETUP_ZIP) -d $(dir $@) $(notdir $@)
	touch $@
$(PROJ_DIR)$(GPU_SETUP_ZIP):
	curl -L $(GPU_SETUP_URL_ZIP) -o $(PROJ_DIR)$(GPU_SETUP_ZIP)

get-gpu-setup: $(PROJ_DIR)$(GPU_SETUP_ZIP) $(BINDIR_GPU_SETUP_LIBS)
.PHONY: get-gpu-setup

go-env: get-gpu-setup
	go env -w CGO_CFLAGS="$(CGO_CFLAGS)"
	go env -w CGO_LDFLAGS="$(CGO_LDFLAGS)"
.PHONY: go-env

go-env-test: get-gpu-setup
	go env -w CGO_CFLAGS="$(CGO_CFLAGS)"
	go env -w CGO_LDFLAGS="$(CGO_TEST_LDFLAGS)"
.PHONY: go-env-test
