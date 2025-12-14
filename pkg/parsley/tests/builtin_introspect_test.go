package tests

import (
	"strings"
	"testing"
)

func TestBuiltinIntrospection(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		contains []string
	}{
		{
			name: "builtins() returns all categories",
			code: `builtins()`,
			contains: []string{
				"file",
				"time",
				"conversion",
				"introspection",
				"output",
			},
		},
		{
			name: "builtins(category) filters by category",
			code: `let timeBuiltins = builtins("time"); timeBuiltins["time"].length() > 0`,
			contains: []string{
				"true",
			},
		},
		{
			name: "builtin metadata includes required fields",
			code: `
				let allBuiltins = builtins()
				let fileBuiltins = allBuiltins["file"]
				let firstBuiltin = fileBuiltins[0]
				firstBuiltin["name"].length() > 0 and 
				firstBuiltin["description"].length() > 0 and 
				firstBuiltin["arity"].length() > 0
			`,
			contains: []string{
				"true",
			},
		},
		{
			name: "builtin params field exists",
			code: `
				let allBuiltins = builtins()
				let fileBuiltins = allBuiltins["file"]
				let jsonBuiltin = null
				for builtin in fileBuiltins {
					if builtin["name"] == "JSON" {
						jsonBuiltin = builtin
					}
				}
				jsonBuiltin != null and jsonBuiltin["params"].length() > 0
			`,
			contains: []string{
				"true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalHelper(tt.code)

			resultStr := result.Inspect()
			for _, want := range tt.contains {
				if !strings.Contains(resultStr, want) {
					t.Errorf("Expected output to contain %q, got:\n%s", want, resultStr)
				}
			}
		})
	}
}

func TestBuiltinsReturnType(t *testing.T) {
	code := `builtins()`
	result := testEvalHelper(code)

	// Should return a dictionary
	if !strings.Contains(result.Inspect(), "{") {
		t.Errorf("Expected builtins() to return a dictionary, got: %s", result.Inspect())
	}
}

func TestBuiltinsContainsExpectedCategories(t *testing.T) {
	code := `
		let allBuiltins = builtins()
		let hasFile = "file" in allBuiltins
		let hasTime = "time" in allBuiltins
		let hasConversion = "conversion" in allBuiltins
		hasFile and hasTime and hasConversion
	`
	
	result := testEvalHelper(code)

	if result.Inspect() != "true" {
		t.Errorf("Expected builtins() to contain file, time, and conversion categories")
	}
}

