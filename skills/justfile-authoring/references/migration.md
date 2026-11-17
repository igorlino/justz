# Migration

## A repo with the legacy bash help system → `justz`

A `rich` repo from before `justz` has a vendored `scripts/lib/help.sh` (about 1,200 lines of bash, needing `jq`), completion scripts sourced from the skill, a `jz` script, and a `just/common.just` with `_check-doc`. `justz` replaces all of it, reads the same `commands.json`, and removes the per-repo copy that had to stay byte-identical. The registry does not change.

Do this as one reviewed change:

1. **Install `justz`** (see the `justz-cli` skill) and check `justz --version` is 0.5.1 or newer. Stop with install advice if it is missing.
2. **Record the failing tests before starting.** Run the project's tests and keep the list of what already fails, so every later failure has a baseline and none is blamed on the migration. Most of the work in a large repo is in the tests and guard scripts around the justfile, not in the justfile.
3. **Run `justz check` and `justz fmt --check` on the untouched repo first.** They show what the old guard missed (descriptions that diverge, entries with no recipe) so the migration commit does not hide them.
4. **Search the project before you change anything**, and list what the change can break:
   - Code that reads a description: tests and scripts that assert a phrase in `desc`, or parse `commands.json` or the `just --list` text. After `detail` is in use a command's text is `desc` plus `detail`, so move such code to `justz help <group> <command> -j`, which prints both, and do not assert on `desc` alone.
   - The old guard: tests and scripts that name it (`_check-doc`, `check-just-docs`) or that extract recipe bodies from the justfile text. Fix or delete them before the guard goes (step 7).
   - Guards, hooks and path rules that treat `just/` specially. A guard that scans only the `just/` files passes silently once the bodies have moved and nothing is left to scan (step 11).
5. **Rewire the justfile.** Replace the old bodies with the `justz` ones from `references/variant-rich.md`:
   - `_default` → `@justz help`, and each submodule's default recipe → `@justz help <ns>`. Every module needs one, private and first in its file: a module with none runs its first recipe on a bare `just <ns>`, and one that runs `just --list --justfile {{ justfile() }}` lists the root justfile, because `justfile()` inside a module is the root. `justz lint` reports both (`module-dispatcher-missing`, `module-dispatcher-lists-root`) and `justz check` warns about a public `default` (`default-public`). Check `just <ns>` for each module, since the migration is easy to finish without this step.
   - `help *ARGS` → forwards `"$@"` to `justz help` (no words means `--expanded`)
   - `check` → `@justz check` and `@justz fmt --check` instead of `bash scripts/lib/help.sh --check`
   - Switch the help renderer to `justz` before, or in the same change as, any move of long descriptions into `detail` (step 10). The old renderer does not show `detail`, so the facts would disappear from the help people use.
6. **Move the title.** If the old `help.sh` had a hand-edited `_help_header()`, put the title and tagline in `"name"` and `"tagline"` at the top of `commands.json`.
7. **Delete** `scripts/lib/help.sh`, `just/common.just` (and the `import 'common.just'` and `(_check-doc …)` dependencies), `scripts/lint/check-just-docs.sh` if present, and any copied `completion.zsh`/`completion.bash`, once the search in step 4 has nothing left that names them.
8. **Replace the legacy shims.** A wrapper script that exec'd the skill's `jz` or `just --justfile …` becomes `justz shim add <root> --soft`. `justz` will not overwrite a file it did not write, so remove the old wrapper first. Replace the `source …/completion.zsh` line in the shell startup file with `eval "$(justz completions zsh)"` (bash: `bash`).
9. **Run `justz fmt`**, then `justz check`, and fix what it reports: `[doc]` divergences, missing `[group]`, bare or long-flag examples.
10. **Write the descriptions** of any command whose `desc` is long, by hand, from what the recipe does and not from the old text (`references/variant-rich.md` says what goes in `desc` and in `detail`). Keep the old text only as a source of facts.
11. **If bodies move to `scripts/`, prove the move.** Compare `just --dry-run <recipe>` before and after, with each exported variable replaced by its value: the two must be the same command. After you re-point a test or a guard at the new place, delete the line it tests and check that it fails. A test that cannot fail is not a test.
12. **Update docs.** README and `AGENTS.md` should say discovery is `just`, `just <ns>`, `justz help [group] <cmd>`; delete any instruction to read `commands.json` or `scripts/lib/help.sh`. Record the shape and the reason in `AGENTS.md`.
13. **Enforce it.** Removing the old runtime guard also removes its coverage in CI and in commits, so add `justz check` back: a pre-commit hook (`justz-check` from the `igorlino/justz` repository, or the plain `scripts/pre-commit-hook.sh`) and a GitHub Actions job that installs a pinned `justz`. The snippets are in `docs/hooks-and-ci.md` of the `justz` repository. A hook must fail, not skip, when `justz` is missing; `JUSTZ_HOOK_SKIP_MISSING=1` is the explicit, loud opt-out.
14. **Verify** `just`, `just help`, `just <ns>` for every module, `justz help <group> <cmd>`, `justz shim doctor`, and that completion works in a fresh shell. Run the tests again and compare with the baseline from step 2. Then add an undocumented recipe, confirm `just check` fails, and remove it.

Withdrawn advice from older versions: "do not run `just --help`, read `commands.json`." `just --list --list-submodules` and `just --dump --dump-format json` cannot be stale; prefer them for discovery in any repo.

## Converting `rich` → `native`

Only on explicit request, and only after showing the user what they lose: `detail`, `tags`, `exits`, `env`, `seeAlso`, `tiers`, structured `examples` and bare-name lookup have no home in `native`. Count them first and put the number in front of the user:

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

Do this when every machine that renders the help can install `justz` (the rule in `references/choosing-a-variant.md`). First add the `_default`, `help` and `check` recipes from `references/variant-rich.md`, so that the generator lists them. Then generate the registry from the justfile rather than writing it by hand, so the first version cannot already be wrong:

```bash
mkdir -p config
justz generate --stdout > config/commands.json
justz fmt
justz check
```

Remove the `$comment` line: it says the file is generated, and it is now edited by hand. Fix what `justz check` reports; for a `[doc]` with an `example:` line it prints the one-line `[doc]` to paste. Top-level recipes land in the `core` group: move them if the project has other groups.

Then add `detail`, `tags`, `exits`, `env` and `seeAlso` by hand: those are what `native` cannot hold. Keep `[doc]` in place and identical to `desc`; `justz check` will fail if they drift.

## Adding the personal overlay instead

If only one developer wants the richer help, do not convert the repo. See `references/variant-local-rich.md`.
