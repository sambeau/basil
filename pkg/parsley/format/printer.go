package format

import (
	"strings"
)

// Printer manages formatting state and output
type Printer struct {
	output  strings.Builder
	indent  int // Current indentation level (number of IndentWidth spaces)
	linePos int // Current position in the current line
}

// NewPrinter creates a new Printer instance
func NewPrinter() *Printer {
	return &Printer{}
}

// String returns the formatted output
func (p *Printer) String() string {
	return p.output.String()
}

// Reset clears the printer state for reuse
func (p *Printer) Reset() {
	p.output.Reset()
	p.indent = 0
	p.linePos = 0
}

// write appends a string to the output and updates line position
func (p *Printer) write(s string) {
	p.output.WriteString(s)
	// Update line position - handle embedded newlines
	if idx := strings.LastIndex(s, "\n"); idx >= 0 {
		p.linePos = len(s) - idx - 1
	} else {
		p.linePos += len(s)
	}
}

// writeln appends a string followed by a newline
func (p *Printer) writeln(s string) {
	p.write(s)
	p.newline()
}

// newline writes a newline character and resets line position
func (p *Printer) newline() {
	p.output.WriteString("\n")
	p.linePos = 0
}

// writeIndent writes the current indentation
func (p *Printer) writeIndent() {
	indent := strings.Repeat(IndentString, p.indent)
	p.write(indent)
}

// indentInc increases the indentation level
func (p *Printer) indentInc() {
	p.indent++
}

// indentDec decreases the indentation level
func (p *Printer) indentDec() {
	if p.indent > 0 {
		p.indent--
	}
}

// currentIndentWidth returns the current indentation width in characters
func (p *Printer) currentIndentWidth() int {
	return p.indent * IndentWidth
}

// fitsOnLine checks if a string would fit on the current line within the given threshold
func (p *Printer) fitsOnLine(s string, threshold int) bool {
	// Don't count embedded newlines - if string has newlines, it doesn't fit
	if strings.Contains(s, "\n") {
		return false
	}
	return p.linePos+len(s) <= threshold
}

// fitsInThreshold checks if a string fits within the given threshold from start of line
// (ignoring current position)
func fitsInThreshold(s string, threshold int) bool {
	if strings.Contains(s, "\n") {
		return false
	}
	return len(s) <= threshold
}

// wouldFitOnLine checks if a string would fit starting from indent position
func (p *Printer) wouldFitOnLine(s string, threshold int) bool {
	if strings.Contains(s, "\n") {
		return false
	}
	return p.currentIndentWidth()+len(s) <= threshold
}
