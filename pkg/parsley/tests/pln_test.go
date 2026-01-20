package tests

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

// testPLNCode evaluates Parsley code that uses PLN functions
func testPLNCode(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return &evaluator.Error{Message: p.Errors()[0]}
	}
	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// ---- Basic Serialize/Deserialize Tests ----

func TestPLNSerializeInteger(t *testing.T) {
	result := testPLNCode(`serialize(42)`)
	strObj, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %v", result, result)
	}
	if strObj.Value != "42" {
		t.Errorf("expected '42', got %q", strObj.Value)
	}
}

func TestPLNSerializeFloat(t *testing.T) {
	result := testPLNCode(`serialize(3.14)`)
	strObj, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %v", result, result)
	}
	if !strings.Contains(strObj.Value, "3.14") {
		t.Errorf("expected '3.14', got %q", strObj.Value)
	}
}

func TestPLNSerializeString(t *testing.T) {
	result := testPLNCode(`serialize("hello world")`)
	strObj, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %v", result, result)
	}
	if strObj.Value != `"hello world"` {
		t.Errorf("expected '\"hello world\"', got %q", strObj.Value)
	}
}

func TestPLNSerializeArray(t *testing.T) {
	result := testPLNCode(`serialize([1, 2, 3])`)
	strObj, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %v", result, result)
	}
	if strObj.Value != "[1, 2, 3]" {
		t.Errorf("expected '[1, 2, 3]', got %q", strObj.Value)
	}
}

func TestPLNSerializeDict(t *testing.T) {
	result := testPLNCode(`serialize({name: "Alice"})`)
	strObj, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %v", result, result)
	}
	if !strings.Contains(strObj.Value, "name") || !strings.Contains(strObj.Value, "Alice") {
		t.Errorf("expected dict with name, got %q", strObj.Value)
	}
}

func TestPLNSerializeFunctionFails(t *testing.T) {
	result := testPLNCode(`serialize(fn(x) { x })`)
	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error for serializing function, got %T: %v", result, result)
	}
	if !strings.Contains(errObj.Message, "function") && !strings.Contains(errObj.Message, "FUNCTION") {
		t.Errorf("expected error about function, got %q", errObj.Message)
	}
}

// ---- Deserialize Tests ----

func TestPLNDeserializeInteger(t *testing.T) {
	result := testPLNCode(`deserialize("42")`)
	intObj, ok := result.(*evaluator.Integer)
	if !ok {
		if err, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("unexpected error: %s", err.Message)
		}
		t.Fatalf("expected Integer, got %T: %v", result, result)
	}
	if intObj.Value != 42 {
		t.Errorf("expected 42, got %d", intObj.Value)
	}
}

func TestPLNDeserializeArray(t *testing.T) {
	result := testPLNCode(`deserialize("[1, 2, 3]")`)
	arrObj, ok := result.(*evaluator.Array)
	if !ok {
		if err, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("unexpected error: %s", err.Message)
		}
		t.Fatalf("expected Array, got %T: %v", result, result)
	}
	if len(arrObj.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arrObj.Elements))
	}
}

func TestPLNDeserializeDict(t *testing.T) {
	result := testPLNCode(`deserialize("{name: \"Alice\", age: 30}")`)
	dictObj, ok := result.(*evaluator.Dictionary)
	if !ok {
		if err, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("unexpected error: %s", err.Message)
		}
		t.Fatalf("expected Dictionary, got %T: %v", result, result)
	}
	if len(dictObj.Pairs) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(dictObj.Pairs))
	}
}

func TestPLNDeserializeInvalidFails(t *testing.T) {
	result := testPLNCode(`deserialize("{invalid")`)
	_, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error for invalid PLN, got %T: %v", result, result)
	}
}

// ---- Round Trip Tests ----

func TestPLNRoundTripInteger(t *testing.T) {
	result := testPLNCode(`
		let x = 42
		let pln = serialize(x)
		let y = deserialize(pln)
		y == x
	`)
	boolObj, ok := result.(*evaluator.Boolean)
	if !ok {
		if err, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("unexpected error: %s", err.Message)
		}
		t.Fatalf("expected Boolean, got %T: %v", result, result)
	}
	if !boolObj.Value {
		t.Error("round-trip integer comparison failed")
	}
}

func TestPLNRoundTripString(t *testing.T) {
	result := testPLNCode(`
		let x = "hello world"
		let pln = serialize(x)
		let y = deserialize(pln)
		y == x
	`)
	boolObj, ok := result.(*evaluator.Boolean)
	if !ok {
		if err, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("unexpected error: %s", err.Message)
		}
		t.Fatalf("expected Boolean, got %T: %v", result, result)
	}
	if !boolObj.Value {
		t.Error("round-trip string comparison failed")
	}
}

func TestPLNRoundTripArray(t *testing.T) {
	result := testPLNCode(`
		let x = [1, 2, 3]
		let pln = serialize(x)
		let y = deserialize(pln)
		y.length() == x.length()
	`)
	boolObj, ok := result.(*evaluator.Boolean)
	if !ok {
		if err, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("unexpected error: %s", err.Message)
		}
		t.Fatalf("expected Boolean, got %T: %v", result, result)
	}
	if !boolObj.Value {
		t.Error("round-trip array length comparison failed")
	}
}

func TestPLNRoundTripNestedDict(t *testing.T) {
	result := testPLNCode(`
		let x = {
			name: "Alice",
			profile: {
				email: "alice@example.com",
				active: true
			},
			tags: ["admin", "user"]
		}
		let pln = serialize(x)
		let y = deserialize(pln)
		y.name == "Alice" && y.profile.email == "alice@example.com" && y.tags.length() == 2
	`)
	boolObj, ok := result.(*evaluator.Boolean)
	if !ok {
		if err, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("unexpected error: %s", err.Message)
		}
		t.Fatalf("expected Boolean, got %T: %v", result, result)
	}
	if !boolObj.Value {
		t.Error("round-trip nested dict comparison failed")
	}
}

// ---- Complex Type Tests ----

func TestPLNSerializeNull(t *testing.T) {
	result := testPLNCode(`serialize(null)`)
	strObj, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %v", result, result)
	}
	if strObj.Value != "null" {
		t.Errorf("expected 'null', got %q", strObj.Value)
	}
}

func TestPLNRoundTripNull(t *testing.T) {
	result := testPLNCode(`
		let pln = serialize(null)
		let y = deserialize(pln)
		y == null
	`)
	boolObj, ok := result.(*evaluator.Boolean)
	if !ok {
		if err, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("unexpected error: %s", err.Message)
		}
		t.Fatalf("expected Boolean, got %T: %v", result, result)
	}
	if !boolObj.Value {
		t.Error("round-trip null comparison failed")
	}
}

func TestPLNRoundTripEmptyArray(t *testing.T) {
	result := testPLNCode(`
		let x = []
		let pln = serialize(x)
		let y = deserialize(pln)
		y.length() == 0
	`)
	boolObj, ok := result.(*evaluator.Boolean)
	if !ok {
		if err, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("unexpected error: %s", err.Message)
		}
		t.Fatalf("expected Boolean, got %T: %v", result, result)
	}
	if !boolObj.Value {
		t.Error("round-trip empty array failed")
	}
}

func TestPLNRoundTripEmptyDict(t *testing.T) {
	result := testPLNCode(`
		let x = {}
		let pln = serialize(x)
		let y = deserialize(pln)
		y.keys().length() == 0
	`)
	boolObj, ok := result.(*evaluator.Boolean)
	if !ok {
		if err, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("unexpected error: %s", err.Message)
		}
		t.Fatalf("expected Boolean, got %T: %v", result, result)
	}
	if !boolObj.Value {
		t.Error("round-trip empty dict failed")
	}
}

// ---- Error Handling Tests ----

func TestPLNDeserializeExpression(t *testing.T) {
	// PLN should reject expressions (code execution)
	result := testPLNCode(`deserialize("1 + 1")`)
	_, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error for expression in PLN, got %T: %v", result, result)
	}
}

func TestPLNDeserializeFunctionCall(t *testing.T) {
	// PLN should reject function calls
	result := testPLNCode(`deserialize("print(42)")`)
	_, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error for function call in PLN, got %T: %v", result, result)
	}
}

// ---- File Loading Tests ----

func TestPLNFileLoading(t *testing.T) {
	// Create a temporary PLN file
	tmpDir := t.TempDir()
	plnFile := filepath.Join(tmpDir, "test.pln")
	content := `{
  name: "Test",
  count: 42,
  enabled: true
}`
	if err := os.WriteFile(plnFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test loading via file() builtin
	code := `
		let f = file("` + plnFile + `")
		let data <== f
		data
	`
	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %s", p.Errors()[0])
	}

	env := evaluator.NewEnvironment()
	env.Filename = plnFile
	result := evaluator.Eval(program, env)

	if err, isErr := result.(*evaluator.Error); isErr {
		t.Fatalf("unexpected error: %s", err.Message)
	}

	dictObj, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T: %v", result, result)
	}

	// Verify we have the expected fields
	if len(dictObj.Pairs) != 3 {
		t.Errorf("expected 3 pairs, got %d", len(dictObj.Pairs))
	}
	if _, hasName := dictObj.Pairs["name"]; !hasName {
		t.Error("missing 'name' field")
	}
	if _, hasCount := dictObj.Pairs["count"]; !hasCount {
		t.Error("missing 'count' field")
	}
	if _, hasEnabled := dictObj.Pairs["enabled"]; !hasEnabled {
		t.Error("missing 'enabled' field")
	}
}

func TestPLNBuiltinLoading(t *testing.T) {
	// Create a temporary PLN file
	tmpDir := t.TempDir()
	plnFile := filepath.Join(tmpDir, "test.pln")
	content := `[1, 2, 3, 4, 5]`
	if err := os.WriteFile(plnFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test loading via PLN() builtin
	code := `
		let f = PLN("` + plnFile + `")
		let data <== f
		data.length() == 5
	`
	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %s", p.Errors()[0])
	}

	env := evaluator.NewEnvironment()
	env.Filename = plnFile
	result := evaluator.Eval(program, env)

	boolObj, ok := result.(*evaluator.Boolean)
	if !ok {
		if err, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("unexpected error: %s", err.Message)
		}
		t.Fatalf("expected Boolean, got %T: %v", result, result)
	}
	if !boolObj.Value {
		t.Error("PLN builtin loading test failed")
	}
}

// ---- Edge Cases ----

func TestPLNTrailingComma(t *testing.T) {
	// PLN should accept trailing commas (like Parsley)
	result := testPLNCode(`deserialize("[1, 2, 3,]")`)
	arrObj, ok := result.(*evaluator.Array)
	if !ok {
		if err, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("unexpected error: %s", err.Message)
		}
		t.Fatalf("expected Array, got %T: %v", result, result)
	}
	if len(arrObj.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arrObj.Elements))
	}
}

func TestPLNQuotedKeys(t *testing.T) {
	// PLN should accept both quoted and unquoted keys
	result := testPLNCode(`deserialize("{\"name\": \"Alice\", age: 30}")`)
	dictObj, ok := result.(*evaluator.Dictionary)
	if !ok {
		if err, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("unexpected error: %s", err.Message)
		}
		t.Fatalf("expected Dictionary, got %T: %v", result, result)
	}
	if len(dictObj.Pairs) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(dictObj.Pairs))
	}
}

func TestPLNStringEscapes(t *testing.T) {
	// Test string escapes
	result := testPLNCode(`deserialize("\"hello\\nworld\"")`)
	strObj, ok := result.(*evaluator.String)
	if !ok {
		if err, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("unexpected error: %s", err.Message)
		}
		t.Fatalf("expected String, got %T: %v", result, result)
	}
	if !strings.Contains(strObj.Value, "\n") {
		t.Errorf("expected newline in string, got %q", strObj.Value)
	}
}

func TestPLNNegativeNumbers(t *testing.T) {
	result := testPLNCode(`deserialize("-42")`)
	intObj, ok := result.(*evaluator.Integer)
	if !ok {
		if err, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("unexpected error: %s", err.Message)
		}
		t.Fatalf("expected Integer, got %T: %v", result, result)
	}
	if intObj.Value != -42 {
		t.Errorf("expected -42, got %d", intObj.Value)
	}
}
