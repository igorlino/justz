package help

// The JSON nodes of the help views, for tools and agents. The shapes are part
// of the contract of justz: add fields, do not rename or remove them.

// OverviewNode is the list of groups.
type OverviewNode struct {
	CLI    string         `json:"cli"`
	Groups []OverviewItem `json:"groups"`
	Core   []string       `json:"core"`
}

// OverviewItem is one group of the overview.
type OverviewItem struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Commands int    `json:"commands"`
}

// CommandSummary is a command listed inside a group.
type CommandSummary struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// ExampleNode is an example with its command ready to type.
type ExampleNode struct {
	Cmd  string `json:"cmd"`
	Desc string `json:"desc"`
}

// GroupNode describes one group.
type GroupNode struct {
	Path     string           `json:"path"`
	Invoke   string           `json:"invoke"`
	Label    string           `json:"label"`
	Desc     string           `json:"desc"`
	Commands []CommandSummary `json:"commands"`
	Examples []ExampleNode    `json:"examples"`
	SeeAlso  []string         `json:"seeAlso"`
}

// ArgNode is an argument of a command. Desc is the plain description; the
// allowed values of an enum are in Values.
type ArgNode struct {
	Name   string   `json:"name"`
	Desc   string   `json:"desc"`
	Values []string `json:"values,omitempty"`
}

// CommandNode describes one command.
type CommandNode struct {
	Path     string        `json:"path"`
	Invoke   string        `json:"invoke"`
	Desc     string        `json:"desc"`
	Args     []ArgNode     `json:"args"`
	Examples []ExampleNode `json:"examples"`
}

// OverviewJSON is the node for a project with nothing selected.
func OverviewJSON(proj *Project) OverviewNode {
	n := OverviewNode{CLI: proj.CLI, Groups: []OverviewItem{}, Core: []string{}}
	for _, g := range proj.Groups {
		n.Groups = append(n.Groups, OverviewItem{Name: g.Key, Label: g.Label, Commands: len(g.Commands) + len(g.Extra)})
	}
	if proj.Core != nil {
		for _, c := range proj.Core.Commands {
			n.Core = append(n.Core, c.Name)
		}
	}
	return n
}

// GroupJSON is the node of one group. Its commands are the runnable ones and
// the documented extras, sorted by name.
func GroupJSON(proj *Project, g *Group) GroupNode {
	n := GroupNode{
		Path: g.Key, Invoke: proj.CLI + " " + g.Key, Label: g.Label, Desc: g.Desc,
		Commands: []CommandSummary{}, Examples: []ExampleNode{}, SeeAlso: []string{},
	}
	for _, c := range append(append([]Command{}, g.Commands...), g.Extra...) {
		n.Commands = append(n.Commands, CommandSummary{Name: c.Name, Desc: c.Desc})
	}
	sortSummaries(n.Commands)
	for _, e := range g.Examples {
		n.Examples = append(n.Examples, ExampleNode{Cmd: withCLI(proj.CLI, e.Cmd), Desc: e.Desc})
	}
	n.SeeAlso = append(n.SeeAlso, g.SeeAlso...)
	return n
}

// CommandJSON is the node of one command.
func CommandJSON(proj *Project, g *Group, c *Command) CommandNode {
	path := c.Name
	if g.Key != "core" {
		path = g.Key + " " + c.Name
	}
	n := CommandNode{
		Path: path, Invoke: proj.CLI + " " + path, Desc: c.Desc,
		Args: []ArgNode{}, Examples: []ExampleNode{},
	}
	for _, a := range c.Args {
		n.Args = append(n.Args, ArgNode{Name: a.Name, Desc: a.Desc, Values: a.Values})
	}
	for _, e := range commandExamples(proj, g, c, path) {
		n.Examples = append(n.Examples, ExampleNode{Cmd: e.Cmd, Desc: e.Desc})
	}
	return n
}

func sortSummaries(s []CommandSummary) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j].Name < s[j-1].Name; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
