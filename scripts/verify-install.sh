#!/bin/sh
# Installs the published justz release in clean containers and checks that it
# runs. This tests what users actually get, not what was built.
#
#   scripts/verify-install.sh [vX.Y.Z]     (default: the latest release)
#
# Needs Docker. Exits non-zero if any image fails.
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
version="${1:-}"

failures=""

# The command run inside each container, after the image's own prerequisites.
# The single quotes are deliberate: $HOME and $PATH must expand in the container.
# shellcheck disable=SC2016
check='sh /install.sh && export PATH="$HOME/.local/bin:$PATH" && justz --version && justz doctor --json'

run_image() {
    img="$1"
    prereq="$2"
    printf '%-24s ' "$img"
    if out="$(docker run --rm \
            -e JUSTZ_VERSION="$version" \
            -v "$root/install.sh:/install.sh:ro" \
            "$img" sh -c "$prereq; $check" 2>&1)"; then
        echo "ok   $(printf '%s\n' "$out" | grep '^justz ' | head -n 1)"
    else
        echo "FAIL"
        printf '%s\n' "$out" | sed 's/^/    /'
        failures="$failures $img"
    fi
}

# justz doctor exits 1 when just is missing, so the check above needs just in
# the container; installing it is the realistic setup anyway.
run_image ubuntu:24.04       'apt-get update -qq >/dev/null && apt-get install -y -qq curl just >/dev/null'
run_image debian:stable-slim 'apt-get update -qq >/dev/null && apt-get install -y -qq curl just >/dev/null'
run_image fedora:latest      'dnf install -y -q just >/dev/null'
run_image alpine:latest      'apk add --no-cache curl just >/dev/null'

# Homebrew image: install through the tap.
printf '%-24s ' "homebrew/brew"
if out="$(docker run --rm homebrew/brew sh -c 'brew install igorlino/tap/justz >/dev/null && justz --version && justz doctor --json' 2>&1)"; then
    echo "ok   $(printf '%s\n' "$out" | grep '^justz ' | head -n 1)"
else
    echo "FAIL"
    printf '%s\n' "$out" | tail -n 20 | sed 's/^/    /'
    failures="$failures homebrew/brew"
fi

if [ -n "$failures" ]; then
    echo "install verification failed for:$failures" >&2
    exit 1
fi
echo "all images installed and ran justz"
