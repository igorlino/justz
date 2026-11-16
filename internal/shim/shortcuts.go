package shim

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/igorlino/justz/internal/atomicfile"
)

// shortcutNames are the built-in shortcuts for the project around the current
// directory: jz runs a command, jzh shows its help.
var shortcutNames = []string{"jz", "jzh"}

// What Shortcuts did, or would do, for one name.
const (
	ShortcutCreated   = "created"   // the stub was written (or would be, in a dry run)
	ShortcutExists    = "exists"    // a stub that justz wrote is already there
	ShortcutInstalled = "installed" // a package put the name next to the justz binary
	ShortcutSkipped   = "skipped"   // something else has the name, so nothing was written
)

// Shortcut is the result for one of jz and jzh.
type Shortcut struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Action string `json:"action"`
	Detail string `json:"detail,omitempty"`
}

// localStubContent is the text of a jz or jzh stub. Unlike a pinned shim it has
// no registry entry: it asks justz to act for the project around the current
// directory.
func (d Dirs) localStubContent(name string) string {
	if d.GOOS == "windows" {
		return "@echo off\r\n" +
			"rem " + localMarker + "; remove it by deleting this file\r\n" +
			"justz --local " + name + " %*\r\n"
	}
	return "#!/bin/sh\n" +
		"# " + localMarker + "; remove it by deleting this file\n" +
		"exec justz --local " + name + " \"$@\"\n"
}

// Shortcuts writes the jz and jzh stubs into the shim directory, for an
// install that did not bring them (the Homebrew formula links them next to the
// binary at self). It follows the rules of a shim: it never overwrites a file
// justz did not write, and it never takes a name that another command on the
// PATH or a registered shim already has. A name it leaves alone is reported as
// skipped, with the reason; that is not an error, and neither is a stub that
// cannot be written.
//
// The two are added together: when one name is taken, the other is not
// written either, so jzh never shows the help of justz while jz is another
// tool.
func (d Dirs) Shortcuts(self string, dryRun bool) ([]Shortcut, error) {
	reg, err := d.Load()
	if err != nil {
		return nil, err
	}
	var out []Shortcut
	for _, n := range shortcutNames {
		out = append(out, d.decideShortcut(reg, self, n))
	}

	for i := range out {
		other := out[len(out)-1-i]
		if out[i].Action == ShortcutCreated && other.Action == ShortcutSkipped {
			out[i].Action = ShortcutSkipped
			out[i].Detail = fmt.Sprintf("%s was not added, and the two shortcuts are added together", other.Name)
		}
	}

	if dryRun {
		return out, nil
	}
	for i := range out {
		if out[i].Action != ShortcutCreated {
			continue
		}
		err := os.MkdirAll(d.Shim, 0o755)
		if err == nil {
			err = atomicfile.Write(out[i].Path, []byte(d.localStubContent(out[i].Name)), 0o755)
		}
		if err != nil {
			out[i].Action = ShortcutSkipped
			out[i].Detail = fmt.Sprintf("cannot write %s: %v", out[i].Path, err)
			// The two are added together: take back one written just before.
			for j := range out[:i] {
				if out[j].Action == ShortcutCreated && os.Remove(out[j].Path) == nil {
					out[j].Action = ShortcutSkipped
					out[j].Detail = fmt.Sprintf("%s could not be written, and the two shortcuts are added together", out[i].Name)
				}
			}
		}
	}
	return out, nil
}

// decideShortcut says what would happen to one name. It writes nothing.
func (d Dirs) decideShortcut(reg Registry, self, n string) Shortcut {
	sp := d.stubPath(n)
	s := Shortcut{Name: n, Path: sp, Action: ShortcutSkipped}
	packaged := filepath.Join(filepath.Dir(self), n)
	switch found, _ := d.firstOnPath(n); {
	case d.registered(reg, n) != "":
		s.Detail = fmt.Sprintf("the shim %q already uses this name", d.registered(reg, n))
	case isLocalStub(sp):
		s.Action = ShortcutExists
	case exists(sp):
		s.Detail = fmt.Sprintf("%s already exists and is not a shortcut that justz setup wrote, so it will not be overwritten", sp)
	case self != "" && exists(packaged):
		s.Action = ShortcutInstalled
		s.Path = packaged
	case found != "":
		s.Detail = fmt.Sprintf("%s is already a command on the PATH", found)
	default:
		s.Action = ShortcutCreated
	}
	return s
}

// registered returns the name of the registered shim that uses n as its
// command or its help command, or "".
func (d Dirs) registered(reg Registry, n string) string {
	for _, e := range reg.Shims {
		if d.sameName(e.Name, n) || d.sameName(e.HelpName(), n) {
			return e.Name
		}
	}
	return ""
}
