MAKEFLAGS += --warn-undefined-variables
SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := all
.DELETE_ON_ERROR:
.SUFFIXES:

DOCKER_CMD   ?= docker
DOCKER_ARGS  ?= --rm --user "$$(id -u)" -v "$${PWD}:/src" --workdir /src

# Project parameters
BINARY_NAME ?= lieutenant-operator

VERSION ?= $(shell git describe --tags --always --dirty --match=v* || (echo "command failed $$?"; exit 1))

IMAGE_NAME ?= docker.io/projectsyn/$(BINARY_NAME):$(VERSION)

ANTORA_PREVIEW_CMD ?= $(DOCKER_CMD) run --rm --publish 2020:2020 --volume "${PWD}":/antora vshn/antora-preview:2.3.3 --style=syn --antora=docs

VALE_CMD  ?= $(DOCKER_CMD) run $(DOCKER_ARGS) --volume "$${PWD}"/docs/modules:/pages vshn/vale:2.1.1
VALE_ARGS ?= --minAlertLevel=error --config=/pages/ROOT/pages/.vale.ini /pages

# Linting parameters
YAML_FILES      ?= $(shell find . -type f -name '*.yaml' -or -name '*.yml' -not -name 'syn.tools_*_crd.yaml')
YAMLLINT_ARGS   ?= --no-warnings
YAMLLINT_CONFIG ?= .yamllint.yml
YAMLLINT_IMAGE  ?= docker.io/cytopia/yamllint:latest
YAMLLINT_DOCKER ?= $(DOCKER_CMD) run $(DOCKER_ARGS) $(YAMLLINT_IMAGE)


# Go parameters
GOCMD   ?= go
GOBUILD ?= $(GOCMD) build
GOCLEAN ?= $(GOCMD) clean
GOTEST  ?= $(GOCMD) test

.PHONY: all
all: lint test build

.PHONY: generate
generate:
	go generate ./...

.PHONY: build
build: generate
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -v \
		-o $(BINARY_NAME) \
		-ldflags "-X main.Version=$(VERSION) -X 'main.BuildDate=$(shell date)'" \
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

.PHONY: lint
lint: lint_yaml docs-vale

.PHONY: lint_yaml
lint_yaml: $(YAML_FILES)
	$(YAMLLINT_DOCKER) -f parsable -c $(YAMLLINT_CONFIG) $(YAMLLINT_ARGS) -- $?

.PHONY: docs-serve
docs-serve:
	$(ANTORA_PREVIEW_CMD)

.PHONY: docs-vale
docs-vale:
	$(VALE_CMD) $(VALE_ARGS)
