#!/bin/sh
# Render the Homebrew formula for a release.
#
#   scripts/render-formula.sh <version> <checksums-file>
#
# <version> is the plain version (no leading v), and <checksums-file> is the
# justz-<version>-checksums.txt asset of the release. The formula goes to stdout.
set -eu

[ $# -eq 2 ] || { echo "usage: $0 <version> <checksums-file>" >&2; exit 2; }
version="$1"
checksums="$2"
repo="igorlino/justz"

sum() {
    archive="justz-${version}-$1-$2.tar.gz"
    s="$(grep " ${archive}\$" "$checksums" | cut -d ' ' -f 1)"
    [ -n "$s" ] || { echo "render-formula: $archive is not in $checksums" >&2; exit 1; }
    printf '%s' "$s"
}
url() { printf 'https://github.com/%s/releases/download/v%s/justz-%s-%s-%s.tar.gz' "$repo" "$version" "$version" "$1" "$2"; }

darwin_arm64="$(sum darwin arm64)"
darwin_amd64="$(sum darwin amd64)"
linux_arm64="$(sum linux arm64)"
linux_amd64="$(sum linux amd64)"

cat <<EOF
class Justz < Formula
  desc "Help, checks and shims for just"
  homepage "https://github.com/${repo}"
  version "${version}"
  license "MIT"

  on_macos do
    on_arm do
      url "$(url darwin arm64)"
      sha256 "${darwin_arm64}"
    end
    on_intel do
      url "$(url darwin amd64)"
      sha256 "${darwin_amd64}"
    end
  end

  on_linux do
    on_arm do
      url "$(url linux arm64)"
      sha256 "${linux_arm64}"
    end
    on_intel do
      url "$(url linux amd64)"
      sha256 "${linux_amd64}"
    end
  end

  depends_on "just"

  def install
    bin.install "justz"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/justz --version")
  end
end
EOF
