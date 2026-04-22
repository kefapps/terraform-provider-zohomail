#!/usr/bin/env bash

set -euo pipefail

VERSION="${GORELEASER_VERSION:-v2.15.4}"
TMPDIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMPDIR}"
}
trap cleanup EXIT

GOBIN="${TMPDIR}/bin"
export GOBIN

GOWORK=off GOFLAGS= go install "github.com/goreleaser/goreleaser/v2@${VERSION}"
"${GOBIN}/goreleaser" "$@"
