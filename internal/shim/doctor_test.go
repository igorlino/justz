package shim

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func justzFound(string) (string, error)   { return "/usr/local/bin/justz", nil }
func justzMissing(string) (string, error) { return "", errors.New("not found") }

func findingFor(findings []Finding, shim, check string) *Finding {
	for i := range findings {
		if findings[i].Shim == shim && findings[i].Check == check {
			return &findings[i]
		}
	}
	return nil
}

func TestDoctorHealthy(t *testing.T) {
	f := newFixture(t, nil, nil)
	if _, err := f.d.Add(f.justfile("proj"), opts("zzdoc")); err != nil {
		t.Fatal(err)
	}
	findings, err := f.d.Doctor(justzFound)
	if err != nil {
		t.Fatal(err)
	}
	if !Healthy(findings) {
		t.Errorf("a fresh setup should be healthy: %+v", findings)
	}
	for _, check := range []string{"command zzdoc", "command zzdoch", "justfile"} {
		if findingFor(findings, "zzdoc", check) == nil {
			t.Errorf("missing finding %q", check)
		}
	}
}

func TestDoctorNoShims(t *testing.T) {
	f := newFixture(t, nil, nil)
	findings, err := f.d.Doctor(justzFound)
	if err != nil || len(findings) != 1 || !findings[0].OK {
		t.Errorf("with no shims only the justz check should run: %+v, %v", findings, err)
	}
}

func TestDoctorJustzMissing(t *testing.T) {
	f := newFixture(t, nil, nil)
	findings, _ := f.d.Doctor(justzMissing)
	if Healthy(findings) || findingFor(findings, "", "justz-on-path").OK {
		t.Errorf("a missing justz should be a problem: %+v", findings)
	}
}

func TestDoctorFindsAMovedJustfile(t *testing.T) {
	f := newFixture(t, nil, nil)
	jf := f.justfile("proj")
	if _, err := f.d.Add(jf, opts("zzdoc")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(jf); err != nil {
		t.Fatal(err)
	}
	findings, _ := f.d.Doctor(justzFound)
	got := findingFor(findings, "zzdoc", "justfile")
	if got == nil || got.OK || !strings.Contains(got.Detail, "does not exist") {
		t.Errorf("finding = %+v", got)
	}
}

func TestDoctorFindsAMissingStub(t *testing.T) {
	f := newFixture(t, nil, nil)
	if _, err := f.d.Add(f.justfile("proj"), opts("zzdoc")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(f.d.stubPath("zzdoc")); err != nil {
		t.Fatal(err)
	}
	findings, _ := f.d.Doctor(justzFound)
	got := findingFor(findings, "zzdoc", "command zzdoc")
	if got == nil || got.OK || !strings.Contains(got.Detail, "missing") {
		t.Errorf("finding = %+v", got)
	}
}

func TestDoctorFindsAReplacedStub(t *testing.T) {
	f := newFixture(t, nil, nil)
	if _, err := f.d.Add(f.justfile("proj"), opts("zzdoc")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.d.stubPath("zzdoc"), []byte("#!/bin/sh\necho other\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	findings, _ := f.d.Doctor(justzFound)
	got := findingFor(findings, "zzdoc", "command zzdoc")
	if got == nil || got.OK || !strings.Contains(got.Detail, "no longer a justz stub") {
		t.Errorf("finding = %+v", got)
	}
}

func TestDoctorFindsALaterShadowingCommand(t *testing.T) {
	f := newFixture(t, nil, nil)
	if _, err := f.d.Add(f.justfile("proj"), opts("zzdoc")); err != nil {
		t.Fatal(err)
	}
	// A package installs a command with the same name in an earlier directory.
	early := f.dir("early")
	f.command(early, "zzdoc")
	f.d.Path = []string{early, f.d.Shim}

	findings, _ := f.d.Doctor(justzFound)
	got := findingFor(findings, "zzdoc", "command zzdoc")
	if got == nil || got.OK || !strings.Contains(got.Detail, "never runs") || !strings.Contains(got.Detail, early) {
		t.Errorf("finding = %+v", got)
	}
	if ok := findingFor(findings, "zzdoc", "command zzdoch"); ok == nil || !ok.OK {
		t.Errorf("the help command was not shadowed: %+v", ok)
	}
}

func TestDoctorShimDirNotOnPath(t *testing.T) {
	f := newFixture(t, nil, nil)
	if _, err := f.d.Add(f.justfile("proj"), opts("zzdoc")); err != nil {
		t.Fatal(err)
	}
	f.d.Path = []string{f.dir("elsewhere")}
	findings, _ := f.d.Doctor(justzFound)
	got := findingFor(findings, "", "shim-dir-on-path")
	if got == nil || got.OK {
		t.Errorf("finding = %+v", got)
	}
	if cmd := findingFor(findings, "zzdoc", "command zzdoc"); cmd == nil || cmd.OK {
		t.Errorf("a shim the shell cannot find should not be ok: %+v", cmd)
	}
}
