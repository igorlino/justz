package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/igorlino/justz/internal/shim"
)

type check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type doctorReport struct {
	Version string  `json:"version"`
	OK      bool    `json:"ok"`
	Checks  []check `json:"checks"`
}

func runDoctor(args []string, e env) int {
	asJSON, err := parseJSONFlag(args)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz doctor: %v\n", err)
		return 2
	}

	report := doctorReport{Version: currentVersion(), OK: true}
	report.Checks = append(report.Checks, checkJust(e))
	report.Checks = append(report.Checks, checkAliases()...)
	for _, c := range report.Checks {
		if !c.OK {
			report.OK = false
		}
	}

	if asJSON {
		enc := json.NewEncoder(e.stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(e.stderr, "justz doctor: %v\n", err)
			return 1
		}
	} else {
		fmt.Fprintf(e.stdout, "justz %s\n", report.Version)
		for _, c := range report.Checks {
			status := "ok"
			if !c.OK {
				status = "fail"
			}
			fmt.Fprintf(e.stdout, "[%s] %s: %s\n", status, c.Name, c.Detail)
		}
	}

	if !report.OK {
		return 1
	}
	return 0
}

func checkJust(e env) check {
	path, err := e.lookPath("just")
	if err != nil {
		return check{
			Name:   "just",
			Detail: "not found on PATH; install it from https://github.com/casey/just",
		}
	}
	ver, err := e.runCmd(path, "--version")
	if err != nil || ver == "" {
		return check{Name: "just", Detail: fmt.Sprintf("found at %s but `just --version` failed", path)}
	}
	return check{Name: "just", OK: true, Detail: fmt.Sprintf("%s (%s)", path, ver)}
}

// checkAliases reports on the jz and jzh names installed next to this binary,
// if there are any: an older script of the same name earlier on the PATH
// would hide them.
func checkAliases() []check {
	self, err := os.Executable()
	if err != nil {
		return nil
	}
	d, err := shim.DefaultDirs()
	if err != nil {
		return nil
	}
	var out []check
	for _, f := range d.CheckAliases(self) {
		out = append(out, check{Name: f.Check, OK: f.OK, Detail: f.Detail})
	}
	return out
}
