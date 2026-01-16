package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestTagPropsErrorPositions tests that errors in tag prop interpolations report correct line numbers
// Note: Column accuracy depends on the complexity of props parsing; we verify line numbers are correct
func TestTagPropsErrorPositions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLine int
	}{
		{
			// <div class={missing}/>
			name:     "undefined var in singleton tag prop expression",
			input:    `<div class={missing}/>`,
			wantLine: 1,
		},
		{
			// Error on line 2
			// let x = 1
			// <div id={missing}/>
			name:     "undefined var on line 2 in tag prop",
			input:    "let x = 1\n<div id={missing}/>",
			wantLine: 2,
		},
		{
			// Paired tag with error
			// <div class={missing}>content</div>
			name:     "undefined var in paired tag prop expression",
			input:    `<div class={missing}>content</div>`,
			wantLine: 1,
		},
		{
			// Error on line 3 in paired tag
			name:     "undefined var on line 3 in paired tag",
			input:    "let x = 1\nlet y = 2\n<div id={missing}>hi</div>",
			wantLine: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			errObj, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected error, got %T: %v", result, result)
			}

			if !strings.Contains(strings.ToLower(errObj.Message), "identifier not found") {
				t.Fatalf("expected 'identifier not found' error, got: %s", errObj.Message)
			}

			if errObj.Line != tt.wantLine {
				t.Errorf("wrong line: got %d, want %d", errObj.Line, tt.wantLine)
			}

			// Column should be at least 1 (not 0)
			if errObj.Column < 1 {
				t.Errorf("column too low: got %d, want at least 1", errObj.Column)
			}
		})
	}
}
