# Reading a project's commands

## Find and run a command

1. `justz help` shows the core commands and the groups.
2. `justz help <group>` shows the commands of one group.
3. `justz help [group] <command>` shows one command, examples first, then options and details.
4. Run it with `just <group> <command> ...`, or with the shim, for example `tz <group> <command>`. Pass flags directly; do not insert `--`.

A group or command can be an exact name or a unique prefix. An ambiguous name exits `2` and lists the groups.

## The zoom JSON

`justz help <group> <command> -j` is the supported way to read what a command is documented as, in a script, a test or an agent.

| Field | Content | How to use it |
|---|---|---|
| `desc` | One line: what the command does. Empty when the recipe has no `[doc]`. | |
| `invoke` | The command as the project writes it. Its first word can be a shim that is not installed. | Do not run it. Run `just <path>`. |
| `detail` | Facts the `desc` leaves out: what the command needs, what it changes. | Read it when the group listing says `hasDetail: true`. |
| `args` | Argument and flag rows. In a project with no registry and no overlay, every parameter is listed, with an empty `desc` for one that has no help text (0.10.0 or newer). | An empty array on 0.9.0 or older, or with an overlay, is not proof that the command takes nothing: check `just --usage <recipe>`. |
| `exits` | An object from exit code to meaning. | Know which codes are expected and what to do on each. |
| `env` | Rows of `name`, `desc`, and optionally `required` and `default`. | Set a `required` variable around the command. It is not a flag. |
| `tags` | An array of free strings. | Read it before running the command: `justz help <group> <command> -j \| jq .tags`. |
| `seeAlso` | Related commands as `group.command` keys. | |

The text zoom prints the same as **Details**, **Options**, **Exit codes**, **Environment**, a **Tags** line and a **See also** line. `detail` is not in the array printed for an ambiguous name.

`justz help <group> -j` lists the commands of a group. Each has `hasDetail`, so you know which zooms are worth a call.

`justz help -j --expanded` adds `commands` to the overview (0.10.0 or newer): every command of the project with its `path`, `desc`, `hasDetail` and `tags`. Use it to find what is tagged `destructive` or `production` in one call. An entry with `"extra": true` is documented but does not run as `just <path>`. When the field is absent, the `justz` is older: zoom into each group.

## `desc` plus `detail` is the text of a command

The long text of a command can move from `desc` to `detail` without any change to the command. Match on both, never on `desc` alone.

```sh
justz help core test -j | jq '{desc, detail}'
```

```json
{"desc": "Run the test suite", "detail": "Runs only the unit tests; ..."}
```

Reading `config/commands.json` or `just --list` text directly is not part of the contract.

## What the fields do not promise

- `exits`, `env`, `detail` and `tags` are only as current as their author.
- A command without them is not documented as safe, or as failing in any way.
- A command with no `tags`, or without `read-only`, is unknown, not harmless. Do not read a missing tag as a permission.
- `justz check` verifies `asks-first` against `[confirm]`, and contradictions between tags. It verifies nothing else about tags.

## Text is safe to read

The text `justz` prints from a project's registry or justfile has terminal control characters replaced by U+FFFD (0.7.0 or newer), in text and in `--json`. The output of `just` and of the recipes passes through unchanged.

- In `-j`, a `detail` shows `\n` where it held `\r\n`.
- A single-line field such as `desc` shows a space where it held a newline or tab.
- A U+FFFD in a description means the source held a control or invalid character, not that `justz` lost text.
