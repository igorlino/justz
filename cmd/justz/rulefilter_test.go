package main

import (
	"encoding/json"
	"strings"
	"testing"
)

const manyDump = `{"recipes":{
 "a":{"private":false,"doc":null,"body":[],"parameters":[],"attributes":[]},
 "b":{"private":false,"doc":null,"body":[],"parameters":[],"attributes":[]},
 "c":{"private":false,"doc":null,"body":[],"parameters":[],"attributes":[]}},"modules":{}}`

func TestCheckPrintsTheFixOncePerRule(t *testing.T) {
	t.Chdir(gitProject(t))
	r := checkEnv(manyDump)
	run([]string{"check"}, r.env)
	out := r.out.String()
	if n := strings.Count(out, "fix: add [doc("); n != 1 {
		t.Errorf("doc-missing fix printed %d times:\n%s", n, out)
	}
	if n := strings.Count(out, "fix: add [group("); n != 1 {
		t.Errorf("group-missing fix printed %d times:\n%s", n, out)
	}
	if !strings.Contains(out, "applies to all 3 findings of this rule") {
		t.Errorf("the run length is missing:\n%s", out)
	}
	if n := strings.Count(out, "[doc-missing]"); n != 3 {
		t.Errorf("every finding must stay listed, got %d:\n%s", n, out)
	}
}

func TestCheckJSONKeepsEveryFix(t *testing.T) {
	t.Chdir(gitProject(t))
	r := checkEnv(manyDump)
	run([]string{"check", "--json"}, r.env)
	var rep struct {
		Findings []struct{ Rule, Fix string }
	}
	if err := json.Unmarshal(r.out.Bytes(), &rep); err != nil {
		t.Fatal(err)
	}
	if len(rep.Findings) != 6 {
		t.Fatalf("findings = %d", len(rep.Findings))
	}
	for _, f := range rep.Findings {
		if f.Fix == "" {
			t.Errorf("a finding lost its fix: %+v", f)
		}
	}
}

func TestCheckRuleFiltersFindingsCountsAndExit(t *testing.T) {
	for _, args := range [][]string{
		{"check", "--rule", "doc-missing"},
		{"check", "--rule=doc-missing"},
		{"check", "--rule", "doc-missing,doc-missing"},
		{"check", "--rule", "doc-missing", "--rule", "doc-missing"},
	} {
		t.Chdir(gitProject(t))
		r := checkEnv(manyDump)
		code := run(args, r.env)
		out := r.out.String()
		if code != 1 || strings.Contains(out, "[group-missing]") || !strings.Contains(out, "justz check: 3 errors, 0 warnings") || !strings.Contains(out, "rule doc-missing") {
			t.Errorf("%v: code %d\n%s", args, code, out)
		}
	}
	// The exit code follows what is shown: a rule with no finding passes.
	t.Chdir(gitProject(t))
	r := checkEnv(manyDump)
	if code := run([]string{"check", "--rule", "variadic-interpolation"}, r.env); code != 0 || !strings.Contains(r.out.String(), "no findings for rule variadic-interpolation") {
		t.Errorf("code %d\n%s", code, r.out.String())
	}
	// --json carries the filter and the filtered counts.
	r = checkEnv(manyDump)
	run([]string{"check", "--json", "--rule", "group-missing"}, r.env)
	var rep struct {
		Errors   int
		Rules    []string
		Findings []struct{ Rule string }
	}
	if err := json.Unmarshal(r.out.Bytes(), &rep); err != nil || rep.Errors != 3 || len(rep.Findings) != 3 || rep.Rules[0] != "group-missing" {
		t.Errorf("report = %+v, err %v", rep, err)
	}
}

func TestRuleWithStrictUsesTheFilteredWarnings(t *testing.T) {
	t.Chdir(gitProject(t))
	dump := lintDump(45, "")
	r := checkEnv(dump)
	if code := run([]string{"lint", "--strict", "--rule", "secret-parameter"}, r.env); code != 0 {
		t.Errorf("a filtered-out warning must not fail --strict, exit %d\n%s", code, r.out.String())
	}
	r = checkEnv(dump)
	if code := run([]string{"lint", "--strict", "--rule", "recipe-long"}, r.env); code != 1 {
		t.Errorf("exit %d, want 1\n%s", code, r.out.String())
	}
}

func TestRuleUnknownIsAUsageError(t *testing.T) {
	for _, cmd := range []string{"check", "lint"} {
		t.Chdir(gitProject(t))
		r := checkEnv(manyDump)
		if code := run([]string{cmd, "--rule", "nope"}, r.env); code != 2 {
			t.Errorf("%s: exit %d, want 2", cmd, code)
		}
		msg := r.errOut.String()
		if !strings.Contains(msg, `unknown rule "nope"`) || !strings.Contains(msg, "valid rules:") {
			t.Errorf("%s: message %q", cmd, msg)
		}
	}
	r := checkEnv(manyDump)
	if code := run([]string{"check", "--rule"}, r.env); code != 2 {
		t.Errorf("a missing value: exit %d, want 2", code)
	}
	r = checkEnv(manyDump)
	if msg := func() string { run([]string{"lint", "--rule", "doc-missing"}, r.env); return r.errOut.String() }(); !strings.Contains(msg, "recipe-long") {
		t.Errorf("lint must list lint rules: %q", msg)
	}
}

func TestLintPrintsTheFixOncePerRule(t *testing.T) {
	t.Chdir(gitProject(t))
	body := strings.TrimSuffix(strings.Repeat(`["echo"],`, 45), ",")
	rc := `{"private":false,"doc":"D","body":[` + body + `],"parameters":[],"attributes":[]}`
	r := checkEnv(`{"recipes":{"a":` + rc + `,"b":` + rc + `,"c":` + rc + `},"modules":{}}`)
	run([]string{"lint"}, r.env)
	out := r.out.String()
	if n := strings.Count(out, "fix: move the logic"); n != 1 || !strings.Contains(out, "all 3 findings") || strings.Count(out, "[recipe-long]") != 3 {
		t.Errorf("output:\n%s", out)
	}
}

func TestFixPrintedOnceWhenARuleSpansBothSeverities(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, dir+"/config/commands.json", `{"groups":{
	  "core":{"commands":{"check":{"desc":"C"},"help":{"desc":"H"}},"examples":[{"cmd":"just check"},{"cmd":"just help"}]},
	  "a":{"commands":{"r1":{"desc":"R"}},"examples":[{"cmd":"just a r1"}]}}}`)
	t.Chdir(dir)
	dump := `{"recipes":{"check":{"private":false,"doc":"C","body":[],"parameters":[],"attributes":[]},"help":{"private":false,"doc":"H","body":[],"parameters":[],"attributes":[]}},
	 "modules":{"a":{"recipes":{"r1":{"private":false,"doc":"R","body":[],"parameters":[],"attributes":[]}},"modules":{}}}}`
	for _, args := range [][]string{{"check"}, {"check", "--rule", "example-bare-command"}} {
		r := checkEnv(dump)
		run(args, r.env)
		out := r.out.String()
		if n := strings.Count(out, "fix: remove it, or show"); n != 1 || strings.Count(out, "[example-bare-command]") != 3 || !strings.Contains(out, "applies to all 3 findings of this rule") {
			t.Errorf("%v: output:\n%s", args, out)
		}
	}
}

func TestRuleNotesHiddenErrorsOnStderr(t *testing.T) {
	t.Chdir(gitProject(t))
	r := checkEnv(manyDump)
	if code := run([]string{"check", "--strict", "--rule", "variadic-interpolation"}, r.env); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if note := r.errOut.String(); !strings.Contains(note, "note: --rule hides 6 other findings (6 errors)") || !strings.Contains(note, "gate CI") {
		t.Errorf("note = %q", note)
	}
	r = checkEnv(manyDump)
	run([]string{"check", "--json", "--rule", "variadic-interpolation"}, r.env)
	if r.errOut.Len() != 0 {
		t.Errorf("JSON mode must not print the note: %q", r.errOut.String())
	}
	r = checkEnv(manyDump)
	run([]string{"check", "--rule", "doc-missing,group-missing"}, r.env)
	if r.errOut.Len() != 0 {
		t.Errorf("nothing hidden, so no note: %q", r.errOut.String())
	}
}

func TestRuleRepeatedNameIsShownOnce(t *testing.T) {
	t.Chdir(gitProject(t))
	r := checkEnv(manyDump)
	run([]string{"check", "--rule", "doc-missing", "--rule", "doc-missing,doc-missing"}, r.env)
	if out := r.out.String(); !strings.Contains(out, "rule doc-missing)") {
		t.Errorf("header repeats the rule:\n%s", out)
	}
}

func TestCheckTextPrintsThePasteLineOnItsOwnLine(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, dir+"/config/commands.json", `{"groups":{"core":{"commands":{"build":{"desc":"It's new"}}}}}`)
	t.Chdir(dir)
	r := checkEnv(`{"recipes":{"build":{"private":false,"doc":"Old","body":[],"parameters":[],"attributes":[]}},"modules":{}}`)
	run([]string{"check"}, r.env)
	out := r.out.String()
	if !strings.Contains(out, "\n          paste: [doc(\"It's new\")]\n") {
		t.Errorf("paste line:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "paste:") && strings.Contains(line, "[registry-desc-diverges]") {
			t.Errorf("the rule tag follows the paste line: %q", line)
		}
	}
	r = checkEnv(`{"recipes":{"build":{"private":false,"doc":"Old","body":[],"parameters":[],"attributes":[]}},"modules":{}}`)
	run([]string{"check", "--json"}, r.env)
	var rep struct {
		Findings []struct{ Message, Paste string }
	}
	if err := json.Unmarshal(r.out.Bytes(), &rep); err != nil || rep.Findings[0].Paste != `[doc("It's new")]` || strings.Contains(rep.Findings[0].Message, "paste") {
		t.Errorf("json = %s", r.out.String())
	}
}
