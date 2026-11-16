# Shims

A shim is a short command, such as `tz`, that runs `justz` for one project's justfile from **any directory**. Collision checks make sure a shim never hides a real command on your system.

## Two kinds

| Kind | Example | Behavior |
|---|---|---|
| Local | `jz`, `jzh` | Find the justfile from the current directory. `jz <group> <cmd>` runs a command, `jzh` shows its help. They exist only when the binary is started under that name; Homebrew installs the symlinks. |
| Pinned | `tz`, `tzh`, or any name | Wrap one specific project and work from any directory. Created with `justz shim add`. |

## Create one

```sh
justz setup                        # create the shim directory and check the machine
justz shim add ~/projects/toolz    # the name comes from "cli" in its commands.json
justz shim add --name tz           # or choose the name (required when there is no registry)
```

`shim add` also creates a help command named `<name><suffix>` (`tzh` by default).

| Flag | Meaning |
|---|---|
| `--name NAME` | Use this shim name. |
| `--suffix S` | Help command suffix (default `h`). |
| `--soft` | Fall back to plain `just` when the help system is missing. |
| `--replace` | Re-point a shim that another project owns, or hide a command that comes later on the PATH. |
| `--force` | Allow a shim that an earlier command on the PATH would shadow. |
| `--dry-run` | Show what would happen, write nothing. |

`justz shim add` never overwrites a file in the shim directory that it did not write.

## Maintain

```sh
justz shim list
justz shim doctor       # check every registered shim
justz shim remove tz
```

Run `justz completions zsh` (or `bash`) again after adding or removing a shim, so the new names are completed. `justz` never edits your PATH or shell startup files; `justz setup` tells you how to put the shim directory on your PATH.
