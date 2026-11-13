# Inventory of the existing tests

`justz` replaces a set of bash tools: a help renderer, a documentation checker, shell completion, a `jz`/`jzh` shortcut shim, and a generator and assessment script. They come with their own tests. This page lists every one of them and says what happens to it, so nothing stops being tested without a decision.

Outcomes:

- **Becomes a justz test**, with the milestone where it lands.
- **Dropped**, with the reason.
- **Replaced**, covered by something different.

The suites are named by what they check, not by file name. 122 cases in total.

## Help renderer: 25 cases

| Cases | Outcome |
|---|---|
| A command found by a bare name, a unique prefix, two words, a group plus a command prefix, or a core command (6) | Becomes a justz test in M4: lookup rules for `justz help <group> <command>`. |
| The group view equals the dispatcher view, minus color (1) | M4. |
| Ambiguous name exits 2, no match exits 2, three words exits 1, unknown group exits 2 (4) | M4. The exit codes are kept unless the differences ledger says otherwise. |
| `-j` zoom: exit 0, path, args, examples and invoke fields; ambiguous name exits 2 and gives an array of both nodes (4) | M4: `justz help --json`. |
| `-b` without a terminal exits 0 and never prompts (1) | M4: the browse mode must not prompt without a TTY. |
| Every registered group and command resolves (loop, 1) | M4, as a table test over a fixture registry. |
| Colorizing of usage text: flags, `--flag=value`, placeholders, quoted text, negative numbers, redirects, tabs, no color without a palette (8) | M4: unit tests of a pure function, plus `NO_COLOR` and piped output. |

## Documentation checker: 14 cases

| Cases | Outcome |
|---|---|
| Valid native justfile passes; valid rich justfile and registry pass (2) | M5: `justz check`. |
| Missing `[doc]` on a submodule recipe; missing `[group]` on a top-level recipe (2) | M5. |
| Recipe missing from the registry; stale recipe in the registry; `[doc]` differs from the registry description (3) | M5. The audit of a large repo showed this check is also needed where recipes are not documented at all. |
| Bare command in a group example; redundant single boolean-flag example; long flag used where a short alias exists; enum-valued flag missing from the group examples (4) | M5: the example rules. |
| Unsorted commands, args and examples are detected and the format command fixes them (1) | M5 reports it; the fixing part is already `justz fmt`, with a byte-for-byte test against the old output. |
| Outdated help script version stamp and the drift notice (1) | **Dropped.** There is no vendored script to go stale. A minimum `justz` version in the skill replaces it. |
| Variadic `{{ args }}` interpolation is detected and a safer form recommended (1) | M5 as a `check` rule, or M8 as a `lint` rule if it proves to be advice and not an error. |

## Shell completion: 31 cases

| Cases | Outcome |
|---|---|
| Top level completes core recipes and groups sorted; group subcommands with descriptions; scan flags in short and long form (4) | M5: `justz completions`. |
| Enum values complete after a flag; boolean flags offer the remaining flags; free-text flags fall back to files; positional values come first (8) | M5. |
| `help` completion: core recipes then groups, bare names, a group's commands, its own flags (6) | M5. |
| Finding the registry from a subdirectory, and in a hidden `.*/config` directory (2) | M5, and a **finding**: `justz` discovery only looks in the current directory today. It needs to walk up to the project root. |
| `jzh` and `jz` completion from the project registry, bare names, flags and enum values (6) | M5, through the shim system. |
| Name resolution: a unique command gives group and name; a name in two groups, a group word, a core recipe and an extra entry are not resolved (5) | M3: `justz resolve`. |

## Shortcut shim (`jz`, `jzh`): 51 cases

| Cases | Outcome |
|---|---|
| No justfile, justfile without the help script, missing or unreadable version marker, version older than the minimum, numeric version comparison (11) | **Dropped.** They guard against a vendored help script that will not exist. `justz doctor` and the minimum version in the skill take over. A missing justfile still needs a clear exit 2, which becomes a M3 test. |
| Runs `just` in the project root, walks up to find the justfile, passes the recipe exit code through (3) | M3. |
| Bare name runs a command without its group, echoes the resolved path, prefixes never run, ambiguous names exit 2 and list the candidates, a real root recipe wins, explicit paths and core recipes and dash-first arguments pass through, extras are not run (11) | M3: `justz run` and the shim. |
| `--list` shows the recipes; `--which` prints the path, exits 2 for ambiguous, 3 for unknown or a group word, 2 without a name (6) | M3: `justz resolve`. |
| A pinned justfile (`JZ_JUSTFILE`) works from any directory; the name used in messages; soft mode falls through to plain `just`, including a missing justfile, while an ambiguous name still exits 2; a group path takes the fast path (7) | M3: pinned shims with the registry settings. Soft mode stays opt-in per shim. |
| No control variable leaks into the recipe's environment (1) | M3. Easy to lose, so it stays an explicit test. |
| Behavior against the old help script version (older than bare-name support): plain `just` error, `--which` exit 2, update message, explicit paths still work (4) | **Dropped** with the vendored script. |
| `jzh` bare prints the summary and usage hint; `jzh <command>` zooms from a subdirectory without a help recipe and shows the description; `jzh -j` prints valid JSON (6) | M4, through the `help` command and the shim. |

## Terminal smoke test and the test runner

| Item | Outcome |
|---|---|
| zsh pseudo-terminal completion smoke test (1 case) | M5, as an end-to-end test in a container with zsh. |
| The runner script that executes all suites | **Replaced** by `just ci`. |

## Totals

Of 122 cases: 16 are dropped because they only guard the vendored help script or its version, and the other 106 become `justz` tests in M3 to M5. The runner script is replaced by `just ci`. The generator and assessment scripts had no tests of their own, so M3 adds snapshot tests for `justz generate` and `justz assess`.

## Findings to carry forward

- Registry discovery has to walk up from a subdirectory to the project root.
- The summary and expanded views of the old tool ignore `NO_COLOR` (see the reference snapshots).
- Exit codes of the zoom lookups are part of the tested behavior. Keep them unless the differences ledger records a change.
