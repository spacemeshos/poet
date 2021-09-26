BINARY := poet
RUNNER_BINARY := runner
DOCKER_IMAGE_REPO := poet

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

dockerbuild-go:
	docker build --build-arg GCLOUD_KEY="$(GCLOUD_KEY)" \
	             -t $(DOCKER_IMAGE_REPO):$(BRANCH) .
.PHONY: dockerbuild-go


dockerpush: dockerbuild-go
	echo "$(DOCKER_PASSWORD)" | docker login -u "$(DOCKER_USERNAME)" --password-stdin
	docker tag $(DOCKER_IMAGE_REPO):$(BRANCH) spacemeshos/$(DOCKER_IMAGE_REPO):$(BRANCH)
	docker push spacemeshos/$(DOCKER_IMAGE_REPO):$(BRANCH)
.PHONY: dockerpush


buildrunner:
	cd cmd/runner ; go build -o $(RUNNER_BINARY)
.PHONY: build
