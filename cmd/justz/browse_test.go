package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// failingReader fails the test if anything tries to read from it: without a
// terminal, browse must never wait for input.
type failingReader struct{ t *testing.T }

func (f failingReader) Read([]byte) (int, error) {
	f.t.Error("browse must not read input without a terminal")
	return 0, errors.New("no input")
}

func browseProject(t *testing.T) {
	t.Helper()
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), zoomRegistry)
	t.Chdir(dir)
}

func TestBrowseWithoutATerminalPrintsTheOverview(t *testing.T) {
	browseProject(t)
	r := helpEnv(false, 100)
	r.env.interactive = false
	r.env.stdin = failingReader{t}
	r.env.runFzf = func(string, string, []string) (string, error) {
		t.Error("fzf must not run without a terminal")
		return "", nil
	}
	if code := run([]string{"help", "--browse"}, r.env); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
	}
	if !strings.Contains(r.out.String(), "dm  — groups") || !strings.Contains(r.out.String(), "sbx") {
		t.Errorf("want the overview:\n%s", r.out.String())
	}

	r = helpEnv(false, 100)
	r.env.interactive = false
	if code := run([]string{"help", "-b", "-j"}, r.env); code != 0 || !strings.Contains(r.out.String(), `"cli": "dm"`) {
		t.Errorf("json overview: code %d, out %q", code, r.out.String())
	}
}

func TestBrowseWithFzf(t *testing.T) {
	browseProject(t)
	var gotLines []string
	r := helpEnv(false, 100)
	r.env.interactive = true
	r.env.lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	r.env.runFzf = func(fzf, self string, lines []string) (string, error) {
		if fzf != "/usr/bin/fzf" {
			t.Errorf("fzf path = %q", fzf)
		}
		gotLines = lines
		return "tool git\tdm tool git — Git helper", nil
	}
	if code := run([]string{"help", "--browse"}, r.env); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
	}
	if !strings.Contains(r.out.String(), "dm tool git") || !strings.Contains(r.out.String(), "Git helper") {
		t.Errorf("should show the chosen command:\n%s", r.out.String())
	}
	if len(gotLines) < 5 || !strings.Contains(strings.Join(gotLines, "\n"), "features\tdm features — List features") {
		t.Errorf("lines for fzf = %q", gotLines)
	}
	for _, l := range gotLines {
		if strings.Count(l, "\t") != 1 {
			t.Errorf("each line needs exactly one tab (path, label): %q", l)
		}
	}

	// Picking a group, and the JSON form.
	r.out.Reset()
	r.env.runFzf = func(string, string, []string) (string, error) { return "tool\tdm tool — Tools", nil }
	run([]string{"help", "-b"}, r.env)
	if !strings.Contains(r.out.String(), "tool · Tools") {
		t.Errorf("a group should show its view:\n%s", r.out.String())
	}
	r.out.Reset()
	run([]string{"help", "-b", "--json"}, r.env)
	if !strings.Contains(r.out.String(), `"path": "tool"`) {
		t.Errorf("json of the chosen group: %q", r.out.String())
	}
}

func TestBrowseCancelAndFailure(t *testing.T) {
	browseProject(t)
	r := helpEnv(false, 100)
	r.env.interactive = true
	r.env.lookPath = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	r.env.runFzf = func(string, string, []string) (string, error) { return "", nil }
	if code := run([]string{"help", "-b"}, r.env); code != 0 || r.out.Len() != 0 {
		t.Errorf("cancel: code %d, out %q", code, r.out.String())
	}
	r.env.runFzf = func(string, string, []string) (string, error) { return "", errors.New("fzf crashed") }
	if code := run([]string{"help", "-b"}, r.env); code != 1 || !strings.Contains(r.errOut.String(), "fzf crashed") {
		t.Errorf("failure: code %d, stderr %q", code, r.errOut.String())
	}
}

func TestBrowseWithoutFzfUsesANumberedMenu(t *testing.T) {
	browseProject(t)
	menu := func(input string) *runtimeEnv {
		r := helpEnv(false, 100)
		r.env.interactive = true
		r.env.lookPath = func(file string) (string, error) { return "", errors.New("not found") }
		r.env.stdin = strings.NewReader(input)
		return r
	}

	// Rows are the groups first (ai, sbx, tool), then commands.
	r := menu("1\n")
	if code := run([]string{"help", "-b"}, r.env); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
	}
	if !strings.Contains(r.out.String(), "ai · AI agents") {
		t.Errorf("choice 1 should be the first group:\n%s", r.out.String())
	}
	if !strings.Contains(r.errOut.String(), "fzf was not found") || !strings.Contains(r.errOut.String(), "  1) dm ai") {
		t.Errorf("the menu and the fzf note go to stderr:\n%s", r.errOut.String())
	}

	for _, cancel := range []string{"q\n", "Q\n", "\n", ""} {
		r = menu(cancel)
		if code := run([]string{"help", "-b"}, r.env); code != 0 || r.out.Len() != 0 {
			t.Errorf("input %q should cancel: code %d, out %q", cancel, code, r.out.String())
		}
	}
	for _, bad := range []string{"0\n", "99\n", "abc\n"} {
		r = menu(bad)
		if code := run([]string{"help", "-b"}, r.env); code != 1 || !strings.Contains(r.errOut.String(), "is not a number between 1 and") {
			t.Errorf("input %q: code %d, stderr %q", bad, code, r.errOut.String())
		}
	}
}

func TestQuotedTwoWordTermsWork(t *testing.T) {
	browseProject(t)
	r := helpEnv(false, 100)
	if code := run([]string{"help", "tool git"}, r.env); code != 0 || !strings.Contains(r.out.String(), "dm tool git") {
		t.Errorf("fzf passes a quoted path as one word: code %d, out %q", code, r.out.String())
	}
}

func TestRunFzfPreviewCommandIsAFixedString(t *testing.T) {
	// The preview command must not contain the path of the binary or any
	// registry text: those travel in the environment and through fzf's quoting.
	src, err := os.ReadFile("browse.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "\"$JUSTZ_BIN\" help {1}") {
		t.Error("the preview command should be the fixed string \"$JUSTZ_BIN\" help {1}")
	}
}
