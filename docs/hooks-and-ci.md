# Run `justz check` automatically

`justz check` finds drift between the justfile and its documentation. These snippets run it before a commit and in CI, so that a project that drops its own runtime guard keeps the coverage. Copy the one you need.

`justz check` exits `1` on an error. A warning fails it only with `--strict`.

## Pre-commit framework

Add the hook to `.pre-commit-config.yaml` and pin a release in `rev`. Replace `vX.Y.Z` with the first release that ships this hook (it is not in v0.7.0 or earlier), and then keep it pinned; the `justz` version in the Actions job below may stay on any existing release.

```yaml
repos:
  - repo: https://github.com/igorlino/justz
    rev: vX.Y.Z
    hooks:
      - id: justz-check
```

The hook (`.pre-commit-hooks.yaml` at the root of this repository) uses `language: system`, runs `justz check` with `pass_filenames: false`, and runs when the justfile, a `*.just` file or the registry (`commands.json`, `.just-local-rich.json`) is part of the commit. If `justz` is not installed, pre-commit stops with `Executable 'justz' not found` and the commit fails. The hook never passes silently.

## Plain git hook

For a project without the framework, copy [`scripts/pre-commit-hook.sh`](../scripts/pre-commit-hook.sh) to `.git/hooks/pre-commit` and make it executable:

```sh
curl -fsSL https://raw.githubusercontent.com/igorlino/justz/vX.Y.Z/scripts/pre-commit-hook.sh -o .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

A repository with no `justfile` (`justfile`, `Justfile`, `JUSTFILE` or `.justfile` in the root) has nothing to check, so the hook prints a short note on stderr and exits `0`, before it looks for `justz`. When `justz` is not on PATH the hook prints on stderr that `justz check` did not run, and exits `1`: the commit fails. To skip instead, on a machine that has no `justz`, set `JUSTZ_HOOK_SKIP_MISSING=1`. The hook then prints a loud warning and exits `0`. Only the value `1` skips. Otherwise the hook exits with the status of `justz check`.

## GitHub Actions

This job installs `justz` with `install.sh`, which verifies the SHA256 of the download before it installs anything, and runs `justz check`. Pin the version in `JUSTZ_VERSION`: the install script and the binary then both come from that release, and a new release cannot change your CI. Pin the actions to a commit as well, as this repository does.

```yaml
name: justz
on:
  push:
  pull_request:

permissions:
  contents: read

jobs:
  check:
    runs-on: ubuntu-latest
    env:
      JUSTZ_VERSION: v0.7.0
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
      - uses: extractions/setup-just@53165ef7e734c5c07cb06b3c8e7b647c5aa16db3 # v4.0.0
      - name: Install justz
        run: |
          curl -fsSL "https://raw.githubusercontent.com/igorlino/justz/${JUSTZ_VERSION}/install.sh" | sh -s -- --version "${JUSTZ_VERSION}"
          echo "$HOME/.local/bin" >> "$GITHUB_PATH"
      - run: justz check
```

`justz` needs `just`, which `setup-just` installs. Add `justz fmt --check` after `justz check` for a project with a registry, and `--strict` to make warnings fail.
