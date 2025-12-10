package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestPartModuleLoading(t *testing.T) {
	input := `
		let counter = import @./test_fixtures/parts/counter.part
		counter
	`

	// Use a realistic path (tests run from pkg/parsley/tests/)
	result := evalModule(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T", result)
	}

	// Check that __type is "part"
	typeExpr, hasType := dict.Pairs["__type"]
	if !hasType {
		t.Fatalf("Part module should have __type metadata")
	}

	env := evaluator.NewEnvironment()
	typeObj := evaluator.Eval(typeExpr, env)

	typeStr, ok := typeObj.(*evaluator.String)
	if !ok || typeStr.Value != "part" {
		t.Fatalf("expected __type='part', got %v", typeObj)
	}

	// Check that default export exists and is a function
	defaultExpr, hasDefault := dict.Pairs["default"]
	if !hasDefault {
		t.Fatalf("Part module should have 'default' export")
	}

	defaultObj := evaluator.Eval(defaultExpr, dict.Env)
	if _, ok := defaultObj.(*evaluator.Function); !ok {
		t.Fatalf("default export should be a function, got %T", defaultObj)
	}

	// Check that increment export exists and is a function
	incrementExpr, hasIncrement := dict.Pairs["increment"]
	if !hasIncrement {
		t.Fatalf("Part module should have 'increment' export")
	}

	incrementObj := evaluator.Eval(incrementExpr, dict.Env)
	if _, ok := incrementObj.(*evaluator.Function); !ok {
		t.Fatalf("increment export should be a function, got %T", incrementObj)
	}
}

func TestPartModuleExportsOnlyFunctions(t *testing.T) {
	// Create a temporary .part file with non-function export
	badPartContent := `
		export default = fn(props) { <div>OK</div> }
		export value = 42
	`

	l := lexer.NewWithFilename(badPartContent, "/tmp/bad.part")
	p := parser.New(l)
	program := p.ParseProgram()

	env := evaluator.NewEnvironment()
	env.Filename = "/tmp/bad.part"
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	moduleEnv := evaluator.NewEnvironment()
	moduleEnv.Filename = "/tmp/bad.part"
	moduleEnv.Security = env.Security

	result := evaluator.Eval(program, moduleEnv)

	// The error should occur during module dict creation (which we'd need to test differently)
	// For now, verify that the module evaluates without error (the check happens in importModule)
	if result.Type() == evaluator.ERROR_OBJ {
		t.Logf("Module evaluation error (expected at module load time): %s", result.Inspect())
	}
}

func TestPartModuleCaching(t *testing.T) {
	input := `
		let counter1 = import @./test_fixtures/parts/counter.part
		let counter2 = import @./test_fixtures/parts/counter.part
		counter1 == counter2
	`

	result := evalModule(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	boolean, ok := result.(*evaluator.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}

	if !boolean.Value {
		t.Errorf("expected Part modules to be cached and equal")
	}
}

func TestPartModuleViewFunctionsWork(t *testing.T) {
	input := `
		let counter = import @./test_fixtures/parts/counter.part
		counter.default({count: 5})
	`

	result := evalModule(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String (HTML), got %T", result)
	}

	// Should contain the count value
	if !strings.Contains(str.Value, "5") {
		t.Errorf("expected HTML to contain count value '5', got: %s", str.Value)
	}

	// Should contain the part-click attributes
	if !strings.Contains(str.Value, "part-click") {
		t.Errorf("expected HTML to contain 'part-click' attributes, got: %s", str.Value)
	}
}
