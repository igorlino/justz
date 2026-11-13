package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/igorlino/justz/internal/help"
)

// browse lets a person pick a group or command and shows it. It never prompts
// without a terminal: then it prints the overview of groups and exits 0.
func browse(proj *help.Project, opts help.Options, asJSON bool, e env) int {
	if !e.interactive {
		if asJSON {
			return printJSON(e, help.OverviewJSON(proj))
		}
		fmt.Fprint(e.stdout, help.Overview(proj, opts))
		return 0
	}

	rows := proj.Rows()
	if len(rows) == 0 {
		fmt.Fprintln(e.stderr, "justz help: this project has no commands to browse")
		return 1
	}

	var path string
	var err error
	if fzf, lookErr := e.lookPath("fzf"); lookErr == nil {
		self, _ := os.Executable()
		lines := make([]string, len(rows))
		for i, r := range rows {
			lines[i] = r.Path() + "\t" + proj.CLI + " " + r.Path() + " — " + r.Desc
		}
		var chosen string
		if chosen, err = e.runFzf(fzf, self, lines); err == nil {
			path = strings.SplitN(chosen, "\t", 2)[0]
		}
	} else {
		fmt.Fprintln(e.stderr, "justz help: fzf was not found, using a simple menu (install fzf for a searchable picker with a preview)")
		path, err = pickByNumber(rows, proj, e)
	}
	if err != nil {
		fmt.Fprintf(e.stderr, "justz help: %v\n", err)
		return 1
	}
	if path == "" {
		return 0 // cancelled
	}

	terms := strings.Fields(path)
	if asJSON {
		return zoomJSON(proj, terms, e)
	}
	return zoom(proj, terms, opts, e)
}

// pickByNumber shows a numbered menu and reads the choice from stdin. An empty
// answer, "q" or the end of input cancels.
func pickByNumber(rows []help.Row, proj *help.Project, e env) (string, error) {
	for i, r := range rows {
		fmt.Fprintf(e.stderr, "%3d) %s %s — %s\n", i+1, proj.CLI, r.Path(), r.Desc)
	}
	fmt.Fprint(e.stderr, "Select a number (q to quit): ")
	line, _ := bufio.NewReader(e.stdin).ReadString('\n') // the end of input arrives as an empty answer
	answer := strings.TrimSpace(line)
	if answer == "" || strings.EqualFold(answer, "q") {
		return "", nil
	}
	n, convErr := strconv.Atoi(answer)
	if convErr != nil || n < 1 || n > len(rows) {
		return "", fmt.Errorf("%q is not a number between 1 and %d", answer, len(rows))
	}
	return rows[n-1].Path(), nil
}

// runFzf shows lines of "path<TAB>label" in fzf, searching and showing the
// label, and previews the highlighted entry by running justz help on its path.
//
// The preview command is a fixed string. The binary is passed in the
// environment, and the entry reaches it through fzf's own quoting of {1}, so
// nothing from the registry is ever put into a shell string by justz.
func runFzf(fzf, self string, lines []string) (string, error) {
	cmd := exec.Command(fzf,
		"--delimiter=\t",
		"--with-nth=2",
		"--ansi",
		"--preview", `"$JUSTZ_BIN" help {1}`,
		"--preview-window=right:60%:wrap",
	)
	cmd.Env = append(os.Environ(), "JUSTZ_BIN="+self)
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n") + "\n")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		// 130 is the user pressing escape, 1 means no match: both are a cancel.
		if ee, ok := err.(*exec.ExitError); ok && (ee.ExitCode() == 130 || ee.ExitCode() == 1) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}
