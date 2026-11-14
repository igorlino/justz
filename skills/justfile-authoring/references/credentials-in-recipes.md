# Credentials in Recipes — Security & Best Practices

Use this reference when handling secrets (1Password tokens, API keys, passwords) in justfile recipes. Covers safe patterns, common anti-patterns, and integration with 1Password.

## The Critical Anti-Pattern: File-Scope Resolution

**❌ NEVER do this:**

```just
# ⚠️ ANTI-PATTERN: Token resolved when justfile is LOADED
registryToken := env('REGISTRY_TOKEN')

helm env='sandbox':
  REGISTRY_PASSWORD=$registryToken \
  helmfile -f deploy/helmfile.yaml.gotmpl -e {{env}} template
```

**Impact:**
- `registryToken` is resolved on **every** `just` invocation
- Even `just help` or `just --list` resolves it
- With 1Password: **biometric prompt on every `just` command**
- Users approve prompts for commands they never run

**Real-world cost:** Developer runs `just --list` → 20s wait for biometric approval → angry developer

---

## Pattern 1: Recipe-Scope Resolution (✅ CORRECT)

**Always resolve credentials WITHIN the recipe that needs them:**

```just
helm env='sandbox':
    #!/usr/bin/env bash
    set -euo pipefail
    
    # Resolve only when helm recipe runs, not on every `just` invocation
    REGISTRY_TOKEN=$(OP_BIOMETRIC_UNLOCK_ENABLED=true op read --no-newline \
      'op://Private/GitHub/token')
    
    REGISTRY_PASSWORD="$REGISTRY_TOKEN" \
    REGISTRY_USERNAME="$(git config user.name)" \
    helmfile -f deploy/helmfile.yaml.gotmpl -e {{env}} template
```

**Why:**
- Credential only resolved when recipe actually runs
- User approves prompt only for commands they chose to run
- Can be bypassed: `REGISTRY_TOKEN=... just helm production` (no prompt)

---

## Pattern 2: Shared Helper Script (✅ RECOMMENDED FOR TEAMS)

Centralize credential resolution in a project-specific helper script:

```bash
# scripts/local/op-secrets.sh
# Shared credential resolvers for the project

resolve_registry_token() {
  # Short-circuit: if already in env, use it (CI/automation)
  if [[ -n "${REGISTRY_TOKEN:-}" ]]; then
    echo "$REGISTRY_TOKEN"
    return 0
  fi
  
  # Otherwise, resolve from 1Password with biometric prompt
  OP_BIOMETRIC_UNLOCK_ENABLED=true op read --no-newline \
    'op://Private/GitHub/token'
}

resolve_api_token() {
  if [[ -n "${API_TOKEN:-}" ]]; then
    echo "$API_TOKEN"
    return 0
  fi
  
  OP_BIOMETRIC_UNLOCK_ENABLED=true op read --no-newline \
    'op://Private/Service/api-token'
}
```

Then in justfile:
```just
sync:
    #!/usr/bin/env bash
    set -euo pipefail
    source scripts/local/op-secrets.sh
    
    export SERVICE_ACCESS_TOKEN=$(resolve_api_token)
    ./scripts/sync.sh
```

**Why:**
- Testable: functions can be sourced and called independently
- Discoverable: `grep resolve_` shows all available secrets
- Maintainable: single source of truth for each secret path
- Overrideable: env vars skip resolution (CI-friendly)

---

## Pattern 3: Committed `.op.env` with `op run` (✅ FOR OP RUN PATTERN)

If using 1Password's `op run` command:

```bash
# .op.env (safe to commit — contains REFERENCES only)
REGISTRY_TOKEN=op://Private/GitHub/token
SERVICE_TOKEN=op://Private/Service/api-token
DB_PASSWORD=op://Private/Database/prod-password
```

Then in justfile:
```just
sync:
    OP_BIOMETRIC_UNLOCK_ENABLED=true op run --env-file=.op.env -- \
      bash -c './scripts/sync.sh'

migrate-db:
    OP_BIOMETRIC_UNLOCK_ENABLED=true op run --env-file=.op.env -- \
      bash -c 'psql -U postgres -c "..."'
```

**Why:**
- Clean separation: references in `.op.env`, recipes in justfile
- One approval covers all referenced secrets
- Safe to commit `.op.env` (no actual secrets)
- Rotation-proof: re-resolves on each invocation

---

## Safety Rules: Never Let Secrets Reach Durable Surfaces

**❌ Never echo a secret:**
```bash
echo "Token: $GITLAB_TOKEN"  # Captured in logs forever ✗
```

**✅ Verify by length only:**
```bash
printf '%s' "$GITLAB_TOKEN" | wc -c  # Non-zero = success ✓
```

**❌ Never pass in argv:**
```bash
curl --header "Authorization: $TOKEN" ...  # Visible in `ps` ✗
```

**✅ Pass via environment or stdin:**
```bash
curl --header "Authorization: $(cat)" < <(echo "$TOKEN") ...  # Hidden ✓
```

**❌ Never fold stderr into logs:**
```bash
TOKEN=$(op read 'op://...' 2>&1 | tee -a deploy.log)  # Exposes error context ✗
```

**✅ Keep stderr on TTY or discard:**
```bash
TOKEN=$(op read 'op://...' 2>/dev/null)  # Discarded safely ✓
```

**❌ Never include in `set -x` output:**
```bash
set -x  # Traces all assignments, including secrets ✗
```

**✅ Unset before traced sections:**
```bash
TOKEN=$(op read 'op://...')
set +x  # Secrets resolved before tracing starts
set -x
# ... rest of script traced
```

---

## Maintaining 1Password References

### Per-Project Best Practice

Add to `AGENTS.md` or project README:

```markdown
## Secrets & Credential Management

### 1Password Vault Structure

All project secrets live in `op://Private/<service>/`:

| Service | Secret | Used By | Rotation | Owner |
|---------|--------|---------|----------|-------|
| GitHub | token | CI, helm, deploy | 90 days | Platform team |
| API Service | api-token | `just sync` | 180 days | Backend team |

### Resolution Pattern

Use `scripts/local/op-secrets.sh` for credential resolution:

```bash
source scripts/local/op-secrets.sh
REGISTRY_TOKEN=$(resolve_registry_token)
```

### Adding a New Secret

1. Create secret in 1Password under `op://Private/<service>/<name>`
2. Add resolver function to `scripts/local/op-secrets.sh`
3. Update table above with service, secret, owner, rotation
4. Link this file to [`biometric-1password` skill](link) for setup

### Debugging "no active session" Errors

See [`biometric-1password` skill](link) → references → diagnostics.md
```

### Centralized Documentation

Maintain in version control, not in wiki or Slack:
- Audit trail: `git log -p` shows when secrets moved
- Discoverability: `grep resolve_` finds all secrets
- Rotation tracking: Commit message documents new tokens
- Onboarding: New team members read the same source

---

## Anti-Patterns Summary

| Pattern | Problem | Fix |
|---------|---------|-----|
| **File-scope resolution** | Prompt on every `just` | Move to recipe-scope |
| **No env var override** | Can't bypass prompt in CI | Check `${VAR:-}` first, resolve if empty |
| **Scattered references** | Can't audit or rotate | Centralize in helper script + AGENTS.md |
| **No validation** | Silent failures on rotation | Check token length: `[[ ${#TOKEN} -gt 0 ]]` |
| **Unencrypted in logs** | Leaked credentials | Never echo, verify by length only |
| **Token in process list** | Visible to other users | Never pass in argv, use env var |

---

## Integration with biometric-1password Skill

For comprehensive guidance on 1Password credential resolution patterns, see the [`biometric-1password` skill](../../biometric-1password/SKILL.md):
- [`authoring.md`](../../biometric-1password/references/authoring.md) — Wiring credentials into scripts
- [`npm-registry-auth.md`](../../biometric-1password/references/npm-registry-auth.md) — npm/package manager credentials
- [`diagnostics.md`](../../biometric-1password/references/diagnostics.md) — Troubleshooting "no active session"

---

## Related Patterns

See also:
- [recipe-patterns.md](recipe-patterns.md) — General recipe patterns
- [antipatterns.md](antipatterns.md) — Other common justfile mistakes
