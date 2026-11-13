package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// version is stamped at build time: go build -ldflags "-X main.version=1.2.3".
var version = "dev"

const usage = `justz - help, checks and shims for just

Usage:
  justz <command> [flags]

Commands:
  doctor        Check that justz can do its job (is just installed?)
  shim list     List registered shims

Flags:
  --version     Print the version
  --help        Print this help

Add --json to a command to get machine-readable output.

Exit codes:
  0  success
  1  a check failed
  2  usage error
`

// env holds everything a command needs from the outside world, so tests can
// replace it.
type env struct {
	stdout, stderr io.Writer
	lookPath       func(file string) (string, error)
	runCmd         func(name string, args ...string) (string, error)
}

func osEnv() env {
	return env{
		stdout:   os.Stdout,
		stderr:   os.Stderr,
		lookPath: exec.LookPath,
		runCmd: func(name string, args ...string) (string, error) {
			out, err := exec.Command(name, args...).Output()
			return strings.TrimSpace(string(out)), err
		},
	}
}

func run(args []string, e env) int {
	if len(args) == 0 {
		fmt.Fprint(e.stderr, usage)
		return 2
	}
	switch args[0] {
	case "--version", "-version", "version":
		fmt.Fprintln(e.stdout, "justz "+version)
		return 0
	case "--help", "-h", "help":
		fmt.Fprint(e.stdout, usage)
		return 0
	case "doctor":
		return runDoctor(args[1:], e)
	case "shim":
		return runShim(args[1:], e)
	default:
		fmt.Fprintf(e.stderr, "justz: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

// parseJSONFlag accepts only an optional --json and rejects anything else.
func parseJSONFlag(args []string) (asJSON bool, err error) {
	for _, a := range args {
		if a != "--json" {
			return false, fmt.Errorf("unexpected argument %q", a)
		}
		asJSON = true
	}
	return asJSON, nil
}

func main() {
	os.Exit(run(os.Args[1:], osEnv()))
}
