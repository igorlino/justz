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
	for _, c := range parsed.Core {
		headings[c.Name] = c.Heading
	}
	if p.Core != nil {
		for i := range p.Core.Commands {
			p.Core.Commands[i].Heading = headings[p.Core.Commands[i].Name]
		}
	}
	return p, nil
}
