# Getting started

In about ten minutes you go from nothing to a project whose commands are discoverable, checked on every commit and completed in your shell. Each step says what you should see.

## 1. Install and check the machine

```sh
brew install igorlino/tap/justz     # other ways: installation.md
justz setup
```

`justz setup` confirms that `just` is on your PATH and new enough (1.37.0 or newer), and adds the `jz` and `jzh` shortcuts if your install did not bring them. If it warns, see [FAQ](faq.md#unknown-attribute-errors).

## 2. Look at a project

In a directory that has a justfile:

```sh
justz help                    # a compact summary: core commands, then each group
justz help --expanded         # the full reference
justz help <group>            # one group
justz help <group> <command>  # one command: usage, examples, options, details
justz help --browse           # pick interactively (fzf if installed)
```

With the shortcuts, `jzh` is `justz help` and `jz <command>` runs a command. In a project from `justz init --variant rich`, plain `just` and `just help <group> <command>` show the same help, because its recipes call `justz`.

A group or command can be an exact name or a unique start of it: in a project with a `tool git` command, `justz help to gi` finds it.

## 3. Check the documentation

```sh
justz check
```

A project with no problems prints `ok: <n> public recipes checked (<variant>), no problems`. Otherwise each finding names the recipe, the rule and a fix. Errors exit `1`. Warnings fail the run only with `--strict`. What each rule means: [Checks and lint rules](checks.md).

## 4. Start a new project (optional)

```sh
justz init --variant native --module build --module test my-project
cd my-project && justz check
```

`init` writes a justfile, a shared `just/preamble.just` and one module per `--module`. It never overwrites a file. Use `--dry-run` to list the files first.

Pick a variant:

| Variant | Documentation lives in | Choose it when |
|---|---|---|
| `native` | the justfile: `[doc]`, `[group]`, `[arg]` attributes | you want only `just` and nothing else to maintain |
| `rich` | `config/commands.json`, next to the justfile | you want examples, details, exit codes, env rows and tags |
| `local-rich` | `.just-local-rich.json`, personal and git-ignored | the team did not adopt a registry and you want richer help for yourself |

Not sure? `justz assess` reports facts about your justfile and recommends one with its reason.

## 5. Run the check automatically

Add the pre-commit hook or the CI job from [Hooks and CI](hooks-and-ci.md), so drift fails a commit instead of surviving for months.

## 6. Add completion and a shortcut

```sh
eval "$(justz completions zsh)"   # in ~/.zshrc, after compinit (bash: ~/.bashrc)
justz shim add --name tz          # a short command pinned to this project, from any directory
```

See [Shims](shims.md).

## Next

- Write richer documentation: [The command registry](registry.md).
- Let an agent use all of this: [For agents](for-agents.md).
