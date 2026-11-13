#!/bin/sh
# Runs inside a clean container (see test-shim-container.sh). It uses the real
# justz binary to check setup, shim add and the collision rules on a machine
# justz has never seen.
set -u

failures=0
pass() { echo "ok   - $1"; }
fail() { echo "FAIL - $1"; failures=$((failures + 1)); }

# expect <description> <expected exit code> <command...>
expect() {
    desc="$1"; want="$2"; shift 2
    out="$("$@" 2>&1)"; got=$?
    if [ "$got" -eq "$want" ]; then pass "$desc"; else fail "$desc (exit $got, wanted $want)"; echo "$out" | sed 's/^/       /'; fi
    LAST_OUT="$out"
}
# assert <description> <command...>: passes when the command succeeds.
assert() { desc="$1"; shift; if "$@" >/dev/null 2>&1; then pass "$desc"; else fail "$desc"; fi; }
has() { case "$LAST_OUT" in *"$1"*) pass "  output mentions: $1" ;; *) fail "  output should mention: $1"; echo "$LAST_OUT" | sed 's/^/       /' ;; esac; }

export HOME=/root
orig_path="$PATH"
shim_dir="$HOME/.local/bin"

project() { # name
    mkdir -p "/proj/$1/.git" "/proj/$1/config"
    printf 'default:\n' > "/proj/$1/justfile"
    printf '{"cli":"%s","groups":{}}\n' "$1" > "/proj/$1/config/commands.json"
}
for n in zzct zzearly zzlate zzfile zzown; do project $n; done

# A fresh machine: the shim directory does not exist and is not on the PATH.
expect "setup on a fresh machine succeeds" 0 justz setup
has "created $shim_dir"
has "is not on your PATH"
assert "shim directory exists" test -d "$shim_dir"

PATH="$shim_dir:$orig_path"; export PATH

expect "shim add for a clean name" 0 justz shim add /proj/zzct
has "added shim zzct (and zzcth)"
assert "stub is an executable file" test -x "$shim_dir/zzct"
assert "the shell finds the new command" command -v zzct
expect "adding the same shim again is an update" 0 justz shim add /proj/zzct
has "updated shim zzct"
expect "shim doctor is healthy" 0 justz shim doctor

# 1. A command earlier on the PATH shadows the shim.
mkdir -p /early && printf '#!/bin/sh\n' > /early/zzearly && chmod +x /early/zzearly
PATH="/early:$shim_dir:$orig_path"
expect "a shadowed shim is refused" 1 justz shim add /proj/zzearly
has "would never run"
has "--force"
expect "--replace alone does not allow a shadowed shim" 1 justz shim add /proj/zzearly --replace
expect "--force allows it" 0 justz shim add /proj/zzearly --force
expect "doctor then reports the shadowing" 1 justz shim doctor
has "never runs"

# 2. A command later on the PATH would be hidden.
mkdir -p /late && printf '#!/bin/sh\n' > /late/zzlate && chmod +x /late/zzlate
PATH="$shim_dir:$orig_path:/late"
expect "hiding an existing command is refused" 1 justz shim add /proj/zzlate
has "would hide it"
has "--replace"
expect "--force alone does not allow hiding" 1 justz shim add /proj/zzlate --force
expect "--replace allows hiding" 0 justz shim add /proj/zzlate --replace

# 3. A file in the shim directory that justz did not write is never touched.
printf '#!/bin/sh\necho mine\n' > "$shim_dir/zzfile" && chmod +x "$shim_dir/zzfile"
expect "a file justz did not write is refused" 1 justz shim add /proj/zzfile --force --replace
has "was not written by justz"
assert "the existing file is untouched" grep -q 'echo mine' "$shim_dir/zzfile"

# 4. Two projects, one name.
mkdir -p /proj/zzown2/.git /proj/zzown2/config
printf 'default:\n' > /proj/zzown2/justfile
printf '{"cli":"zzown","groups":{}}\n' > /proj/zzown2/config/commands.json
PATH="$shim_dir:$orig_path"
expect "first project takes the name" 0 justz shim add /proj/zzown
expect "second project is refused and told who owns it" 1 justz shim add /proj/zzown2
has "already points at"
expect "--replace re-points the name" 0 justz shim add /proj/zzown2 --replace

# 5. Reserved and invalid names.
expect "a reserved name is refused" 1 justz shim add /proj/zzct --name git --force --replace
expect "an invalid name is refused" 1 justz shim add /proj/zzct --name Bad_Name

# 6. Dry runs write nothing.
before="$(find "$shim_dir" -type f | wc -l)"
expect "dry run" 0 justz shim add /proj/zzct --name zzdry --dry-run
after="$(find "$shim_dir" -type f | wc -l)"
assert "dry run left the shim directory alone" test "$before" = "$after"

# 7. Removal leaves files that are not ours.
expect "remove deletes our stubs" 0 justz shim remove zzct
assert "stub is gone" test ! -e "$shim_dir/zzct"
expect "removing an unknown shim fails" 1 justz shim remove zzct

if [ "$failures" -ne 0 ]; then
    echo "$failures check(s) failed"
    exit 1
fi
echo "all container checks passed"
