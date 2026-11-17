package help

import (
	"github.com/igorlino/justz/internal/generate"
)

// FromDump builds the help model of a project that has no registry, straight
// from the output of just --dump --dump-format json. Root recipes are the core
// commands and each top-level module is a group, exactly as in the overlay
// that justz generate writes. A root recipe's [group] attribute becomes a
// display heading in the summary; it never changes how the recipe is run.
// name is shown in the header, typically the project directory.
func FromDump(dumpJSON []byte, name string) (*Project, error) {
	res, err := generate.Build(dumpJSON, nil)
	if err != nil {
		return nil, err
	}
	p, err := FromRegistry(res.Doc, name)
	if err != nil {
		return nil, err
	}
	p.Name = name
	p.CLI = "just"

	parsed, err := generate.Parse(dumpJSON)
	if err != nil {
		return nil, err
	}
	headings := map[string]string{}
	params := map[string][]generate.Arg{} // by "command", or "group command"
	for _, c := range parsed.Core {
		headings[c.Name] = c.Heading
		params[c.Name] = c.AllArgs
	}
	for _, m := range parsed.Modules {
		for _, c := range m.Commands {
			params[m.Name+" "+c.Name] = c.AllArgs
		}
	}
	if p.Core != nil {
		for i := range p.Core.Commands {
			c := &p.Core.Commands[i]
			c.Heading = headings[c.Name]
			c.Args = withUndocumented(c.Args, params[c.Name])
		}
	}
	for _, g := range p.Groups {
		for i := range g.Commands {
			c := &g.Commands[i]
			c.Args = withUndocumented(c.Args, params[g.Key+" "+c.Name])
		}
	}
	p.sanitize()
	return p, nil
}

// withUndocumented lists every parameter of a recipe, in the order of the
// recipe: the documented ones as they are, the others with an empty
// description. A justfile is the only source here, so a parameter left out
// would look like a command that takes nothing, and one out of place would
// swap two positional arguments.
func withUndocumented(args []Arg, all []generate.Arg) []Arg {
	documented := map[string]Arg{}
	for _, a := range args {
		documented[a.Name] = a
	}
	out := make([]Arg, 0, len(all))
	for _, a := range all {
		if d, ok := documented[a.Name]; ok {
			out = append(out, d)
		} else {
			out = append(out, Arg{Name: a.Name})
		}
	}
	return out
}
