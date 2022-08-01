ORG ?= spacemeshos
IMAGE ?= poet
BINARY := poet


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

image:
	docker build -t $(ORG)/$(IMAGE):$(BRANCH) .
.PHONY: image


push:
	docker push $(ORG)/$(IMAGE):$(BRANCH)
.PHONY: push
