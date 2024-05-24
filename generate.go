//go:build generate
// +build generate

package main

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=${CRD_ROOT_DIR}/bases crd:crdVersions=v1

// Generate API reference documentation
//go:generate go run github.com/elastic/crd-ref-docs --source-path=api/v1alpha1 --config=docs/api-gen-config.yaml --renderer=asciidoctor --templates-dir=docs/api-templates --output-path=${CRD_DOCS_REF_PATH}
