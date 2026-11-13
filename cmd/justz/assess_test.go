package main

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

const assessDump = `{
  "recipes": {
    "build": {"private": false, "doc": "Build", "shebang": false, "body": [["echo"]], "parameters": []},
    "test":  {"private": false, "doc": null,    "shebang": true,  "body": [["echo"]], "parameters": []},
    "_aux":  {"private": true,  "doc": null,    "shebang": false, "body": [["echo"]], "parameters": []}
  },
  "modules": {
    "tool": {"recipes": {"clean": {"private": false, "doc": "Clean", "shebang": false, "body": [["echo"]], "parameters": []}}}
  }
}`

func assessEnv() *runtimeEnv {
	r := newRuntimeEnv()
	r.runCmd = func(name string, args ...string) (string, error) {
		if name == "just" {
			return assessDump, nil
		}
		return "", errors.New("unexpected command")
	}
	return r
}

func TestAssessRichProject(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), `{"groups":{"core":{"commands":{"build":{},"old":{}}},"tool":{"commands":{"clean":{}}}}}`)
	t.Chdir(dir)

	r := assessEnv()
	if code := run([]string{"assess"}, r.env); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
	}
	out := r.out.String()
	for _, want := range []string{
		"variant        rich (",
		"recipes        4 (3 public, 1 private)",
		"2 of 3 public recipes have a doc",
		"modules        1",
		"shebang        1 recipes",
		"1 public recipes missing from it, 1 stale entries",
		"missing: test",
		"stale: old",
		"recommendation: native.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output should contain %q:\n%s", want, out)
		}
	}
}

func TestAssessNativeAndOverlay(t *testing.T) {
	dir := gitProject(t)
	t.Chdir(dir)
	r := assessEnv()
	if code := run([]string{"assess"}, r.env); code != 0 || !strings.Contains(r.out.String(), "variant        native\n") {
		t.Errorf("native: code %d, out %q", code, r.out.String())
	}
	if strings.Contains(r.out.String(), "\nregistry ") {
		t.Errorf("a native project has no registry line: %q", r.out.String())
	}

	writeFile(t, filepath.Join(dir, ".just-local-rich.json"), `{"groups":{}}`)
	r = assessEnv()
	if code := run([]string{"assess"}, r.env); code != 0 || !strings.Contains(r.out.String(), "variant        local-rich (") {
		t.Errorf("overlay: code %d, out %q", code, r.out.String())
	}
}

func TestAssessJSON(t *testing.T) {
	dir := gitProject(t)
	t.Chdir(dir)
	r := assessEnv()
	if code := run([]string{"assess", "--json"}, r.env); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var got struct {
		Justfile       string
		Variant        string
		Recipes        struct{ Total, Public, Private, Documented, Undocumented int }
		Modules        int
		Recommendation struct{ Variant, Reason string }
	}
	if err := json.Unmarshal(r.out.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, r.out.String())
	}
	if got.Justfile == "" || got.Variant != "native" || got.Recipes.Total != 4 || got.Recipes.Undocumented != 1 || got.Modules != 1 || got.Recommendation.Reason == "" {
		t.Errorf("report = %+v", got)
	}
}

func TestAssessErrors(t *testing.T) {
	dir := gitProject(t)
	t.Chdir(dir)
	r := assessEnv()
	r.runCmd = func(name string, args ...string) (string, error) { return "", errors.New("just is missing") }
	if code := run([]string{"assess"}, r.env); code != 1 || !strings.Contains(r.errOut.String(), "could not read the justfile") {
		t.Errorf("just failing: code %d, stderr %q", code, r.errOut.String())
	}
	r = assessEnv()
	if code := run([]string{"assess", "--wat"}, r.env); code != 2 {
		t.Errorf("bad flag: code %d", code)
	}
}
