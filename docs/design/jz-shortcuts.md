# Design note: `justz setup` adds the `jz` and `jzh` shortcuts

Status: implemented in one PR (the stubs, `setup`, `doctor`, the tests and the documentation).

## Problem

`jz` and `jzh` are the short way to use `justz` in the project around the current directory: `jz <command>` runs a command and `jzh` shows its help. They work when the `justz` binary is started under one of those names. Only the Homebrew formula made that happen, with two links next to the binary. After the install script, `go install` or a release archive, the names did not exist, so the documentation could not show them without a caveat.

## Decision

`justz setup` writes the two names into the shim directory, the directory it already creates and that pinned shims use.

Each shortcut is a stub, like the stub of a pinned shim, not a link:

```sh
#!/bin/sh
# managed by justz (justz setup); remove it by deleting this file
exec justz --local jz "$@"
```

On Windows it is `jz.cmd` with `justz --local jz %*`. A stub finds `justz` through the PATH, so it survives an upgrade that moves the binary, and it works where a link to an `.exe` under another name does not. `--local` already existed: it is what `main` adds when the binary is started as `jz`.

A shortcut has no entry in the shim registry. The registry maps a name to one justfile, and a shortcut has none: it finds the justfile from the working directory.

## Rules

For each of `jz` and `jzh`, the first case that applies:

| Case | Action | JSON `action` |
|---|---|---|
| A registered shim uses the name, as its command or its help command | Nothing. The shim keeps the name. | `skipped` |
| The stub of `justz setup` is already there | Nothing. | `exists` |
| Another file is at the stub path | Nothing. `justz` never overwrites a file it did not write. | `skipped` |
| The name exists next to the `justz` binary (Homebrew) | Nothing. | `installed` |
| A command with the name is anywhere on the PATH, before or after the shim directory | Nothing. A shortcut never hides a command, and is never written where it could not run. | `skipped` |
| Otherwise | Write the stub. With `--dry-run`, only say so. | `created` |

The two names are then looked at together. If one is skipped and the other would be created, the other is skipped too: `jzh` must not show the help of `justz` while `jz` is another tool. A stub that cannot be written (a read-only shim directory) is also reported as `skipped`, with the error, and `setup` goes on to its checks.

A skipped shortcut is not a failure: `setup` still exits `0`, and the text and the `detail` field name what has the name. There is no `--force` and no `--replace`. A user who wants the name for something else has it; a user who wants the shortcut removes the other command and runs `justz setup` again.

## What else changes

- `setup --json` gains `shortcuts`, an array of `name`, `path`, `action` and an optional `detail`. A field is added; none is renamed.
- `justz doctor` checked the names next to the binary. It now also checks a stub of `justz setup`: is it the command a shell finds first? A stub of a registered shim called `jz` is not reported there; `shim doctor` covers it.
  - The shim directory is not on the PATH: a warning (`[warn]`, and `warn` in the JSON), not a failure. `setup` has just printed how to add it, and the two commands must not disagree.
  - A stub and a packaged link are both present (an install script install that moved to Homebrew): fine, whichever comes first.
  - The binary is itself in the shim directory (the install script puts it in `~/.local/bin`): a file next to it is then not "packaged", so a foreign `jz` there is not reported as ours.
- `justz shim add --name jz` still works and replaces the stub, because the stub is one that `justz` wrote and the user asked for the name.
- Removal is deleting the file. `justz shim remove jz` is for registered shims, and reports that there is none.

## Not done

- `justz doctor` does not check the pairing later. If another tool takes `jz` after `setup` wrote both, `doctor` reports that `jz` is hidden, but when only a `jzh` stub is left it says nothing about `jz`.
- `justz shim add --name jz --suffix x` pins `jz` to one project and leaves the `jzh` shortcut in place, and `shim add --name jzh` replaces the `jzh` shortcut without a word. Both are deliberate uses of the names; a warning from `shim add` would be the fix.

- `justz setup` does not remove a shortcut. A flag for that is cheap to add if someone asks.
- The Windows `.zip` and winget do not install the names; on Windows `justz setup` is the way.
