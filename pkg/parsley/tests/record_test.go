package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
)

// evalRecordTest helper that evaluates Parsley code using the full evaluator
func evalRecordTest(t *testing.T, input string) evaluator.Object {
	t.Helper()
	result, err := parsley.Eval(input)
	if err != nil {
		t.Fatalf("evaluation error: %v", err)
	}
	if result == nil || result.Value == nil {
		t.Fatal("result is nil")
	}
	return result.Value
}

// =============================================================================
// Record Creation Tests
// =============================================================================

func TestRecordCreation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "create record from schema with data",
			input: `
@schema User {
    name: string
    age: int
}
let record = User({name: "Alice", age: 30})
record.name`,
			expected: "Alice",
		},
		{
			name: "create record with integer field",
			input: `
@schema AgeOnly {
    age: int
}
let record = AgeOnly({age: 25})
record.age`,
			expected: "25",
		},
		{
			name: "create record with missing optional field",
			input: `
@schema Person {
    name: string
    bio: string
}
let record = Person({name: "Bob"})
record.name`,
			expected: "Bob",
		},
		{
			name: "record type is RECORD",
			input: `
@schema NameOnly {
    name: string
}
let record = NameOnly({name: "Test"})
record.type()`,
			expected: "record",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// =============================================================================
// Validation Tests - Required Fields
// =============================================================================

func TestRecordValidationRequired(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectValid bool
	}{
		{
			name: "valid with required field present",
			input: `
@schema ReqName1 {
    name: string
}
let record = ReqName1({name: "Alice"}).validate()
record.isValid()`,
			expectValid: true,
		},
		{
			name: "invalid with required field missing",
			input: `
@schema ReqName2 {
    name: string
}
let record = ReqName2({}).validate()
record.isValid()`,
			expectValid: false,
		},
		{
			name: "empty string passes required check",
			input: `
@schema ReqName3 {
    name: string
}
let record = ReqName3({name: ""}).validate()
record.isValid()`,
			expectValid: true, // Empty string "" passes required check
		},
		{
			name: "optional field can be missing",
			input: `
@schema OptName {
    name: string?
}
let record = OptName({}).validate()
record.isValid()`,
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			boolVal, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolVal.Value != tt.expectValid {
				t.Errorf("expected isValid()=%v, got %v", tt.expectValid, boolVal.Value)
			}
		})
	}
}

// =============================================================================
// Validation Tests - Type Checking
// =============================================================================

func TestRecordValidationType(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectValid bool
	}{
		{
			name: "valid integer type",
			input: `
@schema IntAge {
    age: int
}
let record = IntAge({age: 25}).validate()
record.isValid()`,
			expectValid: true,
		},
		{
			name: "integer accepts string that looks like integer",
			input: `
@schema IntAge2 {
    age: int
}
let record = IntAge2({age: "25"}).validate()
record.isValid()`,
			expectValid: true,
		},
		{
			name: "invalid integer type",
			input: `
@schema IntAge3 {
    age: int
}
let record = IntAge3({age: "not a number"}).validate()
record.isValid()`,
			expectValid: false,
		},
		{
			name: "valid float type",
			input: `
@schema FloatPrice {
    price: float
}
let record = FloatPrice({price: 19.99}).validate()
record.isValid()`,
			expectValid: true,
		},
		{
			name: "valid boolean type",
			input: `
@schema BoolActive {
    active: bool
}
let record = BoolActive({active: true}).validate()
record.isValid()`,
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			boolVal, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolVal.Value != tt.expectValid {
				t.Errorf("expected isValid()=%v, got %v", tt.expectValid, boolVal.Value)
			}
		})
	}
}

// =============================================================================
// Validation Tests - Format Constraints
// =============================================================================

func TestRecordValidationFormat(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectValid bool
	}{
		{
			name: "valid email format",
			input: `
@schema EmailField {
    email: email
}
let record = EmailField({email: "test@example.com"}).validate()
record.isValid()`,
			expectValid: true,
		},
		{
			name: "invalid email format",
			input: `
@schema EmailField2 {
    email: email
}
let record = EmailField2({email: "not-an-email"}).validate()
record.isValid()`,
			expectValid: false,
		},
		{
			name: "valid url format",
			input: `
@schema UrlField {
    website: url
}
let record = UrlField({website: "https://example.com"}).validate()
record.isValid()`,
			expectValid: true,
		},
		{
			name: "valid slug format",
			input: `
@schema SlugField {
    slug: slug
}
let record = SlugField({slug: "my-article-title"}).validate()
record.isValid()`,
			expectValid: true,
		},
		{
			name: "invalid slug format",
			input: `
@schema SlugField2 {
    slug: slug
}
let record = SlugField2({slug: "Has Spaces"}).validate()
record.isValid()`,
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			boolVal, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolVal.Value != tt.expectValid {
				t.Errorf("expected isValid()=%v, got %v", tt.expectValid, boolVal.Value)
			}
		})
	}
}

// =============================================================================
// Validation Tests - Length and Value Constraints
// =============================================================================

func TestRecordValidationConstraints(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectValid bool
	}{
		{
			name: "valid min string length",
			input: `
@schema MinLen1 {
    name: string(min: 3)
}
let record = MinLen1({name: "Alice"}).validate()
record.isValid()`,
			expectValid: true,
		},
		{
			name: "invalid min string length",
			input: `
@schema MinLen2 {
    name: string(min: 3)
}
let record = MinLen2({name: "Al"}).validate()
record.isValid()`,
			expectValid: false,
		},
		{
			name: "valid max string length",
			input: `
@schema MaxLen1 {
    name: string(max: 10)
}
let record = MaxLen1({name: "Alice"}).validate()
record.isValid()`,
			expectValid: true,
		},
		{
			name: "invalid max string length",
			input: `
@schema MaxLen2 {
    name: string(max: 3)
}
let record = MaxLen2({name: "Alice"}).validate()
record.isValid()`,
			expectValid: false,
		},
		{
			name: "valid min numeric value",
			input: `
@schema MinVal1 {
    age: int(min: 18)
}
let record = MinVal1({age: 25}).validate()
record.isValid()`,
			expectValid: true,
		},
		{
			name: "invalid min numeric value",
			input: `
@schema MinVal2 {
    age: int(min: 18)
}
let record = MinVal2({age: 15}).validate()
record.isValid()`,
			expectValid: false,
		},
		{
			name: "valid max numeric value",
			input: `
@schema MaxVal1 {
    age: int(max: 100)
}
let record = MaxVal1({age: 50}).validate()
record.isValid()`,
			expectValid: true,
		},
		{
			name: "invalid max numeric value",
			input: `
@schema MaxVal2 {
    age: int(max: 100)
}
let record = MaxVal2({age: 150}).validate()
record.isValid()`,
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			boolVal, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolVal.Value != tt.expectValid {
				t.Errorf("expected isValid()=%v, got %v", tt.expectValid, boolVal.Value)
			}
		})
	}
}

// =============================================================================
// Validation Tests - Enum
// =============================================================================

func TestRecordValidationEnum(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectValid bool
	}{
		{
			name: "valid enum value",
			input: `
@schema EnumStatus {
    status: enum("draft", "published", "archived")
}
let record = EnumStatus({status: "published"}).validate()
record.isValid()`,
			expectValid: true,
		},
		{
			name: "invalid enum value",
			input: `
@schema EnumStatus2 {
    status: enum("draft", "published", "archived")
}
let record = EnumStatus2({status: "deleted"}).validate()
record.isValid()`,
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			boolVal, ok := result.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if boolVal.Value != tt.expectValid {
				t.Errorf("expected isValid()=%v, got %v", tt.expectValid, boolVal.Value)
			}
		})
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestRecordErrorMethods(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "error returns message for field",
			input: `
@schema ErrName1 {
    name: string
}
let record = ErrName1({}).validate()
record.error("name")`,
			expected: "Name is required",
		},
		{
			name: "error returns empty for valid field",
			input: `
@schema ErrName2 {
    name: string
}
let record = ErrName2({name: "Alice"}).validate()
record.error("name")`,
			expected: "null", // Returns null when no error
		},
		{
			name: "errorCode returns code for field",
			input: `
@schema ErrCode1 {
    name: string
}
let record = ErrCode1({}).validate()
record.errorCode("name")`,
			expected: "REQUIRED",
		},
		{
			name: "hasError returns true for invalid field",
			input: `
@schema HasErr1 {
    name: string
}
let record = HasErr1({}).validate()
record.hasError("name")`,
			expected: "true",
		},
		{
			name: "hasError returns false for valid field",
			input: `
@schema HasErr2 {
    name: string
}
let record = HasErr2({name: "Alice"}).validate()
record.hasError("name")`,
			expected: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestRecordErrorsDictionary(t *testing.T) {
	input := `
@schema MultiErr1 {
    name: string
    email: email
}
let record = MultiErr1({name: "", email: "invalid"}).validate()
let errs = record.errors()
errs.type()`

	result := evalRecordTest(t, input)
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}
	if result.Inspect() != "dictionary" {
		t.Errorf("expected errors() to return dictionary, got %s", result.Inspect())
	}
}

func TestRecordErrorList(t *testing.T) {
	input := `
@schema ListErr1 {
    name: string
    email: email
}
let record = ListErr1({name: "", email: "invalid"}).validate()
let errs = record.errorList()
errs.type()`

	result := evalRecordTest(t, input)
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}
	if result.Inspect() != "array" {
		t.Errorf("expected errorList() to return array, got %s", result.Inspect())
	}
}

// =============================================================================
// Update Tests
// =============================================================================

func TestRecordUpdate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "update single field",
			input: `
@schema UpdPerson1 {
    name: string
    age: int
}
let record = UpdPerson1({name: "Alice", age: 30})
let updated = record.update({name: "Bob"})
updated.name`,
			expected: "Bob",
		},
		{
			name: "update preserves other fields",
			input: `
@schema UpdPerson2 {
    name: string
    age: int
}
let record = UpdPerson2({name: "Alice", age: 30})
let updated = record.update({name: "Bob"})
updated.age`,
			expected: "30",
		},
		{
			name: "update is immutable",
			input: `
@schema UpdPerson3 {
    name: string
}
let record = UpdPerson3({name: "Alice"})
let _ = record.update({name: "Bob"})
record.name`,
			expected: "Alice",
		},
		{
			name: "update clears validation state",
			input: `
@schema UpdPerson4 {
    name: string
}
let record = UpdPerson4({name: "Alice"}).validate()
let updated = record.update({name: ""})
updated.isValid()`,
			expected: "true", // Not validated yet
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// =============================================================================
// Schema Metadata Tests
// =============================================================================

func TestRecordSchemaMetadata(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "title returns auto-generated title from camelCase",
			input: `
@schema MetaTitle1 {
    firstName: string
}
let record = MetaTitle1({firstName: "Alice"})
record.title("firstName")`,
			expected: "First name", // camelCase to Title case
		},
		{
			name: "title returns simple field name capitalized",
			input: `
@schema MetaTitle2 {
    name: string
}
let record = MetaTitle2({name: "Alice"})
record.title("name")`,
			expected: "Name",
		},
		{
			name: "enumValues returns enum options",
			input: `
@schema MetaEnum {
    status: enum("draft", "published")
}
let record = MetaEnum({status: "draft"})
let values = record.enumValues("status")
values[0]`,
			expected: "draft",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// =============================================================================
// Dictionary Compatibility Tests
// =============================================================================

func TestRecordDataMethod(t *testing.T) {
	input := `
@schema DataPerson {
    name: string
    age: int
}
let record = DataPerson({name: "Alice", age: 30})
let data = record.data()
data.type()`

	result := evalRecordTest(t, input)
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}
	if result.Inspect() != "dictionary" {
		t.Errorf("expected data() to return dictionary, got %s", result.Inspect())
	}
}

func TestRecordKeysMethod(t *testing.T) {
	input := `
@schema KeysPerson {
    name: string
    age: int
}
let record = KeysPerson({name: "Alice", age: 30})
let keys = record.keys()
keys.type()`

	result := evalRecordTest(t, input)
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}
	if result.Inspect() != "array" {
		t.Errorf("expected keys() to return array, got %s", result.Inspect())
	}
}

func TestRecordSpread(t *testing.T) {
	// Record spread into dictionary using record.data()
	// The {...record} syntax isn't directly supported; use data() to convert
	input := `
@schema SpreadPerson {
    name: string
}
let record = SpreadPerson({name: "Alice"})
let data = record.data()
data.name`

	result := evalRecordTest(t, input)
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}
	if result.Inspect() != "Alice" {
		t.Errorf("expected 'Alice', got %s", result.Inspect())
	}
}

func TestRecordInTagSpread(t *testing.T) {
	// Record in tag spread uses data() method for now
	input := `
@schema TagAttrs {
    class: string
    id: string
}
let record = TagAttrs({class: "btn", id: "submit"})
let attrs = record.data()
<div class={attrs.class} id={attrs.id}/>`

	result := evalRecordTest(t, input)
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}
	// Check that the HTML contains the attributes
	html := result.Inspect()
	if !strings.Contains(html, `class="btn"`) || !strings.Contains(html, `id="submit"`) {
		t.Errorf("expected HTML with attributes, got %s", html)
	}
}

// =============================================================================
// Schema Method Tests
// =============================================================================

func TestRecordSchemaMethod(t *testing.T) {
	input := `
@schema SchemaPerson {
    name: string
}
let record = SchemaPerson({name: "Alice"})
let s = record.schema()
s.type()`

	result := evalRecordTest(t, input)
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}
	// DSLSchema type string is "schema"
	if result.Inspect() != "schema" {
		t.Errorf("expected schema() to return schema, got %s", result.Inspect())
	}
}

// =============================================================================
// WithError Tests
// =============================================================================

func TestRecordWithError(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "withError adds custom error",
			input: `
@schema WithErrEmail {
    email: string
}
let record = WithErrEmail({email: "test@example.com"})
let withErr = record.withError("email", "Email already exists")
withErr.error("email")`,
			expected: "Email already exists",
		},
		{
			name: "withError sets CUSTOM code",
			input: `
@schema WithErrEmail2 {
    email: string
}
let record = WithErrEmail2({email: "test@example.com"})
let withErr = record.withError("email", "Email already exists")
withErr.errorCode("email")`,
			expected: "CUSTOM",
		},
		{
			name: "withError is immutable",
			input: `
@schema WithErrEmail3 {
    email: string
}
let record = WithErrEmail3({email: "test@example.com"})
let _ = record.withError("email", "Error")
record.hasError("email")`,
			expected: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// =============================================================================
// Property Access Tests
// =============================================================================

func TestRecordPropertyAccess(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "access string property",
			input: `
@schema PropPerson {
    name: string
}
let record = PropPerson({name: "Alice"})
record.name`,
			expected: "Alice",
		},
		{
			name: "access integer property",
			input: `
@schema PropPerson2 {
    age: int
}
let record = PropPerson2({age: 30})
record.age`,
			expected: "30",
		},
		{
			name: "access missing property returns null",
			input: `
@schema PropPerson3 {
    name: string
    bio: string
}
let record = PropPerson3({name: "Alice"})
record.bio == null`,
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}
