#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

KUBE_ROOT=$(dirname "${BASH_SOURCE}")/..

cd "${KUBE_ROOT}"

GOLINT=${GOLINT:-"golint"}
PACKAGES=($(go list ./... | grep -v /vendor/))
bad_files=()
for package in "${PACKAGES[@]}"; do
  out=$("${GOLINT}" -min_confidence=0.9 "${package}" | grep -v -E '(should not use dot imports|internal/file/bindata.go)' || :)
  if [[ -n "${out}" ]]; then
    bad_files+=("${out}")
  fi
done
if [[ "${#bad_files[@]}" -ne 0 ]]; then
  echo "!!! '$GOLINT' problems: "
  echo "${bad_files[@]}"
  exit 1
fi

# ex: ts=2 sw=2 et filetype=sh
