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
  tags: "justz, just, cli, help, agents"
  version: "0.10.0"
---

# `justz` CLI Skill

`justz` shows what a project's [`just`](https://github.com/casey/just) recipes can do, as a grouped reference with JSON output, and checks that the documentation still matches the recipes.
It shells out to `just` only.
It never reads, stores or injects secrets.

## Quick start

```sh
command -v justz && justz --version     # 1. is it installed? If not: tell the user, stop
justz help -j                           # 2. what can this project do?
justz help <group> -j                   # 3. the commands of one group, each with hasDetail
justz help <group> <command> -j         # 4. one command: desc, detail, args, exits, env, tags
just <group> <command> ...              # 5. run it (or a shim: tz <group> <command>)
justz check --json                      # 6. after editing a justfile or registry
```

## Rules

- **Check the binary first.**
  If `justz` is missing, tell the user how to install it and stop.
  Never install it yourself.
  Never fall back to a hand-written help script.
  Install options and version requirements: [references/setup-and-shims.md](references/setup-and-shims.md).
- **Discover from the CLI, not the source.**
  Use `justz --help`, `justz <command> --help` and `justz help`.
  Do not read `commands.json` or the justfile only to learn flags.
- **Parse JSON, do not scrape text.**
  See the `--json` column below.
- **Read a command before you run it.**
  Read its zoom (`justz help <group> <command> -j`) and its tags.
  See [references/reading-commands.md](references/reading-commands.md).
- **Absence never means safe.**
  A command with no `tags`, `exits`, `env` or `detail` is unknown, not harmless.
- **`desc` plus `detail` is the text of a command.**
  Match on both, never on `desc` alone.
- **Run from the project.**
  Commands read the justfile of the current directory.
  A shim (`tz`, `jz`) runs for its own project from any directory.
- **Pass flags directly** to `just <group> <command>`.
  Do not insert `--`.
- **Never edit derived files.**
  `.just-local-rich.json` is written by `justz generate`; run it again.
  `justz` refuses to write it when git tracks it.
- **Do not silence a finding to make a run pass.**

## Commands

| Need | Command | `--json` |
| --- | --- | --- |
| What can this project do? | `justz help`, `justz help --expanded` | `-j` |
| One group, or one command | `justz help <group>`, `justz help [group] <command>` | `-j` |
| Is the documentation in sync? | `justz check` (`--strict`, `--rule <name>`) | yes |
| Advice on the recipes | `justz lint` (`--strict`, `--rule <name>`) | yes |
| Which documentation variant fits? | `justz assess` | yes |
| Which group does a bare name mean? | `justz resolve <name>` | yes |
| Start a self-documenting project | `justz init [dir]` (`--variant native\|rich`, `--name`, `--module`, `--dry-run`) | no |
| Keep the registry sorted | `justz fmt` (`--check`) | no |
| Personal overlay for a project with no registry | `justz generate` (`--stdout`) | no |
| JSON Schema of the registry | `justz schema` | no |
| Shell completion | `eval "$(justz completions zsh)"` or `bash` | no |
| Is the machine ready? | `justz doctor`, `justz setup` (`--dry-run`) | yes |
| Global shims | `justz shim add [path]`, `shim list`, `shim doctor`, `shim remove <name>` | yes |

`shim list` has no `--help` of its own: use `justz shim --help`.
`core` is not a group to zoom into: zoom into its commands with `justz help core <command>`.

## Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success. `check` with only warnings, and `lint`, exit `0` unless `--strict`. |
| `1` | A check failed, or an error. `init` exits `1` when a file is in the way. |
| `2` | Usage error. `resolve`: several matches. `help`: an ambiguous name (the groups are listed). |
| `3` | `resolve` only: no match. |

With `--json`, a failed check or no match still prints its JSON document.
An operational or usage error prints a plain message on stderr and no JSON.

## Tags to read before running a command

| Tag | Meaning |
| --- | --- |
| `destructive`, `production`, `spends` | Deletes something unrecoverable, touches a production system, costs money or quota. |
| `asks-first` | Stops for a confirmation. Do not answer it without the user. |
| `read-only`, `idempotent` | A claim that the command changes nothing, or is safe to repeat. |
| `long-running`, `needs-terminal` | How to run it. |

A tag you do not know is unknown, not an error.

## References

| Topic | File | Use when |
| --- | --- | --- |
| Reading a command | [references/reading-commands.md](references/reading-commands.md) | Before running a project command, or reading its text in a script or test |
| Fixing findings | [references/fixing-findings.md](references/fixing-findings.md) | `justz check` or `justz lint` reports something |
| Setup, shims, troubleshooting | [references/setup-and-shims.md](references/setup-and-shims.md) | Installing, version requirements, `init`, shims, completion, a symptom to diagnose |

The human documentation is at <https://github.com/igorlino/justz/tree/main/docs>.

## Related

`justfile-authoring` covers how to design the justfile and registry that `justz` renders and checks.
This skill only covers running the binary.
