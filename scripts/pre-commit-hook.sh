#!/bin/sh
# A plain git pre-commit hook that runs justz check, for a project that does not
# use the pre-commit framework. Copy it to .git/hooks/pre-commit and make it
# executable, or call it from your own hook.
#
# A repository with no justfile in its root has nothing to check: the hook
# prints a short note on stderr and exits 0, before it looks for justz.
#
# When justz is not on PATH the hook says so on stderr and FAILS (exit 1): a
# check that cannot run must not look like a check that passed. To skip instead,
# for a machine or a CI image that has no justz, set
#
#   JUSTZ_HOOK_SKIP_MISSING=1
#
# The hook then prints a loud warning and exits 0. Otherwise it exits with the
# status of justz check.
set -eu

root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
found=""
for name in justfile Justfile JUSTFILE .justfile; do
    [ -f "$root/$name" ] && found="$name"
done
if [ -z "$found" ]; then
    echo "justz pre-commit hook: no justfile in $root, nothing to check." >&2
    exit 0
fi

if ! command -v justz >/dev/null 2>&1; then
    {
        echo "justz pre-commit hook: justz is not installed, so justz check did NOT run."
        echo "  Install it: curl -fsSL https://raw.githubusercontent.com/igorlino/justz/main/install.sh | sh"
    } >&2
    if [ "${JUSTZ_HOOK_SKIP_MISSING:-}" = "1" ]; then
        echo "  WARNING: JUSTZ_HOOK_SKIP_MISSING=1 is set, so the commit goes ahead UNCHECKED." >&2
        exit 0
    fi
    echo "  Failing the commit. Set JUSTZ_HOOK_SKIP_MISSING=1 to skip the check (with this warning) instead." >&2
    exit 1
fi

cd "$root"
exec justz check
