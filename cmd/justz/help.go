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

const helpUsage = `Usage: justz help [flags]

Show the commands of the current project, rendered from its command registry
(commands.json) or the personal overlay (.just-local-rich.json).

  (no flags)    A compact summary: core commands, then each group
  --expanded    The full reference: every group with its commands
  --wide, -w    Add example columns (on by default from 100 columns)

Colors are used only on a terminal, and never when NO_COLOR is set. The width
comes from COLUMNS, then the terminal, then 80.

Exit codes: 0 done, 1 error, 2 usage error.
`

func runHelp(args []string, e env) int {
	expanded, wide := false, false
	for _, a := range args {
		switch a {
		case "--help", "-h":
			fmt.Fprint(e.stdout, helpUsage)
			return 0
		case "--expanded":
			expanded = true
		case "--wide", "-w":
			wide = true
		default:
			fmt.Fprintf(e.stderr, "justz help: unknown argument %q\n\n%s", a, helpUsage)
			return 2
		}
	}

	proj, err := loadHelpProject()
	if err != nil {
		fmt.Fprintf(e.stderr, "justz help: %v\n", err)
		return 1
	}
	opts := help.Options{Color: e.color, Width: e.width, Wide: wide}
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
