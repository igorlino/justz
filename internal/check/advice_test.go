package check

import (
	"strings"
	"testing"
)

func adviceRun(t *testing.T, cmds string, attrs []map[string]any) Result {
	t.Helper()
	dump := dumpOf(t, map[string]any{"build": rc("Build", attrs)}, nil)
	s := `{"groups":{"core":{"commands":` + cmds + `}}}`
	return run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, s), RegistrySource: []byte(s)})
}

func ignoreAttr(rule string) []map[string]any {
	return []map[string]any{{"metadata": []string{"justz-ignore", rule}}}
}

func TestVolatileText(t *testing.T) {
	hits := []string{"TODO: wire this up", "A placeholder for the build", "Not implemented yet", "Runs 11 checks", "Runs six steps",
		"Costs $4.99 a run", "Costs $10 a run", "Costs $5k", "Costs $5 USD", "Runs 1 step", "Runs one check", "Runs 3-4 steps", "Since 2026-05", "Added in March 2026", "FIXME later", "Coming soon", "Updated 3 Mar 2026"}
	for _, text := range hits {
		for _, field := range []string{"desc", "detail"} {
			res := adviceRun(t, `{"build":{"`+field+`":"`+text+`"}}`, nil)
			var f *Finding
			for i := range res.Findings {
				if res.Findings[i].Rule == RuleVolatileText {
					f = &res.Findings[i]
				}
			}
			if f == nil || f.Severity != Warning || f.Subject != "core.build" || !strings.HasPrefix(f.Message, field+" ") || f.Fix == "" {
				t.Errorf("%s %q: %+v", field, text, f)
			}
		}
	}
	for _, text := range []string{"Build the project", "Runs the checks and the tests", "Runs step by step", "Uses 2 workers", "Fills the template slots", "Takes $HOME", "Version 1.2.3", "Prints awk '{print $1}' and $9", "Version 1.2026-10", "Ids 12026-10", "Runs the steps", "Costs $5 a run", "May the build pass"} {
		if res := adviceRun(t, `{"build":{"desc":"`+text+`"}}`, nil); has(res, RuleVolatileText, "") {
			t.Errorf("%q: %v", text, rules(res))
		}
	}
}

func TestVolatileTextIgnore(t *testing.T) {
	res := adviceRun(t, `{"build":{"desc":"Runs 11 checks"}}`, ignoreAttr(RuleVolatileText))
	if has(res, RuleVolatileText, "") {
		t.Errorf("ignored rule reported: %v", rules(res))
	}
	if !has(adviceRun(t, `{"build":{"desc":"Runs 11 checks"}}`, ignoreAttr(RuleDetailFlagNoRow)), RuleVolatileText, "") {
		t.Errorf("ignoring another rule must not silence this one")
	}
}

func TestDetailFlagNoRow(t *testing.T) {
	cmds := `{"build":{"desc":"B","detail":"Use --fast or -q; see --[no-]color. Pass --out=dir, not --bogus.","args":[{"name":"-f, --fast"},{"name":"-q"},{"name":"--[no-]color"},{"name":"-o, --out <dir>"}]}}`
	res := adviceRun(t, cmds, nil)
	var got []string
	for _, f := range res.Findings {
		if f.Rule == RuleDetailFlagNoRow {
			got = append(got, f.Subject+" "+f.Message)
			if f.Severity != Warning || f.Fix == "" {
				t.Errorf("finding = %+v", f)
			}
		}
	}
	if len(got) != 1 || !strings.Contains(got[0], "core.build") || !strings.Contains(got[0], "--bogus") {
		t.Errorf("findings = %v", got)
	}
}

func TestDetailFlagNoRowFiresWithoutRowsButNotOnProse(t *testing.T) {
	if res := adviceRun(t, `{"build":{"desc":"B","detail":"Pass -x to skip."}}`, nil); !has(res, RuleDetailFlagNoRow, "core.build") {
		t.Errorf("a command with no rows: %v", rules(res))
	}
	for _, detail := range []string{"A non-stop run, see https://x.io/a--b and /tmp/--x or foo-bar", "Between 1-2 and -5 degrees", "Uses a -- terminator", "Say a - b"} {
		if res := adviceRun(t, `{"build":{"desc":"B","detail":"`+detail+`"}}`, nil); has(res, RuleDetailFlagNoRow, "") {
			t.Errorf("%q: %v", detail, rules(res))
		}
	}
	// A variadic row passes flags on, so the prose may name them.
	if res := adviceRun(t, `{"build":{"desc":"B","detail":"Pass --x","args":[{"name":"<args...>"}]}}`, nil); has(res, RuleDetailFlagNoRow, "") {
		t.Errorf("variadic row: %v", rules(res))
	}
}

func TestDetailFlagNoRowIgnore(t *testing.T) {
	res := adviceRun(t, `{"build":{"desc":"B","detail":"Pass --x"}}`, ignoreAttr(RuleDetailFlagNoRow))
	if has(res, RuleDetailFlagNoRow, "") {
		t.Errorf("ignored rule reported: %v", rules(res))
	}
}

func TestVolatileTextReportsEveryHit(t *testing.T) {
	res := adviceRun(t, `{"build":{"desc":"B","detail":"TODO: runs 3-4 steps since 2026-05 for $4.99, TODO again"}}`, nil)
	var msgs []string
	for _, f := range res.Findings {
		if f.Rule == RuleVolatileText {
			msgs = append(msgs, f.Message)
		}
	}
	if len(msgs) != 1 {
		t.Fatalf("findings = %v", msgs)
	}
	want := `detail has text that goes stale: "TODO", "3-4 steps", "2026-05", "$4.99"`
	if msgs[0] != want {
		t.Errorf("message = %q, want %q", msgs[0], want)
	}
}

func nestedAdviceRun(t *testing.T, parent, nested []map[string]any, parentDesc string) Result {
	t.Helper()
	dump := dumpOf(t, map[string]any{},
		map[string]any{"tool": map[string]any{
			"recipes": map[string]any{"x": rc("X", parent)},
			"modules": map[string]any{"sub": module(map[string]any{"x": rc("X", nested)})},
		}})
	s := `{"groups":{"tool":{"commands":{"x":{"desc":"` + parentDesc + `"}}}}}`
	return run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, s), RegistrySource: []byte(s)})
}

func TestAdviceIgnoreNestedSameNameRecipe(t *testing.T) {
	const vol = "Runs 11 checks"
	// An ignore on the nested recipe must not silence the parent.
	if res := nestedAdviceRun(t, nil, ignoreAttr(RuleVolatileText), vol); !has(res, RuleVolatileText, "tool.x") {
		t.Errorf("ignore on the nested recipe silenced the parent: %v", rules(res))
	}
	// An ignore on the parent still applies, whatever the nested recipe has.
	if res := nestedAdviceRun(t, ignoreAttr(RuleVolatileText), nil, vol); has(res, RuleVolatileText, "") {
		t.Errorf("ignore on the parent shadowed by the nested recipe: %v", rules(res))
	}
}

func TestRegistryStaleNotMaskedByNestedRecipe(t *testing.T) {
	dump := dumpOf(t, map[string]any{},
		map[string]any{"tool": map[string]any{
			"recipes": map[string]any{},
			"modules": map[string]any{"sub": module(map[string]any{"x": rc("X", nil)})},
		}})
	s := `{"groups":{"tool":{"commands":{"x":{"desc":"X"}}}}}`
	res := run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, s), RegistrySource: []byte(s)})
	if !has(res, RuleRegistryStale, "tool.x") {
		t.Errorf("nested recipe masked the stale entry: %v", rules(res))
	}
}

func TestDetailEnvNoRow(t *testing.T) {
	cmds := `{"build":{"desc":"B","detail":"Reads NO_COLOR and ${API_TOKEN}, then export DEBUG and HOME=/x; also HAS_ROW.","env":[{"name":"HAS_ROW"}]}}`
	res := adviceRun(t, cmds, nil)
	var got []string
	for _, f := range res.Findings {
		if f.Rule == RuleDetailEnvNoRow {
			got = append(got, f.Message)
			if f.Severity != Warning || f.Fix == "" || f.Subject != "core.build" {
				t.Errorf("finding = %+v", f)
			}
		}
	}
	if len(got) != 4 {
		t.Fatalf("findings = %v", got)
	}
	for i, n := range []string{"NO_COLOR", "API_TOKEN", "DEBUG", "HOME"} {
		if !strings.Contains(got[i], n+",") {
			t.Errorf("finding %d = %q, want %s", i, got[i], n)
		}
	}
}

func TestDetailEnvNoRowMissWhenDeclared(t *testing.T) {
	res := adviceRun(t, `{"build":{"desc":"B","detail":"Reads NO_COLOR.","env":[{"name":"NO_COLOR"}]}}`, nil)
	if has(res, RuleDetailEnvNoRow, "") {
		t.Errorf("declared variable reported: %v", rules(res))
	}
}

func TestDetailEnvNoRowIgnore(t *testing.T) {
	res := adviceRun(t, `{"build":{"desc":"B","detail":"Reads NO_COLOR."}}`, ignoreAttr(RuleDetailEnvNoRow))
	if has(res, RuleDetailEnvNoRow, "") {
		t.Errorf("ignored rule reported: %v", rules(res))
	}
}

func TestDetailEnvNoRowFalsePositives(t *testing.T) {
	for _, detail := range []string{
		"Sends HTTP over TLS, parses JSON from a URL; CI only.",
		"See README and AGENTS.md or docs/README.md, LICENSE.txt, CHANGELOG/",
		"Exit codes: OK 0, FAIL 1, USAGE 2. Use `JSON` output.",
		"Pass --NO_COLOR or -DEBUG; see --FOO_BAR=1 and /opt/MY_DIR and a.B_C.",
		"A NoColor, mixedCASE and xNO_COLOR word, or NO_COLORx, NO_COLOR_",
		"Compare ABC==DEF.",
		"Fetch https://x.com/p?KEY_ONE=2&KEY_TWO=3 now.",
		"The ÉNO_COLOR and NO_COLORé words.",
		"Short: AB, A_ and 1_A.",
	} {
		if res := adviceRun(t, `{"build":{"desc":"B","detail":"`+detail+`"}}`, nil); has(res, RuleDetailEnvNoRow, "") {
			t.Errorf("%q: %v", detail, res.Findings)
		}
	}
}

func TestDetailEnvNoRowExtraEntries(t *testing.T) {
	dump := dumpOf(t, map[string]any{"build": rc("Build", nil)}, nil)
	s := `{"groups":{"core":{"commands":{"build":{"desc":"B"}},"extra":{"ext":{"desc":"E","detail":"Reads NO_COLOR."}}}}}`
	res := run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, s), RegistrySource: []byte(s)})
	if !has(res, RuleDetailEnvNoRow, "core.ext") {
		t.Errorf("extra entry: %v", rules(res))
	}
}

func TestDetailEnvNoRowDollarWithPath(t *testing.T) {
	// Known gap: a sentence-final constant such as README_FIRST. is reported.
	for detail, want := range map[string]string{"Put it in $HOME/bin.": "HOME", "Reads $CONFIG_DIR/app.toml": "CONFIG_DIR", "Reads $FOO_BAR.conf": "FOO_BAR", "Reads ${VAR_X}/x": "VAR_X"} {
		res := adviceRun(t, `{"build":{"desc":"B","detail":"`+detail+`"}}`, nil)
		found := false
		for _, f := range res.Findings {
			if f.Rule == RuleDetailEnvNoRow && strings.Contains(f.Message, want+",") {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: %v", detail, res.Findings)
		}
	}
}
