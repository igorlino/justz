package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/igorlino/justz/internal/assess"
	"github.com/igorlino/justz/internal/textsafe"
)

const assessUsage = `Usage: justz assess [--json]

Report facts about the justfile of the current project: how many recipes it
has and how many are documented, modules, recipes with a shebang, long
recipes, whether the registry covers every public recipe, which documentation
variant (native, rich, local-rich) it uses, and which one is recommended.

It reads the justfile through just --dump, never by searching the text, so the
numbers are exact. The recommendation follows the variant guide: rich if
every machine that renders the help can install justz, and otherwise what the
project has without a registry (native, or local-rich with an overlay). justz
cannot see which holds, so it prints the condition. With --json they are the
fields variant, condition, otherwise and reason of recommendation.

A public default recipe is a dispatcher, not a command: it is left out of the
counts, as justz check leaves it out, and assess advises renaming it _default.
(justz check reports it as the warning default-public, which fails
check --strict.)
It also advises when a registry exists and a default recipe does not run
justz help (shim names such as tz are not recognized), since detail kept in
the registry is then not shown. Advice is not a rule.

Exit codes: 0 done, 1 error, 2 usage error.
`

func runAssess(args []string, e env) int {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			fmt.Fprint(e.stdout, assessUsage)
			return 0
		}
	}
	asJSON, err := parseJSONFlag(args)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz assess: %v\n", err)
		return 2
	}

	ctx, err := loadProjectCtx(e)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz assess: %v\n", err)
		return 1
	}
	in := assess.Input{Dump: []byte(ctx.Dump), Variant: ctx.Variant, RegistryPath: ctx.RegistryPath, Registry: ctx.Registry}
	justfile := ctx.Justfile

	report, err := assess.Assess(in)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz assess: %v\n", err)
		return 1
	}
	if asJSON {
		return printJSON(e, struct {
			Justfile string `json:"justfile"`
			assess.Report
		}{justfile, report})
	}
	printAssess(e, justfile, report)
	return 0
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func printAssess(e env, justfile string, r assess.Report) {
	p := func(label, format string, a ...any) {
		fmt.Fprintf(e.stdout, "%-14s "+format+"\n", append([]any{label}, a...)...)
	}
	p("justfile", "%s", textsafe.Line(justfile))
	if r.RegistryPath != "" {
		p("variant", "%s (%s)", r.Variant, r.RegistryPath)
	} else {
		p("variant", "%s", r.Variant)
	}
	if r.Recipes.Dispatchers > 0 {
		p("recipes", "%d (%d public, %d private, %d dispatcher)", r.Recipes.Total, r.Recipes.Public, r.Recipes.Private, r.Recipes.Dispatchers)
	} else {
		p("recipes", "%d (%d public, %d private)", r.Recipes.Total, r.Recipes.Public, r.Recipes.Private)
	}
	p("documented", "%d of %d public recipes have a doc", r.Recipes.Documented, r.Recipes.Public)
	p("modules", "%d", r.Modules)
	p("shebang", "%d recipes", r.ShebangRecipes)
	if r.Long.Longest > 0 {
		p("long bodies", "%d over 40 lines, %d over 100; longest is %d lines (%s)", r.Long.Over40, r.Long.Over100, r.Long.Longest, r.Long.LongestName)
	}
	p("many options", "%d commands with more than two documented parameters or registry args", r.ManyOptions)
	if r.Registry != nil {
		p("registry", "%d public recipes missing from it, %d stale entries", len(r.Registry.Missing), len(r.Registry.Stale))
		printList(e, "missing", r.Registry.Missing)
		printList(e, "stale", r.Registry.Stale)
	}
	for _, a := range r.Advice {
		fmt.Fprintf(e.stdout, "advice: %s\n", textsafe.Line(a))
	}
	fmt.Fprintf(e.stdout, "\nrecommendation: %s if %s, otherwise %s. %s\n", r.Recommendation.Variant, r.Recommendation.Condition, r.Recommendation.Otherwise, r.Recommendation.Reason)
}

func printList(e env, label string, items []string) {
	if len(items) == 0 {
		return
	}
	shown := items
	more := ""
	if len(items) > 10 {
		shown, more = items[:10], fmt.Sprintf(" and %d more", len(items)-10)
	}
	fmt.Fprintf(e.stdout, "  %s: %s%s\n", label, strings.Join(shown, ", "), more)
}
