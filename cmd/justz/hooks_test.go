package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func repoFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// yamlScalars reads the "key: value" lines of a YAML mapping that has only
// scalar values, as the hook definition does. A comment line, a blank line and
// the "- " that starts the list item are skipped; a tab is an error in YAML.
func yamlScalars(t *testing.T, src string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, line := range strings.Split(src, "\n") {
		if strings.Contains(line, "\t") {
			t.Errorf("a tab in YAML: %q", line)
		}
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		trim = strings.TrimPrefix(trim, "- ")
		key, val, ok := strings.Cut(trim, ": ")
		if !ok {
			t.Errorf("not a key: value line: %q", line)
			continue
		}
		val = strings.TrimSpace(val)
		if len(val) >= 2 && val[0] == '\'' && val[len(val)-1] == '\'' {
			val = val[1 : len(val)-1] // a single-quoted scalar
		}
		out[key] = val
	}
	return out
}

func TestPreCommitHookDefinition(t *testing.T) {
	h := yamlScalars(t, repoFile(t, ".pre-commit-hooks.yaml"))
	for _, key := range []string{"id", "name", "entry", "language", "pass_filenames", "files"} {
		if h[key] == "" {
			t.Errorf("the hook has no %s", key)
		}
	}
	if h["id"] != "justz-check" || h["entry"] != "justz check" || h["language"] != "system" || h["pass_filenames"] != "false" {
		t.Errorf("hook = %v", h)
	}
	re, err := regexp.Compile(h["files"])
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"justfile", "Justfile", ".justfile", "JUSTFILE", "sub/justfile", "foo.just", "just/tool.just", "config/commands.json", ".just-local-rich.json"} {
		if !re.MatchString(f) {
			t.Errorf("the hook should run for %s", f)
		}
	}
	for _, f := range []string{"README.md", "main.go", "notjustfile.txt"} {
		if re.MatchString(f) {
			t.Errorf("the hook should not run for %s", f)
		}
	}
	if strings.Contains(repoFile(t, ".pre-commit-hooks.yaml"), "v0.7.0") {
		t.Errorf("the header must not pin a release that predates the hook")
	}
}

// The yaml blocks of docs/hooks-and-ci.md are snippets to copy. Without a YAML
// library the test checks what a copy needs: no tabs, the keys, the pinned
// install and the check.
func TestHooksAndCISnippets(t *testing.T) {
	doc := repoFile(t, "docs/hooks-and-ci.md")
	blocks := regexp.MustCompile("(?s)```yaml\n(.*?)```").FindAllStringSubmatch(doc, -1)
	if len(blocks) != 2 {
		t.Fatalf("yaml blocks = %d, want 2 (pre-commit config, workflow)", len(blocks))
	}
	for _, b := range blocks {
		if strings.Contains(b[1], "\t") {
			t.Errorf("a tab in a yaml snippet:\n%s", b[1])
		}
	}
	for _, want := range []string{"repos:", "repo: https://github.com/igorlino/justz", "rev: vX.Y.Z", "- id: justz-check"} {
		if !strings.Contains(blocks[0][1], want) {
			t.Errorf("pre-commit snippet lacks %q", want)
		}
	}
	for _, want := range []string{"jobs:", "runs-on:", "JUSTZ_VERSION: v", "install.sh", "--version", "setup-just", "run: justz check", "GITHUB_PATH"} {
		if !strings.Contains(blocks[1][1], want) {
			t.Errorf("workflow snippet lacks %q", want)
		}
	}
	if strings.Contains(doc, "justz/v0.7.0/scripts") || strings.Contains(blocks[0][1], "v0.7.0") {
		t.Errorf("a hook snippet pins a release that predates the hook")
	}
}
