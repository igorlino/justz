package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

// The dump shapes the secret rules read (expression trees, assignments, parameter
// defaults, dependency arguments) come from just, so one test runs a real just.
func TestLintFindsSecretsInARealJustfile(t *testing.T) {
	if _, err := exec.LookPath("just"); err != nil {
		t.Skip("just is not installed")
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "justfile"), `gh := env('GH_TOKEN', '')
alias := gh

typed api-token:
    @true

direct:
    @echo {{ quote(alias) }}

via-dep: (target env('DB_PASSWORD'))

target p:
    @echo {{ p }}

safe path=env('TOKEN_FILE'):
    @cat {{ path }}

[metadata('justz-ignore', 'secret-interpolated')]
quiet:
    @echo {{ gh }}

script:
    #!/usr/bin/env bash
    echo {{ gh }}
`)
	real := osEnv()
	var out, errOut bytes.Buffer
	real.stdout, real.stderr = &out, &errOut
	t.Chdir(dir)
	if code := run([]string{"lint", "--json"}, real); code != 0 {
		t.Fatalf("code %d: %s", code, errOut.String())
	}
	var rep struct {
		Findings []struct{ Rule, Subject, Message string } `json:"findings"`
	}
	if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
		t.Fatalf("%v: %s", err, out.String())
	}
	var got []string
	for _, f := range rep.Findings {
		got = append(got, f.Rule+" "+f.Subject)
		if strings.Contains(f.Message, "S3CR3T") {
			t.Errorf("message leaks: %s", f.Message)
		}
	}
	want := "secret-interpolated just direct,secret-interpolated just target,secret-parameter just typed"
	if g := strings.Join(got, ","); g != want {
		t.Errorf("findings = %q\nwant       %q", g, want)
	}
}
