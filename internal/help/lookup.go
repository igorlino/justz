package help

import "strings"

// Row is one thing a lookup can find: a group, or a command of a group
// (including the core group and documented extras).
type Row struct {
	IsGroup bool
	Group   string
	Command string // empty for a group
	Desc    string
}

// Path is how the row is typed after the command prefix: a core command has no
// group word.
func (r Row) Path() string {
	switch {
	case r.IsGroup:
		return r.Group
	case r.Group == "core":
		return r.Command
	}
	return r.Group + " " + r.Command
}

// Outcome of a lookup.
type Outcome int

const (
	Found     Outcome = iota // exactly one row
	Ambiguous                // several rows
	NotFound                 // nothing
)

// rows lists every group and command of the project.
func (p *Project) rows() []Row {
	var out []Row
	for _, g := range p.Groups {
		out = append(out, Row{IsGroup: true, Group: g.Key, Desc: g.Label})
	}
	add := func(g *Group) {
		for _, c := range append(append([]Command{}, g.Extra...), g.Commands...) {
			out = append(out, Row{Group: g.Key, Command: c.Name, Desc: c.Desc})
		}
	}
	if p.Core != nil {
		add(p.Core)
	}
	for _, g := range p.Groups {
		add(g)
	}
	return out
}

// Lookup finds what one or two words stand for, with the rules of the
// existing help tests.
//
// One word: an exact group wins. Otherwise an exact command name, in any
// group. Otherwise anything that starts with the word.
//
// Two words: the first is a group, exactly or as a unique prefix ("core" is
// always a group); the second is a command of that group, exactly or as a
// prefix.
func (p *Project) Lookup(terms ...string) (Outcome, []Row) {
	all := p.rows()
	var found []Row

	switch len(terms) {
	case 1:
		t := terms[0]
		for _, r := range all {
			if r.IsGroup && r.Group == t {
				return Found, []Row{r}
			}
		}
		for _, r := range all {
			if !r.IsGroup && r.Command == t {
				found = append(found, r)
			}
		}
		if len(found) == 0 {
			for _, r := range all {
				name := r.Command
				if r.IsGroup {
					name = r.Group
				}
				if strings.HasPrefix(name, t) {
					found = append(found, r)
				}
			}
		}
	case 2:
		group, ok := p.selectGroup(terms[0])
		if !ok {
			return NotFound, nil
		}
		for _, r := range all {
			if !r.IsGroup && r.Group == group && r.Command == terms[1] {
				found = append(found, r)
			}
		}
		if len(found) == 0 {
			for _, r := range all {
				if !r.IsGroup && r.Group == group && strings.HasPrefix(r.Command, terms[1]) {
					found = append(found, r)
				}
			}
		}
	}

	switch len(found) {
	case 0:
		return NotFound, nil
	case 1:
		return Found, found
	}
	return Ambiguous, found
}

// selectGroup resolves the first of two words to a group key.
func (p *Project) selectGroup(t string) (string, bool) {
	if t == "core" && p.Core != nil {
		return "core", true
	}
	for _, g := range p.Groups {
		if g.Key == t {
			return t, true
		}
	}
	match := ""
	n := 0
	for _, g := range p.Groups {
		if strings.HasPrefix(g.Key, t) {
			match, n = g.Key, n+1
		}
	}
	return match, n == 1
}

// FindCommand returns a group's command or extra by name.
func (g *Group) FindCommand(name string) *Command {
	for i := range g.Commands {
		if g.Commands[i].Name == name {
			return &g.Commands[i]
		}
	}
	for i := range g.Extra {
		if g.Extra[i].Name == name {
			return &g.Extra[i]
		}
	}
	return nil
}

// GroupByKey returns the group with this key, including the core group.
func (p *Project) GroupByKey(key string) *Group {
	if key == "core" {
		return p.Core
	}
	return p.Group(key)
}
