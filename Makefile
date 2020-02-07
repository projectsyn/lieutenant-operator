# Project parameters
BINARY_NAME ?= lieutenant-operator

VERSION ?= $(shell git describe --tags --always --dirty --match=v* || (echo "command failed $$?"; exit 1))

IMAGE_NAME ?= docker.io/projectsyn/$(BINARY_NAME):$(VERSION)

# Antora variables
# Go parameters
GOCMD   ?= go
GOBUILD ?= $(GOCMD) build
GOCLEAN ?= $(GOCMD) clean
GOTEST  ?= $(GOCMD) test
GOGET   ?= $(GOCMD) get

.PHONY: all
all: test build

.PHONY: generate
generate:
	go generate ./...

.PHONY: build
build: generate
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -v \
		-o $(BINARY_NAME) \
		-ldflags "-X github.com/projectsyn/lieutenant-operator/version.Version=$(VERSION) -X 'github.com/projectsyn/lieutenant-operator/version.BuildDate=$(shell date)'" \
		cmd/manager/main.go
	@echo built '$(VERSION)'

.PHONY: test
test: generate
	$(GOTEST) -v -cover ./...

.PHONY: run
run: generate
	go run main.go

.PHONY: clean
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

.PHONY: docker
docker:
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE_NAME) .
	@echo built image $(IMAGE_NAME)
