package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Kind classifies a collision.
type Kind string

const (
	// Invalid: the name is not allowed at all. Never overridable.
	Invalid Kind = "invalid-name"
	// NotOurs: a file in the shim directory is not a justz stub. justz never
	// overwrites a file it did not write. Never overridable.
	NotOurs Kind = "not-ours"
	// Owned: another registered shim, or the same name for another
	// justfile, already uses the name. Overridable with Replace.
	Owned Kind = "owned"
	// Shadowed: a command with this name comes earlier on the PATH, so the
	// shim would never run. Overridable with Force.
	Shadowed Kind = "shadowed"
	// Hides: a command with this name comes later on the PATH, so the shim
	// would hide it. Overridable with Replace.
	Hides Kind = "hides"
)

// Problem is one reason a shim cannot be added as asked.
type Problem struct {
	Kind   Kind   `json:"kind"`
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

func (p Problem) String() string { return fmt.Sprintf("%s: %s", p.Name, p.Detail) }

// Overrides says which flags the caller passed.
type Overrides struct {
	Force   bool // allow a shadowed shim
	Replace bool // allow re-pointing a shim, or hiding an existing command
}

// Blocking returns the problems that the overrides do not cover.
func Blocking(problems []Problem, o Overrides) []Problem {
	var out []Problem
	for _, p := range problems {
		switch p.Kind {
		case Shadowed:
			if o.Force {
				continue
			}
		case Owned, Hides:
			if o.Replace {
				continue
			}
		}
		out = append(out, p)
	}
	return out
}

// Check lists every reason the entry cannot be added. It looks at the names
// themselves, at what is in the shim directory, at the other registered
// shims, and at what is on the PATH.
func (d Dirs) Check(reg Registry, e Entry) []Problem {
	var problems []Problem
	names := []string{e.Name, e.HelpName()}

	if e.Suffix == "" {
		problems = append(problems, Problem{Invalid, e.HelpName(), "the help suffix cannot be empty, or the help command would have the same name as the shim"})
	}
	for _, n := range names {
		if err := ValidateName(n); err != nil {
			problems = append(problems, Problem{Invalid, n, err.Error()})
		}
	}
	if len(problems) > 0 {
		return problems
	}

	for _, n := range names {
		problems = append(problems, d.checkOne(reg, e, n)...)
	}
	return problems
}

func (d Dirs) checkOne(reg Registry, e Entry, n string) []Problem {
	var problems []Problem

	// A file in the shim directory that is not ours: never touch it.
	sp := d.stubPath(n)
	notOurs := exists(sp) && !isOurs(sp)
	if notOurs {
		problems = append(problems, Problem{NotOurs, n, fmt.Sprintf("%s already exists and was not written by justz, so it will not be overwritten", sp)})
	}

	// Another shim already uses the name, or this name is already taken by a
	// different justfile.
	for _, o := range reg.Shims {
		if d.sameName(o.Name, e.Name) {
			if o.Justfile != e.Justfile && n == e.Name {
				problems = append(problems, Problem{Owned, n, fmt.Sprintf("the shim %q already points at %s", o.Name, o.Justfile)})
			}
			continue
		}
		if d.sameName(o.Name, n) || d.sameName(o.HelpName(), n) {
			problems = append(problems, Problem{Owned, n, fmt.Sprintf("the shim %q (%s) already uses this name", o.Name, o.Justfile)})
		}
	}

	// A command with the same name somewhere on the PATH.
	// A file that is not ours at the stub path is already reported above, so it
	// is not reported a second time as a command the shim would hide.
	if found, idx := d.firstOnPath(n); found != "" && !(notOurs && filepath.Clean(found) == filepath.Clean(sp)) {
		shimIdx := d.shimDirIndex()
		switch {
		case shimIdx < 0:
			// The shim directory is not on the PATH at all. That is a
			// warning for the caller, not a collision.
		case idx < shimIdx:
			problems = append(problems, Problem{Shadowed, n, fmt.Sprintf("%s comes earlier on the PATH than the shim directory %s, so the shim would never run", found, d.Shim)})
		default:
			problems = append(problems, Problem{Hides, n, fmt.Sprintf("%s is on the PATH after the shim directory %s, so the shim would hide it", found, d.Shim)})
		}
	}
	return problems
}

// ShimDirOnPath reports whether the shim directory is on the PATH.
func (d Dirs) ShimDirOnPath() bool { return d.shimDirIndex() >= 0 }

func (d Dirs) shimDirIndex() int {
	for i, p := range d.Path {
		if d.sameDir(p, d.Shim) {
			return i
		}
	}
	return -1
}

func (d Dirs) sameDir(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	ca, cb := filepath.Clean(a), filepath.Clean(b)
	if ra, err := filepath.EvalSymlinks(ca); err == nil {
		ca = ra
	}
	if rb, err := filepath.EvalSymlinks(cb); err == nil {
		cb = rb
	}
	return d.sameName(ca, cb)
}

// firstOnPath finds the first command called name on the PATH, ignoring the
// stub in the shim directory, and returns its path and the PATH index.
func (d Dirs) firstOnPath(name string) (string, int) {
	for i, dir := range d.Path {
		if dir == "" {
			continue
		}
		for _, cand := range d.candidates(dir, name) {
			if d.sameDir(dir, d.Shim) && isOurs(cand) {
				continue // our own stub
			}
			if isExecutable(cand, d.GOOS) {
				return cand, i
			}
		}
	}
	return "", -1
}

// candidates are the file names a shell would try for name in dir.
func (d Dirs) candidates(dir, name string) []string {
	if d.GOOS != "windows" {
		return []string{filepath.Join(dir, name)}
	}
	exts := os.Getenv("PATHEXT")
	if exts == "" {
		exts = ".COM;.EXE;.BAT;.CMD"
	}
	out := []string{}
	for _, ext := range strings.Split(exts, ";") {
		if ext != "" {
			out = append(out, filepath.Join(dir, name+strings.ToLower(ext)))
		}
	}
	return out
}

func isExecutable(path, goos string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	if goos == "windows" {
		return true
	}
	return info.Mode().Perm()&0o111 != 0
}
