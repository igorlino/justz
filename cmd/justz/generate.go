package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/igorlino/justz/internal/atomicfile"
	"github.com/igorlino/justz/internal/generate"
	"github.com/igorlino/justz/internal/orderedjson"
	"github.com/igorlino/justz/internal/shim"
)

const generateUsage = `Usage: justz generate [--stdout] [path]

Build the personal local overlay, .just-local-rich.json, from the justfile's own
[doc] and [arg] attributes, through just --dump. It gives you the richer help
and completion on a project that did not adopt a registry.

The file is derived: do not edit it, run this again. "values" lists you added
to arguments by hand are kept. The default path is .just-local-rich.json in the
project root (the directory with .git, or the justfile's directory).

justz refuses to write a file that git tracks, because the overlay is
personal. Add it to your global git ignore file, not to the project's.

  --stdout   Print the result and write nothing

Exit codes: 0 done, 1 error or refused, 2 usage error.
`

func runGenerate(args []string, e env) int {
	toStdout := false
	target := ""
	for _, a := range args {
		switch {
		case a == "--help" || a == "-h":
			fmt.Fprint(e.stdout, generateUsage)
			return 0
		case a == "--stdout":
			toStdout = true
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(e.stderr, "justz generate: unknown flag %q\n", a)
			return 2
		case target == "":
			target = a
		default:
			fmt.Fprintf(e.stderr, "justz generate: only one path is allowed, got %q and %q\n", target, a)
			return 2
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(e.stderr, "justz generate: %v\n", err)
		return 1
	}
	justfile, err := shim.FindJustfile(cwd)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz generate: %v\n", err)
		return 1
	}
	root := projectRoot(filepath.Dir(justfile))
	if target == "" {
		target = filepath.Join(root, ".just-local-rich.json")
	}

	dumpJSON, err := e.runCmd("just", "--justfile", justfile, "--dump", "--dump-format", "json")
	if err != nil {
		fmt.Fprintf(e.stderr, "justz generate: could not read the justfile with just: %v\n", err)
		return 1
	}

	var existing *orderedjson.Value
	if data, err := os.ReadFile(target); err == nil {
		if doc, err := orderedjson.Parse(data); err == nil {
			existing = doc
		}
	}
	res, err := generate.Build([]byte(dumpJSON), existing)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz generate: %v\n", err)
		return 1
	}
	for _, w := range res.Warnings {
		fmt.Fprintf(e.stderr, "justz generate: warning: %s\n", w)
	}
	data, err := orderedjson.Marshal(res.Doc)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz generate: %v\n", err)
		return 1
	}

	if toStdout {
		e.stdout.Write(data)
		return 0
	}
	if _, err := e.runCmd("git", "-C", root, "ls-files", "--error-unmatch", "--", target); err == nil {
		fmt.Fprintf(e.stderr, "justz generate: %s is tracked by git. The overlay is personal, so justz will not write it there. Pass another path, or untrack it first.\n", target)
		return 1
	}
	if err := atomicfile.Write(target, data, 0o644); err != nil {
		fmt.Fprintf(e.stderr, "justz generate: %v\n", err)
		return 1
	}
	fmt.Fprintf(e.stdout, "wrote %s\n", target)
	if _, err := e.runCmd("git", "-C", root, "check-ignore", "-q", "--", target); err != nil {
		fmt.Fprintf(e.stdout, "note: %s is not ignored by git. Add it to your global ignore file, for example:\n  echo %s >> ~/.gitignore_global\n", target, filepath.Base(target))
	}
	return 0
}

// projectRoot is the nearest directory at or above dir that contains .git, or
// dir itself when there is none.
func projectRoot(dir string) string {
	for d := dir; ; {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return dir
		}
		d = parent
	}
}
