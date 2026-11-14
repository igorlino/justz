package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func lintDump(lines int, attrs string) string {
	body := strings.TrimSuffix(strings.Repeat(`["echo"],`, lines), ",")
	return `{"recipes":{"deploy":{"private":false,"doc":"Deploy","body":[` + body + `],"parameters":[],"attributes":[` + attrs + `]}},"modules":{}}`
}

func TestLintWarningsOnlyFailWithStrict(t *testing.T) {
	t.Chdir(gitProject(t))
	dump := lintDump(45, "")

	r := checkEnv(dump)
	if code := run([]string{"lint"}, r.env); code != 0 {
		t.Errorf("exit %d, want 0 (warnings do not fail)", code)
	}
	out := r.out.String()
	for _, want := range []string{"justz lint: 1 warning", "warning just deploy: body has 45 lines (limit 40)  [recipe-long]", "fix: move the logic to scripts/"} {
		if !strings.Contains(out, want) {
			t.Errorf("output should contain %q:\n%s", want, out)
		}
	}

	r = checkEnv(dump)
	if code := run([]string{"lint", "--strict"}, r.env); code != 1 {
		t.Errorf("exit %d, want 1 with --strict", code)
	}
}

func TestLintCleanProject(t *testing.T) {
	t.Chdir(gitProject(t))
	r := checkEnv(lintDump(3, ""))
	if code := run([]string{"lint", "--strict"}, r.env); code != 0 || !strings.HasPrefix(r.out.String(), "ok: 1 recipes linted, nothing to report") {
		t.Errorf("code %d, out %q", code, r.out.String())
	}
}

func TestLintIgnoreAttributeSilences(t *testing.T) {
	t.Chdir(gitProject(t))
	r := checkEnv(lintDump(45, `{"metadata":["justz-ignore","recipe-long"]}`))
	if code := run([]string{"lint", "--strict"}, r.env); code != 0 {
		t.Errorf("exit %d: %s", code, r.out.String())
	}
}

func TestLintJSON(t *testing.T) {
	t.Chdir(gitProject(t))
	r := checkEnv(lintDump(45, ""))
	if code := run([]string{"lint", "--json"}, r.env); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var got struct {
		OK       bool `json:"ok"`
		Checked  int  `json:"checked"`
		Warnings int  `json:"warnings"`
		Findings []struct {
			Rule, Severity, Subject, Message, Fix string
		} `json:"findings"`
	}
	if err := json.Unmarshal(r.out.Bytes(), &got); err != nil {
		t.Fatalf("%v: %s", err, r.out.String())
	}
	if !got.OK || got.Checked != 1 || got.Warnings != 1 || len(got.Findings) != 1 || got.Findings[0].Rule != "recipe-long" || got.Findings[0].Severity != "warning" {
		t.Errorf("report = %+v", got)
	}
}

func TestLintUsageErrors(t *testing.T) {
	for _, args := range [][]string{{"lint", "--nope"}, {"lint", "path"}} {
		r := checkEnv(lintDump(1, ""))
		if code := run(args, r.env); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
}
