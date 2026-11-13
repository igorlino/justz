package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	return execShim(entry.Name, entry.Justfile, entry.Soft, rest, e)
}

// execShim runs just for a shim named label on justfile. rest are the
// arguments the user typed.
func execShim(label, justfile string, soft bool, rest []string, e env) int {
	if info, err := os.Stat(justfile); err != nil || info.IsDir() {
		if soft {
			return e.execJust(append([]string{"--justfile", justfile}, rest...))
		}
		fmt.Fprintf(e.stderr, "%s: the justfile %s is gone. Check with: justz shim doctor\n", label, justfile)
		return 2
	}

	// --which <name> says what a bare name would run, without running it.
	if len(rest) > 0 && rest[0] == "--which" {
		if len(rest) != 2 {
			fmt.Fprintf(e.stderr, "%s: --which needs exactly one name\n", label)
			return 2
		}
		return resolveIn(e, justfile, rest[1], false, label)
	}

	justArgs := append([]string{"--justfile", justfile}, rest...)

	// Only a plain word in first place can be a bare command name.
	if len(rest) > 0 && rest[0] != "" && rest[0][0] != '-' {
		if proj, err := loadProject(e, justfile); err == nil && proj.Registry != nil {
			res := resolve.Exact(proj.Registry, proj.RootRecipes, rest[0])
			switch res.Kind {
			case resolve.Found:
				m := res.Matches[0]
				if os.Getenv("JUSTZ_QUIET") != "1" {
					fmt.Fprintf(e.stderr, "%s: %s %s\n", label, m.Group, m.Command)
				}
				justArgs = append([]string{"--justfile", justfile, m.Group, m.Command}, rest[1:]...)
			case resolve.Ambiguous:
				fmt.Fprintf(e.stderr, "%s: %q is ambiguous. It could be:\n", label, rest[0])
				for _, m := range res.Matches {
					fmt.Fprintf(e.stderr, "  %s %s %s\n", label, m.Group, m.Command)
				}
				fmt.Fprintf(e.stderr, "Use the full path, or check with: %s --which %s\n", label, rest[0])
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

// invokedAs returns "jz" or "jzh" when the justz binary was started under one
// of those names (a link to it), and "" otherwise.
func invokedAs(argv0 string) string {
	base := strings.ToLower(filepath.Base(argv0))
	base = strings.TrimSuffix(base, ".exe")
	if base == "jz" || base == "jzh" {
		return base
	}
	return ""
}

// runLocal is the built-in jz shortcut: it behaves like a shim for the justfile
// of the current project, found by walking up from the working directory.
func runLocal(name string, rest []string, e env) int {
	if name == "jzh" {
		fmt.Fprintf(e.stderr, "%s: the help command is not available in this version of justz yet\n", name)
		return 1
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(e.stderr, "%s: %v\n", name, err)
		return 1
	}
	justfile, err := shim.FindJustfile(cwd)
	if err != nil {
		fmt.Fprintf(e.stderr, "%s: %v\n", name, err)
		return 2
	}
	return execShim(name, justfile, false, rest, e)
}
