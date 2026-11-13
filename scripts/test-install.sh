#!/bin/sh
# Tests install.sh against a fake release served from a local directory:
# the happy path, a tampered archive, and an archive missing from the checksums.
#
#   sh scripts/test-install.sh
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

goos="$(go env GOOS)"
goarch="$(go env GOARCH)"
version="9.9.9"
archive="justz-$version-$goos-$goarch.tar.gz"
rel="$tmp/rel"
mkdir -p "$rel" "$tmp/build"

echo "building a fake release ($goos/$goarch)"
(cd "$root" && go build -ldflags "-X main.version=$version" -o "$tmp/build/justz" ./cmd/justz)
tar -czf "$rel/$archive" -C "$tmp/build" justz
if command -v sha256sum >/dev/null 2>&1; then
    (cd "$rel" && sha256sum "$archive" > "justz-$version-checksums.txt")
else
    (cd "$rel" && shasum -a 256 "$archive" > "justz-$version-checksums.txt")
fi

install() {
    JUSTZ_BASE_URL="file://$rel" JUSTZ_VERSION="v$version" JUSTZ_INSTALL_DIR="$1" sh "$root/install.sh"
}

failures=0
pass() { echo "ok   - $1"; }
fail() { echo "FAIL - $1"; failures=$((failures + 1)); }

# 1. Happy path.
if install "$tmp/bin1" >/dev/null 2>&1 && [ "$("$tmp/bin1/justz" --version)" = "justz $version" ]; then
    pass "installs a verified release"
else
    fail "installs a verified release"
fi

# 2. A tampered archive must be refused, and nothing may be installed.
cp "$rel/$archive" "$tmp/good.tar.gz"
printf 'tampered' >> "$rel/$archive"
if install "$tmp/bin2" >"$tmp/out2" 2>&1; then
    fail "refuses a tampered archive (it installed anyway)"
elif grep -q 'checksum mismatch' "$tmp/out2" && [ ! -e "$tmp/bin2/justz" ]; then
    pass "refuses a tampered archive"
else
    fail "refuses a tampered archive (wrong error or leftover file)"
fi
cp "$tmp/good.tar.gz" "$rel/$archive"

# 3. An archive missing from the checksums file must be refused.
printf '' > "$rel/justz-$version-checksums.txt"
if install "$tmp/bin3" >"$tmp/out3" 2>&1; then
    fail "refuses an archive that is not in the checksums file"
elif grep -q 'not listed' "$tmp/out3" && [ ! -e "$tmp/bin3/justz" ]; then
    pass "refuses an archive that is not in the checksums file"
else
    fail "refuses an archive that is not in the checksums file (wrong error)"
fi

if [ "$failures" -ne 0 ]; then
    echo "$failures test(s) failed"
    exit 1
fi
echo "all install tests passed"
