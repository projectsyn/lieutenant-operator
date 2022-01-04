module github.com/projectsyn/lieutenant-operator

go 1.16

require (
	github.com/banzaicloud/bank-vaults/pkg/sdk v0.7.0
	github.com/elastic/crd-ref-docs v0.0.7
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0
	github.com/hashicorp/vault/api v1.1.1
	github.com/imdario/mergo v0.3.12
	github.com/ryankurte/go-structparse v1.2.0
	github.com/stretchr/testify v1.7.0
	github.com/xanzy/go-gitlab v0.50.1
	go.uber.org/zap v1.20.0
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	k8s.io/utils v0.0.0-20210527160623-6fdb442a123b
	sigs.k8s.io/controller-runtime v0.9.1
	sigs.k8s.io/controller-runtime/tools/setup-envtest v0.0.0-20210623192810-985e819db7af
	sigs.k8s.io/controller-tools v0.4.1
	sigs.k8s.io/kind v0.11.1
	sigs.k8s.io/kustomize/kustomize/v3 v3.10.0
)
