# Writing documentation

How the documentation of this repository is organized, for people and agents who change it.
The goal: a reader finds a fact in one obvious place, and a new feature does not make an existing page harder to read.

## One home per fact

Every fact has one page that owns it.
Other pages link to that page; they do not restate it.

| When you add or change... | Update | Do not |
| --- | --- | --- |
| A command or a flag | its `--help` text, then [commands.md](commands.md) and the table in `skills/justz-cli/SKILL.md` | describe the flag in the README |
| A `check` or `lint` rule | `--help`, then [checks.md](checks.md) | list rules in the README or in `SKILL.md` |
| A registry field | `schema/commands.schema.json`, then [registry.md](registry.md) | |
| What a variant gives, or how to choose one | [variants.md](variants.md), then `references/choosing-a-variant.md` in the `justfile-authoring` skill | compare variants on another page |
| A JSON field or an exit code | [for-agents.md](for-agents.md) and `skills/justz-cli/` | |
| An install method or a version requirement | [installation.md](installation.md) | |
| A symptom and its fix | [faq.md](faq.md) | |
| The pitch, the quickstart or the ways to type a command | the README, then the same section of `site/index.html` | let the website and the README disagree |
| An example shown in the website terminal | the list of runs in `scripts/sitedemo.go` or the project in `testdata/site-demo`, then `just site-demo` | edit `site/demo.js` by hand |
| The one-sentence description of `justz` | the README, then the title, description and headline of `site/index.html`, and the first paragraph of `llms.txt` | |
| A page for users added to `docs/` | the Documentation table of the README, the same list on the website, and `llms.txt` | |
| Who can contribute, and how | `CONTRIBUTING.md` | restate the policy in the README |
| A Markdown style rule | `.rumdl.toml`, then [Markdown style](#markdown-style) | |
| Anything a user will notice | `CHANGELOG.md`, under Unreleased | |

The README changes only when the pitch, the quickstart or the list of pages changes.
It is a front page: what `justz` is, why, how to start, where to read more.
The website in `site/` is a second front page with the same pitch and quickstart: change both in the same PR.

## Rules

- **Add a row, not a clause.**
  A new fact is a new table row, a new bullet or a new section.
  Never extend an existing sentence with "and", a semicolon or a parenthesis to fit a feature in.
- **One idea per bullet.**
  A bullet that needs a second sentence of exceptions is a section.
- **Tables for lookups, numbered steps for procedures, prose for reasons.**
- **Paste real output.**
  Run the binary on a fixture in `testdata/` and paste what it prints.
  Do not write output from memory.
- **Describe the present.**
  No "available", "new", "coming soon" or version history in a page.
  History goes in `CHANGELOG.md`.
- **A skill is self-contained.**
  A skill is installed without this repository, so `SKILL.md` links only to its own `references/`, or to a full URL.
  Keep `SKILL.md` an entry point: a quick start, short rules and tables.
  Long material goes in `references/`.
- **The top of a page is what people see.**
  GitHub, skills.sh and package listings show the first lines and cut the rest.
  Put the one-sentence answer first.

## Markdown style

[rumdl](https://github.com/rvben/rumdl) checks every Markdown file with the rules in `.rumdl.toml`.
`just md-lint` reports the findings, and `just ci` runs it.
`just md-fix` applies the fixes; read its diff before you commit it.
A commit that only reformats goes in `.git-blame-ignore-revs`, so `git blame` skips it.

| Choice | Why |
| --- | --- |
| One sentence per line, no width limit | A word change is a one-line diff, and tables and pasted output need no wrapping |
| Code blocks are not reflowed | They hold real output of the binary |
| Compact tables (`\| a \| b \|`) | Editing one cell does not re-pad the whole table |
| The `gfm` flavor | The docs are read on GitHub |
| Only one opt-in rule, MD060 (tables) | The others are off in rumdl for reasons that hold here, see [its opt-in rules](https://github.com/rvben/rumdl/blob/main/docs/rules.md#opt-in-rules) |

When a finding is wrong, put the exception where it applies:

| Scope | Where |
| --- | --- |
| The whole repository | `.rumdl.toml`, with a comment that says why |
| A file of another format, such as a GitHub template | `[per-file-ignores]` in `.rumdl.toml` |
| One line | `<!-- rumdl-disable-next-line MDxxx -->` above it |

## What the tests enforce

`internal/doclinks` runs with `just test`:

| Test | Fails when |
| --- | --- |
| `TestRelativeLinksResolve` | a relative link in the README, `docs/` or `skills/` points at a file that does not exist |
| `TestEveryCommandIsDocumented` | a command of the CLI is not in [commands.md](commands.md) or in the `justz-cli` skill |
| `TestEveryRuleIsDocumented` | a `check` or `lint` rule is not in [checks.md](checks.md) |
| `TestReadmeBudget` | the README is over 1150 words, or has a bullet over 40 words |
| `TestSkillBudget` | a `SKILL.md` is over 500 lines or 3000 words |

A failing budget means the text is in the wrong place.
Move it to the page that owns it; do not raise the limit.
