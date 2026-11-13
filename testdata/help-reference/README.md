# Reference output from the existing help tooling

These files record what the bash help renderer that `justz` replaces (version 7.3.0) prints for a synthetic registry, `demo/config/commands.json`. They exist so that, when `justz help`, `check` and `resolve` are built, we can tell what changed on purpose from what changed by accident. They are not a promise to match the old output byte for byte.

Each file in `demo/out/` is one case. The last line is `[exit N]`, the exit code.

## How they were produced

The registry was copied to `config/commands.json` in a scratch project root next to the renderer at `scripts/lib/help.sh`, and each case ran in a clean environment: `env -i TERM=dumb NO_COLOR=1 COLUMNS=<width>`, with stdout and stderr captured together.

| Cases | Invocation |
|---|---|
| `summary-{80,100,140}` | `show_help summary` |
| `expanded-{80,100,140}` | `show_help expanded` |
| `expanded-wide-100` | `show_help expanded -w` |
| `zoom-group-{build,tool}` | `show_help expanded <group>` |
| `zoom-group-prefix-dep` | `show_help expanded dep` (a unique prefix of `deploy`) |
| `zoom-command-*` | `show_help expanded <group> <command>`, or a bare command name such as `test` |
| `zoom-command-missing` | `show_help expanded tool nope` |
| `json-*` | `show_help expanded -j [group [command]]` |
| `dispatcher-{tool,build}` | `show_help dispatcher <name>` |
| `resolve-{test,build,nope}` | `help.sh --resolve <name>` |
| `version` | `help.sh --version` |

## Things the snapshots show that we need to decide

- The summary and expanded views print ANSI color codes even when the output is piped and `NO_COLOR=1` is set. The zoom views do not. `justz` should honor `NO_COLOR` and non-terminal output everywhere. Tests that compare against these files should strip ANSI codes first.
- `--resolve build` exits 3 (no match), although `build` is both a command in `core` and a group name. The rules for exact resolution when a name is both need writing down before `justz resolve` exists.
- A missing zoom target prints a hint pointing at `help -b` and exits 2.
