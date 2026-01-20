package pln_test

import (
	"os"
	"path/filepath"
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
