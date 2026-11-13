# Input model

What `justz` reads, where it looks, and how the pieces combine. The renderers, `check`, `fmt`, `completions` and `generate` are built on this, so it is written down before they are.

## The three variants

A project documents its commands in one of three ways, and `justz` supports all of them. Fixtures for each are in `testdata/variants/`.

| Variant | Where the documentation lives | Needs |
|---|---|---|
| `native` | In the justfile itself: `[doc]`, `[group]` and `[arg(..., help=...)]` attributes | Only `just` |
| `rich` | In a registry file, `commands.json`, committed to the repository, next to the justfile | `just` and the registry |
| `local-rich` | In a personal, git-ignored registry-shaped file, `.just-local-rich.json`, generated from a `native` justfile | `just` and the generated file |

`local-rich` exists so one developer can get the richer help and completion on a `native` repository the team did not convert. It leaves no file in the repository that the team has to accept.

## Inputs

1. **The justfile dump (always).** `justz` reads the justfile through `just --dump --dump-format json`, including nested `modules`. It never parses justfile text itself. `just` is called with a fixed argument list and never through a shell.
2. **The registry, `commands.json` (rich).** Found by walking up from the working directory to the project root, looking in each directory for `.*/config/commands.json` (sorted by name), then `config/commands.json`, then `just/commands.json`. `--config <path>` and `$JUSTZ_CONFIG` override the search.
3. **The local overlay, `.just-local-rich.json` (local-rich).** Found in the project root. It has the same shape as the registry.

The project root is the first directory that contains a `.git` entry, or the filesystem root if there is none.

## Which variant is in effect

Decided in this order, first match wins:

1. A registry (input 2) exists: the project is `rich`. The overlay is ignored, because a committed registry is the team's choice and a personal file should not disagree with it.
2. Otherwise a local overlay (input 3) exists: `local-rich`.
3. Otherwise: `native`. Help is rendered from the dump alone.

`justz doctor` will report the variant it found and the files it used.

## How `native` is rendered from the dump

| Shown | Comes from |
|---|---|
| Group heading | The recipe's `[group]` attribute |
| Description | The recipe's `[doc]`, first line |
| Example | A `[doc]` line starting with `example:`, text before a `\|` is the command |
| Arguments | The recipe's `[arg(..., help=...)]` attributes. A flag is `-s, --long`, a positional is its name |

## What the old generator did, and what we do differently

The shell generator that produced the overlay behaves in ways we should not copy without deciding. Each is checked against `testdata/variants/native/local-rich.reference.json`.

| Observed behavior | Decision for `justz generate` |
|---|---|
| Every root recipe lands in `core`, ignoring its `[group]` attribute (a recipe grouped `deploy` ends up under `core`). Only submodules become groups | Keep it. In a registry a group is also the path used to run a command (`just <group> <command>`), so a `[group]` attribute, which is only a display heading, must not become a registry group: it would break name resolution and shims. Groups are modules, root recipes are `core`. Only top-level modules are generated; nested ones are skipped with a warning, because a group name is a single word. |
| A flag declared with `value='true'` is rendered as `-w, --watch <watch>`, as if it took a value. The dump marks such a parameter with `value` set while `flag` is `false`, and the old generator only looked at `flag` | Treat a parameter with a `value` as a flag with no placeholder. |
| The output is derived and nothing survives a re-run, except that people hand-add `values: [...]` for enum completion and lose them | Preserve hand-added `values` when regenerating. |
| Multi-line `[doc]` text is cut to its first line | Keep that, and have `check` warn about multi-line comments, because `just --list` shows only their last line. |

The reference file is what the shell generator produced, so the first two rows can be seen in it.

## Versioning and failure

- The registry gets an optional `schemaVersion`. A newer major version than `justz` understands is an error, not a guess.
- If `just` is missing, every command that needs the dump stops with a clear message and a distinct exit code. `justz doctor` reports the same.
- If the dump format changes, `justz` stops with the `just` version it saw, rather than rendering something wrong.
- Unknown fields in the registry are kept and written back unchanged by `fmt`.

## Open questions

- Should `justz` write a `schemaVersion` when `fmt` rewrites a registry that has none? Leaning no: `fmt` should not add fields.
- Do we want a way to name the variant explicitly (a flag or a registry field), for projects that have both a registry and an overlay and want the overlay?
