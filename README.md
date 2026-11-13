# justz

> Just addons for a superior agent and developer experience

[`just`](https://github.com/casey/just) is a great task runner. Past a dozen recipes, though, two things get hard: new people (and AI agents) can't tell what the project can do, and the docs, the help text and the recipes slowly drift apart.

`justz` is a small, single-binary CLI that fixes both, without replacing `just`. You keep your justfile. `justz` adds the layer around it.

> **Status: early development.** Available today: `justz fmt`, the shim system (`justz setup`, `justz shim add`, `list`, `remove` and `doctor`, plus the `jz` shortcut), `justz resolve`, `justz generate`, `justz assess` and `justz doctor`. The help renderer, `check` and completion are what comes next, and they are not available yet.

## What you get

- **Help people actually read.** A grouped, searchable command reference with examples, rendered from your project's own data. Zoom into a group or a single command, or print it as JSON for tools.
- **Checks that keep docs honest.** `justz check` fails when a recipe has no documentation, when the registry and the justfile disagree, or when an example no longer parses. Run it in pre-commit and CI instead of discovering drift months later.
- **Canonical formatting** *(available)*. `justz fmt` sorts your command registry into one stable order, so diffs stay small and reviews stay calm. Use `justz fmt --check` in CI to fail on an unformatted registry.
- **Shortcuts that never break** *(available)*. Pin a command such as `tz` to a project's justfile and run it from any directory, with collision checks so it never hides a real command on your system.
- **Completion for free.** Shell completion generated from the same data as the help.
- **Built for agents.** Every command has stable JSON output and exit codes, so an AI agent can discover what a repo offers and verify its own changes without guessing. Two agent skills, `justz-cli` and `justfile-authoring`, teach agents how.

## Why another tool?

- **One binary, almost no dependencies.** No `jq`, `awk`, Python or Node to install. It only needs `just`.
- **One source of truth.** The help, the checks and the completion all read the same data, so they cannot disagree.
- **It reads `just` the supported way**, through `just --dump --dump-format json`, so it keeps working as `just` evolves.
- **It stays out of your secrets.** `justz` never reads, stores or injects credentials.
- **Verified releases.** Releases ship with checksums and build provenance attestations.

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

## Agent skills (planned)

- `justz-cli`: how an agent runs and troubleshoots `justz`.
- `justfile-authoring`: how to design a discoverable, well-documented justfile, built on `justz`.

## Contributing

The project is not open for code contributions yet. Ideas and feedback are welcome as issues once the first release is out.

## License

[MIT](LICENSE)
