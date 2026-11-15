package help

import (
	"strings"
	"testing"
)

// marked shows the escape codes of a styled line as readable tags.
func marked(s string) string {
	r := strings.NewReplacer("\x1b[32m", "{g}", "\x1b[36m", "{c}", "\x1b[33m", "{y}", "\x1b[4m", "{u}", "\x1b[39m", "{/}", "\x1b[24m", "{/u}")
	return r.Replace(s)
}

func hlOne(text string) string {
	return marked(strings.Join(style{on: true}.highlightLines(200, text), "\n"))
}

func TestHighlightTokens(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "just words here", "just words here"},
		{"span", "run `just ewf report` now", "run {g}`just ewf report`{/} now"},
		{"span with flag and placeholder", "run `just x --open <slug>` now", "run {g}`just x {c}--open{g} {y}<slug>{g}`{/} now"},
		{"span with flag value", "`go test --run=Foo`", "{g}`go test {c}--run=Foo{g}`{/}"},
		{"span before punctuation", "see (`x`), then", "see ({g}`x`{/}), then"},
		{"flag", "pass --dry-run to it", "pass {c}--dry-run{/} to it"},
		{"short flag", "use -w here", "use {c}-w{/} here"},
		{"flag with value", "use --out=PATH, please", "use {c}--out=PATH{/}, please"},
		{"flag with underscore", "use --no_cache now", "use {c}--no_cache{/} now"},
		{"flag in parens", "(--flag)", "({c}--flag{/})"},
		{"placeholder", "give <slug> now", "give {y}<slug>{/} now"},
		{"placeholder with comma", "give <slug>, then", "give {y}<slug>{/}, then"},
		{"path of placeholders is plain", "in <A>/<b>/raw", "in <A>/<b>/raw"},
		{"url", "see https://x.test/y now", "see {u}https://x.test/y{/u} now"},
		{"url in parens with dot", "(see https://x.test/y).", "(see {u}https://x.test/y{/u})."},
		{"url keeps its closing paren", "see https://en.test/Foo_(bar) now", "see {u}https://en.test/Foo_(bar){/u} now"},
		{"url in angle brackets", "see <https://x.test/y> now", "see <{u}https://x.test/y{/u}> now"},
		{"url in a span stays green", "`curl https://x.test/y`", "{g}`curl https://x.test/y`{/}"},
		{"bare scheme is not a url", "http:// alone", "http:// alone"},
		{"negative number", "got -1 and 3-5", "got -1 and 3-5"},
		{"bare dashes", "a - b -- c", "a - b -- c"},
		{"hyphenated words", "an e-mail and well-known thing", "an e-mail and well-known thing"},
		{"upper case and paths", "set EWF_DEPLOY_LEGACY_STATIC in /etc/x", "set EWF_DEPLOY_LEGACY_STATIC in /etc/x"},
		{"unmatched backtick", "a ` b --flag c", "a ` b {c}--flag{/} c"},
		{"doubled backtick", "a `` b --flag c", "a `` b {c}--flag{/} c"},
		{"backtick inside a word", "a`b c", "a`b c"},
		{"flag in brackets", "[--f]", "[{c}--f{/}]"},
		{"flag in braces", "{--f}", "{{c}--f{/}}"},
		{"span after an angle bracket", "<`a`>", "<{g}`a`{/}>"},
		{"span before an angle bracket", "`a --f`>", "{g}`a {c}--f{g}`{/}>"},
		{"url with an unmatched close paren", "see https://x.test/a) now", "see {u}https://x.test/a{/u}) now"},
		{"url with a matched paren and a close paren", "(https://x.test/a_(b))", "({u}https://x.test/a_(b){/u})"},
		{"blank runs between words", "a   --f    `x   y`   b", "a {c}--f{/} {g}`x y`{/} b"},
		{"closing backtick needs a boundary", "`a`b --f", "`a`b {c}--f{/}"},
		{"two spans", "`a` and `b --f`", "{g}`a`{/} and {g}`b {c}--f{g}`{/}"},
		{"quoted span words", "`say 'hi' --now`", "{g}`say 'hi' {c}--now{g}`{/}"},
	}
	for _, tt := range tests {
		if got := hlOne(tt.in); got != tt.want {
			t.Errorf("%s:\n in   %q\n got  %q\n want %q", tt.name, tt.in, got, tt.want)
		}
	}
}

// A span never crosses a source line, so one stray backtick cannot colour the
// rest of the text.
func TestHighlightSpanDoesNotCrossALine(t *testing.T) {
	if got, want := hlOne("a `b\nc` d --x"), "a `b\nc` d {c}--x{/}"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestHighlightIsOnlyColor(t *testing.T) {
	texts := []string{
		"Builds `just ewf report --open <slug>` and see https://example.test/a/b?c=d, then (--dry-run).\n\nSecond   paragraph with  `a b` spans, -w and <x>.",
		"a ` unmatched `` pair e-mail -1 https://x.test/very/long/url/that/will/be/cut/at/the/width/of/the/view/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa end",
		"`a very long span of words that is wrapped across several lines because the width is small --flag-in-span <ph> and more`",
		"\n\n  indented \t tabs\t`x`  \n",
		"",
	}
	for _, width := range []int{10, 20, 40, 94} {
		for _, text := range texts {
			plain := wrap(width, text)
			colored := style{on: true}.highlightLines(width, text)
			if len(plain) != len(colored) {
				t.Fatalf("width %d: %d lines, want %d for %q", width, len(colored), len(plain), text)
			}
			for i := range plain {
				if got := ansi.ReplaceAllString(colored[i], ""); got != plain[i] {
					t.Errorf("width %d line %d: stripped %q, plain %q", width, i, got, plain[i])
				}
				assertBalanced(t, colored[i])
			}
			if off := (style{}).highlightLines(width, text); strings.Join(off, "\n") != strings.Join(plain, "\n") {
				t.Errorf("colors off differs from wrap for %q", text)
			}
		}
	}
}

// Every line closes what it opened, so a style never leaks into the next one.
func assertBalanced(t *testing.T, line string) {
	t.Helper()
	color, under := false, false
	for _, m := range ansi.FindAllString(line, -1) {
		switch m {
		case "\x1b[32m", "\x1b[36m", "\x1b[33m":
			color = true
		case "\x1b[39m":
			color = false
		case "\x1b[4m":
			under = true
		case "\x1b[24m":
			under = false
		default:
			t.Errorf("unexpected code %q in %q", m, line)
		}
	}
	if color || under {
		t.Errorf("a style is still open at the end of %q", marked(line))
	}
}

func TestHighlightCutUrlIsUnderlinedOnEveryPiece(t *testing.T) {
	lines := style{on: true}.highlightLines(12, "see https://example.test/aaaaaaaaaaaaaaaaaaaa now")
	n := 0
	for _, l := range lines {
		if strings.Contains(l, "\x1b[4m") {
			n++
		}
	}
	if n < 3 {
		t.Errorf("a cut URL should be underlined on each piece, got %d in %q", n, lines)
	}
}

func TestHighlightSpanAcrossALineBreakKeepsItsColor(t *testing.T) {
	lines := style{on: true}.highlightLines(16, "`aaaa bbbb cccc dddd eeee`")
	if len(lines) < 2 {
		t.Fatalf("expected a wrap, got %q", lines)
	}
	for _, l := range lines {
		if !strings.HasPrefix(l, "\x1b[32m") || !strings.HasSuffix(l, "\x1b[39m") {
			t.Errorf("each line of the span should be green on its own: %q", marked(l))
		}
	}
}

// The joining blank of a wrapped line comes from the first blank of the run
// between two words, so a run of blanks is one blank on the line, and it is
// green only inside a span.
func TestHighlightJoinBlankFollowsItsSource(t *testing.T) {
	lines := style{on: true}.highlightLines(40, "`a     b`   c    --f")
	if got, want := marked(strings.Join(lines, "\n")), "{g}`a b`{/} c {c}--f{/}"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// wrap works on runes, so a byte that is not UTF-8 comes out as U+FFFD. The
// registry never holds one (the JSON decoder replaces it); a native justfile
// doc could.
func TestWrapReplacesInvalidUTF8(t *testing.T) {
	if got := wrap(20, "a\xffb c"); len(got) != 1 || got[0] != "a\ufffdb c" {
		t.Errorf("wrap = %q", got)
	}
}
