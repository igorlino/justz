package orderedjson

import (
	"strings"
	"testing"
)

func roundTrip(t *testing.T, in string) string {
	t.Helper()
	v, err := Parse([]byte(in))
	if err != nil {
		t.Fatalf("Parse(%q): %v", in, err)
	}
	out, err := Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return string(out)
}

func TestKeepsKeyOrder(t *testing.T) {
	in := "{\n  \"zeta\": 1,\n  \"alpha\": 2,\n  \"mid\": {\n    \"b\": true,\n    \"a\": false\n  }\n}\n"
	if got := roundTrip(t, in); got != in {
		t.Errorf("round trip changed the document:\n got: %q\nwant: %q", got, in)
	}
}

func TestKeepsNumberText(t *testing.T) {
	in := "[\n  1.0,\n  1e3,\n  -0,\n  12345678901234567890\n]\n"
	if got := roundTrip(t, in); got != in {
		t.Errorf("numbers changed:\n got: %q\nwant: %q", got, in)
	}
}

func TestScalarsAndEmptyContainers(t *testing.T) {
	in := "{\n  \"a\": null,\n  \"b\": [],\n  \"c\": {},\n  \"d\": \"x\",\n  \"e\": false\n}\n"
	if got := roundTrip(t, in); got != in {
		t.Errorf("got %q, want %q", got, in)
	}
}

func TestDoesNotEscapeHTMLCharacters(t *testing.T) {
	in := "[\n  \"a < b && c > d\"\n]\n"
	if got := roundTrip(t, in); got != in {
		t.Errorf("got %q, want %q", got, in)
	}
}

func TestNormalizesEscapesAndWhitespace(t *testing.T) {
	got := roundTrip(t, `{"k":"caf\u00e9 \n","a":[1,2]}`)
	want := "{\n  \"k\": \"café \\n\",\n  \"a\": [\n    1,\n    2\n  ]\n}\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGet(t *testing.T) {
	v, err := Parse([]byte(`{"a":{"b":"x"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := v.Get("a").Get("b"); got == nil || got.Str != "x" {
		t.Errorf("Get chain failed: %+v", got)
	}
	if v.Get("missing") != nil || v.Get("a").Get("b").Get("c") != nil {
		t.Error("Get on a missing key or a non-object should return nil")
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"duplicate key", `{"a":1,"a":2}`, "duplicate object key"},
		{"trailing data", `{"a":1} 2`, "unexpected data after"},
		{"empty input", ``, "unexpected end"},
		{"truncated object", `{"a":1`, ""},
		{"missing comma", `[1 2]`, ""},
		{"bare word", `nope`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.in))
			if err == nil {
				t.Fatalf("Parse(%q) should fail", tt.in)
			}
			if tt.want != "" && !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q should contain %q", err, tt.want)
			}
		})
	}
}
