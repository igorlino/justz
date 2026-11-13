package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Discover finds the command registry for the project that contains dir. In
// each directory it looks, in this order, for any .<name>/config/commands.json
// (sorted by name), then config/commands.json, then just/commands.json. It
// starts in dir and walks up through the parents, and stops after the first
// directory that contains a .git entry (the project root) or at the
// filesystem root, so it never wanders into an unrelated project.
func Discover(dir string) (string, error) {
	start := dir
	for {
		if found, ok := discoverIn(dir); ok {
			return found, nil
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no commands.json found in %s or its parents (looked in .*/config, config and just)", start)
}

func discoverIn(dir string) (string, bool) {
	var candidates []string
	hidden, _ := filepath.Glob(filepath.Join(dir, ".*", "config", "commands.json"))
	sort.Strings(hidden)
	candidates = append(candidates, hidden...)
	candidates = append(candidates,
		filepath.Join(dir, "config", "commands.json"),
		filepath.Join(dir, "just", "commands.json"),
	)
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.Mode().IsRegular() {
			return c, true
		}
	}
	return "", false
}
