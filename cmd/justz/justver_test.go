package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVersionBelow(t *testing.T) {
	for _, tc := range []struct {
		have string
		want bool
	}{
		{"just 1.21.0", true},
		{"just 1.36.2", true},
		{"just 0.9.9", true},
		{"just " + minJust, false},
		{"just 1.37.1", false},
		{"just 1.59.0", false},
		{"just 2.0.0", false},
		{"just 1.40.0-rc1", false},
		{"", false},
		{"nonsense", false},
	} {
		if got := versionBelow(tc.have, minJust); got != tc.want {
			t.Errorf("versionBelow(%q) = %v, want %v", tc.have, got, tc.want)
		}
	}
}

func doctorWithJust(t *testing.T, version string) (string, int) {
	t.Helper()
	freshMachine(t)
	e, out, _ := testEnv(true)
	e.runCmd = func(name string, args ...string) (string, error) { return version, nil }
	code := runDoctor(nil, e)
	return out.String(), code
}

func TestDoctorJustVersion(t *testing.T) {
	// Aliases are checked too, so look only at the just line.
	line := func(out string) string {
		for _, l := range strings.Split(out, "\n") {
			if strings.Contains(l, "just: ") {
				return l
			}
		}
		return ""
	}
	below, code := doctorWithJust(t, "just 1.21.0")
	l := line(below)
	if !strings.HasPrefix(l, "[warn] just:") || !strings.Contains(l, minJust) || !strings.Contains(l, "brew install just") {
		t.Errorf("below the minimum should warn with how to upgrade: %q", l)
	}
	if code != 0 {
		t.Errorf("a warning must not fail doctor, exit %d", code)
	}
	for _, v := range []string{"just " + minJust, "just 1.59.0"} {
		out, _ := doctorWithJust(t, v)
		if l := line(out); !strings.HasPrefix(l, "[ok] just:") {
			t.Errorf("%s should be ok: %q", v, l)
		}
	}
}

func TestJustReadErrorPassesStderrThrough(t *testing.T) {
	stderr := "error: Unknown attribute `doc`\n ——▶ justfile:3:2\n"
	e, _, _ := testEnv(true)
	e.runCmd = func(name string, args ...string) (string, error) {
		return "just 1.21.0", nil
	}
	err := justReadError(e, &cmdError{err: errors.New("exit status 1"), stderr: stderr})
	for _, want := range []string{"could not read the justfile with just: exit status 1", "Unknown attribute `doc`", "an old just is a likely cause", "just 1.21.0", minJust} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should contain %q:\n%s", want, err)
		}
	}
}

func TestJustReadErrorNoHintForOtherErrors(t *testing.T) {
	e, _, _ := testEnv(true) // just 1.59.0
	err := justReadError(e, &cmdError{err: errors.New("exit status 1"), stderr: "error: Recipe `a` not found\n"})
	if !strings.Contains(err.Error(), "Recipe `a` not found") || strings.Contains(err.Error(), "old just") {
		t.Errorf("a plain error on a new just should have no hint:\n%s", err)
	}
	if got := justReadError(e, errors.New("exit status 1")).Error(); got != "could not read the justfile with just: exit status 1" {
		t.Errorf("no stderr, no extra text: %q", got)
	}
}

func TestJustReadErrorIsBoundedAndSafe(t *testing.T) {
	var b strings.Builder
	b.WriteString("error: bad \x1b]0;pwned\x07\x1b[31mred\n")
	for i := 0; i < 50; i++ {
		b.WriteString("line " + strings.Repeat("x", 100) + "\n")
	}
	e, _, _ := testEnv(true)
	err := justReadError(e, &cmdError{err: errors.New("exit status 1"), stderr: b.String()}).Error()
	if strings.ContainsRune(err, 0x1b) || strings.ContainsRune(err, 0x07) {
		t.Errorf("control characters must not reach the terminal: %q", err)
	}
	if n := strings.Count(err, "\n"); n > stderrLines+2 {
		t.Errorf("%d lines, want at most %d of stderr", n, stderrLines)
	}
	if len(err) > stderrBytes+400 {
		t.Errorf("error is %d bytes, want it bounded", len(err))
	}
}

// A fake just that prints an error and exits 1: the stderr must reach the
// message of `justz check`, through the real command runner.
func TestCheckShowsJustStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script")
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'just 1.21.0'; exit 0; fi\necho 'error: Unknown attribute `doc`' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "just"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Chdir(gitProject(t))
	r := newRuntimeEnv()
	r.runCmd = runOutput
	if code := run([]string{"check"}, r.env); code == 0 {
		t.Fatal("check should fail")
	}
	got := r.errOut.String()
	for _, want := range []string{"Unknown attribute `doc`", "an old just", "just 1.21.0"} {
		if !strings.Contains(got, want) {
			t.Errorf("stderr should contain %q:\n%s", want, got)
		}
	}
}

func hintEnv(version string, verr error) env {
	e, _, _ := testEnv(true)
	e.runCmd = func(name string, args ...string) (string, error) { return version, verr }
	return e
}

func TestJustReadErrorHintFollowsTheVersion(t *testing.T) {
	failure := &cmdError{err: errors.New("exit status 1"), stderr: "error: Unknown attribute `foo`\n"}
	for _, tc := range []struct {
		name string
		e    env
		hint bool
	}{
		{"current just: a typo, no hint", hintEnv("just 1.59.0", nil), false},
		{"old just: hint", hintEnv("just 1.21.0", nil), true},
		{"unreadable version: the name decides", hintEnv("", errors.New("no just")), true},
	} {
		got := justReadError(tc.e, failure).Error()
		if !strings.Contains(got, "Unknown attribute `foo`") {
			t.Errorf("%s: stderr missing:\n%s", tc.name, got)
		}
		if has := strings.Contains(got, "old just"); has != tc.hint {
			t.Errorf("%s: hint = %v, want %v:\n%s", tc.name, has, tc.hint, got)
		}
	}
}

// failingJust puts a fake just on the PATH that prints an error and exits 1,
// and moves into a project.
func failingJust(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script")
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'just 1.21.0'; exit 0; fi\necho 'error: Unknown attribute `doc`' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "just"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Chdir(gitProject(t))
}

func TestEveryReaderShowsJustStderr(t *testing.T) {
	for _, cmd := range []string{"help", "generate", "lint", "check"} {
		t.Run(cmd, func(t *testing.T) {
			failingJust(t)
			r := newRuntimeEnv()
			r.runCmd = runOutput
			if code := run([]string{cmd}, r.env); code == 0 {
				t.Fatalf("%s should fail", cmd)
			}
			if got := r.errOut.String(); !strings.Contains(got, "Unknown attribute `doc`") {
				t.Errorf("%s stderr should show just's reason:\n%s", cmd, got)
			}
		})
	}
}
