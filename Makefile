# Project parameters
BINARY_NAME ?= lieutenant-operator

VERSION ?= $(shell git describe --tags --always --dirty --match=v* || (echo "command failed $$?"; exit 1))

IMAGE_NAME ?= docker.io/projectsyn/$(BINARY_NAME):$(VERSION)

# Antora variables
docker_cmd  ?= docker
docker_opts ?= --rm --tty --user "$$(id -u)"

antora_cmd  ?= $(docker_cmd) run $(docker_opts) --volume "$${PWD}":/antora vshn/antora:1.3
antora_opts ?= --cache-dir=.cache/antora

vale_img ?= docker.io/vshn/vale:1.1

vale_cmd ?= $(docker_cmd) run $(docker_opts) --volume "$${PWD}"/docs/modules/ROOT/pages:/pages $(vale_img) \
	--minAlertLevel=error \
	--config=/pages/.vale.ini /pages

# Go parameters
GOCMD   ?= go
GOBUILD ?= $(GOCMD) build
GOCLEAN ?= $(GOCMD) clean
GOTEST  ?= $(GOCMD) test

.PHONY: all
all: test build

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

.PHONY: docs
docs: generate $(web_dir)/index.html

$(web_dir)/index.html: playbook.yml $(pages)
	$(antora_cmd) $(antora_opts) $<

.PHONY: check
check: generate
	$(vale_cmd)
