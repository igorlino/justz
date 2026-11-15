package help

import (
	"strings"

	"github.com/igorlino/justz/internal/textsafe"
)

// sanitize makes every string of the model safe to print: the registry or the
// justfile is project text, and a terminal acts on escape sequences in it. The
// two prose fields, a command's detail and a group's footer, keep their
// newlines; everything else is one row of text. It is idempotent, and it runs
// at the end of both loaders, so every view, completion and the JSON read the
// same cleaned text. Project.Doc, the raw registry, is not touched: lookups
// that read it only use names that textsafe.IsName accepts.
func (p *Project) sanitize() {
	p.Name = textsafe.Line(p.Name)
	p.Tagline = textsafe.Line(p.Tagline)
	p.CLI = textsafe.Line(p.CLI)
	if p.Core != nil {
		p.Core.sanitize()
	}
	for _, g := range p.Groups {
		g.sanitize()
	}
	for i := range p.Tiers {
		t := &p.Tiers[i]
		t.Label = textsafe.Line(t.Label)
		t.Hint = textsafe.Line(t.Hint)
		for j := range t.Groups {
			t.Groups[j] = textsafe.Line(t.Groups[j])
		}
	}
}

func (g *Group) sanitize() {
	g.Key = textsafe.Line(g.Key)
	g.Label = textsafe.Line(g.Label)
	g.Title = textsafe.Line(g.Title)
	g.Desc = textsafe.Line(g.Desc)
	g.Footer = textsafe.Text(g.Footer)
	for i := range g.Commands {
		g.Commands[i].sanitize()
	}
	for i := range g.Extra {
		g.Extra[i].sanitize()
	}
	for i := range g.Examples {
		g.Examples[i].Cmd = textsafe.Line(g.Examples[i].Cmd)
		g.Examples[i].Desc = textsafe.Line(g.Examples[i].Desc)
	}
	for i := range g.SeeAlso {
		g.SeeAlso[i] = textsafe.Line(g.SeeAlso[i])
	}
}

func (c *Command) sanitize() {
	c.Name = textsafe.Line(c.Name)
	c.Desc = textsafe.Line(c.Desc)
	c.Example = textsafe.Line(c.Example)
	c.Heading = textsafe.Line(c.Heading)
	for i := range c.Exits {
		c.Exits[i].Meaning = textsafe.Line(c.Exits[i].Meaning)
	}
	for i := range c.SeeAlso {
		c.SeeAlso[i] = textsafe.Line(c.SeeAlso[i])
	}
	c.Detail = textsafe.Text(c.Detail)
	// A detail of only blanks is no detail: the zoom prints no section for it,
	// so the JSON must not carry it either.
	if strings.TrimSpace(c.Detail) == "" {
		c.Detail = ""
	}
	for i := range c.Args {
		a := &c.Args[i]
		a.Name = textsafe.Line(a.Name)
		a.Desc = textsafe.Line(a.Desc)
		for j := range a.Values {
			a.Values[j] = textsafe.Line(a.Values[j])
		}
	}
}
