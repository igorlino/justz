# justz

> Just addons for a superior agent and developer experience

[`just`](https://github.com/casey/just) is a great task runner. Past a dozen recipes, though, two things get hard: new people (and AI agents) can't tell what the project can do, and the docs, the help text and the recipes slowly drift apart.

`justz` is a small, single-binary CLI that fixes both, without replacing `just`. You keep your justfile. `justz` adds the layer around it.

> **Status: early development.** Available today: `justz help` (summary, `--expanded`, zoom into a group or command, `--json`, `--browse`), `justz check`, `justz completions` (zsh and bash), `justz fmt`, the shim system (`justz setup`, `justz shim add`, `list`, `remove` and `doctor`, plus the `jz` and `jzh` shortcuts), `justz resolve`, `justz generate`, `justz assess` and `justz doctor`. Everything in the list below is available; `justz init` and `justz lint` are what comes next.

## What you get

- **Help people actually read** *(available)*. A grouped, searchable command reference with examples, rendered from your project's own data. Zoom into a group or a single command, or print it as JSON for tools.
- **Checks that keep docs honest** *(available)*. `justz check` fails when a recipe has no documentation, when the registry and the justfile disagree, or when an example no longer parses. Run it in pre-commit and CI instead of discovering drift months later.
- **Canonical formatting** *(available)*. `justz fmt` sorts your command registry into one stable order, so diffs stay small and reviews stay calm. Use `justz fmt --check` in CI to fail on an unformatted registry.
- **Shortcuts that never break** *(available)*. Pin a command such as `tz` to a project's justfile and run it from any directory, with collision checks so it never hides a real command on your system.
- **Completion for free** *(available)*. Shell completion generated from the same data as the help.
- **Built for agents.** Every command has stable JSON output and exit codes, so an AI agent can discover what a repo offers and verify its own changes without guessing. The agent skill `justz-cli` teaches agents how, and `justfile-authoring` is planned.

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

**What `justz` adds.** The tools above help you write and run justfiles. `justz` is about what happens once a project has many of them: can a new person or an agent find the right command, and does the documentation still match the recipes? It brings a grouped command reference with examples, checks you can run in CI, a single registry that the help and the completion share, global shortcuts that cannot hide a real command, and JSON output for tools. It also ships an agent skill of its own, `justz-cli`, and `justfile-authoring` is planned, both focused on a discoverable command surface and on keeping documentation honest.

**How they combine.** Use `just-lsp` while you write, `justz check` before you commit, and `just-mcp` if your agent should run recipes itself. The syntax skills and `justz-cli` do not compete: install both and your agent gets syntax help and knows how to use `justz`.

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

Then `justz help`, `justz check`, `justz fmt` and `justz completions` work from that file and from your justfile.

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

### Verify a download

Releases include SHA256 checksums and build provenance attestations. To check an archive you downloaded:

```sh
gh attestation verify justz-<version>-<os>-<arch>.tar.gz --repo igorlino/justz
```

## Agent skills

- `justz-cli`: how an agent runs and troubleshoots `justz`.
- `justfile-authoring`: how to design a discoverable, well-documented justfile, built on `justz` (coming).

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
