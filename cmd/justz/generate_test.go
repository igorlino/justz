package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// nativeDump is read once at load time: the tests change directory later.
var nativeDump = func() []byte {
	b, err := os.ReadFile("../../internal/generate/testdata/native.dump.json")
	if err != nil {
		panic(err)
	}
	return b
}()

// generateEnv fakes just (returning the real dump of the native fixture) and
// git (answering whether the target is tracked and whether it is ignored).
func generateEnv(t *testing.T, tracked, ignored bool) *runtimeEnv {
	t.Helper()
	dump := nativeDump
	r := newRuntimeEnv()
	r.runCmd = func(name string, args ...string) (string, error) {
		switch name {
		case "just":
			return string(dump), nil
		case "git":
			for _, a := range args {
				if a == "ls-files" {
					if tracked {
						return "", nil
					}
					return "", errors.New("not tracked")
				}
				if a == "check-ignore" {
					if ignored {
						return "", nil
					}
					return "", errors.New("not ignored")
				}
			}
		}
		return "", errors.New("unexpected command " + name)
	}
	return r
}

func gitProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "justfile"), "default:\n")
	return dir
}

func TestGenerateWritesTheOverlay(t *testing.T) {
	dir := gitProject(t)
	sub := filepath.Join(dir, "src")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub) // from a subdirectory: the file still goes in the project root

	r := generateEnv(t, false, false)
	if code := run([]string{"generate"}, r.env); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
	}
	target := filepath.Join(dir, ".just-local-rich.json")
	got := readFile(t, target)
	for _, want := range []string{`"Target environment"`, `"-w, --watch"`, `"<env>"`, `"example": "just test -w"`, `"tool"`} {
		if !strings.Contains(got, want) {
			t.Errorf("the overlay should contain %s:\n%s", want, got)
		}
	}
	if !strings.Contains(r.out.String(), "wrote "+target) || !strings.Contains(r.out.String(), "not ignored by git") {
		t.Errorf("stdout = %q", r.out.String())
	}

	// Already ignored: no hint.
	r = generateEnv(t, false, true)
	run([]string{"generate"}, r.env)
	if strings.Contains(r.out.String(), "not ignored") {
		t.Errorf("no hint expected when git ignores the file: %q", r.out.String())
	}
}

func TestGenerateStdoutWritesNothing(t *testing.T) {
	dir := gitProject(t)
	t.Chdir(dir)
	r := generateEnv(t, false, false)
	if code := run([]string{"generate", "--stdout"}, r.env); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(r.out.String(), `"groups"`) {
		t.Errorf("stdout = %q", r.out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".just-local-rich.json")); !os.IsNotExist(err) {
		t.Error("--stdout must not write the file")
	}
}

func TestGenerateRefusesATrackedFile(t *testing.T) {
	dir := gitProject(t)
	t.Chdir(dir)
	r := generateEnv(t, true, false)
	if code := run([]string{"generate"}, r.env); code != 1 || !strings.Contains(r.errOut.String(), "tracked by git") {
		t.Errorf("code %d, stderr %q", code, r.errOut.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".just-local-rich.json")); !os.IsNotExist(err) {
		t.Error("nothing may be written when the file is tracked")
	}
}

func TestGenerateKeepsHandAddedValues(t *testing.T) {
	dir := gitProject(t)
	t.Chdir(dir)
	target := filepath.Join(dir, ".just-local-rich.json")
	writeFile(t, target, `{"groups":{"core":{"commands":{"deploy":{"args":[{"name":"<env>","desc":"x","values":["dev","prod"]}]}}}}}`)

	r := generateEnv(t, false, true)
	if code := run([]string{"generate"}, r.env); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
	}
	got := readFile(t, target)
	if !strings.Contains(got, `"prod"`) || !strings.Contains(got, `"Target environment"`) {
		t.Errorf("values should be kept and descriptions refreshed:\n%s", got)
	}
}

func TestGenerateCustomPathAndErrors(t *testing.T) {
	dir := gitProject(t)
	t.Chdir(dir)
	custom := filepath.Join(t.TempDir(), "mine.json")
	r := generateEnv(t, false, true)
	if code := run([]string{"generate", custom}, r.env); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if _, err := os.Stat(custom); err != nil {
		t.Errorf("the custom path was not written: %v", err)
	}

	r = generateEnv(t, false, true)
	r.runCmd = func(name string, args ...string) (string, error) { return "", errors.New("just is missing") }
	if code := run([]string{"generate", "--stdout"}, r.env); code != 1 || !strings.Contains(r.errOut.String(), "could not read the justfile") {
		t.Errorf("just failing: code %d, stderr %q", code, r.errOut.String())
	}

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"generate", "--wat"}, "unknown flag"},
		{[]string{"generate", "a", "b"}, "only one path"},
	} {
		r := generateEnv(t, false, true)
		if code := run(tc.args, r.env); code != 2 || !strings.Contains(r.errOut.String(), tc.want) {
			t.Errorf("%v: code %d, stderr %q", tc.args, code, r.errOut.String())
		}
	}
}
