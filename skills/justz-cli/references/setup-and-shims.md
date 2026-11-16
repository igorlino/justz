# Setup, shims and troubleshooting

## Install `justz`

Tell the user; do not install it yourself.

| Method | Command |
|---|---|
| Homebrew | `brew install igorlino/tap/justz` |
| Go | `go install github.com/igorlino/justz/cmd/justz@latest` |
| Install script | `curl -fsSL https://raw.githubusercontent.com/igorlino/justz/main/install.sh \| sh` |
| Windows | the `.zip` from the release archives |

## Version requirements

| Needs | Version |
|---|---|
| `justz` at all | `just` 1.37.0 or newer. `justz doctor` warns when it is older. |
| `[metadata]`, used by the `justz-ignore` opt-out | `just` 1.42.0 |
| `init`, `lint`, `schema` | `justz` 0.6.0 |
| A command's `detail`, and terminal-safe output | `justz` 0.7.0 |
| `hasDetail`, `exits`, `env`, `tags`, a command's `seeAlso` | `justz` 0.8.0 |

`lint` needs `just`. `init` and `schema` do not.

## Check the machine

- `justz doctor --json`: is `just` installed and new enough?
- `justz setup` (`--dry-run`): create the shim directory, check `just`, run the shim checks. It never edits the PATH or shell startup files.

## Start a project

Do this only when the user asks for a new justfile.

1. `justz init --dry-run` lists the files. Show the list to the user.
2. `justz init` writes them. Add `--variant rich` for a `commands.json` registry.

It never overwrites: if a file is in the way it writes nothing and exits `1`. A bad name or option exits `2`. For an existing repository, use `justz assess` and the `justfile-authoring` skill.

## Choose a variant

`justz assess --json` reports facts about the justfile and recommends a variant.

- A public `default` recipe is a dispatcher, not a command. It is not counted, as in `check`. `check --strict` fails on it.
- `advice` lists hints, such as a registry whose default recipes do not run `justz help`. Shim names such as `tz` are not recognized there.

`justz generate` (`--stdout` writes nothing) builds the personal overlay for a project with no registry.

## Shims

`jz` and `jzh` are shortcuts for the project around the current directory: `jz <group> <cmd>` runs a command, `jzh` shows its help. They exist only when the binary is started under that name (Homebrew installs the symlinks).

Register a pinned shim:

```sh
justz shim add /path/to/project --soft
justz shim doctor
```

- The name comes from the registry's `cli` field, unless `--name` is given. With no registry, `--name` is required.
- `justz` never overwrites a file it did not write. A refusal means a foreign file is in the shim directory.
- `--replace` re-points a shim owned by another project. It cannot override a file `justz` did not write.
- `justz shim list --json` and `justz shim remove <name>` list and remove.

## Completion

`eval "$(justz completions zsh)"` or `bash`. Run it again after adding or removing a shim.

## Troubleshoot

| Symptom | Check |
|---|---|
| `just` not found | `justz doctor`; install `just` |
| `Unknown attribute` errors | `just` is older than 1.37.0; update it |
| Shim runs the wrong command or not at all | `justz shim doctor --json`: PATH order, stale stub, missing justfile |
| Shim stub reports a file justz did not write | Move or delete the file, then `justz shim add` again |
| Completion lists the wrong commands | The project is the one around the current directory (or the shim's justfile); run it from the right place |
| Descriptions are misaligned in a narrow terminal | Update `justz`; wrapping happens only when stdout is a terminal (piped output stays one line per command) |
