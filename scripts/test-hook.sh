#!/bin/sh
# Tests scripts/pre-commit-hook.sh: it fails loudly when justz is missing, skips
# loudly only when JUSTZ_HOOK_SKIP_MISSING=1, and otherwise returns the exit
# status of justz check.
#
#   sh scripts/test-hook.sh
set -eu

root="$(cd "$(dirname "$0")/.." && pwd)"
hook="$root/scripts/pre-commit-hook.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

failures=0
pass() { echo "ok   - $1"; }
fail() { echo "FAIL - $1"; failures=$((failures + 1)); }

# A project directory with a justfile, and one without.
mkdir "$tmp/proj" "$tmp/bare"
: > "$tmp/proj/justfile"
cd "$tmp/proj"

# A PATH with no justz in it (the hook only needs shell builtins to notice).
mkdir "$tmp/empty"
# A PATH with a stand-in justz that exits with $FAKE_STATUS and records its args.
mkdir "$tmp/fake"
cat > "$tmp/fake/justz" <<'SH'
#!/bin/sh
echo "$*" > "${FAKE_LOG:?}"
exit "${FAKE_STATUS:-0}"
SH
chmod +x "$tmp/fake/justz"

# 1. justz missing: fail by default, loudly.
set +e
PATH="$tmp/empty" /bin/sh "$hook" >"$tmp/out" 2>"$tmp/err"
code=$?
set -e
if [ "$code" = 1 ] && grep -q "NOT run" "$tmp/err" && grep -q "Failing the commit" "$tmp/err" && [ ! -s "$tmp/out" ]; then
    pass "missing justz fails by default and says so on stderr"
else
    fail "missing justz fails by default and says so on stderr (exit $code: $(cat "$tmp/err"))"
fi

# 2. justz missing, opt-in skip: exit 0 with a loud warning.
set +e
PATH="$tmp/empty" JUSTZ_HOOK_SKIP_MISSING=1 /bin/sh "$hook" >"$tmp/out" 2>"$tmp/err"
code=$?
set -e
if [ "$code" = 0 ] && grep -q "NOT run" "$tmp/err" && grep -q "WARNING" "$tmp/err" && grep -q "UNCHECKED" "$tmp/err"; then
    pass "JUSTZ_HOOK_SKIP_MISSING=1 skips with a loud warning"
else
    fail "JUSTZ_HOOK_SKIP_MISSING=1 skips with a loud warning (exit $code: $(cat "$tmp/err"))"
fi

# 3. Only the value 1 opts in.
set +e
PATH="$tmp/empty" JUSTZ_HOOK_SKIP_MISSING=0 /bin/sh "$hook" >/dev/null 2>&1
code=$?
set -e
if [ "$code" = 1 ]; then
    pass "JUSTZ_HOOK_SKIP_MISSING=0 does not skip"
else
    fail "JUSTZ_HOOK_SKIP_MISSING=0 does not skip (exit $code)"
fi

# 4. justz present: the hook returns its status, and runs justz check.
for status in 0 1 3; do
    set +e
    PATH="$tmp/fake:$PATH" FAKE_LOG="$tmp/log" FAKE_STATUS="$status" /bin/sh "$hook" >/dev/null 2>&1
    code=$?
    set -e
    if [ "$code" = "$status" ] && [ "$(cat "$tmp/log")" = "check" ]; then
        pass "justz check exit status $status is the hook's"
    else
        fail "justz check exit status $status is the hook's (got $code, ran: $(cat "$tmp/log" 2>/dev/null))"
    fi
done

# 5. justz present: the skip variable changes nothing.
set +e
PATH="$tmp/fake:$PATH" FAKE_LOG="$tmp/log" FAKE_STATUS=1 JUSTZ_HOOK_SKIP_MISSING=1 /bin/sh "$hook" >/dev/null 2>&1
code=$?
set -e
if [ "$code" = 1 ]; then
    pass "a failing check still fails with the skip variable set"
else
    fail "a failing check still fails with the skip variable set (exit $code)"
fi

# 6. No justfile: nothing to check, so skip with a short note, even without justz.
for name in justfile Justfile JUSTFILE .justfile; do
    d="$tmp/has-$name"
    mkdir "$d"
    : > "$d/$name"
    set +e
    (cd "$d" && PATH="$tmp/empty" /bin/sh "$hook" >/dev/null 2>&1)
    code=$?
    set -e
    if [ "$code" = 1 ]; then
        pass "a $name is found, so the missing justz fails"
    else
        fail "a $name is found, so the missing justz fails (exit $code)"
    fi
done
set +e
(cd "$tmp/bare" && PATH="$tmp/empty" /bin/sh "$hook" >"$tmp/out" 2>"$tmp/err")
code=$?
set -e
if [ "$code" = 0 ] && grep -q "nothing to check" "$tmp/err" && ! grep -q "NOT run" "$tmp/err"; then
    pass "no justfile: exit 0 with a short note, before the missing-justz failure"
else
    fail "no justfile: exit 0 with a short note (exit $code: $(cat "$tmp/err"))"
fi
set +e
(cd "$tmp/bare" && PATH="$tmp/fake:$PATH" FAKE_LOG="$tmp/log" FAKE_STATUS=1 /bin/sh "$hook" >/dev/null 2>&1)
code=$?
set -e
if [ "$code" = 0 ]; then
    pass "no justfile: justz is not run"
else
    fail "no justfile: justz is not run (exit $code)"
fi

if [ "$failures" -gt 0 ]; then
    echo "$failures hook test(s) failed"
    exit 1
fi
echo "all hook tests passed"
