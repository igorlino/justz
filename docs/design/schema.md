# Design note: `justz schema`

Status: proposed.

## Purpose

Print the JSON Schema of the command registry (`commands.json`), so that an editor can complete and check the file as it is typed, and a CI job can check its shape with any JSON Schema tool.

```
justz schema
```

It prints the schema to standard output and exits `0`. It takes no flag except `--help`, needs no project and no `just`, and reads nothing from the working directory. `justz schema > commands.schema.json` writes the file.

An unknown argument exits `2`, as in the other commands.

## What the schema is for

The schema answers one question: is the file the right shape? It does not answer whether the registry matches the justfile. That stays in `justz check`, which needs the dump: `registry-missing`, `registry-stale`, `registry-desc-diverges`, `registry-key-signature`, the `example-*` rules and the description length. A schema cannot express any of those, and none of them moves into it.

What the shape check buys that nothing does today: `justz` reads the registry leniently. A field of the wrong type is skipped without a word (`"desc": 3` is an empty description, `"args": {}` is no arguments, a `commands` list instead of an object renders an empty group). An editor that knows the schema shows the error where it is typed.

## The schema describes what `justz` reads

The input model says the `commands.json` interface does not change in v1, and that unknown fields are kept and written back by `fmt`. So the schema is permissive:

- `additionalProperties` is not set to `false` anywhere. A field `justz` does not know is not an error.
- Only `groups` is required. Every other field is optional, as in the code (`FromRegistry` fails only when `groups` is missing or is not an object).
- It checks types and nesting, not meaning. No `required` on `desc`: an empty description is `check`'s finding, and a registry that is half written should still load in the editor.
- No `pattern` on group or command keys, no `enum` on argument `type`. A registry that `justz` accepts today must stay valid.

The fields, from `internal/help/model.go` and `internal/registry/canonical.go`:

| Level | Field | Type | Notes |
|---|---|---|---|
| root | `groups` | object of groups | required |
| root | `name`, `tagline`, `cli` | string | `cli` is the name shown in usage lines and the shim name |
| root | `title`, `subtitle` | string | aliases `justz` reads when `name` or `tagline` is missing |
| root | `tiers` | array of tier | |
| root | `order`, `dispatchers`, `lab` | array | `fmt` sorts them and `justz` reads nothing else, so no item type is given |
| root | `$schema`, `$comment` | string | allowed so that a file can name its schema |
| tier | `label`, `hint` | string | |
| tier | `groups` | array of string | |
| group | `label`, `title`, `desc`, `footer` | string | |
| group | `commands`, `extra` | object of command | `extra` holds entries that do not run as `just <group> <cmd>` |
| group | `examples` | array of `{cmd, desc}` | both strings |
| group | `seeAlso` | array of string | |
| command | `desc`, `example` | string | |
| command | `args` | array of argument | |
| argument | `name`, `desc` | string | |
| argument | `values` | array of string | an enum: drives completion and renders `(a\|b\|c)` |
| argument | `type` | string | present in real registries (`boolean`, `string`); `justz` does not read it, so it is typed but not an enum |

`schemaVersion` is not in the table. `docs/input-model.md` names it, but nothing reads it yet. A schema that lists a field that does nothing would mislead; it is added when the code reads it. Until then it is accepted like any unknown field.

## Delivery

- The file is `schema/commands.schema.json` (the layout in the plan). A small package `schema` next to it embeds it with `go:embed`, so the schema that `justz schema` prints is the one of the installed binary and is never fetched.
- `$schema` of the file is draft-07. Nothing here needs a later draft, and draft-07 is read by VS Code, JetBrains and `ajv` without a flag.
- `$id` is the raw URL of the file on the main branch of the repository. A registry may point at it with a `$schema` line to get completion in an editor. Because the schema is permissive and only grows, a moving URL is acceptable; someone who wants a fixed copy runs `justz schema > file`.
- The command is listed in the usage text and in the completion tables.

## Keeping it true

The schema is hand-written JSON, and the Go code that reads the registry does not mention it. Two tests keep them from drifting, without a new dependency:

1. **Valid inputs.** Every hand-written registry fixture the repository already has (`internal/registry/testdata/*.in.json`, `testdata/help-reference/demo/config/commands.json`, `testdata/variants/rich/config/commands.json`) and the registry `generate` writes must validate. The test uses a small validator in the test file that understands only `type`, `properties`, `items`, `required` and `additionalProperties`. A second test fails when the schema uses any other keyword, so the validator cannot skip one without notice.
2. **Wrong shapes.** A table of broken registries (`desc` as a number, `args` as an object, `groups` as a list, `values` with a number in it) must each fail, each with a message that names the path.

A third check is by hand and part of review: when a field is added to `internal/help/model.go`, the table above is the place to update. The note says so, because a test that derives the fields from Go code would cost more than the drift it prevents.

## Not in scope

- Running the schema inside `justz check`. It would need a JSON Schema validator (a dependency, or a second hand-written one that must agree with the file), and the findings would repeat what an editor shows. A CI job that wants the check can run `justz schema > s.json` and any validator such as `check-jsonschema`.
- A schema for `.just-local-rich.json`. It is generated and never hand-edited.
- A schema for the justfile dump.

## Plan

1. `schema/commands.schema.json`, the `schema` package, `justz schema`, completion entries and the tests above. One PR.
2. At release time: the README, the `justz-cli` skill and the `justfile-authoring` reference (`variant-rich.md`, "schema quick reference") say that the schema exists and show the `$schema` line. These describe released behaviour, so they wait for the release that ships the command.

## Open questions

1. **Should `justz init --variant rich` write a `"$schema"` line?** New files would get editor completion for free, and `fmt` keeps unknown fields. The cost is a URL in every scaffolded registry, and a changed output that the init tests pin. Recommendation: not in v1. The skill shows the line; add it to `init` if the pilot shows people want it.
2. **Aliases `title` and `subtitle`.** `justz` reads them, no documentation mentions them, and the real registries use `name`. Recommendation: list them in the schema as they are read, and say in the description that `name` and `tagline` are the documented names.
3. **`order`, `dispatchers`, `lab`.** They come from the large private registry in the plan's audit; `fmt` sorts them and nothing reads them. Recommendation: type them as arrays with no item type, and ask at the M9 pilot whether they belong in the registry at all.
