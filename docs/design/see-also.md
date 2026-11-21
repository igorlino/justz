# Design note: a per-command `seeAlso`

Status: implemented in one PR (the field, the zoom line, the JSON field, the schema property, the `check` rule, the `fmt` order and the tests). The open questions are decided as recommended.

## Problem

A group has `seeAlso`, a list of related groups. A command has nothing like it. When one command is related to another, the author writes it in prose in `detail` or `desc`: "mirrors", "sibling of", "alias for". About 14 of the 149 long details of a large registry do this. The relation has no field, so the help cannot show it as a line, `help -j` cannot give it to an agent, and `check` cannot tell that the command it names was renamed or removed.

## Proposal

An optional array of strings `seeAlso` on a command, each the key of another command.

```json
"deploy-staging": {
  "desc": "Deploy the report to staging",
  "seeAlso": ["ops.deploy", "core.build"]
}
```

**Keys only.** A relation word ("mirrors", "alias for") was considered and left out. The word is prose, so it cannot be checked or sorted, and it grows into a vocabulary the schema would have to fix. A reader who wants to say how two commands relate says it in `detail`, next to a field that now names the same command in a form tools can follow. An object form can be added later without breaking this one, because a schema that accepts strings today can accept strings or objects tomorrow.

**The key form is `group.command`**, the form `check` already uses as the subject of a finding (`tool.clean`). A top-level command has the group `core`: `core.build`. The key is matched as a whole against the group key, a dot and the command key, so a group or command key with a dot in it still resolves, and nothing parses the string. The match is exact: it is case and whitespace sensitive, and `Tool.clean`, or `tool.clean` with a trailing space, is not `tool.clean`. A command in `extra` can be named too: it is documented, it is in the registry, and a reader may need to find it.

## Where it shows

| View | Shows `seeAlso`? |
|---|---|
| `justz help <group> <command>` (the zoom, and so `--browse`) | yes, a **See also** line |
| `justz help <group> <command> -j` | yes, as `seeAlso` (left out when empty) |
| Summary, `--expanded`, group zoom, group `-j` listing, completion | no |

The keys are listed once each (the first), without a reference to the command itself, and sorted by key before they are written, so the zoom and the JSON show the same order. The line is at the end of the zoom, after Details, in the style of the group line: `See also: dm ops deploy, dm build`. A key that names a command is written as the invocation (`core.build` is `dm build`, `ops.deploy` is `dm ops deploy`); a key that matches no command in the help is written as it is, so a stale reference stays visible. The keys are sorted, as the group line sorts its own. When the group has `seeAlso` too, its line follows directly under the command's. A command without `seeAlso` renders byte for byte as before, so the goldens do not change.

## What else changes

- **Schema.** `seeAlso` is added to the `command` definition as an array of strings. It only makes the schema accept more. The registry-shape check of `justz check` reads the same schema and so checks the type with no change.
- **`check`.** A new error, `registry-see-also`, for a key that is not `group.command` of an entry in the registry (a command or an `extra` entry), for a key that names the command itself, and for a key listed twice. It is an error, like `registry-stale`, the other finding about a registry key that points at nothing. The subject is the command (`tool.clean`), the message names the key, and the fix is the same for every finding: use the key of another command, as `group.command`, or remove it.
- **`fmt`.** It sorts the list A to Z, as it does for a group's `seeAlso`, and keeps every other part of the author's order.
- **Terminal safety.** The keys are project text, so the zoom and the JSON clean them like the group's.
- **Completion, `generate`, native variant.** Untouched. The overlay is derived from the justfile and has no source for the field.
- **Older `justz`.** A version that does not know the field ignores it and keeps it in `fmt`.

## Not in scope

- A relation word or any other structure for a reference.
- A group in a command's `seeAlso`. The group line already exists on the group.
- Reverse links ("referenced by"). They can be derived, and nothing needs them yet.
- A rule for prose such as "mirrors" in `detail`. `volatile-text` and `detail-flag-no-row` are the only rules about prose.

## Plan

1. The model, the sanitizer, the zoom line, the JSON field, the schema property, the `registry-see-also` rule and the `fmt` order, with tests. One PR.
2. The docs: the README, `docs/design/schema.md` (its field table), the `justz-cli` skill and the variant-rich reference (when to use `seeAlso` instead of "mirrors" or "sibling of" in prose).

### Tests

- **Zoom:** a command with `seeAlso` (the line, the sorting, an unknown key written as it is, the group line under it, colour stripped equals plain), and a command without it unchanged (the existing goldens).
- **JSON:** `seeAlso` present when set and absent when empty.
- **Placement:** not in the summary, the expanded view or the group zoom.
- **`check`:** a key that exists (a command and an `extra` entry), one that does not, one that names the command itself, a duplicate, a type mistake (reported by `registry-shape`), and the rule in `Rules`.
- **`fmt`:** the list is sorted.

## Open questions

1. **Keys only, or keys with a relation word.** Recommendation: keys only (see above).
2. **Key form.** `group.command`, or a path with a space as in a command line. Recommendation: `group.command`: it is one token, it is what `check` prints, and a space is a separator in the arrays of `args`.
3. **Error or warning.** Recommendation: an error, as for `registry-stale`.
4. **Where in the zoom.** Last, with the group line, or straight under the description. Recommendation: last: it is the pointer to the next read, and the actionable text stays first.
