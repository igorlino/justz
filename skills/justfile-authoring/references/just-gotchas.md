# `just` gotchas that break discoverability

This skill does not teach `just` syntax: run `just --man` or read the manual, and see `cli-just` or `justfile-expert` for syntax skills. These are the traps that break listings, arguments and modules, each **verified on `just` 1.59.0** unless marked *(manual)*. Re-check them against `just --version` before relying on one.

## Listings and documentation

- **A comment block above a recipe shows only its last line in `just --list`.** Use `[doc('…')]` for the text you want shown. `justz check` reports this as `doc-multiline-comment` *(manual, reported by `justz check`)*.
- **`--list` shows only the whole `[doc]` or nothing.** A multi-line `[doc]` prints its block above the recipe and drops the inline description. Keep `[doc]` one line and put long prose elsewhere (`references/variant-native.md`, section 5).
- **Private is not protected.** `[private]` and a leading underscore hide a recipe from `just --list` and `--summary` only. `just p` and `just _q` still run.
- **A leading underscore does not hide a module.** `mod _int` still lists as `_int ...`. `[private]` above `mod pm` hides it, and `just pm y` still runs. One public skill claims otherwise.
- **`--list` shows a module as `name ...`.** Use `just --list name` or `just --list --list-submodules` to see its recipes. A module with no `_default` runs its first recipe when called bare, so give each module a private `_default`.
- **A public `default` recipe appears as a row.** Use `_default`, first in the file.
- **Flag order:** `just --unsorted --list <ns>` works; `just --list --unsorted <ns>` fails because `<ns>` is parsed as a recipe argument.

## Arguments

- **Interpolating a variadic splits on whitespace.** `{{ args }}` shreds an argument that contains a space. Use `set positional-arguments` and `"$@"`, which keeps each argument one word (`[positional-arguments]` does the same for one recipe).
- **Required and optional variadics.** `+files` with no argument fails with `got 0 positional arguments but takes at least 1`; `*flags` accepts none. With `name="w" *rest`, `just c n x y` gives `name=n` and `rest="x y"`.
- **Never put a positional ahead of a variadic** in a wrapper; it swallows the first forwarded token. Make it a flag.
- **`[arg('n', long='num')]`** makes `just al --num 5` work and lists the recipe as `al [OPTIONS]`. It is recent: set `set minimum-version` if the repo must support older `just`.
- **`just FOO=bar <ns> <cmd>` is silently ignored for module variables** *(1.58.0)*. Anything a caller overrides must be an environment variable.

## Modules and settings

- **A module recipe runs with its working directory set to the module's directory.** `justfile_directory()` inside a module is still the **project root**, and `source_directory()` is the module's own directory. `[no-cd]` runs the recipe where the user is.
- **Settings do not carry into modules.** With `set quiet` in the root, `just m up` still echoed its command, because the module had no `set quiet`. Repeat `set shell`, `set quiet`, `set positional-arguments` in each module that needs them.
- **`mod? maybe`** (optional module) lets a justfile load when the file is missing.
- **`set fallback`**: running `just g` from `m/` found the parent's recipe `g`.
- **Modules parse on every invocation**, even when you run a different recipe. Avoid load-time work in them (backticks in assignments, asserts) *(manual)*.
- **A top-level backtick runs for every recipe that uses the variable, and at load time for listing and completion** when used in a `set` value or an `assert` *(manual)*. Keep slow commands inside a recipe.

## Recipe bodies

- **Each body line runs in its own shell.** `cd build` on one line does not reach the next. Use a shebang recipe, `[script]`, or one `&&` line *(manual)*.
- **Shebang recipes ignore `set shell`.** Add `set -euo pipefail` in the body *(manual)*.
- **`[script("python3")]`** names the interpreter in the attribute and runs the body as one file (`print("script ok")` ran).
- **Interpolation is not quoting.** `{{ x }}` pastes text; use `quote()` for values that may contain spaces, or `"$@"` with `set positional-arguments` *(manual)*.
- **An orphaned indented block after a shebang recipe** is absorbed into the preceding recipe. Cross-check `just --list` *(manual)*.
- **`env_var()` is deprecated;** use `env()` *(manual)*.
- **`.env` loading with `set dotenv-load` is global** to every recipe, so keep real credentials out of it and resolve them per recipe (`references/credentials-in-recipes.md`).

## Verified here

On `just` 1.59.0: private recipes and modules, variadic argument behaviour, `[arg]` with `long=`, `[script]`, `[no-cd]`, `mod?`, module working directory and `justfile_directory()`/`source_directory()`, settings not inheriting into a module, `set fallback`, `just --dump --dump-format json`, `--groups`, `--summary`, `--show`, and `--fmt --check`.
