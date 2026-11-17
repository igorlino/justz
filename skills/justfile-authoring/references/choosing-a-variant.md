# Choosing a variant

Two supported team shapes, plus a personal overlay. Recommend `rich` when every machine
that renders the help can install `justz`, and `native` when it cannot. The size of the
repository does not decide.

| | `rich` | `native` |
|---|---|---|
| Descriptions live in | `commands.json` (`desc`); any `[doc]` must equal it | `[doc('…')]` on the recipe |
| Rendering | `justz help` | `just --list` (optionally `justz help`) |
| Arg documentation | `args` rows in JSON | `[arg(…, help='…')]` — real CLI options |
| Examples | `examples` in JSON | convention inside a multiline `[doc]` |
| Details, safety tags, exit codes, env rows, see-also | `detail`, `tags`, `exits`, `env`, `seeAlso` | **nowhere** |
| Bare-name lookup (`jz clean` finds `tool clean`) | yes | no |
| Metadata file | yes | **none** |
| Dependencies | `justz` on every machine that renders help | none (`justz` optional) |
| Checks | `justz check`, `justz fmt --check` | an inline guard, or `justz check` |

The same comparison for people, with what a developer keeps when only they install
`justz`: https://github.com/igorlino/justz/blob/main/docs/variants.md

## Assess the repo first, then recommend

Print a short assessment — facts, not prose — before proposing anything:

```
  <repo> — assessment

  just               present, 7 recipes, no groups, no [doc]
  namespaces         1 (flat)
  flag surfaces      none
  cross-cutting      none
  just version       1.59 → [group], [arg], --explain available

  Recommendation: the `rich` variant, if every machine that renders the help can
  install `justz`. Reason: safety tags, details and exit codes have no place in a
  justfile. Otherwise `native`.
```

`justz assess --json` reports these numbers exactly, with its own recommendation and the
reason.

## The one question

Can every teammate, and every CI job that renders the help, install `justz`?

| Answer | Recommend |
|---|---|
| Yes | `rich` |
| No | `native` |
| Not known | Ask. It is the first row of the batched question in step 2 |

Default to **the `rich` variant**. Three reasons, in the order to give them:

- **Safety tags exist only in the registry.** In `native`, nothing tells an agent or a
  newcomer that a command is `destructive` or touches `production`. That is as true of
  four recipes as of forty, which is why size is not the test.
- **The registry is the only structured place** for details, exit codes, environment
  rows and cross-references. Without one they end up in a README, or nowhere.
- **`justz check` fails when `desc` and `[doc]` differ.** The cost measured below came
  from two copies that nothing compared.

Say what `native` gives up in one line when the user picks it: no details, no tags, no
exit codes, no environment rows, no bare-name lookup. A developer who installs `justz`
still gets `justz help`, `check`, `lint` and completion on a `native` repo.

`rich` has one condition that is not negotiable: `justz check` runs in the `check`
recipe, and in CI where there is one. Without it the two copies drift, as below.

## What a registry costs without a check — measured, not asserted

From a six-month-old `rich` adoption (33 recipes, 4 namespaces) that used the older
vendored-bash renderer:

- **Descriptions diverged in 20 of 32 recipes.** `desc` in `commands.json` and
  the recipe's own doc comment are two copies of one fact, and the guard only
  checked that a JSON entry *existed*, never that it matched. Several
  divergences were semantic, not cosmetic.
- **Long descriptions make two copies harder to keep equal.** In the largest
  registry we have seen, most commands had a `desc` over 120 characters.
  With `justz` 0.7.0 the long text goes in `detail`, so `desc` and `[doc]` stay
  one short, equal line. That makes the check easier to keep green; it does not
  make the divergence above any less true.
- **The drifted copy is the one most tools show.** `just --list`, shell
  completions, the IDE extension and `just --dump` all read the justfile; only
  the custom renderer read the JSON.
- **The runtime guard never fired.** It was opt-in, existence-only, and
  triggered only when a recipe *runs*. `justz check` is static and compares text,
  which is what closes this gap.
- **A hard dependency was never added to `doctor`**, despite an explicit checklist
  item. Prose instructions decay; a check that fails does not.
- **Tiers were over-built.** The schema supports N tiers; that repo defines one.
- Converting to the `native` variant removed **421 lines** (734 → 313) and made the
  divergence structurally impossible. `justz check` now makes it detectable in `rich`
  too, which narrows the gap between the two.

From the same repo, what the registry bought: its UI namespace
carries 11 arg rows, 28 examples, a 5-line footer and 4 cross-references. That
is the best documentation in the repo and it exists *because there was a
structured place to put it*. The `native` variant reproduces it only with `[arg]` plus a
doc-text convention, and the result is looser.

## What `just` may add upstream

The weakness of the `native` listing traces to one `just` limitation:
`--list` renders either the whole doc or nothing — it cannot show the first line
of a multiline doc (casey/just#2684, open, with a maintainer proposal for
short/long `[doc]` arguments).

- `[group]`, `--groups`, `--explain`, `set default-list` and `[arg(…, help=…)]`
  already landed. When short/long `[doc]` lands, `native` gets a short and a long
  description for free.
- Safety tags, exit codes and environment rows are not part of that proposal. They
  stay in the registry.

Mention this in one line when the user chooses `native`: the gap narrows, and it
does not close.

## Do not offer a third way

Rendering examples and footers with `printf` in the justfile was tried and
rejected: 22 `printf` lines with hand-counted padding, examples reduced to
string literals that nothing can query. If a repo wants structured examples, it
wants the `rich` variant.

This does not rule out `references/variant-local-rich.md`: a personal,
gitignored, *generated* `commands.json` one developer builds for themselves
on a `native` repo with `justz generate`. It is not a third team-level shape (nothing changes for
anyone else, and it is never offered in the step-2 batched question); it is
`rich`'s renderer and completions, opted into individually,
without the divergence risk, because the file is derived from the justfile
each time rather than hand-maintained.
