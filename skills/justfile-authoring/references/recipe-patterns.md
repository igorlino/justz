# Recipe Patterns — Best Practices for Justfile Recipes

Use this reference when designing recipes to understand common patterns that improve discoverability, maintainability, and correctness.

## Variadic Arguments & Flag Forwarding

**Pattern: Forward flags to the underlying tool**

When a recipe wraps a tool that has its own flags, forward them instead of consuming them. This preserves the tool's own namespace and lets users discover flags via `tool --help` rather than duplicating them in recipe docs.

```just
set positional-arguments

build *ARGS:
    #!/usr/bin/env bash
    set -euo pipefail
    cmd=(npm run build)
    extra=()
    for arg in "$@"; do
      case "$arg" in
        --) ;;
        --watch|-w) cmd=(npm run build:watch) ;;
        --prod) extra+=(--production) ;;
        --all) extra+=(--all) ;;
        *) extra+=("$arg") ;;
      esac
    done
    "${cmd[@]}" ${extra[@]+-- "${extra[@]}"}
```

**Why:**
- Users can discover flags from the underlying tool
- Recipe stays simple and maintainable
- Avoids duplication in help text

**Key rule:** Always set `set positional-arguments` and iterate `"$@"` (never `{{ ARGS }}`, which splits arguments on whitespace). Also ignore a leading `--` separator in option loops so callers who type `just <ns> <cmd> -- -w` out of habit do not get `Unknown argument: --`.

---

## Private Recipes & Implementation Details

**Pattern: Mark internal recipes as private**

Use `[private]` annotation or `_` prefix for recipes that are implementation details, not part of the public API.

```just
# Public recipe — documented, discoverable
[group('build')]
bundle:
    @just _check-deps
    npm run bundle

# Private recipe — hidden from help, for internal use only
[private]
_check-deps:
    #!/usr/bin/env bash
    command -v node > /dev/null || { echo "❌ node required"; exit 1; }
    command -v npm > /dev/null || { echo "❌ npm required"; exit 1; }
```

**Annotations:**
- `[private]` — Hidden from `just --list` and help (preferred, more explicit)
- `_prefix` — By convention, hidden from help (legacy, still works)

**Why:**
- Keeps help/discoverability clean
- Signals intent: "This is for internal implementation"
- Prevents accidental public API surface

---

## Environment Variable Overrides

**Pattern: Allow users to override without editing files**

When your recipe needs a value that users might want to customize, use `env_var_or_default()` to let them override via environment.

```just
# Default image, but allow override
LINTER_IMAGE := env_var_or_default('LINTER_IMAGE', 'ghcr.io/example/linter-toolkit:2.3.0')

lint-check *flags:
    docker run --rm --volume "$(pwd):/workspace" \
      "{{ LINTER_IMAGE }}" isolate check {{ flags }}
```

**Usage:**
```bash
just lint-check                                            # Uses default image
LINTER_IMAGE=myregistry.com/image:latest just lint-check  # Override for one run
```

**Why:**
- Users can adapt without forking your justfile
- Encourages one-time overrides over permanent edits
- Clear in the justfile what can be overridden

---

## Script Extraction

**Pattern: Keep recipe bodies short; extract complex logic to scripts**

If a recipe body is more than 10-15 lines, move it to a script file in `scripts/<namespace>/<recipe>.sh`.

### Public vs. Internal Scripts

**Public scripts** — Can be called directly by users, part of the tool's API:

```bash
# scripts/cidaas/find-account.sh
#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'EOF'
Usage: find-account.sh -t <cmd> (-u | -s) <identifier>

Arguments:
  identifier          the value to look up
  -u, --user-id       an email, mobile_number or username custom field
  -s, --sub           the sub / dcid (uuid)
  -t, --token-cmd     command that emits an access token

Examples:
  find-account.sh -t ./op-access-token.sh -u tony@example.com
  find-account.sh -t ./op-access-token.sh -s 38934000-a89b-4a31-b696-5fb4ec7f10ac
EOF
  exit "${1:-2}"
}

while (( $# > 0 )); do
  case "$1" in
    --) shift ;;
    -h|--help) usage 0 ;;
    # ... rest of script
  esac
done
```

**Called both ways:**
```bash
# Direct call (by users or in CI)
./scripts/cidaas/find-account.sh --user-id tony@example.com

# Via justfile (wraps the script)
just cidaas find-account-by-email tony@example.com
```

**Pros:**
- ✅ Self-documenting: `./script.sh --help` works
- ✅ Portable: runs in CI/automation without `just`
- ✅ Professional: shows the tool is maintained
- ✅ Testable: can be called directly for debugging

---

**Internal scripts** — Called only from justfile, not part of public API:

```bash
# scripts/validate/check-todos.sh
#!/usr/bin/env bash
set -euo pipefail

# Simple, focused task. Called only from justfile.
for file in src/**/*.go; do
  if grep -q "TODO:" "$file"; then
    count=$(grep -c "TODO:" "$file")
    if [[ $count -gt 5 ]]; then
      echo "⚠️ Too many TODOs in $file: $count"
      exit 1
    fi
  fi
done
```

**Called only via justfile:**
```just
validate:
    ./scripts/validate/check-todos.sh
```

**Decision: Add usage() if:**
- Script might be called directly by users
- Script is complex (>20 lines)
- Script might be reused elsewhere in the future
- Script is in a public scripts/ directory (not internal)

**Skip usage() if:**
- Script is truly internal and focused
- Script is simple (<10 lines)
- Only ever called from one justfile recipe

**Rule of thumb:** When in doubt, add it. The maintenance cost is low, and it makes the script more discoverable and professional.

**❌ Avoid:**
```just
complex-operation:
    #!/usr/bin/env bash
    set -euo pipefail
    # 30 lines of logic here...
```

**✅ Do instead:**
```just
complex-operation:
    ./scripts/operations/complex-operation.sh

# scripts/operations/complex-operation.sh
#!/usr/bin/env bash
set -euo pipefail
# 30 lines of logic here, easy to test and lint
```

**Why:**
- Scripts can be parsed, linted, and tested independently
- Recipe body stays readable
- Easier to debug (script runs in its own context)
- Can be called from other tools (CI, shell) without `just`

---

## Argument Validation

**Pattern: Fail early if required arguments are missing or invalid**

Validate arguments at the top of a recipe so the user gets immediate feedback, not a cryptic error later.

```just
deploy ENVIRONMENT:
    #!/usr/bin/env bash
    set -euo pipefail
    
    # Fail early if ENVIRONMENT is not recognized
    case "{{ ENVIRONMENT }}" in
      sandbox|demo|production-eu|production-us)
        : # valid
        ;;
      *)
        echo "❌ Unknown environment: {{ ENVIRONMENT }}"
        echo "   Valid: sandbox, demo, production-eu, production-us"
        exit 1
        ;;
    esac
    
    # Safe to use ENVIRONMENT now
    helmfile -e {{ ENVIRONMENT }} apply
```

**Why:**
- Fast feedback (fail at argument validation, not after 5 minutes of work)
- Clear error messages guide users to the right fix
- Single source of truth for valid values

---

## Named Parameters vs. Positional

**Rule: Positional parameters require careful ordering**

A positional parameter declared **before** a variadic swallows the first forwarded argument.

**❌ Wrong:**
```just
test COMPONENT *FLAGS:  # COMPONENT eats the first *FLAGS value
    ./run-tests {{ COMPONENT }} {{ FLAGS }}
```

```bash
$ just test my-component --watch
# "my-component" gets COMPONENT ✓
# "--watch" gets eaten by COMPONENT, not FLAGS ✗
```

**✅ Right — use flags with `"$@"` instead:**
```just
set positional-arguments

test *ARGS:
    #!/usr/bin/env bash
    set -euo pipefail
    COMPONENT=""
    WATCH=false
    
    for arg in "$@"; do
      case "$arg" in
        --) ;;
        --component=*) COMPONENT="${arg#*=}" ;;
        --watch|-w) WATCH=true ;;
        *) COMPONENT="$arg" ;;
      esac
    done
    
    ./run-tests --component "$COMPONENT" $([ "$WATCH" = true ] && echo "--watch" || true)
```

Or keep it simple:
```just
set positional-arguments

test *ARGS:
    ./run-tests "$@"
```

---

## Multi-Line Help with Examples

**Pattern: Use multi-line `[doc]` for complex recipes**

Document not just what a recipe does, but how to use it with examples.

```just
# Build the project.
#
# Usage:
#   just build                 # Single build
#   just build -w              # Watch mode (--watch)
#   just build -p              # Production build (--prod)
#   just build -p -w           # Prod + watch
#
# See also: just bundle (optimized for release)
[group('build')]
build *ARGS:
    ...
```

**When to use multi-line docs:**
- Recipe has non-obvious usage patterns
- Flags or arguments need examples
- Cross-reference related recipes

---

## Related Patterns

See also:
- [credentials-in-recipes.md](credentials-in-recipes.md) — Handling 1Password and other secrets safely
- [antipatterns.md](antipatterns.md) — Common mistakes to avoid
- [shared-mechanics.md](shared-mechanics.md) — `mod` layout and `positional-arguments`
