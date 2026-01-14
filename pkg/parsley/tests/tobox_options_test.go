package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// TestToBoxStyleOption tests the style option for toBox
func TestToBoxStyleOption(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string // strings that should be present
		excludes []string // strings that should NOT be present
	}{
		{
			name:     "default style (single line)",
			input:    `[1, 2, 3].toBox()`,
			contains: []string{"┌", "┐", "└", "┘", "│", "─"},
		},
		{
			name:     "explicit single style",
			input:    `[1, 2, 3].toBox({style: "single"})`,
			contains: []string{"┌", "┐", "└", "┘", "│", "─"},
		},
		{
			name:     "double style",
			input:    `[1, 2, 3].toBox({style: "double"})`,
			contains: []string{"╔", "╗", "╚", "╝", "║", "═"},
		},
		{
			name:     "ascii style",
			input:    `[1, 2, 3].toBox({style: "ascii"})`,
			contains: []string{"+", "|", "-"},
			excludes: []string{"┌", "╔"},
		},
		{
			name:     "rounded style",
			input:    `[1, 2, 3].toBox({style: "rounded"})`,
			contains: []string{"╭", "╮", "╰", "╯", "│", "─"},
		},
		{
			name:     "dict with double style",
			input:    `{a: 1, b: 2}.toBox({style: "double"})`,
			contains: []string{"╔", "╗", "╚", "╝", "║", "═", "a", "b"},
		},
		{
			name:     "table with ascii style",
			input:    `table([{name: "Alice"}]).toBox({style: "ascii"})`,
			contains: []string{"+", "|", "-", "Alice", "name"},
			excludes: []string{"┌", "╔"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTest(t, tt.input)
			str, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String, got %s: %s", result.Type(), result.Inspect())
			}

			for _, c := range tt.contains {
				if !strings.Contains(str.Value, c) {
					t.Errorf("expected to contain %q, got:\n%s", c, str.Value)
				}
			}

			for _, e := range tt.excludes {
				if strings.Contains(str.Value, e) {
					t.Errorf("expected to NOT contain %q, got:\n%s", e, str.Value)
				}
			}
		})
	}
}

// TestToBoxTitleOption tests the title option for toBox
func TestToBoxTitleOption(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "array with title",
			input:    `[1, 2, 3].toBox({title: "Numbers"})`,
			contains: []string{"Numbers", "1", "2", "3"},
		},
		{
			name:     "dict with title",
			input:    `{name: "Alice", age: 30}.toBox({title: "Person"})`,
			contains: []string{"Person", "name", "Alice", "age", "30"},
		},
		{
			name:     "table with title",
			input:    `table([{x: 1}, {x: 2}]).toBox({title: "Data"})`,
			contains: []string{"Data", "x", "1", "2"},
		},
		{
			name:     "empty title (no title row)",
			input:    `[1, 2].toBox({title: ""})`,
			contains: []string{"1", "2"},
		},
		{
			name:     "long title expands box",
			input:    `[1].toBox({title: "This is a very long title"})`,
			contains: []string{"This is a very long title", "1"},
		},
		{
			name:     "title with style",
			input:    `[1, 2].toBox({title: "Test", style: "double"})`,
			contains: []string{"Test", "1", "2", "╔", "║"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTest(t, tt.input)
			str, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String, got %s: %s", result.Type(), result.Inspect())
			}

			for _, c := range tt.contains {
				if !strings.Contains(str.Value, c) {
					t.Errorf("expected to contain %q, got:\n%s", c, str.Value)
				}
			}
		})
	}
}

// TestToBoxMaxWidthOption tests the maxWidth option for toBox
func TestToBoxMaxWidthOption(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name:     "truncate long string",
			input:    `["This is a very long string that should be truncated"].toBox({maxWidth: 10})`,
			contains: []string{"..."},
			excludes: []string{"truncated"},
		},
		{
			name:     "no truncation needed",
			input:    `["short"].toBox({maxWidth: 20})`,
			contains: []string{"short"},
			excludes: []string{"..."},
		},
		{
			name:     "dict value truncation",
			input:    `{key: "This is a very long value"}.toBox({maxWidth: 10})`,
			contains: []string{"key", "..."},
			excludes: []string{"very long value"},
		},
		{
			name:     "table cell truncation",
			input:    `table([{name: "This is a very long name"}]).toBox({maxWidth: 10})`,
			contains: []string{"name", "..."},
			excludes: []string{"very long"},
		},
		{
			name:     "maxWidth with title",
			input:    `["Very long content here"].toBox({maxWidth: 10, title: "Data"})`,
			contains: []string{"Data", "..."},
		},
		{
			name:     "maxWidth of 0 means no limit",
			input:    `["This is long"].toBox({maxWidth: 0})`,
			contains: []string{"This is long"},
			excludes: []string{"..."},
		},
		{
			name:     "very small maxWidth (<=3) ignored",
			input:    `["Hello"].toBox({maxWidth: 3})`,
			contains: []string{"Hello"},
			excludes: []string{"..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTest(t, tt.input)
			str, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String, got %s: %s", result.Type(), result.Inspect())
			}

			for _, c := range tt.contains {
				if !strings.Contains(str.Value, c) {
					t.Errorf("expected to contain %q, got:\n%s", c, str.Value)
				}
			}

			for _, e := range tt.excludes {
				if strings.Contains(str.Value, e) {
					t.Errorf("expected to NOT contain %q, got:\n%s", e, str.Value)
				}
			}
		})
	}
}

// TestToBoxCombinedOptions tests combining multiple options
func TestToBoxCombinedOptions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name:     "style + title",
			input:    `[1, 2, 3].toBox({style: "double", title: "Numbers"})`,
			contains: []string{"Numbers", "1", "2", "3", "╔", "║"},
		},
		{
			name:     "style + maxWidth",
			input:    `["This is a long string"].toBox({style: "ascii", maxWidth: 10})`,
			contains: []string{"+", "|", "..."},
			excludes: []string{"long string"},
		},
		{
			name:     "title + maxWidth",
			input:    `["Very long content"].toBox({title: "Data", maxWidth: 8})`,
			contains: []string{"Data", "..."},
		},
		{
			name:     "all three options",
			input:    `["Some long text here"].toBox({style: "rounded", title: "Info", maxWidth: 10})`,
			contains: []string{"╭", "Info", "..."},
			excludes: []string{"text here"},
		},
		{
			name:     "with direction option",
			input:    `[1, 2, 3].toBox({direction: "horizontal", style: "double", title: "Row"})`,
			contains: []string{"Row", "1", "2", "3", "╔"},
		},
		{
			name:     "with align option",
			input:    `[1, 2, 3].toBox({align: "center", title: "Centered"})`,
			contains: []string{"Centered", "1", "2", "3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTest(t, tt.input)
			str, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String, got %s: %s", result.Type(), result.Inspect())
			}

			for _, c := range tt.contains {
				if !strings.Contains(str.Value, c) {
					t.Errorf("expected to contain %q, got:\n%s", c, str.Value)
				}
			}

			for _, e := range tt.excludes {
				if strings.Contains(str.Value, e) {
					t.Errorf("expected to NOT contain %q, got:\n%s", e, str.Value)
				}
			}
		})
	}
}

// TestToBoxStyleErrors tests error handling for invalid style values
func TestToBoxStyleErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		errorContains string
	}{
		{
			name:          "invalid style name",
			input:         `[1, 2].toBox({style: "fancy"})`,
			errorContains: "invalid style",
		},
		{
			name:          "style wrong type",
			input:         `[1, 2].toBox({style: 123})`,
			errorContains: "style",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalHelper(tt.input)
			if result.Type() != evaluator.ERROR_OBJ {
				t.Errorf("expected error, got %s: %s", result.Type(), result.Inspect())
				return
			}

			errStr := result.Inspect()
			if !strings.Contains(strings.ToLower(errStr), strings.ToLower(tt.errorContains)) {
				t.Errorf("expected error containing %q, got: %s", tt.errorContains, errStr)
			}
		})
	}
}

// TestToBoxMaxWidthErrors tests error handling for invalid maxWidth values
func TestToBoxMaxWidthErrors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		errorContains string
	}{
		{
			name:          "negative maxWidth",
			input:         `[1, 2].toBox({maxWidth: -5})`,
			errorContains: "maxWidth",
		},
		{
			name:          "maxWidth wrong type",
			input:         `[1, 2].toBox({maxWidth: "ten"})`,
			errorContains: "maxWidth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalHelper(tt.input)
			if result.Type() != evaluator.ERROR_OBJ {
				t.Errorf("expected error, got %s: %s", result.Type(), result.Inspect())
				return
			}

			errStr := result.Inspect()
			if !strings.Contains(strings.ToLower(errStr), strings.ToLower(tt.errorContains)) {
				t.Errorf("expected error containing %q, got: %s", tt.errorContains, errStr)
			}
		})
	}
}

// TestToBoxTitleTypeError tests that title must be a string
func TestToBoxTitleTypeError(t *testing.T) {
	result := testEvalHelper(`[1, 2].toBox({title: 123})`)
	if result.Type() != evaluator.ERROR_OBJ {
		t.Errorf("expected error for non-string title, got %s: %s", result.Type(), result.Inspect())
		return
	}

	errStr := result.Inspect()
	if !strings.Contains(strings.ToLower(errStr), "title") {
		t.Errorf("expected error about title, got: %s", errStr)
	}
}

// TestToBoxHorizontalWithOptions tests horizontal direction with options
func TestToBoxHorizontalWithOptions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "horizontal with title",
			input:    `[1, 2, 3].toBox({direction: "horizontal", title: "Items"})`,
			contains: []string{"Items", "1", "2", "3"},
		},
		{
			name:     "horizontal with style",
			input:    `[1, 2, 3].toBox({direction: "horizontal", style: "ascii"})`,
			contains: []string{"+", "|", "1", "2", "3"},
		},
		{
			name:     "horizontal with maxWidth",
			input:    `["Long", "Short"].toBox({direction: "horizontal", maxWidth: 3})`,
			contains: []string{"Long", "Short"}, // maxWidth too small, ignored
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTest(t, tt.input)
			str, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String, got %s: %s", result.Type(), result.Inspect())
			}

			for _, c := range tt.contains {
				if !strings.Contains(str.Value, c) {
					t.Errorf("expected to contain %q, got:\n%s", c, str.Value)
				}
			}
		})
	}
}

// TestToBoxGridWithOptions tests grid direction with options
func TestToBoxGridWithOptions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "grid with title",
			input:    `[[1, 2], [3, 4]].toBox({direction: "grid", title: "Matrix"})`,
			contains: []string{"Matrix", "1", "2", "3", "4"},
		},
		{
			name:     "grid with double style",
			input:    `[[1, 2], [3, 4]].toBox({direction: "grid", style: "double"})`,
			contains: []string{"╔", "║", "1", "2", "3", "4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalTest(t, tt.input)
			str, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String, got %s: %s", result.Type(), result.Inspect())
			}

			for _, c := range tt.contains {
				if !strings.Contains(str.Value, c) {
					t.Errorf("expected to contain %q, got:\n%s", c, str.Value)
				}
			}
		})
	}
}
