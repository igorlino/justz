#!/bin/sh
# Loads the generated completion scripts into real bash and zsh in a clean
# Ubuntu container and checks what they offer, including a zsh run through a
# pseudo-terminal.
#
#   sh scripts/test-completion-container.sh
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
    -v "$root/testdata/help-reference/demo/config:/fixtures:ro" \
    -v "$root/scripts/completion-container-inner.sh:/inner.sh:ro" \
    ubuntu:24.04 \
    sh -c 'apt-get update -qq >/dev/null && apt-get install -y -qq zsh >/dev/null && sh /inner.sh'
