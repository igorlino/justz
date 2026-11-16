package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// freshMachine points justz at an empty config and a shim directory that does
// not exist yet, with the shim directory on the PATH.
func freshMachine(t *testing.T) (shimDir string) {
	t.Helper()
	root := t.TempDir()
	shimDir = filepath.Join(root, "bin") // not created
	t.Setenv("JUSTZ_CONFIG_DIR", filepath.Join(root, "cfg"))
	t.Setenv("JUSTZ_SHIM_DIR", shimDir)
	t.Setenv("PATH", shimDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return shimDir
}

func TestSetupCreatesTheShimDirectory(t *testing.T) {
	shimDir := freshMachine(t)
	e, out, _ := testEnv(true) // just and justz are findable in the fake environment
	if code := run([]string{"setup"}, e); code != 0 {
		t.Fatalf("exit %d, out %q", code, out.String())
	}
	if info, err := os.Stat(shimDir); err != nil || !info.IsDir() {
		t.Errorf("the shim directory was not created: %v", err)
	}
	if !strings.Contains(out.String(), "created "+shimDir) || !strings.Contains(out.String(), "[ok] just:") {
		t.Errorf("output = %q", out.String())
	}

	// Running it again is safe and says so.
	e, out, _ = testEnv(true)
	if code := run([]string{"setup"}, e); code != 0 || !strings.Contains(out.String(), "already exists") {
		t.Errorf("second run: code %d, out %q", code, out.String())
	}
}

func TestSetupDryRunCreatesNothing(t *testing.T) {
	shimDir := freshMachine(t)
	e, out, _ := testEnv(true)
	if code := run([]string{"setup", "--dry-run"}, e); code != 0 || !strings.Contains(out.String(), "would create") {
		t.Fatalf("code %d, out %q", code, out.String())
	}
	if _, err := os.Stat(shimDir); !os.IsNotExist(err) {
		t.Error("a dry run must not create the shim directory")
	}
}

func TestSetupPrintsAPathHintButNeverEditsPath(t *testing.T) {
	root := t.TempDir()
	shimDir := filepath.Join(root, "bin")
	t.Setenv("JUSTZ_CONFIG_DIR", filepath.Join(root, "cfg"))
	t.Setenv("JUSTZ_SHIM_DIR", shimDir)
	before := os.Getenv("PATH")
	e, out, _ := testEnv(true)
	if code := run([]string{"setup", "--dry-run"}, e); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "is not on your PATH") {
		t.Errorf("want a PATH hint, got %q", out.String())
	}
	if os.Getenv("PATH") != before {
		t.Error("setup must never change the PATH")
	}
}

func TestSetupFailsWithoutJust(t *testing.T) {
	freshMachine(t)
	e, out, _ := testEnv(false)
	if code := run([]string{"setup"}, e); code != 1 || !strings.Contains(out.String(), "[fail] just:") {
		t.Errorf("code %d, out %q", code, out.String())
	}
}

func TestSetupJSON(t *testing.T) {
	freshMachine(t)
	e, out, _ := testEnv(true)
	if code := run([]string{"setup", "--json"}, e); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var got struct {
		OK       bool
		Created  bool
		ShimDir  string
		Findings []struct {
			Check string
			OK    bool
		}
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || !got.OK || !got.Created || got.ShimDir == "" || len(got.Findings) < 2 {
		t.Errorf("setup --json = %s (%v)", out.String(), err)
	}
}

func TestSetupUsageError(t *testing.T) {
	e, _, errOut := testEnv(true)
	if code := run([]string{"setup", "--wat"}, e); code != 2 || !strings.Contains(errOut.String(), `unexpected argument "--wat"`) {
		t.Errorf("code %d, stderr %q", code, errOut.String())
	}
}

func TestShimDoctorExitCodes(t *testing.T) {
	shimSetup(t)
	proj := project(t, "zzdoc")

	e, _, _ := testEnv(true)
	if code := run([]string{"shim", "add", proj}, e); code != 0 {
		t.Fatalf("add: %d", code)
	}

	e, out, _ := testEnv(true)
	if code := run([]string{"shim", "doctor"}, e); code != 0 || !strings.Contains(out.String(), "[ok] zzdoc: justfile") {
		t.Errorf("healthy: code %d, out %q", code, out.String())
	}

	// Move the project away: doctor must notice.
	if err := os.Remove(filepath.Join(proj, "justfile")); err != nil {
		t.Fatal(err)
	}
	e, out, _ = testEnv(true)
	if code := run([]string{"shim", "doctor"}, e); code != 1 || !strings.Contains(out.String(), "[fail] zzdoc: justfile") {
		t.Errorf("broken: code %d, out %q", code, out.String())
	}

	e, out, _ = testEnv(true)
	if code := run([]string{"shim", "doctor", "--json"}, e); code != 1 {
		t.Errorf("json: code %d", code)
	}
	var report struct {
		OK       bool
		Findings []struct{ Shim, Check string }
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil || report.OK || len(report.Findings) == 0 {
		t.Errorf("doctor --json = %s (%v)", out.String(), err)
	}

	e, _, errOut := testEnv(true)
	if code := run([]string{"shim", "doctor", "--wat"}, e); code != 2 || !strings.Contains(errOut.String(), "unexpected argument") {
		t.Errorf("bad flag: code %d, stderr %q", code, errOut.String())
	}
}

// shortcutMachine is a fresh machine whose PATH holds only the shim directory,
// so no jz of the machine running the tests is found.
func shortcutMachine(t *testing.T) (shimDir string) {
	t.Helper()
	shimDir = freshMachine(t)
	t.Setenv("PATH", shimDir)
	return shimDir
}

func TestSetupAddsTheJzShortcuts(t *testing.T) {
	shimDir := shortcutMachine(t)
	e, out, _ := testEnv(true)
	if code := run([]string{"setup"}, e); code != 0 {
		t.Fatalf("exit %d, out %q", code, out.String())
	}
	for _, n := range []string{"jz", "jzh"} {
		_, errPlain := os.Stat(filepath.Join(shimDir, n))
		_, errCmd := os.Stat(filepath.Join(shimDir, n+".cmd"))
		if errPlain != nil && errCmd != nil {
			t.Errorf("no %s shortcut in %s", n, shimDir)
		}
		if !strings.Contains(out.String(), "(the "+n+" shortcut)") {
			t.Errorf("output does not name the %s shortcut: %q", n, out.String())
		}
	}

	// Again: nothing new, and the JSON says so.
	e, out, _ = testEnv(true)
	if code := run([]string{"setup", "--json"}, e); code != 0 {
		t.Fatalf("exit %d, out %q", code, out.String())
	}
	var report struct {
		Shortcuts []struct{ Name, Action string }
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Shortcuts) != 2 || report.Shortcuts[0].Action != "exists" || report.Shortcuts[1].Action != "exists" {
		t.Errorf("shortcuts = %+v", report.Shortcuts)
	}
}

func TestSetupDryRunAddsNoShortcut(t *testing.T) {
	shimDir := shortcutMachine(t)
	e, out, _ := testEnv(true)
	if code := run([]string{"setup", "--dry-run"}, e); code != 0 || !strings.Contains(out.String(), "(the jz shortcut)") {
		t.Fatalf("code %d, out %q", code, out.String())
	}
	if _, err := os.Stat(shimDir); !os.IsNotExist(err) {
		t.Error("a dry run must not create the shim directory or a shortcut")
	}
}

func TestSetupWarnsAboutAnOldJustLikeDoctor(t *testing.T) {
	shortcutMachine(t)
	e, out, _ := testEnv(true)
	e.runCmd = func(name string, args ...string) (string, error) { return "just 1.30.0", nil }
	if code := run([]string{"setup"}, e); code != 0 || !strings.Contains(out.String(), "[warn] just:") {
		t.Errorf("code %d, out %q", code, out.String())
	}
}
