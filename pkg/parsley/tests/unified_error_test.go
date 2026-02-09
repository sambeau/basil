package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// evalUnifiedError evaluates Parsley code for unified error tests
func evalUnifiedError(t *testing.T, input string) evaluator.Object {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// getDictString extracts a string value from a dictionary by key
func getDictString(t *testing.T, dict *evaluator.Dictionary, key string) string {
	t.Helper()
	obj := evaluator.GetDictValue(dict, key)
	if obj == nil {
		t.Fatalf("dict missing key %q", key)
	}
	str, ok := obj.(*evaluator.String)
	if !ok {
		t.Fatalf("expected string for key %q, got %T: %s", key, obj, obj.Inspect())
	}
	return str.Value
}

// getDictInt extracts an integer value from a dictionary by key
func getDictInt(t *testing.T, dict *evaluator.Dictionary, key string) int64 {
	t.Helper()
	obj := evaluator.GetDictValue(dict, key)
	if obj == nil {
		t.Fatalf("dict missing key %q", key)
	}
	num, ok := obj.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected integer for key %q, got %T: %s", key, obj, obj.Inspect())
	}
	return num.Value
}

// =============================================================================
// T1: fail(string) backward compatibility — produces *Error with UserDict
// =============================================================================

func TestUnifiedFailString(t *testing.T) {
	input := `
let {result, error} = try fn() { fail("oops") }()
error
`
	result := evalUnifiedError(t, input)

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected error dict, got %T: %s", result, result.Inspect())
	}

	if msg := getDictString(t, dict, "message"); msg != "oops" {
		t.Errorf("expected message 'oops', got %q", msg)
	}
	if code := getDictString(t, dict, "code"); code != "USER-0001" {
		t.Errorf("expected code 'USER-0001', got %q", code)
	}
}

func TestUnifiedFailStringResultNull(t *testing.T) {
	input := `
let {result, error} = try fn() { fail("oops") }()
result
`
	result := evalUnifiedError(t, input)
	if result != evaluator.NULL {
		t.Errorf("expected null result, got %T: %s", result, result.Inspect())
	}
}

// =============================================================================
// T2: fail(dict) with all fields
// =============================================================================

func TestUnifiedFailDict(t *testing.T) {
	input := `
let {result, error} = try fn() {
  fail({code: "NO_STOCK", message: "Out of stock", status: 400})
}()
error
`
	result := evalUnifiedError(t, input)

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected error dict, got %T: %s", result, result.Inspect())
	}

	if msg := getDictString(t, dict, "message"); msg != "Out of stock" {
		t.Errorf("expected message 'Out of stock', got %q", msg)
	}
	if code := getDictString(t, dict, "code"); code != "NO_STOCK" {
		t.Errorf("expected code 'NO_STOCK', got %q", code)
	}
	if status := getDictInt(t, dict, "status"); status != 400 {
		t.Errorf("expected status 400, got %d", status)
	}
}

func TestUnifiedFailDictCustomFields(t *testing.T) {
	input := `
let {error} = try fn() {
  fail({message: "Quota exceeded", code: "RATE-001", status: 429, retryAfter: 60})
}()
error
`
	result := evalUnifiedError(t, input)

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected error dict, got %T: %s", result, result.Inspect())
	}

	if msg := getDictString(t, dict, "message"); msg != "Quota exceeded" {
		t.Errorf("expected message 'Quota exceeded', got %q", msg)
	}
	if code := getDictString(t, dict, "code"); code != "RATE-001" {
		t.Errorf("expected code 'RATE-001', got %q", code)
	}
	if status := getDictInt(t, dict, "status"); status != 429 {
		t.Errorf("expected status 429, got %d", status)
	}
	if retryAfter := getDictInt(t, dict, "retryAfter"); retryAfter != 60 {
		t.Errorf("expected retryAfter 60, got %d", retryAfter)
	}
}

// =============================================================================
// T3: fail(dict) without message key → TYPE error
// =============================================================================

func TestUnifiedFailDictNoMessage(t *testing.T) {
	input := `fail({code: "X"})`
	result := evalUnifiedError(t, input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T: %s", result, result.Inspect())
	}
	if errObj.Class != evaluator.ClassType {
		t.Errorf("expected ClassType error, got %s", errObj.Class)
	}
}

// =============================================================================
// T4: fail(non-string-non-dict) → TYPE error
// =============================================================================

func TestUnifiedFailInteger(t *testing.T) {
	input := `fail(123)`
	result := evalUnifiedError(t, input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T: %s", result, result.Inspect())
	}
	if errObj.Class != evaluator.ClassType {
		t.Errorf("expected ClassType error, got %s", errObj.Class)
	}
}

func TestUnifiedFailBoolean(t *testing.T) {
	input := `fail(true)`
	result := evalUnifiedError(t, input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T: %s", result, result.Inspect())
	}
	if errObj.Class != evaluator.ClassType {
		t.Errorf("expected ClassType error, got %s", errObj.Class)
	}
}

// =============================================================================
// T5–T6: api.* helpers produce unified error dicts with correct fields
// =============================================================================

func TestUnifiedAPINotFound(t *testing.T) {
	input := `
let api = import @std/api
let {error} = try fn() { api.notFound("User not found") }()
error
`
	result := evalUnifiedError(t, input)
	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected error dict, got %T: %s", result, result.Inspect())
	}

	if code := getDictString(t, dict, "code"); code != "HTTP-404" {
		t.Errorf("expected code 'HTTP-404', got %q", code)
	}
	if msg := getDictString(t, dict, "message"); msg != "User not found" {
		t.Errorf("expected message 'User not found', got %q", msg)
	}
	if status := getDictInt(t, dict, "status"); status != 404 {
		t.Errorf("expected status 404, got %d", status)
	}
}

func TestUnifiedAPIBadRequest(t *testing.T) {
	input := `
let api = import @std/api
let {error} = try fn() { api.badRequest("Invalid email") }()
error
`
	result := evalUnifiedError(t, input)
	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected error dict, got %T: %s", result, result.Inspect())
	}

	if code := getDictString(t, dict, "code"); code != "HTTP-400" {
		t.Errorf("expected code 'HTTP-400', got %q", code)
	}
	if status := getDictInt(t, dict, "status"); status != 400 {
		t.Errorf("expected status 400, got %d", status)
	}
}

func TestUnifiedAPIForbidden(t *testing.T) {
	input := `
let api = import @std/api
let {error} = try fn() { api.forbidden("Access denied") }()
error
`
	result := evalUnifiedError(t, input)
	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected error dict, got %T: %s", result, result.Inspect())
	}

	if code := getDictString(t, dict, "code"); code != "HTTP-403" {
		t.Errorf("expected code 'HTTP-403', got %q", code)
	}
	if status := getDictInt(t, dict, "status"); status != 403 {
		t.Errorf("expected status 403, got %d", status)
	}
}

func TestUnifiedAPIUnauthorized(t *testing.T) {
	input := `
let api = import @std/api
let {error} = try fn() { api.unauthorized("Not logged in") }()
error
`
	result := evalUnifiedError(t, input)
	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected error dict, got %T: %s", result, result.Inspect())
	}

	if code := getDictString(t, dict, "code"); code != "HTTP-401" {
		t.Errorf("expected code 'HTTP-401', got %q", code)
	}
	if status := getDictInt(t, dict, "status"); status != 401 {
		t.Errorf("expected status 401, got %d", status)
	}
}

func TestUnifiedAPIConflict(t *testing.T) {
	input := `
let api = import @std/api
let {error} = try fn() { api.conflict("Already exists") }()
error
`
	result := evalUnifiedError(t, input)
	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected error dict, got %T: %s", result, result.Inspect())
	}

	if code := getDictString(t, dict, "code"); code != "HTTP-409" {
		t.Errorf("expected code 'HTTP-409', got %q", code)
	}
	if status := getDictInt(t, dict, "status"); status != 409 {
		t.Errorf("expected status 409, got %d", status)
	}
}

func TestUnifiedAPIServerError(t *testing.T) {
	input := `
let api = import @std/api
let {error} = try fn() { api.serverError("Something broke") }()
error
`
	result := evalUnifiedError(t, input)
	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected error dict, got %T: %s", result, result.Inspect())
	}

	if code := getDictString(t, dict, "code"); code != "HTTP-500" {
		t.Errorf("expected code 'HTTP-500', got %q", code)
	}
	if status := getDictInt(t, dict, "status"); status != 500 {
		t.Errorf("expected status 500, got %d", status)
	}
}

// =============================================================================
// T7: Error dict truthiness — if (error) guard works
// =============================================================================

func TestUnifiedErrorDictTruthy(t *testing.T) {
	input := `
let {result, error} = try fn() { fail("bad") }()
if (error) { "has error" } else { "no error" }
`
	result := evalUnifiedError(t, input)
	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "has error" {
		t.Errorf("expected 'has error', got %q", str.Value)
	}
}

func TestUnifiedNullErrorFalsy(t *testing.T) {
	input := `
let {result, error} = try fn() { 42 }()
if (error) { "has error" } else { "no error" }
`
	result := evalUnifiedError(t, input)
	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "no error" {
		t.Errorf("expected 'no error', got %q", str.Value)
	}
}

// =============================================================================
// T8–T9: String coercion — "" + errorDict → message
// =============================================================================

func TestUnifiedStringCoercionSimple(t *testing.T) {
	input := `
let {error} = try fn() { fail("oops") }()
"Error: " + error
`
	result := evalUnifiedError(t, input)
	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "Error: oops" {
		t.Errorf("expected 'Error: oops', got %q", str.Value)
	}
}

func TestUnifiedStringCoercionWithStatus(t *testing.T) {
	input := `
let {error} = try fn() { fail({message: "bad input", status: 400}) }()
"Error: " + error
`
	result := evalUnifiedError(t, input)
	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "Error: bad input" {
		t.Errorf("expected 'Error: bad input', got %q", str.Value)
	}
}

func TestUnifiedStringCoercionNoMessageKey(t *testing.T) {
	// Dict without "message" key should use normal Inspect behavior
	input := `"Value: " + {name: "x"}`
	result := evalUnifiedError(t, input)
	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	// Should contain dict Inspect output, not a message
	if strings.Contains(str.Value, "x") && !strings.Contains(str.Value, "name") {
		t.Errorf("dict without message should use Inspect, got %q", str.Value)
	}
}

// =============================================================================
// T14: Internal catchable error produces dict with message and code
// =============================================================================

func TestUnifiedInternalCatchableError(t *testing.T) {
	// url() with invalid URL returns a Format-class catchable error (no UserDict)
	input := `
let {error} = try url(":::invalid:::")
error
`
	result := evalUnifiedError(t, input)

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected error dict, got %T: %s", result, result.Inspect())
	}

	// Should have message
	msgObj := evaluator.GetDictValue(dict, "message")
	if msgObj == nil {
		t.Fatal("error dict missing 'message' key")
	}
	msgStr, ok := msgObj.(*evaluator.String)
	if !ok {
		t.Fatalf("expected string message, got %T", msgObj)
	}
	if msgStr.Value == "" {
		t.Error("expected non-empty error message")
	}

	// Should have code
	codeObj := evaluator.GetDictValue(dict, "code")
	if codeObj == nil {
		t.Fatal("error dict missing 'code' key")
	}
	codeStr, ok := codeObj.(*evaluator.String)
	if !ok {
		t.Fatalf("expected string code, got %T", codeObj)
	}
	if codeStr.Value == "" {
		t.Error("expected non-empty error code")
	}
}

// =============================================================================
// T15: Non-catchable errors still propagate (not caught by try)
// =============================================================================

func TestUnifiedNonCatchableTypePropagates(t *testing.T) {
	input := `try url(123)`
	result := evalUnifiedError(t, input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error to propagate, got %T: %s", result, result.Inspect())
	}
	if errObj.Class != evaluator.ClassType {
		t.Errorf("expected ClassType, got %s", errObj.Class)
	}
}

func TestUnifiedNonCatchableArityPropagates(t *testing.T) {
	input := `try time()`
	result := evalUnifiedError(t, input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error to propagate, got %T: %s", result, result.Inspect())
	}
	if errObj.Class != evaluator.ClassArity {
		t.Errorf("expected ClassArity, got %s", errObj.Class)
	}
}

// =============================================================================
// T11: record.failIfInvalid() — valid record returns record
// =============================================================================

func TestUnifiedFailIfInvalidValid(t *testing.T) {
	input := `
@schema UserV1 { name: string(required) email: string(required) }
let user = UserV1({name: "Alice", email: "alice@example.com"})
let validated = user.validate()
let result = validated.failIfInvalid()
result.name
`
	result := evalUnifiedError(t, input)

	str, ok := result.(*evaluator.String)
	if !ok {
		if errObj, ok := result.(*evaluator.Error); ok {
			t.Fatalf("got error: %s", errObj.Inspect())
		}
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "Alice" {
		t.Errorf("expected 'Alice', got %q", str.Value)
	}
}

// =============================================================================
// T12: record.failIfInvalid() — invalid record produces structured error
// =============================================================================

func TestUnifiedFailIfInvalidInvalid(t *testing.T) {
	input := `
@schema UserV2 { name: string(required) email: string(required) }
let user = UserV2({name: ""})
let validated = user.validate()
let {error} = try fn() { validated.failIfInvalid() }()
error
`
	result := evalUnifiedError(t, input)

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		if errObj, ok := result.(*evaluator.Error); ok {
			t.Fatalf("got error: %s", errObj.Inspect())
		}
		t.Fatalf("expected error dict, got %T: %s", result, result.Inspect())
	}

	if code := getDictString(t, dict, "code"); code != "VALIDATION" {
		t.Errorf("expected code 'VALIDATION', got %q", code)
	}
	if msg := getDictString(t, dict, "message"); msg != "Validation failed" {
		t.Errorf("expected message 'Validation failed', got %q", msg)
	}
	if status := getDictInt(t, dict, "status"); status != 400 {
		t.Errorf("expected status 400, got %d", status)
	}

	// Check fields array exists
	fieldsObj := evaluator.GetDictValue(dict, "fields")
	if fieldsObj == nil {
		t.Fatal("error dict missing 'fields' key")
	}
	fieldsArr, ok := fieldsObj.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array for fields, got %T", fieldsObj)
	}
	if len(fieldsArr.Elements) == 0 {
		t.Error("expected at least one field error")
	}
}

// =============================================================================
// T13: Existing validation API still works
// =============================================================================

func TestUnifiedExistingValidationAPIsIsValid(t *testing.T) {
	input := `
@schema UserV3 { name: string(required) }
let user = UserV3({name: null})
let validated = user.validate()
validated.isValid()
`
	result := evalUnifiedError(t, input)

	boolVal, ok := result.(*evaluator.Boolean)
	if !ok {
		if errObj, ok := result.(*evaluator.Error); ok {
			t.Fatalf("got error: %s", errObj.Inspect())
		}
		t.Fatalf("expected Boolean, got %T: %s", result, result.Inspect())
	}
	if boolVal.Value {
		t.Error("expected isValid() to be false")
	}
}

func TestUnifiedExistingValidationAPIsErrorList(t *testing.T) {
	input := `
@schema UserV4 { name: string(required) }
let user = UserV4({name: null})
let validated = user.validate()
validated.errorList().length()
`
	result := evalUnifiedError(t, input)

	intVal, ok := result.(*evaluator.Integer)
	if !ok {
		if errObj, ok := result.(*evaluator.Error); ok {
			t.Fatalf("got error: %s", errObj.Inspect())
		}
		t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
	}
	if intVal.Value == 0 {
		t.Error("expected errorList to have entries")
	}
}

// =============================================================================
// failIfInvalid on un-validated record returns record (no-op)
// =============================================================================

func TestUnifiedFailIfInvalidUnvalidated(t *testing.T) {
	input := `
@schema UserV5 { name: string(required) }
let user = UserV5({name: "Alice"})
let result = user.failIfInvalid()
result.name
`
	result := evalUnifiedError(t, input)

	str, ok := result.(*evaluator.String)
	if !ok {
		if errObj, ok := result.(*evaluator.Error); ok {
			t.Fatalf("got error: %s", errObj.Inspect())
		}
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "Alice" {
		t.Errorf("expected 'Alice', got %q", str.Value)
	}
}

// =============================================================================
// fail() Error struct has correct UserDict, Code, and Message
// =============================================================================

func TestUnifiedFailErrorStruct(t *testing.T) {
	input := `fail("test message")`
	result := evalUnifiedError(t, input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T: %s", result, result.Inspect())
	}

	if errObj.Message != "test message" {
		t.Errorf("expected Message 'test message', got %q", errObj.Message)
	}
	if errObj.Code != "USER-0001" {
		t.Errorf("expected Code 'USER-0001', got %q", errObj.Code)
	}
	if errObj.Class != evaluator.ClassValue {
		t.Errorf("expected ClassValue, got %s", errObj.Class)
	}
	if errObj.UserDict == nil {
		t.Fatal("expected UserDict to be set")
	}
}

func TestUnifiedFailDictErrorStruct(t *testing.T) {
	input := `fail({message: "bad", code: "CUSTOM-01", status: 422})`
	result := evalUnifiedError(t, input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T: %s", result, result.Inspect())
	}

	if errObj.Message != "bad" {
		t.Errorf("expected Message 'bad', got %q", errObj.Message)
	}
	if errObj.Code != "CUSTOM-01" {
		t.Errorf("expected Code 'CUSTOM-01', got %q", errObj.Code)
	}
	if errObj.UserDict == nil {
		t.Fatal("expected UserDict to be set")
	}

	// UserDict should be the original dict with all fields
	statusObj := evaluator.GetDictValue(errObj.UserDict, "status")
	if statusObj == nil {
		t.Fatal("UserDict missing 'status' key")
	}
	if num, ok := statusObj.(*evaluator.Integer); !ok || num.Value != 422 {
		t.Errorf("expected status 422, got %s", statusObj.Inspect())
	}
}

// =============================================================================
// fail(dict) without code field defaults to USER-0001
// =============================================================================

func TestUnifiedFailDictDefaultCode(t *testing.T) {
	input := `fail({message: "no code given"})`
	result := evalUnifiedError(t, input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T: %s", result, result.Inspect())
	}

	if errObj.Code != "USER-0001" {
		t.Errorf("expected default code 'USER-0001', got %q", errObj.Code)
	}
}

// =============================================================================
// fail(dict) with non-string message → TYPE error
// =============================================================================

func TestUnifiedFailDictNonStringMessage(t *testing.T) {
	input := `fail({message: 123})`
	result := evalUnifiedError(t, input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T: %s", result, result.Inspect())
	}
	if errObj.Class != evaluator.ClassType {
		t.Errorf("expected ClassType, got %s", errObj.Class)
	}
}

// =============================================================================
// api.* helpers via try preserve all dict fields
// =============================================================================

func TestUnifiedAPICatchedViaFunctionCode(t *testing.T) {
	input := `
let api = import @std/api
let handler = fn(id) {
  if (id == null) { api.notFound("Item not found") }
  {data: "found"}
}
let {error} = try handler(null)
error.code
`
	result := evalUnifiedError(t, input)

	str, ok := result.(*evaluator.String)
	if !ok {
		if errObj, ok := result.(*evaluator.Error); ok {
			t.Fatalf("got error: %s", errObj.Inspect())
		}
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "HTTP-404" {
		t.Errorf("expected 'HTTP-404', got %q", str.Value)
	}
}

func TestUnifiedAPICatchedViaFunctionStatus(t *testing.T) {
	input := `
let api = import @std/api
let handler = fn(id) {
  if (id == null) { api.notFound("Item not found") }
  {data: "found"}
}
let {error} = try handler(null)
error.status
`
	result := evalUnifiedError(t, input)

	num, ok := result.(*evaluator.Integer)
	if !ok {
		if errObj, ok := result.(*evaluator.Error); ok {
			t.Fatalf("got error: %s", errObj.Inspect())
		}
		t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
	}
	if num.Value != 404 {
		t.Errorf("expected 404, got %d", num.Value)
	}
}

func TestUnifiedAPICatchedViaFunctionResult(t *testing.T) {
	input := `
let api = import @std/api
let handler = fn(id) {
  if (id == null) { api.notFound("Item not found") }
  {data: "found"}
}
let {result} = try handler(null)
result
`
	result := evalUnifiedError(t, input)

	if result != evaluator.NULL {
		t.Errorf("expected null result, got %s", result.Inspect())
	}
}

// =============================================================================
// String coercion in template context
// =============================================================================

func TestUnifiedStringCoercionTemplate(t *testing.T) {
	input := "let {error} = try fn() { fail({message: \"disk full\", code: \"IO-001\"}) }()\n`Got error: {error}`"

	result := evalUnifiedError(t, input)

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "Got error: disk full" {
		t.Errorf("expected 'Got error: disk full', got %q", str.Value)
	}
}

// =============================================================================
// api.* error propagation through check..else
// =============================================================================

func TestUnifiedAPICheckElse(t *testing.T) {
	input := `
let api = import @std/api
let lookup = fn(id) {
  let found = null
  check found else api.notFound("Not found")
  found
}
let {error} = try lookup("123")
error.code
`
	result := evalUnifiedError(t, input)

	// The check..else evaluates api.notFound() which returns *Error,
	// which propagates through isError check in evalCheckStatement
	str, ok := result.(*evaluator.String)
	if !ok {
		if errObj, ok := result.(*evaluator.Error); ok {
			t.Fatalf("got propagated error: %s", errObj.Inspect())
		}
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "HTTP-404" {
		t.Errorf("expected 'HTTP-404', got %q", str.Value)
	}
}

// =============================================================================
// ApiFailError exported function (for server package)
// =============================================================================

func TestUnifiedApiFailErrorExport(t *testing.T) {
	err := evaluator.ApiFailError("HTTP-404", "Not found", 404)

	if err.Code != "HTTP-404" {
		t.Errorf("expected code 'HTTP-404', got %q", err.Code)
	}
	if err.Message != "Not found" {
		t.Errorf("expected message 'Not found', got %q", err.Message)
	}
	if err.Class != evaluator.ClassValue {
		t.Errorf("expected ClassValue, got %s", err.Class)
	}
	if err.UserDict == nil {
		t.Fatal("expected UserDict to be set")
	}

	statusObj := evaluator.GetDictValue(err.UserDict, "status")
	if statusObj == nil {
		t.Fatal("UserDict missing 'status'")
	}
	if num, ok := statusObj.(*evaluator.Integer); !ok || num.Value != 404 {
		t.Errorf("expected status 404, got %s", statusObj.Inspect())
	}
}

// =============================================================================
// Error message field access via dot notation
// =============================================================================

func TestUnifiedErrorDotAccessMessage(t *testing.T) {
	input := `
let {error} = try fn() { fail({message: "oops", code: "E1", status: 500}) }()
error.message
`
	result := evalUnifiedError(t, input)

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "oops" {
		t.Errorf("expected 'oops', got %q", str.Value)
	}
}

func TestUnifiedErrorDotAccessCode(t *testing.T) {
	input := `
let {error} = try fn() { fail({message: "oops", code: "E1", status: 500}) }()
error.code
`
	result := evalUnifiedError(t, input)

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "E1" {
		t.Errorf("expected 'E1', got %q", str.Value)
	}
}

func TestUnifiedErrorDotAccessStatus(t *testing.T) {
	input := `
let {error} = try fn() { fail({message: "oops", code: "E1", status: 500}) }()
error.status
`
	result := evalUnifiedError(t, input)

	num, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
	}
	if num.Value != 500 {
		t.Errorf("expected 500, got %d", num.Value)
	}
}

// =============================================================================
// Error dict from fail(string) has message and code fields
// =============================================================================

func TestUnifiedFailStringDictFieldsMessage(t *testing.T) {
	input := `
let {error} = try fn() { fail("simple error") }()
error.message
`
	result := evalUnifiedError(t, input)

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "simple error" {
		t.Errorf("expected 'simple error', got %q", str.Value)
	}
}

func TestUnifiedFailStringDictFieldsCode(t *testing.T) {
	input := `
let {error} = try fn() { fail("simple error") }()
error.code
`
	result := evalUnifiedError(t, input)

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "USER-0001" {
		t.Errorf("expected 'USER-0001', got %q", str.Value)
	}
}

// =============================================================================
// Multiple try catches in sequence
// =============================================================================

func TestUnifiedMultipleTryCatchesFirst(t *testing.T) {
	input := `
let api = import @std/api
let r1 = try fn() { api.notFound("a") }()
r1.error.status
`
	result := evalUnifiedError(t, input)

	num, ok := result.(*evaluator.Integer)
	if !ok {
		if errObj, ok := result.(*evaluator.Error); ok {
			t.Fatalf("got error: %s", errObj.Inspect())
		}
		t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
	}
	if num.Value != 404 {
		t.Errorf("expected 404, got %d", num.Value)
	}
}

func TestUnifiedMultipleTryCatchesSecond(t *testing.T) {
	input := `
let api = import @std/api
let r2 = try fn() { api.badRequest("b") }()
r2.error.status
`
	result := evalUnifiedError(t, input)

	num, ok := result.(*evaluator.Integer)
	if !ok {
		if errObj, ok := result.(*evaluator.Error); ok {
			t.Fatalf("got error: %s", errObj.Inspect())
		}
		t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
	}
	if num.Value != 400 {
		t.Errorf("expected 400, got %d", num.Value)
	}
}

// =============================================================================
// Ensure special typed dicts (path, url, datetime) are NOT affected by
// message-key string coercion
// =============================================================================

func TestUnifiedStringCoercionDoesNotAffectPath(t *testing.T) {
	// path() creates a dict with __type: "path" — should use path coercion, not message
	input := `"" + path("./test.txt")`
	result := evalUnifiedError(t, input)

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	// Should contain "test.txt", using path coercion not message coercion
	if !strings.Contains(str.Value, "test.txt") {
		t.Errorf("expected path string containing 'test.txt', got %q", str.Value)
	}
}
