# Design note: terminal-safe text, and highlighting in `detail`

Status: part 1 (sanitization) is implemented; part 2 (highlighting) is not started. Two changes in one note, in two PRs. The first is a security fix and stands without the second.

## 1. Text from a project must not control the terminal

### Problem

`justz` prints text that comes from files of the project in the current directory: the registry (`commands.json`: names, descriptions, examples, argument rows, labels), and for a native project the `[doc]`, `[group]` and module text of the justfile, plus the directory and file names themselves. Anyone who controls those files, for example the author of a repository you just cloned, controls those strings. `justz help` is the first command a user runs in an unfamiliar repository, which is the case to protect.

Measured on the current code, with a registry and a justfile that carry ESC `[31m`, an OSC title (`ESC ] 0 ; t BEL`), BEL, NUL, CR, DEL, U+009B and U+202E: all of them reach the terminal raw (`cat -v` shows `^[[31m`, `^[]0;t^G`, `^@`, `C^MX`, `D^?`, `M-BM-^[`). Depending on the terminal, such bytes change the window title, move the cursor and overwrite earlier output so the screen reads as something else, write to the clipboard (OSC 52), or reorder displayed text.

The paths found, which are more than the help views:

| Where it prints | Source of the text |
|---|---|
| summary, expanded, group and command zoom, `ambiguous:` lines | every registry string; native `[doc]`, `[group]` heading, the directory name used as the project name |
| `--browse` menu and fzf entries | the same; a tab or newline also corrupts the `path<TAB>label` entries |
| completion candidates (value and description) | registry keys and descriptions; a key is inserted on the command line |
| `resolve`, text and `--json` | registry keys, read from the raw document, not from the help model |
| `check` and `assess`, text and `--json` | registry keys and descriptions in subjects and messages; `registry-desc-diverges` prints two texts |
| messages that echo a path | `assess` (the registry path), `fmt`, `generate`, `shim list`, `shim doctor` |
| messages that echo what the user typed | `no group or command matches '%s'`, `ambiguous: '%s'` (text and JSON) |
| a single-line field that holds a newline | `desc` of `a\n    fake   # forged row` prints a forged row when piped; a CR overwrites the start of a row |

Not covered, and the note says so: `just`'s own output. `just --list` prints a `[doc]` raw, so justz can protect its own views only.

### JSON is not safe either

An earlier draft said JSON escapes the bytes. That is true only for C0 controls and U+2028/2029. Measured in `help -j` and `check --json`: DEL, U+009B and U+202E come out raw, as valid JSON strings. A tool that decodes the string and prints it (`jq -r`, an agent that echoes a field) sends the character to a terminal.

### Proposal

A leaf package `internal/textsafe` that imports only the standard library, so help, check, assess, resolve, complete and cmd can all use it. Two functions, with one character set.

**Replaced with U+FFFD:**

- C0 controls except `\n` and `\t`, and DEL;
- C1 controls (U+0080 to U+009F, which includes U+0085);
- a carriage return that is not part of `\r\n`;
- U+2028 and U+2029;
- the bidirectional overrides and isolates, U+202A to U+202E and U+2066 to U+2069, and the deprecated format controls U+206A to U+206F;
- the tag characters, U+E0000 to U+E007F, which are invisible and used to smuggle text that a human cannot see and an agent reads.

**Kept:** letters in every script, emoji and their joiners (U+200D), U+200B, U+200C, U+2060, U+FEFF, the marks U+200E, U+200F and U+061C that right-to-left text needs, variation selectors, private use, tabs and newlines where allowed. Banning the zero-width characters would break emoji and several scripts; the spoofing they allow in a name is closed by the identifier rule below. Banning the isolates is acceptable for hand-written registries: legitimate text rarely uses them, and the cost of a false hit is a visible U+FFFD.

**`Text(s)`** for the two prose fields, `detail` and a group `footer`: first folds `\r\n` to `\n`, then replaces the characters above. Newlines and tabs stay. The fold is part of `Text` on purpose. If the model were cleaned first and the detail cleaner ran later, the later step would never see a `\r\n`, and every line of a CRLF detail would end in U+FFFD.

**`Line(s)`** for every other field (names, labels, descriptions, examples, argument text, tier text, the project name): as `Text`, but a newline or a tab becomes one space. This removes forged rows and keeps fzf entries intact.

Replacing with U+FFFD and not deleting is deliberate: `\u001b[31mRED` becomes `�[31mRED`, so the residue shows that something was there. Swapping one rune for one rune keeps every width calculation. Invalid UTF-8 never reaches the model: Go's JSON decoder already turns it into U+FFFD.

### Where it applies

- **The finished help model, at the end of both loaders.** One pass over the `*Project` after `FromRegistry` and after `FromDump` (which sets the name, and the `[group]` heading, after `FromRegistry` has returned, so cleaning only inside `FromRegistry` would miss native). Completion, browse and every help view read the model.
- **Names that are inserted or consumed.** Completion and `resolve` only offer or report a group or command key that matches `^[A-Za-z0-9_][A-Za-z0-9_.-]*$`. Other keys, such as one with a space or an escape, still show in the help (cleaned) but are never inserted on a command line or printed for a shell to consume. `resolve` reads the raw document today; it uses the same rule. This also stops a key that looks like one command and is another.
- **`check` and `assess`, at construction.** Finding subjects, messages and fixes, and the missing and stale lists, are cleaned where they are built, so text and JSON agree. A finding that carries a registry key or a description is not an exact copy any more when it held a control character; that is the point.
- **Messages that echo what the user typed or a path** go through `Line`, not `%q`: a normal word or path prints exactly as before (`%q` would add quotes to every message and change the formats that tests and scripts read), and a control character shows as U+FFFD. `resolve` keeps its `%q`. This covers the no-match and ambiguous messages of `help`, and the paths in `assess`, `fmt`, `generate`, `init`, `doctor` and `shim`.
- **Standard error.** The real binary writes its own messages through `textsafe.NewWriter`, which applies `Text`, so a path in an error (the working directory in "no commands.json found in %s", a PATH entry, an `err` from the OS) cannot reach the terminal raw. The output of `just` itself is passed through untouched.
- **JSON written by justz, and the overlay.** Every `--json` output goes through `printJSON`, which writes through `NewWriter`: Go's encoder escapes ESC and BEL but leaves DEL, C1 and bidi characters raw. `generate` cleans the registry it writes in the same way, for `--stdout` and for the file. The structure of both is ASCII, so cleaning the bytes cannot break it.
- **Pasteable and eval'd text.** The `setup` PATH hint is a command to paste, so a directory with a quote, `$`, backtick, backslash or control character gets a plain sentence instead. The completion script lists shim names without quotes and is `eval`'d, so a name from `shims.json` that `shim add` would not accept is left out.
- **Option values.** Completion offers an enum value only when it is made of letters, digits and `_ . , : @ % + = / ~ -` (`textsafe.IsValue`), so a value with a space, `;`, `$(...)` or a newline is not inserted on a command line. Flag names already match `-{1,2}[A-Za-z0-9][A-Za-z0-9-]*`.
- **`help -j`** carries the cleaned strings. One cleaned text everywhere is simpler to reason about than two.

`orderedjson.Parse` must not clean: `fmt` and `generate` write the same tree back to disk and would rewrite the user's file.

### A contract change

`help -j` carried `\r\n` inside a `detail` verbatim. After this change it carries `\n`, a newline or tab inside a single-line field (`desc`, `example`, a label) becomes a space, and any string with a character from the lists above carries U+FFFD. The JSON shapes are documented as a contract, so this is stated here and in the release notes.

### Not in scope

- Display width. `wrap` counts runes, so CJK and emoji overflow a narrow terminal. It is a layout limit and applies to `desc` too.
- `just`'s own output, and anything a recipe prints.
- Unicode confusables in names beyond the identifier rule.
- How zsh draws a control character inside a completion description. Not tested; after the model is cleaned there is none.
- `TERM=dumb` and enabling VT processing on Windows: existing gaps for the colors, not changed here.

### Tests

A table for `Text` and `Line`: ESC sequences, an OSC title, BEL, NUL, lone CR, CRLF, U+009B, U+0085, U+2028, bidi overrides and isolates, tag characters, valid text in several scripts, emoji ZWJ sequences, RLM, tab and newline, and the fast path (clean text comes back identical). Then every entry point with a registry and a native justfile that carry the bytes: summary, expanded, zooms, browse entries, completion, `resolve`, `check` and `assess` text and JSON, and the message paths. Each asserts that the output has no byte below 0x20 other than newline (and tab where allowed), no C1, no bidi and no tag character, and that a forged row does not appear.

## 2. Highlighting in `detail`

### Proposal

When colors are on, the zoom highlights these in `detail`, and nothing else. It styles; it never adds or removes a character.

| Token | Style |
|---|---|
| A backtick span (`` `just ewf report --open` ``) | green, with a flag inside cyan and a `<placeholder>` inside yellow |
| A flag outside a span (`--dry-run`, `-w`, `--out=PATH`) | cyan, the whole token including `=value` |
| A placeholder outside a span (`<slug>`) | yellow |
| A URL (`http://`, `https://`) | underlined |

Not highlighted: bold, italic, headings, lists, tables, `UPPER_CASE` words, file paths, quoted strings. They make `detail` harder to read as plain text in the registry, in `-j` and for agents.

### Token rules

- **Order of recognition:** span first, then URL, then flag, then placeholder. A span wins over a URL inside it. A URL in angle brackets (`<https://x>`) is a URL.
- **Punctuation:** before classifying a word, a leading `( [ { " ' <` and a trailing `) ] } " ' , ; : . ! ?` are set aside and not styled. For a URL, a trailing `)` stays when the URL has an unmatched `(`, so `Foo_(bar)` keeps it and `(see https://x/y).` underlines `https://x/y`. The same trim applies to flags and placeholders (`<slug>,`, `(--flag)`).
- **Backticks** pair left to right inside one source line (detail is split on newlines first, so a span never crosses one). An opening backtick must follow the start or a space or an opening bracket, and a closing one must precede a space, the end or closing punctuation. An unmatched backtick and an empty pair are plain text, so one stray backtick cannot colour the rest of the paragraph.
- **Flags** in prose match `^--?[A-Za-z][A-Za-z0-9_-]*(=.*)?$`. Underscores are allowed, unlike the Examples pattern, which is not changed. A bare `-`, `--`, `-1` and `3-5` are not flags, and `e-mail` and `well-known` do not start with a dash.
- **Placeholders** match `^<[^<>'"]*>$` as a whole word after the trim. A path made of several (`<A>/<b>/raw`) is plain.

### The rule that keeps it safe

**Highlighting only adds color.** With colors off (a pipe, `NO_COLOR`, an agent) the output is byte for byte what it is today. With colors on, removing the escape sequences gives exactly the colors-off output. The backticks and every other character are kept. A test asserts this for every fixture, and the existing `ansi` expression already strips every code the new styling uses.

### How

The word-by-word idea in the first draft was the wrong unit: a backtick can sit inside a word, a span can hold several words, and a long word is cut by runes. Instead:

1. Classify each rune of a paragraph into a style, with the rules above.
2. Generalise `wrap` to return the rune ranges of each line. The existing `wrap` becomes a thin caller of it, so the line breaks fall where they fall today by construction, and `desc` and the option rows are unchanged.
3. Emit each line from its ranges, opening a style when it starts and closing it with a targeted code (`ESC[39m` for color, `ESC[24m` for underline) before the line ends or the style changes. A break never leaves a style open, and the code does not use a full reset, which would end an outer `dim` if `desc` is ever highlighted too.

The span styling does not reuse `colorizeCmd`: it wraps a whole line in green and restores green after a flag, which assumes it is inside a command, and quote handling is wrong for prose (apostrophes). The span applies the same colors with its own small scanner.

This is a larger change than a line count suggests: the `wrap` refactor and the tests, about 150 lines.

### Where it shows

`detail` in the zoom only. Examples already have styling, and `desc` and the arguments appear in rows where alignment matters.

In `--browse` the preview runs `justz help <entry>` with its output piped, so colors are off and the preview shows plain text. After a selection the zoom is printed to the terminal with colors. Coloring the preview would need a new decision (an environment variable the preview sets) and is not part of this note.

`TestZoomDetailColorIsOnlyOnTheHeading` asserts that nothing but the heading is colored; it changes in this PR to assert the new styles and the invariant.

### Tests

Each token class with colors on and off; a span across a line break; an unmatched and a doubled backtick; a backtick inside a word; a URL longer than the width (cut, and each piece underlined); punctuation around a URL, a flag and a placeholder; `--flag=value`; `--no_cache`; a negative number; `e-mail`; a URL in a span; a span with quotes; and the invariant over the existing fixtures.

## Plan

1. `internal/textsafe`, the pass over the model, the identifier rule for completion and `resolve`, the `check` and `assess` construction, the `%q` messages, and the tests of section 1. One PR.
2. The `wrap` refactor, the classifier and the tests of section 2. One PR, after the first.
3. At release time, the docs that describe `detail` also say what is highlighted, and the README gets one line about the sanitization. The release notes state the `-j` contract change.

## Open questions

1. **Replace or delete.** Replace with U+FFFD, so tampering stays visible. Recommendation: replace.
2. **The character set.** The list above: no zero-width or LRM/RLM/ALM bans, tag characters and the isolates banned. Recommendation: as listed.
3. **JSON.** Clean `help -j`, `check --json` and `assess --json`. Recommendation: yes, at construction, so text and JSON agree.
4. **Names.** Completion and `resolve` only use keys that match the identifier rule. Recommendation: yes; a key with a space cannot be run as a command anyway.
5. **Backticks.** Keep them visible when colors are on, so the invariant holds. Recommendation: keep.
6. **URLs.** Underline only, no OSC 8 hyperlinks: some terminals print the raw escape, and a link whose target differs from its text is the spoofing this note removes. Recommendation: underline only.
7. **Environment variables and paths.** Not highlighted: a rule that colors `GATEWAY_MODE` also colors `ZMQ` and `JSON`. Recommendation: leave out.
8. **Underscores in prose flags.** Allowed in the prose pattern, not in the shared Examples pattern. Recommendation: allow in prose only.
