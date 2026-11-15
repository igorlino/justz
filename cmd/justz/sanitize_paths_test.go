package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/igorlino/justz/internal/shim"
)

// escDir is a directory name a cloned repository can hold.
const escDir = "bad\x1b[31mdir\x1b]0;T\x07"

func escProject(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), escDir)
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "justfile"), "default:\n")
	return dir
}

func TestErrorsOnTheRealStderrDoNotCarryTheWorkingDirectory(t *testing.T) {
	dir := escProject(t)
	t.Chdir(dir)

	f, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	old := os.Stderr
	os.Stderr = f
	defer func() { os.Stderr = old }()

	e := osEnv()
	if code := run([]string{"fmt"}, e); code == 0 {
		t.Fatal("fmt should fail without a registry")
	}
	b, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "no commands.json found") {
		t.Fatalf("expected the discovery error, got %q", b)
	}
	assertCLISafe(t, "stderr", string(b))
}

func TestAssessPathsAreSafe(t *testing.T) {
	dir := escProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), `{"groups":{"core":{"commands":{"build":{}}}}}`)
	t.Chdir(dir)

	r := assessEnv()
	if code := run([]string{"assess"}, r.env); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
	}
	assertCLISafe(t, "assess output", r.out.String())
}

func TestShimDoctorFindingsAreSafe(t *testing.T) {
	e, out, _ := testEnv(true)
	printFindings(e, []shim.Finding{{Check: "path\x1b[31m", Shim: "s\x07", Detail: "d\x1b]0;t\x07"}})
	assertCLISafe(t, "findings", out.String())
	if !strings.Contains(out.String(), "�") {
		t.Errorf("the replacement is missing: %q", out.String())
	}
}

func TestDoctorPrintsASafeJustPath(t *testing.T) {
	e, out, _ := testEnv(true)
	e.lookPath = func(file string) (string, error) { return "/x\x1b[31m/" + file, nil }
	run([]string{"doctor"}, e)
	assertCLISafe(t, "doctor", out.String())
}
