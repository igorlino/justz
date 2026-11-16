# AGENTS.md

Instructions for coding agents that work in this repository.

## What this is

`justz` is a single-binary Go CLI that adds a help system, documentation checks, shims and shell completion on top of [`just`](https://github.com/casey/just). It also publishes two agent skills in `skills/`.

## Using `justz` (in any repository)

Read [docs/for-agents.md](docs/for-agents.md): the JSON contract, the exit codes and the workflow to follow before running a project's command.

## Working on this repository

| Task | Command |
|---|---|
| List the recipes | `just` |
| Build `./justz` | `just build` |
| Run the unit tests | `just test` |
| Everything CI runs | `just ci` |
| Format Go files | `just fmt` |

Run `just ci` before you say a change is done.

## Layout

| Path | Content |
|---|---|
| `cmd/justz/` | The CLI entry point and the `--help` text of every command |
| `internal/` | One package per feature: `help`, `check`, `lint`, `shim`, `registry`, `scaffold`... |
| `schema/commands.schema.json` | The JSON Schema of the registry |
| `skills/` | The `justz-cli` and `justfile-authoring` agent skills |
| `docs/` | User documentation. `docs/design/` holds design notes |
| `testdata/` | Fixtures. `testdata/variants/` has one project per documentation variant, and `testdata/site-demo/` is the project of the website demo |

## Rules

- **Documentation: read [docs/writing-docs.md](docs/writing-docs.md) before you change any.** Every fact has one page that owns it; its table says which. Add a row or a section, never a clause to an existing sentence. The README changes only when the pitch or the quickstart changes.
- A change to a command, flag, rule, JSON field or exit code also changes its `--help` text, the page that owns it, and `CHANGELOG.md`. `just test` fails when a command or rule is undocumented, or when the README or a `SKILL.md` grows past its budget.
- **Rule ids, JSON fields and exit codes are a contract.** Add fields; do not rename or remove them.
- **Do not copy real output by hand.** Run the binary on a fixture in `testdata/` and paste what it prints.
- `justz` never reads, stores or injects secrets, and never edits a user's justfile, PATH or shell startup files. Keep it that way.
- Commit messages follow Conventional Commits (`feat(check): ...`, `fix(doctor): ...`, `docs: ...`).
