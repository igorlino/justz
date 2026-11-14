package lint

import (
	"fmt"
	"strings"
	"testing"
)

func TestSecretLikeNames(t *testing.T) {
	yes := []string{"API_TOKEN", "apiToken", "api-token", "authtoken", "dbpassword", "dbPass", "SSH_PASS", "pw",
		"SSH_KEY_PASSPHRASE", "APIKEY", "SIGNING_KEY", "SECRETS", "GH_TOKEN", "DB_PASSWORD", "X_SECRET", "TESTS_PASS",
		"MYSQL_PWD", "PASSWORDS", "ENCRYPTION_KEY", "MASTER_KEY", "PRIVATEKEY", "SECRETKEY", "TOKEN2", "dbPassword2",
		"AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN", "PGPASSWORD", "REDIS_PASS", "OP_SERVICE_ACCOUNT_TOKEN",
		"ANTHROPIC_API_KEY", "STRIPE_SECRET_KEY", "SECRET_KEY_BASE", "PRIVATE_KEY", "api_key", "x-api-key", "passwordHash"}
	no := []string{"SSH_KEY_FILE", "TOKEN_PATH", "GITHUB_TOKEN_FILE", "AWS_ACCESS_KEY_ID", "TOKEN_TTL", "TOKEN_ENDPOINT",
		"PASSWORD_LENGTH", "OP_TOKEN_REF", "SECRET_ARN", "PASS_THROUGH_ARGS", "TOKENIZER", "PASSTHROUGH", "BYPASS",
		"COMPASS", "KEYCLOAK_URL", "KEY", "", "_", "name", "args", "TOKENS_PER_MINUTE", "PASSWORD_POLICY", "PASSWORD_PROMPT",
		"TOKEN_EXPIRES", "TOKEN_EXPIRY", "SECRET_MANAGER", "SECRET_VERSION", "TOKEN_VAR", "PASSWORD_FILE", "SECRETS_DIR",
		"PASSWORD_RESET_URL", "CREDENTIALS_FILE", "KUBECONFIG", "PRIVATE_KEY_FILE",
		// accepted misses: no name list finds these without flagging far more
		"GH_PAT", "DATABASE_URL", "VAULT_SECRET_ID", "GITHUB_PAT", "DOCKER_AUTH_CONFIG", "BASIC_AUTH", "Authorization", "bearer", "SSH_KEY"}
	for _, n := range yes {
		if !SecretLike(n) {
			t.Errorf("%q should be secret-like", n)
		}
	}
	for _, n := range no {
		if SecretLike(n) {
			t.Errorf("%q should not be secret-like", n)
		}
	}
}

// fixture builds a dump with one recipe: its parameters, its body lines, whether
// it is a shebang recipe, and the assignments of the justfile.
func fixture(params, body string, shebang bool, assignments string, attrs string) string {
	return fmt.Sprintf(`{"assignments":{%s},"recipes":{"r":{"attributes":[%s],"parameters":[%s],"body":[%s],"shebang":%v}},"modules":{}}`,
		assignments, attrs, params, body, shebang)
}

func param(name, def string) string {
	return fmt.Sprintf(`{"name":%q,"default":%s}`, name, def)
}

func assign(name, value string) string {
	return fmt.Sprintf(`%q:{"value":%s}`, name, value)
}

func got(t *testing.T, d string) string {
	t.Helper()
	return strings.Join(rules(run(t, d)), ",")
}

func TestSecretParameter(t *testing.T) {
	if g := got(t, fixture(param("api-token", "null")+","+param("host", "null")+","+param("token_file", "null"), "", false, "", "")); g != "secret-parameter just r" {
		t.Errorf("findings = %q", g)
	}
	res := run(t, fixture(param("db_password", "null"), "", false, "", ""))
	f := res.Findings[0]
	if !strings.Contains(f.Message, `"db_password"`) || f.Line != 0 || f.Severity != "warning" {
		t.Errorf("finding = %+v", f)
	}
}

func TestSecretParameterAppliesToShebangRecipes(t *testing.T) {
	if g := got(t, fixture(param("secret", "null"), `["#!/bin/sh"]`, true, "", "")); g != "secret-parameter just r" {
		t.Errorf("findings = %q", g)
	}
}

func TestSecretInterpolatedShapes(t *testing.T) {
	tests := []struct {
		name, params, body, assigns string
		want                        string // the finding names, or ""
	}{
		{"env call", "", `["@echo ",[["call","env","GH_TOKEN"]]]`, "", "GH_TOKEN"},
		{"env_var", "", `["@echo ",[["call","env_var","DB_PASSWORD"]]]`, "", "DB_PASSWORD"},
		{"env_var_or_default", "", `["@echo ",[["call","env_var_or_default","X_SECRET","d"]]]`, "", "X_SECRET"},
		{"nested in a call", "", `["@echo ",[["call","quote",["call","env","GH_TOKEN"]]]]`, "", "GH_TOKEN"},
		{"concatenation", "", `["@echo ",[["concatenate","x",["call","env","GH_TOKEN"]]]]`, "", "GH_TOKEN"},
		{"if branch", "", `["@echo ",[["if",["==",["call","env","A"],""],"a",["call","env","GH_TOKEN"]]]]`, "", "GH_TOKEN"},
		{"secret-like assignment", "", `["@echo ",[["variable","gh_token"]]]`, assign("gh_token", `"x"`), "gh_token"},
		{"alias of env", "", `["@echo ",[["variable","tok"]]]`, assign("tok", `["call","env","GH_TOKEN",""]`), "GH_TOKEN"},
		{"alias chain", "", `["@echo ",[["variable","b"]]]`, assign("b", `["variable","a"]`) + "," + assign("a", `["call","env","GH_TOKEN"]`), "GH_TOKEN"},
		{"alias cycle does not hang", "", `["@echo ",[["variable","a"]]]`, assign("a", `["variable","b"]`) + "," + assign("b", `["variable","a"]`), ""},
		{"parameter default from env", param("tok", `["call","env","GH_TOKEN"]`), `["@echo ",[["variable","tok"]]]`, "", "GH_TOKEN"},
		{"two names on one line", "", `["@echo ",[["call","env","GH_TOKEN"]]," ",[["call","env","DB_PASSWORD"]]]`, "", "DB_PASSWORD, GH_TOKEN"},
		{"the same name twice is listed once", "", `["@echo ",[["call","env","GH_TOKEN"]]," ",[["call","env","GH_TOKEN"]]]`, "", "GH_TOKEN"},
		{"comment lines are interpolated by just", "", `["# note ",[["call","env","GH_TOKEN"]]]`, "", "GH_TOKEN"},
		{"a path is not a secret", "", `["@cat ",[["call","env","TOKEN_FILE"]]]`, "", ""},
		{"a plain variable", "", `["@echo ",[["variable","host"]]]`, assign("host", `"h"`), ""},
		{"a parameter that is not secret-like", param("host", "null"), `["@echo ",[["variable","host"]]]`, "", ""},
		{"text mentioning a secret is not an interpolation", "", `["@echo API_TOKEN"]`, "", ""},
		{"env with a non-literal name", "", `["@echo ",[["call","env",["variable","n"]]]]`, assign("n", `"GH_TOKEN"`), ""},
		{"backtick", "", `["@echo ",[["evaluate","cmd"]]]`, "", ""},
	}
	for _, tt := range tests {
		res := run(t, fixture(tt.params, tt.body, false, tt.assigns, ""))
		var found []string
		for _, f := range res.Findings {
			if f.Rule == RuleSecretInterpol {
				found = append(found, f.Message)
				if f.Line != 1 {
					t.Errorf("%s: line = %d, want 1", tt.name, f.Line)
				}
			}
		}
		if tt.want == "" {
			if len(found) != 0 {
				t.Errorf("%s: unexpected findings %v", tt.name, found)
			}
			continue
		}
		if len(found) != 1 || !strings.Contains(found[0], "interpolates "+tt.want+" into") {
			t.Errorf("%s: findings = %v, want names %q", tt.name, found, tt.want)
		}
	}
}

func TestSecretInterpolatedReportsTheBodyLineNumber(t *testing.T) {
	body := `["@echo one"],["@echo ",[["call","env","GH_TOKEN"]]]`
	res := run(t, fixture("", body, false, "", ""))
	if len(res.Findings) != 1 || res.Findings[0].Line != 2 {
		t.Errorf("findings = %+v", res.Findings)
	}
}

func TestSecretInterpolatedSkipsShebangRecipes(t *testing.T) {
	if g := got(t, fixture("", `["#!/bin/sh"],["echo ",[["call","env","GH_TOKEN"]]]`, true, "", "")); g != "" {
		t.Errorf("findings = %q", g)
	}
}

func TestOwnSecretParameterIsReportedOnce(t *testing.T) {
	// The parameter finding covers its use in the body; no second finding for it.
	g := got(t, fixture(param("api-token", "null"), `["@echo ",[["variable","api-token"]]]`, false, "", ""))
	if g != "secret-parameter just r" {
		t.Errorf("findings = %q", g)
	}
}

func TestSecretRulesCanBeIgnoredPerRecipe(t *testing.T) {
	params := param("api-token", "null")
	body := `["@echo ",[["call","env","GH_TOKEN"]]]`
	for _, tt := range []struct{ attrs, want string }{
		{`{"metadata":["justz-ignore","secret-parameter"]}`, "secret-interpolated just r"},
		{`{"metadata":["justz-ignore","secret-interpolated"]}`, "secret-parameter just r"},
		{`{"metadata":["justz-ignore","secret-parameter","secret-interpolated"]}`, ""},
	} {
		if g := got(t, fixture(params, body, false, "", tt.attrs)); g != tt.want {
			t.Errorf("%s: findings = %q, want %q", tt.attrs, g, tt.want)
		}
	}
}

func TestMessagesNeverCarryValuesOrLineText(t *testing.T) {
	// A literal value assigned to a secret-like name must not be echoed.
	d := fixture("", `["@curl -H hunter2 ",[["variable","gh_token"]]]`, false, assign("gh_token", `"hunter2"`), "")
	for _, f := range run(t, d).Findings {
		if strings.Contains(f.Message+f.Fix, "hunter2") || strings.Contains(f.Message+f.Fix, "curl") {
			t.Errorf("message leaks: %+v", f)
		}
	}
}

func TestAssignmentsOfTheModuleAreUsedForItsRecipes(t *testing.T) {
	d := `{"assignments":{},"recipes":{},"modules":{"m":{"assignments":{"tok":{"value":["call","env","GH_TOKEN"]}},"settings":{"shell":{"command":"bash"}},"recipes":{"r":{"attributes":[],"parameters":[],"body":[["@echo ",[["variable","tok"]]]],"shebang":false}},"modules":{}}}}`
	if g := got(t, d); g != "secret-interpolated just m r" {
		t.Errorf("findings = %q", g)
	}
}

func TestIgnoringTheParameterRuleDoesNotHideItsInterpolation(t *testing.T) {
	// The interpolation rule must still fire: only the parameter finding is silenced.
	attrs := `{"metadata":["justz-ignore","secret-parameter"]}`
	g := got(t, fixture(param("token", "null"), `["@echo ",[["variable","token"]]]`, false, "", attrs))
	if g != "secret-interpolated just r" {
		t.Errorf("findings = %q", g)
	}
}

func TestParameterDefaultThatNamesAnAssignmentOfTheSameName(t *testing.T) {
	// host := env('GH_TOKEN') and  a host=host:  the default is the assignment.
	d := fixture(param("host", `["variable","host"]`), `["@echo ",[["variable","host"]]]`, false, assign("host", `["call","env","GH_TOKEN"]`), "")
	if g := got(t, d); g != "secret-interpolated just r" {
		t.Errorf("findings = %q", g)
	}
}

func TestOnlyTheBranchesOfAnIfCount(t *testing.T) {
	cond := `["@echo ",[["if",["==",["call","env","GH_TOKEN"],"x"],"a","b"]]]`
	if g := got(t, fixture("", cond, false, "", "")); g != "" {
		t.Errorf("a compared secret should not count: %q", g)
	}
	branch := `["@echo ",[["if",["==",["variable","h"],"x"],["call","env","GH_TOKEN"],"b"]]]`
	if g := got(t, fixture("", branch, false, assign("h", `"h"`), "")); g != "secret-interpolated just r" {
		t.Errorf("a secret in a branch should count: %q", g)
	}
}

func depDump(args string, targetParams string, targetBody string) string {
	return fmt.Sprintf(`{"assignments":{},"modules":{},"recipes":{
		"dep":{"attributes":[],"parameters":[],"body":[],"shebang":false,"dependencies":[{"recipe":"b","arguments":[%s]}]},
		"b":{"attributes":[],"parameters":[%s],"body":[%s],"shebang":false,"dependencies":[]}}}`, args, targetParams, targetBody)
}

func TestDependencyArgumentsFlowIntoTheReceivingRecipe(t *testing.T) {
	// dep: (b env('GH_TOKEN'))  with  b p:  @echo {{p}}: the value is in the line.
	d := depDump(`["call","env","GH_TOKEN"]`, param("p", "null"), `["@echo ",[["variable","p"]]]`)
	res := run(t, d)
	if got := strings.Join(rules(res), ","); got != "secret-interpolated just b" || !strings.Contains(res.Findings[0].Message, "GH_TOKEN") {
		t.Errorf("findings = %v", res.Findings)
	}
	// A literal argument carries nothing.
	if g := got(t, depDump(`"x"`, param("p", "null"), `["@echo ",[["variable","p"]]]`)); g != "" {
		t.Errorf("literal: %q", g)
	}
	// The second of two arguments, and a variadic receiver taking the rest.
	d = depDump(`"x",["call","env","Y_TOKEN"]`, param("a", "null")+","+param("rest", "null"), `["@echo ",[["variable","rest"]]]`)
	if g := got(t, d); g != "secret-interpolated just b" {
		t.Errorf("second argument: %q", g)
	}
	// An argument the receiver does not use in a line is not reported.
	if g := got(t, depDump(`["call","env","GH_TOKEN"]`, param("p", "null"), `["@true"]`)); g != "" {
		t.Errorf("unused: %q", g)
	}
}
