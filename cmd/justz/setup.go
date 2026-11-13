package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/igorlino/justz/internal/shim"
)

const setupUsage = `Usage: justz setup [--dry-run] [--json]

Get this machine ready, and check it. Safe to run again.

  - creates the shim directory if it does not exist
  - checks that just is installed
  - runs the shim checks (see: justz shim doctor)
  - tells you how to put the shim directory on your PATH if it is not there

justz never edits your PATH or your shell startup files. --dry-run shows what
would be created without creating anything.

Exit codes: 0 ready, 1 problems found, 2 usage error.
`

type setupReport struct {
	OK       bool           `json:"ok"`
	DryRun   bool           `json:"dryRun"`
	ShimDir  string         `json:"shimDir"`
	Created  bool           `json:"created"`
	PathHint string         `json:"pathHint,omitempty"`
	Findings []shim.Finding `json:"findings"`
}

func runSetup(args []string, e env) int {
	dryRun, asJSON := false, false
	for _, a := range args {
		switch a {
		case "--help", "-h":
			fmt.Fprint(e.stdout, setupUsage)
			return 0
		case "--dry-run":
			dryRun = true
		case "--json":
			asJSON = true
		default:
			fmt.Fprintf(e.stderr, "justz setup: unexpected argument %q\n\n%s", a, setupUsage)
			return 2
		}
	}

	d, err := shim.DefaultDirs()
	if err != nil {
		fmt.Fprintf(e.stderr, "justz setup: %v\n", err)
		return 1
	}

	report := setupReport{DryRun: dryRun, ShimDir: d.Shim}
	if _, err := os.Stat(d.Shim); os.IsNotExist(err) {
		report.Created = true
		if !dryRun {
			if err := os.MkdirAll(d.Shim, 0o755); err != nil {
				fmt.Fprintf(e.stderr, "justz setup: %v\n", err)
				return 1
			}
		}
	}

	justCheck := checkJust(e)
	report.Findings = append(report.Findings, shim.Finding{Check: justCheck.Name, OK: justCheck.OK, Detail: justCheck.Detail})
	findings, err := d.Doctor(e.lookPath)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz setup: %v\n", err)
		return 1
	}
	report.Findings = append(report.Findings, findings...)
	report.OK = shim.Healthy(report.Findings)

	if !d.ShimDirOnPath() {
		report.PathHint = pathHint(d)
	}

	if asJSON {
		if code := printJSON(e, report); code != 0 {
			return code
		}
	} else {
		switch {
		case report.Created && dryRun:
			fmt.Fprintf(e.stdout, "would create %s\n", d.Shim)
		case report.Created:
			fmt.Fprintf(e.stdout, "created %s\n", d.Shim)
		default:
			fmt.Fprintf(e.stdout, "%s already exists\n", d.Shim)
		}
		printFindings(e, report.Findings)
		if report.PathHint != "" {
			fmt.Fprintf(e.stdout, "\n%s is not on your PATH. To add it:\n  %s\n", d.Shim, report.PathHint)
		}
	}
	if !report.OK {
		return 1
	}
	return 0
}

// pathHint is a command the user can run themselves. justz never runs it.
func pathHint(d shim.Dirs) string {
	if d.GOOS == "windows" {
		return fmt.Sprintf(`[Environment]::SetEnvironmentVariable("Path", $env:Path + ";%s", "User")   (PowerShell, then open a new terminal)`, d.Shim)
	}
	return fmt.Sprintf(`echo 'export PATH="%s:$PATH"' >> ~/.%src   (then open a new terminal)`, d.Shim, shellName())
}

func shellName() string {
	sh := os.Getenv("SHELL")
	if i := strings.LastIndex(sh, "/"); i >= 0 {
		sh = sh[i+1:]
	}
	if sh == "zsh" || sh == "bash" {
		return sh
	}
	return "bash"
}
