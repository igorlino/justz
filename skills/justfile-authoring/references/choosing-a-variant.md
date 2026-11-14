# Choosing a variant

Two supported team shapes, plus a personal overlay. They produce the same *kind* of
thing, a discoverable, grouped command surface, but they put the documentation in
different places, and that single difference drives every trade-off below.

| | `native` | `rich` |
|---|---|---|
| Descriptions live in | `[doc('…')]` on the recipe | `commands.json` (`desc`) |
| Rendering | `just --list` (optionally `justz help`) | `justz help` |
| Arg documentation | `[arg(…, help='…')]` — real CLI options | `args` rows in JSON |
| Examples / see-also | convention inside a multiline `[doc]` | `examples`, `seeAlso` in JSON |
| Metadata file | **none** | yes |
| Dependencies | none (`justz` optional) | `justz` |
| Checks | an inline guard, or `justz check` | `justz check`, `justz fmt --check` |

## Assess the repo first, then recommend

Print a short assessment — facts, not prose — before proposing anything:

```
  <repo> — assessment

  just               present, 7 recipes, no groups, no [doc]
  namespaces         1 (flat)
  flag surfaces      none
  cross-cutting      none
  just version       1.59 → [group], [arg], --explain available

  Recommendation: the `native` variant. Reason: 7 flat recipes; a registry and a
  dependency on a renderer would serve 7 one-line descriptions.
```

`justz assess --json` reports these numbers exactly, with its own recommendation and the
reason. Signals that justify the `rich` variant's extra machinery: **all** of the first
three should hold, not just one:

| Signal | Threshold |
|---|---|
| Size | ≥3 namespaces **and** ≥25 recipes |
| Flag surface | ≥1 recipe with more than two meaningful options |
| Cross-cutting caveats | ≥1 namespace where a newcomer will get it wrong without prose |
| Ownership | someone will own the registry, and every machine that renders help installs `justz` |

Default to **the `native` variant**. Most repos with a task runner have a handful of flat
recipes, and for those `rich` is machinery without a payload.

## Evidence — measured, not asserted

From a six-month-old `rich` adoption (33 recipes, 4 namespaces) that used the older
vendored-bash renderer:

- **Descriptions diverged in 20 of 32 recipes.** `desc` in `commands.json` and
  the recipe's own doc comment are two copies of one fact, and the guard only
  checked that a JSON entry *existed*, never that it matched. Several
  divergences were semantic, not cosmetic.
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

Counter-evidence, in `rich`'s favour, from the same repo: its UI namespace
carries 11 arg rows, 28 examples, a 5-line footer and 4 cross-references. That
is the best documentation in the repo and it exists *because there was a
structured place to put it*. The `native` variant reproduces it only with `[arg]` plus a
doc-text convention, and the result is looser.

## The upstream bet

Every remaining weakness of the `native` variant traces to one `just` limitation:
`--list` renders either the whole doc or nothing — it cannot show the first line
of a multiline doc (casey/just#2684, open, with a maintainer proposal for
short/long `[doc]` arguments).

- The `rich` variant bets that gap persists, and pays a registry and a dependency on `justz`.
- The `native` variant bets it closes. `[group]`, `--groups`, `--explain`,
  `set default-list` and `[arg(…, help=…)]` already landed; when short/long
  `[doc]` lands, `native` gets most of `rich`'s output for free and the registry
  becomes optional in every repo that has one.

State this bet to the user in one line when recommending. It is the actual
decision; everything else is detail.

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
