package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/igorlino/justz/internal/atomicfile"
)

// AddOptions are the choices for adding a shim.
type AddOptions struct {
	Name   string // the shim name
	Suffix string // the help suffix, usually "h"
	Soft   bool
	Overrides
	DryRun bool
}

// AddResult describes what Add did, or would do with DryRun.
type AddResult struct {
	Entry    Entry    `json:"entry"`
	Files    []string `json:"files"`
	Updated  bool     `json:"updated"`
	DryRun   bool     `json:"dryRun"`
	Warnings []string `json:"warnings"`
}

// CollisionError is returned when the shim cannot be added as asked.
type CollisionError struct{ Problems []Problem }

func (e *CollisionError) Error() string {
	var b strings.Builder
	for i, p := range e.Problems {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(p.String())
	}
	return b.String()
}

// Add registers a shim for justfile and writes its stubs.
func (d Dirs) Add(justfile string, o AddOptions) (AddResult, error) {
	var res AddResult
	abs, err := filepath.Abs(justfile)
	if err != nil {
		return res, err
	}
	if info, err := os.Stat(abs); err != nil || info.IsDir() {
		return res, fmt.Errorf("%s is not a file", abs)
	}

	e := Entry{Name: o.Name, Justfile: abs, Suffix: o.Suffix, Soft: o.Soft}
	reg, err := d.Load()
	if err != nil {
		return res, err
	}
	if blocking := Blocking(d.Check(reg, e), o.Overrides); len(blocking) > 0 {
		return res, &CollisionError{Problems: blocking}
	}

	res.Entry = e
	res.DryRun = o.DryRun
	res.Files = []string{d.stubPath(e.Name), d.stubPath(e.HelpName())}
	res.Warnings = []string{}
	if !d.ShimDirOnPath() {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s is not on your PATH, so the new commands will not be found until you add it", d.Shim))
	}
	idx := d.find(reg, e.Name)
	res.Updated = idx >= 0
	if o.DryRun {
		return res, nil
	}

	if err := os.MkdirAll(d.Shim, 0o755); err != nil {
		return res, err
	}
	for _, n := range []string{e.Name, e.HelpName()} {
		if err := atomicfile.Write(d.stubPath(n), []byte(d.stubContent(n)), 0o755); err != nil {
			return res, err
		}
	}
	if idx >= 0 {
		// If the help suffix changed, remove the old help stub if it is ours.
		if old := reg.Shims[idx]; old.HelpName() != e.HelpName() {
			if sp := d.stubPath(old.HelpName()); isOurs(sp) {
				os.Remove(sp)
			}
		}
		reg.Shims[idx] = e
	} else {
		reg.Shims = append(reg.Shims, e)
	}
	if err := d.Save(reg); err != nil {
		return res, err
	}
	return res, nil
}

// RemoveResult describes what Remove did.
type RemoveResult struct {
	Name    string   `json:"name"`
	Removed []string `json:"removed"`
	Skipped []string `json:"skipped"`
	DryRun  bool     `json:"dryRun"`
}

// Remove deletes a shim from the registry and removes its stubs, but only
// stubs that justz wrote.
func (d Dirs) Remove(name string, dryRun bool) (RemoveResult, error) {
	res := RemoveResult{Name: name, Removed: []string{}, Skipped: []string{}, DryRun: dryRun}
	reg, err := d.Load()
	if err != nil {
		return res, err
	}
	idx := d.find(reg, name)
	if idx < 0 {
		return res, fmt.Errorf("there is no shim named %q", name)
	}
	e := reg.Shims[idx]
	for _, n := range []string{e.Name, e.HelpName()} {
		sp := d.stubPath(n)
		switch {
		case !exists(sp):
			// Nothing to remove.
		case !isOurs(sp):
			res.Skipped = append(res.Skipped, sp)
		default:
			res.Removed = append(res.Removed, sp)
			if !dryRun {
				if err := os.Remove(sp); err != nil {
					return res, err
				}
			}
		}
	}
	if dryRun {
		return res, nil
	}
	reg.Shims = append(reg.Shims[:idx], reg.Shims[idx+1:]...)
	return res, d.Save(reg)
}

// FindJustfile returns the justfile for a project. start may be the justfile
// itself, or a directory: the search then walks up to the project root (the
// first directory with a .git entry) looking for justfile, Justfile or
// .justfile.
func FindJustfile(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return abs, nil
	}
	dir := abs
	for {
		for _, n := range []string{"justfile", "Justfile", ".justfile"} {
			p := filepath.Join(dir, n)
			if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() {
				return p, nil
			}
		}
		if exists(filepath.Join(dir, ".git")) {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no justfile found in %s or its parents", abs)
}
