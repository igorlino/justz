package scaffold

import (
	"strings"
	"testing"

	"github.com/igorlino/justz/internal/orderedjson"
	"github.com/igorlino/justz/internal/registry"
)

func paths(fs []File) []string {
	var p []string
	for _, f := range fs {
		p = append(p, f.Path)
	}
	return p
}

func find(t *testing.T, fs []File, p string) string {
	t.Helper()
	for _, f := range fs {
		if f.Path == p {
			return f.Content
		}
	}
	t.Fatalf("no file %s in %v", p, paths(fs))
	return ""
}

func TestBuildNativeLayout(t *testing.T) {
	fs, err := Build(Options{Name: "demo", Variant: Native, Modules: []string{"build", "ops"}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(paths(fs), ","), "justfile,just/preamble.just,just/build.just,just/ops.just"; got != want {
		t.Fatalf("files = %s, want %s", got, want)
	}
	root := find(t, fs, "justfile")
	for _, want := range []string{"import 'just/preamble.just'", "mod build 'just/build.just'", "[group('commands')]", "@just --unsorted --list --list-heading $'\\n  demo\\n\\n'"} {
		if !strings.Contains(root, want) {
			t.Errorf("justfile misses %q:\n%s", want, root)
		}
	}
	if strings.Contains(root, "justz") {
		t.Errorf("native justfile must not need justz:\n%s", root)
	}
	if m := find(t, fs, "just/build.just"); !strings.Contains(m, "import 'preamble.just'") || !strings.Contains(m, "@just --unsorted --list build") {
		t.Errorf("module:\n%s", m)
	}
}

func TestBuildRichHasRegistryInCanonicalOrder(t *testing.T) {
	fs, err := Build(Options{Name: "demo", Variant: Rich, Modules: []string{"zeta", "alpha"}})
	if err != nil {
		t.Fatal(err)
	}
	reg := find(t, fs, "config/commands.json")
	doc, err := orderedjson.Parse([]byte(reg))
	if err != nil {
		t.Fatal(err)
	}
	registry.Canonicalize(doc)
	again, _ := orderedjson.Marshal(doc)
	if string(again) != reg {
		t.Errorf("registry is not canonical:\n%s\nvs\n%s", reg, again)
	}
	for _, want := range []string{`"name": "demo"`, `"alpha"`, `"zeta"`} {
		if !strings.Contains(reg, want) {
			t.Errorf("registry misses %s:\n%s", want, reg)
		}
	}
	root := find(t, fs, "justfile")
	if !strings.Contains(root, "@justz help\n") || !strings.Contains(root, "exec justz help \"$@\"") || !strings.Contains(root, "@justz fmt --check") {
		t.Errorf("rich justfile:\n%s", root)
	}
	// The recipe docs must equal the registry descriptions, or check fails.
	if !strings.Contains(root, "[doc('"+richHelpDesc+"')]") || !strings.Contains(root, "[doc('"+richCheckDesc+"')]") {
		t.Errorf("[doc] and desc differ")
	}
	if m := find(t, fs, "just/alpha.just"); !strings.Contains(m, "@justz help alpha") {
		t.Errorf("module:\n%s", m)
	}
}

func TestBuildWithoutModules(t *testing.T) {
	fs, err := Build(Options{Name: "demo", Variant: Native})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(paths(fs), ","); got != "justfile,just/preamble.just" {
		t.Errorf("files = %s", got)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		o    Options
		want string
	}{
		{Options{Name: "it's", Variant: Native}, "project name"},
		{Options{Name: "", Variant: Native}, "project name"},
		{Options{Name: "x", Variant: "fancy"}, "variant"},
		{Options{Name: "x", Variant: Native, Modules: []string{"Bad"}}, "module"},
		{Options{Name: "x", Variant: Native, Modules: []string{"../x"}}, "module"},
		{Options{Name: "x", Variant: Native, Modules: []string{"core"}}, "reserved"},
		{Options{Name: "x", Variant: Native, Modules: []string{"preamble"}}, "reserved"},
		{Options{Name: "x", Variant: Native, Modules: []string{"a", "a"}}, "twice"},
	}
	for _, tt := range tests {
		_, err := Build(tt.o)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("Build(%+v) error = %v, want %q", tt.o, err, tt.want)
		}
	}
}
