package lint

import (
	"fmt"
	"regexp"
	"strings"
)

// interpolations is what the body of a recipe has of {{ }}. The counts decide
// whether moving the body to a script is mechanical: a value can be exported
// and read from the environment, except inside single quotes or a quoted
// here-document, where the shell does not expand it.
type interpolations struct {
	Total   int // interpolations, escapes not counted
	Quoted  int // those inside single quotes or a quoted here-document
	Escaped int // {{{{ and {{"{{"}}: text that only looks like an interpolation
}

// heredocStart finds <<TAG, <<'TAG', <<"TAG" and <<-TAG. The leading run of <
// is captured so that a <<< here-string is told from a here-document.
var heredocStart = regexp.MustCompile(`(<+)(-?)\s*(?:'([^']*)'|"([^"]*)"|([A-Za-z_][A-Za-z0-9_]*))`)

// quoteState follows the quoting of one shell, roughly.
type quoteState struct {
	single, double bool
	ansi           bool // inside $'...', where a backslash escapes the quote
	esc            bool // the next character is escaped
	prev           rune // the last character seen, for the start of a word
	comment        bool // an unquoted # that starts a word has been seen
}

// scan follows s and returns the offset where a comment starts, or -1.
func (q *quoteState) scan(s string) int {
	for off, r := range s {
		prev := q.prev
		q.prev = r
		if q.comment {
			continue
		}
		switch {
		case q.esc:
			q.esc = false
		case r == '\\' && (!q.single || q.ansi):
			q.esc = true
		case q.single:
			if r == '\'' {
				q.single, q.ansi = false, false
			}
		case q.double:
			if r == '"' {
				q.double = false
			}
		case r == '\'':
			q.single, q.ansi = true, prev == '$'
		case r == '"':
			q.double = true
		case r == '#' && (prev == 0 || prev == ' ' || prev == '\t'):
			q.comment = true
			return off
		}
	}
	return -1
}

// countInterpolations reads the body as the dump gives it: a line is a list of
// fragments, text (a string) or an interpolation (a list holding one
// expression). just keeps {{{{ as text, and {{"{{"}} is an interpolation of
// the string {{, so both are counted as escapes. The quote state is followed
// roughly, across lines for a shebang recipe (one script) and per line
// otherwise (one shell each). Text after an unquoted # that starts a word is a
// comment and is not scanned. It is advice, not a shell parser.
func countInterpolations(body []any, shebang bool) interpolations {
	var out interpolations
	var q quoteState
	heredoc, heredocQuoted, heredocDash := "", false, false

	for _, raw := range body {
		frags, ok := raw.([]any)
		if !ok {
			continue
		}
		q.comment, q.esc, q.prev = false, false, 0
		if !shebang {
			q = quoteState{}
		}
		// The text of the line, to find a here-document terminator.
		var text strings.Builder
		for _, fr := range frags {
			if s, ok := fr.(string); ok {
				text.WriteString(s)
			}
		}
		line := text.String()
		if heredoc != "" {
			t := line
			if heredocDash {
				t = strings.TrimLeft(t, "\t")
			}
			if len(frags) == 1 && isText(frags[0]) && strings.TrimRight(t, " ") == heredoc {
				heredoc = ""
				continue
			}
		}

		code := len(line) // the part of the line before a comment
		var seen int      // text consumed so far
		for _, fr := range frags {
			if s, ok := fr.(string); ok {
				out.Escaped += strings.Count(s, "{{{{")
				if heredoc == "" {
					if at := q.scan(s); at >= 0 && seen+at < code {
						code = seen + at
					}
				}
				seen += len(s)
				continue
			}
			// An interpolation: a list with one expression.
			exprs, ok := fr.([]any)
			if !ok {
				continue
			}
			if len(exprs) == 1 {
				if lit, ok := exprs[0].(string); ok && lit == "{{" {
					out.Escaped++
					continue
				}
			}
			out.Total++
			if heredoc == "" {
				q.prev = 'x' // a word character: a # right after is not a comment
			}
			if (heredoc == "" && q.single && !q.comment) || (heredoc != "" && heredocQuoted) {
				out.Quoted++
			}
		}

		// A here-document only exists in a script: a linewise recipe runs each
		// line in its own shell.
		if heredoc == "" && shebang {
			if tag, quoted, dash, ok := findHeredoc(line[:code]); ok {
				heredoc, heredocQuoted, heredocDash = tag, quoted, dash
			}
		}
	}
	return out
}

// findHeredoc finds the opener of a here-document in the code of a line. A <<<
// here-string, a shift inside (( )), and an unquoted tag that is not followed by
// white space, the end of the line or a command separator are not openers.
func findHeredoc(code string) (tag string, quoted, dash, ok bool) {
	for _, m := range heredocStart.FindAllStringSubmatchIndex(code, -1) {
		if m[3]-m[2] != 2 { // not exactly <<
			continue
		}
		before := code[:m[0]]
		if strings.Count(before, "((") > strings.Count(before, "))") {
			continue // arithmetic: $((a<<b))
		}
		sub := func(g int) string {
			if m[2*g] < 0 {
				return ""
			}
			return code[m[2*g]:m[2*g+1]]
		}
		tag = sub(3) + sub(4) + sub(5)
		quoted = sub(5) == ""
		if !quoted {
			if rest := code[m[1]:]; rest != "" && !strings.ContainsRune(" \t;|&<>", rune(rest[0])) {
				continue
			}
		}
		return tag, quoted, sub(2) == "-", true
	}
	return "", false, false, false
}

func isText(v any) bool {
	_, ok := v.(string)
	return ok
}

// describe is the part of the recipe-long message about interpolations.
func (in interpolations) describe() string {
	var b strings.Builder
	switch {
	case in.Total == 0:
		b.WriteString(", no interpolations")
	case in.Total == 1:
		b.WriteString(", 1 interpolation")
	default:
		fmt.Fprintf(&b, ", %d interpolations", in.Total)
	}
	if in.Quoted > 0 {
		fmt.Fprintf(&b, " (%d in single quotes or a quoted here-document: not mechanical to move)", in.Quoted)
	}
	if in.Escaped > 0 {
		fmt.Fprintf(&b, ", %d escaped", in.Escaped)
	}
	return b.String()
}
