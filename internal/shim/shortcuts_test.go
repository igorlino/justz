package shim

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// actions returns the action of jz and of jzh.
func actions(t *testing.T, got []Shortcut) (string, string) {
	t.Helper()
	if len(got) != 2 || got[0].Name != "jz" || got[1].Name != "jzh" {
		t.Fatalf("want one result for jz and one for jzh, got %+v", got)
	}
	return got[0].Action, got[1].Action
}

func TestShortcutsAreCreatedOnceAndCheckedByCheckAliases(t *testing.T) {
	f := newFixture(t, nil, nil)
	self := filepath.Join(f.dir("pkg"), "justz") // nothing is installed next to the binary

	got, err := f.d.Shortcuts(self, false)
	if err != nil {
		t.Fatal(err)
	}
	if jz, jzh := actions(t, got); jz != ShortcutCreated || jzh != ShortcutCreated {
		t.Fatalf("first run: %+v", got)
	}
	for _, n := range []string{"jz", "jzh"} {
		data, err := os.ReadFile(f.d.stubPath(n))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "justz --local "+n+" ") || !strings.Contains(string(data), marker) {
			t.Errorf("stub of %s = %q", n, data)
		}
	}

	// Running it again writes nothing new.
	got, err = f.d.Shortcuts(self, false)
	if err != nil {
		t.Fatal(err)
	}
	if jz, jzh := actions(t, got); jz != ShortcutExists || jzh != ShortcutExists {
		t.Errorf("second run: %+v", got)
	}

	// doctor sees the stubs as the commands a shell finds.
	found := f.d.CheckAliases(self)
	if len(found) != 2 || !found[0].OK || !found[1].OK {
		t.Errorf("CheckAliases = %+v", found)
	}
}

func TestShortcutsDryRunWritesNothing(t *testing.T) {
	f := newFixture(t, nil, nil)
	got, err := f.d.Shortcuts(filepath.Join(f.dir("pkg"), "justz"), true)
	if err != nil {
		t.Fatal(err)
	}
	if jz, jzh := actions(t, got); jz != ShortcutCreated || jzh != ShortcutCreated {
		t.Errorf("dry run: %+v", got)
	}
	if exists(f.d.stubPath("jz")) || exists(f.d.stubPath("jzh")) {
		t.Error("a dry run must not write a stub")
	}
}

func TestShortcutsNeverTakeANameThatIsInUse(t *testing.T) {
	t.Run("a file justz did not write", func(t *testing.T) {
		f := newFixture(t, nil, nil)
		f.foreignStub("jz")
		before, _ := os.ReadFile(f.d.stubPath("jz"))
		got, err := f.d.Shortcuts(filepath.Join(f.dir("pkg"), "justz"), false)
		if err != nil {
			t.Fatal(err)
		}
		// The two are added together: jzh is not written when jz is taken.
		if jz, jzh := actions(t, got); jz != ShortcutSkipped || jzh != ShortcutSkipped || !strings.Contains(got[1].Detail, "added together") {
			t.Errorf("got %+v", got)
		}
		if after, _ := os.ReadFile(f.d.stubPath("jz")); string(after) != string(before) {
			t.Error("the foreign file was overwritten")
		}
		if exists(f.d.stubPath("jzh")) {
			t.Error("jzh was written although jz is another tool")
		}
	})

	t.Run("a command earlier or later on the PATH", func(t *testing.T) {
		for _, place := range []string{"before", "after"} {
			f := newFixture(t, nil, nil)
			other := f.dir("other")
			f.command(other, "jz")
			if place == "before" {
				f.d.Path = []string{other, f.d.Shim}
			} else {
				f.d.Path = []string{f.d.Shim, other}
			}
			got, err := f.d.Shortcuts(filepath.Join(f.dir("pkg"), "justz"), false)
			if err != nil {
				t.Fatal(err)
			}
			if jz, jzh := actions(t, got); jz != ShortcutSkipped || jzh != ShortcutSkipped || !strings.Contains(got[0].Detail, "already a command on the PATH") {
				t.Errorf("%s: got %+v", place, got)
			}
			if exists(f.d.stubPath("jz")) || exists(f.d.stubPath("jzh")) {
				t.Errorf("%s: a stub was written over a name in use", place)
			}
		}
	})

	t.Run("a registered shim", func(t *testing.T) {
		f := newFixture(t, nil, nil)
		justfile := filepath.Join(f.dir("proj"), "justfile")
		if err := os.WriteFile(justfile, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := f.d.Add(justfile, AddOptions{Name: "jz", Suffix: "h"}); err != nil {
			t.Fatal(err)
		}
		before, _ := os.ReadFile(f.d.stubPath("jz"))
		got, err := f.d.Shortcuts(filepath.Join(f.dir("pkg"), "justz"), false)
		if err != nil {
			t.Fatal(err)
		}
		if jz, jzh := actions(t, got); jz != ShortcutSkipped || jzh != ShortcutSkipped {
			t.Errorf("got %+v", got)
		}
		if after, _ := os.ReadFile(f.d.stubPath("jz")); string(after) != string(before) {
			t.Error("the stub of the registered shim was rewritten")
		}
		// The pinned stub is not reported as the shortcut of justz setup.
		if found := f.d.CheckAliases(filepath.Join(f.root, "pkg", "justz")); len(found) != 0 {
			t.Errorf("CheckAliases = %+v", found)
		}
	})
}

func TestShortcutsLeaveAPackagedLinkAlone(t *testing.T) {
	f := newFixture(t, nil, nil)
	pkg := f.dir("pkg")
	for _, n := range []string{"jz", "jzh"} {
		if err := os.WriteFile(filepath.Join(pkg, n), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	f.d.Path = []string{pkg, f.d.Shim}
	got, err := f.d.Shortcuts(filepath.Join(pkg, "justz"), false)
	if err != nil {
		t.Fatal(err)
	}
	if jz, jzh := actions(t, got); jz != ShortcutInstalled || jzh != ShortcutInstalled {
		t.Errorf("got %+v", got)
	}
	if got[0].Path != filepath.Join(pkg, "jz") || exists(f.d.stubPath("jz")) {
		t.Errorf("got %+v", got)
	}
}

func TestLocalStubContentOnWindows(t *testing.T) {
	d := Dirs{GOOS: "windows"}
	if got := d.localStubContent("jzh"); !strings.Contains(got, "justz --local jzh %*\r\n") || !strings.Contains(got, marker) {
		t.Errorf("stub = %q", got)
	}
}

func TestShortcutThatCannotBeWrittenIsSkippedNotAnError(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a directory that cannot be written")
	}
	f := newFixture(t, nil, nil)
	if err := os.Chmod(f.d.Shim, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(f.d.Shim, 0o755) })
	got, err := f.d.Shortcuts(filepath.Join(f.dir("pkg"), "justz"), false)
	if err != nil {
		t.Fatalf("a stub that cannot be written must not stop setup: %v", err)
	}
	if jz, jzh := actions(t, got); jz != ShortcutSkipped || jzh != ShortcutSkipped || !strings.Contains(got[0].Detail, "cannot write") {
		t.Errorf("got %+v", got)
	}
}

func TestCheckAliasesWithSetupStubs(t *testing.T) {
	t.Run("the shim directory is not on the PATH: a warning, not a failure", func(t *testing.T) {
		f := newFixture(t, nil, nil)
		self := filepath.Join(f.dir("pkg"), "justz")
		if _, err := f.d.Shortcuts(self, false); err != nil {
			t.Fatal(err)
		}
		f.d.Path = []string{f.dir("elsewhere")}
		got := f.d.CheckAliases(self)
		if len(got) != 2 || !got[0].OK || !got[0].Warn || !strings.Contains(got[0].Detail, "not on the PATH") || !Healthy(got) {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("a stub and a packaged link: either one first is fine", func(t *testing.T) {
		f := newFixture(t, nil, nil)
		pkg := f.dir("pkg")
		self := filepath.Join(pkg, "justz")
		if _, err := f.d.Shortcuts(self, false); err != nil {
			t.Fatal(err)
		}
		for _, n := range []string{"jz", "jzh"} {
			if err := os.WriteFile(filepath.Join(pkg, n), []byte("#!/bin/sh\n"), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		for _, path := range [][]string{{f.d.Shim, pkg}, {pkg, f.d.Shim}} {
			f.d.Path = path
			if got := f.d.CheckAliases(self); len(got) != 2 || !got[0].OK || !got[1].OK || got[0].Warn {
				t.Errorf("PATH %v: got %+v", path, got)
			}
		}
	})

	t.Run("the binary is in the shim directory: a foreign jz there is not ours", func(t *testing.T) {
		f := newFixture(t, nil, nil)
		f.foreignStub("jz")
		if got := f.d.CheckAliases(filepath.Join(f.d.Shim, "justz")); len(got) != 0 {
			t.Errorf("got %+v", got)
		}
	})
}

func TestIsLocalStubReadsOnlyTheHead(t *testing.T) {
	f := newFixture(t, nil, nil)
	// A large file that mentions the marker far from its start, as the justz
	// binary itself does.
	p := filepath.Join(f.d.Shim, "big")
	if err := os.WriteFile(p, []byte(strings.Repeat("x", 8192)+localMarker), 0o755); err != nil {
		t.Fatal(err)
	}
	if isLocalStub(p) || isLocalStub(f.d.Shim) || isLocalStub(filepath.Join(f.d.Shim, "missing")) {
		t.Error("only a small file that starts with the stub text is a local stub")
	}
}
