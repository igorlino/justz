// Package shim manages shims: small named commands that run justz for a
// justfile. A registry file holds their settings, and each shim is a tiny
// stub file in a directory on the PATH.
package shim

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/igorlino/justz/internal/atomicfile"
)

// Entry is one registered shim.
type Entry struct {
	Name     string `json:"name"`
	Justfile string `json:"justfile"`
	Suffix   string `json:"suffix"`
	Soft     bool   `json:"soft"`
}

// HelpName is the name of the shim's help command, for example "tzh".
func (e Entry) HelpName() string { return e.Name + e.Suffix }

// Registry is the content of shims.json.
type Registry struct {
	Shims []Entry `json:"shims"`
}

// Dirs says where the registry and the shim stubs live, and what the PATH is.
// Tests build their own; DefaultDirs reads the environment.
type Dirs struct {
	Config string   // directory of shims.json
	Shim   string   // directory the stubs are written to
	Path   []string // the PATH entries, in order
	GOOS   string
}

// DefaultDirs works out the directories from the environment.
//
// Registry: $JUSTZ_CONFIG_DIR, else %APPDATA%\justz on Windows, else
// $XDG_CONFIG_HOME/justz, else ~/.config/justz.
// Stubs: $JUSTZ_SHIM_DIR, else %LOCALAPPDATA%\justz\bin on Windows, else
// ~/.local/bin.
func DefaultDirs() (Dirs, error) {
	d := Dirs{GOOS: runtime.GOOS, Path: filepath.SplitList(os.Getenv("PATH"))}
	home, herr := os.UserHomeDir()

	switch {
	case os.Getenv("JUSTZ_CONFIG_DIR") != "":
		d.Config = os.Getenv("JUSTZ_CONFIG_DIR")
	case d.GOOS == "windows" && os.Getenv("APPDATA") != "":
		d.Config = filepath.Join(os.Getenv("APPDATA"), "justz")
	case os.Getenv("XDG_CONFIG_HOME") != "":
		d.Config = filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "justz")
	case herr == nil:
		d.Config = filepath.Join(home, ".config", "justz")
	default:
		return d, fmt.Errorf("cannot work out the config directory: %v", herr)
	}

	switch {
	case os.Getenv("JUSTZ_SHIM_DIR") != "":
		d.Shim = os.Getenv("JUSTZ_SHIM_DIR")
	case d.GOOS == "windows" && os.Getenv("LOCALAPPDATA") != "":
		d.Shim = filepath.Join(os.Getenv("LOCALAPPDATA"), "justz", "bin")
	case herr == nil:
		d.Shim = filepath.Join(home, ".local", "bin")
	default:
		return d, fmt.Errorf("cannot work out the shim directory: %v", herr)
	}
	return d, nil
}

func (d Dirs) registryPath() string { return filepath.Join(d.Config, "shims.json") }

// caseInsensitive reports whether file names are compared without case here.
func (d Dirs) caseInsensitive() bool { return d.GOOS == "darwin" || d.GOOS == "windows" }

func (d Dirs) sameName(a, b string) bool {
	if d.caseInsensitive() {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// Load reads the registry. A missing file is an empty registry.
func (d Dirs) Load() (Registry, error) {
	var r Registry
	data, err := os.ReadFile(d.registryPath())
	if os.IsNotExist(err) {
		return Registry{Shims: []Entry{}}, nil
	}
	if err != nil {
		return r, err
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return r, fmt.Errorf("%s is not valid: %v", d.registryPath(), err)
	}
	if r.Shims == nil {
		r.Shims = []Entry{}
	}
	return r, nil
}

// Save writes the registry atomically.
func (d Dirs) Save(r Registry) error {
	if err := os.MkdirAll(d.Config, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return atomicfile.Write(d.registryPath(), append(data, '\n'), 0o644)
}

// find returns the index of the entry with this name, or -1.
func (d Dirs) find(r Registry, name string) int {
	for i, e := range r.Shims {
		if d.sameName(e.Name, name) {
			return i
		}
	}
	return -1
}
