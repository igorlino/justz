# For agents

`justz` is designed so that an AI agent can discover what a repository offers and verify its own changes without guessing. This page is the contract.

## Install the skills

Two skills ship in this repository:

- `justz-cli`: how an agent runs and troubleshoots `justz`.
- `justfile-authoring`: how to design a discoverable, well-documented justfile, built on `justz`.

Claude Code:

```sh
/plugin marketplace add igorlino/justz
/plugin install justz --marketplace igorlino/justz
```

The one-step install needs Claude Code 2.1.275 or later; on older versions use `/plugin install justz@justz`. The skills then appear as `/justz:justz-cli`. A third-party marketplace does not auto-update by default: run `/plugin marketplace update justz`.

Any other agent (Codex, Copilot, Antigravity and others):

```sh
npx skills add igorlino/justz
```

The skills never install `justz`. They check for it and tell you how to install it.

## The contract

| Topic | Guarantee |
|---|---|
| Exit codes | `0` success, `1` a check failed or an error, `2` usage error, `3` (`resolve` only) no match. |
| JSON | `check`, `lint`, `assess`, `doctor`, `resolve`, `setup` and every `shim` command accept `--json`. `help` uses `-j`. `fmt`, `generate`, `completions`, `init` and `schema` do not. |
| JSON failures | A failed check or no match still prints its JSON document. An operational or usage error prints a plain message on stderr. |
| Evolution | Fields may be added. They are not renamed or removed. Ignore unknown fields. |
| Terminal safety | Text from a registry or justfile has control characters replaced by U+FFFD, in text and in JSON. The output of `just` and of recipes passes through unchanged. |
| Secrets | `justz` never reads, stores or injects credentials. |

Parse the JSON. Do not scrape the text tables, and do not read `commands.json` or `just --list` text to learn what a command is.

## Workflow an agent should follow

1. **Check the binary.** `command -v justz && justz --version`. If missing, tell the user how to install it ([Installation](installation.md)) and stop. Do not install it, and do not fall back to a hand-written help script.
2. **Discover.** `justz help -j` (it has `core`, the commands that run without a group, and `groups`), then `justz help <group> -j`, then `justz help <group> <command> -j`. `core` itself is not a group to zoom into; zoom into its commands with `justz help core <command> -j`.
3. **Read the zoom before running a command.** A group listing has `hasDetail` per command; when it is true, the zoom has a `detail` with facts the `desc` leaves out. The zoom also has `exits`, `env` and `tags` when the command documents them.
4. **Read the tags.**

   | Tag | Meaning |
   |---|---|
   | `destructive`, `production`, `spends` | deletes something unrecoverable, touches a production system, costs money or quota |
   | `asks-first` | stops for a confirmation: do not answer it without the user |
   | `read-only`, `idempotent` | claims that the command changes nothing, or is safe to repeat |
   | `long-running`, `needs-terminal` | how to run it |

   **Absence never means safe.** A command with no tags is unknown, not harmless. A tag you do not know is unknown, not an error.
5. **Run it** with `just <group> <command> ...` (or a shim such as `tz <group> <command>`).
6. **Verify your own changes.** After editing a justfile or registry: `justz check --json`, and `justz lint --json` for advice.

## `setup --json`

`justz setup --json` has `shortcuts`, one entry for each of `jz` and `jzh`: `name`, `path`, `action` and an optional `detail`.

| `action` | Meaning |
|---|---|
| `created` | The shortcut was written. With `--dry-run`, it would be. |
| `exists` | A shortcut from an earlier `justz setup` is there. |
| `installed` | A package, such as Homebrew, installed it next to the binary. |
| `skipped` | It was not added. `detail` says why: another command or a shim has the name, or the file cannot be written. |

A `skipped` shortcut does not change the exit code.

## Reading a command's text

`desc` plus `detail` is the text of a command. Match on both, never on `desc` alone:

```sh
justz help core test -j | jq '{desc, detail}'
```

Details: [The command registry](registry.md#read-a-commands-text-from-a-script-or-a-test).

## Derived files

`.just-local-rich.json` is written by `justz generate`. Never edit it by hand: run `justz generate` again. `justz` refuses to write it when git tracks it.

## Other agent tooling

`justz` does not run recipes for an agent or teach `just` syntax. For those, see the [ecosystem](ecosystem.md): `just-mcp` and the `just` syntax skills work alongside `justz`.
