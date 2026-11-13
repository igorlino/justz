#!/bin/sh
# Runs the shim collision checks with the real justz binary inside a clean
# Ubuntu container, so they see a machine justz has never touched.
#
#   sh scripts/test-shim-container.sh
#
# Needs Docker and Go.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

echo "building a static Linux binary"
(cd "$root" && CGO_ENABLED=0 GOOS=linux GOARCH="$(go env GOARCH)" go build -o "$tmp/justz" ./cmd/justz)

docker run --rm \
    -v "$tmp/justz:/usr/local/bin/justz:ro" \
    -v "$root/scripts/shim-container-inner.sh:/inner.sh:ro" \
    ubuntu:24.04 \
    sh -c 'apt-get update -qq >/dev/null && apt-get install -y -qq just >/dev/null && sh /inner.sh'
