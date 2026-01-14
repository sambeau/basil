package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// testEvalProcess is a local helper that enables execute-all for process tests
func testEvalProcess(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	// Enable execute-all for tests
	env.Security = &evaluator.SecurityPolicy{
		AllowExecuteAll: true,
	}
	return evaluator.Eval(program, env)
}

// TestProcessExecutionToken tests the <=#=> token
func TestProcessExecutionToken(t *testing.T) {
	input := `let result = @shell("echo") <=#=> "hello"`

	l := lexer.New(input)
	foundExecuteWith := false
	for tok := l.NextToken(); tok.Type != lexer.EOF; tok = l.NextToken() {
		if tok.Type == lexer.EXECUTE_WITH {
			foundExecuteWith = true
			break
		}
	}

	if !foundExecuteWith {
		t.Errorf("Expected to find EXECUTE_WITH token in input")
	}
}

// TestShellLiteral creates command handles via @shell
func TestShellLiteral(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "@shell with binary only",
			input:   `@shell("echo")`,
			wantErr: false,
		},
		{
			name:    "@shell with binary and args",
			input:   `@shell("echo", ["hello", "world"])`,
			wantErr: false,
		},
		{
			name:    "@shell with all options",
			input:   `@shell("ls", ["-la"], {env: {PATH: "/usr/bin"}, dir: "/tmp"})`,
			wantErr: false,
		},
		{
			name:    "@shell with no arguments",
			input:   `@shell()`,
			wantErr: true,
		},
		{
			name:    "@shell with non-string binary",
			input:   `@shell(123)`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalProcess(tt.input)
			if tt.wantErr {
				if _, ok := result.(*evaluator.Error); !ok {
					t.Errorf("Expected error, got %T: %v", result, result)
				}
			} else {
				if _, ok := result.(*evaluator.Error); ok {
					t.Errorf("Expected success, got error: %v", result)
				}
				// Check it's a dictionary with __type: "command"
				if dict, ok := result.(*evaluator.Dictionary); ok {
					if typeExpr, exists := dict.Pairs["__type"]; exists {
						evaluated := evaluator.Eval(typeExpr, evaluator.NewEnvironment())
						if str, ok := evaluated.(*evaluator.String); ok {
							if str.Value != "command" {
								t.Errorf("Expected __type='command', got '%s'", str.Value)
							}
						} else {
							t.Errorf("Expected __type to be string, got %T", evaluated)
						}
					} else {
						t.Errorf("Expected __type field in command handle")
					}
				} else {
					t.Errorf("Expected Dictionary, got %T", result)
				}
			}
		})
	}
}

// TestProcessExecution tests basic process execution
func TestProcessExecution(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		checkStdout func(string) bool
	}{
		{
			name:    "echo command",
			input:   `let result = @shell("echo", ["hello"]) <=#=> null; result.stdout`,
			wantErr: false,
			checkStdout: func(s string) bool {
				return strings.TrimSpace(s) == "hello"
			},
		},
		{
			name:    "echo with input (ignored)",
			input:   `let result = @shell("echo", ["test"]) <=#=> "input data"; result.stdout`,
			wantErr: false,
			checkStdout: func(s string) bool {
				return strings.Contains(s, "test")
			},
		},
		{
			name:    "command with exit code",
			input:   `let result = @shell("echo", ["ok"]) <=#=> null; result.exitCode`,
			wantErr: false,
			checkStdout: func(s string) bool {
				return s == "0"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalProcess(tt.input)
			if tt.wantErr {
				if _, ok := result.(*evaluator.Error); !ok {
					t.Errorf("Expected error, got %T: %v", result, result)
				}
			} else {
				if err, ok := result.(*evaluator.Error); ok {
					t.Errorf("Expected success, got error: %v", err.Message)
					return
				}
				if str, ok := result.(*evaluator.String); ok {
					if !tt.checkStdout(str.Value) {
						t.Errorf("Output check failed for: %s", str.Value)
					}
				} else if integer, ok := result.(*evaluator.Integer); ok {
					if !tt.checkStdout(integer.Inspect()) {
						t.Errorf("Output check failed for: %d", integer.Value)
					}
				} else {
					t.Errorf("Expected String or Integer, got %T", result)
				}
			}
		})
	}
}

// TestProcessExecutionResult tests result structure
func TestProcessExecutionResult(t *testing.T) {
	input := `let result = @shell("echo", ["test output"]) <=#=> null; result.exitCode == 0`

	result := testEvalProcess(input)
	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("Expected success, got error: %v", err.Message)
	}

	boolean, ok := result.(*evaluator.Boolean)
	if !ok {
		t.Fatalf("Expected Boolean, got %T", result)
	}

	if !boolean.Value {
		t.Errorf("Expected exitCode == 0 to be true")
	}

	// Test all fields exist
	input2 := `let result = @shell("echo", ["test"]) <=#=> null; [result.stdout, result.stderr, result.exitCode, result.error]`

	result2 := testEvalProcess(input2)
	if err, ok := result2.(*evaluator.Error); ok {
		t.Fatalf("Expected success, got error: %v", err.Message)
	}

	arr, ok := result2.(*evaluator.Array)
	if !ok {
		t.Fatalf("Expected Array, got %T", result2)
	}

	if len(arr.Elements) != 4 {
		t.Errorf("Expected 4 elements in result, got %d", len(arr.Elements))
	}
}

// TestJSONSerializationMethods tests parseJSON and toJSON methods
func TestJSONSerializationMethods(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(evaluator.Object) bool
	}{
		{
			name:    "parseJSON simple object",
			input:   `"{\"name\":\"Alice\",\"age\":30}".parseJSON()`,
			wantErr: false,
			check: func(obj evaluator.Object) bool {
				dict, ok := obj.(*evaluator.Dictionary)
				return ok && len(dict.Pairs) == 2
			},
		},
		{
			name:    "parseJSON array",
			input:   `"[1,2,3]".parseJSON()`,
			wantErr: false,
			check: func(obj evaluator.Object) bool {
				arr, ok := obj.(*evaluator.Array)
				return ok && len(arr.Elements) == 3
			},
		},
		{
			name:    "toJSON object",
			input:   `{name: "Bob", age: 25}.toJSON()`,
			wantErr: false,
			check: func(obj evaluator.Object) bool {
				str, ok := obj.(*evaluator.String)
				return ok && strings.Contains(str.Value, "Bob") && strings.Contains(str.Value, "25")
			},
		},
		{
			name:    "toJSON array",
			input:   `[1, 2, 3].toJSON()`,
			wantErr: false,
			check: func(obj evaluator.Object) bool {
				str, ok := obj.(*evaluator.String)
				return ok && strings.Contains(str.Value, "[") && strings.Contains(str.Value, "]")
			},
		},
		{
			name:    "parseJSON invalid JSON",
			input:   `"{invalid}".parseJSON()`,
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalProcess(tt.input)
			if tt.wantErr {
				if _, ok := result.(*evaluator.Error); !ok {
					t.Errorf("Expected error, got %T: %v", result, result)
				}
			} else {
				if err, ok := result.(*evaluator.Error); ok {
					t.Errorf("Expected success, got error: %v", err.Message)
					return
				}
				if tt.check != nil && !tt.check(result) {
					t.Errorf("Check failed for result: %v", result)
				}
			}
		})
	}
}

// TestCSVFormatMethods tests parseCSV and toCSV methods
func TestCSVFormatMethods(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(evaluator.Object) bool
	}{
		{
			name:    "parseCSV without header",
			input:   `"a,b,c\n1,2,3".parseCSV(false)`,
			wantErr: false,
			check: func(obj evaluator.Object) bool {
				arr, ok := obj.(*evaluator.Array)
				return ok && len(arr.Elements) == 2
			},
		},
		{
			name:    "parseCSV with header (default)",
			input:   `"name,age\nAlice,30".parseCSV()`,
			wantErr: false,
			check: func(obj evaluator.Object) bool {
				// CSV with headers now returns Table
				tbl, ok := obj.(*evaluator.Table)
				if !ok || len(tbl.Rows) == 0 {
					return false
				}
				// First row should be a dictionary with keys from header
				dict := tbl.Rows[0]
				return len(dict.Pairs) == 2
			},
		},
		{
			name:    "parseCSV with explicit header option",
			input:   `"name,age\nAlice,30".parseCSV(true)`,
			wantErr: false,
			check: func(obj evaluator.Object) bool {
				// CSV with headers now returns Table
				tbl, ok := obj.(*evaluator.Table)
				if !ok || len(tbl.Rows) == 0 {
					return false
				}
				// First row should be a dictionary with keys from header
				dict := tbl.Rows[0]
				return len(dict.Pairs) == 2
			},
		},
		{
			name:    "toCSV array of arrays",
			input:   `[["a","b"],["1","2"]].toCSV()`,
			wantErr: false,
			check: func(obj evaluator.Object) bool {
				str, ok := obj.(*evaluator.String)
				return ok && strings.Contains(str.Value, "a,b")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalProcess(tt.input)
			if tt.wantErr {
				if _, ok := result.(*evaluator.Error); !ok {
					t.Errorf("Expected error, got %T: %v", result, result)
				}
			} else {
				if err, ok := result.(*evaluator.Error); ok {
					t.Errorf("Expected success, got error: %v", err.Message)
					return
				}
				if tt.check != nil && !tt.check(result) {
					t.Errorf("Check failed for result: %v", result)
				}
			}
		})
	}
}

// TestJSONRoundTrip tests JSON parse/stringify round-trip
func TestJSONRoundTrip(t *testing.T) {
	input := `let obj = {name: "Test", value: 42, items: [1, 2, 3]}; let json = obj.toJSON(); let parsed = json.parseJSON(); parsed.name`

	result := testEvalProcess(input)
	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("Expected success, got error: %v", err.Message)
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	if str.Value != "Test" {
		t.Errorf("Expected 'Test', got '%s'", str.Value)
	}
}
