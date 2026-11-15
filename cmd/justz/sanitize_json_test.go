package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/igorlino/justz/internal/shim"
)

// Go's JSON encoder escapes ESC and BEL but leaves DEL, C1 and bidi characters
// raw, so assertCLISafe must see none of them in the encoded output.
const rawOnly = "a\u007fb\u009bc‮d"

func TestPrintJSONCleansWhatTheEncoderLeavesRaw(t *testing.T) {
	e, out, _ := testEnv(true)
	if code := printJSON(e, map[string]string{"name": rawOnly}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	assertCLISafe(t, "printJSON", out.String())
	var back map[string]string
	if err := json.Unmarshal(out.Bytes(), &back); err != nil {
		t.Fatalf("the output is not JSON: %v\n%s", err, out.String())
	}
	if back["name"] != "a�b�c�d" {
		t.Errorf("name = %q", back["name"])
	}
}

func TestDoctorJSONIsClean(t *testing.T) {
	e, out, _ := testEnv(true)
	e.lookPath = func(file string) (string, error) { return "/x" + rawOnly + "/" + file, nil }
	run([]string{"doctor", "--json"}, e)
	assertCLISafe(t, "doctor --json", out.String())
	if !json.Valid(out.Bytes()) {
		t.Errorf("not valid JSON: %s", out.String())
	}
}

func TestShimListJSONIsClean(t *testing.T) {
	shimSetup(t)
	d, err := shim.DefaultDirs()
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(d.Config, "shims.json"),
		`{"shims":[{"name":"ab\u007fc\u009b‮x","justfile":"/p/justfile"}]}`)
	e, out, errOut := testEnv(true)
	if code := run([]string{"shim", "list", "--json"}, e); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	assertCLISafe(t, "shim list --json", out.String())
}

func TestGenerateWritesNoRawControlCharacters(t *testing.T) {
	dir := gitProject(t)
	t.Chdir(dir)
	r := generateEnv(t, false, false)
	dirty := strings.ReplaceAll(string(nativeDump), "Compile the project", `Compile\u007f\u009b‮ the project`)
	base := r.runCmd
	r.runCmd = func(name string, args ...string) (string, error) {
		if name == "just" {
			return dirty, nil
		}
		return base(name, args...)
	}
	if code := run([]string{"generate", "--stdout"}, r.env); code != 0 {
		t.Fatalf("exit %d: %s", code, r.errOut.String())
	}
	assertCLISafe(t, "generate --stdout", r.out.String())
	if !strings.Contains(r.out.String(), "Compile�") {
		t.Errorf("the doc should be present with U+FFFD:\n%s", r.out.String())
	}

	r2 := generateEnv(t, false, false)
	r2.runCmd = func(name string, args ...string) (string, error) {
		if name == "just" {
			return dirty, nil
		}
		return base(name, args...)
	}
	if code := run([]string{"generate"}, r2.env); code != 0 {
		t.Fatalf("exit %d: %s", code, r2.errOut.String())
	}
	b, err := os.ReadFile(filepath.Join(dir, ".just-local-rich.json"))
	if err != nil {
		t.Fatal(err)
	}
	assertCLISafe(t, "the overlay file", string(b))
}

func TestPathHintIsNotInjectable(t *testing.T) {
	for _, goos := range []string{"linux", "windows"} {
		for _, dir := range []string{"/h/it's;$(id)", `/h/a"b`, "/h/a`b", `/h/a\b`, "/h/a\x1b[31mb"} {
			h := pathHint(shim.Dirs{GOOS: goos, Shim: dir})
			if strings.Contains(h, "echo ") || strings.Contains(h, "SetEnvironmentVariable") {
				t.Errorf("%s %q: a runnable command was offered: %s", goos, dir, h)
			}
			assertCLISafe(t, "pathHint", h)
		}
	}
	// A plain directory, with a space, still gets the command.
	if h := pathHint(shim.Dirs{GOOS: "linux", Shim: "/home/me/my bin"}); !strings.Contains(h, `export PATH="/home/me/my bin:$PATH"`) {
		t.Errorf("hint = %s", h)
	}
}

func TestCompletionNamesSkipsAnInvalidShimName(t *testing.T) {
	shimSetup(t)
	d, err := shim.DefaultDirs()
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(d.Config, "shims.json"),
		`{"shims":[{"name":"good","justfile":"/p/justfile","suffix":"h"},{"name":"bad;id","justfile":"/p/justfile"},{"name":"ok","justfile":"/p/justfile","suffix":"‮"}]}`)
	got := strings.Join(completionNames(), " ")
	if !strings.Contains(got, "good") || !strings.Contains(got, "goodh") {
		t.Errorf("names = %q, want good and goodh", got)
	}
	if strings.ContainsAny(got, ";‮") || strings.Contains(got, "ok") {
		t.Errorf("an invalid name got into the script names: %q", got)
	}
}
