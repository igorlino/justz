package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/igorlino/justz/internal/assess"
)

const assessUsage = `Usage: justz assess [--json]

Report facts about the justfile of the current project: how many recipes it
has and how many are documented, modules, recipes with a shebang, long
recipes, whether the registry covers every public recipe, and which
documentation variant (native, rich, local-rich) fits.

It reads the justfile through just --dump, never by searching the text, so the
numbers are exact. The recommendation follows the thresholds of the variant
guide and always comes with its reason.

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
	p("justfile", "%s", justfile)
	if r.RegistryPath != "" {
		p("variant", "%s (%s)", r.Variant, r.RegistryPath)
	} else {
		p("variant", "%s", r.Variant)
	}
	p("recipes", "%d (%d public, %d private)", r.Recipes.Total, r.Recipes.Public, r.Recipes.Private)
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
	fmt.Fprintf(e.stdout, "\nrecommendation: %s. %s\n", r.Recommendation.Variant, r.Recommendation.Reason)
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
