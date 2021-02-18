module github.com/projectsyn/lieutenant-operator

go 1.13

require (
	github.com/ahmetb/gen-crd-api-reference-docs v0.2.0
	github.com/banzaicloud/bank-vaults/pkg/sdk v0.6.0
	github.com/go-logr/logr v0.1.0
	github.com/hashicorp/vault/api v1.0.4
	github.com/imdario/mergo v0.3.11
	github.com/operator-framework/operator-lifecycle-manager/v3/v3 v3.11.0
	github.com/operator-framework/operator-sdk v0.17.1
	github.com/ryankurte/go-structparse v1.2.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	github.com/xanzy/go-gitlab v0.38.2
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	k8s.io/utils v0.0.0-20191114200735-6ca3b61696b6
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
)
