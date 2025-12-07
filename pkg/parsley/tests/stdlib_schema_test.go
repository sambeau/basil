package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// evalSchemaTest helper that evaluates Parsley code
func evalSchemaTest(t *testing.T, input string) evaluator.Object {
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

func TestSchemaModuleImport(t *testing.T) {
	input := `let {string, email, integer} = import("std/schema")
string`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.BUILTIN_OBJ {
		t.Errorf("expected BUILTIN, got %s", result.Type())
	}
}

func TestSchemaModuleImportAll(t *testing.T) {
	input := `let schema = import("std/schema")
schema.string`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.BUILTIN_OBJ {
		t.Errorf("expected BUILTIN, got %s", result.Type())
	}
}

// =============================================================================
// Type Factory Tests
// =============================================================================

func TestSchemaStringType(t *testing.T) {
	input := `let schema = import("std/schema")
let spec = schema.string()
spec.type`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.STRING_OBJ {
		t.Errorf("expected STRING, got %s", result.Type())
	}

	if result.(*evaluator.String).Value != "string" {
		t.Errorf("expected 'string', got '%s'", result.(*evaluator.String).Value)
	}
}

func TestSchemaStringTypeWithOptions(t *testing.T) {
	input := `let schema = import("std/schema")
let spec = schema.string({min: 3, max: 100, required: true})
spec.required`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.BOOLEAN_OBJ {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}

	if !result.(*evaluator.Boolean).Value {
		t.Error("expected required to be true")
	}
}

func TestSchemaEmailType(t *testing.T) {
	input := `let schema = import("std/schema")
let spec = schema.email()
spec.type`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if str, ok := result.(*evaluator.String); ok {
		if str.Value != "email" {
			t.Errorf("expected 'email', got '%s'", str.Value)
		}
	} else {
		t.Errorf("expected STRING, got %s", result.Type())
	}
}

func TestSchemaIntegerType(t *testing.T) {
	input := `let schema = import("std/schema")
let spec = schema.integer({min: 0, max: 100})
spec.type`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if str, ok := result.(*evaluator.String); ok {
		if str.Value != "integer" {
			t.Errorf("expected 'integer', got '%s'", str.Value)
		}
	} else {
		t.Errorf("expected STRING, got %s", result.Type())
	}
}

func TestSchemaEnumType(t *testing.T) {
	input := `let schema = import("std/schema")
let spec = schema.enum("pending", "active", "completed")
spec.type`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if str, ok := result.(*evaluator.String); ok {
		if str.Value != "enum" {
			t.Errorf("expected 'enum', got '%s'", str.Value)
		}
	} else {
		t.Errorf("expected STRING, got %s", result.Type())
	}
}

func TestSchemaIDType(t *testing.T) {
	input := `let schema = import("std/schema")
let spec = schema.id()
spec.format`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if str, ok := result.(*evaluator.String); ok {
		if str.Value != "ulid" {
			t.Errorf("expected 'ulid', got '%s'", str.Value)
		}
	} else {
		t.Errorf("expected STRING, got %s", result.Type())
	}
}

// =============================================================================
// Schema Definition Tests
// =============================================================================

func TestSchemaDefine(t *testing.T) {
	input := `let schema = import("std/schema")
let UserSchema = schema.define("User", {
  email: schema.email({required: true}),
  age: schema.integer({min: 0})
})
UserSchema.name`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if str, ok := result.(*evaluator.String); ok {
		if str.Value != "User" {
			t.Errorf("expected 'User', got '%s'", str.Value)
		}
	} else {
		t.Errorf("expected STRING, got %s", result.Type())
	}
}

// =============================================================================
// Validation Tests
// =============================================================================

func TestSchemaValidateValid(t *testing.T) {
	input := `let schema = import("std/schema")
let UserSchema = schema.define("User", {
  email: schema.email({required: true}),
  age: schema.integer({min: 0})
})
let result = schema.validate(UserSchema, {
  email: "test@example.com",
  age: 25
})
result.valid`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if b, ok := result.(*evaluator.Boolean); ok {
		if !b.Value {
			t.Error("expected validation to pass")
		}
	} else {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}
}

func TestSchemaValidateMissingRequired(t *testing.T) {
	input := `let schema = import("std/schema")
let UserSchema = schema.define("User", {
  email: schema.email({required: true}),
  age: schema.integer({min: 0})
})
let result = schema.validate(UserSchema, {
  age: 25
})
result.valid`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if b, ok := result.(*evaluator.Boolean); ok {
		if b.Value {
			t.Error("expected validation to fail for missing required field")
		}
	} else {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}
}

func TestSchemaValidateInvalidEmail(t *testing.T) {
	input := `let schema = import("std/schema")
let UserSchema = schema.define("User", {
  email: schema.email({required: true})
})
let result = schema.validate(UserSchema, {
  email: "not-an-email"
})
result.valid`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if b, ok := result.(*evaluator.Boolean); ok {
		if b.Value {
			t.Error("expected validation to fail for invalid email")
		}
	} else {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}
}

func TestSchemaValidateIntegerRange(t *testing.T) {
	input := `let schema = import("std/schema")
let AgeSchema = schema.define("Age", {
  value: schema.integer({min: 0, max: 150})
})
let result = schema.validate(AgeSchema, {
  value: -5
})
result.valid`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if b, ok := result.(*evaluator.Boolean); ok {
		if b.Value {
			t.Error("expected validation to fail for value below minimum")
		}
	} else {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}
}

func TestSchemaValidateStringLength(t *testing.T) {
	input := `let schema = import("std/schema")
let NameSchema = schema.define("Name", {
  name: schema.string({min: 2, max: 50})
})
let result = schema.validate(NameSchema, {
  name: "X"
})
result.valid`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if b, ok := result.(*evaluator.Boolean); ok {
		if b.Value {
			t.Error("expected validation to fail for string below minimum length")
		}
	} else {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}
}

func TestSchemaValidateErrorDetails(t *testing.T) {
	input := `let schema = import("std/schema")
let UserSchema = schema.define("User", {
  email: schema.email({required: true})
})
let result = schema.validate(UserSchema, {})
result.errors[0].field`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if str, ok := result.(*evaluator.String); ok {
		if str.Value != "email" {
			t.Errorf("expected error field 'email', got '%s'", str.Value)
		}
	} else {
		t.Errorf("expected STRING, got %s", result.Type())
	}
}
