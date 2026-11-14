package complete

import "strings"

// justzCommands are the commands of justz, in the order they are offered.
var justzCommands = []Candidate{
	{"assess", "Report facts about the justfile and which variant fits"},
	{"check", "Check how the project documents its commands"},
	{"completions", "Print a shell completion script"},
	{"doctor", "Check that justz can do its job"},
	{"fmt", "Rewrite the command registry in canonical order"},
	{"generate", "Build the personal .just-local-rich.json"},
	{"help", "Show the project's commands"},
	{"init", "Scaffold a justfile that documents itself"},
	{"lint", "Find antipatterns in the recipes, as advice"},
	{"resolve", "Print the group path a bare command name stands for"},
	{"setup", "Get this machine ready and check it"},
	{"shim", "Manage shims"},
	{"--help", "Print the usage"},
	{"--version", "Print the version"},
}

var shimCommands = []Candidate{
	{"add", "Register a shim for a project"},
	{"doctor", "Check every registered shim"},
	{"list", "List registered shims"},
	{"remove", "Remove a shim and its stubs"},
}

var jsonFlag = Candidate{"--json", "Print JSON"}

// justzFlags are the flags of each command that has some.
var justzFlags = map[string][]Candidate{
	"assess":   {jsonFlag},
	"check":    {jsonFlag, {"--strict", "Warnings fail the check too"}},
	"doctor":   {jsonFlag},
	"fmt":      {{"--check", "Do not write; exit 1 if not formatted"}},
	"generate": {{"--stdout", "Print the result and write nothing"}},
	"init": {
		{"--dry-run", "List the files and write nothing"},
		{"--module", "A command group (repeat the flag)"},
		{"--name", "The heading of the help"},
		{"--variant", "native or rich"},
	},
	"lint":    {jsonFlag, {"--strict", "Warnings fail the run too"}},
	"resolve": {jsonFlag},
	"setup":   {{"--dry-run", "Show what would be created"}, jsonFlag},
	"help": {
		{"--browse", "Pick a group or command"},
		{"--expanded", "Show the full reference"},
		jsonFlag,
		{"--wide", "Add example columns"},
		{"-b", "Pick a group or command"},
		{"-j", "Print JSON"},
		{"-w", "Add example columns"},
	},
	"shim add": {
		{"--dry-run", "Show what would happen"},
		{"--force", "Allow a shim an earlier command would shadow"},
		jsonFlag,
		{"--name", "Use this shim name"},
		{"--replace", "Take a name over, or hide a later command"},
		{"--soft", "Fall back to plain just"},
		{"--suffix", "Help command suffix"},
	},
	"shim doctor": {jsonFlag},
	"shim list":   {jsonFlag},
	"shim remove": {{"--dry-run", "Show what would be removed"}, jsonFlag},
}

// Justz completes the words typed after justz.
func Justz(words []string, e Env) Result {
	if len(words) == 0 {
		words = []string{""}
	}
	cur := words[len(words)-1]
	prior := words[:len(words)-1]

	if len(prior) == 0 {
		return Result{Candidates: filter(justzCommands, cur)}
	}

	cmd := prior[0]
	if cmd == "shim" {
		if len(prior) == 1 {
			return Result{Candidates: filter(shimCommands, cur)}
		}
		cmd = "shim " + prior[1]
		prior = prior[1:] // so the rest is relative to the subcommand
	}
	rest := prior[1:]

	flags := justzFlags[cmd]
	if strings.HasPrefix(cur, "-") {
		return Result{Candidates: filter(flags, cur)}
	}

	switch cmd {
	case "completions":
		if len(rest) == 0 {
			return Result{Candidates: filter([]Candidate{{"bash", "Bash completion script"}, {"zsh", "Zsh completion script"}}, cur)}
		}
		return Result{}
	case "fmt", "generate", "init", "shim add":
		// A file or directory, and the flags once a dash is typed.
		return Result{Files: true}
	case "shim remove":
		var cands []Candidate
		for _, n := range e.Shims() {
			cands = append(cands, Candidate{n, "shim"})
		}
		return Result{Candidates: filter(cands, cur)}
	case "resolve":
		if proj := e.Project(); proj != nil {
			return Result{Candidates: filter(bareNames(proj), cur)}
		}
		return Result{}
	case "help":
		if proj := e.Project(); proj != nil {
			var terms []string
			for _, w := range rest {
				if !strings.HasPrefix(w, "-") {
					terms = append(terms, w)
				}
			}
			switch len(terms) {
			case 0:
				return Result{Candidates: filter(groupAndCommandCandidates(proj, cur), cur)}
			case 1:
				return Result{Candidates: filter(commandsOfGroup(proj, terms[0]), cur)}
			}
		}
		return Result{}
	}
	return Result{}
}
