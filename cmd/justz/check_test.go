package main

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

const checkDump = `{
  "recipes": {
    "build": {"private": false, "doc": "Build", "body": [["echo"]], "parameters": [], "attributes": [{"group": "core"}]},
    "nodoc": {"private": false, "doc": null,    "body": [["echo"]], "parameters": [], "attributes": [{"group": "core"}]}
  },
  "modules": {}
}`

func checkEnv(dump string) *runtimeEnv {
	r := newRuntimeEnv()
	r.runCmd = func(name string, args ...string) (string, error) {
		if name == "just" {
			return dump, nil
		}
		return "", errors.New("unexpected command")
	}
	return r
}

func TestCheckFailsOnErrors(t *testing.T) {
	t.Chdir(gitProject(t)) // a native project
	r := checkEnv(checkDump)
	if code := run([]string{"check"}, r.env); code != 1 {
		t.Errorf("exit %d, want 1 (stdout %q)", code, r.out.String())
	}
	out := r.out.String()
	for _, want := range []string{"justz check: 1 error, 0 warnings", "error   just nodoc: no [doc]  [doc-missing]", "fix: add [doc("} {
		if !strings.Contains(out, want) {
			t.Errorf("output should contain %q:\n%s", want, out)
		}
	}
}

func TestCheckPassesOnACleanProject(t *testing.T) {
	t.Chdir(gitProject(t))
	clean := `{"recipes":{"build":{"private":false,"doc":"Build","body":[],"parameters":[],"attributes":[{"group":"core"}]}},"modules":{}}`
	r := checkEnv(clean)
	if code := run([]string{"check"}, r.env); code != 0 || !strings.HasPrefix(r.out.String(), "ok: 1 public recipes checked (native), no problems") {
		t.Errorf("code %d, out %q", code, r.out.String())
	}
}

func TestCheckWarningsOnlyFailWithStrict(t *testing.T) {
	dir := gitProject(t)
	long := strings.Repeat("word ", 40)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), `{"groups":{"core":{"commands":{"build":{"desc":"`+long+`"}}}}}`)
	t.Chdir(dir)
	dump := `{"recipes":{"build":{"private":false,"doc":"` + long + `","body":[],"parameters":[],"attributes":[]}},"modules":{}}`

	r := checkEnv(dump)
	if code := run([]string{"check"}, r.env); code != 0 || !strings.Contains(r.out.String(), "1 warning") {
		t.Errorf("a warning alone must pass: code %d, out %q", code, r.out.String())
	}
	r = checkEnv(dump)
	if code := run([]string{"check", "--strict"}, r.env); code != 1 {
		t.Errorf("--strict must fail on a warning: code %d", code)
	}
}

func TestCheckJSON(t *testing.T) {
	t.Chdir(gitProject(t))
	r := checkEnv(checkDump)
	if code := run([]string{"check", "--json"}, r.env); code != 1 {
		t.Fatalf("exit %d", code)
	}
	var got struct {
		OK       bool
		Variant  string
		Checked  int
		Errors   int
		Findings []struct{ Rule, Severity, Subject, Message, Fix string }
	}
	if err := json.Unmarshal(r.out.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, r.out.String())
	}
	if got.OK || got.Variant != "native" || got.Checked != 2 || got.Errors != 1 || len(got.Findings) != 1 || got.Findings[0].Rule != "doc-missing" || got.Findings[0].Fix == "" {
		t.Errorf("report = %+v", got)
	}
}

func TestCheckRegistryProject(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), `{"groups":{"core":{"commands":{"b":{"desc":"B"},"a":{"desc":"A"}}}}}`)
	t.Chdir(dir)
	dump := `{"recipes":{"a":{"private":false,"doc":"A","body":[],"parameters":[],"attributes":[]},"b":{"private":false,"doc":"B","body":[],"parameters":[],"attributes":[]}},"modules":{}}`
	r := checkEnv(dump)
	if code := run([]string{"check"}, r.env); code != 1 || !strings.Contains(r.out.String(), "registry-unsorted") || !strings.Contains(r.out.String(), "fix: run: justz fmt") {
		t.Errorf("an unsorted registry should fail with a hint: code %d, out %q", code, r.out.String())
	}
}

func TestCheckErrors(t *testing.T) {
	t.Chdir(gitProject(t))
	r := checkEnv(checkDump)
	r.runCmd = func(string, ...string) (string, error) { return "", errors.New("just is missing") }
	if code := run([]string{"check"}, r.env); code != 1 || !strings.Contains(r.errOut.String(), "could not read the justfile") {
		t.Errorf("just failing: code %d, stderr %q", code, r.errOut.String())
	}
	r = checkEnv(checkDump)
	if code := run([]string{"check", "--wat"}, r.env); code != 2 {
		t.Errorf("bad flag: code %d", code)
	}
	r = checkEnv(checkDump)
	if code := run([]string{"check", "--help"}, r.env); code != 0 || !strings.Contains(r.out.String(), "Usage: justz check") {
		t.Errorf("--help: code %d", code)
	}
}

// The full comment block comes from the text of just --dump, not from the JSON.
func TestCheckFindsMultilineComments(t *testing.T) {
	t.Chdir(gitProject(t))
	dump := `{"recipes":{"multi":{"private":false,"doc":"third line","body":[],"parameters":[],"attributes":[{"group":"core"}]}},"modules":{}}`
	text := "# First line\n# second line\n# third line\n[group('core')]\nmulti:\n    @echo multi\n"
	r := newRuntimeEnv()
	r.runCmd = func(name string, args ...string) (string, error) {
		for _, a := range args {
			if a == "--dump-format" {
				return dump, nil
			}
		}
		return text, nil
	}
	if code := run([]string{"check"}, r.env); code != 0 {
		t.Fatalf("a warning alone must pass: exit %d, out %q", code, r.out.String())
	}
	if !strings.Contains(r.out.String(), "doc-multiline-comment") || !strings.Contains(r.out.String(), "comment block of 3 lines") {
		t.Errorf("output:\n%s", r.out.String())
	}
}
