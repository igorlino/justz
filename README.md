# justz

**Make a justfile discoverable and keep its documentation honest.** `justz` is a single-binary CLI that sits next to [`just`](https://github.com/casey/just): it shows a project's commands as a grouped, searchable reference, and it fails CI when the docs and the recipes drift apart. You keep your justfile.

```text
$ justz help

  demo

  Core
    build               # Compile the project
    deploy              # Deploy to an environment
    test                # Run the test suite

  Commands  (run without args to see sub-commands)
    tool                # Tools

  Run justz help --expanded for the full command reference.
```

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

`justz` fixes both with one source of truth: the help, the completion and the checks all read the same data, so they cannot disagree.

## Quickstart

You need [`just`](https://github.com/casey/just) 1.37.0 or newer.

```sh
brew install igorlino/tap/justz     # more options: docs/installation.md
justz doctor                        # is this machine ready?

cd your-project
justz help                          # what can this project do?
justz check                         # is it documented, and does the doc still match?
```

No justfile yet? `justz init` writes one that passes `justz check` on the first run. Full walkthrough: [Getting started](docs/getting-started.md).

## What you get

| Need | Command |
|---|---|
| See what a project can do, grouped, with flags and examples | `justz help` · `justz help <group> <command>` |
| Fail CI when docs and recipes disagree | `justz check` ([pre-commit and CI snippets](docs/hooks-and-ci.md)) |
| Get advice on recipe antipatterns (long bodies, secrets on the command line) | `justz lint` |
| Start a self-documenting project | `justz init` |
| Keep the command registry in one stable order | `justz fmt` |
| Tab completion from the same data as the help | `justz completions` |
| Run a project's commands from any directory, safely | `justz shim add` ([shims](docs/shims.md)) |
| Editor completion and type checks for `commands.json` | `justz schema` |

Every command and flag: [Command reference](docs/commands.md).

## Built for agents

An agent should not have to guess. Every command has stable exit codes. The ones that report something print JSON with `--json`. A command's text, exit codes, environment variables and safety tags (`destructive`, `production`, `read-only`...) are one call away:

```sh
justz help core test -j | jq '{desc, detail, tags}'
```

Two agent skills, `justz-cli` and `justfile-authoring`, teach an agent how to use `justz` and how to write a justfile it can discover:

```sh
npx skills add igorlino/justz
```

Claude Code plugin, the JSON contract and the workflow: [For agents](docs/for-agents.md). Machine-readable index: [`llms.txt`](llms.txt).

## Design principles

- **One binary.** No `jq`, `awk`, Python or Node. It needs only `just`.
- **It reads `just` the supported way**, through `just --dump --dump-format json`, so it keeps working as `just` evolves.
- **It stays out of your secrets.** `justz` never reads, stores or injects credentials.
- **Text from a repository cannot drive your terminal.** Escape sequences and control characters in descriptions are replaced before they are printed.
- **Verified releases.** SHA256 checksums and build provenance attestations.

## Documentation

| Read this | When you want to |
|---|---|
| [Getting started](docs/getting-started.md) | go from zero to a documented project, step by step |
| [Installation](docs/installation.md) | install by Homebrew, script or Go, or verify a download |
| [Command reference](docs/commands.md) | look up a command, flag, JSON output or exit code |
| [The command registry](docs/registry.md) | write `commands.json`: `detail`, `args`, `exits`, `env`, `tags`, `seeAlso` |
| [Checks and lint rules](docs/checks.md) | understand a `check` or `lint` finding, or silence one |
| [Hooks and CI](docs/hooks-and-ci.md) | run `justz check` on commit and in GitHub Actions |
| [Shims](docs/shims.md) | pin a short command such as `tz` to a project |
| [For agents](docs/for-agents.md) | wire `justz` into an AI agent, or read the JSON contract |
| [Ecosystem](docs/ecosystem.md) | see how `justz` fits with `just-lsp`, `just-mcp` and others |
| [FAQ and troubleshooting](docs/faq.md) | fix `Unknown attribute`, a missing shim, or an empty help |

## Status

Early development (current release: see [Releases](https://github.com/igorlino/justz/releases)). The command surface and the JSON contract are described in the docs above; fields may be added, never renamed or removed. Not open for code contributions yet; ideas and bug reports are welcome as [issues](https://github.com/igorlino/justz/issues). Security reports: [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE)
