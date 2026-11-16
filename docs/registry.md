# The command registry

The registry, `config/commands.json`, is where the `rich` variant keeps documentation that does not fit in a justfile attribute: examples, longer text, exit codes, environment variables and tags. `justz help`, `check`, `fmt` and `completions` read it together with the justfile.

Editor support: `justz schema > commands.schema.json`, or add this at the top of the file:

```json
"$schema": "https://raw.githubusercontent.com/igorlino/justz/main/schema/commands.schema.json"
```

The schema checks shape only. Whether the registry matches the justfile is `justz check`'s job.

## Minimal example

```json
{
  "groups": {
    "core": {
      "label": "Core",
      "commands": {
        "test": {
          "desc": "Run the test suite",
          "args": [
            { "name": "-k <pattern>", "desc": "Filter tests by name pattern" },
            { "name": "-w, --watch", "desc": "Re-run on file changes" }
          ]
        }
      }
    }
  }
}
```

`justz init --variant rich` writes a working registry. A larger demo is in `testdata/help-reference/demo/config/commands.json`.

## Fields of a command

| Field | Purpose | Shown as |
|---|---|---|
| `desc` | One line: what the command does. | The listing, completion, and the zoom heading. |
| `detail` | The few facts that do not fit in `desc`: what it needs, what it changes. Not steps. | **Details** in the zoom. With colors on, `` `command spans` ``, flags, `<placeholders>` and URLs are picked out. Listings stay one line each. |
| `args` | Arguments and flags: `name`, `desc`, optional `type` and `values` (for an enum). | **Options** |
| `exits` | Map from exit code to meaning. | **Exit codes**; `exits` in JSON |
| `env` | Environment variables the command reads: `name`, `desc`, `required`, `default`. A flag goes in `args`, a variable in `env`. | **Environment**; `env` in JSON |
| `tags` | Free words for what it touches: `read-only`, `idempotent`, `destructive`, `production`, `spends`, `asks-first`, `long-running`, `needs-terminal`. | **Tags** line; `tags` in JSON |
| `seeAlso` | Related commands by key (`group.command`). | **See also**; verified by `justz check` |

A command with no `tags` is not declared safe. `justz check` keeps `asks-first` in step with the recipe's `[confirm]` and warns on a contradiction or a duplicate.

## Read a command's text from a script or a test

`justz help <group> <command> -j` is the supported way to read what a command is documented as. **`desc` plus `detail` is the text of a command.** A sentence can move from `desc` to `detail` without any change to the command, so code that asserts a phrase in `desc` alone breaks when it does.

```sh
justz help core test -j | jq '{desc, detail}'
```

```json
{
  "desc": "Run the test suite",
  "detail": "Runs only the unit tests; integration tests need `test --integration`.\n\nNeeds no network."
}
```

Reading `config/commands.json` directly is not part of the contract: its layout can change, and a native project has no such file.

## Group fields

`label` (listing heading), `title`, `desc`, `commands`, `examples` (`cmd` and `desc` pairs) and `seeAlso` (other groups, by name). Top level: `cli` (the shim name), `name` (the help heading) and `tiers` (which groups are listed under a labeled heading, with a `hint`). The full list is in `justz schema`.

## Keep it sorted

`justz fmt` writes the registry in one stable order, so diffs stay small. Run `justz fmt --check` in CI.

## What goes in `detail`

Write what the reader cannot see from the name and flags. Do not restate `desc`, list steps, or write text that goes stale (a TODO, a count, a price, a date). `justz check` warns on these: see [Checks](checks.md). The [`justfile-authoring` skill](../skills/justfile-authoring/references/variant-rich.md) says what goes in each field.
