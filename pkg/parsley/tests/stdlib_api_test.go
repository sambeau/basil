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
// Error Helper Tests
// =============================================================================

func TestAPINotFound(t *testing.T) {
	input := `let api = import @std/api
let err = api.notFound("User not found")
err`

	result := evalAPITest(t, input)

	// Should return an APIError
	if _, ok := result.(*evaluator.APIError); !ok {
		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("evaluation error: %s", result.Inspect())
		}
		t.Errorf("expected APIError, got %s", result.Type())
	}
}

func TestAPINotFoundDefault(t *testing.T) {
	input := `let api = import @std/api
let err = api.notFound()
err`

	result := evalAPITest(t, input)

	if apiErr, ok := result.(*evaluator.APIError); ok {
		if apiErr.Status != 404 {
			t.Errorf("expected status 404, got %d", apiErr.Status)
		}
	} else if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}
}

func TestAPIForbidden(t *testing.T) {
	input := `let api = import @std/api
let err = api.forbidden("Access denied")
err`

	result := evalAPITest(t, input)

	if apiErr, ok := result.(*evaluator.APIError); ok {
		if apiErr.Status != 403 {
			t.Errorf("expected status 403, got %d", apiErr.Status)
		}
	} else if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}
}

func TestAPIBadRequest(t *testing.T) {
	input := `let api = import @std/api
let err = api.badRequest("Invalid input")
err`

	result := evalAPITest(t, input)

	if apiErr, ok := result.(*evaluator.APIError); ok {
		if apiErr.Status != 400 {
			t.Errorf("expected status 400, got %d", apiErr.Status)
		}
	} else if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}
}

func TestAPIUnauthorized(t *testing.T) {
	input := `let api = import @std/api
let err = api.unauthorized("Not logged in")
err`

	result := evalAPITest(t, input)

	if apiErr, ok := result.(*evaluator.APIError); ok {
		if apiErr.Status != 401 {
			t.Errorf("expected status 401, got %d", apiErr.Status)
		}
	} else if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}
}

func TestAPIConflict(t *testing.T) {
	input := `let api = import @std/api
let err = api.conflict("Resource already exists")
err`

	result := evalAPITest(t, input)

	if apiErr, ok := result.(*evaluator.APIError); ok {
		if apiErr.Status != 409 {
			t.Errorf("expected status 409, got %d", apiErr.Status)
		}
	} else if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}
}

func TestAPIServerError(t *testing.T) {
	input := `let api = import @std/api
let err = api.serverError("Something went wrong")
err`

	result := evalAPITest(t, input)

	if apiErr, ok := result.(*evaluator.APIError); ok {
		if apiErr.Status != 500 {
			t.Errorf("expected status 500, got %d", apiErr.Status)
		}
	} else if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
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
