# Set Shell to bash, otherwise some targets fail with dash/zsh etc.
SHELL := /bin/bash

# Disable built-in rules
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --no-builtin-variables
.SUFFIXES:
.SECONDARY:

PROJECT_ROOT_DIR = .
include Makefile.vars.mk

e2e_make := $(MAKE) -C e2e
go_build ?= go build -o $(BIN_FILENAME) main.go

setup-envtest ?= go run sigs.k8s.io/controller-runtime/tools/setup-envtest

# Run tests (see https://book.kubebuilder.io/reference/envtest.html?highlight=envtest#configuring-envtest-for-integration-tests)
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin

all: build ## Invokes the build target

.PHONY: test
test: ## Run tests
	go test ./... -coverprofile cover.out

# See https://storage.googleapis.com/kubebuilder-tools/ for list of supported K8s versions
integration-test: export ENVTEST_K8S_VERSION = 1.27.x
integration-test: export KUBEBUILDER_ATTACH_CONTROL_PLANE_OUTPUT = $(INTEGRATION_TEST_DEBUG_OUTPUT)
integration-test: generate $(testbin_created) ## Run integration tests with envtest
	$(setup-envtest) use '$(ENVTEST_K8S_VERSION)!'
	export KUBEBUILDER_ASSETS="$$($(setup-envtest) use -i -p path '$(ENVTEST_K8S_VERSION)!')"; \
		env | grep KUBEBUILDER; \
		go test -tags=integration -v ./... -coverprofile cover.out

.PHONY: build
build: generate fmt vet $(BIN_FILENAME) ## Build manager binary

.PHONY: run
run: fmt vet ## Run against the configured Kubernetes cluster in ~/.kube/config
	go run ./main.go -skip-vault-setup -watch-namespace lieutenant

.PHONY: install
install: generate ## Install CRDs into a cluster
	$(KUSTOMIZE) build $(CRD_ROOT_DIR) | kubectl apply $(KIND_KUBECTL_ARGS) -f -

.PHONY: uninstall
uninstall: generate ## Uninstall CRDs from a cluster
	$(KUSTOMIZE) build $(CRD_ROOT_DIR) | kubectl delete -f -

.PHONY: deploy
deploy: generate ## Deploy controller in the configured Kubernetes cluster in ~/.kube/config
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: generate
generate: ## Generate manifests e.g. CRD, RBAC etc.
	@CRD_ROOT_DIR="$(CRD_ROOT_DIR)" CRD_DOCS_REF_PATH="$(CRD_DOCS_REF_PATH)" go generate -tags=generate generate.go
	@rm -f config/*.yaml

.PHONY: crd
crd: generate ## Generate CRD to file
	$(KUSTOMIZE) build $(CRD_ROOT_DIR) > $(CRD_FILE)

.PHONY: fmt
fmt: ## Run go fmt against code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code
	go vet ./...

.PHONY: lint
lint: fmt vet ## Invokes the fmt and vet targets
	@echo 'Check for uncommitted changes ...'
	git diff --exit-code

.PHONY: docker-build
docker-build: export GOOS = linux
docker-build: $(BIN_FILENAME) ## Build the docker image
	docker build . -t $(DOCKER_IMG) -t $(QUAY_IMG) -t $(E2E_IMG)

.PHONY: docker-push
docker-push: ## Push the docker image
	docker push $(DOCKER_IMG)
	docker push $(QUAY_IMG)

clean: export KUBECONFIG = $(KIND_KUBECONFIG)
clean: e2e-clean kind-clean ## Cleans up the generated resources
	rm -r testbin/ dist/ bin/ cover.out $(BIN_FILENAME) || true

.PHONY: help
help: ## Show this help
	@grep -E -h '\s##\s' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

###
### Assets
###

$(testbin_created):
	mkdir -p $(TESTBIN_DIR)
	# a marker file must be created, because the date of the
	# directory may update when content in it is created/updated,
	# which would cause a rebuild / re-initialization of dependants
	@touch $(testbin_created)

# Build the binary without running generators
.PHONY: $(BIN_FILENAME)
$(BIN_FILENAME): export CGO_ENABLED = 0
$(BIN_FILENAME):
	$(go_build)

###
### KIND
###

.PHONY: kind-setup
kind-setup: ## Creates a kind instance if one does not exist yet.
	@$(e2e_make) kind-setup


kind-env: export KUBECONFIG = $(KIND_KUBECONFIG)
kind-env: kind-setup
kind-env: install
kind-env:
	kubectl create namespace lieutenant

.PHONY: kind-clean
kind-clean: ## Removes the kind instance if it exists.
	@$(e2e_make) kind-clean

.PHONY: kind-run
kind-run: export KUBECONFIG = $(KIND_KUBECONFIG)
kind-run: kind-setup install run ## Runs the operator on the local host but configured for the kind cluster

kind-e2e-image: docker-build
	$(e2e_make) kind-e2e-image

###
### E2E Test
###

.PHONY: e2e-test
e2e-test: export KUBECONFIG = $(KIND_KUBECONFIG)
e2e-test: export BATS_FILES := $(BATS_FILES)
e2e-test: e2e-setup docker-build install ## Run the e2e tests
	@$(e2e_make) test

.PHONY: e2e-setup
e2e-setup: export KUBECONFIG = $(KIND_KUBECONFIG)
e2e-setup: ## Run the e2e setup
	@$(e2e_make) setup

.PHONY: e2e-clean-setup
e2e-clean-setup: export KUBECONFIG = $(KIND_KUBECONFIG)
e2e-clean-setup: ## Clean the e2e setup (e.g. to rerun the e2e-setup)
	@$(e2e_make) clean-setup

.PHONY: e2e-clean
e2e-clean: ## Remove all e2e-related resources (incl. all e2e Docker images)
	@$(e2e_make) clean

###
### Documentation
###

include ./docs/Makefile
