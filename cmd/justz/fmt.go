package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/igorlino/justz/internal/atomicfile"
	"github.com/igorlino/justz/internal/orderedjson"
	"github.com/igorlino/justz/internal/registry"
)

const fmtUsage = `Usage: justz fmt [--check] [path]

Rewrite the command registry (commands.json) in its canonical order.

Without a path, justz looks for .*/config/commands.json, config/commands.json
and just/commands.json in the current directory (or uses $JUSTZ_CONFIG).

  --check   Do not write anything. Exit 1 if the file is not formatted.

Exit codes: 0 formatted (or already was), 1 not formatted with --check or an
error, 2 usage error.
`

func runFmt(args []string, e env) int {
	check := false
	path := ""
	for _, a := range args {
		switch {
		case a == "--check":
			check = true
		case a == "--help" || a == "-h":
			fmt.Fprint(e.stdout, fmtUsage)
			return 0
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(e.stderr, "justz fmt: unknown flag %q\n\n%s", a, fmtUsage)
			return 2
		case path == "":
			path = a
		default:
			fmt.Fprintf(e.stderr, "justz fmt: only one path is allowed, got %q and %q\n", path, a)
			return 2
		}
	}

	if path == "" {
		path = os.Getenv("JUSTZ_CONFIG")
	}
	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(e.stderr, "justz fmt: %v\n", err)
			return 1
		}
		if path, err = registry.Discover(cwd); err != nil {
			fmt.Fprintf(e.stderr, "justz fmt: %v\n", err)
			return 1
		}
	}

	original, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz fmt: %v\n", err)
		return 1
	}
	doc, err := orderedjson.Parse(original)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz fmt: %s is not valid JSON: %v\n", path, err)
		return 1
	}
	registry.Canonicalize(doc)
	formatted, err := orderedjson.Marshal(doc)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz fmt: %v\n", err)
		return 1
	}

	if bytes.Equal(original, formatted) {
		fmt.Fprintf(e.stdout, "%s is already formatted\n", path)
		return 0
	}
	if check {
		fmt.Fprintf(e.stdout, "%s is not formatted; run: justz fmt %s\n", path, path)
		return 1
	}
	if err := atomicfile.Write(path, formatted, 0o644); err != nil {
		fmt.Fprintf(e.stderr, "justz fmt: %v\n", err)
		return 1
	}
	fmt.Fprintf(e.stdout, "formatted %s\n", path)
	return 0
}
