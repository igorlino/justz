# Design note: `justz lint`

Status: proposed, second draft after an independent review against just 1.59.0. Nothing is implemented. The open questions at the end need a decision before the code starts.

## Purpose

Find antipatterns that can be seen in the structure of a justfile, as advice. `justz check` stays about documentation and parity. `justz lint` is about recipe parameters and bodies, and every rule is a warning.

```
justz lint [--json] [--strict]
```

It finds the justfile from the working directory, as `check` does, and takes no path.

Exit codes follow `check`: `1` when there is an error, or with `--strict` when there is a warning; `1` when the justfile cannot be loaded; `2` for a usage error; otherwise `0`. Every rule in this note is a warning, so without `--strict` a finding never fails a run.

`--json` prints the same envelope as `check` (`ok`, `checked`, `errors`, `warnings`, `findings`). A finding has the `check` fields (`rule`, `severity`, `subject`, `message`, `fix`) plus `line` where a rule points into a body.

## Input

The JSON of `just --dump --dump-format json`. `justz` does not parse justfile text and never runs a recipe. What the review confirmed about the dump:

- A recipe body is a list of lines. A line is a list of text fragments and interpolations. Comment lines, blank lines, the shebang line and each continuation line are separate entries.
- An interpolation is an expression tree, not a name: `["variable","tok"]`, `["call","quote",["variable","tok"]]`, `["concatenate",…]`, `["if",…]`, `["call","env","GH_TOKEN"]`.
- The dump has no source line and no file name. Recipes from an `import` are merged into the importing file.
- A module does not inherit the root settings; its `settings.shell` is `null` when it sets none.

So `line` is the 1-based index into the recipe body as dumped (the first body line is 1, and comment and blank lines count). It is not a line of the file.

**Scope.** Every recipe is looked at: public, private, `default`, and recipes of nested modules. `check` skips private recipes because they need no documentation; a long body or a secret is as real in a private helper. The subject of a finding is `just` and the recipe path, as in `check` (`just deploy`, `just ops deploy`, `just ops db restore`).

## Opt-out

Per recipe, with the attribute `just` already has. It is in the dump, so no comment parsing is needed:

```just
[metadata('justz-ignore', 'secret-parameter', 'recipe-long')]
deploy token:
```

- The first value must be exactly `justz-ignore`. Every further value is one rule id. Several such attributes on a recipe add up.
- An id that is not a lint rule, or an attribute that names no rule, is a finding (`ignore-unknown-rule`), so a typo cannot hide a finding.
- `just` rejects `[metadata]` on a `mod` line (`module has invalid attribute metadata`), so the attribute cannot silence a module rule. The one module rule below has its own opt-out.
- Verified on just 1.59.0 without `set unstable`. Which older versions have `[metadata]` is not verified; a dump without it means no opt-outs, never an error.

## Rules

| Rule | Reports |
|---|---|
| `recipe-long` | A recipe body over 40 lines |
| `secret-parameter` | A recipe parameter with a secret-like name |
| `secret-interpolated` | A secret-like name inside `{{ … }}` in a body line that `just` hands to a shell |
| `module-shell-unset` | A module with linewise recipes and no `set shell` |
| `ignore-unknown-rule` | A `justz-ignore` value that is not a lint rule |

### `recipe-long`

The count is the number of body entries in the dump, the same measure `justz assess` uses (`long.over40`), so the two agree on a recipe. Blank lines, comments, the shebang line and continuation lines all count: subtracting some of them needs rules for `[script]` recipes (no shebang line in the body) and for continuations, and buys little at a threshold of 40. `assess` counts public recipes only; `lint` reports private ones too. The fix names `scripts/<group>/<name>.sh` and a thin recipe.

### The two secret rules

The first draft looked for secret-like names in the arguments of `ssh`, `sudo` and `env`. The review showed that this misses the main exposure and is fragile, so it is replaced. What was measured on just 1.59.0:

- `just deploy S3CR3T` puts the value in the argument list of `just` itself, before any body line runs. With `set positional-arguments` it is also in the argument list of the recipe's shell, even when the body never names it.
- `just` runs each linewise line as `<shell> -c '<line>'` with `{{ … }}` already replaced. So an interpolated value is in the shell's argument list whatever the command is (`echo`, `printf`, a comment line), and `just` prints the line unless it starts with `@`.
- `$NAME` in the body is expanded by the shell later. It reaches a process list only when it ends up in the arguments of an external command.

The first two need no shell parsing, so they are the rules.

**`secret-parameter`.** A parameter of the recipe has a secret-like name. The message: the value is typed on the command line, so it is visible in the process list and stays in the shell history. The fix: take no secret as a parameter; read it inside the recipe from the environment, a mode `0600` file or the password manager (`references/credentials-in-recipes.md` in the `justfile-authoring` skill).

**`secret-interpolated`.** A body line of a linewise recipe (comment lines included, since `just` interpolates and prints them) has an interpolation whose expression tree contains:

- a variable or parameter with a secret-like name;
- a call to `env`, `env_var` or `env_var_or_default` whose first argument is a secret-like name;
- a variable that the dump's `assignments` define, directly or through other variables, by one of the above (`tok := env('GH_TOKEN')`, then `{{ tok }}`).

The fix: use the shell's own variable (`$GH_TOKEN`, or an exported `just` variable as `$tok`) so the value stays in the environment. The message never contains the line text or a value.

Shebang and `[script]` recipes are not reported by this rule: there the interpolated text goes into a temporary script file, not into an argument list. That is a smaller exposure (a file in the user's runtime directory) and is left out of v1.

**Secret-like name.** Lower-case the name and split it on `_`, `-` and camel-case boundaries.

- It is secret-like when a part ends in `token`, `tokens`, `password`, `passwd`, `passphrase`, `secret`, `secrets`, `credential`, `credentials` or `apikey`; or when the last part is `pass` or `pw`; or when it has `key` right after `api`, `private`, `access`, `secret` or `signing`.
- It is not secret-like when its last part says it is something else: `file`, `path`, `dir`, `name`, `id`, `user`, `url`, `host`, `ref`, `arn`, `ttl`, `endpoint`, `length`, `count`, `type`, `args`.

| Name | Result |
|---|---|
| `API_TOKEN`, `apiToken`, `api-token`, `authtoken`, `dbpassword`, `dbPass`, `SSH_PASS`, `pw`, `SSH_KEY_PASSPHRASE`, `APIKEY`, `SIGNING_KEY`, `SECRETS` | secret-like |
| `SSH_KEY_FILE`, `TOKEN_PATH`, `GITHUB_TOKEN_FILE`, `AWS_ACCESS_KEY_ID`, `TOKEN_TTL`, `TOKEN_ENDPOINT`, `PASSWORD_LENGTH`, `OP_TOKEN_REF`, `SECRET_ARN`, `PASS_THROUGH_ARGS` | not secret-like |
| `TOKENIZER`, `PASSTHROUGH`, `BYPASS`, `COMPASS`, `KEYCLOAK_URL`, `KEY` | not secret-like |
| `GH_PAT`, `DATABASE_URL`, `VAULT_SECRET_ID`, `TOKEN2` | missed, accepted: no name list finds these without flagging far more |
| `TESTS_PASS` | flagged wrongly, accepted: silence it with `justz-ignore` |

Fixtures in `internal/lint/testdata` are synthetic recipes with exactly these names and shapes, never copied from a private repository.

### `module-shell-unset`

A module that has at least one linewise recipe and whose `settings.shell` is `null` runs those recipes with `sh -cu`: no `pipefail`, and not the shell the root file chose. The fix is to import a shared settings file (`justz init` writes `just/preamble.just`) or to add a `set shell` line.

- A module with only shebang or `[script]` recipes is not reported: the setting does not affect them.
- Any `set shell` line or `set windows-shell` line silences the rule, including `set shell := ['sh', '-cu']`: a windows-shell means the author thought about shells. That is the opt-out: the rule is about a module that did not choose, not about which shell is right.
- The root file is not reported. A root without `set shell` is a normal small justfile.

## Not in scope

- Reading, resolving, storing or scanning for secret values. Use a dedicated scanner such as `gitleaks`.
- `shellcheck`. A long recipe is reported so that it can move to a script that can be linted.
- Running `lint` from `check`.
- `$NAME` inside shell text. See open question 1.

## Plan

1. `internal/lint` with `recipe-long`, the opt-out and `ignore-unknown-rule`, then `justz lint` with completion entries.
2. `module-shell-unset` (done).
3. `secret-parameter` and `secret-interpolated` with the name matcher and its table as tests.

## Open questions

1. **Secrets in shell text (`ssh host "run $TOKEN"`, `curl -H "Authorization: Bearer $TOKEN"`, `docker run -e`, `sshpass -p`).** This is the shape the plan's audit found, and the two rules above catch it only when the secret arrives through a parameter or `{{ … }}`. Finding `$NAME` in the arguments of an external command means parsing shell: quoting, heredocs, `$( … )`, prefixes such as `timeout` and `exec`, and strings over several lines. The review built wrong results for each. Recommendation: not in v1. Run the M9 pilot with the two structural rules, look at what they miss in the real file, and design a shell-text rule against that evidence, or keep the guidance in the skill.
2. **Shebang recipes in `secret-interpolated`.** Recommendation: leave them out of v1, as above.
3. **Line limit.** Recommendation: 40 body entries, identical to `assess`.
4. **`secret-parameter` when the parameter has a default from the environment** (`deploy token=env('GH_TOKEN')`). The usual call types nothing, but the parameter still invites it. Recommendation: report it; the opt-out exists.
