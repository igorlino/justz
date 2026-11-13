package main

import (
	"fmt"
	"strings"

	chk "github.com/igorlino/justz/internal/check"
)

const checkUsage = `Usage: justz check [--json] [--strict]

Check how the current project documents its commands, so a pre-commit hook or
CI job can fail on drift instead of a guard running inside recipes.

It reads the justfile through just --dump and, when there is one, the registry
(commands.json) or the local overlay. Rules:

  doc-missing, group-missing, variadic-interpolation
  registry-missing, registry-stale, registry-desc-diverges,
  registry-key-signature, registry-desc-long, registry-unsorted
  example-bare-command, example-single-flag, example-long-flag,
  example-enum-missing

  --json     Print the findings as JSON
  --strict   Warnings fail the check too

Exit codes: 0 no errors, 1 errors (or warnings with --strict), 2 usage error.
`

type checkReport struct {
	OK       bool          `json:"ok"`
	Variant  string        `json:"variant"`
	Checked  int           `json:"checked"`
	Errors   int           `json:"errors"`
	Warnings int           `json:"warnings"`
	Findings []chk.Finding `json:"findings"`
}

func runCheck(args []string, e env) int {
	asJSON, strict := false, false
	for _, a := range args {
		switch a {
		case "--help", "-h":
			fmt.Fprint(e.stdout, checkUsage)
			return 0
		case "--json":
			asJSON = true
		case "--strict":
			strict = true
		default:
			fmt.Fprintf(e.stderr, "justz check: unexpected argument %q\n\n%s", a, checkUsage)
			return 2
		}
	}

	ctx, err := loadProjectCtx(e)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz check: %v\n", err)
		return 1
	}
	res, err := chk.Run(chk.Input{Dump: []byte(ctx.Dump), Variant: ctx.Variant, Registry: ctx.Registry, RegistrySource: ctx.Source})
	if err != nil {
		fmt.Fprintf(e.stderr, "justz check: %v\n", err)
		return 1
	}

	errs, warns := res.Count(chk.Error), res.Count(chk.Warning)
	failed := errs > 0 || (strict && warns > 0)
	report := checkReport{OK: !failed, Variant: ctx.Variant, Checked: res.Checked, Errors: errs, Warnings: warns, Findings: res.Findings}

	if asJSON {
		if code := printJSON(e, report); code != 0 {
			return code
		}
	} else {
		printCheck(e, report)
	}
	if failed {
		return 1
	}
	return 0
}

func printCheck(e env, r checkReport) {
	if len(r.Findings) == 0 {
		fmt.Fprintf(e.stdout, "ok: %d public recipes checked (%s), no problems\n", r.Checked, r.Variant)
		return
	}
	fmt.Fprintf(e.stdout, "justz check: %s, %s (%d public recipes checked, %s)\n\n", plural(r.Errors, "error"), plural(r.Warnings, "warning"), r.Checked, r.Variant)
	for _, f := range r.Findings {
		fmt.Fprintf(e.stdout, "  %-7s %s: %s  [%s]\n", f.Severity, f.Subject, f.Message, f.Rule)
		if f.Fix != "" {
			fmt.Fprintf(e.stdout, "          fix: %s\n", strings.TrimSpace(f.Fix))
		}
	}
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
