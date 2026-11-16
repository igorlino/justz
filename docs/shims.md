# Shims

A shim is a short command, such as `tz`, that runs `justz` for one project's justfile from **any directory**. Collision checks make sure a shim never hides a real command on your system.

## Two kinds

| Kind | Example | Behavior |
|---|---|---|
| Local | `jz`, `jzh` | Find the justfile from the current directory. `jz <group> <cmd>` runs a command, `jzh` shows its help. Homebrew installs them, and `justz setup` adds them for every other install. |
| Pinned | `tz`, `tzh`, or any name | Wrap one specific project and work from any directory. Created with `justz shim add`. |

## The `jz` and `jzh` shortcuts

```sh
justz setup          # adds jz and jzh to the shim directory, if they are not installed yet
jzh                  # the help of the project around the current directory
jz clean             # with a registry, a bare name finds its group: this runs "just tool clean"
```

`justz setup` writes one small file per shortcut. It follows the rules of a shim:

- It never overwrites a file that `justz` did not write.
- It never takes a name that another command on the PATH, or a registered shim, already has. It says which one, and adds nothing.
- The two are added together. When `jz` is taken, `jzh` is not added either, and the reverse.
- A shortcut it cannot add is not an error: `justz setup` says why and goes on.
- With Homebrew the shortcuts are links next to the binary, so `justz setup` leaves them alone.

To remove a shortcut, delete its file. `justz setup --dry-run` shows what would be written, and `justz doctor` checks that each shortcut is the command your shell finds.

In a `native` project, with no registry, `jzh` renders the help from the justfile. `jz` with no command runs the default recipe, as `just` does: the listing, in a project from `justz init`. A bare name does not find its group there, so type the group: `jz tool clean`.

## Create a pinned shim

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
