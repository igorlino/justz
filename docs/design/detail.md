# Design note: a `detail` field for long command descriptions

Status: implemented in one PR (the field, the zoom section, the JSON, the schema property, the check fix text and the tests). The open questions are decided as recommended.

## Problem

A command's `desc` has three jobs at once: it is a row in the help, it is the `[doc]` of the recipe (`justz check` requires the two to be the same text), and it is where an author puts everything worth knowing about the command. The third job breaks the first two. `check` warns when `desc` is over 120 characters (`registry-desc-long`), and a long `[doc]` makes `just --list` unreadable.

Measured on the large private registry of the pilot: 149 of its 249 commands have a description over 120 characters, 33 are over 500, and the longest is 2,165. The text is useful (what a recipe does step by step, which flags matter, what it needs), but it is prose for someone who is already looking at one command. Today it has nowhere else to go.

## Proposal

An optional string `detail` on a command, shown only where the reader has zoomed into that command.

```json
"deploy": {
  "desc": "Deploy the EWF report to the cloud and reindex it",
  "detail": "Builds today's day directory, runs archive-doctor, uploads it to the storage box and reindexes on the cloud host.\n\nThe legacy static rsync is off unless EWF_DEPLOY_LEGACY_STATIC=1.",
  "args": [ ... ]
}
```

- `desc` stays one short line, at most 120 characters, and equal to the `[doc]`. Nothing about `check` changes.
- `detail` is the rest. It is not compared with anything.

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

## What it does for the pilot registry

The 149 long descriptions split mechanically: `desc` is the first sentence, `detail` is the rest, and the `[doc]` is the new `desc`. 122 of the 149 have a first clause (up to a sentence end, a semicolon, a colon or a dash) of 120 characters or fewer; by full sentences alone it is 82. The other 27 need a short summary written by hand, and the removed text stays whole in `detail`. Nothing is lost and `registry-desc-long` goes away.

## Not in scope

- Markup (Markdown, ANSI) inside `detail`, and links.
- Detail for groups. A group already has `footer`, which its zoom prints.
- A per-argument long text.
- A line limit on `detail`.
- A way to give the same text to native justfiles (for example through `[metadata]`).

## Plan

1. `Detail` on the help model, the zoom section, the JSON field, the schema property and the new `registry-desc-long` fix text, with tests. One PR.
2. At release time, the docs that describe the field wait for the release that ships it, as before: `skills/justfile-authoring/references/variant-rich.md` (the `desc` bullet, the check table and the schema reference), `SKILL.md` (the description rules), `references/choosing-a-variant.md` (a note beside the 20-of-32 evidence), `docs/design/schema.md` (its field table, which says to update it), and the `justz-cli` skill only if it names description length. The README has no such text.
3. In a consumer: split the long descriptions as above.

### Drift

`detail` is the one prose field that nothing compares with anything, so it can go stale. What the field changes is the other side: `desc` and `[doc]` become short, so `check` can keep enforcing that they are the same. It does not make the 20-of-32 divergence in the skill's evidence any less true. Keeping `detail` current is the author's job, and the skill says so.

### Tests

- **Zoom:** a command with a one-paragraph, a multi-line and a long wrapped `detail`, with and without colour and at two widths; leading and trailing newlines, an empty line and a word longer than the width; a command without `detail` is unchanged (the existing zoom goldens).
- **JSON:** `detail` is present when set and absent when empty, and absent from the group listing and from the ambiguous array.
- **Placement:** `detail` is not in the summary, the expanded view or the group zoom.
- **Schema:** a string passes (add it to the all-fields document), and `{"detail": 3}` fails with `$.groups.a.commands.x.detail: want string, got number`.
- **`check`:** the `registry-desc-long` fix text for a committed registry and for the overlay.

## Open questions

1. **Heading.** A **Details** heading, or the prose straight under the description with a blank line. Recommendation: the heading, so that a reader can tell the one-line summary from the longer text.
2. **Field name.** `detail`, `details`, `long` or `help`. Recommendation: `detail`: singular like `desc`, and not `help`, which is already a command.
3. **`generate` carrying `detail`.** Like `values`, a hand-added `detail` in an old overlay could survive regeneration. The overlay is personal and derived, and there is nothing to carry from. Recommendation: no.
4. **A rule for the opposite case.** A `detail` that only repeats `desc`. Recommendation: no; it is harmless and rare.
5. **Placement.** After Options (the zoom shows the actionable text first) or straight after the description. Recommendation: after Options. The cost is that a reader who wants the prose scrolls past the flags; the gain is that a long `detail` never hides the examples in the `--browse` preview.
