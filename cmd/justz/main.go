package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// version is stamped at build time: go build -ldflags "-X main.version=1.2.3".
var version = "dev"

const usage = `justz - help, checks and shims for just

Usage:
  justz <command> [flags]

Commands:
  doctor        Check that justz can do its job (is just installed?)
  fmt [path]    Rewrite the command registry in its canonical order
  help          Show the project's commands (summary, or --expanded)
  check         Check how the project documents its commands (for CI)
  completions   Print a shell completion script (zsh or bash)
  assess        Report facts about the justfile and which variant fits
  generate      Build the personal .just-local-rich.json from a justfile
  resolve NAME  Print the group path a bare command name stands for
  shim add      Register a shim for a project (from any directory)
  shim list     List registered shims
  shim remove   Remove a shim
  shim doctor   Check every registered shim
  setup         Get this machine ready and check it

Flags:
  --as NAME     Act as the shim NAME (used by shim stubs)
  --local NAME  Act as the built-in jz shortcut (used when started as jz)
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
	// execJust runs just with the terminal attached and returns its exit code.
	execJust func(args []string) int
	// color and width describe the output for the help views.
	color bool
	width int
	// interactive is true when both input and output are a terminal, so it is
	// safe to prompt. stdin is where an answer is read from.
	interactive bool
	stdin       io.Reader
	// runFzf shows lines in fzf and returns the one chosen, or "" if the user
	// cancelled. self is the path of this binary, used by the preview.
	runFzf func(fzf, self string, lines []string) (string, error)
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
		execJust:    execJust,
		color:       useColor(os.Stdout),
		width:       terminalWidth(os.Stdout),
		interactive: term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())),
		stdin:       os.Stdin,
		runFzf:      runFzf,
	}
}

// execJust runs just with this process's terminal and the JUSTZ_* variables
// removed, so that justz's own settings never leak into a recipe.
func execJust(args []string) int {
	cmd := exec.Command("just", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "JUSTZ_") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	err := cmd.Run()
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	fmt.Fprintf(os.Stderr, "justz: could not run just: %v\n", err)
	return 127
}

func run(args []string, e env) int {
	if len(args) == 0 {
		// Asking for nothing is not a mistake: show what justz can do. A post-install
		// check that runs the executable without arguments must see success.
		fmt.Fprint(e.stdout, usage)
		return 0
	}
	switch args[0] {
	case "--version", "-version", "version":
		fmt.Fprintln(e.stdout, "justz "+currentVersion())
		return 0
	case "--help", "-h":
		fmt.Fprint(e.stdout, usage)
		return 0
	case "--as":
		return runAs(args[1:], e)
	case "help":
		return runHelp(args[1:], e)
	case "__complete":
		return runComplete(args[1:], e)
	case "completions":
		return runCompletions(args[1:], e)
	case "check":
		return runCheck(args[1:], e)
	case "assess":
		return runAssess(args[1:], e)
	case "--local":
		if len(args) < 2 {
			fmt.Fprintln(e.stderr, "justz: --local needs a name")
			return 2
		}
		return runLocal(args[1], args[2:], e)
	case "generate":
		return runGenerate(args[1:], e)
	case "resolve":
		return runResolve(args[1:], e)
	case "doctor":
		return runDoctor(args[1:], e)
	case "fmt":
		return runFmt(args[1:], e)
	case "setup":
		return runSetup(args[1:], e)
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
	args := os.Args[1:]
	// Started as jz or jzh (a link to the justz binary): act as that shortcut.
	if name := invokedAs(os.Args[0]); name != "" {
		args = append([]string{"--local", name}, args...)
	}
	os.Exit(run(args, osEnv()))
}
