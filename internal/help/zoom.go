package help

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// viewWidth is the width of a zoom view: the terminal width, at most 100 and at
// least 40.
func (o Options) viewWidth() int {
	w := o.width()
	if w > 100 {
		w = 100
	}
	if w < 40 {
		w = 40
	}
	return w
}

// wrap breaks text at spaces into lines of at most width columns, and cuts a
// word that is longer than a line. Trailing blanks are stripped. A newline in
// the text starts a new line.
func wrap(width int, text string) []string {
	lines := wrapRunes(width, text)
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = string(l.runes)
	}
	return out
}

// wrapLine is one wrapped line. src[i] is the index, in the runes of the
// original text, of the rune runes[i] came from. The space that joins two words
// comes from the first blank of the run between them.
type wrapLine struct {
	runes []rune
	src   []int
}

// wrapRunes is wrap with the source position of every rune, so a caller can
// style the output by what the text said at that place.
func wrapRunes(width int, text string) []wrapLine {
	src := []rune(text)
	var out []wrapLine
	start := 0
	for start <= len(src) {
		end := start
		for end < len(src) && src[end] != '\n' {
			end++
		}
		out = append(out, wrapParagraph(width, src, start, end)...)
		start = end + 1
	}
	if len(out) == 0 {
		return []wrapLine{{}}
	}
	return out
}

// wrapParagraph wraps src[from:to], which holds no newline.
func wrapParagraph(width int, src []rune, from, to int) []wrapLine {
	var out []wrapLine
	var line wrapLine
	flush := func() {
		out = append(out, line)
		line = wrapLine{}
	}
	i := from
	for i < to {
		if unicode.IsSpace(src[i]) {
			i++
			continue
		}
		// A word: the run of non-blanks from i, and the blank before it.
		gap := i - 1
		for gap > from && unicode.IsSpace(src[gap-1]) {
			gap--
		}
		j := i
		for j < to && !unicode.IsSpace(src[j]) {
			j++
		}
		for j-i > width {
			if len(line.runes) > 0 {
				flush()
			}
			out = append(out, wrapLine{runes: src[i : i+width], src: seq(i, i+width)})
			i += width
		}
		word := wrapLine{runes: src[i:j], src: seq(i, j)}
		switch {
		case len(line.runes) == 0:
			line = word
		case len(line.runes)+1+len(word.runes) <= width:
			line.runes = append(append(line.runes[:len(line.runes):len(line.runes)], ' '), word.runes...)
			line.src = append(append(line.src, gap), word.src...)
		default:
			flush()
			line = word
		}
		i = j
	}
	flush()
	return out
}

func seq(a, b int) []int {
	s := make([]int, b-a)
	for i := range s {
		s[i] = a + i
	}
	return s
}

var (
	flagToken        = regexp.MustCompile(`^--?[A-Za-z][A-Za-z0-9-]*(=.*)?$`)
	placeholderToken = regexp.MustCompile(`^<[^<>'"]*>$`)
)

// colorizeCmd shows a command line in green, with flags in cyan and <placeholders>
// in yellow. It splits on whitespace outside quotes, so quoted text stays green
// and only whole words are colored. Negative numbers are not flags.
func (s style) colorizeCmd(line string) string {
	if !s.on {
		return line
	}
	var out, tok strings.Builder
	var quote rune
	flushTok := func() {
		if tok.Len() == 0 {
			return
		}
		out.WriteString(s.token(tok.String()))
		tok.Reset()
	}
	for _, c := range line {
		switch {
		case quote != 0:
			tok.WriteRune(c)
			if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
			tok.WriteRune(c)
		case c == ' ' || c == '\t':
			flushTok()
			out.WriteRune(c)
		default:
			tok.WriteRune(c)
		}
	}
	flushTok()
	return "\x1b[32m" + out.String() + reset
}

func (s style) token(t string) string {
	switch {
	case flagToken.MatchString(t):
		name, rest := t, ""
		if i := strings.Index(t, "="); i >= 0 {
			name, rest = t[:i], t[i:]
		}
		return "\x1b[36m" + name + "\x1b[32m" + rest
	case placeholderToken.MatchString(t):
		return "\x1b[33m" + t + "\x1b[32m"
	}
	return t
}

// withCLI replaces a leading "just " (or one after whitespace) with the
// command prefix of the project, so examples can be typed as shown.
var justWord = regexp.MustCompile(`(^|\s)just `)

func withCLI(cli, cmd string) string {
	return justWord.ReplaceAllString(cmd, "${1}"+cli+" ")
}

// seeAlso is the cross-reference line of a group, or "".
func (p *printer) seeAlso(proj *Project, g *Group) {
	if len(g.SeeAlso) == 0 {
		return
	}
	refs := append([]string{}, g.SeeAlso...)
	sort.Strings(refs)
	for i := range refs {
		refs[i] = proj.CLI + " " + refs[i]
	}
	p.line("  %s", p.s.dim("See also: "+strings.Join(refs, ", ")))
}

// seeAlsoKeys is the list of related commands a view shows: each key once (the
// first), without a reference to the command itself, sorted by key. The zoom and
// the JSON both read it, so they agree.
func seeAlsoKeys(g *Group, c *Command) []string {
	self := g.Key + "." + c.Name
	seen := map[string]bool{}
	var out []string
	for _, key := range c.SeeAlso {
		if key == self || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// commandSeeAlso is the cross-reference line of a command. A key that names a
// command is written as its invocation, one that does not as it is, so a stale
// reference stays visible. The keys are sorted before they are written, so the
// order is the one of the JSON.
func (p *printer) commandSeeAlso(proj *Project, g *Group, c *Command) {
	keys := seeAlsoKeys(g, c)
	if len(keys) == 0 {
		return
	}
	refs := make([]string, len(keys))
	for i, key := range keys {
		refs[i] = key
		if path, ok := proj.commandPath(key); ok {
			refs[i] = proj.CLI + " " + path
		}
	}
	p.line("  %s", p.s.dim("See also: "+strings.Join(refs, ", ")))
}

// commandPath turns a key (group.command) into the words that run the command:
// the group key and the command, or only the command for core. It reports
// false when no command or extra entry has that key.
func (proj *Project) commandPath(key string) (string, bool) {
	groups := proj.Groups
	if proj.Core != nil {
		groups = append([]*Group{proj.Core}, groups...)
	}
	for _, g := range groups {
		for _, list := range [][]Command{g.Commands, g.Extra} {
			for _, c := range list {
				if g.Key+"."+c.Name == key {
					if g.Key == "core" {
						return c.Name, true
					}
					return g.Key + " " + c.Name, true
				}
			}
		}
	}
	return "", false
}

// GroupView is the detail view of one group: its usage, commands with their
// arguments, examples, footer and cross-references.
func GroupView(proj *Project, g *Group, o Options) string {
	p := newPrinter(o)
	p.b.WriteByte('\n')
	p.line("  %s", p.s.bold(g.Key+" · "+g.Label))
	p.b.WriteByte('\n')

	usage := fmt.Sprintf("%s %s <command>", proj.CLI, g.Key)
	for _, c := range g.Commands {
		if len(c.Args) > 0 {
			usage += " [options]"
			break
		}
	}
	p.line("  %s %s", p.s.bold("Usage:"), usage)
	p.b.WriteByte('\n')
	p.line("  %s", p.s.bold("Commands:"))
	for _, c := range g.Extra {
		p.fullRow(c.Name, c.Desc, c.Example)
		p.argRows(c.Args)
	}
	for _, c := range g.Commands {
		p.fullRow(c.Name, c.Desc, c.Example)
		p.argRows(c.Args)
	}

	if len(g.Examples) > 0 {
		p.b.WriteByte('\n')
		p.line("  %s", p.s.bold("Examples:"))
		width := 0
		cmds := make([]string, len(g.Examples))
		for i, e := range g.Examples {
			cmds[i] = withCLI(proj.CLI, e.Cmd)
			if n := utf8.RuneCountInString(cmds[i]); n > width {
				width = n
			}
		}
		width += 2
		for i, e := range g.Examples {
			if e.Desc == "" {
				p.line("    %s", p.s.colorizeCmd(cmds[i]))
				continue
			}
			gap := strings.Repeat(" ", width-utf8.RuneCountInString(cmds[i]))
			p.line("    %s%s %s %s", p.s.colorizeCmd(cmds[i]), gap, p.s.dim("#"), e.Desc)
		}
	}
	if g.Footer != "" {
		p.b.WriteByte('\n')
		p.line("  %s", p.s.dim(g.Footer))
	}
	p.seeAlso(proj, g)
	p.b.WriteByte('\n')
	return p.b.String()
}

// commandExamples are the examples of the group that mention this command,
// sorted, with "just" replaced by the command prefix. If there are none, the
// command's own example is used.
func commandExamples(proj *Project, g *Group, c *Command, path string) []Example {
	var out []Example
	for _, e := range g.Examples {
		if strings.Contains(e.Cmd, "just "+path+" ") || strings.HasSuffix(e.Cmd, "just "+path) {
			out = append(out, Example{Cmd: withCLI(proj.CLI, e.Cmd), Desc: e.Desc})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Cmd < out[j].Cmd })
	if len(out) == 0 && c.Example != "" {
		one := c.Example
		inv := proj.CLI + " " + path
		if strings.HasPrefix(one, "…") {
			one = strings.ReplaceAll(inv+" "+strings.TrimPrefix(one, "…"), "  ", " ")
		}
		out = append(out, Example{Cmd: one})
	}
	return out
}

// CommandView is the focused view of one command: how to invoke it, what it
// does, examples and options.
func CommandView(proj *Project, g *Group, c *Command, o Options) string {
	p := newPrinter(o)
	w := o.viewWidth()
	path := c.Name
	if g.Key != "core" {
		path = g.Key + " " + c.Name
	}
	inv := proj.CLI + " " + path

	p.b.WriteByte('\n')
	p.line("  %s", p.s.boldCyan(inv))
	p.line("  %s", p.s.dim(strings.Repeat("─", w-4)))
	for _, l := range wrap(w-4, c.Desc) {
		p.line("  %s", l)
	}
	if len(c.Tags) > 0 {
		for i, l := range wrap(w-4, "Tags: "+strings.Join(c.Tags, ", ")) {
			if i == 0 {
				p.line("  %s%s", p.s.dim("Tags:"), strings.TrimPrefix(l, "Tags:"))
			} else {
				p.line("  %s", l)
			}
		}
	}

	if exs := commandExamples(proj, g, c, path); len(exs) > 0 {
		p.b.WriteByte('\n')
		p.line("  %s", p.s.bold("Examples"))
		for _, e := range exs {
			p.b.WriteByte('\n')
			if e.Desc != "" {
				for _, l := range wrap(w-6, e.Desc) {
					p.line("    %s", p.s.dim(l))
				}
			}
			p.line("    %s %s", p.s.dim("$"), p.s.colorizeCmd(e.Cmd))
		}
	}

	if len(c.Args) > 0 {
		p.b.WriteByte('\n')
		p.line("  %s", p.s.bold("Options"))
		col := 0
		for _, a := range c.Args {
			if n := utf8.RuneCountInString(a.Name); n > col {
				col = n
			}
		}
		col += 2
		dw := w - 4 - col
		if dw < 24 {
			dw = 24
		}
		for _, a := range c.Args {
			lines := wrap(dw, argDescription(a))
			p.line("    %s%s", p.s.cyan(pad(a.Name, col)), p.s.dim(lines[0]))
			for _, l := range lines[1:] {
				p.line("    %s%s", strings.Repeat(" ", col), p.s.dim(l))
			}
		}
	}

	if len(c.Exits) > 0 {
		p.b.WriteByte('\n')
		p.line("  %s", p.s.bold("Exit codes"))
		col := 0
		for _, e := range c.Exits {
			if n := len(strconv.Itoa(e.Code)); n > col {
				col = n
			}
		}
		col += 2
		mw := w - 4 - col
		if mw < 24 {
			mw = 24
		}
		for _, e := range c.Exits {
			lines := wrap(mw, e.Meaning)
			p.line("    %s%s", p.s.cyan(pad(strconv.Itoa(e.Code), col)), p.s.dim(lines[0]))
			for _, l := range lines[1:] {
				p.line("    %s%s", strings.Repeat(" ", col), p.s.dim(l))
			}
		}
	}

	if len(c.Env) > 0 {
		p.b.WriteByte('\n')
		p.line("  %s", p.s.bold("Environment"))
		col := 0
		for _, e := range c.Env {
			if n := utf8.RuneCountInString(e.Name); n > col {
				col = n
			}
		}
		col += 2
		dw := w - 4 - col
		if dw < 24 {
			dw = 24
		}
		for _, e := range c.Env {
			lines := wrap(dw, envDescription(e))
			p.line("    %s%s", p.s.cyan(pad(e.Name, col)), p.s.dim(lines[0]))
			for _, l := range lines[1:] {
				p.line("    %s%s", strings.Repeat(" ", col), p.s.dim(l))
			}
		}
	}

	if detail := cleanDetail(c.Detail); detail != "" {
		p.b.WriteByte('\n')
		p.line("  %s", p.s.bold("Details"))
		for _, l := range p.s.highlightLines(w-6, detail) {
			if l == "" {
				p.b.WriteByte('\n')
				continue
			}
			p.line("    %s", l)
		}
	}

	groupLine := g.Key != "core" && len(g.SeeAlso) > 0
	if len(seeAlsoKeys(g, c)) > 0 || groupLine {
		p.b.WriteByte('\n')
	}
	p.commandSeeAlso(proj, g, c)
	if groupLine {
		p.seeAlso(proj, g)
	}
	p.b.WriteByte('\n')
	return p.b.String()
}

// cleanDetail trims the newlines around a command's detail text and turns
// Windows line ends into newlines, so the section never starts or ends with a
// blank line, even when a line at the edge holds only spaces. A text of only
// blanks is no text.
func cleanDetail(s string) string {
	return strings.Trim(strings.ReplaceAll(s, "\r\n", "\n"), " \t\r\n")
}

// Overview is the short list of groups: the next zoom level when nothing is
// selected.
func Overview(proj *Project, o Options) string {
	p := newPrinter(o)
	p.b.WriteByte('\n')
	p.line("  %s  %s", p.s.boldCyan(proj.CLI), p.s.dim("— groups"))
	p.b.WriteByte('\n')
	for _, g := range proj.Groups {
		n := len(g.Commands) + len(g.Extra)
		p.line("  %s %s %s", p.s.green(pad(g.Key, 10)), pad(g.Label, 36), p.s.dim(fmt.Sprintf("%d commands", n)))
	}
	if proj.Core != nil && len(proj.Core.Commands) > 0 {
		names := make([]string, len(proj.Core.Commands))
		for i, c := range proj.Core.Commands {
			names[i] = c.Name
		}
		p.b.WriteByte('\n')
		p.line("  %s %s", p.s.dim("No group:"), strings.Join(names, ", "))
	}
	p.b.WriteByte('\n')
	p.line("  %s %s  %s  %s %s", p.s.dim("Next:"), p.s.green(o.helpCmd()+" <group>"), p.s.green(o.helpCmd()+" [group] <command>"), p.s.green(o.helpCmd()+" --browse"), p.s.dim("(browse)"))
	p.b.WriteByte('\n')
	return p.b.String()
}

// envDescription is the text of an env row: the description, then (required)
// or (default: ...). A required variable has no default worth showing.
func envDescription(e EnvVar) string {
	d := e.Desc
	switch {
	case e.Required:
		d = strings.TrimSpace(d + " (required)")
	case e.Default != "":
		d = strings.TrimSpace(d + " (default: " + e.Default + ")")
	}
	return d
}
