# Command reference

Run `justz --help` for the list and `justz <command> --help` for the full text of a command (`doctor` and `shim list` have no help of their own; use `justz --help` and `justz shim --help`).

## Conventions

| | |
|---|---|
| Where it runs | Commands read the justfile of the current directory. A [shim](shims.md) runs for its own project from any directory. |
| JSON | Add `--json` for machine-readable output (`help` uses `-j`). Fields may be added; they are not renamed or removed. A consumer must ignore unknown fields. |
| Exit codes | `0` success, `1` a check failed or an error, `2` usage error. `resolve` adds `3` for no match. |
| Color | Only on a terminal, never when `NO_COLOR` is set. Width comes from `COLUMNS`, then the terminal, then 80. |
| Terminal safety | Escape sequences, control characters and bidirectional overrides in a description are shown as U+FFFD, in text and in JSON. |

## Discover

| Command | What it does |
|---|---|
| `justz help` | Compact summary: core commands, then each group. |
| `justz help --expanded` | Full reference. `--wide` / `-w` adds example columns (on by default from 100 columns). |
| `justz help <group>` | One group: usage, commands, examples. `core` is not a group to zoom into: its commands are in the summary, and zoom with `justz help core <command>` or the bare command name. |
| `justz help [group] <command>` | One command: how to invoke it, examples, options, **Details**, **Exit codes**, **Environment**, **Tags**, **See also**. |
| `justz help --browse` | Pick a group or command (fzf if installed, else a numbered menu). Without a terminal it prints the groups instead of prompting. |
| `justz help ... -j` | The same information as JSON. A group listing carries `hasDetail` per command. |
| `justz resolve <name>` | Print the group path a bare name stands for (`git` is `tool git`). Needs a registry. Exit `0` one match, `2` several, `3` none. |

## Check

| Command | What it does |
|---|---|
| `justz check` | Does the project document its commands, and do the docs still match? Errors exit `1`; `--strict` fails on warnings too; `--rule <name>` shows one rule (for reading, not for gating CI). Rules: [Checks](checks.md). |
| `justz lint` | Advice on the recipes themselves. Every finding is a warning. Same flags as `check`. |
| `justz assess` | Facts about the justfile (recipes, documented share, modules, long recipes) and the recommended [variant](variants.md), with the reason. |

## Create and maintain

| Command | What it does |
|---|---|
| `justz init [dir]` | Scaffold a self-documenting justfile. `--variant rich\|native` (default `rich`), `--name`, `--module` (repeatable), `--dry-run`. Never overwrites; refuses and exits `1` if any file exists. |
| `justz fmt [path]` | Sort the registry into its canonical order. `--check` writes nothing and exits `1` if unsorted. Looks in the current directory, or takes a path or `$JUSTZ_CONFIG`. |
| `justz generate [path]` | Build the personal `.just-local-rich.json` from the justfile's `[doc]` and `[arg]` attributes. `--stdout` writes nothing. Refuses to write a file that git tracks. |
| `justz schema` | Print the JSON Schema of `commands.json`. |

## Shell and machine

| Command | What it does |
|---|---|
| `justz completions <zsh\|bash>` | Print a completion script. Add `eval "$(justz completions zsh)"` to `~/.zshrc` (after `compinit`) or `~/.bashrc`; run it again after adding or removing a shim. |
| `justz doctor` | Is `just` installed and new enough, and are the `jz` shortcuts reachable on the PATH? |
| `justz setup` | Create the shim directory, add the `jz` and `jzh` shortcuts to it, check `just`, run the shim checks, and say how to put the shim directory on your PATH. A shortcut is not added when it is already installed or another command has the name. Never edits your PATH or shell startup files. `--dry-run` creates nothing. Safe to re-run. |
| `justz shim add [path]` | Register a shim. See [Shims](shims.md). |
| `justz shim list` / `remove <name>` / `doctor` | List, remove or verify shims. |

## Where the registry is found

`justz` walks up from the current directory to the project root (the first directory with a `.git` entry) and looks in each directory for `.*/config/commands.json`, then `config/commands.json`, then `just/commands.json`. There is no override for this search, except for `justz fmt`. Which variant applies and how the inputs combine: [input model](input-model.md).
