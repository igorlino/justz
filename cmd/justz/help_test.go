package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// demoHelpRegistry is read once at load time: the tests change directory later.
var demoHelpRegistry = func() string {
	b, err := os.ReadFile("../../testdata/help-reference/demo/config/commands.json")
	if err != nil {
		panic(err)
	}
	return string(b)
}()

func helpProject(t *testing.T) string {
	t.Helper()
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), demoHelpRegistry)
	return dir
}

func helpEnv(color bool, width int) *runtimeEnv {
	r := newRuntimeEnv()
	r.env.color, r.env.width = color, width
	return r
}

func TestHelpSummaryAndExpanded(t *testing.T) {
	dir := helpProject(t)
	sub := filepath.Join(dir, "src")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub) // found by walking up

	r := helpEnv(false, 100)
	if code := run([]string{"help"}, r.env); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, r.errOut.String())
	}
	out := r.out.String()
	for _, want := range []string{"  demo\n", "  Core\n", "build               # Compile the project", "Commands  (run without args to see sub-commands)", "justz help --expanded"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary should contain %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "build docker") {
		t.Errorf("the summary must not list group commands:\n%s", out)
	}

	r = helpEnv(false, 100)
	if code := run([]string{"help", "--expanded"}, r.env); code != 0 {
		t.Fatalf("expanded: exit %d", code)
	}
	for _, want := range []string{"build  — Build", "build docker                 # Build the container image", "-k <pattern>"} {
		if !strings.Contains(r.out.String(), want) {
			t.Errorf("expanded should contain %q:\n%s", want, r.out.String())
		}
	}
}

func TestHelpColorFollowsTheEnvironment(t *testing.T) {
	t.Chdir(helpProject(t))
	ansi := regexp.MustCompile(`\x1b\[`)

	r := helpEnv(false, 80)
	run([]string{"help"}, r.env)
	if ansi.MatchString(r.out.String()) {
		t.Error("no colors when the output is not a terminal")
	}
	r = helpEnv(true, 80)
	run([]string{"help"}, r.env)
	if !ansi.MatchString(r.out.String()) {
		t.Error("colors expected on a terminal")
	}
}

func TestHelpWideFlag(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), `{"groups":{"core":{"commands":{"test":{"desc":"Run tests","example":"just test -w"}}}}}`)
	t.Chdir(dir)
	r := helpEnv(false, 80)
	run([]string{"help", "--expanded"}, r.env)
	if strings.Contains(r.out.String(), "just test -w") {
		t.Error("no example column at 80 columns")
	}
	r = helpEnv(false, 80)
	run([]string{"help", "--expanded", "-w"}, r.env)
	if !strings.Contains(r.out.String(), "just test -w") {
		t.Errorf("-w should add the example column:\n%s", r.out.String())
	}
}

func TestHelpUsesTheLocalOverlay(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, ".just-local-rich.json"), `{"groups":{"core":{"commands":{"build":{"desc":"From the overlay"}}}}}`)
	t.Chdir(dir)
	r := helpEnv(false, 80)
	if code := run([]string{"help"}, r.env); code != 0 || !strings.Contains(r.out.String(), "From the overlay") {
		t.Errorf("code %d, out %q, err %q", code, r.out.String(), r.errOut.String())
	}
}

func TestHelpErrors(t *testing.T) {
	t.Chdir(gitProject(t)) // no registry, no overlay
	r := helpEnv(false, 80)
	if code := run([]string{"help"}, r.env); code != 1 || !strings.Contains(r.errOut.String(), "justz generate") {
		t.Errorf("no registry: code %d, stderr %q", code, r.errOut.String())
	}

	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), `{not json`)
	t.Chdir(dir)
	r = helpEnv(false, 80)
	if code := run([]string{"help"}, r.env); code != 1 || !strings.Contains(r.errOut.String(), "not valid JSON") {
		t.Errorf("invalid registry: code %d, stderr %q", code, r.errOut.String())
	}

	r = helpEnv(false, 80)
	if code := run([]string{"help", "--wat"}, r.env); code != 2 || !strings.Contains(r.errOut.String(), `unknown argument "--wat"`) {
		t.Errorf("bad flag: code %d, stderr %q", code, r.errOut.String())
	}
	r = helpEnv(false, 80)
	if code := run([]string{"help", "--help"}, r.env); code != 0 || !strings.Contains(r.out.String(), "Usage: justz help") {
		t.Errorf("--help: code %d", code)
	}
}

func TestTerminalWidth(t *testing.T) {
	t.Setenv("COLUMNS", "120")
	if got := terminalWidth(os.Stdout); got != 120 {
		t.Errorf("COLUMNS=120 gave %d", got)
	}
	for _, bad := range []string{"", "abc", "0", "-5"} {
		t.Setenv("COLUMNS", bad)
		if got := terminalWidth(os.Stdin); got <= 0 {
			t.Errorf("COLUMNS=%q gave %d, want a positive fallback", bad, got)
		}
	}
}

func TestNoColorWins(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if useColor(os.Stdout) {
		t.Error("NO_COLOR must turn colors off")
	}
}
