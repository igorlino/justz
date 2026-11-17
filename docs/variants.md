# Variants

A project keeps the documentation of its commands in one of three places, called variants. Use `rich` when every machine that shows the help can install `justz`. Use `native` when that is not possible.

| Variant | The documentation lives in | It needs |
|---|---|---|
| `rich` | `config/commands.json`, the [command registry](registry.md), committed next to the justfile | `justz` on every machine that shows the help |
| `native` | the justfile: `[doc]`, `[group]` and `[arg]` attributes | only `just` |
| `local-rich` | `.just-local-rich.json`, a personal file that git ignores, generated from a `native` justfile | `justz` on your machine only |

## How to choose

Ask one question: can every teammate, and every CI job that shows the help, install `justz`?

| Answer | Use |
|---|---|
| Yes | `rich` |
| No | `native` |
| The team uses `native` and you want more on your own machine | `native`, with the `local-rich` overlay for yourself |

The size of the project does not decide. A safety tag such as `destructive` is as useful on four commands as on forty.

`justz assess` prints facts about a justfile, and ends with this recommendation on a `native` project:

```text
recommendation: rich if every machine that renders the help can install justz, otherwise native. a registry adds details, safety tags, exit codes, environment rows and bare-name lookup, which the justfile alone cannot hold
```

## What each variant gives

In a `native` project, a developer who installs `justz` gets most of it. What `native` cannot hold is the text and the tags that only the registry has a place for.

| | `native`, only `just` | `native`, with `justz` | `rich` |
|---|---|---|---|
| Grouped list of commands | `just --list` | `justz help` | `justz help` |
| Options of one command | `just --usage`, in a recent `just` | yes | yes |
| Examples | inside the `[doc]` text | yes | yes |
| Details, safety tags, exit codes, environment rows, see also | no | no | yes |
| Run a command by its bare name: `jz clean` for `tool clean` | no | no | yes |
| `justz check`, `justz lint`, tab completion, shims, JSON output | no | yes | yes |
| `justz fmt` | no | no | yes |
| Each description is written | once | once | twice: `desc` and `[doc]`, which `justz check` keeps equal |
| The `just` and `just help` that `justz init` writes work without `justz` | yes | yes | no |

The last two rows are the cost of `rich`. Its recipes call `justz`, so on a machine without it:

```text
$ just
_default: line 1: justz: command not found
error: recipe `_default` failed on line 8 with exit code 127
```

The safety tags matter most to an agent. In a `native` project, `justz help tool clean -j` has no `tags`, so nothing says that `clean` is destructive. How an agent reads them: [For agents](for-agents.md).

## How to get `rich`

| You have | Do this |
|---|---|
| An agent | Install the `justfile-authoring` skill and ask the agent to document the justfile. It reads the recipes and writes the registry, on a new project or an existing one. See [For agents](for-agents.md). |
| No agent, and no justfile | `justz init` writes a justfile and a registry that pass `justz check`. The registry lists only `help` and `check`; you add every other command by hand. |
| No agent, and a `native` justfile | Follow the steps below. |

### From `native` to `rich`, by hand

1. Make the default recipe run `justz help`, and add a `check` recipe that runs `justz check`. Run `justz init` in an empty directory to get a justfile that has both, and copy them.
2. Build the registry from the attributes the justfile has:

   ```sh
   mkdir -p config
   justz generate --stdout > config/commands.json
   ```

3. Remove the `$comment` line from the file: it says the file is generated, and from now on you edit it.
4. Run `justz fmt`, then `justz check`, and fix what it reports.
   - A `[doc]` with an `example:` line is reported: `check` prints the one-line `[doc]` to paste.
   - Top-level recipes land in the `core` group: move them in the registry if you want other groups.
5. Add what the justfile could not hold: `detail`, `tags`, `exits`, `env`. Each field: [The command registry](registry.md).

After this, a new recipe also needs an entry in the registry, or `justz check` fails.

## The `local-rich` overlay

`justz generate` writes `.just-local-rich.json` in the project root, from the justfile's `[doc]` and `[arg]` attributes. It is for one developer in a `native` project, and the team sees nothing of it.

- It holds what the attributes hold. It adds no details, no tags and no bare-name lookup.
- `values` lists that you add to an argument by hand are kept when you run `justz generate` again.
- `justz` refuses to write it when git tracks it. Add the file name to your global git ignore file, not to the project's.

When a registry exists, `justz` ignores the overlay. How `justz` finds each file: [Input model](input-model.md).
