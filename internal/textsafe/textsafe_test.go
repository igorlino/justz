package textsafe

import "testing"

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
		if r != '�' && (r < 0x20 || dangerous(r)) {
			t.Fatalf("U+%04X is still in the output", r)
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
