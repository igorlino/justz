# FAQ and troubleshooting

## `Unknown attribute` errors

Your `just` is too old. `justz` needs 1.37.0 or newer (Ubuntu's package is 1.21.0). `justz doctor` warns about it and shows `just`'s own error. Install a current one: [Installation](installation.md#requirements).

## `justz check` reports `doc-missing` on a recipe that has `[doc]`

On `just` 1.33.0 to 1.36.0 the JSON dump leaves out the `doc` of a `[doc]` recipe. Upgrade to 1.37.0 or newer.

## `justz help` shows no details, examples or tags

Those live in the registry (`rich` variant). A `native` project documents commands only with `[doc]`, `[group]` and `[arg]`. Run `justz assess`: it advises when a registry exists but the `default` recipe does not run `justz help`. See [Variants](variants.md).

## `justz check` warns `default-public`

A public `default` recipe is a row in every listing. Rename it `_default` and keep it first in the file.

## `--rule` made my CI pass with errors

`--rule` is for reading: the counts and exit code follow the shown findings, so it can hide errors. Do not use it to gate CI.

## The commit hook fails with `Executable 'justz' not found`

Deliberate: the hook never passes silently when `justz` is missing. Install `justz`, or on a machine that has none set `JUSTZ_HOOK_SKIP_MISSING=1` to skip with a loud warning. See [Hooks and CI](hooks-and-ci.md).

## `justz generate` refuses to write

`.just-local-rich.json` is personal, so `justz` refuses to write it when git tracks it. Add it to your global git ignore file, not to the project's.

## `jz` or `jzh`: command not found

Run `justz setup`. It adds the two shortcuts to the shim directory and tells you how to put that directory on your PATH. If it says a shortcut was not added, another command already has the name; it names that command.

## My shim is not found

Run `justz setup` to see whether the shim directory is on your PATH, and `justz shim doctor` to check each shim.

## Does `justz` replace `just`?

No. You keep your justfile. `justz` adds a layer around it. See [Ecosystem](ecosystem.md).

## Does `justz` touch my secrets?

No. It never reads, stores or injects credentials. `justz lint` reads parameter names only, to warn about a secret typed on a command line.

## Where do I report a bug or a security problem?

Bugs and ideas: [issues](https://github.com/igorlino/justz/issues). Security: [SECURITY.md](../SECURITY.md); do not open a public issue.
