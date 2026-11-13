#!/bin/sh
# Install justz from a GitHub release.
#
#   curl -fsSL https://raw.githubusercontent.com/igorlino/justz/main/install.sh | sh
#
# Options (flags or environment variables):
#   --version vX.Y.Z   JUSTZ_VERSION      install this release instead of the latest
#   --dir PATH         JUSTZ_INSTALL_DIR  install here (default: ~/.local/bin)
#
# For testing only:
#   JUSTZ_BASE_URL     download from this URL instead of GitHub Releases
#
# The script never uses sudo. It downloads the archive, verifies its SHA256
# against the release's checksums file, and only then installs the binary.

set -eu

REPO="igorlino/justz"

say() { printf '%s\n' "$*"; }
fail() { printf 'install.sh: %s\n' "$*" >&2; exit 1; }

version="${JUSTZ_VERSION:-}"
install_dir="${JUSTZ_INSTALL_DIR:-$HOME/.local/bin}"

while [ $# -gt 0 ]; do
    case "$1" in
        --version) [ $# -ge 2 ] || fail "--version needs a value"; version="$2"; shift 2 ;;
        --dir) [ $# -ge 2 ] || fail "--dir needs a value"; install_dir="$2"; shift 2 ;;
        -h|--help) sed -n '2,15p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
        *) fail "unknown option: $1" ;;
    esac
done

need() { command -v "$1" >/dev/null 2>&1 || fail "$1 is required but was not found"; }
need uname
need tar
need mktemp

# Work out which archive we need.
case "$(uname -s)" in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) fail "unsupported operating system: $(uname -s). On Windows use: winget install igorlino.justz" ;;
esac
case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
esac

# Download helper: curl or wget.
if command -v curl >/dev/null 2>&1; then
    download() { curl -fsSL -o "$2" "$1"; }
    latest_tag() { curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest" | sed 's|.*/||'; }
elif command -v wget >/dev/null 2>&1; then
    download() { wget -q -O "$2" "$1"; }
    latest_tag() { wget -q --max-redirect=0 -S -O /dev/null "https://github.com/$REPO/releases/latest" 2>&1 | sed -n 's|.*[Ll]ocation: .*/||p' | tr -d '\r' | head -n 1; }
else
    fail "curl or wget is required"
fi

# SHA256 helper.
if command -v sha256sum >/dev/null 2>&1; then
    sha256() { sha256sum "$1" | cut -d ' ' -f 1; }
elif command -v shasum >/dev/null 2>&1; then
    sha256() { shasum -a 256 "$1" | cut -d ' ' -f 1; }
else
    fail "sha256sum or shasum is required to verify the download"
fi

if [ -z "$version" ]; then
    version="$(latest_tag)"
    [ -n "$version" ] || fail "could not work out the latest version; pass --version vX.Y.Z"
fi
tag="$version"
case "$tag" in v*) ;; *) tag="v$tag" ;; esac
plain="${tag#v}"

archive="justz-${plain}-${os}-${arch}.tar.gz"
checksums="justz-${plain}-checksums.txt"
base="${JUSTZ_BASE_URL:-https://github.com/$REPO/releases/download/$tag}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

say "Installing justz $tag ($os/$arch)"
download "$base/$archive" "$tmp/$archive" || fail "could not download $base/$archive"
download "$base/$checksums" "$tmp/$checksums" || fail "could not download $base/$checksums"

# Verify before touching anything on disk.
expected="$(grep " $archive\$" "$tmp/$checksums" | cut -d ' ' -f 1)"
[ -n "$expected" ] || fail "$archive is not listed in $checksums"
actual="$(sha256 "$tmp/$archive")"
if [ "$expected" != "$actual" ]; then
    fail "checksum mismatch for $archive (expected $expected, got $actual); nothing was installed"
fi

tar -xzf "$tmp/$archive" -C "$tmp" justz || fail "could not extract justz from $archive"
mkdir -p "$install_dir" || fail "could not create $install_dir"
cp "$tmp/justz" "$install_dir/justz.new" || fail "could not write to $install_dir"
chmod 0755 "$install_dir/justz.new"
mv -f "$install_dir/justz.new" "$install_dir/justz"

say "Installed $install_dir/justz"
case ":$PATH:" in
    *":$install_dir:"*) ;;
    *) say "Note: $install_dir is not on your PATH. Add it, for example:"
       say "  export PATH=\"$install_dir:\$PATH\"" ;;
esac
say "Run 'justz doctor' to check your setup."
