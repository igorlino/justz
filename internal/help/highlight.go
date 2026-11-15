package help

import (
	"regexp"
	"strings"
	"unicode"
)

// Highlighting of a command's detail text. It only adds color: with the escape
// sequences removed, the output is exactly what the plain wrap gives.

// A style is a color in the low bits and an underline flag.
type hlStyle uint8

const (
	hlGreen  hlStyle = 1 // a backtick span
	hlCyan   hlStyle = 2 // a flag
	hlYellow hlStyle = 3 // a <placeholder>
	hlColor  hlStyle = 3
	hlUnder  hlStyle = 4 // a URL
)

const (
	leadPunct  = "([{\"'"
	trailPunct = ")]}\"',;:.!?"
)

func isBlank(r rune) bool { return unicode.IsSpace(r) }

// classify gives every rune of one paragraph (text with no newline) its style.
func classify(r []rune) []hlStyle {
	st := make([]hlStyle, len(r))
	inSpan := make([]bool, len(r))

	// Backtick spans, left to right. An opening backtick follows the start, a
	// blank or an opening bracket, and a closing one precedes a blank, the end
	// or closing punctuation. An empty pair and an unmatched backtick are text.
	for i := 0; i < len(r); i++ {
		if r[i] != '`' || !(i == 0 || isBlank(r[i-1]) || strings.ContainsRune(leadPunct+"<", r[i-1])) {
			continue
		}
		if i+1 < len(r) && r[i+1] == '`' {
			i++
			continue
		}
		for j := i + 1; j < len(r); j++ {
			if r[j] == '`' && (j+1 == len(r) || isBlank(r[j+1]) || strings.ContainsRune(trailPunct+">", r[j+1])) {
				for k := i; k <= j; k++ {
					st[k], inSpan[k] = hlGreen, true
				}
				// The words inside a span: flags and placeholders stand out.
				words(r, i+1, j, func(a, b int) {
					if c := wordStyle(r[a:b]); c != 0 {
						for k := a; k < b; k++ {
							st[k] = c
						}
					}
				})
				i = j
				break
			}
		}
	}

	// Words outside spans.
	i := 0
	for i < len(r) {
		if isBlank(r[i]) || inSpan[i] {
			i++
			continue
		}
		j := i
		for j < len(r) && !isBlank(r[j]) && !inSpan[j] {
			j++
		}
		a, b := trimWord(r, i, j)
		c := hlStyle(0)
		if a < b && r[a] == '<' && hasURLPrefix(string(r[a+1:b])) {
			// An angle-bracketed URL: underline the URL, not the brackets.
			a++
			if r[b-1] == '>' {
				b--
			}
			c = hlUnder
		} else if hasURLPrefix(string(r[a:b])) {
			c = hlUnder
		} else {
			c = wordStyle(r[a:b])
		}
		for k := a; k < b; k++ {
			st[k] = c
		}
		i = j
	}
	return st
}

// words calls f with the bounds of each run of non-blanks in r[from:to].
func words(r []rune, from, to int, f func(a, b int)) {
	i := from
	for i < to {
		if isBlank(r[i]) {
			i++
			continue
		}
		j := i
		for j < to && !isBlank(r[j]) {
			j++
		}
		a, b := trimWord(r, i, j)
		f(a, b)
		i = j
	}
}

// trimWord sets aside the punctuation around a word and returns the bounds of
// what is left. The "<" of a placeholder or of an angle-bracketed URL is not
// punctuation here. A ")" that closes a "(" inside the word stays.
func trimWord(r []rune, a, b int) (int, int) {
	for a < b && strings.ContainsRune(leadPunct, r[a]) {
		a++
	}
	for b > a && strings.ContainsRune(trailPunct, r[b-1]) {
		if r[b-1] == ')' && count(r[a:b-1], '(') > count(r[a:b-1], ')') {
			break
		}
		b--
	}
	return a, b
}

func count(r []rune, c rune) int {
	n := 0
	for _, x := range r {
		if x == c {
			n++
		}
	}
	return n
}

// flagInProse is a flag as prose writes it. Unlike the Examples pattern it
// allows underscores (--no_cache).
var flagInProse = regexp.MustCompile(`^--?[A-Za-z][A-Za-z0-9_-]*(=.*)?$`)

// wordStyle is the style of one trimmed word that is not a URL, or 0.
func wordStyle(w []rune) hlStyle {
	s := string(w)
	switch {
	case flagInProse.MatchString(s):
		return hlCyan
	case placeholderToken.MatchString(s):
		return hlYellow
	}
	return 0
}

func hasURLPrefix(s string) bool {
	for _, p := range []string{"http://", "https://"} {
		if strings.HasPrefix(s, p) && len(s) > len(p) {
			return true
		}
	}
	return false
}

// highlightLines wraps text like wrap and styles each line. The line breaks
// are those of wrap. Each line closes what it opens, with the code for that
// attribute alone (39 for color, 24 for underline), so nothing leaks into the
// next line and an outer style is not reset.
func (s style) highlightLines(width int, text string) []string {
	if !s.on {
		return wrap(width, text)
	}
	src := []rune(text)
	// Styles per paragraph: classify each source line, in text order.
	sts := make([]hlStyle, len(src))
	for start := 0; start <= len(src); {
		end := start
		for end < len(src) && src[end] != '\n' {
			end++
		}
		copy(sts[start:end], classify(src[start:end]))
		start = end + 1
	}
	lines := wrapRunes(width, text)
	out := make([]string, len(lines))
	for n, l := range lines {
		var b strings.Builder
		var cur hlStyle
		for i, r := range l.runes {
			next := sts[l.src[i]]
			if next != cur {
				b.WriteString(transition(cur, next))
				cur = next
			}
			b.WriteRune(r)
		}
		b.WriteString(transition(cur, 0))
		out[n] = b.String()
	}
	return out
}

func transition(from, to hlStyle) string {
	var b strings.Builder
	if from&hlUnder != 0 && to&hlUnder == 0 {
		b.WriteString("\x1b[24m")
	}
	if from&hlColor != 0 && to&hlColor != from&hlColor {
		if to&hlColor == 0 {
			b.WriteString("\x1b[39m")
		}
	}
	if to&hlColor != 0 && to&hlColor != from&hlColor {
		b.WriteString(map[hlStyle]string{hlGreen: "\x1b[32m", hlCyan: "\x1b[36m", hlYellow: "\x1b[33m"}[to&hlColor])
	}
	if to&hlUnder != 0 && from&hlUnder == 0 {
		b.WriteString("\x1b[4m")
	}
	return b.String()
}
