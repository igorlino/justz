# The `native` variant — documentation in the justfile

Everything lives in the justfile as `just` attributes. There is **no
`commands.json`**, no `desc` duplication, and **zero vendored helper scripts**.
`just --dump --dump-format json` is the data interface: structured, queryable,
and identical in every `just` project.

All behaviour below was verified on **just 1.58.0** and `[arg]` and
module behaviour re-checked on **1.59.0**. Check `just --version` before promising any of it.

## 1. Annotate every recipe

```just
[group('core')]
[doc('Compile, run tests and assemble jars (./gradlew build)')]
build:
    ./gradlew build
```

- `[doc('…')]` is the one-line description. Imperative, specific, no trailing
  period. It is what `just --list`, `--show`, `--dump`, shell completions and the
  IDE extension all show — one source, so they cannot disagree.
- `[group('…')]` sections the listing. Top-level recipes need it; recipes inside
  a `mod` submodule do not (the module name already groups them).
- `mod` statements take both attributes, which is how namespaces get a section
  and a one-line summary:

  ```just
  [group('commands')]
  [doc('Local development: bootstrap, run, ui, watch, status, logs, stop')]
  mod local 'just/local.just'
  ```

  An **ungrouped** `mod` is listed above every group heading, so group them if
  placement matters.

## 2. Ordering: `--unsorted` is mandatory

`just --list` sorts recipes *and* groups alphabetically. With `--unsorted` both
follow source order, so `build → test → check → format → ci` survives. There is
no `set unsorted`, so the flag must come from the wrapper recipe — which is the
reason the `native` variant has one.

## 3. The listing recipes

```just
# Private, so it stays out of its own output. `just` with no arguments runs the
# first recipe in the file, so this must be first.
_default:
    @just --unsorted --list-heading $'\n  <project> — <tagline>\n\n' --list

[group('core')]
[doc('Expand every namespace in one listing (bare `just` keeps them collapsed)')]
help:
    @just --unsorted --list-submodules --list-heading $'\n  <project> — every recipe\n\n' --list
```

- Use `_default`, never `default` — a public `default` shows up as a row in
  every listing.
- `$'…'` (bash ANSI-C quoting) is how the heading keeps its newlines.
  `--list-heading "$(printf '…\n')"` does **not** work: command substitution
  strips the trailing newline and the first group heading lands on the same line.
- `--list-heading ''` suppresses just's own `Available recipes:`.

Each submodule gets the same treatment so a bare namespace lists instead of
running something:

```just
_default:
    @just --unsorted --list <ns>
```

`just <mod>` with no `default` recipe **runs the module's first recipe**. That is
the footgun this prevents.

Flag order matters: `just --list --unsorted <ns>` fails with
`the argument '--list [<MODULE>...]' cannot be used with '[ARGUMENTS]...'`,
because `<ns>` is parsed as a recipe argument. Write `just --unsorted --list <ns>`.

## 4. Arguments: `[arg]`, not prose

`[arg]` turns a parameter into a documented, validated CLI option:

```just
[doc('Build the filterable HTML gallery from the screenshots')]
[arg('all',    long='all',    short='a', flag, help='Also convert to webp and bundle a zip')]
[arg('bundle', long='bundle', short='b', flag, help='Zip the gallery for reviewers')]
gallery all bundle:
```

```
$ just ui gallery -a -b        # parsed, not forwarded as a string
$ just ui gallery --nope
error: recipe `gallery` does not have option `--nope`
```

**Short flag rule:** Proactively provide `short='x'` for all flag and option
parameters whose name is 3 or more letters long. Do not leave them long-only.
**2-letter parameter exemption & UX/DX:** Aliases exist to serve UX/DX. Parameters
with 2 letters (e.g. `long='id'`, `long='ci'`) stay long-only by default because
they are already minimal. If a specific UX/DX reason suggests aliasing a 2-letter
parameter, ask the user.

This is strictly better than documenting a `*ARGS` string that gets forwarded
blindly. Syntax rules, each one a distinct error if broken:

| Rule | Error if violated |
|---|---|
| Attribute keys use `=`, not `:` | `expected ',', '=', or ')'` |
| `flag` takes no value — write bare `flag`, not `flag=true` | ``flag`` attribute … takes no value` |
| `flag` requires `long=` or `short=` | `only valid with 'long' or 'short'` |
| A `flag` parameter may not have a default | `flag parameter 'all' may not have a default` |
| Boolean `flag` args require `set lists` | ``flag`` arguments require `set lists`` |

`[arg(help='…')]` also documents **positional and variadic** parameters, and
those need no settings at all:

```just
[doc('Zip the gallery for sharing')]
[arg('name', help='Subject for the file name (ticket, release)')]
bundle name='':

[doc('Report catalogs that need isolation')]
[arg('flags', help="Forwarded verbatim to 'toolkit isolate check' (e.g. --json)")]
check *flags:
```

### `set unstable` + `set lists` — scope it, and know what it changes

Only boolean `flag` parameters need them. Options that take a value, and `help`
on positional or variadic parameters, do not. Settings are **module-scoped**, so
opt in only in the submodule that needs it and leave the root justfile clean.

`set lists` is an unstable feature and it changes semantics you already rely on:
a dependency argument becomes a **list**, and list values interpolate by
*concatenation*. A mapped dependency that worked before starts producing
garbage:

```just
# Before: sends `--diff --json`.  Under `set lists`: sends `--diff--json`.
diff *flags: (transform "--diff" flags)
```

Rewrite such a recipe to call the underlying command directly, and leave a
comment saying why. Variadic arguments passed on the *command line* are
unaffected. If `set lists` is ever withdrawn, boolean flags revert to positional
parameters — say so in a comment next to the settings.

## 5. Long-form documentation

`just --list` shows either the whole `[doc]` or nothing; it cannot show the first
line of a multiline doc (casey/just#2684). Pick per repo:

**Single-line `[doc]` + plain `#` comments above the recipe.** The listing stays
aligned; the long form is readable only by opening the file — `just --show` prints
attributes and body, not preceding comments. Good default when the prose is for
maintainers.

**Multiline `[doc]`.** The full text becomes reachable:

```just
[doc("""
Build the filterable HTML gallery from the screenshots

Needs screenshots on disk: run 'just ui capture' first.
example: just ui gallery --all | rebuild, convert to webp and zip it
see: just ui capture
""")]
```

```
$ just --explain --dry-run ui::gallery      # prints the whole doc, runs nothing
```

Cost: `just --list` prints that block above the recipe name and drops its inline
description, for that recipe and any other with a multiline doc. Use it where the
prose earns the space, not everywhere.

Wrap the reader in a recipe so it is discoverable:

```just
[group('core')]
[doc('Print the full documentation for one recipe (e.g. just explain local::ui)')]
explain recipe:
    @just --explain --dry-run {{ recipe }}
```

Module recipe paths use `::` — `local::ui`, not `local ui`.

## 6. Examples and cross-references stay data

Put them in the multiline `[doc]` behind a prefix, one per line (always prefer
short flag aliases in examples):

```
example: just i18n pull -b feature/ABC-123 | pull strings from a branch
see: just ui
```

They remain queryable — the dump is the interface, so no metadata file is needed:

```bash
just --dump --dump-format json | jq -r '
  .modules.ui.recipes | to_entries[]
  | .key as $r | (.value.doc // "") | split("\n")[]
  | select(startswith("example:")) | ltrimstr("example:") | split("|")
  | "\($r): \(.[0]|ltrimstr(" ")) — \(.[1]|ltrimstr(" "))"'
```

**Never render examples or footers with `printf` in a recipe.** It was tried:
22 `printf` lines, hand-counted padding, and examples reduced to string literals
nothing can query. A repo that wants structured examples wants the `rich` variant.

## 7. The inline documentation guard (zero scripts)

The `native` variant copies **no helper scripts** into the repository. Wire a
single inline check into `check` (and `ci`) that inspects `just --list` directly
and fails if any public recipe lacks `[doc]` or any top-level recipe lacks `[group]`:

```just
[group('core')]
[doc('Verify every recipe has [doc] and every top-level recipe has [group]')]
check:
    @just --list --list-submodules --list-heading '' | awk '/^    [a-zA-Z0-9_-]+/ && !/#/ && !/:$/ { print "missing [doc]: " $1; err=1 } END { exit err }'
    @just --list --list-heading '' | awk '/^    \[/ { g=1; next } /^    [a-zA-Z0-9_-]+/ && !g { print "missing [group]: " $1; err=1 } END { exit err }'
```

Static, zero files to vendor or sync, no `jq` dependency, and fails for the
exact same reason a reader would complain — a bare recipe name or an ungrouped
top-level recipe.

If `justz` is installed, `justz check` is the stronger guard for a `native` repo too:
it reads `just --dump` instead of the listing text, and also reports multi-line comment
blocks that `--list` would truncate (`doc-multiline-comment`) and variadic interpolation
(`variadic-interpolation`). It stays optional here: `native` must work with only `just`,
so do not make `check` fail for a contributor who has not installed it.

## 8. Verify

```bash
just                                   # compact listing, groups in source order
just help                              # every namespace expanded
just <ns>                              # one namespace
just explain <ns>::<recipe>            # full doc text
just <ns> <cmd> --flag                 # declared options parse
just <ns> <cmd> --nope                 # undeclared option is rejected
just check                             # inline doc/group check passes
just --unsorted --summary              # source order, private recipes absent
```

Add a deliberately undocumented recipe and confirm `just check` fails, then remove
it. A guard nobody has seen fail is a guard nobody can trust.
