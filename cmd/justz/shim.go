package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/igorlino/justz/internal/registry"
	"github.com/igorlino/justz/internal/shim"
)

const shimUsage = `Usage: justz shim <command>

Commands:
  add [path]      Register a shim for the project at path (default: .)
  list            List registered shims
  remove <name>   Remove a shim and its stubs
  doctor          Check every registered shim

Run "justz shim <command> --help" for details.
`

const shimAddUsage = `Usage: justz shim add [path] [flags]

Register a shim: a command that runs justz for a project's justfile, from any
directory. path is a project directory or a justfile (default: the current
directory). The shim name comes from the "cli" field of the project's
commands.json, unless --name is given. A help command called <name><suffix> is
added too.

Flags:
  --name NAME     Use this shim name
  --suffix S      Help command suffix (default: h)
  --soft          Fall back to plain just when the help system is missing
  --replace       Re-point a shim that another project owns, or hide a command
                  that comes later on the PATH
  --force         Allow a shim that an earlier command on the PATH would shadow
  --dry-run       Show what would happen, write nothing
  --json          Machine-readable output

justz never overwrites a file in the shim directory that it did not write.

Exit codes: 0 done, 1 refused or failed, 2 usage error.
`

func runShim(args []string, e env) int {
	if len(args) == 0 {
		fmt.Fprint(e.stderr, shimUsage)
		return 2
	}
	switch args[0] {
	case "--help", "-h", "help":
		fmt.Fprint(e.stdout, shimUsage)
		return 0
	case "list":
		return runShimList(args[1:], e)
	case "add":
		return runShimAdd(args[1:], e)
	case "remove":
		return runShimRemove(args[1:], e)
	case "doctor":
		return runShimDoctor(args[1:], e)
	}
	fmt.Fprintf(e.stderr, "justz shim: unknown command %q\n\n%s", args[0], shimUsage)
	return 2
}

func printJSON(e env, v any) int {
	enc := json.NewEncoder(e.stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(e.stderr, "justz: %v\n", err)
		return 1
	}
	return 0
}

func runShimList(args []string, e env) int {
	asJSON, err := parseJSONFlag(args)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz shim list: %v\n", err)
		return 2
	}
	d, err := shim.DefaultDirs()
	if err != nil {
		fmt.Fprintf(e.stderr, "justz shim list: %v\n", err)
		return 1
	}
	reg, err := d.Load()
	if err != nil {
		fmt.Fprintf(e.stderr, "justz shim list: %v\n", err)
		return 1
	}
	if asJSON {
		return printJSON(e, reg)
	}
	if len(reg.Shims) == 0 {
		fmt.Fprintln(e.stdout, "No shims registered.")
		return 0
	}
	for _, s := range reg.Shims {
		soft := ""
		if s.Soft {
			soft = "  (soft)"
		}
		fmt.Fprintf(e.stdout, "%s\t%s%s\n", s.Name, s.Justfile, soft)
	}
	return 0
}

func runShimAdd(args []string, e env) int {
	var (
		path, name string
		suffix     = "h"
		soft       bool
		asJSON     bool
		dryRun     bool
		over       shim.Overrides
		haveName   bool
		havePath   bool
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		value := func() (string, bool) {
			if k := strings.Index(a, "="); k >= 0 && strings.HasPrefix(a, "--") {
				return a[k+1:], true
			}
			if i+1 < len(args) {
				i++
				return args[i], true
			}
			return "", false
		}
		flag := a
		if k := strings.Index(a, "="); k >= 0 && strings.HasPrefix(a, "--") {
			flag = a[:k]
		}
		switch flag {
		case "--help", "-h":
			fmt.Fprint(e.stdout, shimAddUsage)
			return 0
		case "--name":
			v, ok := value()
			if !ok || v == "" {
				fmt.Fprintln(e.stderr, "justz shim add: --name needs a value")
				return 2
			}
			name, haveName = v, true
		case "--suffix":
			v, ok := value()
			if !ok {
				fmt.Fprintln(e.stderr, "justz shim add: --suffix needs a value")
				return 2
			}
			suffix = v
		case "--soft":
			soft = true
		case "--replace":
			over.Replace = true
		case "--force":
			over.Force = true
		case "--dry-run":
			dryRun = true
		case "--json":
			asJSON = true
		default:
			if strings.HasPrefix(a, "-") {
				fmt.Fprintf(e.stderr, "justz shim add: unknown flag %q\n\n%s", a, shimAddUsage)
				return 2
			}
			if havePath {
				fmt.Fprintf(e.stderr, "justz shim add: only one path is allowed, got %q and %q\n", path, a)
				return 2
			}
			path, havePath = a, true
		}
	}
	if path == "" {
		path = "."
	}

	justfile, err := shim.FindJustfile(path)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz shim add: %v\n", err)
		return 1
	}
	if !haveName {
		reg, err := registry.Discover(filepath.Dir(justfile))
		if err != nil {
			fmt.Fprintf(e.stderr, "justz shim add: no shim name given and %v\nPass --name <name>.\n", err)
			return 1
		}
		if name, err = registry.CLIName(reg); err != nil {
			fmt.Fprintf(e.stderr, "justz shim add: %v\nPass --name <name>.\n", err)
			return 1
		}
	}

	d, err := shim.DefaultDirs()
	if err != nil {
		fmt.Fprintf(e.stderr, "justz shim add: %v\n", err)
		return 1
	}
	res, err := d.Add(justfile, shim.AddOptions{Name: name, Suffix: suffix, Soft: soft, Overrides: over, DryRun: dryRun})
	var ce *shim.CollisionError
	if errors.As(err, &ce) {
		fmt.Fprintf(e.stderr, "justz shim add: cannot add %q:\n", name)
		for _, p := range ce.Problems {
			fmt.Fprintf(e.stderr, "  %s\n", p)
			switch p.Kind {
			case shim.Shadowed:
				fmt.Fprintln(e.stderr, "    use --force to add it anyway")
			case shim.NotOurs:
				fmt.Fprintln(e.stderr, "    move or delete that file and run the command again; --replace does not cover a file justz did not write")
			case shim.Owned, shim.Hides:
				fmt.Fprintln(e.stderr, "    use --replace to take the name over")
			}
		}
		return 1
	}
	if err != nil {
		fmt.Fprintf(e.stderr, "justz shim add: %v\n", err)
		return 1
	}

	if asJSON {
		return printJSON(e, res)
	}
	verb := "added"
	if res.DryRun {
		verb = "would add"
	} else if res.Updated {
		verb = "updated"
	}
	fmt.Fprintf(e.stdout, "%s shim %s (and %s) for %s\n", verb, res.Entry.Name, res.Entry.HelpName(), res.Entry.Justfile)
	for _, f := range res.Files {
		fmt.Fprintf(e.stdout, "  %s\n", f)
	}
	for _, w := range res.Warnings {
		fmt.Fprintf(e.stdout, "warning: %s\n", w)
	}
	return 0
}

func runShimRemove(args []string, e env) int {
	var name string
	var dryRun, asJSON bool
	for _, a := range args {
		switch {
		case a == "--help" || a == "-h":
			fmt.Fprint(e.stdout, "Usage: justz shim remove <name> [--dry-run] [--json]\n\nRemove a shim from the registry and delete its stubs. Files that justz did not write are left alone.\n")
			return 0
		case a == "--dry-run":
			dryRun = true
		case a == "--json":
			asJSON = true
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(e.stderr, "justz shim remove: unknown flag %q\n", a)
			return 2
		case name == "":
			name = a
		default:
			fmt.Fprintf(e.stderr, "justz shim remove: only one name is allowed, got %q and %q\n", name, a)
			return 2
		}
	}
	if name == "" {
		fmt.Fprintln(e.stderr, "justz shim remove: expected a shim name")
		return 2
	}
	d, err := shim.DefaultDirs()
	if err != nil {
		fmt.Fprintf(e.stderr, "justz shim remove: %v\n", err)
		return 1
	}
	res, err := d.Remove(name, dryRun)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz shim remove: %v\n", err)
		return 1
	}
	if asJSON {
		return printJSON(e, res)
	}
	verb := "removed"
	if res.DryRun {
		verb = "would remove"
	}
	fmt.Fprintf(e.stdout, "%s shim %s\n", verb, res.Name)
	for _, f := range res.Removed {
		fmt.Fprintf(e.stdout, "  deleted %s\n", f)
	}
	for _, f := range res.Skipped {
		fmt.Fprintf(e.stdout, "  left %s alone: justz did not write it\n", f)
	}
	return 0
}
