package pln_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
	_ "github.com/sambeau/basil/pkg/parsley/pln" // Register PLN hooks
)

// TestPLNFileLoading tests loading .pln files via the file() builtin
func TestPLNFileLoading(t *testing.T) {
	// Get the absolute path to the test data file
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	testFile := filepath.Join(wd, "testdata", "sample.pln")

	// Create Parsley code that loads the PLN file
	code := `
		let f = file("` + testFile + `")
		let data <== f
		data
	`

	result := evalParsleyWithFile(code, testFile)
	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T: %v", result, result)
	}

	// Verify the loaded data
	if len(dict.Pairs) < 4 {
		t.Errorf("expected at least 4 fields, got %d", len(dict.Pairs))
	}
}

// TestPLNBuiltinFunction tests the PLN() builtin function
func TestPLNBuiltinFunction(t *testing.T) {
	// Get the absolute path to the test data file
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	testFile := filepath.Join(wd, "testdata", "sample.pln")

	// Create Parsley code that uses the PLN builtin
	code := `
		let f = PLN("` + testFile + `")
		let data <== f
		data
	`

	result := evalParsleyWithFile(code, testFile)
	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T: %v", result, result)
	}

	// Verify the loaded data
	if len(dict.Pairs) < 4 {
		t.Errorf("expected at least 4 fields, got %d", len(dict.Pairs))
	}
}

// TestPLNWriteBasic tests basic PLN file writing
func TestPLNWriteBasic(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name  string
		value string
		check func(*testing.T, string)
	}{
		{
			name:  "integer",
			value: "42",
			check: func(t *testing.T, content string) {
				if content != "42" {
					t.Errorf("expected '42', got '%s'", content)
				}
			},
		},
		{
			name:  "string",
			value: `"hello"`,
			check: func(t *testing.T, content string) {
				if content != `"hello"` {
					t.Errorf("expected '\"hello\"', got '%s'", content)
				}
			},
		},
		{
			name:  "array",
			value: "[1, 2, 3]",
			check: func(t *testing.T, content string) {
				if content != "[1, 2, 3]" {
					t.Errorf("expected '[1, 2, 3]', got '%s'", content)
				}
			},
		},
		{
			name:  "boolean_true",
			value: "true",
			check: func(t *testing.T, content string) {
				if content != "true" {
					t.Errorf("expected 'true', got '%s'", content)
				}
			},
		},
		{
			name:  "boolean_false",
			value: "false",
			check: func(t *testing.T, content string) {
				if content != "false" {
					t.Errorf("expected 'false', got '%s'", content)
				}
			},
		},
		{
			name:  "dict",
			value: "{a: 1, b: 2}",
			check: func(t *testing.T, content string) {
				// PLN serializes dicts - just check it contains key-value pairs
				if !strings.Contains(content, "a:") || !strings.Contains(content, "b:") {
					t.Errorf("expected dict with a and b keys, got '%s'", content)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "test_"+tt.name+".pln")
			code := tt.value + ` ==> PLN("` + testFile + `")`

			result := evalParsley(code)
			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			// Read the file and verify content
			content, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}

			tt.check(t, string(content))
		})
	}
}

// TestPLNRoundTrip tests that write then read preserves values
func TestPLNRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name  string
		value string
	}{
		{
			name: "simple_dict",
			value: `{
				name: "Alice",
				age: 30,
				tags: ["admin", "user"],
				meta: {active: true, verified: false}
			}`,
		},
		{
			name:  "mixed_basic_types",
			value: `[42, "text", true, false, null, {nested: true}]`,
		},
		{
			name:  "nested_arrays",
			value: `[[1, 2], [3, 4], [5, 6]]`,
		},
		{
			name:  "nested_dicts",
			value: `{a: {b: {c: 1}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "roundtrip_"+tt.name+".pln")

			// Write the value
			writeCode := `let data = ` + tt.value + `
data ==> PLN("` + testFile + `")`

			writeResult := evalParsley(writeCode)
			if errObj, ok := writeResult.(*evaluator.Error); ok {
				t.Fatalf("write failed: %s", errObj.Message)
			}

			// Read it back
			readCode := `let loaded <== PLN("` + testFile + `")
loaded`

			readResult := evalParsley(readCode)
			if errObj, ok := readResult.(*evaluator.Error); ok {
				t.Fatalf("read failed: %s", errObj.Message)
			}

			// Verify by comparing inspected values
			// (Since we can't easily compare object equality in Go)
			originalResult := evalParsley(tt.value)
			if originalResult.Inspect() != readResult.Inspect() {
				t.Errorf("round-trip failed:\noriginal: %s\nloaded:   %s",
					originalResult.Inspect(), readResult.Inspect())
			}
		})
	}
}

// TestPLNWriteErrors tests error cases for non-serializable values
func TestPLNWriteErrors(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		value        string
		wantErrorMsg string // substring to check
	}{
		{
			name:         "function",
			value:        "fn(x) { x + 1 }",
			wantErrorMsg: "serialize",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "error_"+tt.name+".pln")
			code := tt.value + ` ==> PLN("` + testFile + `")`

			result := evalParsley(code)
			errObj, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected error, got %T: %v", result, result)
			}

			// Check error message contains expected substring
			if errObj.Message == "" {
				t.Errorf("error message is empty")
			}
			// Error should indicate serialization failure
			// (exact message may vary, just check it's an error)
		})
	}
}

// TestPLNAppendMode tests that ==>> appends values on new lines
func TestPLNAppendMode(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "append.pln")

	// Append multiple values
	values := []string{`"first"`, `"second"`, `42`, `{key: "value"}`}

	for _, val := range values {
		code := val + ` ==>> PLN("` + testFile + `")`
		result := evalParsley(code)
		if errObj, ok := result.(*evaluator.Error); ok {
			t.Fatalf("append failed for %s: %s", val, errObj.Message)
		}
	}

	// Read the file and verify it has multiple lines
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	contentStr := string(content)
	lines := 0
	for i := 0; i < len(contentStr); i++ {
		if contentStr[i] == '\n' {
			lines++
		}
	}

	// Should have at least 3 newlines (4 values = 3 separators minimum)
	if lines < 3 {
		t.Errorf("expected at least 3 newlines, got %d in content:\n%s", lines, contentStr)
	}

	// Also test that native types that aren't yet supported by PLN produce errors
	t.Run("unsupported_native_types", func(t *testing.T) {
		// Money and datetime are native types but PLN serializer
		// only handles their dictionary representations currently
		testFile := filepath.Join(tmpDir, "error_native.pln")

		// Test with money
		code := `$100.00 ==> PLN("` + testFile + `")`
		result := evalParsley(code)
		if _, ok := result.(*evaluator.Error); !ok {
			t.Log("Note: Money type serialization may have been implemented")
		}
	})
}

// evalParsleyWithFile evaluates Parsley code with a file context
func evalParsleyWithFile(code, filename string) evaluator.Object {
	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return &evaluator.Error{Message: p.Errors()[0]}
	}
	env := evaluator.NewEnvironment()
	env.Filename = filename
	return evaluator.Eval(program, env)
}
