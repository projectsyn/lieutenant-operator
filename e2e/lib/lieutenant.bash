#!/bin/bash

errcho() {
	echo >&2 "${@}"
}

if [ -z "${E2E_IMAGE}" ]; then
	errcho "The environment variable 'E2E_IMAGE' is undefined or empty."
	exit 1
fi

timestamp() {
	date +%s
}

require_args() {
	if [ "${#}" != 2 ]; then
		errcho "$0 expected 2 arguments, got ${#}."
		exit 1
	fi

	if [ "${1}" != "${2}" ]; then
		errcho "Expected ${1} arguments, got ${2}."
		exit 1
	fi
}

setup() {
	debug "-- $BATS_TEST_DESCRIPTION"
	debug "-- $(date)"
	debug ""
	debug ""
}

setup_file() {
	reset_debug
}

teardown() {
	cp -r /tmp/detik debug || true
}

kustomize() {
	go run sigs.k8s.io/kustomize/kustomize/v5 "${@}"
}

replace_in_file() {
	require_args 3 ${#}

	local file var_name var_value
	file=${1}
	var_name=${2}
	var_value=${3}

	sed -i \
		-e "s|\$${var_name}|${var_value}|" \
		"${file}"
}

prepare() {
	require_args 1 ${#}

	local definition_dir target_dir target_file
	definition_dir=${1}
	target_dir="debug/${definition_dir}"
	target_file="${target_dir}/main.yml"

	mkdir -p "${target_dir}"
	kustomize build "${definition_dir}" -o "${target_file}"

	replace_in_file "${target_file}" E2E_IMAGE "'${E2E_IMAGE}'"
	replace_in_file "${target_file}" ID "$(id -u)"
}

apply() {
	require_args 1 ${#}

	prepare "${1}"
	kubectl apply -f "debug/${1}/main.yml"
}

given_a_clean_ns() {
	kubectl delete namespace "${DETIK_CLIENT_NAMESPACE}" --ignore-not-found
	kubectl delete pv subject-pv --ignore-not-found
	clear_pv_data
	kubectl create namespace "${DETIK_CLIENT_NAMESPACE}"
	echo "✅  The namespace '${DETIK_CLIENT_NAMESPACE}' is ready."
}

given_a_subject() {
	require_args 2 ${#}

	apply definitions/subject
	echo "✅  The subject is ready"
}

given_a_running_operator() {
	apply definitions/operator

	NAMESPACE=lieutenant-system \
		wait_until deployment/lieutenant-operator available
	echo "✅  A running operator is ready"
}

wait_until() {
	require_args 2 ${#}

	local object condition ns
	object=${1}
	condition=${2}
	ns=${NAMESPACE=${DETIK_CLIENT_NAMESPACE}}

	echo "Waiting for '${object}' in namespace '${ns}' to become '${condition}' ..."
	kubectl -n "${ns}" wait --timeout 1m --for "condition=${condition}" "${object}"
}
