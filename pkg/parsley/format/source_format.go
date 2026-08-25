// Package format provides AST-based formatting for Parsley source code.
// This file lifts the read-lex-parse-format pipeline out of the pars/basil
// command binaries so both share ONE implementation of "format a source file".
package format

import (
	"fmt"
	"strings"

	perrors "github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// FormatSource lexes and parses source, then returns the canonically
// formatted equivalent. filename is used only for diagnostics (it names the
// lexer so token positions carry it).
//
// If parsing fails, FormatSource returns the first structured parse error as a
// *errors.ParsleyError (with Line/Column populated) so callers can report
// file:line; the caller already knows the filename it passed in. On success the
// result is FormatProgram's output with a single trailing newline guaranteed.
//
// The transformation is exactly what `pars fmt` performed inline, so formatting
// is unchanged across the two binaries.
func FormatSource(filename, source string) (string, error) {
	l := lexer.NewWithFilename(source, filename)
	p := parser.New(l)

	program := p.ParseProgram()
	if errs := p.StructuredErrors(); len(errs) != 0 {
		// The parser only ever records the first error (later ones are
		// cascading noise), so errs[0] is the whole story.
		return "", errs[0]
	}

	formatted := FormatProgram(program)

	// Ensure the result ends with exactly one trailing newline.
	if !strings.HasSuffix(formatted, "\n") {
		formatted += "\n"
	}

	return formatted, nil
}

// IsFormatted reports whether source is already in canonical form, i.e.
// FormatSource(filename, source) == source. A parse error is surfaced (and
// reported as not-formatted) rather than swallowed.
func IsFormatted(filename, source string) (bool, error) {
	formatted, err := FormatSource(filename, source)
	if err != nil {
		return false, err
	}
	return formatted == source, nil
}

// AsParsleyError type-asserts an error returned by FormatSource/IsFormatted back
// to the structured *errors.ParsleyError it always is, for callers that want the
// Line/Column fields. The second return is false for a nil or non-structured
// error.
func AsParsleyError(err error) (*perrors.ParsleyError, bool) {
	pe, ok := err.(*perrors.ParsleyError)
	return pe, ok
}

// Diff renders a simple, line-numbered diff between original and formatted
// source. It is deliberately the same hand-rolled format `pars fmt -d` has
// always printed (a "diff <file>" header followed by -/+ lines), lifted here so
// both binaries emit identical diffs. The result always contains at least the
// "diff <file>" header line (each line it emits ends with a newline), followed
// by a -/+ pair for every line that differs; when the inputs are identical it
// contains only that header line.
func Diff(filename, original, formatted string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "diff %s\n", filename)

	origLines := strings.Split(original, "\n")
	fmtLines := strings.Split(formatted, "\n")

	maxLines := len(fmtLines)
	if len(origLines) > maxLines {
		maxLines = len(origLines)
	}

	for i := 0; i < maxLines; i++ {
		origLine := ""
		fmtLine := ""
		if i < len(origLines) {
			origLine = origLines[i]
		}
		if i < len(fmtLines) {
			fmtLine = fmtLines[i]
		}

		if origLine != fmtLine {
			if origLine != "" {
				fmt.Fprintf(&sb, "-%d: %s\n", i+1, origLine)
			}
			if fmtLine != "" {
				fmt.Fprintf(&sb, "+%d: %s\n", i+1, fmtLine)
			}
		}
	}

	return sb.String()
}
