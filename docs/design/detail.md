# Design note: a `detail` field for long command descriptions

Status: implemented in one PR (the field, the zoom section, the JSON, the schema property, the check fix text and the tests). The open questions are decided as recommended. Later changes: with colors on, the zoom highlights command spans, flags, placeholders and URLs in `detail` (`docs/design/terminal-text.md`, part 2), which replaces the "no markup" line under "Not in scope" for those four; and the README now has an entry for the field. Real use corrected the first version of this note: descriptions are written by hand, not split mechanically, a `detail` holds facts and not steps, and advice rules limit its length (see "Writing desc and detail" and "Decision after real use").

## Problem

A command's `desc` has three jobs at once: it is a row in the help, it is the `[doc]` of the recipe (`justz check` requires the two to be the same text), and it is where an author puts everything worth knowing about the command. The third job breaks the first two. `check` warns when `desc` is over 120 characters (`registry-desc-long`), and a long `[doc]` makes `just --list` unreadable.

Measured on a large registry, most commands had a description over 120 characters, a good many were over 500, and the longest was over 2,000. The text is useful (which flags matter, what the command needs, what it changes), but it is prose for someone who is already looking at one command. Today it has nowhere else to go.

## Proposal

An optional string `detail` on a command, shown only where the reader has zoomed into that command.

```json
"deploy": {
  "desc": "Deploy the report to the cloud host and reindex it",
  "detail": "Does not touch the legacy static copy unless DEPLOY_LEGACY_STATIC=1.\n\nNeeds the storage box mounted. Changes the live index; undo with `deploy --rollback`.",
  "args": [ ... ]
}
```

- `desc` stays one short line, at most 120 characters, and equal to the `[doc]`. Nothing about `check` changes.
- `detail` is the few facts that do not fit in `desc`. It is not compared with anything. "Writing desc and detail" below says what goes in each.

## Where it shows

| View | Shows `detail`? |
|---|---|
| `justz help <group> <command>` (the zoom; `jzh` is the same) | yes, a **Details** section |
| `justz help <group> <command> -j` | yes, as `detail` (left out when empty) |
| `justz help --browse` preview | yes, it is the zoom |
| Summary, `--expanded`, group zoom, completion | no |
| `just --list` | no (it reads `[doc]`, which is `desc`) |

Order in the zoom: invocation, `desc`, Examples, Options, **Details**, See also. The actionable text comes first: a reader of the zoom, or an agent, wants the examples and the flags before the prose, and a 2,000-character `detail` is about 30 lines, which would push both out of the `--browse` preview (it is the zoom, in a window that does not scroll).

### Rendering

The heading is at two spaces and the text at four, wrapped at the zoom width minus six, as for Examples and Options. The zoom width is `COLUMNS`, else the terminal, else 80, clamped to 40 to 100 columns; wrapping is the same on a pipe, so a pipe without `COLUMNS` gets 80. The existing wrap function treats a newline as a line break and an empty line as a blank line, so a short list or two paragraphs survive. Its limits are kept and documented: runs of spaces and tabs collapse to one space, a line cannot be indented, and a word longer than the width is cut. Leading and trailing newlines of `detail` are trimmed, and an empty line is written as a bare newline, with no trailing spaces. No markup is interpreted.

A command without `detail` renders exactly as before, byte for byte, so the existing snapshots do not change.

## What else changes

- **Schema.** `detail` is added to the `command` definition as a string. It only makes the schema accept more, which is the rule in `docs/design/schema.md`.
- **`check`.** No new rule. The fix text of `registry-desc-long` says where the rest of the text goes, and depends on the variant, because `check` also runs on the generated overlay where `detail` has no source: for a committed registry, `keep a one-line description equal to the [doc], and move the rest to "detail" in the registry`; for the overlay, `shorten the [doc] in the justfile, then run justz generate`. The rule still applies to `desc` only, and `registry-desc-diverges` is unchanged: `detail` is compared with nothing.
- **`fmt`.** It keeps the author's field order inside a command and the content of every string (checked: `\n`, quotes, unicode, `<`, `>` and `&` survive), as it does today. It still sorts `args`. The convention, in the skill, is to write `detail` after `desc`.
- **`extra` entries.** `detail` is allowed on them as on commands: the same reader serves both.
- **Group zoom and JSON listings.** The group zoom, its `-j` listing (`CommandSummary`) and the expanded views stay without `detail`, so listings stay small and an agent zooms per command. When a name is ambiguous, `-j` prints an array of commands; those entries leave `detail` out, since the lookup is a choice and not a read.
- **Empty `desc`.** Unchanged: the zoom prints the line it prints today, and `check` still flags an empty description.
- **`generate`.** The overlay is derived from the justfile and has no source for `detail`, so it never writes one. Not carried over from an earlier overlay either (see question 3).
- **Older `justz`.** Versions that do not know the field ignore it, keep it in `fmt`, and show the short `desc`. A registry can use `detail` before every machine has the new version.
- **Native variant.** Nothing. A native justfile has `[doc]` and no place for long text. Not in scope.

## Writing desc and detail

`check` can measure the length of a description. It cannot tell a good one from a bad one. A first clause that fits in 120 characters but only names the subject ("Deploy the report") gives a reader nothing to choose a command by. So a description is never produced by splitting a long one at its first sentence: it is written by hand, from what the recipe does, and the facts that do not fit are moved to `detail`, not cut.

### `desc`

One line, equal to the `[doc]` (`check` requires it). It is what a reader sees in `just --list` and in every listing, so it is how they choose.

- Aim for 80 to 100 characters. The hard limit stays 120 (`registry-desc-long`).
- Start with a verb: "Deploy the report to the cloud host and reindex it".
- Say the effect, not the mechanism: what changes for the user, not which tool runs.
- If the command hesitates (it asks, it dry-runs by default, it needs a flag to do the real thing), say that fact.
- Words over numbers: a count goes stale (`volatile-text` finds the common cases).
- No quotes or backticks: the text is also a `[doc]` string, and `check` prints the line to paste.
- No trailing period.

### `detail`

Empty is correct for most commands. Write one only when a reader who has found the command still needs a fact that `desc` cannot carry.

- Aim for about 400 characters. The advice rules below warn over 600.
- Facts, not narrative: no history, no "we", no tour of the script.
- In this order, and stop when the facts run out:
  1. What the command does not do.
  2. What it needs first (a mount, a login, a running service).
  3. What it changes, and how to undo it.
  4. When to run it and what to run next.
  5. Its exit codes and where its output goes.
  6. How it differs from a sibling command (and name the sibling in `seeAlso`).
- Steps are not facts for a reader of the help. A list of what a script does, in order, belongs in the script.

### Where other content belongs

| Content | Where |
|---|---|
| What a flag or an argument does | its `args` row |
| A worked command line | `examples` of the group, or `example` of the command |
| A related command | `seeAlso` of the command |
| The steps of a script | the header comment of `scripts/<group>/<name>.sh` |
| Why it is built this way | a decision record |
| A procedure for a person on call | a runbook |

### Moving existing long descriptions

Do it by hand, one command at a time: read what the recipe does, write the `desc`, then keep the text that is a fact (in the order above) as `detail` and drop what is narrative or a list of steps. Nothing mechanical is reliable here. `registry-desc-long` goes away when each description is short, and the advice rules below keep the `detail` from growing back into the problem it was added to solve.

## Not in scope

- Markup (Markdown, ANSI) inside `detail`, and links.
- Detail for groups. A group already has `footer`, which its zoom prints.
- A per-argument long text.
- A line limit on `detail` (reversed as advice: see "Decision after real use").
- A way to give the same text to native justfiles (for example through `[metadata]`).

## Plan

1. `Detail` on the help model, the zoom section, the JSON field, the schema property and the new `registry-desc-long` fix text, with tests. One PR.
2. At release time, the docs that describe the field wait for the release that ships it, as before: `skills/justfile-authoring/references/variant-rich.md` (the `desc` bullet, the check table and the schema reference), `SKILL.md` (the description rules), `references/choosing-a-variant.md` (a note beside the 20-of-32 evidence), `docs/design/schema.md` (its field table, which says to update it), and the `justz-cli` skill only if it names description length. The README has no such text.
3. In a consumer: rewrite the long descriptions by hand, as in "Moving existing long descriptions".

### Drift

`detail` is the one prose field that nothing compares with anything, so it can go stale. What the field changes is the other side: `desc` and `[doc]` become short, so `check` can keep enforcing that they are the same. It does not make the 20-of-32 divergence in the skill's evidence any less true. Keeping `detail` current is the author's job, and the skill says so.

### Tests

- **Zoom:** a command with a one-paragraph, a multi-line and a long wrapped `detail`, with and without colour and at two widths; leading and trailing newlines, an empty line and a word longer than the width; a command without `detail` is unchanged (the existing zoom goldens).
- **JSON:** `detail` is present when set and absent when empty, and absent from the group listing and from the ambiguous array.
- **Placement:** `detail` is not in the summary, the expanded view or the group zoom.
- **Schema:** a string passes (add it to the all-fields document), and `{"detail": 3}` fails with `$.groups.a.commands.x.detail: want string, got number`.
- **`check`:** the `registry-desc-long` fix text for a committed registry and for the overlay.

## Decision after real use

Two calls above were reopened after the field was used on a large registry, and decided in [issue 126](https://github.com/igorlino/justz/issues/126). Its details had no line breaks, the median was about 280 characters, some were over 800, the longest renders as about 30 lines, and most opened by restating the `desc`. Both earlier calls are reversed, as advice only. The rules are the enforcement of "Writing desc and detail":

- **A limit on `detail`** was out of scope ("A line limit on `detail`"). A long `detail` is as bad as a long `desc`: the zoom buries the options under it. `check` now warns, with `detail-long` (over 600 characters) and `detail-no-paragraphs` (over 300 characters with no blank line between paragraphs). 600 is about 9 lines at the zoom width, which keeps the examples and options in the `--browse` window next to it; 300 is about the median, so a text longer than a typical `detail` has earned a break.
- **A rule for the opposite case** (a `detail` that repeats `desc`) was rejected as harmless and rare (open question 4). It was neither. `detail-restates-desc` warns when the words of the `desc`, lowercased, without punctuation and with a trailing `s` removed, are the first words of the `detail`. A prefix test was chosen over a similarity score: it has no threshold to tune, an author can see why it fired, and a `detail` that only starts with the same verb does not match. It misses a paraphrase, which is accepted. A `desc` of fewer than four words is not compared, since a short one is a prefix of too many honest details.

All three are warnings, never errors, because an empty `detail` is correct for most commands and nothing here is wrong enough to fail a build, although `check --strict` fails on a warning. A deliberately list-style `detail` without blank lines is silenced on its recipe. The fix text names the thresholds, and a recipe silences a rule with `[metadata('justz-ignore', '<rule>')]`, as for `volatile-text`. The thresholds are constants of `internal/check/detail.go`.

## Open questions

1. **Heading.** A **Details** heading, or the prose straight under the description with a blank line. Recommendation: the heading, so that a reader can tell the one-line summary from the longer text.
2. **Field name.** `detail`, `details`, `long` or `help`. Recommendation: `detail`: singular like `desc`, and not `help`, which is already a command.
3. **`generate` carrying `detail`.** Like `values`, a hand-added `detail` in an old overlay could survive regeneration. The overlay is personal and derived, and there is nothing to carry from. Recommendation: no.
4. **A rule for the opposite case.** A `detail` that only repeats `desc`. Recommendation: no; it is harmless and rare. (Reversed: see "Decision after real use".)
5. **Placement.** After Options (the zoom shows the actionable text first) or straight after the description. Recommendation: after Options. The cost is that a reader who wants the prose scrolls past the flags; the gain is that a long `detail` never hides the examples in the `--browse` preview.
