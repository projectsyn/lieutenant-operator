// +build generate

package generate

//go:generate go run github.com/operator-framework/operator-sdk/cmd/operator-sdk generate k8s
//go:generate go run github.com/operator-framework/operator-sdk/cmd/operator-sdk generate crds
//go:generate ./create-api-docs.sh
