// Package help builds and renders the help views of a project from its
// command registry.
package help

import (
	"fmt"
	"sort"

	"github.com/igorlino/justz/internal/orderedjson"
	"github.com/igorlino/justz/internal/registry"
)

// Arg is one option or positional argument of a command.
type Arg struct {
	Name   string
	Desc   string
	Values []string // the allowed values of an enum, if any
}

// Command is one runnable command.
type Command struct {
	Name    string
	Desc    string
	Example string
	Args    []Arg
}

// Example is a worked example of a group.
type Example struct {
	Cmd  string
	Desc string
}

// Group is a namespace of commands. The group "core" holds the commands that
// run without a group name.
type Group struct {
	Key      string
	Label    string
	Title    string
	Desc     string
	Footer   string
	Commands []Command
	Extra    []Command // documented entries that are not runnable by bare name
	Examples []Example
	SeeAlso  []string
}

// Tier is a headed section of groups in the summary.
type Tier struct {
	Label  string
	Hint   string
	Groups []string
}

// Project is everything the help views need.
type Project struct {
	Name    string
	Tagline string
	CLI     string // the command prefix shown in usage lines
	Core    *Group // nil when the registry has no core group
	Groups  []*Group
	Tiers   []Tier
}

// Group returns the non-core group with this key, or nil.
func (p *Project) Group(key string) *Group {
	for _, g := range p.Groups {
		if g.Key == key {
			return g
		}
	}
	return nil
}

// FromRegistry reads a project from a parsed registry. It puts the registry in
// canonical order first, so the views never depend on how the file was
// written. fallbackName is used when the registry has no name.
func FromRegistry(doc *orderedjson.Value, fallbackName string) (*Project, error) {
	if doc == nil || doc.Kind != orderedjson.Object {
		return nil, fmt.Errorf("the registry is not a JSON object")
	}
	registry.Canonicalize(doc)

	p := &Project{
		Name:    firstString(doc, fallbackName, "name", "title"),
		Tagline: firstString(doc, "", "tagline", "subtitle"),
		CLI:     firstString(doc, "just", "cli"),
	}

	groups := doc.Get("groups")
	if groups == nil || groups.Kind != orderedjson.Object {
		return nil, fmt.Errorf(`the registry has no "groups" object`)
	}
	for _, m := range groups.Obj {
		if m.Val == nil || m.Val.Kind != orderedjson.Object {
			continue
		}
		g := readGroup(m.Key, m.Val)
		if m.Key == "core" {
			p.Core = g
		} else {
			p.Groups = append(p.Groups, g)
		}
	}
	p.Tiers = readTiers(doc, p)
	return p, nil
}

func readGroup(key string, v *orderedjson.Value) *Group {
	g := &Group{
		Key:    key,
		Label:  firstString(v, key, "label"),
		Title:  firstString(v, "", "title"),
		Desc:   firstString(v, "", "desc"),
		Footer: firstString(v, "", "footer"),
	}
	g.Commands = readCommands(v.Get("commands"))
	g.Extra = readCommands(v.Get("extra"))
	if ex := v.Get("examples"); ex != nil && ex.Kind == orderedjson.Array {
		for _, e := range ex.Arr {
			g.Examples = append(g.Examples, Example{Cmd: firstString(e, "", "cmd"), Desc: firstString(e, "", "desc")})
		}
	}
	if sa := v.Get("seeAlso"); sa != nil && sa.Kind == orderedjson.Array {
		for _, s := range sa.Arr {
			if s.Kind == orderedjson.String {
				g.SeeAlso = append(g.SeeAlso, s.Str)
			}
		}
	}
	return g
}

func readCommands(v *orderedjson.Value) []Command {
	if v == nil || v.Kind != orderedjson.Object {
		return nil
	}
	var out []Command
	for _, m := range v.Obj {
		c := Command{Name: m.Key, Desc: firstString(m.Val, "", "desc"), Example: firstString(m.Val, "", "example")}
		if args := m.Val.Get("args"); args != nil && args.Kind == orderedjson.Array {
			for _, a := range args.Arr {
				arg := Arg{Name: firstString(a, "", "name"), Desc: firstString(a, "", "desc")}
				if vals := a.Get("values"); vals != nil && vals.Kind == orderedjson.Array {
					for _, s := range vals.Arr {
						if s.Kind == orderedjson.String {
							arg.Values = append(arg.Values, s.Str)
						}
					}
				}
				c.Args = append(c.Args, arg)
			}
		}
		out = append(out, c)
	}
	return out
}

// readTiers reads the tiers. Without any, every non-core group goes in one
// default tier, so a registry that does not group its groups still renders.
func readTiers(doc *orderedjson.Value, p *Project) []Tier {
	var tiers []Tier
	if ts := doc.Get("tiers"); ts != nil && ts.Kind == orderedjson.Array {
		for _, t := range ts.Arr {
			tier := Tier{Label: firstString(t, "", "label"), Hint: firstString(t, "", "hint")}
			if gs := t.Get("groups"); gs != nil && gs.Kind == orderedjson.Array {
				for _, g := range gs.Arr {
					if g.Kind == orderedjson.String && p.Group(g.Str) != nil {
						tier.Groups = append(tier.Groups, g.Str)
					}
				}
			}
			tiers = append(tiers, tier)
		}
	}
	if len(tiers) > 0 {
		return tiers
	}
	keys := make([]string, 0, len(p.Groups))
	for _, g := range p.Groups {
		keys = append(keys, g.Key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return nil
	}
	return []Tier{{Label: "Commands", Hint: "run without args to see sub-commands", Groups: keys}}
}

// firstString returns the first of the named string fields that is set.
func firstString(v *orderedjson.Value, fallback string, names ...string) string {
	for _, n := range names {
		if s := v.Get(n); s != nil && s.Kind == orderedjson.String && s.Str != "" {
			return s.Str
		}
	}
	return fallback
}
