package help

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Options control how a view is rendered.
type Options struct {
	Color bool // use ANSI colors; callers turn it off for pipes and NO_COLOR
	Width int  // terminal width in columns; 0 means 80
	Wide  bool // show example columns; implied by a width of 100 or more
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

// compactRow is a row of the summary: a name, a hash and a description.
func (p *printer) compactRow(name, desc string) {
	p.line("    %s %s %s", p.s.green(pad(name, compactName)), p.s.dim("#"), desc)
}

// fullRow is a row of the expanded view. With the wide layout and an example,
// the example goes in a third column.
func (p *printer) fullRow(name, desc, example string) {
	if p.o.wide() && example != "" {
		p.line("    %s %s %s %s", p.s.green(pad(name, fullName)), p.s.dim("#"), pad(desc, exampleDesc), p.s.dim(example))
		return
	}
	p.line("    %s %s %s", p.s.green(pad(name, fullName)), p.s.dim("#"), desc)
}

// argRows prints the arguments of a command under its row.
func (p *printer) argRows(args []Arg) {
	for _, a := range args {
		desc := argDescription(a)
		if desc == "" {
			p.line("      %s", p.s.cyan(a.Name))
			continue
		}
		p.line("      %s %s", p.s.cyan(pad(a.Name, fullName)), p.s.dim(desc))
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

// Summary is the compact overview: the core commands, then each tier of
// groups with a one-line description.
func Summary(proj *Project, o Options) string {
	p := newPrinter(o)
	p.header(proj)
	p.b.WriteByte('\n')

	if proj.Core != nil {
		p.line("  %s", p.s.bold("Core"))
		for _, c := range proj.Core.Commands {
			p.compactRow(c.Name, c.Desc)
		}
		p.b.WriteByte('\n')
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
	p.line("  %s", p.s.dim("Run ")+p.s.green("justz help --expanded")+p.s.dim(" for the full command reference."))
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
		p.line("  %s", p.s.bold("Core"))
		for _, c := range proj.Core.Commands {
			p.fullRow(c.Name, c.Desc, c.Example)
			p.argRows(c.Args)
		}
		p.b.WriteByte('\n')
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
		p.line("  %s", p.s.dim(fmt.Sprintf("Tip: run `justz help %s` for argument details", first)))
	}
	p.b.WriteByte('\n')
	return p.b.String()
}
