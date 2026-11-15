// Package textsafe makes text that comes from a project safe to print on a
// terminal. A registry or a justfile can hold escape sequences, and a terminal
// acts on them: it can change the window title, move the cursor over earlier
// output, write to the clipboard or reorder what is shown. The functions here
// replace such characters with U+FFFD, so the residue shows that something was
// there, and leave ordinary text alone.
package textsafe

import (
	"io"
	"strings"
	"unicode/utf8"
)

const replacement = '�'

// Text is for prose that may span lines. Newlines and tabs stay; a CRLF pair
// becomes a newline; every other dangerous character becomes U+FFFD.
func Text(s string) string {
	return clean(s, false)
}

// Line is for text that sits in one row: names, labels, descriptions, paths.
// Like Text, but a newline or a tab becomes one space, so a value cannot start
// a forged row.
func Line(s string) string {
	return clean(s, true)
}

func clean(s string, oneLine bool) string {
	if !needsWork(s, oneLine) {
		return s
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n' || r == '\t':
			if oneLine {
				b.WriteByte(' ')
			} else {
				b.WriteRune(r)
			}
		case dangerous(r):
			b.WriteRune(replacement)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// needsWork is the fast path: clean text comes back unchanged and without a
// copy. A newline or tab only needs work in a line.
func needsWork(s string, oneLine bool) bool {
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			if c == '\n' || c == '\t' {
				if oneLine {
					return true
				}
				i++
				continue
			}
			if c < 0x20 || c == 0x7f {
				return true
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			return true
		}
		if dangerous(r) {
			return true
		}
		i += size
	}
	return false
}

// dangerous reports whether a terminal could act on r, or whether r changes
// how the text around it is displayed. A newline and a tab are not in this
// set; the caller decides about them.
func dangerous(r rune) bool {
	switch {
	case r < 0x20 && r != '\n' && r != '\t':
		return true
	case r == 0x7f:
		return true
	case r >= 0x80 && r <= 0x9f: // C1 controls, including U+009B (CSI) and U+0085
		return true
	case r == 0x2028 || r == 0x2029:
		return true
	case r >= 0x202a && r <= 0x202e: // bidirectional embeddings and overrides
		return true
	case r >= 0x2066 && r <= 0x2069: // bidirectional isolates
		return true
	case r >= 0x206a && r <= 0x206f: // deprecated format controls
		return true
	case r >= 0xe0000 && r <= 0xe007f: // tag characters: invisible text
		return true
	}
	return false
}

// IsValue reports whether s can be offered as an option value that a shell
// will consume as one word: letters, digits and _ . , : @ % + = / ~ -.
func IsValue(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case strings.IndexByte("_.,:@%+=/~-", c) >= 0:
		default:
			return false
		}
	}
	return true
}

// IsName reports whether s can be offered or printed as a group or command
// name that a shell will consume: letters, digits, underscore, dot and dash,
// and not starting with a dot or a dash.
func IsName(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_':
		case (c == '.' || c == '-') && i > 0:
		default:
			return false
		}
	}
	return true
}

type writer struct{ w io.Writer }

// NewWriter returns a writer that passes everything through Text, for the
// messages a program prints on its own standard error, where an echoed path or
// name can carry a control character. It reports the length of what it was
// given, so callers see no short write. A multi-byte character split across two
// writes is replaced; one Fprintf is one write.
func NewWriter(w io.Writer) io.Writer { return writer{w} }

func (x writer) Write(p []byte) (int, error) {
	if _, err := io.WriteString(x.w, Text(string(p))); err != nil {
		return 0, err
	}
	return len(p), nil
}
