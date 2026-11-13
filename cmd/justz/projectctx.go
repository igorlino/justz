package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/igorlino/justz/internal/orderedjson"
	"github.com/igorlino/justz/internal/registry"
	"github.com/igorlino/justz/internal/shim"
)

// projectCtx is what the commands that inspect a project share: the justfile,
// the output of just --dump, and the registry or overlay if there is one.
type projectCtx struct {
	Justfile     string
	Dump         string
	Variant      string // native, rich or local-rich
	RegistryPath string
	Registry     *orderedjson.Value
	Source       []byte // the registry file as written
}

// loadProjectCtx finds the project around the working directory.
func loadProjectCtx(e env) (*projectCtx, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	justfile, err := shim.FindJustfile(cwd)
	if err != nil {
		return nil, err
	}
	dump, err := e.runCmd("just", "--justfile", justfile, "--dump", "--dump-format", "json")
	if err != nil {
		return nil, fmt.Errorf("could not read the justfile with just: %v", err)
	}

	ctx := &projectCtx{Justfile: justfile, Dump: dump, Variant: "native"}
	if path, err := registry.Discover(filepath.Dir(justfile)); err == nil {
		ctx.Variant, ctx.RegistryPath = "rich", path
	} else if overlay := filepath.Join(projectRoot(filepath.Dir(justfile)), ".just-local-rich.json"); fileExists(overlay) {
		ctx.Variant, ctx.RegistryPath = "local-rich", overlay
	}
	if ctx.RegistryPath != "" {
		if ctx.Source, err = os.ReadFile(ctx.RegistryPath); err != nil {
			return nil, err
		}
		if ctx.Registry, err = orderedjson.Parse(ctx.Source); err != nil {
			return nil, fmt.Errorf("%s is not valid JSON: %v", ctx.RegistryPath, err)
		}
	}
	return ctx, nil
}

// loadSources reads the formatted text that just --dump prints for the root
// justfile and for each top-level module, keyed by module name ("" for the
// root). It keeps the whole comment block above each recipe, which the JSON
// dump does not. A file that cannot be dumped is left out.
func (c *projectCtx) loadSources(e env) map[string]string {
	out := map[string]string{}
	if text, err := e.runCmd("just", "--justfile", c.Justfile, "--dump"); err == nil {
		out[""] = text
	}
	var d struct {
		Modules map[string]struct {
			Source string `json:"source"`
		} `json:"modules"`
	}
	if json.Unmarshal([]byte(c.Dump), &d) == nil {
		for name, m := range d.Modules {
			if m.Source == "" {
				continue
			}
			if text, err := e.runCmd("just", "--justfile", m.Source, "--dump"); err == nil {
				out[name] = text
			}
		}
	}
	return out
}
