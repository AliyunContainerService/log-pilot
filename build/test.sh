#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

if [ -z "${PKG}" ]; then
    echo "PKG must be set"
    exit 1
fi

go test -v -race -tags "cgo" $(go list ${PKG}/... | grep -v vendor | grep -v "docs/examples")
