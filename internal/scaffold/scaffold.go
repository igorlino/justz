// Package scaffold builds the files justz init writes: a justfile, a shared
// preamble, one module per command group and, for the rich variant, the
// registry. It does no I/O; the caller decides what to write.
package scaffold

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/igorlino/justz/internal/orderedjson"
	"github.com/igorlino/justz/internal/registry"
)

const (
	Native = "native"
	Rich   = "rich"
)

// Options describe the project to scaffold.
type Options struct {
	Name    string   // heading shown by the help; must be plain text
	Variant string   // Native or Rich
	Modules []string // one command group each
}

// File is a path relative to the project root and its content.
type File struct {
	Path    string
	Content string
}

var (
	namePattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._-]*$`)
	modulePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	// A module named like a core recipe or the shared file would collide with it.
	reserved = map[string]bool{"core": true, "help": true, "check": true, "preamble": true}
)

const preamble = `# Settings every file shares: the root justfile and each module import this one.
# just does not inherit settings into a module, so without the import a module
# runs recipes with sh and no pipefail.
#
# working-directory is not set here on purpose: the root runs in the project
# directory and a module in the directory of its file, so no single value fits
# both. Use justfile_directory() for paths from the project root.
set shell := ['bash', '-eu', '-o', 'pipefail', '-c']
set positional-arguments
`

const richHelp = `[doc('Full command reference, or the focused docs of one group or command (help [group] [term]); -b browses, -j emits JSON')]
[group('core')]
[no-exit-message]
help *ARGS:
    #!/usr/bin/env bash
    set -euo pipefail
    # No words: the full reference. Otherwise zoom, with -b, -j and -w.
    if [[ $# -eq 0 ]]; then exec justz help --expanded; fi
    exec justz help "$@"

[doc('Check the documentation against the justfile and the registry order')]
[group('core')]
check:
    @justz check
    @justz fmt --check
`

const (
	richHelpDesc  = "Full command reference, or the focused docs of one group or command (help [group] [term]); -b browses, -j emits JSON"
	richCheckDesc = "Check the documentation against the justfile and the registry order"
)

// Validate reports the first problem with the options.
func (o Options) Validate() error {
	if !namePattern.MatchString(o.Name) {
		return fmt.Errorf("project name %q: use letters, digits, spaces, dot, dash and underscore, starting with a letter or digit", o.Name)
	}
	if o.Variant != Native && o.Variant != Rich {
		return fmt.Errorf("variant %q: use %s or %s", o.Variant, Native, Rich)
	}
	seen := map[string]bool{}
	for _, m := range o.Modules {
		switch {
		case !modulePattern.MatchString(m):
			return fmt.Errorf("module %q: use lowercase letters, digits and dashes, starting with a letter", m)
		case reserved[m]:
			return fmt.Errorf("module %q: the name is reserved (%s)", m, "core, help, check, preamble")
		case seen[m]:
			return fmt.Errorf("module %q is given twice", m)
		}
		seen[m] = true
	}
	return nil
}

// Build returns the files to write, in a stable order.
func Build(o Options) ([]File, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	files := []File{
		{"justfile", rootJustfile(o)},
		{"just/preamble.just", preamble},
	}
	for _, m := range o.Modules {
		files = append(files, File{path.Join("just", m+".just"), moduleFile(o, m)})
	}
	if o.Variant == Rich {
		reg, err := registryJSON(o)
		if err != nil {
			return nil, err
		}
		files = append(files, File{"config/commands.json", reg})
	}
	return files, nil
}

func rootJustfile(o Options) string {
	var b strings.Builder
	b.WriteString("import 'just/preamble.just'\n\n")
	for _, m := range o.Modules {
		fmt.Fprintf(&b, "[group('commands')]\n[doc('%s')]\nmod %s 'just/%s.just'\n\n", moduleDoc(m), m, m)
	}
	if o.Variant == Rich {
		b.WriteString("_default:\n    @justz help\n\n")
		b.WriteString(richHelp)
		return b.String()
	}
	// Private, so it stays out of its own listing. just runs the first recipe
	// of the file when given no arguments, so keep this one first of the recipes.
	fmt.Fprintf(&b, "_default:\n    @just --unsorted --list --list-heading $'\\n  %s\\n\\n'\n\n", o.Name)
	b.WriteString("[doc('Expand every namespace in one listing (bare `just` keeps them collapsed)')]\n[group('core')]\n")
	fmt.Fprintf(&b, "help:\n    @just --unsorted --list-submodules --list-heading $'\\n  %s: every recipe\\n\\n' --list\n", o.Name)
	return b.String()
}

func moduleFile(o Options, m string) string {
	var b strings.Builder
	b.WriteString("import 'preamble.just'\n\n")
	if o.Variant == Rich {
		fmt.Fprintf(&b, "_default:\n    @justz help %s\n", m)
	} else {
		fmt.Fprintf(&b, "_default:\n    @just --unsorted --list %s\n", m)
	}
	b.WriteString("\n# Add recipes here. Every public recipe needs [doc('...')].\n")
	return b.String()
}

func moduleDoc(m string) string { return m + " commands" }

func registryJSON(o Options) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, `{"name":%q,"groups":{"core":{"label":"Core","commands":{`, o.Name)
	fmt.Fprintf(&b, `"check":{"desc":%q},"help":{"desc":%q}}}`, richCheckDesc, richHelpDesc)
	for _, m := range o.Modules {
		fmt.Fprintf(&b, `,%q:{"label":%q,"commands":{}}`, m, strings.ToUpper(m[:1])+m[1:])
	}
	b.WriteString("}}")
	doc, err := orderedjson.Parse([]byte(b.String()))
	if err != nil {
		return "", err
	}
	registry.Canonicalize(doc)
	out, err := orderedjson.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
