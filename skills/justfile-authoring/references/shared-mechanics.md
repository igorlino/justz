# Shared mechanics

Applies to both variants. Read after `references/choosing-a-variant.md` and
alongside the chosen variant's reference.

## Namespace submodules (`mod`)

Use `just`'s native `mod` system: one `.just` file per namespace under a `just/`
directory, linked from the root justfile. This gives each group its own file,
clean `just <ns> <cmd>` invocation, and avoids monolithic `case` dispatchers.

**Root justfile — link each module:**

```just
mod trading  'just/trading.just'
mod strategy 'just/strategy.just'
mod quality  'just/quality.just'
```

**Submodule file — `just/<ns>.just`:**

```just
set shell := ["bash", "-eu", "-o", "pipefail", "-c"]
set working-directory := '..'
set positional-arguments

py := ".venv/bin/python"

_default:
    # the `rich` variant: @justz help <ns>
    # the `native` variant: @just --unsorted --list <ns>

subcommand-a *args:
    {{py}} -m some.module "$@"
```

Key rules:
- `set working-directory := '..'` makes paths relative to the repo root, not the `just/` dir.
- A `_default` recipe, **first in the file and private**, makes `just <ns>` list
  instead of running the module's first recipe. See the variant reference for its body.
- The `rich` variant only: add the namespace as a group in `commands.json` (and to a
  tier's `groups` array if the registry uses tiers), then run `justz fmt`.

**Recipe argument shape** — decide this before writing the `args` rows, because
the wrong shape cannot be fixed by documentation afterwards:

| Input | Shape |
|---|---|
| The primary, repeatable, or wrapped-tool-native input (paths, files, a target) | Positional variadic — `*paths` |
| A modifier, a boolean, anything optional | A named flag the recipe parses — `--force`, `--filter until=…` |
| Something whose value is an enum with exactly one member today | **Nothing.** Drop it; a knob with one legal value is noise |

Two hard rules:

- **Never put a positional parameter in front of a variadic.** The positional eats
  the first forwarded token, so `just <ns> run -- --resume` becomes impossible
  without first typing the positional's only sensible value. Once a recipe has a
  variadic, every *other* input has to be a flag or an environment variable.
- **A recipe that wraps a single command must not consume the wrapped tool's
  flags.** Detect and forward instead, so the tool keeps its own flag namespace:

  ```just
  prune *flags:
      #!/usr/bin/env bash
      set -euo pipefail
      for f in "$@"; do
          case "$f" in --force|-f) exec sometool prune "$@" ;; esac
      done
      exec sometool prune --dry-run "$@"
  ```

  A recipe that terminates (it is not a thin wrapper) may consume its own flags.

**`set positional-arguments`** — add it to any module whose recipes parse or
forward arguments, and iterate `"$@"` rather than `{{ var }}`:

```just
set positional-arguments

run *paths:
    #!/usr/bin/env bash
    set -euo pipefail
    for p in "$@"; do echo "got <$p>"; done
```

Interpolating a variadic re-splits it on whitespace, so `{{ paths }}` shreds any
argument containing a space — `just m naive "a dir with spaces"` arrives as four
separate tokens. Iterating `"$@"` preserves exactly what the caller quoted.
Verified on just 1.58.0 and 1.59.0, including inside a `mod` submodule and on `[no-cd]`
recipes. Three details that matter:

- `{{ var }}` interpolation keeps working; this only *adds* `"$@"`.
- **`"$@"` is every recipe argument, not just the variadic.** A recipe declared
  `deploy env *rest:` sees `$1` as `env`, so `shift` past the named parameters
  before iterating, or you will treat them as forwarded input.
- It is module-scoped, so adding it affects every recipe in that file.

**`mod` behaviour that will surprise you** (verified on just 1.58.0; `justfile_directory()`, `source_directory()` and `[no-cd]` re-checked on 1.59.0):

| Expression | Inside a `mod` submodule |
|---|---|
| `justfile_directory()` | Returns the **root** justfile's directory, not the submodule's — use it to build repo-root paths |
| `source_directory()` | Returns the submodule's own directory (`just/`) |
| `set working-directory := '..'` | Resolved relative to `source_directory()`, so `..` is the repo root |
| `[no-cd]` on a recipe | Overrides the above and runs in the user's current directory — required for recipes that act on wherever the user happens to be |
| `just FOO=bar <ns> <cmd>` | **Does not work.** Module variables cannot be set from the CLI: 1.58.0 ignored it and exited 0, 1.59.0 fails with an error. Anything a caller must override has to be an environment variable |


## Recipe bodies belong in `scripts/<ns>/`

A justfile is a dispatch table, not a place to store shell. Inline bodies cannot
be syntax-checked, linted, unit-tested, or run without `just`, and they hide
quoting bugs until the recipe is first invoked.

**Extract** when a body has a `#!/usr/bin/env bash` shebang, shell control flow,
more than one statement, or non-trivial quoting. **Keep inline** when it is a
single tool invocation — `npm run build`, `{{py}} -m pytest "$@"`.

```just
set positional-arguments
sh := justfile_directory() / "scripts/deploy"

staging *ARGS:
    @{{ sh }}/staging.sh "$@"
```

Each extracted script then stands on its own:

```bash
bash -n scripts/deploy/staging.sh           # parses
shellcheck -S warning -x scripts/**/*.sh    # lints
scripts/deploy/staging.sh --help            # runs without just
```

When moving existing scripts, use `git mv` to preserve history and grep for
references first — shell files are routinely pinned by paths outside the repo's
control (`.envrc`, CI config, editor tasks, other scripts). Sourced-only helpers
stay non-executable; prefix them with `_` so they do not read as commands.

Derive `$ROOT` inside each script from its own `BASH_SOURCE`, never from the
caller's working directory — `[no-cd]` recipes deliberately run elsewhere.
Once three or more scripts share a preamble, factor it into a single
`scripts/lib/bootstrap.sh` that they all source.

## The `doctor` recipe

Add a `doctor` recipe that checks every tool a contributor needs. Its body goes
in `scripts/core/doctor.sh`, per "Recipe bodies belong in `scripts/<ns>/`" above.
Include `justz` in a `rich` repo, since `help`, `check` and the shims need it (it
needs only `just` itself); derive everything else from the project rather than
hardcoding a generic list.

```just
doctor:
    @{{ sh }}/doctor.sh
```

Not every tool reports its version through `--version`: `kubectl`, `kustomize`
and `go` use a subcommand, and guessing wrong prints an error message into the
version column instead of a version. Take the version arguments per tool:

```bash
check() {   # check <tool> <install-hint> [version-args...]
    local tool="$1" hint="$2"; shift 2
    if command -v "$tool" &>/dev/null; then
        printf "  ✅  %-12s %s\n" "$tool" "$("$tool" "${@:---version}" 2>&1 | head -1)"
    else
        printf "  ❌  %-12s not found — %s\n" "$tool" "$hint"
        ok=0
    fi
}

check justz   "brew install igorlino/tap/justz"
check kubectl "https://kubernetes.io/docs/tasks/tools/" version --client
```

Check more than binaries: a missing `.env`, an uninitialised direnv, or an
absent config file strands a new contributor just as effectively. Separate
required tools from optional ones so an optional gap does not fail the run.

Document `doctor` the same way as every other recipe: a `[doc]`/`[group]` pair, and
also a `commands.json` entry in the `rich` variant.

`justz` is a hard dependency of a `rich` repo's `help` and `check` recipes. Put the
check in the scaffolded `doctor` script itself — a dependency that was only a prose
checklist item was never added in the one repo we can audit. `justz doctor` and
`justz shim doctor` check `just` and the shims.

## Update developer workflow docs

Search the repository for any documentation or agent instructions that still
reference raw tool commands (e.g. `make build`, `python -m pytest`, `npm run`,
`cargo test`) and replace them with their `just` equivalents. Common locations:

- `README.md`
- `AGENTS.md` (or equivalent agent instructions file)
- Any skill checklists under `.agents/skills/*/SKILL.md`

