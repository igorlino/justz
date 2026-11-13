package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	if err := writeFileAtomic(path, formatted); err != nil {
		fmt.Fprintf(e.stderr, "justz fmt: %v\n", err)
		return 1
	}
	fmt.Fprintf(e.stdout, "formatted %s\n", path)
	return 0
}

// writeFileAtomic replaces path with data through a temporary file in the same
// directory, so a crash never leaves a half-written registry. It refuses to
// write through a symbolic link.
func writeFileAtomic(path string, data []byte) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symbolic link; refusing to write through it", path)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".justz-fmt-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Chmod(tmpName, info.Mode().Perm()); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	return nil
}
