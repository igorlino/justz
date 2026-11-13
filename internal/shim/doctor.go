package shim

import (
	"fmt"
	"os"
	"path/filepath"
)

// Finding is one result of Doctor.
type Finding struct {
	Shim   string `json:"shim,omitempty"` // empty for checks about the whole setup
	Check  string `json:"check"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// Doctor re-checks every registered shim, so problems that appeared after the
// shim was added (a new package took the name, a repository moved, the PATH
// changed) are found. It cannot see shell aliases or functions, because those
// do not exist in a child process.
//
// lookPath finds a command on the PATH; tests replace it.
func (d Dirs) Doctor(lookPath func(string) (string, error)) ([]Finding, error) {
	reg, err := d.Load()
	if err != nil {
		return nil, err
	}
	var out []Finding

	if p, err := lookPath("justz"); err != nil {
		out = append(out, Finding{Check: "justz-on-path", Detail: "justz is not on the PATH, so the shims cannot start it"})
	} else {
		out = append(out, Finding{Check: "justz-on-path", OK: true, Detail: p})
	}

	if len(reg.Shims) > 0 {
		if d.ShimDirOnPath() {
			out = append(out, Finding{Check: "shim-dir-on-path", OK: true, Detail: d.Shim})
		} else {
			out = append(out, Finding{Check: "shim-dir-on-path", Detail: fmt.Sprintf("%s is not on the PATH, so the shims will not be found", d.Shim)})
		}
	}

	for _, e := range reg.Shims {
		if info, err := os.Stat(e.Justfile); err != nil || info.IsDir() {
			out = append(out, Finding{Shim: e.Name, Check: "justfile", Detail: fmt.Sprintf("%s does not exist; the project moved or was deleted (run: justz shim add <new path> --replace)", e.Justfile)})
		} else {
			out = append(out, Finding{Shim: e.Name, Check: "justfile", OK: true, Detail: e.Justfile})
		}
		for _, n := range []string{e.Name, e.HelpName()} {
			out = append(out, d.checkStub(e, n))
		}
	}
	return out, nil
}

func (d Dirs) checkStub(e Entry, n string) Finding {
	f := Finding{Shim: e.Name, Check: "command " + n}
	sp := d.stubPath(n)
	switch {
	case !exists(sp):
		f.Detail = fmt.Sprintf("the stub %s is missing (run: justz shim add %s --name %s)", sp, e.Justfile, e.Name)
		return f
	case !isOurs(sp):
		f.Detail = fmt.Sprintf("%s is no longer a justz stub; something replaced it", sp)
		return f
	}
	first, _ := d.firstExecutable(n)
	switch {
	case first == "":
		f.Detail = fmt.Sprintf("%s exists but the shell would not find %q on the PATH", sp, n)
	case !sameFile(first, sp):
		f.Detail = fmt.Sprintf("%s comes first on the PATH, so the shim %s never runs", first, n)
	default:
		f.OK = true
		f.Detail = sp
	}
	return f
}

// firstExecutable is like firstOnPath but does not skip justz stubs: it is the
// command a shell would really run.
func (d Dirs) firstExecutable(name string) (string, int) {
	for i, dir := range d.Path {
		if dir == "" {
			continue
		}
		for _, cand := range d.candidates(dir, name) {
			if isExecutable(cand, d.GOOS) {
				return cand, i
			}
		}
	}
	return "", -1
}

func sameFile(a, b string) bool {
	ia, err1 := os.Stat(a)
	ib, err2 := os.Stat(b)
	return err1 == nil && err2 == nil && os.SameFile(ia, ib)
}

// Healthy reports whether every finding is ok.
func Healthy(findings []Finding) bool {
	for _, f := range findings {
		if !f.OK {
			return false
		}
	}
	return true
}

// CheckAliases looks at the jz and jzh names installed next to the justz binary
// at self (for example by the Homebrew formula) and reports, for each one that
// exists, whether it is the command a shell finds first. An older script of the
// same name earlier on the PATH would hide it.
func (d Dirs) CheckAliases(self string) []Finding {
	var out []Finding
	dir := filepath.Dir(self)
	for _, n := range []string{"jz", "jzh"} {
		ours := filepath.Join(dir, n)
		if !exists(ours) {
			continue
		}
		f := Finding{Check: "command " + n}
		switch first, _ := d.firstExecutable(n); {
		case first == "":
			f.Detail = fmt.Sprintf("%s is installed but %s is not on the PATH", ours, dir)
		case !sameFile(first, ours):
			f.Detail = fmt.Sprintf("%s comes first on the PATH and hides the %s that justz installed (%s); remove or rename it", first, n, ours)
		default:
			f.OK = true
			f.Detail = first
		}
		out = append(out, f)
	}
	return out
}
