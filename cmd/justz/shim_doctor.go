package main

import (
	"fmt"

	"github.com/igorlino/justz/internal/shim"
)

const shimDoctorUsage = `Usage: justz shim doctor [--json]

Check every registered shim: that justz is on the PATH, that the shim
directory is on the PATH, that each project's justfile still exists, that each
stub is still the one justz wrote, and that no other command comes first on
the PATH and hides the shim.

Shell aliases and functions cannot be seen from here, so a shim hidden by one
is not reported.

Exit codes: 0 healthy, 1 problems found, 2 usage error.
`

type shimDoctorReport struct {
	OK       bool           `json:"ok"`
	Findings []shim.Finding `json:"findings"`
}

func runShimDoctor(args []string, e env) int {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			fmt.Fprint(e.stdout, shimDoctorUsage)
			return 0
		}
	}
	asJSON, err := parseJSONFlag(args)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz shim doctor: %v\n", err)
		return 2
	}
	d, err := shim.DefaultDirs()
	if err != nil {
		fmt.Fprintf(e.stderr, "justz shim doctor: %v\n", err)
		return 1
	}
	findings, err := d.Doctor(e.lookPath)
	if err != nil {
		fmt.Fprintf(e.stderr, "justz shim doctor: %v\n", err)
		return 1
	}
	report := shimDoctorReport{OK: shim.Healthy(findings), Findings: findings}
	if asJSON {
		if code := printJSON(e, report); code != 0 {
			return code
		}
	} else {
		printFindings(e, findings)
	}
	if !report.OK {
		return 1
	}
	return 0
}

func printFindings(e env, findings []shim.Finding) {
	for _, f := range findings {
		status := "ok"
		if !f.OK {
			status = "fail"
		}
		label := f.Check
		if f.Shim != "" {
			label = f.Shim + ": " + f.Check
		}
		fmt.Fprintf(e.stdout, "[%s] %s: %s\n", status, label, f.Detail)
	}
}
