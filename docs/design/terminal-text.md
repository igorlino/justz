# Design note: terminal-safe registry text, and highlighting in `detail`

Status: proposed. Two changes in one note, in two PRs: first the sanitization, then the highlighting. The first is a security fix and stands without the second.

## 1. Registry text must not control the terminal

### Problem

`justz help`, `jzh`, completion, `--browse`, `check` and `assess` print text that comes from files of the project in the current directory: `commands.json` (names, descriptions, examples, argument rows, labels), and for a native project the `[doc]` and group text of the justfile. Anyone who controls those files, for example the author of a repository you just cloned, controls those strings.

The views print them as they are. Measured on the current code with a registry whose `desc` and `detail` hold `\u001b[31mRED\u001b[0m`, `\u001b]0;title\u0007` and a NUL byte: all three reach the terminal raw (`cat -v` shows `^[[31m`, `^[]0;title^G`, `^@`). The same holds for `desc` in the summary. JSON output is not affected, because JSON escapes the bytes.

What a terminal does with such bytes, depending on the terminal: change the window title, move the cursor and overwrite earlier lines so the output reads as something else, hide text, write to the clipboard (OSC 52), and show a link whose text and target differ. `justz help` is the command a user runs first in an unfamiliar repository, which is the case to protect.

### Proposal

One function, `textsafe.Clean(s string) string`, in its own small package. It replaces with U+FFFD:

- C0 controls except newline and tab, which are the only ones the views use (ESC, NUL, BEL, CR, BS, VT, FF and the rest);
- DEL;
- C1 controls (U+0080 to U+009F), because some terminals treat U+009B as a CSI introducer;
- the bidirectional overrides and isolates (U+202A to U+202E and U+2066 to U+2069), which reorder what is displayed.

A carriage return is replaced too. A CRLF inside a JSON string is an escape, so it can only reach the views as a deliberate `\r\n`; the detail cleaner turns that pair into a newline first, as it does today.

It does not touch ordinary text: letters in any script, emoji, combining marks, tabs, newlines, the em dash.

### Where it applies

At the point where registry text enters the program, so every view and the JSON show the same text:

- `help.FromRegistry` and the native reader: every string field (name, tagline, cli, titles, labels, descriptions, `detail`, `footer`, examples, argument names, descriptions and `values`, `seeAlso`, tier labels and hints). Completion, browse, resolve and every help view read the model, so they are covered by this one place.
- The text printers of `check` and `assess`, which put registry keys and descriptions into messages (`registry-desc-diverges` prints both texts, `registry-stale` prints a key). These clean at the printer, so the JSON of those commands keeps the exact strings that were found, which is what a tool comparing them wants.
- `lint` prints recipe names, which `just` restricts to identifiers, and rule ids. No change.

Replacing with U+FFFD, not deleting, is deliberate: `\u001b[31mRED` becomes `�[31mRED`. The residue is visible, so the reader sees that something was there. Deleting would show a clean-looking `[31mRED`.

JSON output of `help` carries the cleaned text. A tool that reads `-j` and prints the strings (an agent, a script) is a terminal path too, and one cleaned text everywhere is simpler to reason about than two.

### Not in scope

- Display width. `wrap` counts runes, so CJK and emoji overflow a narrow terminal. That is a layout limit, not a safety one, and it affects `desc` too.
- Text from `just` itself (its own error output) and from recipes that run.
- Terminal escapes that `justz` writes on purpose (the colors).

### Tests

A table of strings (ESC sequences, an OSC title, BEL, NUL, CR, U+009B, a bidi override, valid text in several scripts, tab and newline) for `Clean`. Then each entry point with a registry that carries the bytes: the summary, the expanded view, the zoom, the group zoom, `-j`, completion, `check` and `assess` text, and the native reader. Each asserts that no byte below 0x20 other than newline and tab, and no C1 or bidi code point, is in the output.

## 2. Highlighting in `detail`

### Proposal

When colors are on, the zoom highlights three things in `detail`, and nothing else. It styles; it never adds or removes a character.

| Token | Style | Notes |
|---|---|---|
| A backtick span (`` `just ewf report --open` ``) | like a command in the Examples section: green, with flags cyan and `<placeholders>` yellow inside | the backticks stay in the text |
| A flag (`--dry-run`, `-w`, `--out=PATH`) outside a span | cyan | same pattern as the Examples |
| A placeholder (`<slug>`) outside a span | yellow | same pattern as the Examples |
| A URL (`http://` or `https://`) | underlined | no clickable-link escape (OSC 8) |

Trailing and leading punctuation next to a token is not styled: `(see https://example.invalid/x).` underlines the URL and not the `)` and `.`. A backtick without a partner on the same line of the source text is plain text.

Not highlighted: bold, italic, headings, lists, tables, `UPPER_CASE` words (too many acronyms), file paths, quoted strings. They would make `detail` harder to read as plain text in the registry, in `-j` and for agents, and they need a parser and escaping rules.

### The rule that keeps it safe

**Highlighting only adds color.** With colors off (a pipe, `NO_COLOR`, an agent) the output is byte for byte what it is today. With colors on, removing the escape sequences gives exactly the colors-off output. The backticks and every other character are kept. A test asserts this for every fixture.

### How

Today `wrap` breaks plain text into lines and the zoom prints them. Styling after the break fails for a backtick span that contains spaces and so spans a break, and for a long URL that the wrap cuts into pieces. So the styled wrap works on words:

1. Split the paragraph into words and classify each word (inside a span, flag, placeholder, URL, plain), carrying the "inside a span" state from word to word.
2. Run the same line-breaking as `wrap`, counting only the visible runes of a word, so the breaks fall where they fall today.
3. Emit each word with its style, closed at the end of the word, so a line break never leaves a color open.

The code goes in `internal/help` next to `colorizeCmd`, which it reuses for the inside of a span. About 60 lines.

### Scope

`detail` in the zoom only. The one-line `desc` is also shown in listings where column alignment matters, and the Examples already have their own styling. Applying the same highlighting to `desc` in the zoom is a later, small step.

### Tests

Per token class, with colors on and off. Then: a span across a line break, an unmatched backtick, a URL longer than the width (cut, and each piece underlined), punctuation around a URL and a flag, a flag-like word in prose, `--flag=value`, a negative number (not a flag), and the invariant over the existing fixtures: colors-off output is unchanged, and colors-on output with the escapes removed equals it.

## Plan

1. `internal/textsafe`, the model and printer changes, and the tests of section 1. One PR.
2. The styled wrap and the tests of section 2. One PR, after the first.
3. At release time, the docs that describe `detail` also say what is highlighted, and the README gets one line about the sanitization. They wait for the release, as before.

## Open questions

1. **Replace or delete.** Replace with U+FFFD, so tampering stays visible. Recommendation: replace.
2. **Bidi controls.** Include U+202A to U+202E and U+2066 to U+2069. Recommendation: include; legitimate registry text does not use them.
3. **JSON.** Clean the strings in `help -j` too. Recommendation: yes, so a tool that forwards the strings to a terminal is also safe. `check` and `assess` keep exact strings in their JSON and clean only their text output.
4. **Backticks.** Keep them visible when colors are on, so the invariant holds. Recommendation: keep.
5. **URLs.** Underline only. Recommendation: no OSC 8 hyperlinks for now: some terminals show the raw escape, and a link whose target differs from its text is the spoofing this note removes.
6. **Environment variables and paths.** Not highlighted. Recommendation: leave out; a rule that colors `GATEWAY_MODE` also colors `ZMQ` and `JSON`.
