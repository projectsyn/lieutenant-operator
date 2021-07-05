IMG_TAG ?= latest

CURDIR ?= $(shell pwd)
BIN_FILENAME ?= $(CURDIR)/$(PROJECT_ROOT_DIR)/lieutenant-operator
TESTBIN_DIR ?= $(CURDIR)/$(PROJECT_ROOT_DIR)/testbin/bin

CRD_FILE ?= lieutenant-crd.yaml
CRD_FILE_LEGACY ?= lieutenant-crd-legacy.yaml
CRD_ROOT_DIR ?= config/crd/

CRD_DOCS_REF_PATH ?= docs/modules/ROOT/pages/references/api-reference.adoc

KIND_VERSION ?= 0.11.1
KIND_NODE_VERSION ?= v1.20.0
KIND ?= $(TESTBIN_DIR)/kind

ENABLE_LEADER_ELECTION ?= false

KIND_KUBECONFIG ?= $(TESTBIN_DIR)/kind-kubeconfig-$(KIND_NODE_VERSION)
KIND_CLUSTER ?= lieutenant-$(KIND_NODE_VERSION)
KIND_KUBECTL_ARGS ?= --validate=true

SHASUM ?= $(shell command -v sha1sum > /dev/null && echo "sha1sum" || echo "shasum -a1")
E2E_TAG ?= e2e_$(shell $(SHASUM) $(BIN_FILENAME) | cut -b-8)
E2E_REPO ?= local.dev/lieutenant/e2e
E2E_IMG = $(E2E_REPO):$(E2E_TAG)
BATS_FILES ?= .

INTEGRATION_TEST_DEBUG_OUTPUT ?= false

KUSTOMIZE ?= go run sigs.k8s.io/kustomize/kustomize/v3

# Image URL to use all building/pushing image targets
DOCKER_IMG ?= docker.io/projectsyn/lieutenant:$(IMG_TAG)
QUAY_IMG ?= quay.io/projectsyn/lieutenant:$(IMG_TAG)

testbin_created = $(TESTBIN_DIR)/.created
