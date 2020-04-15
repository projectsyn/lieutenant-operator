#!/bin/bash

set -euo pipefail

mkdir tempgopath

orig_path=$(pwd)

export GOPATH="$(pwd)/tempgopath"
export GO111MODULE="on"

mkdir -p "$GOPATH/src/github.com/projectsyn"

ln -s "$(pwd)" "$GOPATH/src/github.com/projectsyn/lieutenant-operator"

cd "$GOPATH/src/github.com/projectsyn/lieutenant-operator"

go run github.com/ahmetb/gen-crd-api-reference-docs -config gen-api.json -api-dir ./pkg/apis -out-file docs/modules/ROOT/partials/crds.html

# the permissions are weird for some reason after go run...
chmod -R 770 tempgopath

cd "$orig_path"

rm -rf tempgopath
