# Reviewing an existing registry

A clean `justz check` means every recipe has a row and the rows agree with the justfile.
It does not mean the rows are true: an example can fail when it runs, a tag can be missing on a command that places a real order, and a label can say nothing.
Use this procedure when asked to review, audit or improve a `commands.json` that already passes the checks.

## Procedure

1. **Run the mechanical checks first, and print the counts.**
   `justz assess --json`, `justz check`, `justz lint` and `justz fmt --check`.
   Fix what they report before the semantic pass, so its findings are not mixed with theirs.
2. **Find what reads the registry before changing it.**
   Search the tests and scripts for `commands.json`, `just --list` and `justz help`, and note what they assert.
   A test that pins a label or a `desc` breaks when the wording improves; it is the first thing CI fails on.
   Note any repository guard on the file too, such as a limit on the size of a diff, and its documented override.
3. **Read every example as a command.**
   A command's `example` holds one command; the others belong in the group's `examples`, each with a `desc`, where each one is checked.
   When the group has examples for a command, its zoom hides the `example`: check that it appears among them.
   An example that puts `--` before the flags fails unless the recipe drops it, since `just` forwards it.
   `just --dry-run <ns> <cmd> <args>` checks the arity of an example, but it still evaluates backtick assignments, so skip it in a justfile where those run commands.
4. **Check the flags of a wrapper against the tool it wraps.**
   Read the wrapped tool's `--help` and compare its flags with the argument rows and the examples.
   Where it is safe (a `read-only` command), run one documented example: a flag the rows do not declare can still be wrong, such as `-tail=200` passed through a catch-all row to a Go boolean flag.
5. **Check each tag against the script, starting with what costs most.**
   Commands that place real orders, delete data, deploy or spend come first.
   Untagged is unknown, not safe; use the vocabulary in `variant-rich.md` and say `read-only` only after reading the script.
6. **Check the summary.**
   A group label that only repeats its key tells a reader nothing; write what the group does.
   Use tiers by purpose, or none: one tier that lists every group renders like no tiers, and `registry-tiers` (0.11.0 or newer) reports a group that no tier lists, which the summary hides.
7. **Report before changing what is the team's to decide.**
   Rank the findings by cost, fix the mechanical ones, and preview a taxonomy change (tiers, labels, renames) before writing it.

## After the edit

1. Run `justz fmt`, then `justz check` and `justz lint`.
2. Run the tests found in step 2, or the full suite when unsure.
3. Say in the commit why a guard was overridden, when one was.
