#!/usr/bin/env sh
# Writes the Windows resource file (name, version, publisher) for one build target. It does
# nothing for other systems. Called by GoReleaser before each build:
#   winres.sh <os> <arch> <major> <minor> <patch> <version>
set -eu

[ "$1" = windows ] || exit 0
arch="$2" major="$3" minor="$4" patch="$5" version="$6"

arm=""
[ "$arch" = arm64 ] && arm="-arm"

# The fixed text lives in packaging/windows/versioninfo.json; the version comes from the tag.
# shellcheck disable=SC2086
go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.4.1 $arm -64 \
    -o "cmd/justz/resource_windows_${arch}.syso" \
    -ver-major "$major" -ver-minor "$minor" -ver-patch "$patch" -ver-build 0 \
    -file-version "$major.$minor.$patch.0" -product-version "$version" \
    packaging/windows/versioninfo.json
