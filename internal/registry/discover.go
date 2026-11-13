package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Discover finds the command registry in dir. The search order is any
// .<name>/config/commands.json (sorted by name), then config/commands.json,
// then just/commands.json.
func Discover(dir string) (string, error) {
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
			return c, nil
		}
	}
	return "", fmt.Errorf("no commands.json found in %s (looked in .*/config, config and just)", dir)
}
