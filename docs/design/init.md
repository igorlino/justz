# Design note: `justz init`

## Purpose

Start a project with a justfile that documents itself, so a new repo begins with the layout `justz check` expects and nobody hand-copies a module header.

## Command

```
justz init [--variant native|rich] [--name NAME] [--module NAME]... [--dry-run] [dir]
```

## What it writes

| File | Variant | Content |
|---|---|---|
| `justfile` | both | `import` of the preamble, one `mod` per module (with `[group]` and `[doc]`), private `_default`, `help` (native) or `help` and `check` (rich) |
| `just/preamble.just` | both | the settings every file shares |
| `just/<module>.just` | both | `import 'preamble.just'` and a private `_default` that lists the module |
| `config/commands.json` | rich | `core` with `check` and `help`, one empty group per module, in `justz fmt` order |

`native` is the default, because the skill recommends it for small repos and it needs only `just`. `rich` writes recipes that call `justz`, so it needs `justz` wherever the help is rendered.

## Decisions

1. **Shared settings through `import`.** Verified on just 1.59.0: a `set` line in an imported file applies to the importing file, and to a module that imports it. A module does not inherit the root's settings, so the import is what keeps a module from running with `sh` and no `pipefail`. This settles the open question in the plan.
2. **No `working-directory` in the preamble.** The root runs in the project directory and a module in the directory of its file. A relative value fits only one of them (`'..'` sent the root one level above the project). Recipes use `justfile_directory()` for paths from the project root.
3. **No shared `_default`.** Each module needs its own heading argument, and an imported `_default` collides with the module's own (a duplicate recipe error). Only `set` lines are shared.
4. **Never overwrite.** If any target file exists, or the directory has a `justfile` or `.justfile` under any letter case, nothing is written and the exit code is `1`. All conflicts are checked before the first write, so a refusal never leaves half a project. There is no `--force`: adopting an existing repo is `justz assess` and `justz generate`, not scaffolding.
5. **Plain-text names only.** The project name goes inside a shell-quoted heading, so it is limited to letters, digits, space, dot, dash and underscore. Module names are lowercase and dashes, and `core`, `help`, `check` and `preamble` are reserved because they would collide with a recipe, a registry group or the shared file.
6. **Rich registry is built, not typed.** The text is parsed and canonicalised with the same code as `justz fmt`, so `justz fmt --check` passes on the first run. The `[doc]` text of `help` and `check` equals their `desc`, so `justz check` passes on the first run too.
7. **No network, no `just` call.** `init` only writes files. It does not need `just` installed, and writes with the atomic writer, which refuses to write through a symbolic link. Before the first write it also refuses a symbolic link or a plain file where `just/` or `config/` must be a directory, and a `justfile` under any case (just matches the name case-insensitively, so `JUSTFILE` plus `justfile` is an error).

## Not in scope

- Module recipes: a scaffold with invented recipes teaches the wrong habit. The module file holds a comment instead.
- A `ci` recipe or hook wiring.
- Converting an existing justfile (see the migration reference in the `justfile-authoring` skill).

## Tests

`internal/scaffold` covers the layout, the canonical registry, the matching `[doc]` and `desc`, and every validation rule. `cmd/justz/init_test.go` covers writing, `--dry-run`, refusal without writing, and usage errors. The generated projects were also run with a real `just`, `justz check` and `justz fmt --check`.
