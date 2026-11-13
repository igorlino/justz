package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/igorlino/justz/internal/help"
	"github.com/igorlino/justz/internal/orderedjson"
	"github.com/igorlino/justz/internal/registry"
)

const helpUsage = `Usage: justz help [flags] [group] [command]

Show the commands of the current project, rendered from its command registry
(commands.json) or the personal overlay (.just-local-rich.json).

  (no words)         A compact summary: core commands, then each group
  --expanded         The full reference: every group with its commands
  <group>            One group: its usage, commands, examples
  [group] <command>  One command: how to invoke it, examples, options
  --wide, -w         Add example columns (on by default from 100 columns)
  --json, -j         Print the same information as JSON for tools and agents

A group or command can be given by an exact name, or by a unique start of it
("to gi" finds "tool git"). A name that exists in several groups is ambiguous
and is listed.

Colors are used only on a terminal, and never when NO_COLOR is set. The width
comes from COLUMNS, then the terminal, then 80.

Exit codes: 0 done, 1 error, 2 usage error.
`

func runHelp(args []string, e env) int {
	expanded, wide, asJSON := false, false, false
	var terms []string
	for _, a := range args {
		switch {
		case a == "--help" || a == "-h":
			fmt.Fprint(e.stdout, helpUsage)
			return 0
		case a == "--expanded":
			expanded = true
		case a == "--wide" || a == "-w":
			wide = true
		case a == "--json" || a == "-j":
			asJSON = true
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(e.stderr, "justz help: unknown flag %q\n\n%s", a, helpUsage)
			return 2
		default:
			terms = append(terms, a)
		}
	}
	if len(terms) > 2 {
		fmt.Fprintf(e.stderr, "justz help: expected [group] <term>, got %d words\n%s", len(terms), helpUsage)
		return 1
	}

	proj, err := loadHelpProject()
	if err != nil {
		fmt.Fprintf(e.stderr, "justz help: %v\n", err)
		return 1
	}
	opts := help.Options{Color: e.color, Width: e.width, Wide: wide}
	if asJSON {
		return zoomJSON(proj, terms, e)
	}
	if len(terms) > 0 {
		return zoom(proj, terms, opts, e)
	}
	if expanded {
		fmt.Fprint(e.stdout, help.Expanded(proj, opts))
	} else {
		fmt.Fprint(e.stdout, help.Summary(proj, opts))
	}
	return 0
}

// loadHelpProject finds the registry (or the local overlay) of the current
// project and builds the help model from it.
func loadHelpProject() (*help.Project, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	path, err := registry.Discover(cwd)
	if err != nil {
		// No committed registry: try the personal overlay in the project root.
		root := projectRoot(cwd)
		overlay := filepath.Join(root, ".just-local-rich.json")
		if !fileExists(overlay) {
			return nil, fmt.Errorf("this project has no commands.json and no .just-local-rich.json (create the overlay with: justz generate)")
		}
		path = overlay
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc, err := orderedjson.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s is not valid JSON: %v", path, err)
	}
	return help.FromRegistry(doc, filepath.Base(projectRootOf(path)))
}

// projectRootOf is the project root for a registry path.
func projectRootOf(path string) string { return projectRoot(filepath.Dir(path)) }

// terminalWidth works out the width for help output: COLUMNS, then the
// terminal, then 80.
func terminalWidth(f *os.File) int {
	if c := os.Getenv("COLUMNS"); c != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(c)); err == nil && n > 0 {
			return n
		}
	}
	if w, _, err := term.GetSize(int(f.Fd())); err == nil && w > 0 {
		return w
	}
	return 80
}

// useColor is true only for a terminal, and never when NO_COLOR is set to a
// non-empty value (https://no-color.org).
func useColor(f *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// zoom shows one group or command, with the exit codes of the lookup: 0 found,
// 2 ambiguous or no match.
func zoom(proj *help.Project, terms []string, opts help.Options, e env) int {
	outcome, rows := proj.Lookup(terms...)
	switch outcome {
	case help.Found:
		r := rows[0]
		g := proj.GroupByKey(r.Group)
		if r.IsGroup {
			fmt.Fprint(e.stdout, help.GroupView(proj, g, opts))
		} else {
			fmt.Fprint(e.stdout, help.CommandView(proj, g, g.FindCommand(r.Command), opts))
		}
		return 0
	case help.Ambiguous:
		fmt.Fprintf(e.stderr, "ambiguous: '%s' matches several entries:\n", strings.Join(terms, " "))
		for _, r := range rows {
			fmt.Fprintf(e.stderr, "%s %s — %s\n", proj.CLI, r.Path(), r.Desc)
		}
		return 2
	}
	fmt.Fprintf(e.stderr, "no group or command matches '%s'. Try: justz help --expanded\n", strings.Join(terms, " "))
	return 2
}

// zoomJSON prints the node of what the words select, or the overview of groups
// when there are none. An ambiguous lookup prints an array with every match
// and exits 2.
func zoomJSON(proj *help.Project, terms []string, e env) int {
	if len(terms) == 0 {
		return printJSON(e, help.OverviewJSON(proj))
	}
	node := func(r help.Row) any {
		g := proj.GroupByKey(r.Group)
		if r.IsGroup {
			return help.GroupJSON(proj, g)
		}
		return help.CommandJSON(proj, g, g.FindCommand(r.Command))
	}
	outcome, rows := proj.Lookup(terms...)
	switch outcome {
	case help.Found:
		return printJSON(e, node(rows[0]))
	case help.Ambiguous:
		nodes := make([]any, 0, len(rows))
		for _, r := range rows {
			nodes = append(nodes, node(r))
		}
		if code := printJSON(e, nodes); code != 0 {
			return code
		}
		return 2
	}
	fmt.Fprintf(e.stderr, "no group or command matches '%s'. Try: justz help --json\n", strings.Join(terms, " "))
	return 2
}
