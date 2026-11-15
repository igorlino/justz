package main

import (
	"fmt"

	chk "github.com/igorlino/justz/internal/check"
)

const checkUsage = `Usage: justz check [--json] [--strict] [--rule NAME]...

Check how the current project documents its commands, so a pre-commit hook or
CI job can fail on drift instead of a guard running inside recipes.

It reads the justfile through just --dump and, when there is one, the registry
(commands.json) or the local overlay. Rules:

  doc-missing, group-missing, doc-multiline-comment, variadic-interpolation
  registry-missing, registry-stale, registry-desc-diverges,
  registry-key-signature, registry-shape, registry-desc-long, registry-unsorted
  example-bare-command, example-single-flag, example-long-flag,
  example-enum-missing, example-flag-unknown (a warning), registry-exits,
  registry-env, registry-tags (a warning), volatile-text and
  detail-flag-no-row, detail-env-no-row, detail-long, detail-no-paragraphs, detail-restates-desc,
  tags-confirm (advice, warnings), default-public (a public default recipe: a warning,
  so check --strict fails on it; it is a dispatcher, not a command, so it is
  not counted)

  --json     Print the findings as JSON
  --strict   Warnings fail the check too
  --rule     Show only the findings of this rule. It is for reading, not for
             gating CI: it can hide errors and exit 0. Repeat the flag or separate
             names with commas. The counts and the exit code follow the
             findings that are shown. An unknown name is a usage error that
             lists the valid names

A registry-desc-diverges finding prints the [doc(...)] line to paste on its own
line and has it in the paste field of the JSON (none for a blank description or
one with control characters). justz never edits the justfile.

The text output prints a fix once for a run of findings of one rule that share
it. The JSON keeps the fix on every finding.

volatile-text finds text in a desc or detail that goes stale (TODO, FIXME, WIP,
placeholder, not implemented, coming soon, a count such as "11 checks", a
price, a date). detail-flag-no-row finds a flag that a detail names but no
argument row declares. detail-env-no-row finds an environment variable that a
detail names but no env row declares: an upper case name with an underscore
(NO_COLOR), or one written as $NAME, ${NAME}, export NAME or NAME= (acronyms
like HTTP, flags, paths and file names do not count; a constant like UTF_8 can fire, so silence it;
a name under 3 characters is not caught). detail-long (over 600 characters), detail-no-paragraphs
(over 300 with no blank line) and detail-restates-desc (the detail opens with
the words of the desc, for a desc of four words or more) are about the shape of
a detail. They are warnings, so check --strict fails on them; a detail that is
deliberately a list with no blank lines is silenced with justz-ignore. Silence either on one recipe with
[metadata('justz-ignore', '<rule>')].

Exit codes: 0 no errors, 1 errors (or warnings with --strict), 2 usage error.
`

type checkReport struct {
	OK       bool          `json:"ok"`
	Variant  string        `json:"variant"`
	Checked  int           `json:"checked"`
	Errors   int           `json:"errors"`
	Warnings int           `json:"warnings"`
	Findings []chk.Finding `json:"findings"`
	Rules    []string      `json:"rules,omitempty"` // the --rule filter, when one was given
}

func runCheck(args []string, e env) int {
	asJSON, strict := false, false
	var rules []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if ok, msg := ruleFlag(args, &i, &rules); ok {
			if msg != "" {
				fmt.Fprintf(e.stderr, "justz check: %s\n", msg)
				return 2
			}
			continue
		}
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

	if msg := unknownRule("check", rules, chk.Rules); msg != "" {
		fmt.Fprintln(e.stderr, msg)
		return 2
	}

	ctx, err := loadProjectCtx(e)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz check: %v\n", err)
		return 1
	}
	res, err := chk.Run(chk.Input{Dump: []byte(ctx.Dump), Variant: ctx.Variant, Registry: ctx.Registry, RegistrySource: ctx.Source, Sources: ctx.loadSources(e)})
	if err != nil {
		fmt.Fprintf(e.stderr, "justz check: %v\n", err)
		return 1
	}

	if want := ruleSet(rules); want != nil {
		kept := []chk.Finding{}
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
	report := checkReport{OK: !failed, Variant: ctx.Variant, Checked: res.Checked, Errors: errs, Warnings: warns, Findings: res.Findings, Rules: rules}

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
		if len(r.Rules) > 0 {
			fmt.Fprintf(e.stdout, "ok: %d public recipes checked (%s), no findings for rule %s\n", r.Checked, r.Variant, ruleScope(r.Rules))
			return
		}
		fmt.Fprintf(e.stdout, "ok: %d public recipes checked (%s), no problems\n", r.Checked, r.Variant)
		return
	}
	scope := ""
	if len(r.Rules) > 0 {
		scope = ", rule " + ruleScope(r.Rules)
	}
	fmt.Fprintf(e.stdout, "justz check: %s, %s (%d public recipes checked, %s%s)\n\n", plural(r.Errors, "error"), plural(r.Warnings, "warning"), r.Checked, r.Variant, scope)
	rules, fixes := make([]string, len(r.Findings)), make([]string, len(r.Findings))
	for i, f := range r.Findings {
		rules[i], fixes[i] = f.Rule, f.Fix
	}
	show, count := fixRuns(rules, fixes)
	for i, f := range r.Findings {
		fmt.Fprintf(e.stdout, "  %-7s %s: %s  [%s]\n", f.Severity, f.Subject, f.Message, f.Rule)
		if f.Paste != "" {
			fmt.Fprintf(e.stdout, "          paste: %s\n", f.Paste)
		}
		if f.Fix != "" && show[i] {
			fmt.Fprintf(e.stdout, "          %s\n", fixLine(f.Fix, count[i]))
		}
	}
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
