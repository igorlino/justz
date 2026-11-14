package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/igorlino/justz/internal/atomicfile"
	"github.com/igorlino/justz/internal/scaffold"
)

const initUsage = `Usage: justz init [--variant native|rich] [--name NAME] [--module NAME]... [--dry-run] [dir]

Scaffold a justfile that documents itself, in dir (default: the current
directory):

  justfile              _default and help recipes, and a mod line per module
  just/preamble.just    the settings every file shares (strict bash, positional
                        arguments); the justfile and each module import it
  just/<module>.just    one per --module, each with a _default that lists it
  config/commands.json  the registry, with --variant rich only

  --variant   native (default) keeps the documentation in the justfile and
              needs only just. rich adds the registry and needs justz wherever
              the help is rendered.
  --module    A command group, run as: just <module> <recipe>. Repeat the flag.
  --name      The heading of the help (default: the directory name)
  --dry-run   List the files and write nothing

justz init never overwrites: if any file it would write exists, or the
directory already has a justfile, it writes nothing and exits 1.

Exit codes: 0 done, 1 refused or error, 2 usage error.
`

func runInit(args []string, e env) int {
	opts := scaffold.Options{Variant: scaffold.Native}
	dryRun := false
	dir := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		// value returns the flag's value, from --flag=value or the next argument.
		value := func(flag string) (string, bool) {
			if v, ok := strings.CutPrefix(a, flag+"="); ok {
				return v, true
			}
			if a == flag && i+1 < len(args) {
				i++
				return args[i], true
			}
			return "", false
		}
		switch {
		case a == "--help" || a == "-h":
			fmt.Fprint(e.stdout, initUsage)
			return 0
		case a == "--dry-run":
			dryRun = true
		case a == "--variant" || strings.HasPrefix(a, "--variant="):
			v, ok := value("--variant")
			if !ok {
				fmt.Fprintln(e.stderr, "justz init: --variant needs a value")
				return 2
			}
			opts.Variant = v
		case a == "--name" || strings.HasPrefix(a, "--name="):
			v, ok := value("--name")
			if !ok {
				fmt.Fprintln(e.stderr, "justz init: --name needs a value")
				return 2
			}
			opts.Name = v
		case a == "--module" || strings.HasPrefix(a, "--module="):
			v, ok := value("--module")
			if !ok {
				fmt.Fprintln(e.stderr, "justz init: --module needs a value")
				return 2
			}
			opts.Modules = append(opts.Modules, v)
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(e.stderr, "justz init: unknown flag %q\n", a)
			return 2
		case dir == "":
			dir = a
		default:
			fmt.Fprintf(e.stderr, "justz init: only one directory is allowed, got %q and %q\n", dir, a)
			return 2
		}
	}

	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz init: %v\n", err)
		return 1
	}
	if opts.Name == "" {
		opts.Name = filepath.Base(abs)
	}
	files, err := scaffold.Build(opts)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz init: %v\n", err)
		return 2
	}

	// Check everything before writing anything, so a refusal leaves no half project.
	var clash []string
	for _, name := range []string{"justfile", "Justfile", ".justfile"} {
		if _, err := os.Lstat(filepath.Join(abs, name)); err == nil {
			clash = append(clash, name)
		}
	}
	for _, f := range files {
		if f.Path == "justfile" {
			continue
		}
		if _, err := os.Lstat(filepath.Join(abs, filepath.FromSlash(f.Path))); err == nil {
			clash = append(clash, f.Path)
		}
	}
	if len(clash) > 0 {
		fmt.Fprintf(e.stderr, "justz init: refusing to overwrite %s in %s. Nothing was written.\n", strings.Join(clash, ", "), abs)
		return 1
	}

	for _, f := range files {
		if dryRun {
			fmt.Fprintf(e.stdout, "would write %s\n", f.Path)
			continue
		}
		target := filepath.Join(abs, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			fmt.Fprintf(e.stderr, "justz init: %v\n", err)
			return 1
		}
		if err := atomicfile.Write(target, []byte(f.Content), 0o644); err != nil {
			fmt.Fprintf(e.stderr, "justz init: %v\n", err)
			return 1
		}
		fmt.Fprintf(e.stdout, "wrote %s\n", f.Path)
	}
	if dryRun {
		return 0
	}
	if opts.Variant == scaffold.Rich {
		fmt.Fprintln(e.stdout, "next: add recipes with [doc] and [group], list them in config/commands.json, then run: justz fmt && justz check")
	} else {
		fmt.Fprintln(e.stdout, "next: add recipes with [doc] and [group], then run: just")
	}
	return 0
}
