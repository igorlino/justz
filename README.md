# justz

> Just addons for a superior agent and developer experience

[`just`](https://github.com/casey/just) is a great task runner. Past a dozen recipes, though, two things get hard: new people (and AI agents) can't tell what the project can do, and the docs, the help text and the recipes slowly drift apart.

`justz` is a small, single-binary CLI that fixes both, without replacing `just`. You keep your justfile. `justz` adds the layer around it.

> **Status: early development.** Available today: `justz help` (summary, `--expanded`, zoom into a group or command, `--json`, `--browse`), `justz check`, `justz completions` (zsh and bash), `justz fmt`, the shim system (`justz setup`, `justz shim add`, `list`, `remove` and `doctor`, plus the `jz` and `jzh` shortcuts), `justz resolve`, `justz generate`, `justz assess`, `justz doctor`, `justz init`, `justz lint` and `justz schema`. Everything in the list below is available.

## What you get

- **Help people actually read** *(available)*. A grouped, searchable command reference with examples, rendered from your project's own data. Zoom into a group or a single command, or print it as JSON for tools.
- **Long descriptions where they belong** *(available)*. A command's `detail` field holds the longer text: the few facts that do not fit in `desc`, not steps (the [`justfile-authoring` skill](skills/justfile-authoring/references/variant-rich.md) says what goes in each). The zoom prints it under a **Details** heading and, with colors on, picks out `` `command spans` ``, flags, `<placeholders>` and URLs; listings and completion stay one line each. A command's `exits` maps each exit code to its meaning (an **Exit codes** section in the zoom and in `help -j`), and the group listing of `help -j` has `hasDetail` per command, so an agent knows which zooms are worth a call. A command's `seeAlso` lists related commands by key (`group.command`), which the zoom shows as a **See also** line, `help -j` carries, and `justz check` verifies.
- **Text that cannot drive your terminal** *(available)*. A description with an escape sequence, a control character or a bidirectional override is shown with U+FFFD in its place, in text and in `--json`, so a cloned repository cannot change your window title or rewrite what you read.
- **Checks that keep docs honest** *(available)*. `justz check` fails when a recipe has no documentation, when the registry and the justfile disagree, or when an example no longer parses (a `[doc]` that diverges from the registry comes with the exact line to paste). Advice rules (warnings, never errors) catch text that goes stale (`volatile-text`: a TODO, a count of checks, a price, a date), a flag that a `detail` names without an argument row (`detail-flag-no-row`), and a `detail` that is too long, has no paragraphs or opens by repeating the `desc` (`detail-long`, `detail-no-paragraphs`, `detail-restates-desc`; a deliberately list-style `detail` with no blank lines opts out, and as warnings they fail `check --strict`); one recipe can opt out with `[metadata('justz-ignore', '<rule>')]`. The text output of `check` and `lint` prints a fix once per rule, and `--rule <name>` shows one rule at a time (the counts and the exit code follow the shown findings, so it is for reading, not for gating CI: it can hide errors and exit 0). Run it in pre-commit and CI instead of discovering drift months later.
- **Canonical formatting** *(available)*. `justz fmt` sorts your command registry into one stable order, so diffs stay small and reviews stay calm. Use `justz fmt --check` in CI to fail on an unformatted registry.
- **A scaffold that documents itself** *(available)*. `justz init` writes a justfile, a shared `just/preamble.just` and one module per group (and, for the rich variant, a sorted `config/commands.json`) that pass `justz check` on the first run. It never overwrites a file.
- **Advice on the recipes themselves** *(available)*. `justz lint` points at antipatterns you can see in the structure of a justfile: a recipe body over 40 lines (with a count of its `{{ }}` interpolations, which decides whether moving it to a script is mechanical), a secret-like parameter, a secret interpolated into a command line, a module that runs without a chosen shell, a module whose bare `just <module>` runs a command instead of a dispatcher, and a dispatcher that lists the root justfile. Every finding is a warning, and one recipe can opt out with `[metadata('justz-ignore', '<rule>')]`.
- **Shortcuts that never break** *(available)*. Pin a command such as `tz` to a project's justfile and run it from any directory, with collision checks so it never hides a real command on your system.
- **Completion for free** *(available)*. Shell completion generated from the same data as the help.
- **An editor-ready schema** *(available)*. `justz schema` prints the JSON Schema of `commands.json`, so an editor can complete and type-check the registry.
- **Built for agents.** Every command has stable exit codes, and the ones that report something (`check`, `lint`, `assess`, `doctor`, `resolve`, `setup`, `shim`) print JSON with `--json`, so an AI agent can discover what a repo offers and verify its own changes without guessing. Two agent skills, `justz-cli` and `justfile-authoring`, teach agents how.

## Why another tool?

- **One binary, almost no dependencies.** No `jq`, `awk`, Python or Node to install. It only needs `just`.
- **One source of truth.** The help, the checks and the completion all read the same data, so they cannot disagree.
- **It reads `just` the supported way**, through `just --dump --dump-format json`, so it keeps working as `just` evolves.
- **It stays out of your secrets.** `justz` never reads, stores or injects credentials.
- **Verified releases.** Releases ship with checksums and build provenance attestations.

## Part of the `just` ecosystem

`justz` exists because `just` is good. It does not replace `just`, and it works alongside the other tools people have built around it. Here is where each one shines, and where `justz` fits in.

| If you want to... | Good choices |
|---|---|
| Run tasks, write recipes, use modules and attributes | [`just`](https://github.com/casey/just) itself. Its manual is the reference for syntax. |
| Edit justfiles with syntax checks and completion in your editor | [`just-lsp`](https://github.com/terror/just-lsp), a language server for `just`. |
| Let an agent call your recipes through MCP | [`just-mcp`](https://github.com/PromptExecution/just-mcp), an MCP server for `just`. |
| Teach an agent `just` syntax and conventions | The skills [`cli-just`](https://github.com/paulrberg/dot-agents/tree/main/skills/cli-just), [`justfile-expert`](https://github.com/laurigates/claude-plugins/tree/main/tools-plugin/skills/justfile-expert) and [`justfile`](https://github.com/mizchi/skills/tree/main/justfile). |

**What `justz` adds.** The tools above help you write and run justfiles. `justz` is about what happens once a project has many of them: can a new person or an agent find the right command, and does the documentation still match the recipes? It brings a grouped command reference with examples, checks you can run in CI, a single registry that the help and the completion share, global shortcuts that cannot hide a real command, and JSON output for tools. It also ships two agent skills of its own, `justz-cli` and `justfile-authoring`, focused on a discoverable command surface and on keeping documentation honest.

**How they combine.** Use `just-lsp` while you write, `justz check` before you commit, and `just-mcp` if your agent should run recipes itself. The syntax skills and the `justz` skills do not compete: install both and your agent gets syntax help and a convention for making the result discoverable.

Know a good tool or skill that belongs in this table? Open an issue and tell us. All of this builds on the work of the `just` maintainers, and thanks go to the authors of the tools above.

## How it works

Describe your commands once, next to your justfile:

```json
{
  "groups": {
    "core": {
      "label": "Core",
      "commands": {
        "test": {
          "desc": "Run the test suite",
          "args": [
            { "name": "-k <pattern>", "desc": "Filter tests by name pattern" },
            { "name": "-w, --watch", "desc": "Re-run on file changes" }
          ]
        }
      }
    }
  }
}
```

Then `justz help`, `justz check`, `justz fmt` and `justz completions` work from that file and from your justfile. To start a new project with this layout, run `justz init --variant rich`. To get completion and type checks for the registry in your editor, save the schema with `justz schema > commands.schema.json` and point your editor at it, or add a `"$schema"` line at the top of the registry with `https://raw.githubusercontent.com/igorlino/justz/main/schema/commands.schema.json`.

### Read a command's text from a script or a test

`justz help <group> <command> -j` is the supported way to read what a command is documented as: the JSON has `desc`, and `detail` when the command has one (`exits`, `seeAlso` and the arguments are there too). **`desc` plus `detail` is the text of a command.** A command's long text can move from `desc` to `detail` without any change to the command, so code that asserts a phrase in `desc` alone breaks when it does. The JSON contract is that fields may be added: a consumer must ignore unknown fields, and fields are not renamed or removed. Reading the registry file (`config/commands.json`) directly is not part of the contract: its layout, and where a given sentence lives in it, can change, and for a native project it does not exist. This is the call, with its output, run against the demo registry in `testdata/help-reference/demo` after adding a `detail` to its `test` command:

```sh
justz help core test -j | jq '{desc, detail}'
```

```json
{
  "desc": "Run the test suite",
  "detail": "Runs only the unit tests; integration tests need `test --integration`.\n\nNeeds no network."
}
```

## Install

`justz` needs [`just`](https://github.com/casey/just) on your PATH.

**Homebrew** (macOS, Linux, WSL2):

```sh
brew install igorlino/tap/justz
```

**Install script** (macOS, Linux, WSL2). It verifies the SHA256 of the download before installing anything, installs into `~/.local/bin` and never uses `sudo`:

```sh
curl -fsSL https://raw.githubusercontent.com/igorlino/justz/main/install.sh | sh
```

**Go**:

```sh
go install github.com/igorlino/justz/cmd/justz@latest
```

**Windows**: `winget install igorlino.justz` is coming. Until then, download the `.zip` from the [releases page](https://github.com/igorlino/justz/releases).

Then check your setup:

```sh
justz doctor
```

### Run `justz check` before commits and in CI

A project that drops its own documentation guard needs something that runs `justz check` automatically. [`docs/hooks-and-ci.md`](docs/hooks-and-ci.md) has snippets to copy:

- a hook for the [pre-commit](https://pre-commit.com) framework (`.pre-commit-hooks.yaml` in this repository, hook id `justz-check`);
- a plain git hook, `scripts/pre-commit-hook.sh`, for projects without the framework;
- a GitHub Actions job that installs a pinned `justz` with the verifying install script and runs `justz check`.

Neither hook passes silently when `justz` is missing: the commit fails, and `JUSTZ_HOOK_SKIP_MISSING=1` skips it with a loud warning on stderr.

### Verify a download

Releases include SHA256 checksums and build provenance attestations. To check an archive you downloaded:

```sh
gh attestation verify justz-<version>-<os>-<arch>.tar.gz --repo igorlino/justz
```

## Agent skills

- `justz-cli`: how an agent runs and troubleshoots `justz`.
- `justfile-authoring`: how to design a discoverable, well-documented justfile, built on `justz`.

Claude Code:

```sh
/plugin marketplace add igorlino/justz
/plugin install justz --marketplace igorlino/justz
```

The one-step install needs Claude Code 2.1.275 or later; on older versions use `/plugin install justz@justz`. The skills then appear as `/justz:justz-cli`. A third-party marketplace does not auto-update by default: run `/plugin marketplace update justz` to get new versions. Any other agent (Codex, Copilot, Antigravity and others) can install them with the `skills` CLI:

```sh
npx skills add igorlino/justz
```

The skills never install `justz`. They check for it and tell you how to install it.

## Contributing

The project is not open for code contributions yet. Ideas and feedback are welcome as issues.

## License

[MIT](LICENSE)
