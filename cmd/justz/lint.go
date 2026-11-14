package main

import (
	"fmt"
	"strings"

	chk "github.com/igorlino/justz/internal/check"
	"github.com/igorlino/justz/internal/lint"
)

const lintUsage = `Usage: justz lint [--json] [--strict]

Find antipatterns that can be seen in the structure of the justfile, as advice.
It reads the justfile through just --dump and never runs a recipe. Every
finding is a warning, so a run fails only with --strict. check is the command
for documentation; lint is about the recipes themselves.

Rules:

  recipe-long           A recipe body over 40 lines
  ignore-unknown-rule   A justz-ignore value that is not a rule

Every recipe is looked at, private ones and those in nested modules included.
Silence a rule on one recipe with the attribute that just already has:

  [metadata('justz-ignore', 'recipe-long')]

  --json     Print the findings as JSON
  --strict   Warnings fail the run too

Exit codes: 0 no findings to act on, 1 warnings with --strict (or the justfile
could not be read), 2 usage error.
`

type lintReport struct {
	OK       bool           `json:"ok"`
	Checked  int            `json:"checked"`
	Errors   int            `json:"errors"`
	Warnings int            `json:"warnings"`
	Findings []lint.Finding `json:"findings"`
}

func runLint(args []string, e env) int {
	asJSON, strict := false, false
	for _, a := range args {
		switch a {
		case "--help", "-h":
			fmt.Fprint(e.stdout, lintUsage)
			return 0
		case "--json":
			asJSON = true
		case "--strict":
			strict = true
		default:
			fmt.Fprintf(e.stderr, "justz lint: unexpected argument %q\n\n%s", a, lintUsage)
			return 2
		}
	}

	ctx, err := loadProjectCtx(e)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz lint: %v\n", err)
		return 1
	}
	res, err := lint.Run([]byte(ctx.Dump))
	if err != nil {
		fmt.Fprintf(e.stderr, "justz lint: %v\n", err)
		return 1
	}

	errs, warns := res.Count(chk.Error), res.Count(chk.Warning)
	failed := errs > 0 || (strict && warns > 0)
	report := lintReport{OK: !failed, Checked: res.Checked, Errors: errs, Warnings: warns, Findings: res.Findings}

	if asJSON {
		if code := printJSON(e, report); code != 0 {
			return code
		}
	} else {
		printLint(e, report)
	}
	if failed {
		return 1
	}
	return 0
}

func printLint(e env, r lintReport) {
	if len(r.Findings) == 0 {
		fmt.Fprintf(e.stdout, "ok: %d recipes linted, nothing to report\n", r.Checked)
		return
	}
	fmt.Fprintf(e.stdout, "justz lint: %s (%d recipes linted)\n\n", plural(r.Warnings, "warning"), r.Checked)
	for _, f := range r.Findings {
		line := ""
		if f.Line > 0 {
			line = fmt.Sprintf(" (line %d)", f.Line)
		}
		fmt.Fprintf(e.stdout, "  %-7s %s%s: %s  [%s]\n", f.Severity, f.Subject, line, f.Message, f.Rule)
		if f.Fix != "" {
			fmt.Fprintf(e.stdout, "          fix: %s\n", strings.TrimSpace(f.Fix))
		}
	}
}
