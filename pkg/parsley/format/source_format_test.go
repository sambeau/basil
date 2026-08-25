package format

import (
	"strings"
	"testing"
)

// TestFormatSource_Idempotent verifies that formatting already-canonical source
// leaves it byte-for-byte unchanged (and that a second pass is a no-op).
func TestFormatSource_Idempotent(t *testing.T) {
	inputs := []string{
		"let x = 5\n",
		"let name = \"Alice\"\n",
		"let arr = [1, 2, 3]\n",
	}
	for _, in := range inputs {
		t.Run(strings.TrimSpace(in), func(t *testing.T) {
			once, err := FormatSource("x.pars", in)
			if err != nil {
				t.Fatalf("FormatSource returned error: %v", err)
			}
			if once != in {
				t.Errorf("expected already-formatted source unchanged\n got: %q\nwant: %q", once, in)
			}
			twice, err := FormatSource("x.pars", once)
			if err != nil {
				t.Fatalf("second FormatSource returned error: %v", err)
			}
			if twice != once {
				t.Errorf("formatting is not idempotent\n first: %q\nsecond: %q", once, twice)
			}
		})
	}
}

// TestFormatSource_Reformats verifies that messy source is rewritten and that
// the result is itself canonical (idempotent).
func TestFormatSource_Reformats(t *testing.T) {
	messy := "let    x    =    5"
	got, err := FormatSource("x.pars", messy)
	if err != nil {
		t.Fatalf("FormatSource returned error: %v", err)
	}
	if got == messy {
		t.Fatalf("expected messy source to be reformatted, but it was unchanged: %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("expected trailing newline, got %q", got)
	}
	again, err := FormatSource("x.pars", got)
	if err != nil {
		t.Fatalf("re-formatting returned error: %v", err)
	}
	if again != got {
		t.Errorf("reformatted output is not canonical\n first: %q\nsecond: %q", got, again)
	}
}

// TestFormatSource_TrailingNewline verifies that a canonical body missing its
// trailing newline gains exactly one, and no more are added on re-run.
func TestFormatSource_TrailingNewline(t *testing.T) {
	got, err := FormatSource("x.pars", "let x = 5")
	if err != nil {
		t.Fatalf("FormatSource returned error: %v", err)
	}
	if got != "let x = 5\n" {
		t.Errorf("expected single trailing newline, got %q", got)
	}
	again, err := FormatSource("x.pars", got)
	if err != nil {
		t.Fatalf("second FormatSource returned error: %v", err)
	}
	if again != got {
		t.Errorf("trailing newline not stable: %q -> %q", got, again)
	}
}

// TestFormatSource_ParseError verifies a parse failure returns a structured
// error carrying file:line information and no formatted output.
func TestFormatSource_ParseError(t *testing.T) {
	// An unterminated expression is a parse error.
	got, err := FormatSource("broken.pars", "let x = (1 + ")
	if err == nil {
		t.Fatalf("expected a parse error, got formatted output %q", got)
	}
	if got != "" {
		t.Errorf("expected empty output on parse error, got %q", got)
	}
	pe, ok := AsParsleyError(err)
	if !ok {
		t.Fatalf("expected *errors.ParsleyError, got %T: %v", err, err)
	}
	if pe.Line <= 0 {
		t.Errorf("expected a positive line number for file:line reporting, got %d", pe.Line)
	}
}

// TestIsFormatted verifies the convenience predicate for both canonical and
// non-canonical source, and that a parse error propagates.
func TestIsFormatted(t *testing.T) {
	ok, err := IsFormatted("x.pars", "let x = 5\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Errorf("expected canonical source to report formatted")
	}

	ok, err = IsFormatted("x.pars", "let    x = 5\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Errorf("expected messy source to report NOT formatted")
	}

	if _, err := IsFormatted("broken.pars", "let x = (1 + "); err == nil {
		t.Errorf("expected parse error to propagate from IsFormatted")
	}
}

// TestDiff verifies the diff header and that identical input yields only the
// header (no -/+ lines).
func TestDiff(t *testing.T) {
	empty := Diff("x.pars", "let x = 5\n", "let x = 5\n")
	if empty != "diff x.pars\n" {
		t.Errorf("expected only a header for identical input, got %q", empty)
	}

	d := Diff("x.pars", "let x=5\n", "let x = 5\n")
	if !strings.HasPrefix(d, "diff x.pars\n") {
		t.Errorf("expected diff header, got %q", d)
	}
	if !strings.Contains(d, "-1: let x=5") || !strings.Contains(d, "+1: let x = 5") {
		t.Errorf("expected -/+ lines for the changed line, got %q", d)
	}
}
