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
	input := `let {string, email, integer} = import @std/schema
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
	input := `let schema = import @std/schema
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
	input := `let schema = import @std/schema
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
	input := `let schema = import @std/schema
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
	input := `let schema = import @std/schema
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
	input := `let schema = import @std/schema
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
	input := `let schema = import @std/schema
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
	input := `let schema = import @std/schema
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
	input := `let schema = import @std/schema
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
	input := `let schema = import @std/schema
let UserSchema = schema.define("User", {
  email: schema.email({required: true}),
  age: schema.integer({min: 0})
})
let result = UserSchema.validate({
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
	input := `let schema = import @std/schema
let UserSchema = schema.define("User", {
  email: schema.email({required: true}),
  age: schema.integer({min: 0})
})
let result = UserSchema.validate({
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
	input := `let schema = import @std/schema
let UserSchema = schema.define("User", {
  email: schema.email({required: true})
})
let result = UserSchema.validate({
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
	input := `let schema = import @std/schema
let AgeSchema = schema.define("Age", {
  value: schema.integer({min: 0, max: 150})
})
let result = AgeSchema.validate({
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
	input := `let schema = import @std/schema
let NameSchema = schema.define("Name", {
  name: schema.string({min: 2, max: 50})
})
let result = NameSchema.validate({
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
	input := `let schema = import @std/schema
let UserSchema = schema.define("User", {
  email: schema.email({required: true})
})
let result = UserSchema.validate({})
let err = result.errors[0]
err.schema + "|" + err.field + "|" + err.message`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if str, ok := result.(*evaluator.String); ok {
		expected := "User|email|User schema: Field is required"
		if str.Value != expected {
			t.Errorf("expected '%s', got '%s'", expected, str.Value)
		}
	} else {
		t.Errorf("expected STRING, got %s", result.Type())
	}
}

// =============================================================================
// Method-Style API Tests
// =============================================================================

func TestSchemaValidateMethodStyle(t *testing.T) {
	input := `let schema = import @std/schema
let UserSchema = schema.define("User", {
  email: schema.email({required: true}),
  age: schema.integer({min: 0})
})
let result = UserSchema.validate({
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
			t.Error("expected validation to pass with method-style API")
		}
	} else {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}
}

func TestSchemaValidateMethodStyleWithErrors(t *testing.T) {
	input := `let schema = import @std/schema
let UserSchema = schema.define("User", {
  email: schema.email({required: true}),
  age: schema.integer({min: 0, max: 150})
})
// Method-style validation with errors
let result = UserSchema.validate({
  email: "invalid-email",
  age: 200
})
result.errors.length()`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if i, ok := result.(*evaluator.Integer); ok {
		if i.Value != 2 {
			t.Errorf("expected 2 errors (invalid email + age out of range), got %d", i.Value)
		}
	} else {
		t.Errorf("expected INTEGER, got %s", result.Type())
	}
}

func TestSchemaStillHasDictMethods(t *testing.T) {
	// Schemas should still support regular dictionary methods like keys()
	input := `let schema = import @std/schema
let UserSchema = schema.define("User", {
  email: schema.email({required: true}),
  age: schema.integer({min: 0})
})
// Schema dictionaries should still have dict methods
UserSchema.keys().length()`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if i, ok := result.(*evaluator.Integer); ok {
		// Should have "name" and "fields" keys (not __schema__ since it's internal)
		if i.Value != 2 {
			t.Errorf("expected 2 keys (name, fields), got %d", i.Value)
		}
	} else {
		t.Errorf("expected INTEGER, got %s", result.Type())
	}
}

// =============================================================================
// Auto Constraint Tests (SPEC-AUTO-001 through SPEC-AUTO-005)
// =============================================================================

// SPEC-AUTO-001: Auto fields should be skipped during validation (schema.define API)
func TestSchemaAutoFieldSkippedDuringValidation(t *testing.T) {
	input := `let schema = import @std/schema
let UserSchema = schema.define("User", {
  id: schema.integer({auto: true}),
  email: schema.email({required: true})
})
// Validate without providing id - should pass because id is auto
let result = UserSchema.validate({
  email: "test@example.com"
})
result.valid`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if b, ok := result.(*evaluator.Boolean); ok {
		if !b.Value {
			t.Error("SPEC-AUTO-001: expected validation to pass when auto field is missing")
		}
	} else {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}
}

// SPEC-AUTO-001: Auto fields should also be skipped even with null value (schema.define API)
func TestSchemaAutoFieldWithNullSkipsValidation(t *testing.T) {
	input := `let schema = import @std/schema
let UserSchema = schema.define("User", {
  id: schema.integer({auto: true}),
  name: schema.string({required: true})
})
// Validate with null id - should pass because id is auto
let result = UserSchema.validate({
  id: null,
  name: "Alice"
})
result.valid`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if b, ok := result.(*evaluator.Boolean); ok {
		if !b.Value {
			t.Error("SPEC-AUTO-001: expected validation to pass when auto field is null")
		}
	} else {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}
}

// SPEC-AUTO-005: auto and default MAY be combined (schema.define API)
func TestSchemaAutoWithDefault(t *testing.T) {
	input := `let schema = import @std/schema
let UserSchema = schema.define("User", {
  id: schema.integer({auto: true, default: 0}),
  name: schema.string({required: true})
})
UserSchema.name`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("SPEC-AUTO-005: expected auto+default to be allowed, got error: %s", result.Inspect())
	}

	if str, ok := result.(*evaluator.String); ok {
		if str.Value != "User" {
			t.Errorf("expected 'User', got '%s'", str.Value)
		}
	} else {
		t.Errorf("expected STRING, got %s", result.Type())
	}
}

// Test that auto field shows up in schema.define() field specs
func TestSchemaAutoFieldAccessible(t *testing.T) {
	input := `let schema = import @std/schema
let UserSchema = schema.define("User", {
  id: schema.integer({auto: true}),
  name: schema.string()
})
UserSchema.fields.id.auto`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if b, ok := result.(*evaluator.Boolean); ok {
		if !b.Value {
			t.Error("expected auto to be true for id field")
		}
	} else {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}
}

// =============================================================================
// Auto Constraint Tests with @schema syntax
// =============================================================================

// SPEC-AUTO-001: Auto fields skipped during validation (@schema syntax)
func TestSchemaDeclarationAutoFieldSkipped(t *testing.T) {
	input := `@schema UserAuto1 {
  id: id(auto)
  email: email(required)
}
// Create record without id - should be valid because id is auto
let user = UserAuto1({email: "test@example.com"})
user.validate().isValid()`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if b, ok := result.(*evaluator.Boolean); ok {
		if !b.Value {
			t.Error("SPEC-AUTO-001: expected record to be valid when auto field is missing")
		}
	} else {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}
}

// SPEC-AUTO-003: Auto fields MUST be immutable on update operations
func TestSchemaAutoFieldImmutableOnUpdate(t *testing.T) {
	input := `@schema UpdateUser {
  id: id(auto)
  name: string
}
let user = UpdateUser({name: "Alice"})
user.update({id: "new-id"})`

	result := evalSchemaTest(t, input)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Fatal("SPEC-AUTO-003: expected error when trying to update auto field")
	}

	err := result.(*evaluator.Error)
	if err.Code != "RECORD-0001" {
		t.Errorf("expected error code RECORD-0001, got %s", err.Code)
	}
}

// Test that updating non-auto fields works normally
func TestSchemaAutoFieldAllowsNonAutoUpdate(t *testing.T) {
	input := `@schema UpdateUser2 {
  id: id(auto)
  name: string
}
let user = UpdateUser2({name: "Alice"})
let updated = user.update({name: "Bob"})
updated.name`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if str, ok := result.(*evaluator.String); ok {
		if str.Value != "Bob" {
			t.Errorf("expected 'Bob', got '%s'", str.Value)
		}
	} else {
		t.Errorf("expected STRING, got %s", result.Type())
	}
}

// SPEC-AUTO-004: auto and required MUST NOT be combined (@schema syntax)
func TestSchemaAutoAndRequiredError(t *testing.T) {
	input := `@schema BadSchema {
  id: id(auto, required)
}`

	result := evalSchemaTest(t, input)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Fatal("SPEC-AUTO-004: expected error when combining auto and required")
	}

	err := result.(*evaluator.Error)
	if err.Code != "SCHEMA-0001" {
		t.Errorf("expected error code SCHEMA-0001, got %s", err.Code)
	}
}

// Test bare auto syntax: id(auto) instead of id(auto: true)
func TestSchemaBareBooleanAuto(t *testing.T) {
	input := `@schema BareUser {
  id: id(auto)
  name: string(required)
}
BareUser.fields.id.auto`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if b, ok := result.(*evaluator.Boolean); ok {
		if !b.Value {
			t.Error("expected auto to be true with bare boolean syntax")
		}
	} else {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}
}

// Test bare required syntax: string(required) instead of string(required: true)
func TestSchemaBareBooleanRequired(t *testing.T) {
	input := `@schema BareUser2 {
  name: string(required)
}
BareUser2.fields.name.required`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if b, ok := result.(*evaluator.Boolean); ok {
		if !b.Value {
			t.Error("expected required to be true with bare boolean syntax")
		}
	} else {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}
}

// =============================================================================
// ReadOnly Constraint Tests
// =============================================================================

// Test that readOnly constraint is parsed and stored
func TestSchemaReadOnlyFieldParsed(t *testing.T) {
	input := `@schema RoleUser {
  id: id(auto)
  name: string
  role: enum["user", "admin"](readOnly, default: "user")
}
RoleUser.fields.role.readOnly`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if b, ok := result.(*evaluator.Boolean); ok {
		if !b.Value {
			t.Error("expected readOnly to be true for role field")
		}
	} else {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}
}

// Test that readOnly fields are silently filtered during record creation
func TestSchemaReadOnlyFilteredOnCreate(t *testing.T) {
	input := `@schema ProtectedUser {
  name: string
  role: enum["user", "admin"](readOnly, default: "user")
}
// Try to set role to "admin" - should be filtered, default "user" applied
let user = ProtectedUser({name: "Alice", role: "admin"})
user.role`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if str, ok := result.(*evaluator.String); ok {
		if str.Value != "user" {
			t.Errorf("expected 'user' (default), got '%s' - readOnly field should filter input", str.Value)
		}
	} else {
		t.Errorf("expected STRING, got %s", result.Type())
	}
}

// Test that readOnly fields are silently filtered during record.update()
func TestSchemaReadOnlyFilteredOnUpdate(t *testing.T) {
	input := `@schema ProtectedUser2 {
  name: string
  role: enum["user", "admin"](readOnly, default: "user")
}
let user = ProtectedUser2({name: "Alice"})
// Try to change role via update - should be silently ignored
let updated = user.update({name: "Bob", role: "admin"})
updated.role`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if str, ok := result.(*evaluator.String); ok {
		if str.Value != "user" {
			t.Errorf("expected 'user' (unchanged), got '%s' - readOnly field should be filtered on update", str.Value)
		}
	} else {
		t.Errorf("expected STRING, got %s", result.Type())
	}
}

// Test that readOnly fields are still readable
func TestSchemaReadOnlyFieldsReadable(t *testing.T) {
	input := `@schema ReadableUser {
  name: string
  role: string(readOnly, default: "user")
}
let user = ReadableUser({name: "Alice"})
user.role`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if str, ok := result.(*evaluator.String); ok {
		if str.Value != "user" {
			t.Errorf("expected 'user', got '%s'", str.Value)
		}
	} else {
		t.Errorf("expected STRING, got %s", result.Type())
	}
}

// Test multi-schema pattern: different schemas, different permissions
func TestSchemaMultiSchemaPattern(t *testing.T) {
	// Public schema with readOnly role
	input1 := `@schema PublicUser {
  name: string
  role: enum["user", "admin"](readOnly, default: "user")
}
let user = PublicUser({name: "Alice", role: "admin"})
user.role`

	result1 := evalSchemaTest(t, input1)
	if result1.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result1.Inspect())
	}
	if str, ok := result1.(*evaluator.String); ok {
		if str.Value != "user" {
			t.Errorf("PublicUser: expected 'user', got '%s'", str.Value)
		}
	}

	// Admin schema without readOnly - can set role
	input2 := `@schema AdminUser {
  name: string
  role: enum["user", "admin"](default: "user")
}
let user = AdminUser({name: "Alice", role: "admin"})
user.role`

	result2 := evalSchemaTest(t, input2)
	if result2.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result2.Inspect())
	}
	if str, ok := result2.(*evaluator.String); ok {
		if str.Value != "admin" {
			t.Errorf("AdminUser: expected 'admin', got '%s'", str.Value)
		}
	}
}

// Test that non-readOnly fields can still be updated normally
func TestSchemaReadOnlyDoesNotAffectOtherFields(t *testing.T) {
	input := `@schema MixedUser {
  name: string
  role: string(readOnly, default: "user")
}
let user = MixedUser({name: "Alice"})
let updated = user.update({name: "Bob"})
updated.name`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if str, ok := result.(*evaluator.String); ok {
		if str.Value != "Bob" {
			t.Errorf("expected 'Bob', got '%s'", str.Value)
		}
	} else {
		t.Errorf("expected STRING, got %s", result.Type())
	}
}

// Test readOnly with no default - should be null
func TestSchemaReadOnlyNoDefault(t *testing.T) {
	input := `@schema NoDefaultRole {
  name: string
  internalId: string(readOnly)
}
let user = NoDefaultRole({name: "Alice", internalId: "hacked"})
user.internalId == null`

	result := evalSchemaTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if b, ok := result.(*evaluator.Boolean); ok {
		if !b.Value {
			t.Error("expected internalId to be null (readOnly with no default)")
		}
	} else {
		t.Errorf("expected BOOLEAN, got %s", result.Type())
	}
}
