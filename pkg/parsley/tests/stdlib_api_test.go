package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// evalAPITest helper that evaluates Parsley code
func evalAPITest(t *testing.T, input string) evaluator.Object {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result == nil {
		t.Fatal("result is nil")
	}

	return result
}

// =============================================================================
// Module Import Tests
// =============================================================================

func TestAPIModuleImport(t *testing.T) {
	input := `let {public, notFound} = import @std/api
public`

	result := evalAPITest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.BUILTIN_OBJ {
		t.Errorf("expected BUILTIN, got %s", result.Type())
	}
}

func TestAPIModuleImportAll(t *testing.T) {
	input := `let api = import @std/api
api.public`

	result := evalAPITest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.BUILTIN_OBJ {
		t.Errorf("expected BUILTIN, got %s", result.Type())
	}
}

// =============================================================================
// Auth Wrapper Tests
// =============================================================================

func TestAPIPublicWrapper(t *testing.T) {
	input := `let api = import @std/api
let handler = api.public(fn(req) { "hello" })
handler`

	result := evalAPITest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	// Should return an AuthWrappedFunction
	if result.Type() != evaluator.FUNCTION_OBJ {
		t.Errorf("expected FUNCTION, got %s", result.Type())
	}
}

func TestAPIAuthWrapper(t *testing.T) {
	input := `let api = import @std/api
let handler = api.auth(fn(req) { "protected" })
handler`

	result := evalAPITest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.FUNCTION_OBJ {
		t.Errorf("expected FUNCTION, got %s", result.Type())
	}
}

func TestAPIAdminOnlyWrapper(t *testing.T) {
	input := `let api = import @std/api
let handler = api.adminOnly(fn(req) { "admin only" })
handler`

	result := evalAPITest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.FUNCTION_OBJ {
		t.Errorf("expected FUNCTION, got %s", result.Type())
	}
}

func TestAPIRolesWrapper(t *testing.T) {
	input := `let api = import @std/api
let handler = api.roles(["editor", "admin"], fn(req) { "role protected" })
handler`

	result := evalAPITest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.FUNCTION_OBJ {
		t.Errorf("expected FUNCTION, got %s", result.Type())
	}
}

// =============================================================================
// Error Helper Tests (unified error model — api.* returns *Error with UserDict)
// =============================================================================

// apiErrorTestCase defines a test for an api error helper
type apiErrorTestCase struct {
	name           string
	helperCall     string
	expectedCode   string
	expectedMsg    string
	expectedStatus int64
}

var apiErrorTests = []apiErrorTestCase{
	{"notFound custom", `api.notFound("User not found")`, "HTTP-404", "User not found", 404},
	{"notFound default", `api.notFound()`, "HTTP-404", "Not found", 404},
	{"forbidden custom", `api.forbidden("Access denied")`, "HTTP-403", "Access denied", 403},
	{"badRequest custom", `api.badRequest("Invalid input")`, "HTTP-400", "Invalid input", 400},
	{"unauthorized custom", `api.unauthorized("Not logged in")`, "HTTP-401", "Not logged in", 401},
	{"conflict custom", `api.conflict("Resource already exists")`, "HTTP-409", "Resource already exists", 409},
	{"serverError custom", `api.serverError("Something went wrong")`, "HTTP-500", "Something went wrong", 500},
}

func TestAPIErrorHelpers(t *testing.T) {
	for _, tt := range apiErrorTests {
		t.Run(tt.name, func(t *testing.T) {
			input := `let api = import @std/api
let {error} = try fn() { ` + tt.helperCall + ` }()
error`

			result := evalAPITest(t, input)

			dict, ok := result.(*evaluator.Dictionary)
			if !ok {
				t.Fatalf("expected error dict, got %s: %s", result.Type(), result.Inspect())
			}

			// Check code
			codeObj := evaluator.GetDictValue(dict, "code")
			if codeObj == nil {
				t.Fatal("error dict missing 'code' key")
			}
			if codeStr, ok := codeObj.(*evaluator.String); !ok || codeStr.Value != tt.expectedCode {
				t.Errorf("expected code %q, got %s", tt.expectedCode, codeObj.Inspect())
			}

			// Check message
			msgObj := evaluator.GetDictValue(dict, "message")
			if msgObj == nil {
				t.Fatal("error dict missing 'message' key")
			}
			if msgStr, ok := msgObj.(*evaluator.String); !ok || msgStr.Value != tt.expectedMsg {
				t.Errorf("expected message %q, got %s", tt.expectedMsg, msgObj.Inspect())
			}

			// Check status
			statusObj := evaluator.GetDictValue(dict, "status")
			if statusObj == nil {
				t.Fatal("error dict missing 'status' key")
			}
			if statusInt, ok := statusObj.(*evaluator.Integer); !ok || statusInt.Value != tt.expectedStatus {
				t.Errorf("expected status %d, got %s", tt.expectedStatus, statusObj.Inspect())
			}
		})
	}
}

// =============================================================================
// Wrapper Type Error Tests
// =============================================================================

func TestAPIPublicTypeError(t *testing.T) {
	input := `let api = import @std/api
api.public("not a function")`

	result := evalAPITest(t, input)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Errorf("expected error for non-function argument, got %s", result.Type())
	}
}

func TestAPIRolesTypeError(t *testing.T) {
	input := `let api = import @std/api
api.roles("not an array", fn(req) { "test" })`

	result := evalAPITest(t, input)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Errorf("expected error for non-array roles, got %s", result.Type())
	}
}

// =============================================================================
// Wrapped Function Execution Tests
// =============================================================================

func TestAPIWrappedFunctionExecutes(t *testing.T) {
	input := `let api = import @std/api
let handler = api.public(fn(x) { x * 2 })
handler(21)`

	result := evalAPITest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if num, ok := result.(*evaluator.Integer); ok {
		if num.Value != 42 {
			t.Errorf("expected 42, got %d", num.Value)
		}
	} else {
		t.Errorf("expected INTEGER, got %s", result.Type())
	}
}
