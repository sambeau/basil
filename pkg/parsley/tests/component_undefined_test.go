package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestUndefinedComponentErrors tests that using an undefined component
// produces a clear "Undefined component" error positioned at the component
// tag, with an import hint, instead of confusing parse errors (BUG-010).
func TestUndefinedComponentErrors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantLine int
	}{
		{
			name:     "paired component tag at top level",
			input:    "let x = 1\n<Page title=\"test\">\n  <p>Content</p>\n</Page>",
			wantName: "Page",
			wantLine: 2,
		},
		{
			name:     "self-closing component tag",
			input:    "let x = 1\n<Header/>",
			wantName: "Header",
			wantLine: 2,
		},
		{
			name:     "component inside a function body",
			input:    "let Home = fn(){\n  <Page title=\"test\">\n    <p>Content</p>\n  </Page>\n}\nHome()",
			wantName: "Page",
			wantLine: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			// The component tag must parse cleanly even though the name is
			// not in scope — undefined components are an eval-time error
			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			errObj, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected error, got %T: %v", result, result)
			}

			if errObj.Code != "UNDEF-0003" {
				t.Errorf("expected code UNDEF-0003, got %s", errObj.Code)
			}
			if !strings.Contains(errObj.Message, "Undefined component") ||
				!strings.Contains(errObj.Message, tt.wantName) {
				t.Errorf("expected 'Undefined component' naming %q, got: %s", tt.wantName, errObj.Message)
			}
			if errObj.Line != tt.wantLine {
				t.Errorf("wrong line: got %d, want %d", errObj.Line, tt.wantLine)
			}
			if errObj.Column < 1 {
				t.Errorf("column too low: got %d, want at least 1", errObj.Column)
			}
			if len(errObj.Hints) == 0 {
				t.Errorf("expected an import hint, got none")
			} else if !strings.Contains(errObj.Hints[0], "imported") {
				t.Errorf("expected hint to mention imports, got: %s", errObj.Hints[0])
			}
		})
	}
}
