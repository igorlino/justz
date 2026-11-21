# Checks and lint rules

`justz check` is about **documentation**: does every command have a description, and does it still match the recipes? `justz lint` is about the **recipes**: antipatterns visible in the structure of the justfile. Neither runs a recipe or edits your justfile.

| | `check` | `lint` |
|---|---|---|
| Findings | errors and warnings | warnings only |
| Exit `1` | on an error; on a warning with `--strict` | only with `--strict` |
| Use it | in pre-commit and CI | when you want advice |

Flags for both: `--json`, `--strict`, `--rule <name>` (repeat or comma-separate). `--rule` is for reading, not for gating CI: the counts and the exit code follow the shown findings, so it can hide errors and exit `0`.

Text output prints a fix once per rule. JSON keeps the fix on every finding.

## Silence one rule on one recipe

```just
[metadata('justz-ignore', 'recipe-long')]
build:
    ...
```

Needs `just` 1.42.0 or newer. An unknown rule name in `justz-ignore` is reported by `lint` as `ignore-unknown-rule`.

## `check` rules

### Errors

| Rule | Fires when |
|---|---|
| `doc-missing` | a recipe has no documentation |
| `group-missing` | a recipe has no `[group]` |
| `variadic-interpolation` | a variadic parameter is interpolated with `{{ }}`, which splits on whitespace (use `set positional-arguments` and `"$@"`) |
| `registry-missing` | a public recipe is absent from the registry |
| `registry-stale` | the registry names a recipe that no longer exists |
| `registry-desc-diverges` | the registry `desc` differs from the `[doc]`; the finding prints the exact line to paste |
| `registry-key-signature` | a registry key carries an argument signature instead of a bare recipe name |
| `registry-shape` | a field has the wrong type or nesting |
| `registry-unsorted` | the registry is not in canonical order (run `justz fmt`) |
| `registry-see-also` | `seeAlso` names a command that does not exist, or lists one twice |
| `registry-exits`, `registry-env` | an `exits` or `env` entry is malformed (an error) or questionable (a warning) |
| `example-bare-command` | an example is the bare command, with no flags or arguments, so it shows nothing |
| `example-single-flag` | an example only passes one boolean flag, which the argument rows already show |
| `example-long-flag` | an example uses a long flag that has a short alias |
| `example-enum-missing` | an argument declares `values`, but no example in the group uses it |

The four `example-*` rules are warnings for the `core` group.

### Warnings (fail only with `--strict`)

| Rule | Fires when |
|---|---|
| `example-flag-unknown` | an example uses a flag the recipe does not declare |
| `doc-multiline-comment` | a recipe doc is a comment block; `just` uses only the last line. Add `[doc(...)]` |
| `registry-desc-long` | a registry `desc` is over 120 characters |
| `registry-tags` | a tag list has a contradiction or a duplicate |
| `tags-confirm` | a recipe has `[confirm]` but its command does not declare `asks-first`, or the reverse. A script that asks by itself opts out with `justz-ignore`. |
| `default-public` | a public `default` recipe. It is a dispatcher, so rename it `_default` and keep it first. |
| `volatile-text` | a `desc` or `detail` has text that goes stale: TODO, FIXME, WIP, placeholder, "not implemented", "coming soon", a count such as "11 checks", a price, a date |
| `detail-flag-no-row` | a `detail` names a flag that no `args` row declares |
| `detail-env-no-row` | a `detail` names an environment variable with no `env` row. Caught: an upper-case name with an underscore (`NO_COLOR`), or one written as `$NAME`, `${NAME}`, `export NAME` or `NAME=`. Not caught: acronyms such as HTTP or JSON, flags, paths, file names, and names shorter than 3 characters such as `$CI`. A prose constant such as `UTF_8` can fire; silence it with `justz-ignore`. |
| `detail-long` | a `detail` is over 600 characters |
| `detail-no-paragraphs` | a `detail` is over 300 characters with no blank line. A deliberate list-style `detail` opts out with `justz-ignore`. |
| `detail-restates-desc` | a `detail` opens with the words of the `desc` (for a `desc` of four words or more) |

## `lint` rules

| Rule | Fires when |
|---|---|
| `recipe-long` | a recipe body is over 40 lines. The message counts the `{{ }}` interpolations, which decides whether moving it to a script is mechanical. |
| `module-shell-unset` | a module with linewise recipes sets no shell, so they run with `sh -cu` and no `pipefail`. Set a shell in the module. |
| `module-dispatcher-missing` | a module has no `default` recipe, so a bare `just <module>` runs a command |
| `module-dispatcher-lists-root` | a module `default` lists the root justfile |
| `secret-parameter` | a parameter has a secret-like name (`TOKEN`, `PASSWORD`, `SECRET`, `API_KEY`...): its value is typed on the command line, so it is in the process list |
| `secret-interpolated` | a secret-like name inside `{{ }}` in a linewise recipe: the value is in the shell's command line |
| `ignore-unknown-rule` | a `justz-ignore` value is not a rule |

The secret rules read names only. They never read, print or store a value.

The authoritative list for your version is `justz check --help` and `justz lint --help`.
