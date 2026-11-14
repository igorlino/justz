---
name: justz-cli
description: >
  Teaches coding agents how to run and troubleshoot the `justz` CLI (and its `jz`, `jzh` and
  project shims such as `tz`) on top of `just`. Use when running, suggesting or fixing any `justz`
  command: showing a project's commands with `justz help`, checking justfile and command-registry
  drift with `justz check`, sorting `commands.json` with `justz fmt`, resolving a bare command name
  with `justz resolve`, registering or repairing shims with `justz shim`, shell completion with
  `justz completions`, or preparing a machine with `justz setup` and `justz doctor`. Do not use for
  writing or restructuring a justfile (use `justfile-authoring`) or for running a project's own
  recipes by hand.
metadata:
  author: igorlino
  domain: tooling
  version: "0.1.0"
---

# `justz` CLI Skill

`justz` adds a data-driven help system, checks, shims and shell completion on top of [`just`](https://github.com/casey/just). It shells out to `just` only. It never reads, stores or injects secrets.

## Rules

- **Check the binary first.** `command -v justz && justz --version`. If it is missing, tell the user how to install it and stop; never install it yourself and never fall back to a hand-written help script. Install options: `brew install igorlino/tap/justz`, `go install github.com/igorlino/justz/cmd/justz@latest`, the install script, or the release archives (Windows: the release `.zip`).
- **Discover from the CLI, not the source.** `justz --help`, `justz <command> --help`, and `justz help` for the project's own commands. `doctor` and `shim list` have no `--help` of their own; use `justz --help` and `justz shim --help`. Do not read `commands.json` or the justfile only to learn flags.
- **Use `--json` in agents.** `doctor`, `check`, `assess`, `resolve`, `setup` and every `shim` command accept `--json` (`help` uses `-j`). `fmt`, `generate` and `completions` do not. Parse the JSON; do not scrape the text tables.
- **Read the exit code.** `0` success, `1` a check failed or an error, `2` usage error. `resolve` adds `3` (no match). With `--json`, a failed check or no match still prints its JSON document; an operational or usage error prints a plain message on stderr and no JSON. `check` fails only on errors, so warnings (common) exit `0` unless `--strict`.
- **Run from the project.** Commands read the justfile of the current directory (or `JUSTZ_CONFIG` for the registry). A shim (`tz`, `jz`) runs for its own project from any directory.
- **Never edit derived files.** `.just-local-rich.json` is written by `justz generate`; re-run it. `justz` refuses to write it when git tracks it.

## Commands

| Need | Command |
|---|---|
| What can this project do? | `justz help` (summary), `justz help --expanded`, `justz help <group>`, `justz help [group] <command>`, add `-j` for JSON |
| Focused docs for one command, examples first | `justz help [group] <command>` (exact name or unique prefix; an ambiguous name exits `2` and lists the groups) |
| Is the documentation in sync? | `justz check` (`--json`, `--strict` makes warnings fail) |
| Keep the registry sorted | `justz fmt` (`--check` writes nothing and exits `1` when unsorted) |
| Which group does a bare name mean? | `justz resolve <name>` (exit `0` one match, `2` several, `3` none) |
| Which documentation variant fits? | `justz assess --json` |
| Personal overlay for a project without a registry | `justz generate` (`--stdout` writes nothing) |
| Shell completion | `eval "$(justz completions zsh)"` or `bash`; run it again after adding or removing a shim |
| Is the machine ready? | `justz setup` (`--dry-run`), `justz doctor --json` |
| Global shims | `justz shim add [path]`, `justz shim list --json`, `justz shim doctor --json`, `justz shim remove <name>` |

`jz` and `jzh` are shortcuts for the project around the current directory: `jz <group> <cmd>` runs a command, `jzh` shows its help. They exist only when the binary is started under that name (Homebrew installs the symlinks).

## Common tasks

**Find and run a project command.** `justz help` to see groups, `justz help <group>` for its commands, `justz help <group> <cmd>` for flags and examples, then run `just <group> <cmd> ...` (or the shim, for example `tz <group> <cmd>`). Pass flags directly; do not insert `--`.

**Fix documentation drift.** `justz check --json`. Each finding has `rule`, `severity`, `subject`, `message` and `fix`. Apply the `fix`, re-run until it exits `0`. `registry-unsorted` is fixed with `justz fmt`; `registry-desc-diverges` means the justfile `[doc]` and the registry `desc` must be the same text.

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
