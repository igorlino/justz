// Package doclinks has one test: every relative link in the documentation and
// the skills points at a file that exists.
package doclinks

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// link matches the target of a Markdown link, up to an anchor or the closing
// parenthesis.
var link = regexp.MustCompile(`\]\(([^)#\s]+)`)

func TestRelativeLinksResolve(t *testing.T) {
	root := filepath.Join("..", "..")
	files := []string{"README.md", "AGENTS.md", "CHANGELOG.md", "SECURITY.md"}
	for _, dir := range []string{"docs", "skills"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(p, ".md") {
				rel, err := filepath.Rel(root, p)
				if err != nil {
					return err
				}
				files = append(files, rel)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range link.FindAllStringSubmatch(string(data), -1) {
			target := m[1]
			if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			if _, err := os.Stat(filepath.Join(root, filepath.Dir(f), filepath.FromSlash(target))); err != nil {
				t.Errorf("%s: link to %s does not resolve", f, target)
			}
		}
	}
}
