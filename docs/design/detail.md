# Design note: a `detail` field for long command descriptions

Status: proposed.

## Problem

A command's `desc` has three jobs at once: it is a row in the help, it is the `[doc]` of the recipe (`justz check` requires the two to be the same text), and it is where an author puts everything worth knowing about the command. The third job breaks the first two. `check` warns when `desc` is over 120 characters (`registry-desc-long`), and a long `[doc]` makes `just --list` unreadable.

Measured on the large private registry of the pilot: 149 of 249 descriptions are over 120 characters, 33 are over 500, and the longest is 2,165. The text is useful (what a recipe does step by step, which flags matter, what it needs), but it is prose for someone who is already looking at one command. Today it has nowhere else to go.

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

Order in the zoom: invocation, `desc`, **Details**, Examples, Options, See also. The details are prose about the command, so they come before the examples that use it.

### Rendering

`detail` is wrapped like the description above it: at the zoom width (the terminal width, between 40 and 100 columns, the same on a pipe) and indented the same way. The existing wrap function already treats a newline in the text as a line break, and an empty line stays a blank line, so a short list or two paragraphs survive. No markup is interpreted.

A command without `detail` renders exactly as before, byte for byte, so the existing snapshots do not change.

## What else changes

- **Schema.** `detail` is added to the `command` definition as a string. It only makes the schema accept more, which is the rule in `docs/design/schema.md`.
- **`check`.** No new rule. The fix text of `registry-desc-long` says where the rest of the text goes: "keep the first sentence as `desc` and move the rest to `detail`". The rule still applies to `desc` only.
- **`fmt`.** It keeps the author's field order inside a command, as it does today. The convention, in the skill, is to write `detail` after `desc`.
- **`generate`.** The overlay is derived from the justfile and has no source for `detail`, so it never writes one. Not carried over from an earlier overlay either (see question 3).
- **Older `justz`.** Versions that do not know the field ignore it, keep it in `fmt`, and show the short `desc`. A registry can use `detail` before every machine has the new version.
- **Native variant.** Nothing. A native justfile has `[doc]` and no place for long text. Not in scope.

## What it does for the pilot registry

The 149 long descriptions split mechanically: `desc` is the first sentence, `detail` is the rest, and the `[doc]` is the new `desc`. 122 of the 149 have a first sentence of 120 characters or fewer. The other 27 need a short summary written by hand, and the removed text stays whole in `detail`. Nothing is lost and `registry-desc-long` goes away.

## Not in scope

- Markup (Markdown, ANSI) inside `detail`, and links.
- Detail for groups. A group already has `footer` for prose, and a group zoom lists commands, not text.
- A per-argument long text.
- A line limit on `detail`.
- A way to give the same text to native justfiles (for example through `[metadata]`).

## Plan

1. `Detail` on the help model, the zoom section, the JSON field, the schema property and the new `registry-desc-long` fix text, with tests. One PR.
2. At release time: the README, the `justz-cli` skill and `variant-rich.md` ("schema quick reference" and the rule on description length) describe the field. They wait for the release that ships it, as before.
3. In a consumer: split the long descriptions as above.

### Tests

- **Zoom:** a command with a one-paragraph, a multi-line and a long wrapped `detail`, with and without colour and at two widths; a command without `detail` is unchanged.
- **JSON:** `detail` is present when set and absent when empty.
- **Placement:** `detail` is not in the summary, the expanded view or the group zoom.
- **Schema:** a string passes, a number fails with the path.
- **`check`:** the `registry-desc-long` fix text.

## Open questions

1. **Heading.** A **Details** heading, or the prose straight under the description with a blank line. Recommendation: the heading, so that a reader can tell the one-line summary from the longer text.
2. **Field name.** `detail`, `details`, `long` or `help`. Recommendation: `detail`: singular like `desc`, and not `help`, which is already a command.
3. **`generate` carrying `detail`.** Like `values`, a hand-added `detail` in an old overlay could survive regeneration. The overlay is personal and derived, and there is nothing to carry from. Recommendation: no.
4. **A rule for the opposite case.** A `detail` that only repeats `desc`. Recommendation: no; it is harmless and rare.
