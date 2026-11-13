package main

import (
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
