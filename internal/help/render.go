package help

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// Options control how a view is rendered.
type Options struct {
	Color bool // use ANSI colors; callers turn it off for pipes and NO_COLOR
	Width int  // terminal width in columns; 0 means 80
	Wide  bool // show example columns; implied by a width of 100 or more
	// Wrap breaks a long description at the width, with the lines after the first
	// aligned under it. Callers turn it on for a terminal; piped output keeps one
	// line per command so grep and diff see whole rows.
	Wrap bool
	// HelpCmd is how the help command is typed, for hints in the views:
	// "justz help" by default, "jzh" or "dmh" when run through a shim.
	HelpCmd string
}

func (o Options) helpCmd() string {
	if o.HelpCmd == "" {
		return "justz help"
	}
	return o.HelpCmd
}

// WideAt is the width from which the example column is shown.
const WideAt = 100

func (o Options) width() int {
	if o.Width <= 0 {
		return 80
	}
	return o.Width
}

func (o Options) wide() bool { return o.Wide || o.width() >= WideAt }

// style wraps text in an ANSI color when colors are on.
type style struct{ on bool }

const reset = "\x1b[0m"

func (s style) wrap(code, text string) string {
	if !s.on || text == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + reset
}

func (s style) bold(t string) string     { return s.wrap("1", t) }
func (s style) boldCyan(t string) string { return s.wrap("1;36", t) }
func (s style) dim(t string) string      { return s.wrap("2", t) }
func (s style) green(t string) string    { return s.wrap("32", t) }
func (s style) cyan(t string) string     { return s.wrap("36", t) }

// pad returns text padded with spaces to n display columns.
func pad(text string, n int) string {
	if w := utf8.RuneCountInString(text); w < n {
		return text + strings.Repeat(" ", n-w)
	}
	return text
}

// Column widths of the views.
const (
	compactName = 19 // summary rows
	fullName    = 28 // expanded rows, and argument names under them
	exampleDesc = 40 // description column in the wide layout
)

type printer struct {
	b strings.Builder
	s style
	o Options
}

func newPrinter(o Options) *printer { return &printer{s: style{on: o.Color}, o: o} }

func (p *printer) line(format string, a ...any) {
	fmt.Fprintf(&p.b, format, a...)
	p.b.WriteByte('\n')
}

func (p *printer) header(proj *Project) {
	p.b.WriteByte('\n')
	title := p.s.boldCyan(proj.Name)
	if proj.Tagline != "" {
		title += "  " + p.s.dim("— "+proj.Tagline)
	}
	p.line("  %s", title)
}

// minDescWidth is the narrowest a wrapped description gets. On a terminal
// narrower than the name column plus this, lines run past the edge instead of
// breaking after every word.
const minDescWidth = 20

// descLines splits a description for a row whose text starts at column indent.
// Without Wrap it is one line.
func (p *printer) descLines(indent int, desc string) []string {
	if !p.o.Wrap {
		return []string{desc}
	}
	w := p.o.width() - indent
	if w < minDescWidth {
		w = minDescWidth
	}
	return wrap(w, desc)
}

// compactRow is a row of the summary: a name, a hash and a description.
func (p *printer) compactRow(name, desc string) {
	const indent = 4 + compactName + 3
	lines := p.descLines(indent, desc)
	p.line("    %s %s %s", p.s.green(pad(name, compactName)), p.s.dim("#"), lines[0])
	for _, l := range lines[1:] {
		p.line("%s%s", strings.Repeat(" ", indent), l)
	}
}

// fullRow is a row of the expanded view. With the wide layout and an example,
// the example goes in a third column.
func (p *printer) fullRow(name, desc, example string) {
	if p.o.wide() && example != "" {
		p.line("    %s %s %s %s", p.s.green(pad(name, fullName)), p.s.dim("#"), pad(desc, exampleDesc), p.s.dim(example))
		return
	}
	const indent = 4 + fullName + 3
	lines := p.descLines(indent, desc)
	p.line("    %s %s %s", p.s.green(pad(name, fullName)), p.s.dim("#"), lines[0])
	for _, l := range lines[1:] {
		p.line("%s%s", strings.Repeat(" ", indent), l)
	}
}

// argRows prints the arguments of a command under its row.
func (p *printer) argRows(args []Arg) {
	for _, a := range args {
		desc := argDescription(a)
		if desc == "" {
			p.line("      %s", p.s.cyan(a.Name))
			continue
		}
		const indent = 6 + fullName + 1
		lines := p.descLines(indent, desc)
		p.line("      %s %s", p.s.cyan(pad(a.Name, fullName)), p.s.dim(lines[0]))
		for _, l := range lines[1:] {
			p.line("%s%s", strings.Repeat(" ", indent), p.s.dim(l))
		}
	}
}

// argDescription appends the allowed values of an enum as (a|b|c) unless the
// description already lists them.
func argDescription(a Arg) string {
	if len(a.Values) == 0 {
		return a.Desc
	}
	v := strings.Join(a.Values, "|")
	switch {
	case strings.Contains(a.Desc, v):
		return a.Desc
	case a.Desc == "":
		return "(" + v + ")"
	}
	return a.Desc + " (" + v + ")"
}

// section is a headed list of core commands.
type section struct {
	Title string
	Cmds  []Command
}

// coreSections splits the core commands by their [group] heading. Commands
// without one stay under "Core", which comes first; the others follow, sorted
// by heading. Without any heading there is one "Core" section.
func coreSections(g *Group) []section {
	byHeading := map[string][]Command{}
	var plain []Command
	for _, c := range g.Commands {
		if c.Heading == "" {
			plain = append(plain, c)
		} else {
			byHeading[c.Heading] = append(byHeading[c.Heading], c)
		}
	}
	if len(byHeading) == 0 {
		return []section{{Title: "Core", Cmds: g.Commands}}
	}
	var out []section
	if len(plain) > 0 {
		out = append(out, section{Title: "Core", Cmds: plain})
	}
	keys := make([]string, 0, len(byHeading))
	for k := range byHeading {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, section{Title: k, Cmds: byHeading[k]})
	}
	return out
}

// Summary is the compact overview: the core commands, then each tier of
// groups with a one-line description.
func Summary(proj *Project, o Options) string {
	p := newPrinter(o)
	p.header(proj)
	p.b.WriteByte('\n')

	if proj.Core != nil {
		for _, sec := range coreSections(proj.Core) {
			p.line("  %s", p.s.bold(sec.Title))
			for _, c := range sec.Cmds {
				p.compactRow(c.Name, c.Desc)
			}
			p.b.WriteByte('\n')
		}
	}
	for _, t := range proj.Tiers {
		if t.Hint != "" {
			p.line("  %s  %s", p.s.bold(t.Label), p.s.dim("("+t.Hint+")"))
		} else {
			p.line("  %s", p.s.bold(t.Label))
		}
		for _, key := range t.Groups {
			if g := proj.Group(key); g != nil {
				p.compactRow(g.Key, g.Label)
			}
		}
		p.b.WriteByte('\n')
	}
	p.line("  %s", p.s.dim("Run ")+p.s.green(o.helpCmd()+" --expanded")+p.s.dim(" for the full command reference."))
	p.b.WriteByte('\n')
	return p.b.String()
}

// Expanded is the full reference: every group with every command.
func Expanded(proj *Project, o Options) string {
	p := newPrinter(o)
	p.header(proj)
	p.line("  %s", p.s.dim("Full command reference — all groups expanded"))
	p.b.WriteByte('\n')

	if proj.Core != nil {
		for _, sec := range coreSections(proj.Core) {
			p.line("  %s", p.s.bold(sec.Title))
			for _, c := range sec.Cmds {
				p.fullRow(c.Name, c.Desc, c.Example)
				p.argRows(c.Args)
			}
			p.b.WriteByte('\n')
		}
	}

	first := ""
	for _, t := range proj.Tiers {
		for _, key := range t.Groups {
			g := proj.Group(key)
			if g == nil {
				continue
			}
			if first == "" {
				first = key
			}
			p.line("  %s  %s", p.s.bold(g.Key), p.s.dim("— "+g.Label))
			for _, c := range g.Extra {
				p.fullRow(c.Name, c.Desc, c.Example)
			}
			for _, c := range g.Commands {
				p.fullRow(g.Key+" "+c.Name, c.Desc, c.Example)
			}
			p.b.WriteByte('\n')
		}
	}
	if first != "" {
		p.line("  %s", p.s.dim(fmt.Sprintf("Tip: run `%s %s` for argument details", o.helpCmd(), first)))
	}
	p.b.WriteByte('\n')
	return p.b.String()
}
