package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func evalModule(input string, filename string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	env.Filename = filename
	// Enable execute-all for module import tests
	env.Security = &evaluator.SecurityPolicy{
		AllowExecuteAll: true,
	}
	return evaluator.Eval(program, env)
}

func TestBasicModuleImport(t *testing.T) {
	input := `
		let mod = import @./test_fixtures/modules/simple.pars
		mod.value
	`

	result := evalModule(input, "./test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	integer, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}

	if integer.Value != 42 {
		t.Errorf("expected 42, got %d", integer.Value)
	}
}

func TestModuleDestructuring(t *testing.T) {
	input := `
		let {add, PI} = import @./test_fixtures/modules/math.pars
		add(10, 32)
	`

	result := evalModule(input, "./test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	integer, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}

	if integer.Value != 42 {
		t.Errorf("expected 42, got %d", integer.Value)
	}
}

func TestModuleAlias(t *testing.T) {
	input := `
		let {square as sq} = import @./test_fixtures/modules/math.pars
		sq(8)
	`

	result := evalModule(input, "./test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	integer, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}

	if integer.Value != 64 {
		t.Errorf("expected 64, got %d", integer.Value)
	}
}

func TestModuleCaching(t *testing.T) {
	input := `
		let mod1 = import @./test_fixtures/modules/simple.pars
		let mod2 = import @./test_fixtures/modules/simple.pars
		mod1 == mod2
	`

	result := evalModule(input, "./test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	boolean, ok := result.(*evaluator.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", result)
	}

	if !boolean.Value {
		t.Errorf("expected modules to be cached and equal")
	}
}

func TestModuleStringPath(t *testing.T) {
	input := `
		import @./test_fixtures/modules/simple.pars as mod
		mod.text
	`

	result := evalModule(input, "./test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	if str.Value != "hello" {
		t.Errorf("expected 'hello', got %s", str.Value)
	}
}

func TestModuleNotFound(t *testing.T) {
	input := `
		let mod = import @./nonexistent.pars
	`

	result := evalModule(input, "./test.pars")

	if result.Type() != evaluator.ERROR_OBJ {
		t.Fatalf("expected error for missing module, got %T", result)
	}

	errStr := result.Inspect()
	if !strings.Contains(strings.ToLower(errStr), strings.ToLower("module not found")) {
		t.Errorf("expected file not found error, got: %s", errStr)
	}
}

func TestModuleFunction(t *testing.T) {
	input := `
		let math = import @./test_fixtures/modules/math.pars
		math.add(math.multiply(2, 3), 4)
	`

	result := evalModule(input, "./test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	integer, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}

	// (2 * 3) + 4 = 10
	if integer.Value != 10 {
		t.Errorf("expected 10, got %d", integer.Value)
	}
}

func TestModuleClosures(t *testing.T) {
	input := `
		let {double} = import @./test_fixtures/modules/simple.pars
		double(21)
	`

	result := evalModule(input, "./test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	integer, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", result)
	}

	if integer.Value != 42 {
		t.Errorf("expected 42, got %d", integer.Value)
	}
}

// BUG-005: using a non-destructured module dictionary as a function or
// component should explain the mistake and show the destructuring fix.

// assertModuleDictError checks that result is an error with the given code,
// a message explaining module exports, and a hint showing the destructuring fix.
func assertModuleDictError(t *testing.T, result evaluator.Object, wantCode string) {
	t.Helper()

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected error, got %T: %s", result, result.Inspect())
	}

	if errObj.Code != wantCode {
		t.Errorf("expected code %s, got %s (%s)", wantCode, errObj.Code, errObj.Message)
	}
	if !strings.Contains(errObj.Message, "dictionary of module exports") {
		t.Errorf("expected module-exports explanation, got: %s", errObj.Message)
	}
	hints := strings.Join(errObj.Hints, "\n")
	if !strings.Contains(hints, "let { Logo } = import") {
		t.Errorf("expected destructuring hint naming 'Logo', got hints: %s", hints)
	}
}

func TestModuleDictCalledAsFunction(t *testing.T) {
	input := `
		let Logo = import @./test_fixtures/modules/logo.pars
		Logo()
	`

	result := evalModule(input, "./test.pars")
	assertModuleDictError(t, result, "CALL-0006")
}

func TestModuleDictUsedAsComponent(t *testing.T) {
	input := `
		let Logo = import @./test_fixtures/modules/logo.pars
		<Logo/>
	`

	result := evalModule(input, "./test.pars")
	assertModuleDictError(t, result, "COMP-0003")
}

func TestModuleDictUsedAsComponentTagPair(t *testing.T) {
	input := `
		let Logo = import @./test_fixtures/modules/logo.pars
		<Logo>"content"</Logo>
	`

	result := evalModule(input, "./test.pars")
	assertModuleDictError(t, result, "COMP-0003")
}

func TestModuleDestructuredComponentWorks(t *testing.T) {
	input := `
		let {Logo} = import @./test_fixtures/modules/logo.pars
		<Logo/>
	`

	result := evalModule(input, "./test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if !strings.Contains(result.Inspect(), "<img") {
		t.Errorf("expected rendered img tag, got: %s", result.Inspect())
	}
}
