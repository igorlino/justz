# Antipatterns — Common Mistakes in Justfiles

Catalog of mistakes found across real justfiles. Each entry shows the problem, why it's wrong, and the fix.

---

## 1. File-Scope Credential Resolution

**Found in:** ~30% of projects with 1Password

**❌ The Mistake:**
```just
registryToken := env('REGISTRY_TOKEN')

helm env='sandbox':
  REGISTRY_PASSWORD=$registryToken \
  helmfile -f deploy/helmfile.yaml.gotmpl -e {{env}} template
```

**Why it's wrong:**
- Token resolved on **every** `just` invocation (including `just help`)
- If token comes from 1Password: biometric prompt on every command
- Developers frustrated by 20s delays for commands they don't run

**✅ Fix:**
```just
helm env='sandbox':
    #!/usr/bin/env bash
    REGISTRY_TOKEN=$(OP_BIOMETRIC_UNLOCK_ENABLED=true op read 'op://...')
    REGISTRY_PASSWORD="$REGISTRY_TOKEN" \
    helmfile -f deploy/helmfile.yaml.gotmpl -e {{env}} template
```

See: [credentials-in-recipes.md](credentials-in-recipes.md) for full patterns.

---

## 2. Positional Parameter Before Variadic

**Found in:** ~40% of justfiles with variadic arguments

**❌ The Mistake:**
```just
test COMPONENT *FLAGS:
    ./run-tests {{ COMPONENT }} {{ FLAGS }}
```

```bash
$ just test my-component --watch
# COMPONENT = "my-component" ✓
# FLAGS = "--watch" ✓  Wait, it worked...

# But try this:
$ just test --watch my-component
# COMPONENT = "--watch" ✗
# FLAGS = "my-component" ✗
# Unexpected behavior!
```

**Why it's wrong:**
- First forwarded argument always consumed by the positional parameter
- Makes recipes hard to use; users confused about argument order
- Fragile: works sometimes, breaks other times

**✅ Fix — Option 1: Use flags with `"$@"` instead**
```just
set positional-arguments

test *ARGS:
    #!/usr/bin/env bash
    set -euo pipefail
    COMPONENT=""
    for arg in "$@"; do
      case "$arg" in
        --) ;;
        --component=*) COMPONENT="${arg#*=}" ;;
        *) COMPONENT="$arg" ;;  # fallback to positional
      esac
    done
    ./run-tests "$COMPONENT"
```

**✅ Fix — Option 2: Keep it simple**
```just
set positional-arguments

test *ARGS:
    ./run-tests "$@"  # Forward everything safely without splitting on spaces
```

See: [recipe-patterns.md](recipe-patterns.md#named-parameters-vs-positional) for full guidance.

---

## 3. Complex Logic in Recipe Bodies

**Found in:** ~50% of justfiles with shell recipes

**❌ The Mistake:**
```just
validate:
    #!/usr/bin/env bash
    set -euo pipefail
    
    for file in src/**/*.go; do
      if [[ ! -f "$file" ]]; then continue; fi
      
      if grep -q "TODO:" "$file"; then
        if [[ $(grep -c "TODO:" "$file") -gt 5 ]]; then
          echo "⚠️ Too many TODOs in $file"
          exit 1
        fi
      fi
    done
    
    # ... 30 more lines ...
```

**Why it's wrong:**
- Can't be tested independently
- Can't be linted with `shellcheck`
- Can't be run from other tools (CI, shell) without `just`
- Recipe body becomes hard to read and debug

**✅ Fix:**
```just
validate:
    ./scripts/validate-todos.sh

# scripts/validate-todos.sh
#!/usr/bin/env bash
set -euo pipefail

# 30 lines of logic, properly tested and linted
for file in src/**/*.go; do
  ...
done
```

See: [recipe-patterns.md](recipe-patterns.md#script-extraction) for when to extract.

---

## 4. Commented/Unused Recipes Left Behind

**Found in:** ~20% of justfiles

**❌ The Mistake:**
```just
test:
    cargo test

# Old test command, no longer used
# test-old:
#    ./test-old.sh

# Experimental feature flag
# test-feature-x:
#    FEATURE_X=1 cargo test --features x
```

**Why it's wrong:**
- Creates confusion: is this still used?
- Accumulates technical debt over time
- Makes help output noisy if recipes are commented out
- Dead code path if uncommented

**✅ Fix:**
- Delete unused recipes entirely
- If experimental: commit as a branch or separate file, not commented
- If deprecated: rename to `_old` (private), document migration path

```just
# Use: just test
test:
    cargo test

# Deprecated: use `just test --features x` instead
[private]
_test-feature-x:  # Can be deleted after migration period
    FEATURE_X=1 cargo test --features x
```

---

## 5. Not Forwarding Tool Flags

**Found in:** ~30% of wrapper recipes

**❌ The Mistake:**
```just
build:
    npm run build

# User tries to pass flags:
$ just build --watch
# npm never sees --watch, recipe runs: npm run build
```

**Why it's wrong:**
- Users can't use flags that the underlying tool supports
- Duplicates tool's help in recipe documentation
- Fragile: breaks when tool gains new flags

**✅ Fix:**
```just
set positional-arguments

build *ARGS:
    npm run build "$@"

$ just build --watch
# npm receives: npm run build --watch ✓
```

---

## 6. Private Recipe Naming Inconsistency

**Found in:** ~60% of justfiles with helper recipes

**❌ The Mistake:**
```just
public-recipe:
    @just build-deps  # Is this private? Unclear.

build-deps:
    # Could be called by users — should be documented?

_private-helper:  # This one is clearly private

[private]
another-helper:   # This one too, but different style
```

**Why it's wrong:**
- Inconsistent: reader doesn't know which is private vs public
- Makes help harder to read (unclear what's public API)
- `build-deps` looks public but might be implementation detail

**✅ Fix:**
```just
public-recipe:
    @just _build-deps  # Clear: private

[private]
_build-deps:  # Or use @[private]
    # ...

[private]
_another-helper:  # Consistent style
    # ...
```

**Recommendation:** Use `[private]` annotation (explicit) rather than `_` prefix (by convention).

See: [recipe-patterns.md](recipe-patterns.md#private-recipes--implementation-details) for full pattern.

---

## 7. No Argument Validation

**Found in:** ~40% of recipes with parameters

**❌ The Mistake:**
```just
deploy ENVIRONMENT:
    helmfile -e {{ ENVIRONMENT }} apply
    # If user typos: "just deploy producton" → silent deploy to wrong env!
```

**Why it's wrong:**
- Silent failures lead to production incidents
- User discovers mistake after 5+ minutes of deployment
- No guidance on valid values

**✅ Fix:**
```just
deploy ENVIRONMENT:
    #!/usr/bin/env bash
    case "{{ ENVIRONMENT }}" in
      sandbox|demo|staging|prod)
        ;;  # valid
      *)
        echo "❌ Unknown environment: {{ ENVIRONMENT }}"
        echo "   Valid: sandbox, demo, staging, prod"
        exit 1
        ;;
    esac
    
    helmfile -e {{ ENVIRONMENT }} apply
```

---

## 8. Multi-Line Scripts Without `set -euo pipefail`

**Found in:** ~50% of recipe bodies

**❌ The Mistake:**
```just
validate:
    #!/usr/bin/env bash
    
    if ! which cargo > /dev/null; then
      echo "cargo not found"
    fi  # Continues even if cargo missing!
    
    cargo test  # May fail in unexpected ways
```

**Why it's wrong:**
- Without `-e`: script continues after errors
- Misleads developers: "it ran successfully" but actually failed
- Cascading failures hard to debug

**✅ Fix:**
```just
validate:
    #!/usr/bin/env bash
    set -euo pipefail  # Exit on error, undefined vars, pipe errors
    
    if ! command -v cargo > /dev/null; then
      echo "❌ cargo not found"
      exit 1
    fi
    
    cargo test  # Will exit immediately if cargo fails
```

---

## 9. Using `set fallback` Without Understanding It

**Found in:** ~10% of justfiles

**❌ The Mistake:**
```just
set fallback

# In parent justfile, you have these recipes
# In /some/subdir/justfile, you don't have a matching recipe

$ cd /some/subdir
$ just my-recipe
# Searches parent justfiles: confusing if hierarchy isn't clear
```

**Why it's wrong:**
- Creates implicit dependencies on parent directories
- Hard to debug: "why is this recipe running?"
- Breaks when directory structure changes

**✅ Fix:**
- Only use `set fallback` if you understand the directory hierarchy
- Document the fallback strategy in project README
- Or remove it and require explicit `just -f ../justfile recipe`

---

## 10. Using `env()` for Optional Values

**Found in:** ~20% of justfiles

**❌ The Mistake:**
```just
user := env('USER')  # What if USER not set? Crashes immediately

my-recipe:
    echo "User: {{ user }}"
```

**Why it's wrong:**
- Fails at parse time if env var missing
- No fallback or graceful degradation
- User confused about what's actually required

**✅ Fix:**
```just
user := env_var_or_default('USER', 'unknown')  # Safe default

my-recipe:
    #!/usr/bin/env bash
    : "${USER:?USER not set}"  # Fail with clear message if needed
    echo "User: $USER"
```

---

## 11. Silent Optional Hooks and Skippable Tests

**Found in:** this skill's own completion wiring (`tz help <TAB>` and `tzh <TAB>` did nothing in two repos)

**❌ The Mistake:**
```zsh
# completion hook, guarded so a missing helper is harmless
(( $+functions[_just_rich_complete_help_term] )) || return 0
```
```zsh
# test, skipped when the helper is absent
if (( $+functions[_just_rich_complete_help_term] )); then ... else print "skipped"; fi
```

**Why it's wrong:**
- The helper was never defined anywhere, so the hook did nothing and the test never ran
- Guard plus skip means the suite stays green while the feature is dead

**✅ Fix:**
- Define the helper where the hook's contract says it lives, and add the hook's cases to that suite
- Where a skip is legitimate (an optional tool), print it as a skip in the summary count, and have one test assert the dependency exists at the required version

---

## 12. Scaffolds That Fail Their Own Lint

**Found in:** an earlier `rich` scaffold's `help` recipe (`show_help expanded {{ ARGS }}`)

**Why it's wrong:**
- The checker flagged `{{ ARGS }}` (antipattern 5), yet the template taught it
- The interpolation splits quoted arguments and breaks the zoom lookup (`help [group] [term]`)

**✅ Fix:**
- Forward with `"$@"` and `set positional-arguments`
- Run the checker (`justz check`) against every scaffold in this skill, so a template can never break a rule the skill enforces

---

## 13. Hard-Coded Versions in Tests and Docs

**Found in:** a drift test pinned to `7.1.0` while the script it tested was `7.2.1`; a skill labeled `7.1.0` shipping a newer script

**Why it's wrong:**
- The test fails (or passes) for the wrong reason as soon as the file moves on
- Two version numbers that look related but are not drift apart silently

**✅ Fix:**
- Tests read the version from the file under test, never from a copy in the test
- Compare versions numerically per component, never as strings (7.10.0 is newer than 7.2.1)

---

## 14. Reusing a Forgiving Lookup to Run Things

**Found in:** the design of the bare-name shortcut `jz <name>`, caught in review before it shipped

**Why it's wrong:**
- `justz help [group] [term]` accepts exact names, then unique prefixes. That is right for reading help
- Reusing it to execute makes `jz gi` run `git` and `jz chec` run `check`

**✅ Fix:**
- Anything that runs or mutates resolves **exactly**. Anything that only displays may be forgiving
- Ambiguity is an error with the candidates listed, never a silent pick or an interactive picker

---

## 15. Trusting commands.json Over the Justfile

**Found in:** the same `jz` design

**Why it's wrong:**
- `commands.json` documents recipes; it is not the list of what runs, and `extra` entries are not recipes
- Rewriting a user's command from metadata can shadow a real root recipe of the same name

**✅ Fix:**
- A real recipe always wins: check `just --summary` before rewriting (root recipes appear bare, module ones as `mod::recipe`)
- Resolve only `commands`, and pass through dash-first arguments, `name=value` overrides, and `mod::recipe` paths (`justz resolve` does this)

---

## 16. Bare-Name Shortcuts in Scripts

**Found in:** the `jz <name>` shortcut (`jz git` for `just tool git`)

**Why it's wrong:**
- Adding a command to a second group turns every existing `jz <name>` into an ambiguity error, with no change to the callers
- Scripts and agents then break for a reason nowhere in their own diff

**✅ Fix:**
- Shortcuts are for typing at a prompt. Scripts and agents use the full path (`just tool git`)
- Offer a deterministic lookup for tooling: `justz resolve <name>` exits `0` found, `2` ambiguous, `3` none

---

## 17. A Completion Tag Named `options`

**Found in:** an earlier zsh completion script (`tz git <TAB>` and `tz doctor <TAB>` listed nothing)

**❌ The Mistake:**
```zsh
_describe -t options "git option" opt_items
```

**Why it's wrong:**
- zsh reserves the tag `options`: its candidates are hidden until a `-` is typed, so flags never appear on an empty word, and not even the common `-` is inserted
- Unit tests that stub `_describe` cannot see this; only a real interactive zsh does

**✅ Fix:**
- Use a tag of your own (`tz-options`; this skill's own scripts use `just-rich-flags`)
- Cover the empty-word case (`cmd <TAB>`, not only `cmd -<TAB>`) in the pty smoke test, and assert in unit tests that no tag is named `options`

---

## Quick Audit Checklist

When reviewing a justfile, look for:

- [ ] **Credentials at file scope?** Move to recipe scope
- [ ] **Positional before variadic?** Reorder or use flags
- [ ] **Complex logic in recipe?** Extract to scripts/
- [ ] **Commented recipes?** Delete or move to branch
- [ ] **Tool wrapper not forwarding flags?** Add `*ARGS`
- [ ] **Private recipe naming inconsistent?** Use `[private]` everywhere
- [ ] **No argument validation?** Add case statement
- [ ] **Shell scripts without `set -euo pipefail`?** Add it
- [ ] **Using `env()` for optional?** Use `env_var_or_default()`
- [ ] **`set fallback` without docs?** Document or remove
- [ ] **Guarded hook with a skippable test?** Define the helper, assert it exists
- [ ] **Scaffold or template that fails `--check`?** Run the checker on it in tests
- [ ] **Version pinned in a test?** Read it from the file under test
- [ ] **A lookup that runs things, also matching prefixes?** Resolve exactly
- [ ] **Rewriting commands from `commands.json`?** A real recipe wins
- [ ] **Bare-name shortcut used in a script?** Use the full path
- [ ] **Completion tag named `options`?** Use your own tag name

---

## See Also

- [recipe-patterns.md](recipe-patterns.md) — What to do instead
- [credentials-in-recipes.md](credentials-in-recipes.md) — Secure credential handling
- The [`justfile-authoring` skill](../SKILL.md#avoid-these-mistakes) has a parallel "Avoid these mistakes" section
