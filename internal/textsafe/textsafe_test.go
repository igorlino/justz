package textsafe

import (
	"bytes"
	"testing"
)

func TestTextAndLine(t *testing.T) {
	tests := []struct {
		name, in, text, line string
	}{
		{"plain", "Run the tests", "Run the tests", "Run the tests"},
		{"empty", "", "", ""},
		{"sgr", "a \x1b[31mRED\x1b[0m b", "a �[31mRED�[0m b", "a �[31mRED�[0m b"},
		{"osc title", "x\x1b]0;t\x07y", "x�]0;t�y", "x�]0;t�y"},
		{"nul and bell", "a\x00b\x07c", "a�b�c", "a�b�c"},
		{"backspace and form feed", "a\bb\fc\vd", "a�b�c�d", "a�b�c�d"},
		{"lone carriage return", "C\rX", "C�X", "C�X"},
		{"crlf", "one\r\ntwo", "one\ntwo", "one two"},
		{"newline and tab", "a\n b\tc", "a\n b\tc", "a  b c"},
		{"del", "D\x7f", "D�", "D�"},
		{"c1 csi", "x\u009by", "x�y", "x�y"},
		{"c1 nel", "x\u0085y", "x�y", "x�y"},
		{"line separators", "a b c", "a�b�c", "a�b�c"},
		{"bidi override", "a‮b", "a�b", "a�b"},
		{"bidi isolate", "a⁦b⁩", "a�b�", "a�b�"},
		{"deprecated format", "a⁪b", "a�b", "a�b"},
		{"tag characters", "a\U000e0041b", "a�b", "a�b"},
		{"invalid utf-8", "a\xffb", "a�b", "a�b"},
		// Text that must survive.
		{"cjk and emoji", "日本語 🚀", "日本語 🚀", "日本語 🚀"},
		{"zwj emoji", "👩‍💻", "👩‍💻", "👩‍💻"},
		{"zero width and bom kept", "a\u200bb\u200cc\u2060d\ufeffe", "a\u200bb\u200cc\u2060d\ufeffe", "a\u200bb\u200cc\u2060d\ufeffe"},
		{"rtl marks kept", "a‎b‏c؜d", "a‎b‏c؜d", "a‎b‏c؜d"},
		{"arabic and hebrew", "مرحبا שלום", "مرحبا שלום", "مرحبا שלום"},
		{"variation selector kept", "❤️", "❤️", "❤️"},
		{"em dash and quotes", "a — “b”", "a — “b”", "a — “b”"},
		{"combining mark", "é", "é", "é"},
	}
	for _, tt := range tests {
		if got := Text(tt.in); got != tt.text {
			t.Errorf("%s: Text(%q) = %q, want %q", tt.name, tt.in, got, tt.text)
		}
		if got := Line(tt.in); got != tt.line {
			t.Errorf("%s: Line(%q) = %q, want %q", tt.name, tt.in, got, tt.line)
		}
	}
}

func TestCleanTextIsReturnedWithoutACopy(t *testing.T) {
	s := "Run the tests, then the rest — 日本語"
	if allocs := testing.AllocsPerRun(100, func() { _ = Text(s) }); allocs != 0 {
		t.Errorf("Text allocates %v times on clean text", allocs)
	}
}

// TestOutputHasNothingDangerous checks every rune against ranges written out
// here, not against dangerous itself, so a rune dropped from the set fails.
func TestOutputHasNothingDangerous(t *testing.T) {
	var all []rune
	for r := rune(0); r < 0x110000; r++ {
		if r >= 0xd800 && r <= 0xdfff {
			continue
		}
		all = append(all, r)
	}
	out := Line(string(all))
	for _, r := range out {
		switch {
		case r == '\ufffd':
		case r < 0x20, r >= 0x7f && r <= 0x9f,
			r == 0x2028, r == 0x2029,
			r >= 0x202a && r <= 0x202e,
			r >= 0x2066 && r <= 0x2069,
			r >= 0x206a && r <= 0x206f,
			r >= 0xe0000 && r <= 0xe007f:
			t.Fatalf("U+%04X is still in the output", r)
		}
	}
}

func TestWriterCleansWhatItPassesOn(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	in := "path \x1b]0;T\x07 ok\r\nnext\n"
	n, err := w.Write([]byte(in))
	if err != nil || n != len(in) {
		t.Fatalf("Write = %d, %v; want %d, nil", n, err, len(in))
	}
	if got, want := buf.String(), "path �]0;T� ok\nnext\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIsValue(t *testing.T) {
	for _, s := range []string{"a", "prod-1", "v1.2.3", "a/b", "k=v", "~x", "a,b", "10%", "x:y", "u@h", "+1"} {
		if !IsValue(s) {
			t.Errorf("IsValue(%q) = false", s)
		}
	}
	for _, s := range []string{"", "a b", "a;b", "$(id)", "`x`", "a|b", "a\nb", "a\"b", "a'b", "a&b", "a>b", "*", "日本", "a\x1b"} {
		if IsValue(s) {
			t.Errorf("IsValue(%q) = true", s)
		}
	}
}

func TestIsName(t *testing.T) {
	for _, ok := range []string{"git", "tool", "ewf-deploy", "a_b", "v1.2", "_x", "A9"} {
		if !IsName(ok) {
			t.Errorf("IsName(%q) = false", ok)
		}
	}
	for _, bad := range []string{"", "-x", ".x", "a b", "a;b", "a\x1b[31m", "manifest [n]", "a/b", "日本", "a\nb", "$(x)", "a|b"} {
		if IsName(bad) {
			t.Errorf("IsName(%q) = true", bad)
		}
	}
}
