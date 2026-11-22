package check

import (
	"strings"
	"testing"
)

func exampleRun(t *testing.T, cmds, examples string) Result {
	t.Helper()
	dump := dumpOf(t, map[string]any{}, map[string]any{"tool": module(map[string]any{
		"start": rc("Start", nil), "stop": rc("Stop", nil), "scan": rc("Scan", nil),
	})})
	s := `{"groups":{"tool":{"label":"T","commands":` + cmds + `,"examples":` + examples + `}}}`
	return run(t, Input{Dump: dump, Variant: "rich", Registry: parse(t, s), RegistrySource: []byte(s)})
}

const toolCmds = `{
	"start":{"desc":"Start","args":[{"name":"-p, --port <n>","desc":"Port"},{"name":"-v, --verbose","desc":"Chatty"}]},
	"stop":{"desc":"Stop"},
	"scan":{"desc":"Scan","args":[{"name":"-s, --silent","desc":"Quiet"}]}}`

func TestExampleFlagUnknown(t *testing.T) {
	res := exampleRun(t, toolCmds, `[{"cmd":"just tool start -p 1 --bogus"}]`)
	var got []Finding
	for _, f := range res.Findings {
		if f.Rule == RuleExampleFlagUnknown {
			got = append(got, f)
		}
	}
	if len(got) != 1 || got[0].Severity != Warning || got[0].Subject != "tool.examples" || !strings.Contains(got[0].Message, "--bogus") {
		t.Errorf("findings = %+v", got)
	}
}

func TestExampleFlagUnknownStaysQuiet(t *testing.T) {
	for _, ex := range []string{
		`just tool start --port 3 -v`,
		`just tool start --port=3 --verbose`,
		`just tool start -pv 3`,
		`just tool start -p -1`,
		`just tool start -p 3 -- --anything -x`,
		`just tool start -p '-x'`,
		`just tool stop --whatever`, // stop declares no rows
	} {
		res := exampleRun(t, toolCmds, `[{"cmd":"`+ex+`"}]`)
		if has(res, RuleExampleFlagUnknown, "") {
			t.Errorf("%s: %v", ex, rules(res))
		}
	}
}

func TestExampleEnvPrefixIsStripped(t *testing.T) {
	// The flag rules read the command through the prefix.
	for _, ex := range []string{`PORT=9000 just tool start --port 3 --nope`, `env PORT=9000 FOO="a b" just tool start --nope`} {
		res := exampleRun(t, toolCmds, `[{"cmd":"`+strings.ReplaceAll(ex, `"`, `\"`)+`"}]`)
		if !has(res, RuleExampleFlagUnknown, "tool.examples") {
			t.Errorf("%s: flag rule under an env prefix: %v", ex, rules(res))
		}
	}
	// A prefix is information the rows do not show: bare and single stay quiet.
	for _, ex := range []string{`PORT=9000 just tool scan`, `env PORT=9000 FOO="a b" just tool scan`, `PORT=9000 just tool scan -s`, `env PORT=9000 just tool scan -s`} {
		res := exampleRun(t, toolCmds, `[{"cmd":"`+strings.ReplaceAll(ex, `"`, `\"`)+`"}]`)
		if has(res, RuleExampleBare, "") || has(res, RuleExampleSingle, "") {
			t.Errorf("%s: bare/single under an env prefix: %v", ex, rules(res))
		}
	}
	// env with no assignment sets nothing: bare and single still fire.
	for _, ex := range []string{`env just tool scan`, `env -i just tool scan`} {
		if res := exampleRun(t, toolCmds, `[{"cmd":"`+ex+`"}]`); !has(res, RuleExampleBare, "tool.examples") {
			t.Errorf("%s: bare: %v", ex, rules(res))
		}
	}
	if res := exampleRun(t, toolCmds, `[{"cmd":"env -i just tool scan -s"}]`); !has(res, RuleExampleSingle, "tool.examples") {
		t.Errorf("env -i single: %v", rules(res))
	}
	if res := exampleRun(t, toolCmds, `[{"cmd":"env -i A=1 just tool scan"}]`); has(res, RuleExampleBare, "") {
		t.Errorf("env -i A=1: %v", rules(res))
	}
	// Without a prefix both still fire.
	if res := exampleRun(t, toolCmds, `[{"cmd":"just tool scan"}]`); !has(res, RuleExampleBare, "tool.examples") {
		t.Errorf("bare: %v", rules(res))
	}
	if res := exampleRun(t, toolCmds, `[{"cmd":"just tool scan -s"}]`); !has(res, RuleExampleSingle, "tool.examples") {
		t.Errorf("single: %v", rules(res))
	}
}

func TestCommandExampleFieldGoesThroughTheRules(t *testing.T) {
	cmds := `{"start":{"desc":"S","example":"… --nope","args":[{"name":"-p, --port <n>"}]},
		"scan":{"desc":"Sc","example":"just tool scan","args":[{"name":"-s, --silent"}]},
		"envbare":{"desc":"E","example":"PORT=1 just tool envbare"},
		"envone":{"desc":"E1","example":"env PORT=1 just tool envone -s","args":[{"name":"-s"}]},
		"stop":{"desc":"St","example":"… -s"}}`
	res := exampleRun(t, cmds, `[]`)
	if !has(res, RuleExampleFlagUnknown, "tool.start") {
		t.Errorf("unknown flag in example field: %v", rules(res))
	}
	if !has(res, RuleExampleBare, "tool.scan") {
		t.Errorf("bare example field: %v", rules(res))
	}
	if has(res, RuleExampleBare, "tool.envbare") || has(res, RuleExampleSingle, "tool.envone") {
		t.Errorf("env prefix in example field: %v", rules(res))
	}
	if has(res, RuleExampleFlagUnknown, "tool.stop") {
		t.Errorf("a command with no rows must stay quiet: %v", rules(res))
	}
	ok := exampleRun(t, `{"start":{"desc":"S","example":"… -p 3 -v","args":[{"name":"-p, --port <n>"},{"name":"-v"}]}}`, `[]`)
	for _, f := range ok.Findings {
		if strings.HasPrefix(f.Rule, "example-") {
			t.Errorf("valid example field reported: %+v", f)
		}
	}
}

func TestExampleFlagUnknownStopsAtShellOperators(t *testing.T) {
	for _, ex := range []string{
		`just tool start -p 3 | jq -r .`,
		`just tool start -p 3 ; ls -la`,
		`just tool start -p 3 && echo -x`,
		`just tool start -p 3 || echo -x`,
		`just tool start -p 3 # -zzz`,
		`just tool start -p 3 -- -zzz`,
	} {
		if res := exampleRun(t, toolCmds, `[{"cmd":"`+ex+`"}]`); has(res, RuleExampleFlagUnknown, "") {
			t.Errorf("%s: %v", ex, rules(res))
		}
	}
	// A flag before the operator is still checked.
	if res := exampleRun(t, toolCmds, `[{"cmd":"just tool start --bogus | jq -r ."}]`); !has(res, RuleExampleFlagUnknown, "") {
		t.Errorf("flag before a pipe: %v", rules(res))
	}
	// An escaped quote does not end a quoted value.
	if res := exampleRun(t, toolCmds, `[{"cmd":"just tool start -p \"a \\\" -x\""}]`); has(res, RuleExampleFlagUnknown, "") {
		t.Errorf("escaped quote: %v", rules(res))
	}
}

func TestExampleFlagUnknownNegatableRow(t *testing.T) {
	cmds := `{"start":{"desc":"S","args":[{"name":"--[no-]color","desc":"Colour"}]},"stop":{"desc":"x"},"scan":{"desc":"x"}}`
	for _, ex := range []string{`just tool start --color`, `just tool start --no-color`} {
		if res := exampleRun(t, cmds, `[{"cmd":"`+ex+`"}]`); has(res, RuleExampleFlagUnknown, "") {
			t.Errorf("%s: %v", ex, rules(res))
		}
	}
	if res := exampleRun(t, cmds, `[{"cmd":"just tool start --colour"}]`); !has(res, RuleExampleFlagUnknown, "") {
		t.Errorf("a misspelt flag: %v", rules(res))
	}
}

func TestStripEnv(t *testing.T) {
	for in, want := range map[string]string{
		`PORT=1 just x`:                   "just x",
		`env PORT=1 just x`:               "just x",
		`env -i A=b just x`:               "just x",
		`env -u HOME A=b just x`:          "just x",
		`A="a \"q\" b" B='c d' just x -v`: "just x -v",
		`just x A=1`:                      "just x A=1",
		`justx`:                           "justx",
	} {
		if got := stripEnv(in); got != want {
			t.Errorf("stripEnv(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExpandExample(t *testing.T) {
	for in, want := range map[string]string{
		`… -w`:           "just tool start -w",
		`... -w`:         "just tool start -w",
		`PORT=1 … -w`:    "PORT=1 just tool start -w",
		`PORT=1 ... -w`:  "PORT=1 just tool start -w",
		`just tool x -w`: "just tool x -w",
	} {
		if got := expandExample(in, "tool start"); got != want {
			t.Errorf("expandExample(%q) = %q, want %q", in, got, want)
		}
	}
}

// Without the example field the new coverage adds nothing: the findings of a
// registry that has only group examples are the ones the group rules give.
func TestGroupExamplesUnchangedWithoutExampleField(t *testing.T) {
	res := exampleRun(t, toolCmds, `[{"cmd":"just tool scan"},{"cmd":"just tool scan -s"},{"cmd":"just tool start --port 3 -v"}]`)
	var got []string
	for _, f := range res.Findings {
		if strings.HasPrefix(f.Rule, "example-") {
			got = append(got, f.Rule+":"+f.Subject)
		}
	}
	want := "example-bare-command:tool.examples,example-long-flag:tool.examples,example-single-flag:tool.examples"
	if strings.Join(got, ",") != want {
		t.Errorf("findings = %v, want %s", got, want)
	}
}

const boolCmds = `{
	"start":{"desc":"Start","args":[{"name":"[-tail]","desc":"From the end"},{"name":"[-n <lines>]","desc":"Lines"},{"name":"[--date=YYYY-MM-DD]","desc":"Day"},{"name":"[--only KIND]","desc":"Subset"},{"name":"-v, --verbose","desc":"Chatty"}]},
	"stop":{"desc":"Stop"},"scan":{"desc":"Scan"}}`

func TestExampleBooleanValueFires(t *testing.T) {
	for _, ex := range []string{`just tool start -tail=200`, `just tool start --verbose=true -n 3`} {
		res := exampleRun(t, boolCmds, `[{"cmd":"`+ex+`"}]`)
		f := hasRule(res, RuleExampleBoolValue)
		if f == nil || f.Severity != Warning || f.Subject != "tool.examples" {
			t.Errorf("%s: %v", ex, rules(res))
		}
	}
}

func TestExampleBooleanValueStaysQuiet(t *testing.T) {
	for _, ex := range []string{
		`just tool start -tail -n 200`,
		`just tool start -n=200`,            // <lines> declares a value
		`just tool start --date=2026-05-12`, // =YYYY-MM-DD declares a value
		`just tool start --only=python`,     // a bare word after the flag declares a value
		`just tool start -- -tail=200`,      // after the terminator
		`just tool start -tail 200`,         // 200 may be a positional argument
	} {
		if res := exampleRun(t, boolCmds, `[{"cmd":"`+ex+`"}]`); has(res, RuleExampleBoolValue, "") {
			t.Errorf("%s: %v", ex, rules(res))
		}
	}
}
