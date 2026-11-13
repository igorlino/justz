package main

import (
	"fmt"
	"io"
	"os"
)

// version is stamped at build time: go build -ldflags "-X main.version=1.2.3".
var version = "dev"

const usage = `justz - help, checks and shims for just

Usage:
  justz --version
  justz --help
`

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "--version", "-version", "version":
		fmt.Fprintln(stdout, "justz "+version)
		return 0
	case "--help", "-h", "help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "justz: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
