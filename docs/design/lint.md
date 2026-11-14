# Design note: `justz lint`

Status: proposed. Nothing is implemented. The open questions at the end need a decision before the code starts.

## Purpose

Find antipatterns that can be seen in the structure of a justfile, as advice. `justz check` stays about documentation and parity and keeps failing CI by default. `justz lint` is about the recipe bodies and never fails CI unless asked.

```
justz lint [--json] [--strict] [path]
```

Exit codes: `0` no findings, or only warnings; `1` with `--strict` and at least one finding; `2` usage error. A finding has the same shape as a `check` finding (`rule`, `severity`, `subject`, `message`, `fix`) plus `line`, the line inside the recipe body.

## Input

The JSON of `just --dump --dump-format json`, as for `check`. A recipe body there is a list of lines; a line is a list of text fragments and interpolations. Comments stay in the body, and attributes (including `metadata`) are in the dump. `justz` does not parse justfile text and never runs a recipe.

## Opt-out

Per recipe, with the attribute `just` already has, so it needs no comment parsing and survives `just --dump`:

```just
[metadata('justz-ignore', 'secret-on-command-line')]
deploy host:
```

One attribute names one or more rule ids. An unknown id in the list is itself a finding (`ignore-unknown-rule`), so a typo cannot hide a finding. There is no file-wide switch.

## Rules

| Rule | Severity | Reports |
|---|---|---|
| `recipe-long` | warning | A recipe body over 40 lines (blank lines and comment lines do not count; the shebang line does not count). The fix names `scripts/<group>/<name>.sh` and a thin recipe. |
| `secret-on-command-line` | warning | A secret-like name in the argument list of `ssh`, `sudo` or `env`. See below. |
| `module-shell-not-strict` | warning | A module whose `set shell` is the default (`sh`, no `pipefail`). The fix is `import 'preamble.just'`, which `justz init` writes. The root justfile is not reported. |
| `ignore-unknown-rule` | warning | An id in `justz-ignore` that is not a rule. |

Line-count thresholds are fixed, not configurable. A project that needs another number is better served by `[metadata('justz-ignore', 'recipe-long')]` on the recipe that earns it.

### `secret-on-command-line`

A secret in an argument is visible in the process list, in `sudo` logs and, for `ssh`, on the remote host as well. Base64 only changes the quoting. The rule looks at structure and never at values.

1. **Lines.** Take each body line that is not a comment. Join lines that end in a backslash.
2. **Segments.** Split the line on `|`, `||`, `&&`, `;` and `&` outside quotes. The splitter understands single quotes, double quotes and backslashes, nothing else (no `$(...)` nesting, no heredocs). A line it cannot split (an unclosed quote) is skipped, not guessed at.
3. **Command.** The first word of a segment, after leading `NAME=value` words and after `sudo` with its flags. The segment is in scope when that command is `ssh`, `sudo` or `env`.
4. **Secret-like name.** Split a variable name on `_` and on camel-case boundaries, in upper case. It is secret-like when it contains the segment `TOKEN`, `PASSWORD`, `PASSWD`, `PASS`, `SECRET`, `CREDENTIAL` or `CREDENTIALS`, or one of the pairs `API KEY`, `SECRET KEY`, `PRIVATE KEY`, `ACCESS KEY`. It is not secret-like when its last segment is `FILE`, `PATH`, `DIR`, `NAME`, `ID`, `USER`, `URL` or `HOST`, so `SSH_KEY_FILE`, `TOKEN_PATH` and `AWS_ACCESS_KEY_ID` stay quiet. A bare `KEY` is not enough: it matches too much.
5. **Reference.** In an in-scope segment, report a secret-like name that appears as `$NAME`, `${NAME}`, `{{ name }}` (a parameter or variable), `env('NAME')` or `env_var('NAME')`, or as the left side of `NAME=` (the right side may be literal, and is never copied into the message).

The message names the recipe, the line, the command word and the variable name. It never contains the value or the line text.

The fix: send the secret on standard input (`ssh host 'cat > file' <<<"$TOKEN"` is still an argument to the shell, so use a pipe from a command that reads it from the keychain) or write it to a mode `0600` file and pass the path. The `justfile-authoring` skill already has the 1Password patterns; the message points at `references/credentials-in-recipes.md`.

Fixtures. Each is a synthetic recipe in `internal/lint/testdata`, modelled on the real pattern, never copied from a private repo.

| Fixture | Expected |
|---|---|
| `ssh host "sudo -E env DB_PASSWORD={{pw}} run"` | finding |
| `ssh host "run $API_TOKEN"` | finding |
| `sudo env GH_TOKEN=x make deploy` | finding (name only; the value is not echoed) |
| `printf %s "$API_TOKEN" \| ssh host 'cat > /dev/null'` | none: the name is in `printf`, not in `ssh` |
| `ssh -i "$SSH_KEY_FILE" host` | none: a path |
| `echo "$API_TOKEN"` | none in v1 (see open questions) |
| `ssh host "echo 'a | b' $TOKEN"` | finding: the pipe is inside quotes |
| a line with an unclosed quote | none, no crash |
| `[metadata('justz-ignore', 'secret-on-command-line')]` on the recipe | none |
| the same recipe in a shebang body | the same result |

## Not in scope

- Reading, resolving, storing or scanning for secret values. Use a dedicated scanner such as `gitleaks`; `justz` stays on the structural side of that line.
- `shellcheck`. A long recipe is reported so it can move to a script that can be linted.
- Changing `check`, or running `lint` from `check`.

## `assess`

`justz assess` already counts shebang and long recipes. Once `lint` exists, `assess` can report the number of lint findings from the same detector, so the two cannot disagree. That is a second step, not part of the first PR.

## Plan

1. `internal/lint` with `recipe-long`, `ignore-unknown-rule` and the opt-out, then `justz lint`.
2. `module-shell-not-strict`.
3. `secret-on-command-line`, last, with its fixtures. If it is noisy on the M9 pilot, drop the rule and keep the guidance in the skill.

## Open questions

1. **`curl`.** `curl -H "Authorization: Bearer $TOKEN"` and `curl -u user:$PASS` expose a secret in the process list the same way. Recommendation: not in v1; add after the pilot shows what the real noise is.
2. **`echo` and `printf`.** They are shell builtins, so they do not reach the process list, but they print into CI logs. Recommendation: out of scope for this rule; a separate `secret-in-output` rule can come later.
3. **Line limit.** 40 matches `assess`. Recommendation: keep 40.
4. **Default exit code.** Recommendation: warnings never fail without `--strict`, as the plan says.
