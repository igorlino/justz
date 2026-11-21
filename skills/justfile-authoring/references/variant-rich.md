# The `rich` variant: `commands.json` rendered and checked by `justz`

Documentation lives in a registry, `commands.json`, next to the justfile. `justz` renders the help from it, keeps it sorted, checks it against the justfile, and generates shell completion from it. Read `references/choosing-a-variant.md` first: `rich` is the default, on the condition that every machine that renders the help can install `justz`.

> **Known cost, state it to the user:** `desc` duplicates the recipe's own `[doc]`. In the one adoption we can audit, 20 of 32 descriptions had diverged within six months. `justz check` now reports every divergence (`registry-desc-diverges`), which is what makes the duplication affordable. Keeping the `desc` to one short line (and moving longer prose to `detail`, 0.7.0 or newer) is what makes it cheap to keep in step. Treat `commands.json` as authoritative, keep each `[doc]` a one-liner identical to it, and run `justz check` in `check` and `ci`.

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
| Zoom | `justz help <group>`, `justz help [group] <cmd>` | One group (usage, commands, examples, cross-references) or one command (how to invoke it, examples, options, then its `detail`). An exact name or a unique prefix works; an ambiguous name exits `2` and lists the groups |

Add `-j` (or `--json`) for machine-readable output (a command's zoom carries `detail`, `exits`, `env`, `tags` (left out when empty) and `seeAlso`, and a group's listing carries `hasDetail` per command; listings and the array printed for an ambiguous name do not), `-b` to browse with `fzf` (or a numbered menu).

The text `justz` itself prints from a registry or a justfile is made safe for a terminal (0.7.0 or newer). The output of `just` and of the recipes passes through unchanged.

- An escape sequence, a control character, a bidirectional override or invalid UTF-8 in a description is shown as U+FFFD.
- A group or command key that is not a plain name (letters, digits, `_`, `.` and `-`, not starting with `.` or `-`) still shows in the help, but completion never offers it and `justz resolve` never reports it.
- Completion offers an enum value from `values` only when it is one shell-safe word (letters, digits and `_ . , : @ % + = / ~ -`). A value with a space or a `;` is shown in the help and never inserted.
- `-j` and `--json` carry the same cleaned text: a `\r\n` in a `detail` reads `\n`, and a newline or tab in a single-line field such as `desc` reads as a space.

## 1. Create `config/commands.json`

For each recipe write a description that says what it does, not just which tool it calls:

- `desc` is one line, the same text as the recipe's `[doc]`. Aim for 80 to 100 characters (the limit is 120). Start with a verb, and say the effect, not the mechanism: `Type-check then bundle → dist/static/` beats `Build the project`. Write it from what the recipe does, never from an older description or a split of one; a recipe that only prints steps "prints the steps to ...". Add the fact that would make someone hesitate (it deploys, it deletes, it dry-runs by default). Prefer words to numbers, which go stale. No quotes or backticks, and no trailing period. The test: can a reader pick this command from the list alone.
- A fact that prevents damage goes in `desc`, whatever the length budget says.
- `detail` (0.7.0 or newer) is empty for most commands, and an empty one is correct. Write one only when a reader who has found the command still needs a fact `desc` cannot carry. Aim for about 400 characters, written after `desc`, as facts and not narrative. 400 is what to aim for; `check` only warns at 600, so a `detail` between the two is acceptable but should earn its length. In this order, until the facts run out: what the command does not do; what it needs first; what it changes and how to undo it; when to run it and what next; its exit codes and where output goes; how it differs from a sibling. It works on `commands` and on `extra` entries. An older `justz` ignores the field and shows the short `desc`.
- Document a command's exit codes in `exits`, not in prose: an object from the code, as a string, to one line of meaning, such as `{"0": "no errors", "1": "an error, or a warning with --strict", "2": "usage error"}`. A key is an integer from 0 to 255 (`justz check` reports any other key, `registry-exits`), `justz fmt` sorts the keys by number, and the zoom shows an **Exit codes** section after Options. List the codes a caller acts on; `0` is optional. `justz check` cannot tell whether the recipe really exits with a code, so check each against the script.
- Document the environment variables a command reads in `env`, not in prose: an array of rows like `args`, each with `name` (the variable, such as `STORAGE_HOST`), `desc`, and optionally `required` (a boolean) and `default` (text, shown and not parsed). A flag goes in `args`; a variable that is set around the command goes in `env`; do not list one in both. `justz fmt` sorts the rows by name, the zoom shows an **Environment** section after Options and **Exit codes**, and `justz check` reports a row without a name or with a duplicate name (an error) and a name that is not upper case letters, digits and underscores (a warning), `registry-env`. It cannot tell whether the command reads the variable, since a script the recipe runs reads its own, so check each row against the script.
- Declare what a command touches in `tags`, an array of free strings (the zoom shows a **Tags** line under the description, before the examples; `justz fmt` sorts them A to Z). Use the recommended words so a reader knows them; the list can grow without a release of `justz`, and a word outside it is allowed:

  | Tag | Meaning |
  |---|---|
  | `read-only` | Changes nothing outside the working files: no write to a service, a host or a database |
  | `idempotent` | Running it again has the same effect as running it once |
  | `destructive` | Deletes or overwrites something that cannot be recovered from the command alone |
  | `production` | Touches a production system |
  | `spends` | Costs money or uses a quota |
  | `asks-first` | Stops to ask for a confirmation before it acts |
  | `long-running` | Takes minutes, not seconds |
  | `needs-terminal` | Needs an interactive terminal (a prompt, a pager, a login) |

  **Absence never means safe.** A command with no `tags`, or without `read-only`, is unknown, not harmless: never read a missing tag as a permission, and never require `read-only` of every command. Say `read-only` only when you checked the script. A tag is acted on, so a stale one is worse than stale prose: review each against the script when the script changes. `justz check` verifies two things: `asks-first` and the recipe's `[confirm]` attribute must agree in both directions (`tags-confirm`, a warning, since a script may ask by itself: silence one recipe with `[metadata('justz-ignore', 'tags-confirm')]`; an `extra` entry has no recipe, so no such rule), and `registry-tags` warns on `read-only` together with `destructive`, `spends` or `production`, and on a duplicate. It cannot derive any other tag, and does not warn on a word outside the list. An agent that runs commands must read a command's `tags` first (`justz help <group> <command> -j`).
- Steps do not go in `detail`: the steps of a script belong in the script. `justz check` warns about the shape (`detail-long` over 600 characters, `detail-no-paragraphs` over 300 with no blank line, `detail-restates-desc` for a `detail` that opens with the `desc` again).
- Other content has its own place:

  | Content | Where |
  |---|---|
  | What a flag or argument does | its `args` row |
  | A worked command line | `examples` of the group, or `example` of the command |
  | A related command | `seeAlso` of the command |
  | The steps of a script | the header comment of `scripts/<group>/<name>.sh` |
  | Why it is built this way | a decision record |
  | A procedure for a person on call | a runbook |

- **Review what you wrote.** Check each description against the script it describes: open the recipe and its script, and compare what it really does with the `desc` and the `detail`. In an audit about one in six written descriptions was wrong, and `check` cannot see it, since it compares the `desc` only with the `[doc]`.
- `detail` is shown only in the zoom (`justz help <group> <cmd>`, after the options, under a **Details** heading) and in that zoom's JSON. Listings, completion and `just --list` never show it. `justz check` compares it with nothing, so keep it current yourself.
- `detail` is plain text. A newline starts a new line and a blank line separates paragraphs; runs of spaces collapse to one and indentation is lost. No markup is interpreted.
- With colors on, the zoom colors four things in `detail` and nothing else: a `` `backtick span` `` green (a flag inside it cyan, a `<placeholder>` yellow), a flag such as `--dry-run` cyan, a `<placeholder>` yellow, and a URL underlined. So in `detail`, put commands in backticks. This does not apply to `desc` or `[doc]`.
- Name a related command in `seeAlso` (not in 0.7.0 or earlier), not in prose. A command's `seeAlso` is a list of keys, `group.command` (`core.build` for a top-level command, and `extra` entries count): use it where a `desc` or `detail` would say "mirrors", "sibling of" or "alias for". The zoom ends with a **See also** line that shows each key as the command to run, `help -j` carries the list, `justz fmt` sorts it, and `justz check` reports a key that names no command or the command itself (`registry-see-also`). Say how two commands relate, if it matters, in `detail`.
- Add `args` entries for every meaningful flag or positional parameter. Never use one generic `[args]` placeholder.
- Design short single-letter aliases (`-x, --long`) for every flag whose name has three or more letters. Two-letter names (`--id`, `--ci`, `--ui`) stay long-only unless there is a real UX reason; ask the user.
- Write the short form first and the value placeholder once, at the end: `-s, --surface <name>`.
- Add an `example` string (prefixed with `… `) for parameterised commands, using short flags.
- Every group-level `examples` entry must add information the argument rows do not: a value-taking flag (especially enum `values`), several flags together, passthrough arguments (`-- …`), or an environment override. Never a bare command, and never a single boolean flag that an argument row already shows. `justz check` rejects both (`example-bare-command`, `example-single-flag`) and also flags long options where a short alias exists (`example-long-flag`) and an enum that no example uses (`example-enum-missing`). It also warns about a flag that no argument row declares (`example-flag-unknown`; silent for a command with no rows, for a variadic row and after `--`). A leading `NAME=value` or `env NAME=value` is read through by the flag rules, and it exempts the example from the bare-command and single-flag rules (the environment is the information the argument rows do not show). The per-command `example` field (a leading `…` stands for `just <command>`) goes through the same rules.
- Run **`justz fmt`** after every edit. It writes the canonical order: groups (`core` first, then A to Z), commands A to Z, arguments (positional first, then flags A to Z), `env` rows by name, examples by `cmd`, `tags`, `seeAlso` and `tiers[].groups` A to Z. `justz fmt --check` writes nothing and exits `1` when the file is not formatted.

To start a new project instead of one file, `justz init --variant rich` (0.6.0 or newer) writes the justfile, the modules and a registry that already passes `justz check` and `justz fmt --check`; `--dry-run` lists the files first. It never overwrites.

To start from an existing justfile instead of a blank file, `justz generate --stdout` prints a registry-shaped starting point from the justfile's own `[doc]` and `[arg]` attributes. Review it, then add `detail`, `tags`, `exits`, `env` and `seeAlso`: those are what `native` cannot hold.

## 2. Wire the root `justfile`

The root justfile stays small: `_default`, `help`, `check`, and the `mod` links. Every command-group recipe lives in `just/<ns>.just`.

```just
import 'just/preamble.just'   # set shell := bash strict, set positional-arguments (justz init writes it)

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

`justz check` reads the justfile through `just --dump` and the registry, and fails on errors (`--strict` makes warnings fail too, `--rule <name>` shows one rule and the exit code follows what is shown, `--json` gives machine-readable findings with `rule`, `severity`, `subject`, `message` and `fix`):

| Rule | Meaning |
|---|---|
| `doc-missing`, `group-missing` | A public recipe has no `[doc]`, or a top-level recipe has no `[group]` |
| `doc-multiline-comment` | A comment block above a recipe shows only its last line in `just --list`; use `[doc]` |
| `default-public` | A public `default` recipe (a warning): it is a dispatcher, left out of the counts, but shows as a row; rename it `_default`; `--strict` fails on it |
| `variadic-interpolation` | A variadic parameter is interpolated and so split on whitespace |
| `registry-missing`, `registry-stale` | A recipe has no registry entry, or an entry has no recipe |
| `registry-desc-diverges` | `[doc]` and `desc` differ. The finding prints the exact `[doc(...)]` line to paste (single quotes, or double quotes with `\"` and `\\` when the text has a single quote) and carries it in the `paste` field of the JSON; there is none for a blank description or one with control characters; `justz` never edits the justfile |
| `registry-shape` | A registry field has the wrong type (`args` as a string, `examples` not an array, a row that is not an object). `justz` reads such a field as absent, so the finding names the command key and the field; unknown fields are allowed, and `justz schema` describes the same shape |
| `registry-key-signature` | A key carries an argument signature instead of a bare recipe name |
| `registry-env` | An `env` row has no name, or its name repeats (an error); a name that is not upper case letters, digits and underscores (a warning) |
| `registry-tags` | A command's `tags` has `read-only` together with `destructive`, `spends` or `production`, or repeats a tag (warnings); a tag outside the recommended list is not reported |
| `tags-confirm` | The recipe has `[confirm]` and `tags` lacks `asks-first`, or `tags` has `asks-first` and the recipe has no `[confirm]` (a warning; not applied to `extra` entries). Silence with `[metadata('justz-ignore', 'tags-confirm')]` when the script asks by itself |
| `registry-exits` | A key of a command's `exits` is not an integer from 0 to 255 in decimal (an error), or its meaning is blank (a warning) |
| `registry-see-also` | A command's `seeAlso` key is not `group.command` of a command or `extra` entry, or names the command itself (an error) |
| `registry-desc-long` | A `desc` is over 120 characters (warning). Keep a one-line `desc` equal to the `[doc]` and move the rest to `detail` |
| `registry-unsorted` | Fixed by `justz fmt` |
| `volatile-text` | A `desc` or `detail` has text that goes stale (a warning): `TODO`, `FIXME`, `WIP`, `placeholder`, `not implemented`, `coming soon`, a count such as `11 checks` or `six steps` (checks, tests, steps, jobs, stages, phases), a `$` price, an ISO date or a month with a year. Reword it, or silence one command with `[metadata('justz-ignore', 'volatile-text')]` |
| `detail-long`, `detail-no-paragraphs`, `detail-restates-desc` | The shape of a `detail` (warnings): over 600 characters; over 300 characters with no blank line between paragraphs; or opening with the words of the `desc` again (a `desc` of four or more words). Cut it, split it into paragraphs, or start with what the `desc` does not say. They are warnings, so `--strict` fails on them; a deliberately list-style `detail` with no blank lines is silenced with `[metadata('justz-ignore', '<rule>')]` |
| `detail-flag-no-row` | A `detail` names a flag (`--long` or `-s`) that no argument row of the command declares (a warning): add the row. A command with no rows is reported too. Silence with `[metadata('justz-ignore', 'detail-flag-no-row')]` |
| `detail-env-no-row` | A `detail` names an environment variable that no `env` row of the command declares (a warning): add the row. A name counts when it has an underscore (`NO_COLOR`) or is written as `$NAME`, `${NAME}`, `export NAME` or `NAME=`; acronyms (`HTTP`, `JSON`), flags, paths and file names do not. A prose constant with an underscore (`UTF_8`, `SHA_256`, `READ_ONLY`) can fire, so silence it; a name under 3 characters (`$CI`) is not caught. Silence with `[metadata('justz-ignore', 'detail-env-no-row')]` |
| `example-*` | The example rules above, on group `examples` and on each command `example`; `example-flag-unknown` is always a warning |

Wire `justz check` and `justz fmt --check` into `check` and `ci`, then add a deliberately undocumented recipe and confirm both fail.

## `commands.json` schema quick reference

`justz schema` (0.6.0 or newer) prints the JSON Schema of this file. It checks types and nesting only, allows unknown fields and requires only `groups`; whether the registry matches the justfile stays with `justz check`. Save it with `justz schema > commands.schema.json` and point the editor at it, or add this line at the top of the registry for completion and type checks as you type:

```json
"$schema": "https://raw.githubusercontent.com/igorlino/justz/main/schema/commands.schema.json",
```

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
          "detail": "Optional longer text, shown only in the zoom.\n\nBlank lines make paragraphs.",   // 0.7.0 or newer
          "example": "… <arg>",      // optional; prefix with "… "
          "args": [
            { "name": "-s, --surface <name>", "desc": "Short first, value once at the end" },
            { "name": "-y", "desc": "Short-only flags stay as they are" },
            { "name": "--env <name>", "desc": "Target environment",
              "values": ["demo", "int", "production"] }   // drives enum completion and renders (a|b|c); one shell-safe word each
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

Smoke-test completion in a pseudo-terminal after changing it: unit tests that stub the shell's completion function cannot see wiring faults. Include an empty-word case (`tz git` followed by a space, as well as `tz git -`), because zsh hides option candidates until a `-` is typed.
