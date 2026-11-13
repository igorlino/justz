package main

import (
	"fmt"
	"os"

	"github.com/igorlino/justz/internal/resolve"
	"github.com/igorlino/justz/internal/shim"
)

// runAs is what a shim stub starts: justz --as <name> [arguments...]. The shim
// name picks the registered justfile, so the command works from any directory.
//
// A shim runs just on its justfile. When the first argument is a bare command
// name, it is run without its group ("git" for "tool git") if the project's
// registry says exactly one group has it. Anything else goes to just
// unchanged, so the shim never gets in the way.
func runAs(args []string, e env) int {
	if len(args) == 0 {
		fmt.Fprintln(e.stderr, "justz: --as needs a shim name")
		return 2
	}
	name, rest := args[0], args[1:]

	d, err := shim.DefaultDirs()
	if err != nil {
		fmt.Fprintf(e.stderr, "justz: %v\n", err)
		return 1
	}
	reg, err := d.Load()
	if err != nil {
		fmt.Fprintf(e.stderr, "justz: %v\n", err)
		return 1
	}
	entry, isHelp, ok := findShim(reg, name)
	if !ok {
		fmt.Fprintf(e.stderr, "justz: there is no shim named %q (see: justz shim list)\n", name)
		return 2
	}
	if isHelp {
		fmt.Fprintf(e.stderr, "%s: the help command is not available in this version of justz yet\n", name)
		return 1
	}

	if info, err := os.Stat(entry.Justfile); err != nil || info.IsDir() {
		if entry.Soft {
			return e.execJust(append([]string{"--justfile", entry.Justfile}, rest...))
		}
		fmt.Fprintf(e.stderr, "%s: the justfile %s is gone. Check with: justz shim doctor\n", entry.Name, entry.Justfile)
		return 2
	}

	// --which <name> says what a bare name would run, without running it.
	if len(rest) > 0 && rest[0] == "--which" {
		if len(rest) != 2 {
			fmt.Fprintf(e.stderr, "%s: --which needs exactly one name\n", entry.Name)
			return 2
		}
		return resolveIn(e, entry.Justfile, rest[1], false, entry.Name)
	}

	justArgs := append([]string{"--justfile", entry.Justfile}, rest...)

	// Only a plain word in first place can be a bare command name.
	if len(rest) > 0 && rest[0] != "" && rest[0][0] != '-' {
		if proj, err := loadProject(e, entry.Justfile); err == nil && proj.Registry != nil {
			res := resolve.Exact(proj.Registry, proj.RootRecipes, rest[0])
			switch res.Kind {
			case resolve.Found:
				m := res.Matches[0]
				if os.Getenv("JUSTZ_QUIET") != "1" {
					fmt.Fprintf(e.stderr, "%s: %s %s\n", entry.Name, m.Group, m.Command)
				}
				justArgs = append([]string{"--justfile", entry.Justfile, m.Group, m.Command}, rest[1:]...)
			case resolve.Ambiguous:
				fmt.Fprintf(e.stderr, "%s: %q is ambiguous. It could be:\n", entry.Name, rest[0])
				for _, m := range res.Matches {
					fmt.Fprintf(e.stderr, "  %s %s %s\n", entry.Name, m.Group, m.Command)
				}
				fmt.Fprintf(e.stderr, "Use the full path, or check with: %s --which %s\n", entry.Name, rest[0])
				return 2
			}
		}
	}
	return e.execJust(justArgs)
}

// findShim matches a name against shim names and help names.
func findShim(reg shim.Registry, name string) (entry shim.Entry, isHelp, ok bool) {
	for _, s := range reg.Shims {
		if s.Name == name {
			return s, false, true
		}
		if s.HelpName() == name {
			return s, true, true
		}
	}
	return shim.Entry{}, false, false
}
