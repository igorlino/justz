# Changelog

What changed in each release of `justz`, for people who use it.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Each release on the [releases page](https://github.com/igorlino/justz/releases) lists every commit.

## Unreleased

### Added

- `justz check` warns about an example that gives a value with `=` to a flag declared without one, such as `-tail=200` (`example-boolean-value`).
- `justz check` warns about tiers that do not match the groups (`registry-tiers`): a group in no tier is hidden from the summary, and a name that is not a group is dropped without a word.

## 0.10.0 - 2026-11-17

### Added

- `justz help -j --expanded` prints every command of the project in one document: `commands`, each with its `path`, `desc`, `hasDetail` and `tags`.
- `justz doctor --help`.
- `CONTRIBUTING.md`.
- `jz --help`, and the same for a shim, says on stderr what the shortcut adds, before the help of `just`.
  `JUSTZ_QUIET=1` silences it.

### Changed

- The website keeps its section links on a phone, has a control that pauses the demo, no longer moves the page when the demo changes, and announces a copied command to a screen reader.
  It serves `llms.txt` at its root.
- The installation page and the install script name `justz setup` as the step after an install, as the README does.
  The quickstart of the README and of the website shows `jzh <command>` and `jz <command>`, and the website's starts with the install.

### Fixed

- In a project with no registry and no overlay, `justz help` shows every parameter of a recipe, in its order, not only those with help text.
  `args` in `help -j` has them too, with an empty `desc`.
- In a project with no registry, with or without an overlay, the summary of `justz help` says "run without args to see sub-commands" only when every module has a default recipe.
  Without one, a bare `just <module>` runs the first recipe of the module.
- `justz help` no longer prints a `#` after a command that has no description, in the summary, the expanded reference and a group.
- `justz check` passes on an overlay that `justz generate` has just written, when a `[doc]` has an `example:` line.
  It reported `registry-desc-diverges` and told you to run `justz generate` again.
- `justz resolve --json` prints its document in a project with no registry: `"status": "none"`, `"reason": "no-registry"`, exit `3`.
  It printed a plain message.
- **Breaking.**
  `justz help` with more than two words exits `2`, as every usage error does.
  It exited `1`.
- `justz --help` lists exit code `3`, and `justz help --help` says that the commands can come from the justfile alone.
- `justz init` with a wrong `--variant` no longer suggests `--name`.

## 0.9.0 - 2026-11-17

### Changed

- **Breaking.**
  `justz init` writes a `rich` project by default: a justfile and the registry, `config/commands.json`.
  Pass `--variant native` for a project that needs only `just`.
- **Breaking.**
  `justz assess` recommends `rich` on every project, on one condition: every machine that shows the help can install `justz`.
  The size of the project no longer decides.
  In `assess --json`, `recommendation.variant` is always `rich`: read the condition before you act on it.
- The `justfile-authoring` skill recommends `rich` by the same rule.
- The README is rewritten, and the documentation is split into pages under `docs/`: getting started, installation, command reference, registry, checks, shims, for agents, ecosystem and FAQ.
- The `justz-cli` skill is restructured: a short `SKILL.md` with a quick start and tables, and the long material in three reference files.
- Release archives include the `docs/` directory and this changelog.
- The `justz-cli` skill no longer carries install commands that fetch an unpinned source (`go install ...@latest`, and the install script piped to a shell).
  It names Homebrew, links to the installation page, and says how to verify a release.

### Added

- `justz setup` adds the `jz` and `jzh` shortcuts to the shim directory, so they exist after any install, not only with Homebrew.
  It adds nothing when a shortcut is already installed, or when another command or a registered shim has the name, and the two are added together.
  `setup --json` reports them in `shortcuts`, and `justz doctor` checks them.
- A showcase website in `site/`, deployed to GitHub Pages by `.github/workflows/pages.yml`.
  Its quickstart and its terminal demo show `justz setup`, `jz` and `jzh`, like the README.
- `AGENTS.md` and `llms.txt`.
- `assess --json` has `condition` and `otherwise` in `recommendation`: what the recommended variant needs, and the variant to use when the project cannot give it.
- `docs/variants.md`: what `rich`, `native` and `local-rich` each give, how to choose, and how to move a `native` project to `rich`.
- A test that fails when a relative link in the documentation or the skills points at a file that does not exist.
- `docs/writing-docs.md`, and tests that fail when a command or a rule is undocumented, or when the README or a `SKILL.md` grows past its budget.

## 0.8.1 - 2026-11-15

### Fixed

- `justz doctor` shows `just`'s own error, and warns about a `just` older than 1.37.0.
- `justz check` no longer reports a bare or single-flag example that sets an environment variable.

## 0.8.0 - 2026-11-15

### Added

- Commands in the registry can declare `exits`, `env`, `tags` and `seeAlso`.
  The zoom shows them, and `help -j` carries them.
- The group listing of `help -j` has `hasDetail` per command.
- `justz check`: the rules `registry-shape`, `registry-exits`, `registry-env`, `registry-tags`, `tags-confirm`, `volatile-text`, `detail-flag-no-row`, `detail-env-no-row`, `detail-long`, `detail-no-paragraphs` and `detail-restates-desc`.
- `justz check` verifies example flags against the argument rows.
- `justz check` and `justz lint`: `--rule <name>` shows one rule, and the text output prints a fix once per rule.
- A `registry-desc-diverges` finding prints the exact `[doc]` line to paste.
- `justz lint`: `module-dispatcher-missing` and `module-dispatcher-lists-root`.
  `recipe-long` reports interpolation counts.
- A pre-commit framework hook, a plain git hook and a GitHub Actions snippet that run `justz check`.

### Changed

- `justz assess` agrees with `check` on `default` recipes, and advises when the help is not rendered by `justz`.

### Fixed

- `justz check` skips recipes of nested modules when it maps advice and stale entries.

## 0.7.0 - 2026-11-15

### Added

- A command's `detail` field: longer text shown only in the command zoom, with URLs, flags, placeholders and command spans highlighted.

### Fixed

- Text from a project is safe to print on a terminal: control characters are replaced.
- `justz resolve` never reports a command key that is not a plain name.
- `justz setup` treats curly double quotes as quotes in the PowerShell PATH hint.

## 0.6.0 - 2026-11-14

### Added

- `justz init` scaffolds a justfile that documents itself.
- `justz lint`, with `recipe-long`, `secret-parameter`, `secret-interpolated`, `module-shell-unset` and the `justz-ignore` opt-out.
- `justz schema` prints the JSON Schema of the registry.
- The `justz-cli` and `justfile-authoring` agent skills, published as a Claude Code plugin marketplace.

### Fixed

- `justz help` wraps long descriptions under the description column on a terminal.

## 0.5.1 - 2026-11-14

### Fixed

- A foreign shim stub is reported once, and `--replace` is no longer suggested for it.
- The Windows binary passes the winget post-install checks.

## 0.5.0 - 2026-11-13

### Added

- `justz check` finds documentation drift, including a multi-line comment block used as a recipe doc.
- `justz completions` for bash and zsh.

## 0.4.0 - 2026-11-13

### Added

- `justz help`: the summary and expanded views, the zoom into a group or a command, `--json` and `--browse`.
- `justz help` renders native projects from the justfile.
- Shims run help: `jzh`, and the help names of pinned shims.

## 0.3.0 - 2026-11-13

### Added

- `justz shim add`, `list`, `remove` and `doctor`, with collision checks, and `justz setup`.
- `justz resolve`, `justz generate` and `justz assess`.
- `justz` acts as `jz`, and Homebrew installs the `jz` link.

## 0.2.0 - 2026-11-13

### Added

- `justz fmt` rewrites the registry in its canonical order.

### Fixed

- `justz fmt` finds the registry from a subdirectory.

## 0.1.0 - 2026-11-13

### Added

- `justz --version`, `justz doctor`, and the install script with checksum verification.
