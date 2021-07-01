#!/usr/bin/env bats

load "lib/utils"
load "lib/detik"
load "lib/lieutenant"

DETIK_CLIENT_NAME="kubectl"
DETIK_CLIENT_NAMESPACE="lieutenant-system"
DEBUG_DETIK="true"

@test "Given Operator config, When applying manifests, Then expect running pod" {
	# Remove traces of operator deployments from other tests
	kubectl delete namespace "$DETIK_CLIENT_NAMESPACE" --ignore-not-found
	kubectl create namespace "$DETIK_CLIENT_NAMESPACE" || true

	apply definitions/operator

	try "at most 10 times every 2s to find 1 pod named 'lieutenant-controller-manager-*' with '.spec.containers[*].image' being '${E2E_IMAGE}'"
	try "at most 40 times every 2s to find 1 pod named 'lieutenant-controller-manager-*' with 'status' being 'running'"
}
