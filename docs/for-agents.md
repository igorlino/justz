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
| `help` lookups | Words that match no group or command exit `2`, with a message on stderr. Words that match several exit `2` and list them; with `-j` the list is a JSON array. |
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

## One call for every command

`justz help -j --expanded` adds `commands` to the overview: every command of the project, the core ones first.

| Field | Meaning |
|---|---|
| `path` | What a zoom takes and what `just` runs: `build`, or `tool clean`. |
| `extra` | `true` for an entry that is documented but does not run as `just <path>`. Left out otherwise. |
| `desc` | The one-line description. Empty when the recipe has none. |
| `hasDetail` | The zoom has a `detail`. |
| `tags` | What the command claims to touch. Left out when it declares none. |

Use it to find the commands tagged `destructive` or `production` before you choose one. The zoom has the rest: `detail`, `args`, `exits` and `env`.

## What the JSON cannot tell you

| Field | Limit |
|---|---|
| `invoke` | The command as the project writes it. Its first word is `cli` in the registry, and it can be a shim that is not installed. Run `just <path>`. |
| `args` | With a registry or a personal overlay, the arguments it documents. With neither, every parameter of the recipe, in its order: one with no help text has an empty `desc`. |
| `desc` | Empty when the recipe has no `[doc]`. An empty `desc` and no `tags` mean undocumented, not harmless. |
| `tags` | A claim by the author. `justz check` compares `asks-first` with the recipe's `[confirm]` and reports tags that contradict each other. It cannot verify the rest. |

## `assess --json`

`justz assess --json` has `recommendation`. Its `variant` is not a verdict: it holds only when `condition` is true, and `justz` cannot see whether it is.

| Field | Meaning |
|---|---|
| `variant` | The variant to use when the condition holds: `rich`. |
| `condition` | What that variant needs, in words. Ask the user; do not assume it. |
| `otherwise` | The variant to use when the condition does not hold: `native`, or `local-rich` in a project with a personal overlay. |
| `reason` | Why, in one sentence. |

What the variants are: [Variants](variants.md).

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
