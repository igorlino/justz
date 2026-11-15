package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

const dirtyCLIRegistry = `{
  "cli": "dm",
  "groups": {
    "tool": { "label": "Tools\u001b]0;t\u0007", "commands": {
      "git": {"desc": "Git\u001b[31m helper\u009b‮", "detail": "Walks\u007f⁦ the PATH.",
              "args": [{"name": "-v, --verbose", "desc": "Be\u0000 loud"}]} } }
  }
}`

func assertCLISafe(t *testing.T, what, s string) {
	t.Helper()
	for _, r := range s {
		switch {
		case r == '\n':
		case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f, r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069:
			t.Errorf("%s: U+%04X in %q", what, r, s)
			return
		}
	}
}

func TestHelpOutputOfADirtyProjectIsSafe(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), dirtyCLIRegistry)
	t.Chdir(dir)

	for _, args := range [][]string{{"help"}, {"help", "--expanded"}, {"help", "tool"}, {"help", "tool", "git"}, {"help", "-j", "tool", "git"}, {"help", "-j", "tool"}, {"help", "-j"}} {
		r := helpEnv(false, 100)
		if code := run(args, r.env); code != 0 {
			t.Fatalf("%v: exit %d, stderr %q", args, code, r.errOut.String())
		}
		assertCLISafe(t, strings.Join(args, " ")+" (stdout)", r.out.String())
		isJSON := false
		for _, a := range args {
			isJSON = isJSON || a == "-j"
		}
		if isJSON {
			var v any
			if err := json.Unmarshal(r.out.Bytes(), &v); err != nil {
				t.Fatalf("%v: not JSON: %v", args, err)
			}
			assertCLISafe(t, strings.Join(args, " ")+" (decoded)", fmtAll(v))
		}
	}
}

// fmtAll joins every string of a decoded JSON value.
func fmtAll(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		var s string
		for _, e := range x {
			s += fmtAll(e)
		}
		return s
	case map[string]any:
		var s string
		for k, e := range x {
			s += k + fmtAll(e)
		}
		return s
	}
	return ""
}

func TestHelpErrorsDoNotEchoControlCharacters(t *testing.T) {
	dir := gitProject(t)
	writeFile(t, filepath.Join(dir, "config", "commands.json"), dirtyCLIRegistry)
	t.Chdir(dir)

	for _, args := range [][]string{{"help", "zz\x1b]0;t\x07\u009b"}, {"help", "-j", "zz\x1b[31m‮"}} {
		r := helpEnv(false, 100)
		if code := run(args, r.env); code != 2 {
			t.Fatalf("%q: exit %d", args, code)
		}
		assertCLISafe(t, "stderr of "+strings.Join(args, " "), r.errOut.String())
		if !strings.Contains(r.errOut.String(), "no group or command matches 'zz") {
			t.Errorf("%q: stderr = %q", args, r.errOut.String())
		}
	}
}
