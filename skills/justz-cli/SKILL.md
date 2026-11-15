---
name: justz-cli
description: >
  Teaches coding agents how to run and troubleshoot the `justz` CLI (and its `jz`, `jzh` and
  project shims such as `tz`) on top of `just`. Use when running, suggesting or fixing any `justz`
  command: showing a project's commands with `justz help`, checking justfile and command-registry
  drift with `justz check`, advice on recipes with `justz lint`, scaffolding with `justz init`,
  printing the registry's JSON Schema with `justz schema`, sorting `commands.json` with `justz fmt`, resolving a bare command name
  with `justz resolve`, registering or repairing shims with `justz shim`, shell completion with
  `justz completions`, or preparing a machine with `justz setup` and `justz doctor`. Do not use for
  writing or restructuring a justfile (use `justfile-authoring`) or for running a project's own
  recipes by hand.
metadata:
  author: igorlino
  domain: tooling
  version: "0.4.0"
---

# `justz` CLI Skill

`justz` adds a data-driven help system, checks, shims and shell completion on top of [`just`](https://github.com/casey/just). It shells out to `just` only. It never reads, stores or injects secrets.

## Rules

- **Check the binary first.** `command -v justz && justz --version`. If `justz` is missing, tell the user how to install it and stop; never install it yourself and never fall back to a hand-written help script. Install options: `brew install igorlino/tap/justz`, `go install github.com/igorlino/justz/cmd/justz@latest`, the install script, or the release archives (Windows: the release `.zip`). `init`, `lint` and `schema` need 0.6.0 or newer; a command's `detail` text and the terminal-safe output need 0.7.0 or newer; `hasDetail`, `exits` and a command's `seeAlso` need the release after 0.7.0.
- **Discover from the CLI, not the source.** `justz --help`, `justz <command> --help`, and `justz help` for the project's own commands. `doctor` and `shim list` have no `--help` of their own; use `justz --help` and `justz shim --help`. Do not read `commands.json` or the justfile only to learn flags.
- **Use `--json` in agents.** `doctor`, `check`, `lint`, `assess`, `resolve`, `setup` and every `shim` command accept `--json` (`help` uses `-j`). `fmt`, `generate`, `completions`, `init` and `schema` do not. Parse the JSON; do not scrape the text tables.
- **Read a command's text with `justz help <group> <command> -j`.** Its JSON is the supported way to read what a command is documented as, in a script, a test or an agent. `desc` plus `detail` is the text of a command: the long text of a command can move from `desc` to `detail`, so match on both, never on `desc` alone. Reading `config/commands.json` (or `just --list` text) directly is not part of the contract. Example: `justz help core test -j | jq '{desc, detail}'` prints `{"desc": "Run the test suite", "detail": "Runs only the unit tests; ..."}`.
- **Before running a command, read its zoom when it has one.** `justz help <group> -j` lists a group's commands, each with `hasDetail`: when it is true, `justz help <group> <command> -j` has a `detail` with facts the `desc` leaves out (what it needs, what it changes). The same zoom has `exits`, an object from exit code to meaning, when the command documents its codes: read it to know which codes are expected and what to do on each. `exits` and `detail` are only as current as their author, so a command without them is not documented as safe or as failing in any way.
- **Read the exit code.** `0` success, `1` a check failed or an error, `2` usage error. `resolve` adds `3` (no match). With `--json`, a failed check or no match still prints its JSON document; an operational or usage error prints a plain message on stderr and no JSON. `check` fails only on errors, so warnings (common) exit `0` unless `--strict`. `lint` reports only warnings, so it exits `0` unless `--strict`. `init` exits `1` when it refuses to write because a file is in the way, and `2` for a bad name or option.
- **Run from the project.** Commands read the justfile of the current directory (or `JUSTZ_CONFIG` for the registry). A shim (`tz`, `jz`) runs for its own project from any directory.
- **Never edit derived files.** `.just-local-rich.json` is written by `justz generate`; re-run it. `justz` refuses to write it when git tracks it.

## Commands

| Need | Command |
|---|---|
| What can this project do? | `justz help` (summary), `justz help --expanded`, `justz help <group>`, `justz help [group] <command>`, add `-j` for JSON |
| Focused docs for one command, examples first | `justz help [group] <command>` (exact name or unique prefix; an ambiguous name exits `2` and lists the groups). After the options it prints the command's longer **Details**, if the project wrote any; `-j` carries it as `detail` (not in the array printed for an ambiguous name); its **See also** line and `seeAlso` JSON field list related commands as `group.command` keys |
| Run the check before commits and in CI | the snippets in `docs/hooks-and-ci.md` of the `justz` repository: a pre-commit hook (`justz-check`), a plain git hook that fails loudly without `justz` (`JUSTZ_HOOK_SKIP_MISSING=1` skips), a GitHub Actions job |
| Is the documentation in sync? | `justz check` (`--json`, `--strict` makes warnings fail, `--rule <name>` shows one rule; counts and the exit code follow the shown findings, so it is for reading, not for gating CI: it can hide errors and exit `0`; an unknown name exits `2` and lists the valid ones) |
| Advice on the recipes (long bodies, secrets, a module with no shell) | `justz lint` (`--json`, `--strict` makes warnings fail, `--rule <name>` as in `check`) |
| Start a project with a self-documenting justfile | `justz init [dir]` (`--variant native\|rich`, `--name`, `--module` repeated, `--dry-run` writes nothing) |
| Schema of the registry, for an editor or CI | `justz schema` (prints JSON; `justz schema > commands.schema.json`) |
| Keep the registry sorted | `justz fmt` (`--check` writes nothing and exits `1` when unsorted) |
| Which group does a bare name mean? | `justz resolve <name>` (exit `0` one match, `2` several, `3` none) |
| Which documentation variant fits? | `justz assess --json` (a public `default` recipe is a dispatcher, not counted, as in `check`; `advice` lists hints such as a registry whose default recipes do not run `justz help`; shim names such as `tz` are not recognized; `check --strict` fails on a public `default`) |
| Personal overlay for a project without a registry | `justz generate` (`--stdout` writes nothing) |
| Shell completion | `eval "$(justz completions zsh)"` or `bash`; run it again after adding or removing a shim |
| Is the machine ready? | `justz setup` (`--dry-run`), `justz doctor --json` |
| Global shims | `justz shim add [path]`, `justz shim list --json`, `justz shim doctor --json`, `justz shim remove <name>` |

`jz` and `jzh` are shortcuts for the project around the current directory: `jz <group> <cmd>` runs a command, `jzh` shows its help. They exist only when the binary is started under that name (Homebrew installs the symlinks).

## Common tasks

**Find and run a project command.** `justz help` to see groups, `justz help <group>` for its commands, `justz help <group> <cmd>` for flags and examples, then run `just <group> <cmd> ...` (or the shim, for example `tz <group> <cmd>`). Pass flags directly; do not insert `--`.

**Text is safe to read.** The text `justz` itself prints from a project's registry or justfile has terminal control characters replaced by U+FFFD (0.7.0 or newer), in text and in `--json`. The output of `just` and of the recipes passes through unchanged. In `-j`, a `detail` shows `\n` where it held `\r\n`, and a single-line field such as `desc` shows a space where it held a newline or tab. A U+FFFD in a description means the source held a control or invalid character, not that `justz` lost text.

**Fix documentation drift.** `justz check --json`. Each finding has `rule`, `severity`, `subject`, `message` and `fix`. Apply the `fix`, re-run until it exits `0`. A `registry-desc-diverges` finding also has a `paste` field: the exact `[doc(...)]` line for the justfile, to paste yourself (`justz` never edits it). The text output prints a fix once for a run of findings of one rule; the JSON keeps it on every finding. Work through one rule at a time with `--rule <name>`. `registry-unsorted` is fixed with `justz fmt`; `registry-desc-diverges` means the justfile `[doc]` and the registry `desc` must be the same text; `registry-desc-long` is a warning, so it fails a run only with `--strict`: shorten `desc` to a one-line summary equal to the `[doc]` and move the rest to `detail`; with the generated overlay, shorten the `[doc]` and run `justz generate`.

**Act on lint findings.** `justz lint --json`. Each finding has `rule`, `severity`, `subject`, `message`, `fix` and sometimes `line`, which counts lines of the recipe body as `just --dump` shows them, not lines of the file. Fix the cause: a `recipe-long` finding counts the `{{ }}` interpolations (JSON: `interpolations`, `quotedInterpolations`, `escapedBraces`), and those in single quotes or a quoted here-document are not mechanical to move; move a long body to `scripts/<group>/<name>.sh`, read a secret inside the recipe instead of taking it as a parameter or interpolating it, give a module a `set shell` or import the shared preamble, give a module a default recipe that runs `justz help <module>` (`module-dispatcher-missing`, `module-dispatcher-lists-root`). Silence one recipe only when the finding is wrong, with `[metadata('justz-ignore', '<rule>')]` (the rules `recipe-long`, `secret-parameter` and `secret-interpolated`, and the `check` advice rules `volatile-text`, `detail-flag-no-row`, `detail-long`, `detail-no-paragraphs` and `detail-restates-desc`; `[metadata]` is verified on `just` 1.59.0 only); a typo in the rule name is itself a finding. `module-shell-unset` and the two dispatcher rules have no per-recipe opt-out. Do not turn a finding off to make the run pass.

**Start a project.** `justz init --dry-run` lists the files, then `justz init` writes them. It never overwrites: if a file is in the way it writes nothing and exits `1`. It does not need `just`. Add `--variant rich` for a `commands.json` registry. `lint` needs `just`, `init` and `schema` do not. Do this only when the user asks for a new justfile, after showing the `--dry-run` list; for an existing repo use `justz assess` and the `justfile-authoring` skill.

**Register a shim.** `justz shim add /path/to/project --soft`. The name comes from the registry's `cli` field unless `--name` is given. `justz` never overwrites a file it did not write: a refusal means a foreign file is in the shim directory. `--replace` re-points a shim owned by another project, but cannot override a file justz did not write. Then `justz shim doctor`.

**Troubleshoot.**

| Symptom | Check |
|---|---|
| `just` not found | `justz doctor`; install `just` |
| Shim runs the wrong command or not at all | `justz shim doctor --json`: PATH order, stale stub, missing justfile |
| Shim stub reports a file justz did not write | Move or delete the file, then `justz shim add` again |
| Completion lists the wrong commands | The project is the one around the current directory (or the shim's justfile); run it from the right place |
| Descriptions are misaligned in a narrow terminal | Update `justz`; wrapping happens only when stdout is a terminal (piped output stays one line per command) |

## Related

`justfile-authoring` covers how to design the justfile and registry that `justz` renders and checks. This skill only covers running the binary.
