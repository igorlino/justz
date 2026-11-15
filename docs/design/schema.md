# Design note: `justz schema`

Status: implemented in one PR (the schema file, the schema package, the command, completion and the tests). The open questions are decided as recommended.

## Purpose

Print the JSON Schema of the command registry (`commands.json`), so that an editor can complete and check the file as it is typed, and a CI job can check its shape with any JSON Schema tool.

```
justz schema
```

It prints the schema to standard output and exits `0`. It takes no flag except `--help`, needs no project and no `just`, and reads nothing from the working directory. `justz schema > commands.schema.json` writes the file.

An unknown argument exits `2`, as in the other commands.

## What the schema is for

The schema answers one question: is the file the right shape? It does not answer whether the registry matches the justfile. That stays in `justz check`, which needs the dump: `registry-missing`, `registry-stale`, `registry-desc-diverges`, `registry-key-signature`, the `example-*` rules and the description length. A schema cannot express any of those, and none of them moves into it. The type mistakes the schema describes are also reported by `justz check` as `registry-shape` (an error that names the command key and the field). `check` reads the embedded schema itself, so the two agree on what is valid, and unknown fields stay allowed.

What the shape check buys that nothing does today: `justz` reads the registry leniently, and a field of the wrong type is mostly skipped without a word. Measured: `"desc": 3` is an empty description, `"args": {}` is no arguments, a `commands` list instead of an object renders an empty group, a group that is not an object is dropped, and a non-string in `values` is dropped. An item of the wrong type inside `args` or `examples` (`[1, "s"]`) is not dropped: it renders a blank row. A `groups` that is not an object is the one loud failure (`help` exits 1). An editor that knows the schema shows each of these where it is typed.

## The schema describes what `justz` reads

The input model says the `commands.json` interface does not change in v1, and that unknown fields are kept and written back by `fmt`. So the schema is permissive:

- `additionalProperties` is not set to `false` anywhere. A field `justz` does not know is not an error.
- The root must be an object and `groups` is the only required field. Every other field is optional, as in the code: `FromRegistry` fails only when the root is not an object, or `groups` is missing or is not an object. `fmt` is more lenient still and accepts `{}`, but a registry with no `groups` renders nothing, so the schema reports it.
- It checks types and nesting, not meaning. No `required` on `desc`: an empty description is `check`'s finding, and a registry that is half written should still load in the editor.
- `null` is a wrong type like any other: `justz` reads a null field as absent, and the schema reports it, since a `null` where text belongs is almost always a mistake.
- No `pattern` on group or command keys, no `enum` on argument `type`. A registry that `justz` accepts today must stay valid.

The fields, from `internal/help/model.go` and `internal/registry/canonical.go`:

| Level | Field | Type | Notes |
|---|---|---|---|
| root | `groups` | object of groups | required |
| root | `name`, `tagline`, `cli` | string | `cli` is the name shown in usage lines and the shim name |
| root | `title`, `subtitle` | string | aliases `justz` reads when `name` or `tagline` is missing |
| root | `tiers` | array of tier | |
| root | `order`, `dispatchers`, `lab` | array | `fmt` sorts them (`order` with `core` first, the others A to Z) and `justz` reads nothing else, so no item type is given |
| root | `$schema`, `$comment` | string | `$schema` lets a file name its schema; `$comment` is the header `justz generate` writes, so a generated registry must validate |
| tier | `label`, `hint` | string | |
| tier | `groups` | array of string | |
| group | `label`, `title`, `desc`, `footer` | string | |
| group | `commands`, `extra` | object of command | `extra` holds entries that do not run as `just <group> <cmd>` |
| group | `examples` | array of `{cmd, desc}` | both strings |
| group | `seeAlso` | array of string | |
| command | `desc`, `example` | string | |
| command | `detail` | string | optional longer text, shown only in the zoom (added with 0.7.0) |
| command | `seeAlso` | array of string | keys of related commands, `group.command` (`core.build` for a top-level one); see `docs/design/see-also.md` |
| command | `args` | array of argument | |
| argument | `name`, `desc` | string | |
| argument | `values` | array of string | an enum: drives completion and renders `(a\|b\|c)` |
| argument | `type` | string | present in real registries (`boolean`, `string`); `justz` does not read it, so it is typed but not an enum |

`schemaVersion` is not in the table. `docs/input-model.md` names it, but nothing reads it yet. A schema that lists a field that does nothing would mislead; it is added when the code reads it. Until then it is accepted like any unknown field.

## Delivery

- The file is `schema/commands.schema.json` (the layout in the plan). A small package `schema` next to it embeds it with `go:embed`, so the schema that `justz schema` prints is the one of the installed binary and is never fetched. The package sits at the repository root, not under `internal/`, so it is importable (`github.com/igorlino/justz/schema`). That is intended: a Go tool that wants the schema can embed it, and the surface is one exported string. The release build needs no change, since the file is compiled into the binary.
- `$schema` of the file is draft-07. Nothing here needs a later draft, and draft-07 is read by VS Code, JetBrains and `ajv` without a flag.
- `$id` is the raw URL of the file on the main branch of the repository. A registry may point at it with a `$schema` line to get completion in an editor. The URL follows `main`, so the rule is: a change to the schema may only make it accept more. A change that would reject a registry accepted before needs a new file name (`commands.v2.schema.json`), never an edit of this one. Someone who wants a fixed copy runs `justz schema > file`, or uses a release-tag URL. The raw URL works only while the repository is public, which it is. Listing the schema in SchemaStore, which would match `commands.json` in editors with no `$schema` line, is out of scope: it is a separate submission and waits for the schema to settle.
- The command is listed in the usage text and in the completion tables.

## Keeping it true

The schema is hand-written JSON, and the Go code that reads the registry does not mention it. Two tests keep them from drifting, without a new dependency:

1. **Valid inputs.** Every hand-written registry fixture the repository already has (`internal/registry/testdata/*.in.json` and `full.golden.json`, `minimal.golden.json`, `testdata/help-reference/demo/config/commands.json`, `testdata/variants/rich/config/commands.json`) and the registry `generate` writes must validate. `empty.in.json` and `empty.golden.json` are `{}` and are listed with the wrong shapes instead, because `groups` is required. The test uses a small validator in the test file that understands `type`, `properties`, `items`, `required`, and `additionalProperties` both as a boolean and as a schema (the maps of groups and commands have free keys, so their value shape is an `additionalProperties` schema), plus `$ref` into `definitions` so that `commands` and `extra` share one command shape. It reports the path of the first failure. A second test fails when the schema uses any other keyword, except the annotations `$schema`, `$id`, `$comment`, `title` and `description`, so the validator cannot skip a constraint without notice.
2. **Wrong shapes.** A table of broken registries (`{}`, a root that is a list, `desc` as a number, `args` as an object, an `args` entry that is a string, `groups` as a list, `values` with a number in it) must each fail, each with a message that names the path.

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
