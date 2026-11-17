# justz

**Make a justfile discoverable and keep its documentation honest.** `justz` is a single-binary CLI that sits next to [`just`](https://github.com/casey/just): it shows a project's commands as a grouped, searchable reference, and it fails CI when the docs and the recipes drift apart. You keep your justfile.

```text
$ jz

  demo

  Core
    build               # Compile the project
    deploy              # Deploy to an environment
    test                # Run the test suite

  Commands  (run without args to see sub-commands)
    tool                # Tools

  Run justz help --expanded for the full command reference.

$ jz clean
jz: tool clean
clean
```

`jz` runs a command, and `jzh <command>` shows its help. Commands can sit in groups: `jz clean` found `clean` in the group `tool`, and its first line says so. `jz` and `jzh` are short forms of `justz`: see [How you type it](#how-you-type-it).

And when the documentation drifts from the recipes, `justz check` says where, and what to paste:

```text
$ justz check
justz check: 1 error, 1 warning (4 public recipes checked, rich)

  error   just test: [doc] diverges from the registry description
      justfile: Run the test suite example: just test -w | Re-run on every save
      registry: Run the test suite  [registry-desc-diverges]
          paste: [doc('Run the test suite')]
          fix: make them the same: keep a one-line description in both, and move the rest to "detail" in the registry
  warning core.test: "just test -w" only passes the single boolean flag -w, which the argument rows already show  [example-single-flag]
          fix: remove it, or combine flags, values or environment
```

## Why

`just --list` is fine for a dozen recipes. Past that, two things get hard:

- **Discovery.** A new teammate, or an AI agent, cannot tell what the project can do, which flags a command takes, or which commands are safe to run.
- **Drift.** The help text, the docs and the recipes slowly disagree, and nobody notices until someone is misled.

`justz` fixes both. The help you read and the tab completion come from the same data, and `justz check` fails when that data and the recipes disagree.

## Quickstart

You need [`just`](https://github.com/casey/just) 1.37.0 or newer.

```sh
brew install igorlino/tap/justz     # more options: docs/installation.md
justz setup                         # check this machine; adds jz and jzh if your install did not

cd your-project
jzh                                 # what can this project do?
jzh <command>                       # one command: what it does and how to call it
jz <command>                        # run it
justz check                         # is it documented, and does the doc still match?
```

To document a project's commands, ask an agent that has the [`justfile-authoring` skill](#built-for-agents): it reads the recipes and writes the documentation. Without an agent, `justz init` scaffolds a new project, and you add the commands by hand. Where the documentation lives, and what that needs: [Variants](docs/variants.md). Full walkthrough: [Getting started](docs/getting-started.md).

## How you type it

You keep typing `just`. The short forms are there when you want them.

| You type | What it is | It needs |
|---|---|---|
| `just`, `just help tool clean`, `just check` | Your justfile's own recipes, when it defines them. | A justfile from `justz init` with its defaults, or your own default, `help` and `check` recipes |
| `jzh`, `jzh clean` | The help of the project around the current directory, from any subdirectory. It runs nothing. | `justz setup` (Homebrew installs it) |
| `jz build`, `jz tool clean` | Runs a command of that project. With no command, it runs the default recipe, like `just`. | `justz setup` (Homebrew installs it) |
| `jz clean`, for `tool clean` | A bare name finds its group. | A project with a [command registry](docs/variants.md) |
| `tz`, `tzh`, or a name you choose | A [shim](docs/shims.md) pinned to one project, from any directory. | `justz shim add` |
| `justz help`, `justz check` | The CLI itself. Works in any project with a justfile. | Nothing |

## What you get

| Need | Command |
|---|---|
| See what a project can do, grouped, with flags and examples | `justz help` · `justz help <group> <command>` |
| Fail CI when docs and recipes disagree | `justz check` ([pre-commit and CI snippets](docs/hooks-and-ci.md)) |
| Get advice on recipe antipatterns (long bodies, secrets on the command line) | `justz lint` |
| Start a self-documenting project | `justz init` |
| Keep the command registry in one stable order | `justz fmt` |
| Tab completion from the same data as the help | `justz completions` |
| Check the machine and add the `jz` and `jzh` shortcuts | `justz setup` |
| Run a project's commands from any directory, safely | `justz shim add` ([shims](docs/shims.md)) |
| Editor completion and type checks for the registry file, `commands.json` | `justz schema` |

Every command and flag: [Command reference](docs/commands.md).

## Built for agents

An agent should not have to guess. Every command has stable exit codes. The ones that report something print JSON with `--json`. What a project documents about a command is one call away: its description and details, exit codes, environment variables and safety tags (`destructive`, `production`, `read-only`...).

```sh
justz help tool clean -j | jq '{desc, detail, tags}'
```

Two agent skills come with it. `justz-cli` teaches an agent to use `justz` in any project. `justfile-authoring` teaches it to document a justfile for you: it reads the recipes and writes the documentation.

```sh
npx skills add igorlino/justz
```

Claude Code plugin, the JSON contract and the workflow: [For agents](docs/for-agents.md). Machine-readable index: [`llms.txt`](llms.txt).

## What you can rely on

- **One binary.** No `jq`, `awk`, Python or Node. It needs only `just`.
- **It keeps working as `just` evolves.** It reads a justfile only through the JSON that `just` prints, never by parsing the text.
- **It stays out of your secrets.** `justz` never reads, stores or injects credentials.
- **Text from a repository cannot drive your terminal.** Escape sequences and control characters in descriptions are replaced before they are printed.
- **Verified releases.** SHA256 checksums and build provenance attestations.

## Documentation

| Read this | When you want to |
|---|---|
| [Getting started](docs/getting-started.md) | go from zero to a documented project, step by step |
| [Installation](docs/installation.md) | install by Homebrew, script or Go, or verify a download |
| [Command reference](docs/commands.md) | look up a command, flag, JSON output or exit code |
| [Variants](docs/variants.md) | choose where the documentation lives: a registry, or the justfile alone |
| [The command registry](docs/registry.md) | write `commands.json`: `detail`, `args`, `exits`, `env`, `tags`, `seeAlso` |
| [Checks and lint rules](docs/checks.md) | understand a `check` or `lint` finding, or silence one |
| [Hooks and CI](docs/hooks-and-ci.md) | run `justz check` on commit and in GitHub Actions |
| [Shims](docs/shims.md) | pin a short command such as `tz` to a project |
| [For agents](docs/for-agents.md) | wire `justz` into an AI agent, or read the JSON contract |
| [Ecosystem](docs/ecosystem.md) | see how `justz` fits with `just-lsp`, `just-mcp` and others |
| [FAQ and troubleshooting](docs/faq.md) | fix `Unknown attribute`, a missing shim, or an empty help |

## Status

`justz` is an independent open source project, maintained by one person in their own time. No company backs it. It is in early development: see [Releases](https://github.com/igorlino/justz/releases).

JSON fields and exit codes are a contract: [For agents](docs/for-agents.md#the-contract). Bugs, ideas and how to take part: [CONTRIBUTING.md](CONTRIBUTING.md). Security reports: [SECURITY.md](SECURITY.md).

## Acknowledgements

`justz` builds on [`just`](https://github.com/casey/just). Thanks to [Snyk](https://snyk.io) for supporting open source projects.

## License

[MIT](LICENSE)
