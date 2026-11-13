package shim

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type fixture struct {
	t    *testing.T
	d    Dirs
	root string
}

// newFixture builds Dirs inside a temporary directory. The shim directory is
// on the PATH, after any directory in before and before any in after.
func newFixture(t *testing.T, before, after []string) *fixture {
	t.Helper()
	root := t.TempDir()
	d := Dirs{
		Config: filepath.Join(root, "config"),
		Shim:   filepath.Join(root, "shims"),
		GOOS:   runtime.GOOS,
	}
	if err := os.MkdirAll(d.Shim, 0o755); err != nil {
		t.Fatal(err)
	}
	d.Path = append(append(append([]string{}, before...), d.Shim), after...)
	return &fixture{t: t, d: d, root: root}
}

func (f *fixture) dir(name string) string {
	p := filepath.Join(f.root, name)
	if err := os.MkdirAll(p, 0o755); err != nil {
		f.t.Fatal(err)
	}
	return p
}

// command puts an executable called name in dir.
func (f *fixture) command(dir, name string) {
	f.t.Helper()
	file := name
	if runtime.GOOS == "windows" {
		file += ".exe"
	}
	if err := os.WriteFile(filepath.Join(dir, file), []byte("#!/bin/sh\n"), 0o755); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) justfile(name string) string {
	f.t.Helper()
	p := filepath.Join(f.dir(name), "justfile")
	if err := os.WriteFile(p, []byte("default:\n"), 0o644); err != nil {
		f.t.Fatal(err)
	}
	return p
}

func opts(name string) AddOptions { return AddOptions{Name: name, Suffix: "h"} }

func kinds(err error) []Kind {
	var ce *CollisionError
	if !errors.As(err, &ce) {
		return nil
	}
	var out []Kind
	for _, p := range ce.Problems {
		out = append(out, p.Kind)
	}
	return out
}

func TestAddWritesStubsAndRegistry(t *testing.T) {
	f := newFixture(t, nil, nil)
	jf := f.justfile("proj")

	res, err := f.d.Add(jf, opts("tz"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated || res.DryRun || len(res.Files) != 2 {
		t.Errorf("unexpected result: %+v", res)
	}
	for _, n := range []string{"tz", "tzh"} {
		sp := f.d.stubPath(n)
		data, err := os.ReadFile(sp)
		if err != nil {
			t.Fatalf("stub %s missing: %v", n, err)
		}
		if !strings.Contains(string(data), marker) || !strings.Contains(string(data), "justz --as "+n) {
			t.Errorf("stub %s has unexpected content:\n%s", n, data)
		}
		if runtime.GOOS != "windows" {
			if info, _ := os.Stat(sp); info.Mode().Perm()&0o111 == 0 {
				t.Errorf("stub %s is not executable", n)
			}
		}
	}
	reg, err := f.d.Load()
	if err != nil || len(reg.Shims) != 1 || reg.Shims[0].Name != "tz" || reg.Shims[0].Justfile != jf || reg.Shims[0].Suffix != "h" {
		t.Errorf("registry = %+v, %v", reg, err)
	}
}

func TestAddAgainForTheSameJustfileIsAnUpdate(t *testing.T) {
	f := newFixture(t, nil, nil)
	jf := f.justfile("proj")
	if _, err := f.d.Add(jf, opts("tz")); err != nil {
		t.Fatal(err)
	}
	o := opts("tz")
	o.Soft = true
	res, err := f.d.Add(jf, o)
	if err != nil {
		t.Fatalf("re-adding the same shim should work: %v", err)
	}
	if !res.Updated {
		t.Error("the result should say it was an update")
	}
	reg, _ := f.d.Load()
	if len(reg.Shims) != 1 || !reg.Shims[0].Soft {
		t.Errorf("registry = %+v, want one soft shim", reg)
	}
}

func TestAddNeedsAnExistingJustfile(t *testing.T) {
	f := newFixture(t, nil, nil)
	if _, err := f.d.Add(filepath.Join(f.root, "nope", "justfile"), opts("tz")); err == nil {
		t.Error("a missing justfile should be an error")
	}
	if _, err := f.d.Add(f.dir("adir"), opts("tz")); err == nil {
		t.Error("a directory is not a justfile")
	}
}

func TestInvalidAndReservedNames(t *testing.T) {
	f := newFixture(t, nil, nil)
	jf := f.justfile("proj")
	for _, name := range []string{"just", "justz", "git", "cd", "TZ", "9x", "a b", "", strings.Repeat("a", 33)} {
		_, err := f.d.Add(jf, opts(name))
		if got := kinds(err); len(got) == 0 || got[0] != Invalid {
			t.Errorf("name %q: want an invalid-name problem, got %v (%v)", name, got, err)
		}
	}
	o := opts("tz")
	o.Suffix = ""
	if _, err := f.d.Add(jf, o); len(kinds(err)) == 0 {
		t.Error("an empty help suffix should be refused")
	}
	// Invalid names are never overridable.
	o = opts("git")
	o.Force, o.Replace = true, true
	if _, err := f.d.Add(jf, o); len(kinds(err)) == 0 {
		t.Error("--force and --replace must not allow a reserved name")
	}
}

func TestNeverOverwritesAFileThatIsNotOurs(t *testing.T) {
	f := newFixture(t, nil, nil)
	jf := f.justfile("proj")
	sp := f.d.stubPath("tz")
	if err := os.WriteFile(sp, []byte("#!/bin/sh\necho mine\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	o := opts("tz")
	o.Force, o.Replace = true, true
	_, err := f.d.Add(jf, o)
	if got := kinds(err); len(got) == 0 || got[0] != NotOurs {
		t.Fatalf("want a not-ours problem, got %v (%v)", got, err)
	}
	if b, _ := os.ReadFile(sp); !strings.Contains(string(b), "echo mine") {
		t.Error("the user's file was overwritten")
	}
	if reg, _ := f.d.Load(); len(reg.Shims) != 0 {
		t.Error("nothing should be registered after a refused add")
	}
}

func TestTwoRepositoriesWantTheSameName(t *testing.T) {
	f := newFixture(t, nil, nil)
	a, b := f.justfile("a"), f.justfile("b")
	if _, err := f.d.Add(a, opts("tz")); err != nil {
		t.Fatal(err)
	}
	_, err := f.d.Add(b, opts("tz"))
	got := kinds(err)
	if len(got) != 1 || got[0] != Owned {
		t.Fatalf("want one owned problem, got %v (%v)", got, err)
	}
	if !strings.Contains(err.Error(), a) {
		t.Errorf("the message should name the first owner %s: %v", a, err)
	}

	o := opts("tz")
	o.Replace = true
	if _, err := f.d.Add(b, o); err != nil {
		t.Fatalf("--replace should re-point the shim: %v", err)
	}
	reg, _ := f.d.Load()
	if len(reg.Shims) != 1 || reg.Shims[0].Justfile != b {
		t.Errorf("registry = %+v, want one shim pointing at %s", reg, b)
	}
}

func TestHelpNameOfAnotherShimIsTaken(t *testing.T) {
	f := newFixture(t, nil, nil)
	a, b := f.justfile("a"), f.justfile("b")
	if _, err := f.d.Add(a, opts("tz")); err != nil { // owns tz and tzh
		t.Fatal(err)
	}
	_, err := f.d.Add(b, opts("tzh"))
	if got := kinds(err); len(got) == 0 || got[0] != Owned {
		t.Errorf("a name equal to another shim's help name should be owned, got %v (%v)", got, err)
	}
}

func TestShadowedByEarlierCommand(t *testing.T) {
	f := newFixture(t, nil, nil)
	early := f.dir("early")
	f.d.Path = []string{early, f.d.Shim}
	f.command(early, "tz")
	jf := f.justfile("proj")

	_, err := f.d.Add(jf, opts("tz"))
	if got := kinds(err); len(got) != 1 || got[0] != Shadowed {
		t.Fatalf("want one shadowed problem, got %v (%v)", got, err)
	}
	o := opts("tz")
	o.Replace = true
	if _, err := f.d.Add(jf, o); err == nil {
		t.Error("--replace alone must not allow a shadowed shim")
	}
	o = opts("tz")
	o.Force = true
	if _, err := f.d.Add(jf, o); err != nil {
		t.Errorf("--force should allow a shadowed shim: %v", err)
	}
}

func TestHidesALaterCommand(t *testing.T) {
	f := newFixture(t, nil, nil)
	late := f.dir("late")
	f.d.Path = []string{f.d.Shim, late}
	f.command(late, "tzh")
	jf := f.justfile("proj")

	_, err := f.d.Add(jf, opts("tz"))
	if got := kinds(err); len(got) != 1 || got[0] != Hides {
		t.Fatalf("want one hides problem (for tzh), got %v (%v)", got, err)
	}
	o := opts("tz")
	o.Force = true
	if _, err := f.d.Add(jf, o); err == nil {
		t.Error("--force alone must not allow hiding an existing command")
	}
	o = opts("tz")
	o.Replace = true
	if _, err := f.d.Add(jf, o); err != nil {
		t.Errorf("--replace should allow hiding: %v", err)
	}
}

func TestShimDirNotOnPathIsAWarningNotACollision(t *testing.T) {
	f := newFixture(t, nil, nil)
	other := f.dir("other")
	f.command(other, "tz")
	f.d.Path = []string{other} // the shim directory is not on the PATH
	jf := f.justfile("proj")

	res, err := f.d.Add(jf, opts("tz"))
	if err != nil {
		t.Fatalf("should not be a collision: %v", err)
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "not on your PATH") {
		t.Errorf("warnings = %v", res.Warnings)
	}
}

func TestOwnStubIsNotACollisionOnReAdd(t *testing.T) {
	f := newFixture(t, nil, nil)
	jf := f.justfile("proj")
	for i := 0; i < 2; i++ {
		if _, err := f.d.Add(jf, opts("tz")); err != nil {
			t.Fatalf("add #%d: %v", i+1, err)
		}
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	f := newFixture(t, nil, nil)
	o := opts("tz")
	o.DryRun = true
	res, err := f.d.Add(f.justfile("proj"), o)
	if err != nil || !res.DryRun || len(res.Files) != 2 {
		t.Fatalf("result = %+v, %v", res, err)
	}
	if exists(f.d.stubPath("tz")) || exists(f.d.registryPath()) {
		t.Error("a dry run must not write anything")
	}
}

func TestChangingTheSuffixRemovesTheOldHelpStub(t *testing.T) {
	f := newFixture(t, nil, nil)
	jf := f.justfile("proj")
	if _, err := f.d.Add(jf, opts("tz")); err != nil {
		t.Fatal(err)
	}
	o := opts("tz")
	o.Suffix = "-help"
	if _, err := f.d.Add(jf, o); err != nil {
		t.Fatal(err)
	}
	if exists(f.d.stubPath("tzh")) {
		t.Error("the old help stub should have been removed")
	}
	if !exists(f.d.stubPath("tz-help")) {
		t.Error("the new help stub should exist")
	}
}

func TestRemove(t *testing.T) {
	f := newFixture(t, nil, nil)
	jf := f.justfile("proj")
	if _, err := f.d.Add(jf, opts("tz")); err != nil {
		t.Fatal(err)
	}

	if res, err := f.d.Remove("tz", true); err != nil || len(res.Removed) != 2 || !exists(f.d.stubPath("tz")) {
		t.Errorf("dry run: %+v, %v", res, err)
	}
	if reg, _ := f.d.Load(); len(reg.Shims) != 1 {
		t.Error("a dry run must keep the registry entry")
	}

	// Make one stub no longer ours: it must be skipped, not deleted.
	if err := os.WriteFile(f.d.stubPath("tzh"), []byte("#!/bin/sh\necho edited by hand\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := f.d.Remove("tz", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Removed) != 1 || len(res.Skipped) != 1 {
		t.Errorf("removed %v, skipped %v", res.Removed, res.Skipped)
	}
	if exists(f.d.stubPath("tz")) {
		t.Error("our stub should be gone")
	}
	if !exists(f.d.stubPath("tzh")) {
		t.Error("a file that is not ours must be left alone")
	}
	if reg, _ := f.d.Load(); len(reg.Shims) != 0 {
		t.Error("the registry entry should be gone")
	}
	if _, err := f.d.Remove("tz", false); err == nil {
		t.Error("removing an unknown shim should be an error")
	}
}

func TestFindJustfile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	jf := filepath.Join(root, "justfile")
	if err := os.WriteFile(jf, []byte("x:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, start := range []string{root, deep, jf} {
		got, err := FindJustfile(start)
		if err != nil || got != jf {
			t.Errorf("FindJustfile(%s) = %q, %v; want %q", start, got, err, jf)
		}
	}

	// A justfile above the project root (.git) must not be used.
	outer := t.TempDir()
	if err := os.WriteFile(filepath.Join(outer, "justfile"), []byte("x:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(outer, "project")
	if err := os.MkdirAll(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := FindJustfile(project); err == nil {
		t.Errorf("should stop at the project root, but found %q", got)
	}
}

func TestRegistryRoundTripAndErrors(t *testing.T) {
	f := newFixture(t, nil, nil)
	reg := Registry{Shims: []Entry{{Name: "b", Justfile: "/x/b", Suffix: "h"}, {Name: "a", Justfile: "/x/a", Suffix: "h", Soft: true}}}
	if err := f.d.Save(reg); err != nil {
		t.Fatal(err)
	}
	got, err := f.d.Load()
	if err != nil || len(got.Shims) != 2 || got.Shims[0].Name != "b" || !got.Shims[1].Soft {
		t.Errorf("round trip changed the registry (order must be kept): %+v, %v", got, err)
	}

	if err := os.WriteFile(f.d.registryPath(), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := f.d.Load(); err == nil {
		t.Error("a broken registry should be an error")
	}
	empty := newFixture(t, nil, nil)
	if r, err := empty.d.Load(); err != nil || len(r.Shims) != 0 {
		t.Errorf("a missing registry should be empty: %+v, %v", r, err)
	}
}

func TestNameComparisonDependsOnThePlatform(t *testing.T) {
	if !(Dirs{GOOS: "darwin"}).sameName("TZ", "tz") || !(Dirs{GOOS: "windows"}).sameName("TZ", "tz") {
		t.Error("names should compare case-insensitively on macOS and Windows")
	}
	if (Dirs{GOOS: "linux"}).sameName("TZ", "tz") {
		t.Error("names should compare exactly on Linux")
	}
}

func TestWindowsStubs(t *testing.T) {
	d := Dirs{GOOS: "windows", Shim: `C:\shims`}
	if !strings.HasSuffix(d.stubPath("tz"), "tz.cmd") {
		t.Errorf("stubPath = %q, want a .cmd file", d.stubPath("tz"))
	}
	c := d.stubContent("tz")
	if !strings.Contains(c, "\r\n") || !strings.Contains(c, marker) || !strings.Contains(c, "justz --as tz %*") {
		t.Errorf("windows stub content: %q", c)
	}
}

func TestDefaultDirsHonorOverrides(t *testing.T) {
	t.Setenv("JUSTZ_CONFIG_DIR", "/cfg")
	t.Setenv("JUSTZ_SHIM_DIR", "/shims")
	d, err := DefaultDirs()
	if err != nil || d.Config != "/cfg" || d.Shim != "/shims" {
		t.Errorf("DefaultDirs = %+v, %v", d, err)
	}
}
