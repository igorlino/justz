# Design note: structured fields for agents (`hasDetail`, `exits`, `env`, `tags`)

Status: all three steps of the plan are implemented (`hasDetail` and `exits`, `env` rows, `tags`). Step 1 follows the recommended answers to open questions 1 to 3; steps 2 and 3 follow the answers to questions 4 to 11 that the maintainer accepted. The advice rule that a `detail` names a variable without an `env` row (question 6) is a remaining follow-up. In `check`, the contradiction and duplicate rule is `registry-tags` and the `[confirm]` rule is `tags-confirm`, both warnings; `tags-confirm` can be silenced per recipe with `justz-ignore`, and an `extra` entry, which has no recipe, gets no `tags-confirm`.

## Problem

An agent that reads `justz help -j` gets a command's `desc`, its `detail`, its argument rows and its examples. Some facts that an agent acts on have no field. They are in `detail` as prose, or nowhere. On a large registry, about a third of the details carried structured information of three kinds: environment variables and prerequisites, cost, quota or a confirmation, and exit codes. A person reads these in the zoom. An agent has to parse them out of a sentence, and cannot tell a command that spends money from one that only mentions it.

Four candidates were put forward. Each is an optional field, so a registry that does not use it renders and validates as before. Every field follows the rules the other notes already set:

- **The schema only accepts more** (`docs/design/schema.md`). A field is added as a property, never as a stricter type or an enum on an existing one, so a registry that is valid today stays valid.
- **The JSON is a contract: add fields, do not rename or remove them** (`internal/help/json.go`). A reader must ignore fields it does not know.
- **Unknown fields stay allowed, and `fmt` keeps them.** An older `justz` ignores a new field and keeps it.
- **A command without the field renders byte for byte as before**, so the goldens do not change.
- **Text from the registry is project text.** It goes through the terminal-safe cleaning like `desc` and `detail`.

## 1. `hasDetail` in the group listing

**Shape.** No registry field. The group node of `help -j` lists its commands as `{name, desc}` (`CommandSummary`). Each entry gains a boolean `hasDetail`, true when the command has a non-blank `detail` (the same test that decides whether the zoom prints a **Details** section).

```json
{ "name": "deploy", "desc": "Deploy the report to the cloud host", "hasDetail": true }
```

**Why.** The group listing leaves `detail` out on purpose, so that listings stay small and an agent zooms per command. Without a flag the agent cannot tell which zoom is worth a call, and zooms all of them or none.

**Shown.** In `help <group> -j` only. The text views do not change. The array printed for an ambiguous name is a list of commands, not of summaries, and is not changed.

**`check`.** Nothing to verify: the value is derived from the registry.

**Schema, `fmt`.** Nothing: it is output, not registry. **Compatibility.** An added field on an array element is allowed by the contract. A reader that ignores it still works.

## 2. `exits`

**Shape.** An optional object on a command, from an exit code to what it means:

```json
"check": {
  "desc": "Check that the docs match the justfile",
  "exits": { "0": "no errors", "1": "an error, or a warning with --strict", "2": "usage error" }
}
```

The keys are strings, because JSON object keys are. A key is a non-negative integer written in decimal, 0 to 255. The values are one line of text. It works on `commands` and on `extra` entries.

**Shown.** The zoom prints an **Exit codes** section after Options and before Details, one row per code in numeric order, at the same indentation as Options. `help <group> <command> -j` carries `exits` as the same object, left out when empty. The listings do not carry it.

**`check`.** It can verify the shape of each key and nothing about the meaning:

- A key that is not an integer from 0 to 255 is an error (`registry-exits`), like `registry-shape` for a wrong type. The schema cannot say this, since the schema never uses `pattern` on keys (`docs/design/schema.md`), so it only says object of strings; the key rule lives in `check`, and `check` says so in the fix text.
- It cannot tell whether the recipe really exits with the code. A stale exit code is as stale as stale prose, and the review note in the skill applies: check the table against the script.

**Schema.** `exits`: object, `additionalProperties` a string. **`fmt`.** Sorts the keys by numeric value, as it sorts `args`. The text stays as written.

## 3. Environment variables as rows

**Shape.** An optional array `env` on a command, with rows like `args`:

```json
"deploy": {
  "env": [
    { "name": "DEPLOY_LEGACY_STATIC", "desc": "Also copy the legacy static files", "default": "unset" },
    { "name": "STORAGE_HOST", "desc": "Host of the storage box", "required": true }
  ]
}
```

A row has `name` (a string), `desc` (a string), and optionally `required` (a boolean) and `default` (a string, as text: it is shown, not parsed). It is separate from `args` because a flag and a variable are read differently: one is typed in the command line and the other is set around it. Rows that carry `values` for an enum are left out of this first shape.

**Shown.** The zoom prints an **Environment** section after Options (and after Exit codes when both exist): `NAME`, then the description, with `(required)` or `(default: ...)` after it. `help -j` carries `env` as rows with the same fields, left out when empty.

**`check`.** What can be derived from the justfile is little. The dump has `export` assignments and `env('NAME')` calls in a recipe body, but a script that the recipe runs reads its own variables, so the absence of a name from the body proves nothing. So:

- Shape: a row without a `name`, a duplicate name and a name that is not an identifier of upper case letters, digits and underscores are findings (`registry-env`, a warning for the name pattern, an error for the rest).
- Advice rule `detail-env-no-row` (implemented as the follow-up, a warning, in the style of `detail-flag-no-row`): a `detail` that names an upper case variable that has no row. To keep the noise low it matches only a name with an underscore, or one written as `$NAME`, `${NAME}`, `export NAME` or `NAME=`; acronyms, flags, paths and file names do not count. It has the opt-out that `volatile-text` has: `[metadata('justz-ignore', 'detail-env-no-row')]`.
- It cannot verify that the command reads the variable.

**Schema.** `env`: array of rows with the four properties. **`fmt`.** Sorts rows by `name`, as `args` is sorted by name. **Risk.** A second place that says what a flag-like input is; the guidance in the skill has to say which goes where.

## 4. `tags`

**Shape.** An optional array of strings on a command, for what the command touches:

```json
"deploy": { "desc": "...", "tags": ["asks-first", "production", "spends"] }
```

**A free array, not an enum.** The schema rule is that a later version may only accept more. An enum fixed today would make a registry that uses a tag added next year invalid for the version that wrote the enum, so the schema says array of strings. The vocabulary is a recommendation in the `justfile-authoring` skill, which can grow without a release of `justz`. A reader treats a tag it does not know as unknown, not as an error.

**Recommended vocabulary.**

| Tag | Meaning |
|---|---|
| `read-only` | Changes nothing outside the working files: no write to a service, a host or a database |
| `idempotent` | Running it again has the same effect as running it once |
| `destructive` | Deletes or overwrites something that cannot be recovered from the command alone |
| `production` | Touches a production system |
| `spends` | Costs money or uses a quota |
| `asks-first` | Stops to ask for a confirmation before it acts |
| `long-running` | Takes minutes, not seconds |
| `needs-terminal` | Needs an interactive terminal (a prompt, a pager, a login) |

**Absence never means safe.** A command with no `tags`, or without `read-only`, is unknown, not harmless. An agent must not read a missing tag as a permission. That is why `read-only` is a tag of its own: a command that is safe says so, and the lack of the claim stays a lack. The skill says this in one sentence, and the JSON contract does the same: `tags` is left out when empty, and an empty field means "not declared".

**Shown.** The zoom prints one line under the description, `Tags: asks-first, production, spends`, in the order of the registry after `fmt` (A to Z). `help -j` carries `tags` on the command, left out when empty. The listings do not carry it.

**`check`.** A tag is a claim that nothing can verify, so a stale tag is worse than stale prose: prose is read with doubt, and a tag is acted on. What `check` can do:

- **`asks-first` from `[confirm]`.** just's `[confirm]` attribute is in the dump of the recipe (`{"confirm": null}` or `{"confirm": "Really?"}`). A command whose recipe has it should declare `asks-first`, and a command that declares `asks-first` for a recipe with no `[confirm]` is a finding, since a script may also ask by itself. The recommendation is that `check` keeps the two in step in both directions, rather than `help` deriving the tag: `help` reads the registry and does not run `just` for a `rich` project, and a declared tag keeps the JSON complete without it.
- **Contradictions.** `read-only` together with `destructive`, `spends` or `production` is a warning. A duplicate tag is a warning.
- **Nothing else.** `spends`, `production`, `destructive`, `idempotent`, `long-running` and `needs-terminal` cannot be derived from the recipe. `check` does not guess, and does not warn on an unknown tag: a free array means tags beyond the vocabulary are allowed.

**Schema.** `tags`: array of strings. **`fmt`.** Sorts the array A to Z, as `seeAlso`. **Risk.** The cost of a wrong tag falls on the agent that trusted it. The mitigation is the review step in the skill, and the fact that the first step ships no tag with a safety meaning.

## Where each shows

| Field | Zoom | `help <group> <command> -j` | `help <group> -j` | Summary, expanded, group zoom, completion |
|---|---|---|---|---|
| `hasDetail` | no | no | yes, per command | no |
| `exits` | **Exit codes** section | yes | no | no |
| `env` | **Environment** section | yes | no | no |
| `tags` | **Tags** line | yes | no | no |

The order in the zoom: invocation, `desc`, Tags, Examples, Options, Exit codes, Environment, Details, See also. The actionable text stays first, as in `docs/design/detail.md`.

## Not in scope

- Deriving any tag other than `asks-first`, and reading scripts to find variables or exit codes.
- An enum for tags, or a per-project vocabulary file.
- Groups: the fields are on commands (and `extra` entries).
- Markup in any of the new text.
- `generate`: the overlay is derived from the justfile and has no source for these fields, so it never writes one.
- Native variant: nothing, as for `detail`.

## Plan

Three steps, in this order, because the first is small and the last carries the risk.

1. **`hasDetail` and `exits`.** The listing field, the `exits` field with its zoom section, JSON, schema property, `registry-exits` rule and `fmt` order, with tests. Small: both are concrete, and neither says anything about safety. Risk: low; the key rule lives in `check`, not in the schema.
2. **`env` rows.** The field, the section, JSON, schema, shape rules and `fmt` order. Risk: moderate. It overlaps `args` and `detail`, so the skill guidance has to be written with it, and the check can verify little.
3. **`tags`.** The field, the line, JSON, schema, the `[confirm]` rule, the contradiction warning and the vocabulary in the skill. Risk: the highest. The fields in steps 1 and 2 are descriptions; a tag is a claim an agent acts on, and `check` can verify one of eight. It ships last, with the skill text that says absence is not safety.

Each step updates the README, the `justz-cli` skill, the variant-rich reference and `docs/design/schema.md` (its field table), and keeps the JSON contract text in the README: the fields are additive and `help <group> <command> -j` stays the supported way to read them.

### Tests

- **Zoom and JSON:** a command with the field and one without (unchanged goldens), the section order, left out when empty, colors only add escape codes, a control character in a text is cleaned.
- **Placement:** not in the summary, the expanded view, the group zoom or the group listing (except `hasDetail`).
- **Schema:** a valid value passes, a wrong type fails with the path (and the registry-shape guard test still passes).
- **`check`:** each rule has a hit, a miss and, where it exists, the ignore. The `[confirm]` rule has both directions.
- **`fmt`:** the ordering of each new field, and that unknown fields survive.

## Open questions

These were recommendations for the maintainer. The maintainer accepted questions 4 to 11 as recommended; questions 1 to 3 were implemented as recommended.

1. **`hasDetail` always present, or only when true.** Recommendation: always present, so a reader can tell "no detail" from "an older `justz`".
2. **`exits` keys.** Integers only, or also a word such as `other`. Recommendation: integers only, 0 to 255; it can be extended later.
3. **Must `exits` list `0`?** Recommendation: no. The field says what is worth knowing.
4. **`env` as its own array, or as rows in `args`.** Recommendation: its own array `env` with `name`, `desc`, `required`, `default`.
5. **Should `check` try to verify `env` against the recipe body?** Recommendation: no, only shape; a script reads its own variables.
6. **The `detail`-names-a-variable advice rule.** Recommendation: a follow-up after step 2, with the opt-out.
7. **Where `Tags` shows in the zoom.** Recommendation: one line under the description, before the examples, so a reader sees a warning before the command line they copy.
8. **`asks-first`: declared and kept in step by `check`, or derived by `help`.** Recommendation: declared, with `check` enforcing both directions.
9. **Is `read-only` ever required?** Recommendation: never required; the skill says absence is unknown.
10. **Should `check` warn on a tag outside the vocabulary?** Recommendation: no, for a free array; warn only on contradictions and duplicates.
11. **The release the fields ship in, and whether steps ship together.** Recommendation: steps 1 and 2 may ship together; step 3 waits for the review of the skill text.
