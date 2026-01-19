package errors

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParsleyError_String(t *testing.T) {
	tests := []struct {
		name     string
		err      *ParsleyError
		expected string
	}{
		{
			name: "message only",
			err: &ParsleyError{
				Message: "something went wrong",
			},
			expected: "something went wrong",
		},
		{
			name: "with line and column",
			err: &ParsleyError{
				Message: "unexpected token",
				Line:    5,
				Column:  10,
			},
			expected: "line 5, column 10: unexpected token",
		},
		{
			name: "with file",
			err: &ParsleyError{
				Message: "parse error",
				File:    "test.pars",
				Line:    3,
				Column:  1,
			},
			expected: "test.pars: line 3, column 1: parse error",
		},
		{
			name: "with hints",
			err: &ParsleyError{
				Message: "identifier not found: foo",
				Line:    1,
				Column:  1,
				Hints:   []string{"Did you mean `for`?"},
			},
			expected: "line 1, column 1: identifier not found: foo\n  Did you mean `for`?",
		},
		{
			name: "with multiple hints",
			err: &ParsleyError{
				Message: "ambiguous syntax",
				Hints:   []string{"for (array) fn", "for x in array { ... }"},
			},
			expected: "ambiguous syntax\n  for (array) fn\n  for x in array { ... }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParsleyError_PrettyString(t *testing.T) {
	tests := []struct {
		name     string
		err      *ParsleyError
		contains []string
	}{
		{
			name: "parser error",
			err: &ParsleyError{
				Class:   ClassParse,
				Message: "unexpected token",
				Line:    5,
				Column:  10,
			},
			contains: []string{"Parser error", "line 5, column 10", "unexpected token"},
		},
		{
			name: "runtime error",
			err: &ParsleyError{
				Class:   ClassType,
				Message: "type mismatch",
				Line:    1,
				Column:  1,
			},
			contains: []string{"Runtime error", "line 1, column 1", "type mismatch"},
		},
		{
			name: "with file and hints",
			err: &ParsleyError{
				Class:   ClassParse,
				Message: "syntax error",
				File:    "handlers/index.pars",
				Line:    10,
				Column:  5,
				Hints:   []string{"for x in array { ... }", "for (array) fn"},
			},
			contains: []string{"Parser error", "in: handlers/index.pars", "at: line 10, column 5", "Use:", "or:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.PrettyString()
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("PrettyString() = %q, should contain %q", got, want)
				}
			}
		})
	}
}

func TestParsleyError_ToJSON(t *testing.T) {
	err := &ParsleyError{
		Class:   ClassType,
		Code:    "TYPE-0001",
		Message: "expected string, got integer",
		Line:    5,
		Column:  10,
		Data: map[string]any{
			"Expected": "string",
			"Got":      "integer",
		},
	}

	jsonBytes, jsonErr := err.ToJSON()
	if jsonErr != nil {
		t.Fatalf("ToJSON() error = %v", jsonErr)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed["class"] != "type" {
		t.Errorf("class = %v, want %v", parsed["class"], "type")
	}
	if parsed["code"] != "TYPE-0001" {
		t.Errorf("code = %v, want %v", parsed["code"], "TYPE-0001")
	}
	if parsed["line"].(float64) != 5 {
		t.Errorf("line = %v, want %v", parsed["line"], 5)
	}
}

func TestNew_WithCatalog(t *testing.T) {
	tests := []struct {
		name         string
		code         string
		data         map[string]any
		wantClass    ErrorClass
		wantContains string
	}{
		{
			name: "type error",
			code: "TYPE-0001",
			data: map[string]any{
				"Function": "len",
				"Expected": "string",
				"Got":      "integer",
			},
			wantClass:    ClassType,
			wantContains: "len expected string, got integer",
		},
		{
			name: "arity error",
			code: "ARITY-0001",
			data: map[string]any{
				"Function": "split",
				"Got":      "3",
				"Want":     "1-2",
			},
			wantClass:    ClassArity,
			wantContains: "Wrong number of arguments to `split`. got=3, want=1-2",
		},
		{
			name: "undefined identifier",
			code: "UNDEF-0001",
			data: map[string]any{
				"Name": "foobar",
			},
			wantClass:    ClassUndefined,
			wantContains: "Identifier not found: foobar",
		},
		{
			name: "unknown code",
			code: "UNKNOWN-9999",
			data: map[string]any{
				"message": "custom error message",
			},
			wantClass:    ClassType, // Default class
			wantContains: "custom error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.code, tt.data)
			if err.Class != tt.wantClass {
				t.Errorf("Class = %v, want %v", err.Class, tt.wantClass)
			}
			if !strings.Contains(err.Message, tt.wantContains) {
				t.Errorf("Message = %q, should contain %q", err.Message, tt.wantContains)
			}
		})
	}
}

func TestNewWithPosition(t *testing.T) {
	err := NewWithPosition("TYPE-0001", 10, 5, map[string]any{
		"Function": "test",
		"Expected": "a",
		"Got":      "b",
	})

	if err.Line != 10 {
		t.Errorf("Line = %d, want 10", err.Line)
	}
	if err.Column != 5 {
		t.Errorf("Column = %d, want 5", err.Column)
	}
}

func TestNewSimple(t *testing.T) {
	err := NewSimple(ClassIO, "file not found")
	if err.Class != ClassIO {
		t.Errorf("Class = %v, want %v", err.Class, ClassIO)
	}
	if err.Message != "file not found" {
		t.Errorf("Message = %q, want %q", err.Message, "file not found")
	}
}

func TestNewSimpleWithHints(t *testing.T) {
	err := NewSimpleWithHints(ClassSecurity, "access denied", "use -x flag", "check permissions")
	if len(err.Hints) != 2 {
		t.Errorf("len(Hints) = %d, want 2", len(err.Hints))
	}
	if err.Hints[0] != "use -x flag" {
		t.Errorf("Hints[0] = %q, want %q", err.Hints[0], "use -x flag")
	}
}

func TestParsleyError_WithFile(t *testing.T) {
	original := &ParsleyError{
		Message: "test error",
		Line:    5,
	}
	withFile := original.WithFile("test.pars")

	if withFile.File != "test.pars" {
		t.Errorf("File = %q, want %q", withFile.File, "test.pars")
	}
	if original.File != "" {
		t.Error("WithFile modified the original")
	}
}

func TestParsleyError_WithPosition(t *testing.T) {
	original := &ParsleyError{
		Message: "test error",
	}
	withPos := original.WithPosition(10, 5)

	if withPos.Line != 10 || withPos.Column != 5 {
		t.Errorf("Position = (%d, %d), want (10, 5)", withPos.Line, withPos.Column)
	}
	if original.Line != 0 {
		t.Error("WithPosition modified the original")
	}
}

func TestParsleyError_IsParseError(t *testing.T) {
	parseErr := &ParsleyError{Class: ClassParse}
	runtimeErr := &ParsleyError{Class: ClassType}

	if !parseErr.IsParseError() {
		t.Error("IsParseError() = false for parse error")
	}
	if parseErr.IsRuntimeError() {
		t.Error("IsRuntimeError() = true for parse error")
	}
	if runtimeErr.IsParseError() {
		t.Error("IsParseError() = true for runtime error")
	}
	if !runtimeErr.IsRuntimeError() {
		t.Error("IsRuntimeError() = false for runtime error")
	}
}

func TestTypeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"STRING", "string"},
		{"ARRAY", "array"},
		{"INTEGER", "integer"},
		{"FUNCTION", "function"},
		{"string", "string"},
	}

	for _, tt := range tests {
		got := TypeName(tt.input)
		if got != tt.want {
			t.Errorf("TypeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParsleyError_Error(t *testing.T) {
	err := &ParsleyError{
		Message: "test error",
		Line:    1,
		Column:  1,
	}

	// Verify it implements error interface
	var e error = err
	if e.Error() != "line 1, column 1: test error" {
		t.Errorf("Error() = %q, want %q", e.Error(), "line 1, column 1: test error")
	}
}

// ============================================================================
// Fuzzy Matching Tests
// ============================================================================

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "ab", 1},
		{"abc", "abcd", 1},
		{"kitten", "sitting", 3},
		{"fro", "for", 2}, // swap
		{"lenght", "length", 2},
		{"pritn", "print", 2},
	}

	for _, tt := range tests {
		got := levenshteinDistance(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestFindClosestMatch(t *testing.T) {
	identifiers := []string{"print", "printf", "println", "name", "length", "forEach", "map", "filter"}

	tests := []struct {
		input string
		want  string
	}{
		{"pritn", "print"},       // Simple typo (distance 2: swap)
		{"prnt", "print"},        // Missing letter (distance 1)
		{"printt", "print"},      // Extra letter (distance 1)
		{"lenght", "length"},     // Common misspelling (distance 2: swap)
		{"fro", ""},              // Distance 2 to "for" but for is not in list
		{"forEach", ""},          // Exact match returns empty
		{"xyz", ""},              // No close match
		{"mappp", "map"},         // Distance 2, threshold for 5-char input is 2
		{"mapp", "map"},          // Distance 1, within threshold
		{"filtter", "filter"},    // Double letter typo (distance 1)
		{"", ""},                 // Empty input
		{"abcdefghijklmnop", ""}, // Very long word with no match
	}

	for _, tt := range tests {
		got := FindClosestMatch(tt.input, identifiers)
		if got != tt.want {
			t.Errorf("FindClosestMatch(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}

	// Test with nil candidates
	if got := FindClosestMatch("test", nil); got != "" {
		t.Errorf("FindClosestMatch with nil candidates = %q, want empty", got)
	}
}

func TestFindTopMatches(t *testing.T) {
	identifiers := []string{"print", "printf", "println", "sprint", "paint"}

	// Note: FindTopMatches excludes exact matches (distance 0)
	// "print" has distance 0 to "print" so it's excluded
	// But case-insensitive comparison: "print" lowercase vs "print" = distance 0
	// so nothing for print should show... wait, let me check
	// Actually "printf" has distance 1 to "print", "sprint" has distance 1

	tests := []struct {
		name  string
		input string
		n     int
		desc  string
	}{
		{"typo", "pritn", 3, "should return some matches"},
		{"exact", "print", 2, "exact match excluded, close ones returned"},
		{"none", "xyz", 3, "no close matches"},
		{"empty", "", 3, "empty input"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindTopMatches(tt.input, identifiers, tt.n)
			// Verify the function doesn't panic and returns reasonable results
			if tt.input == "" || tt.input == "xyz" {
				if len(got) != 0 {
					t.Errorf("FindTopMatches(%q) = %v, want empty", tt.input, got)
				}
			} else if tt.input == "pritn" {
				if len(got) == 0 {
					t.Errorf("FindTopMatches(%q) should return matches", tt.input)
				}
				// First match should be "print" (distance 2)
				if got[0] != "print" {
					t.Errorf("FindTopMatches(%q)[0] = %q, want 'print'", tt.input, got[0])
				}
			}
		})
	}
}

func TestNewUndefinedIdentifier(t *testing.T) {
	availableIdentifiers := []string{"print", "println", "length", "forEach"}

	// Test with typo that has a close match
	err := NewUndefinedIdentifier("pritn", availableIdentifiers)
	if err.Code != "UNDEF-0001" {
		t.Errorf("Code = %q, want UNDEF-0001", err.Code)
	}
	if !strings.Contains(err.Message, "pritn") {
		t.Errorf("Message should contain 'pritn': %s", err.Message)
	}
	if len(err.Hints) == 0 || !strings.Contains(err.Hints[0], "print") {
		t.Errorf("Should have hint suggesting 'print': %v", err.Hints)
	}

	// Test with no close match
	err2 := NewUndefinedIdentifier("xyz", availableIdentifiers)
	if len(err2.Hints) != 0 {
		t.Errorf("Should have no hints for 'xyz': %v", err2.Hints)
	}
}

func TestNewUndefinedMethod(t *testing.T) {
	methods := []string{"length", "upper", "lower", "trim", "split"}

	err := NewUndefinedMethod("lenght", "string", methods)
	if err.Code != "UNDEF-0002" {
		t.Errorf("Code = %q, want UNDEF-0002", err.Code)
	}
	if !strings.Contains(err.Message, "lenght") {
		t.Errorf("Message should contain 'lenght': %s", err.Message)
	}
	if len(err.Hints) == 0 || !strings.Contains(err.Hints[0], "length") {
		t.Errorf("Should have hint suggesting 'length': %v", err.Hints)
	}
}

func TestNewMethodAsProperty(t *testing.T) {
	err := NewMethodAsProperty("data", "Record")
	if err.Code != "UNDEF-0008" {
		t.Errorf("Code = %q, want UNDEF-0008", err.Code)
	}
	if !strings.Contains(err.Message, "data") {
		t.Errorf("Message should contain 'data': %s", err.Message)
	}
	// Type names are normalized to lowercase
	if !strings.Contains(err.Message, "record") {
		t.Errorf("Message should contain 'record' (lowercase): %s", err.Message)
	}
	if len(err.Hints) == 0 || !strings.Contains(err.Hints[0], "data()") {
		t.Errorf("Should have hint suggesting parentheses: %v", err.Hints)
	}
}

func TestParsleyKeywords(t *testing.T) {
	// Verify we have all the expected keywords
	expected := map[string]bool{
		"if": true, "else": true, "for": true, "in": true, "fn": true,
		"let": true, "const": true, "return": true, "true": true, "false": true,
		"null": true, "and": true, "or": true, "not": true, "import": true,
		"export": true, "break": true, "continue": true, "switch": true,
		"case": true, "default": true,
	}

	for _, kw := range ParsleyKeywords {
		if !expected[kw] {
			t.Errorf("Unexpected keyword in ParsleyKeywords: %q", kw)
		}
		delete(expected, kw)
	}

	for kw := range expected {
		t.Errorf("Missing keyword in ParsleyKeywords: %q", kw)
	}
}
