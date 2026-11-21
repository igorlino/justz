package doclinks

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/igorlino/justz/internal/check"
	"github.com/igorlino/justz/internal/lint"
)

// The limits of docs/writing-docs.md. Raise one only with a reason in the PR.
const (
	maxReadmeWords      = 1150
	maxReadmeBulletWord = 40
	maxSkillLines       = 500
	maxSkillWords       = 3000
)

func read(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// command matches a subcommand in the dispatch switch of cmd/justz/main.go.
var command = regexp.MustCompile(`(?m)^\tcase "([a-z]+)":`)

// Every command has a row in the command reference and in the justz-cli skill.
func TestEveryCommandIsDocumented(t *testing.T) {
	found := command.FindAllStringSubmatch(read(t, "cmd/justz/main.go"), -1)
	if len(found) < 10 {
		t.Fatalf("found only %d commands in cmd/justz/main.go; the pattern no longer matches the switch", len(found))
	}
	for _, page := range []string{"docs/commands.md", "skills/justz-cli/SKILL.md"} {
		text := read(t, page)
		for _, m := range found {
			if name := m[1]; name != "version" && !strings.Contains(text, "`justz "+name) {
				t.Errorf("%s does not document the command: justz %s", page, name)
			}
		}
	}
}

// Every check and lint rule is in docs/checks.md.
func TestEveryRuleIsDocumented(t *testing.T) {
	text := read(t, "docs/checks.md")
	for _, rule := range append(append([]string{}, check.Rules...), lint.Rules...) {
		if !strings.Contains(text, "`"+rule+"`") {
			t.Errorf("docs/checks.md does not document the rule %s", rule)
		}
	}
}

// The README stays a front page: short, with short bullets. It counts words,
// not lines, because the Markdown style puts one sentence on each line.
func TestReadmeBudget(t *testing.T) {
	lines := strings.Split(strings.TrimRight(read(t, "README.md"), "\n"), "\n")
	if n := countWords(lines); n > maxReadmeWords {
		t.Errorf("README.md has %d words (limit %d): move the detail to a page in docs/", n, maxReadmeWords)
	}
	// A bullet goes on over the indented lines below it.
	for i := 0; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "- ") {
			continue
		}
		end := i + 1
		for end < len(lines) && strings.HasPrefix(lines[end], "  ") {
			end++
		}
		if n := countWords(lines[i:end]); n > maxReadmeBulletWord {
			t.Errorf("README.md:%d: a bullet of %d words (limit %d): move the detail to a page in docs/", i+1, n, maxReadmeBulletWord)
		}
		i = end - 1
	}
}

// countWords counts the words of lines: a table pipe or a list dash is not one.
func countWords(lines []string) int {
	n := 0
	for _, line := range lines {
		for _, f := range strings.Fields(line) {
			if strings.IndexFunc(f, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) >= 0 {
				n++
			}
		}
	}
	return n
}

// A SKILL.md stays an entry point; the long material goes in references/.
func TestSkillBudget(t *testing.T) {
	skills, err := filepath.Glob(filepath.Join("..", "..", "skills", "*", "SKILL.md"))
	if err != nil || len(skills) == 0 {
		t.Fatalf("no skills found: %v", err)
	}
	for _, p := range skills {
		f, err := os.Open(p)
		if err != nil {
			t.Fatal(err)
		}
		lines, words := 0, 0
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
		for sc.Scan() {
			lines++
			words += len(strings.Fields(sc.Text()))
		}
		f.Close()
		if lines > maxSkillLines || words > maxSkillWords {
			t.Errorf("%s has %d lines and %d words (limits %d and %d): move material to its references/", p, lines, words, maxSkillLines, maxSkillWords)
		}
	}
}
