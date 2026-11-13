package main

import (
	"fmt"
	"os"

	"github.com/igorlino/justz/internal/resolve"
	"github.com/igorlino/justz/internal/shim"
)

const resolveUsage = `Usage: justz resolve [--json] <name>

Print the group path that a bare command name stands for in this project, for
example "git" for "tool git". Only exact names resolve; a prefix never does.

Nothing resolves when the name is a real recipe of the justfile, a group, a
core command (those run without a group), or a documented extra.

Exit codes: 0 exactly one match (printed), 2 several matches (listed), 3 no
match, 1 an error, usage errors are 2.
`

type resolveOutput struct {
	Name    string          `json:"name"`
	Status  string          `json:"status"` // found, ambiguous or none
	Matches []resolve.Match `json:"matches"`
	Reason  string          `json:"reason,omitempty"`
}

func runResolve(args []string, e env) int {
	asJSON := false
	name := ""
	for _, a := range args {
		switch {
		case a == "--help" || a == "-h":
			fmt.Fprint(e.stdout, resolveUsage)
			return 0
		case a == "--json":
			asJSON = true
		case len(a) > 0 && a[0] == '-':
			fmt.Fprintf(e.stderr, "justz resolve: unknown flag %q\n", a)
			return 2
		case name == "":
			name = a
		default:
			fmt.Fprintf(e.stderr, "justz resolve: only one name is allowed, got %q and %q\n", name, a)
			return 2
		}
	}
	if name == "" {
		fmt.Fprintln(e.stderr, "justz resolve: expected a command name")
		return 2
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(e.stderr, "justz resolve: %v\n", err)
		return 1
	}
	justfile, err := shim.FindJustfile(cwd)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz resolve: %v\n", err)
		return 1
	}
	return resolveIn(e, justfile, name, asJSON, "justz resolve")
}

// resolveIn looks the name up in the project of justfile and reports it.
func resolveIn(e env, justfile, name string, asJSON bool, who string) int {
	proj, err := loadProject(e, justfile)
	if err != nil {
		fmt.Fprintf(e.stderr, "%s: %v\n", who, err)
		return 1
	}
	if proj.Registry == nil {
		fmt.Fprintf(e.stderr, "%s: this project has no commands.json, so there are no groups to resolve against\n", who)
		return 3
	}
	res := resolve.Exact(proj.Registry, proj.RootRecipes, name)

	out := resolveOutput{Name: name, Matches: res.Matches, Reason: res.Reason}
	if out.Matches == nil {
		out.Matches = []resolve.Match{}
	}
	code := 3
	switch res.Kind {
	case resolve.Found:
		out.Status, code = "found", 0
	case resolve.Ambiguous:
		out.Status, code = "ambiguous", 2
	default:
		out.Status = "none"
	}

	if asJSON {
		if c := printJSON(e, out); c != 0 {
			return c
		}
		return code
	}
	switch res.Kind {
	case resolve.Found:
		fmt.Fprintf(e.stdout, "%s %s\n", res.Matches[0].Group, res.Matches[0].Command)
	case resolve.Ambiguous:
		fmt.Fprintf(e.stderr, "%s: %q is ambiguous. It could be:\n", who, name)
		for _, m := range res.Matches {
			fmt.Fprintf(e.stderr, "  %s %s\n", m.Group, m.Command)
		}
	default:
		fmt.Fprintf(e.stderr, "%s: %q does not resolve (%s)\n", who, name, reasonText(res.Reason))
	}
	return code
}

func reasonText(r string) string {
	switch r {
	case resolve.ReasonRoot:
		return "it is a recipe of the justfile"
	case resolve.ReasonGroup:
		return "it is a group"
	case resolve.ReasonCore:
		return "it is a core command and already runs without a group"
	case resolve.ReasonExtra:
		return "it is a documented extra"
	}
	return "no group has a command with this exact name"
}
