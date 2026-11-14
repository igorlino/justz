# Migration

## A repo with the legacy bash help system → `justz`

Older versions of this skill scaffolded `rich` repos with a vendored `scripts/lib/help.sh` (about 1,200 lines of bash, needing `jq`), completion scripts sourced from the skill, a `jz` script, and a `just/common.just` with `_check-doc`. `justz` replaces all of it, reads the same `commands.json`, and removes the per-repo copy that had to stay byte-identical. The registry does not change.

Do this as one reviewed change:

1. **Install `justz`** (see the `justz-cli` skill) and check `justz --version` is 0.5.1 or newer. Stop with install advice if it is missing.
2. **Run `justz check` and `justz fmt --check` on the untouched repo first.** They show what the old guard missed (descriptions that diverge, entries with no recipe) so the migration commit does not hide them.
3. **Rewire the justfile.** Replace the old bodies with the `justz` ones from `references/variant-rich.md`:
   - `_default` → `@justz help`, and each submodule's `_default` → `@justz help <ns>`
   - `help *ARGS` → forwards `"$@"` to `justz help` (no words means `--expanded`)
   - `check` → `@justz check` and `@justz fmt --check` instead of `bash scripts/lib/help.sh --check`
4. **Delete** `scripts/lib/help.sh`, `just/common.just` (and the `import 'common.just'` and `(_check-doc …)` dependencies), `scripts/lint/check-just-docs.sh` if present, and any copied `completion.zsh`/`completion.bash`.
5. **Move the title.** If the old `help.sh` had a hand-edited `_help_header()`, put the title and tagline in `"name"` and `"tagline"` at the top of `commands.json`.
6. **Replace the legacy shims.** A wrapper script that exec'd the skill's `jz` or `just --justfile …` becomes `justz shim add <root> --soft`. `justz` will not overwrite a file it did not write, so remove the old wrapper first. Replace the `source …/completion.zsh` line in the shell startup file with `eval "$(justz completions zsh)"` (bash: `bash`).
7. **Run `justz fmt`**, then `justz check`, and fix what it reports: `[doc]` divergences, missing `[group]`, bare or long-flag examples.
8. **Update docs.** README and `AGENTS.md` should say discovery is `just`, `just <ns>`, `justz help [group] <cmd>`; delete any instruction to read `commands.json` or `scripts/lib/help.sh`. Record the shape and the reason in `AGENTS.md`.
9. **Verify** `just`, `just help`, `just <ns>`, `justz help <group> <cmd>`, `justz shim doctor`, and that completion works in a fresh shell. Then add an undocumented recipe, confirm `just check` fails, and remove it.

Withdrawn advice from older versions: "do not run `just --help`, read `commands.json`." `just --list --list-submodules` and `just --dump --dump-format json` cannot be stale; prefer them for discovery in any repo.

## Converting `rich` → `native`

Only on explicit request, and only after showing the user what they lose: `args` rows, `examples`, `seeAlso` and `tiers` have no structured home in `native`. Count them first and put the number in front of the user:

```bash
jq '[.groups[].commands[].args // [] | length] | add' config/commands.json
jq '[.groups[].examples // [] | length] | add' config/commands.json
```

1. Move every `desc` into `[doc('…')]` on its recipe. This is the point of the change; do it first and verify with `just --list`.
2. Add `[group('…')]` to top-level recipes and to `mod` statements.
3. Convert `args` rows to `[arg(…, help='…')]`. Boolean flags need `set unstable` + `set lists` in that module; read the warning in `references/variant-native.md` about mapped dependencies breaking.
4. Decide, per namespace, where `examples`, `footer` and `seeAlso` go: a multi-line `[doc]` (reachable with `just explain`, costs `--list` alignment), the README, or nowhere. **Not `printf`.**
5. Replace the `_default` and `help` bodies with the `just --unsorted --list` ones from `references/variant-native.md`.
6. Delete `commands.json`. Install the inline guard (`references/variant-native.md`, section 7).
7. Update README and `AGENTS.md`: the discovery instruction changes to `just --list --list-submodules`. Record the shape and the reason.

Expect roughly a 55% line reduction in the command surface. Confirm `just`, `just help`, `just <ns>`, every parameterised recipe under `just -n`, and the guard all still behave before committing.

## Converting `native` → `rich`

Rare: only when a repo outgrows `native` (thresholds in `references/choosing-a-variant.md`). Generate the registry from the justfile rather than writing it by hand, so the first version cannot already be wrong:

```bash
justz generate --stdout > config/commands.json
justz fmt
justz check
```

Then add `args` detail, `examples`, `seeAlso` and enum `values` by hand: those are the reason to convert. Keep `[doc]` in place and identical to `desc`; `justz check` will fail if they drift. Add `help`, `check` and the `_default` recipes from `references/variant-rich.md`.

## Adding the personal overlay instead

If only one developer wants the richer help, do not convert the repo. See `references/variant-local-rich.md`.
