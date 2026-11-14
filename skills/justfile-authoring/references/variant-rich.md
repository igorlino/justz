# The `rich` variant: `commands.json` rendered and checked by `justz`

Documentation lives in a registry, `commands.json`, next to the justfile. `justz` renders the help from it, keeps it sorted, checks it against the justfile, and generates shell completion from it. Read `references/choosing-a-variant.md` first: `rich` is the exception, not the default.

> **Known cost, state it to the user:** `desc` duplicates the recipe's own `[doc]`. In the one adoption we can audit, 20 of 32 descriptions had diverged within six months. `justz check` now reports every divergence (`registry-desc-diverges`), which is what makes the duplication affordable. Treat `commands.json` as authoritative, keep each `[doc]` a one-liner identical to it, and run `justz check` in `check` and `ci`.

`justz` is required wherever the help is rendered. Check `command -v justz && justz --version` first (0.5.1 or newer) and stop with install advice if it is missing. There is no fallback script.

## The files

| File | Purpose |
|---|---|
| `config/commands.json` | Single source of truth: groups, commands, descriptions, args, examples |
| `justfile` | Root wiring: `_default`, `help`, `check`, `doctor`, and `mod` links |
| `just/<ns>.just` | One submodule per command group, each with its own `_default` |
| `scripts/<ns>/<cmd>.sh` | Recipe implementations, for every body longer than one command |

`justz` finds the registry by walking up from the working directory to the project root (the directory with `.git`), looking in each directory for `.*/config/commands.json`, then `config/commands.json`, then `just/commands.json`. `justz fmt` is the exception: it looks only in the current directory, or takes a path argument or `JUSTZ_CONFIG`, so run it from the project root. A committed registry wins over a personal `.just-local-rich.json`.

Nothing here is vendored. There is no `help.sh` to copy, keep identical across repos, or update: upgrading `justz` upgrades every repo.

## Three help views

| View | Command | Shows |
|---|---|---|
| Summary | `just`, or `justz help` | Core commands, then each group, one line per command |
| Expanded | `justz help --expanded` | Every group with its commands and argument rows. `--wide`, `-w` adds example columns (on by default from 100 columns) |
| Zoom | `justz help <group>`, `justz help [group] <cmd>` | One group (usage, commands, examples, cross-references) or one command (how to invoke it, examples, options). An exact name or a unique prefix works; an ambiguous name exits `2` and lists the groups |

Add `-j` (or `--json`) for machine-readable output, `-b` to browse with `fzf` (or a numbered menu).

## 1. Create `config/commands.json`

For each recipe write a description that says what it does, not just which tool it calls:

- `desc` is imperative, specific and has no trailing period: `Type-check then bundle → dist/static/` beats `Build the project`.
- Add `args` entries for every meaningful flag or positional parameter. Never use one generic `[args]` placeholder.
- Design short single-letter aliases (`-x, --long`) for every flag whose name has three or more letters. Two-letter names (`--id`, `--ci`, `--ui`) stay long-only unless there is a real UX reason; ask the user.
- Write the short form first and the value placeholder once, at the end: `-s, --surface <name>`.
- Add an `example` string (prefixed with `… `) for parameterised commands, using short flags.
- Every group-level `examples` entry must add information the argument rows do not: a value-taking flag (especially enum `values`), several flags together, passthrough arguments (`-- …`), or an environment override. Never a bare command, and never a single boolean flag that an argument row already shows. `justz check` rejects both (`example-bare-command`, `example-single-flag`) and also flags long options where a short alias exists (`example-long-flag`) and an enum that no example uses (`example-enum-missing`).
- Run **`justz fmt`** after every edit. It writes the canonical order: groups (`core` first, then A to Z), commands A to Z, arguments (positional first, then flags A to Z), examples by `cmd`, `seeAlso` and `tiers[].groups` A to Z. `justz fmt --check` writes nothing and exits `1` when the file is not formatted.

To start from an existing justfile instead of a blank file, `justz generate --stdout` prints a registry-shaped starting point from the justfile's own `[doc]` and `[arg]` attributes. Review it, then add the `args`, `examples` and `seeAlso` that are the reason for choosing `rich`.

## 2. Wire the root `justfile`

The root justfile stays small: `_default`, `help`, `check`, and the `mod` links. Every command-group recipe lives in `just/<ns>.just`.

```just
set shell := ["bash", "-eu", "-o", "pipefail", "-c"]
set positional-arguments

mod trading  'just/trading.just'
mod strategy 'just/strategy.just'
mod quality  'just/quality.just'

_default:
    @justz help

[doc('Full command reference, or the focused docs of one group or command (help [group] [term]); -b browses, -j emits JSON')]
[group('core')]
[no-exit-message]
help *ARGS:
    #!/usr/bin/env bash
    set -euo pipefail
    # No words: the full reference. Otherwise zoom, with -b, -j and -w.
    if [[ $# -eq 0 ]]; then exec justz help --expanded; fi
    exec justz help "$@"

[doc('Check the documentation against the justfile and the registry order')]
[group('core')]
check:
    @justz check
    @justz fmt --check
```

`help` forwards with `"$@"` (which needs `set positional-arguments`), never `{{ ARGS }}`: interpolation splits quoted arguments and breaks the zoom lookup, `-b` and `-j`.

Each submodule gets a private `_default` as its first recipe so `just <ns>` shows the group instead of running its first recipe:

```just
_default:
    @justz help <ns>
```

## 3. The checks

`justz check` reads the justfile through `just --dump` and the registry, and fails on errors (`--strict` makes warnings fail too, `--json` gives machine-readable findings with `rule`, `severity`, `subject`, `message` and `fix`):

| Rule | Meaning |
|---|---|
| `doc-missing`, `group-missing` | A public recipe has no `[doc]`, or a top-level recipe has no `[group]` |
| `doc-multiline-comment` | A comment block above a recipe shows only its last line in `just --list`; use `[doc]` |
| `variadic-interpolation` | A variadic parameter is interpolated and so split on whitespace |
| `registry-missing`, `registry-stale` | A recipe has no registry entry, or an entry has no recipe |
| `registry-desc-diverges` | `[doc]` and `desc` differ |
| `registry-key-signature` | A key carries an argument signature instead of a bare recipe name |
| `registry-desc-long` | A description is over 120 characters (warning) |
| `registry-unsorted` | Fixed by `justz fmt` |
| `example-*` | The four example rules above |

Wire `justz check` and `justz fmt --check` into `check` and `ci`, then add a deliberately undocumented recipe and confirm both fail.

## `commands.json` schema quick reference

```jsonc
{
  "name": "myproj",                  // optional; the heading falls back to the directory name
  "tagline": "One-line summary",     // optional
  "cli": "tz",                       // optional; the name shown in usage lines and examples
  "groups": {
    "core": {                        // top-level recipes
      "label": "Core",
      "commands": {
        "<recipe-name>": {
          "desc": "Imperative description, specific, no trailing period",
          "example": "… <arg>",      // optional; prefix with "… "
          "args": [
            { "name": "-s, --surface <name>", "desc": "Short first, value once at the end" },
            { "name": "-y", "desc": "Short-only flags stay as they are" },
            { "name": "--env <name>", "desc": "Target environment",
              "values": ["demo", "int", "production"] }   // drives enum completion and renders (a|b|c)
          ]
        }
      }
    },
    "<ns>": {                        // a module: run as `just <ns> <sub>`
      "label": "Human-readable label",
      "commands": { "<sub-command>": { "desc": "…", "args": [] } },
      "extra": { "<special-cmd>": { "desc": "…" } },     // entries that do not follow `just <ns> <sub>`
      "examples": [ { "cmd": "just <ns> <sub> -x val", "desc": "what this does" } ],
      "footer": "Prose shown under the examples",
      "seeAlso": ["other-ns"]
    }
  },
  "tiers": [                         // optional ordered sections of groups
    { "label": "Commands", "hint": "run without args to see sub-commands", "groups": ["<ns-1>"] }
  ]
}
```

Author examples as `just <ns> <cmd> …`. When `cli` is set, `justz` swaps that prefix at display time (`tz <ns> <cmd> …`) and `justz check` still parses the `just` form. When a command is also reachable through another CLI (an `npx` wrapper, a binary), a second example row for it is the one near-duplicate that earns its place.

## Shims: run the project from anywhere

A shim pins a short command (such as `tz`) to this project's justfile so it works from any directory, in non-interactive shells and for agents, and resolves bare names (`tz git` for `tz tool git`).

```bash
justz shim add /path/to/project --soft    # the name comes from "cli" in commands.json, or pass --name
justz shim doctor                         # checks PATH order, stale stubs, missing justfiles
```

- `--soft` falls back to plain `just` when the help system is missing, so a shim can never break the wrapped command.
- `justz shim add` refuses to overwrite a file it did not write. `--replace` re-points a shim another project owns but cannot override a foreign file.
- A help name `<name>h` (`tzh`) is registered too.
- `jz` and `jzh` are shortcuts for the project around the current directory. They exist when the binary runs under that name (Homebrew installs the links).
- Bare names are for typing at a prompt. They break when a command is later added to a second group. Scripts and agents use the full path (`tz tool git`); `justz resolve <name>` (exit `0` one match, `2` several, `3` none) is the deterministic lookup.

## Shell completion

```bash
eval "$(justz completions zsh)"     # in ~/.zshrc, after compinit
eval "$(justz completions bash)"    # in ~/.bashrc
```

The script is small; each Tab press asks `justz` what can come next from the registry, so what completes always matches the project you are in. It lists groups, commands, flags with descriptions (zsh), and enum values from `values`; a free-text value falls back to file completion. Run the command again after adding or removing a shim so the new names are included.

Smoke-test completion in a pseudo-terminal after changing it: unit tests that stub the shell's completion function cannot see wiring faults. Include an empty-word case (`tz git ` as well as `tz git -`), because zsh hides option candidates until a `-` is typed.
