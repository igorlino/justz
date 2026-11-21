# Fixing `check` and `lint` findings

`justz check` is about documentation.
`justz lint` is about the recipes.
Neither edits the justfile.

| | `check` | `lint` |
| --- | --- | --- |
| Severities | errors and warnings | warnings only |
| Exits `1` | on an error, or on a warning with `--strict` | only with `--strict` |

## Flags of both

- `--json`: every finding has `rule`, `severity`, `subject`, `message` and `fix`.
- `--strict`: warnings fail the run too.
- `--rule <name>`: show one rule at a time.
  **It is for reading, not for gating CI**: the counts and the exit code follow the shown findings, so it can hide errors and exit `0`.
  An unknown name exits `2` and lists the valid ones.

The text output prints a fix once for a run of findings of one rule.
The JSON keeps it on every finding.

## Fix documentation drift

1. Run `justz check --json`.
2. Apply the `fix` of each finding.
   Work through one rule at a time with `--rule <name>`.
3. Run it again until it exits `0`.

| Rule | What to do |
| --- | --- |
| `registry-desc-diverges` | The justfile `[doc]` and the registry `desc` must be the same text. The finding has a `paste` field: the exact `[doc(...)]` line for the justfile. Paste it yourself. |
| `registry-unsorted` | Run `justz fmt`. |
| `registry-desc-long` | A warning. Shorten `desc` to a one-line summary equal to the `[doc]`, and move the rest to `detail`. With the generated overlay, shorten the `[doc]` and run `justz generate`. |

The list of rules is in `justz check --help`.

## Act on lint findings

Run `justz lint --json`.
A finding sometimes has `line`: it counts lines of the recipe body as `just --dump` shows them, not lines of the file.

| Rule | What to do |
| --- | --- |
| `recipe-long` | Move the body to `scripts/<group>/<name>.sh`. The finding counts the `{{ }}` interpolations (JSON: `interpolations`, `quotedInterpolations`, `escapedBraces`). Those in single quotes or a quoted here-document are not mechanical to move. |
| `secret-parameter`, `secret-interpolated` | Read the secret inside the recipe. Do not take it as a parameter, and do not interpolate it. |
| `module-shell-unset` | Give the module a `set shell`, or import the shared preamble. |
| `module-dispatcher-missing`, `module-dispatcher-lists-root` | Give the module a default recipe that runs `justz help <module>`. |

## Silence one recipe

Only when the finding is wrong:

```just
[metadata('justz-ignore', '<rule>')]
```

- It works for the lint rules `recipe-long`, `secret-parameter` and `secret-interpolated`.
- It works for the `check` advice rules `volatile-text`, `detail-flag-no-row`, `detail-env-no-row`, `detail-long`, `detail-no-paragraphs`, `detail-restates-desc` and `tags-confirm`.
- `module-shell-unset` and the two dispatcher rules have no per-recipe opt-out.
- A typo in the rule name is itself a finding.
- `detail-env-no-row`: a prose constant such as `UTF_8` can fire, and a name under 3 characters is not caught.
- `[metadata]` needs `just` 1.42.0 or newer, and is verified on `just` 1.59.0 only.

Do not turn a finding off to make the run pass.

## Run the check before commits and in CI

The snippets are in <https://github.com/igorlino/justz/blob/main/docs/hooks-and-ci.md>:

- a pre-commit framework hook, id `justz-check`;
- a plain git hook that fails loudly without `justz` (`JUSTZ_HOOK_SKIP_MISSING=1` skips);
- a GitHub Actions job.
