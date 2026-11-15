package main

import (
	"fmt"

	chk "github.com/igorlino/justz/internal/check"
	"github.com/igorlino/justz/internal/lint"
)

const lintUsage = `Usage: justz lint [--json] [--strict] [--rule NAME]...

Find antipatterns that can be seen in the structure of the justfile, as advice.
It reads the justfile through just --dump and never runs a recipe. Every
finding is a warning, so a run fails only with --strict. check is the command
for documentation; lint is about the recipes themselves.

Rules:

  recipe-long           A recipe body over 40 lines. The message counts the {{ }}
                        interpolations, and those in single quotes or a quoted
                        here-document, which cannot be moved to a script by
                        exporting the value (JSON: interpolations,
                        quotedInterpolations, escapedBraces). A flat list of
                        invocations is the case for justz-ignore
  module-shell-unset    A module with linewise recipes that sets no shell, so
                        its recipes run with sh -cu and no pipefail. A set
                        shell line is the opt-out; there is no per-recipe one
  module-dispatcher-missing
                        A module with no default recipe, so a bare just <module>
                        runs a command. Only a first recipe named default or
                        _default is the dispatcher
  module-dispatcher-lists-root
                        A module default that runs just --list with justfile()
                        or justfile_directory(), the root's inside a module
  secret-parameter      A parameter with a secret-like name (TOKEN, PASSWORD,
                        SECRET, API_KEY...): its value is typed on the command
                        line, so it is in the process list
  secret-interpolated   A secret-like name inside {{ }} in a recipe that just
                        runs line by line: the value is in the shell's command
                        line. Shebang and [script] recipes are not looked at
  ignore-unknown-rule   A justz-ignore value that is not a rule

Every recipe is looked at, private ones and those in nested modules included.
Silence a rule on one recipe with the attribute that just already has (a rule
for a module has none; set a shell in the module instead):

  [metadata('justz-ignore', 'recipe-long')]

The secret rules read names only. They never read, print or store a value, and
no message carries the text of a recipe line.

  --json     Print the findings as JSON
  --strict   Warnings fail the run too
  --rule     Show only the findings of this rule. It is for reading, not for
             gating CI: it can hide errors and exit 0. Repeat the flag or separate
             names with commas. The counts and the exit code follow the
             findings that are shown. An unknown name is a usage error that
             lists the valid names

The text output prints a fix once for a run of findings of one rule that share
it. The JSON keeps the fix on every finding.

Exit codes: 0 no findings to act on, 1 warnings with --strict (or the justfile
could not be read), 2 usage error.
`

type lintReport struct {
	OK       bool           `json:"ok"`
	Checked  int            `json:"checked"`
	Errors   int            `json:"errors"`
	Warnings int            `json:"warnings"`
	Findings []lint.Finding `json:"findings"`
	Rules    []string       `json:"rules,omitempty"` // the --rule filter, when one was given
}

func runLint(args []string, e env) int {
	asJSON, strict := false, false
	var rules []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if ok, msg := ruleFlag(args, &i, &rules); ok {
			if msg != "" {
				fmt.Fprintf(e.stderr, "justz lint: %s\n", msg)
				return 2
			}
			continue
		}
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

	if msg := unknownRule("lint", rules, lint.Rules); msg != "" {
		fmt.Fprintln(e.stderr, msg)
		return 2
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

	if want := ruleSet(rules); want != nil {
		kept := []lint.Finding{}
		for _, f := range res.Findings {
			if want[f.Rule] {
				kept = append(kept, f)
			}
		}
		hidden, hiddenErrors := len(res.Findings)-len(kept), 0
		for _, f := range res.Findings {
			if !want[f.Rule] && f.Severity == chk.Error {
				hiddenErrors++
			}
		}
		res.Findings = kept
		if note := hiddenNote(hidden, hiddenErrors); note != "" && !asJSON {
			fmt.Fprintln(e.stderr, note)
		}
	}
	errs, warns := res.Count(chk.Error), res.Count(chk.Warning)
	failed := errs > 0 || (strict && warns > 0)
	report := lintReport{OK: !failed, Checked: res.Checked, Errors: errs, Warnings: warns, Findings: res.Findings, Rules: rules}

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
		if len(r.Rules) > 0 {
			fmt.Fprintf(e.stdout, "ok: %d recipes linted, no findings for rule %s\n", r.Checked, ruleScope(r.Rules))
			return
		}
		fmt.Fprintf(e.stdout, "ok: %d recipes linted, nothing to report\n", r.Checked)
		return
	}
	scope := ""
	if len(r.Rules) > 0 {
		scope = ", rule " + ruleScope(r.Rules)
	}
	fmt.Fprintf(e.stdout, "justz lint: %s (%d recipes linted%s)\n\n", plural(r.Warnings, "warning"), r.Checked, scope)
	rules, fixes := make([]string, len(r.Findings)), make([]string, len(r.Findings))
	for i, f := range r.Findings {
		rules[i], fixes[i] = f.Rule, f.Fix
	}
	show, count := fixRuns(rules, fixes)
	for i, f := range r.Findings {
		line := ""
		if f.Line > 0 {
			line = fmt.Sprintf(" (line %d)", f.Line)
		}
		fmt.Fprintf(e.stdout, "  %-7s %s%s: %s  [%s]\n", f.Severity, f.Subject, line, f.Message, f.Rule)
		if f.Fix != "" && show[i] {
			fmt.Fprintf(e.stdout, "          %s\n", fixLine(f.Fix, count[i]))
		}
	}
}
