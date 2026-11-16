# Installation

## Requirements

`justz` needs [`just`](https://github.com/casey/just) 1.37.0 or newer on your PATH. Install a current one with:

```sh
brew install just
# or
curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to ~/.local/bin
# or
cargo install just
```

An older `just` fails with `Unknown attribute` errors (Ubuntu's package is 1.21.0). `justz doctor` warns about it.

## Install justz

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

Then run `justz doctor`.

## Verify a download

Releases include SHA256 checksums and build provenance attestations. Archives are `.tar.gz`, and `.zip` for Windows:

```sh
gh attestation verify justz-<version>-<os>-<arch>.tar.gz --repo igorlino/justz
```

## Which `just` features need which version

`justz check`, `help` and `lint` were run on the `justz init` output (native and rich, with modules) against `just` 1.21.0 to 1.46.0.

| `just` version | What happens |
|---|---|
| up to 1.31.0 | cannot parse `mod` |
| 1.32.0 | rejects `[group]` on a module |
| 1.33.0 to 1.36.0 | parses, but leaves the `doc` of a `[doc]` recipe out of the JSON dump, so `check` reports a false `doc-missing` |
| **1.37.0** | first version where everything works |
| 1.42.0 | `[metadata]`, needed for the `justz-ignore` opt-out |
| 1.46.0 | `[arg]`, for a justfile you write (not needed by `justz` itself) |

## Update and uninstall

Update with the same method you installed with (`brew upgrade justz`, re-run the script, `go install ...@latest`). Uninstall by removing the binary. Shims live in their own directory; remove them first with `justz shim remove <name>`.
