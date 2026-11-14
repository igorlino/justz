# The `local-rich` overlay: `rich` benefits, personal, on a `native` repo

Not a third team-level choice. The batched question in step 2 of `SKILL.md` never offers it. It is something one developer sets up for themselves on a repo the team scaffolded as `native` (or never scaffolded at all), when the team does not want a committed `commands.json` but this developer still wants richer help and completion. Nothing about it is visible to anyone else: no repo file changes and no new dependency for the team.

It needs `justz` 0.5.1 or newer on the developer's machine. Check `command -v justz` first and stop with install advice if it is missing.

## Why this is safe where a hand-maintained registry was not

The `rich` variant's known cost, 20 of 32 descriptions diverging within six months (`references/choosing-a-variant.md`), came from two people editing two copies of one fact. `local-rich` has one person and one source: the file is a *derived view* of the justfile's own `[doc]`, `[group]` and `[arg]` attributes, produced by `justz generate` and never hand-edited. Re-running the command after a justfile change is the whole maintenance burden.

## Setup (once per machine, not per repo)

1. Point git at a personal global ignore file if you have none:

   ```bash
   git config --global core.excludesFile ~/.gitignore_global
   ```

2. Add the generated filename to it, so it is invisible to `git status` in every clone:

   ```bash
   echo '.just-local-rich.json' >> ~/.gitignore_global
   ```

3. Generate it from anywhere inside the repo:

   ```bash
   justz generate
   ```

   It writes `.just-local-rich.json` at the project root (the directory with `.git`, or the justfile's directory). `justz generate --stdout` prints the result and writes nothing. `justz` **refuses to write the file where git tracks it**, because the overlay is personal. It never edits `~/.gitignore_global`; that edit is yours to make.

4. Re-run step 3 after any justfile change. There is no drift check, because there is nothing to drift from: the file either matches the justfile it was just generated from, or you have not regenerated yet.

## What it enables

With the overlay present and no committed registry, `justz help`, `jz`, `jzh`, shims and `justz completions` treat the project as `rich`: grouped help with argument rows, and completion from namespace to recipe to flag to value. A committed `commands.json` always wins over the overlay.

`justz generate` rules worth knowing:

- Root recipes land in `core` whatever their `[group]`. A `[group]` attribute is only a display heading; a registry group is also the path used to run a command, so only top-level **modules** become groups. Nested modules are skipped with a warning.
- A parameter declared with `value=` is a flag with no placeholder.
- Hand-added `"values": [...]` lists are **kept** when you regenerate, so enum completion survives.
- Multi-line `[doc]` text is cut to its first line.

## Assess before treating the generated file as done

`justz assess --json` reports exact facts about the justfile: recipes and how many are documented, modules, shebang and long recipes, whether a registry covers every public recipe, and which variant fits. Use it, not a search of the text.

An **empty description is always worth fixing**; there is never a good reason for one. A low count of `[arg(help=)]` is a *signal to check, not a verdict*: some catch-all `*ARGS` recipes are deliberately documented through their own `[doc]` text. Read the flagged recipes before concluding anything is missing.

If an agent is doing this, the cross-cutting rule in `SKILL.md` applies: preview the concrete plan, recommend one option, then ask. Never an abstract yes/no. In order:

1. Print the coverage specifics from `justz assess`.
2. Read what each flagged recipe's body does. Do not guess `[doc]` or `[arg(help='…')]` text from the recipe name.
3. **Draft the actual text** for the worst offenders and show it as a small diff of the lines you would add, in the same message as the offer.
4. Ask once, batched, with a recommendation (usually: fix recipes with an empty description; treat low argument-help coverage as optional). Declining is valid: the generated file is still better than nothing.
5. Only on an explicit yes, apply it to the real justfile (visible to the whole team, unlike the overlay) and re-run `justz generate`.

## What it cannot do

- **It does not replace `rich`'s group-level `examples`, `footer` and `seeAlso`.** The generator carries a command's `desc` and `args`, plus one `example` parsed from an `example:` line in a multi-line `[doc]`. Group-level fields are left empty; hand-add them if you want them. If they earn their keep for the whole team, the repo has outgrown `native`; propose `rich`.
- **A plain justfile gets descriptions but no flags.** `just --dump` fills `doc` from an ordinary `#` comment on the line directly above the recipe, so a repo with no attributes still gets real per-recipe descriptions. `[arg(help=…)]` is the only source of argument rows, so without it there is namespace and recipe completion with descriptions but no flag or value completion.
- A public `default` recipe is **not** skipped: it is generated as a `core` command with whatever description it has. Rename it to `_default` in the justfile (the fix this skill recommends anyway), or accept the row.

## Verify

```bash
justz generate                            # writes the overlay, prints the target
git status --short .just-local-rich.json  # prints nothing: the global ignore works
justz help                                # richer output than `just --list`
```

Then open a new shell and confirm `just <ns> <TAB>` and `just <ns> <cmd> --<TAB>` complete with descriptions (`references/variant-rich.md`, "Shell completion").
