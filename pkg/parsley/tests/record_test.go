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
		{
			name: "valid UUID format",
			input: `
@schema UUIDField {
    id: uuid
}
let record = UUIDField({id: "123e4567-e89b-12d3-a456-426614174000"}).validate()
record.isValid()`,
			expectValid: true,
		},
		{
			name: "invalid UUID format - wrong length",
			input: `
@schema UUIDField2 {
    id: uuid
}
let record = UUIDField2({id: "123e4567-e89b-12d3"}).validate()
record.isValid()`,
			expectValid: false,
		},
		{
			name: "invalid UUID format - missing dashes",
			input: `
@schema UUIDField3 {
    id: uuid
}
let record = UUIDField3({id: "123e4567e89b12d3a456426614174000"}).validate()
record.isValid()`,
			expectValid: false,
		},
		{
			name: "valid ULID format",
			input: `
@schema ULIDField {
    id: ulid
}
let record = ULIDField({id: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}).validate()
record.isValid()`,
			expectValid: true,
		},
		{
			name: "invalid ULID format - wrong length",
			input: `
@schema ULIDField2 {
    id: ulid
}
let record = ULIDField2({id: "01ARZ3NDEKTSV4RR"}).validate()
record.isValid()`,
			expectValid: false,
		},
		{
			name: "invalid ULID format - invalid characters",
			input: `
@schema ULIDField3 {
    id: ulid
}
let record = ULIDField3({id: "01ARZ3NDEKTSV4RRFFQ69G5FAI"}).validate()
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
			expected: "First Name", // camelCase to Title Case (both words capitalized)
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

// TEST-DICT-002: JSON encoding encodes data only (SPEC-DC-003)
func TestRecordJSONEncoding(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name: "toJSON encodes Record data fields",
			input: `
@schema JSONPerson {
    name: string
    age: int
}
let record = JSONPerson({name: "Alice", age: 30})
record.toJSON()`,
			contains: []string{`"name"`, `"Alice"`, `"age"`, `30`},
			excludes: []string{`schema`, `validated`, `errors`},
		},
		{
			name: "validated Record JSON still only contains data",
			input: `
@schema JSONUser {
    name: string
    email: email
}
let record = JSONUser({name: "Bob", email: "bob@example.com"}).validate()
record.toJSON()`,
			contains: []string{`"name"`, `"Bob"`, `"email"`, `"bob@example.com"`},
			excludes: []string{`validated`, `errors`, `schema`},
		},
		{
			name: "Record with validation errors JSON only contains data",
			input: `
@schema JSONInvalid {
    name: string(min: 10)
}
let record = JSONInvalid({name: "AB"}).validate()
record.toJSON()`,
			contains: []string{`"name"`, `"AB"`},
			excludes: []string{`errors`, `MIN_LENGTH`, `validated`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}

			jsonStr := result.Inspect()

			for _, s := range tt.contains {
				if !strings.Contains(jsonStr, s) {
					t.Errorf("expected JSON to contain %q, got: %s", s, jsonStr)
				}
			}

			for _, s := range tt.excludes {
				if strings.Contains(jsonStr, s) {
					t.Errorf("expected JSON NOT to contain %q (metadata), got: %s", s, jsonStr)
				}
			}
		})
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
		{
			name: "withError with custom code (3-arg)",
			input: `
@schema WithErrEmail4 {
    email: string
}
let record = WithErrEmail4({email: "test@example.com"})
let withErr = record.withError("email", "DUPLICATE", "Email already exists")
withErr.errorCode("email")`,
			expected: "DUPLICATE",
		},
		{
			name: "withError with custom code preserves message",
			input: `
@schema WithErrEmail5 {
    email: string
}
let record = WithErrEmail5({email: "test@example.com"})
let withErr = record.withError("email", "DB_CONSTRAINT", "Unique constraint violated")
withErr.error("email")`,
			expected: "Unique constraint violated",
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

// =============================================================================
// Schema Metadata Tests (Phase 2 - FEAT-091)
// =============================================================================

func TestSchemaMetadataPipeSyntax(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "schema field with title metadata",
			input: `
@schema Person {
    name: string | {title: "Full Name"}
}
let record = Person({name: "Alice"})
record.schema().title("name")`,
			expected: "Full Name",
		},
		{
			name: "schema field with placeholder metadata",
			input: `
@schema LoginForm {
    email: email | {placeholder: "Enter your email"}
}
let record = LoginForm({email: "test@test.com"})
record.schema().placeholder("email")`,
			expected: "Enter your email",
		},
		{
			name: "schema field with multiple metadata",
			input: `
@schema Contact {
    phone: phone | {title: "Phone Number", placeholder: "555-1234", help: "Include area code"}
}
let record = Contact({phone: "555-1234"})
record.schema().meta("phone", "help")`,
			expected: "Include area code",
		},
		{
			name: "title fallback to title case",
			input: `
@schema Item {
    first_name: string
}
let record = Item({first_name: "test"})
record.schema().title("first_name")`,
			expected: "First Name",
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

func TestSchemaFieldsMethods(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "schema.fields returns all field names",
			input: `
@schema User {
    name: string
    age: int
    email: email
}
let record = User({name: "test", age: 1, email: "a@b.com"})
record.schema().fields().length()`,
			expected: "3",
		},
		{
			name: "schema.visibleFields excludes hidden",
			input: `
@schema Profile {
    name: string
    id: int | {hidden: true}
}
let record = Profile({name: "test", id: 1})
record.schema().visibleFields().length()`,
			expected: "1",
		},
		{
			name: "schema.enumValues returns options",
			input: `
@schema Status {
    status: enum("active", "inactive", "pending")
}
let record = Status({status: "active"})
record.schema().enumValues("status").length()`,
			expected: "3",
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

func TestRecordMetadataMethods(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "record.title returns field title",
			input: `
@schema UserForm {
    full_name: string | {title: "Your Name"}
}
let record = UserForm({full_name: "Alice"})
record.title("full_name")`,
			expected: "Your Name",
		},
		{
			name: "record.placeholder returns field placeholder",
			input: `
@schema EmailForm {
    email: email | {placeholder: "you@example.com"}
}
let record = EmailForm({email: "test@test.com"})
record.placeholder("email")`,
			expected: "you@example.com",
		},
		{
			name: "record.meta returns custom metadata",
			input: `
@schema HelpForm {
    field: string | {help: "This is helpful"}
}
let record = HelpForm({field: "value"})
record.meta("field", "help")`,
			expected: "This is helpful",
		},
		{
			name: "record.title fallback to title case",
			input: `
@schema FallbackForm {
    user_name: string
}
let record = FallbackForm({user_name: "test"})
record.title("user_name")`,
			expected: "User Name",
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

func TestRecordFormatMethod(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "format currency",
			input: `
@schema Invoice {
    amount: int | {format: "currency"}
}
let record = Invoice({amount: 1234})
record.format("amount")`,
			expected: "$ 1,234.00",
		},
		{
			name: "format percent",
			input: `
@schema Stats {
    rate: float | {format: "percent"}
}
let record = Stats({rate: 0.15})
record.format("rate")`,
			expected: "15%",
		},
		{
			name: "format number with thousands",
			input: `
@schema BigNumbers {
    count: int | {format: "number"}
}
let record = BigNumbers({count: 1234567})
record.format("count")`,
			expected: "1,234,567",
		},
		{
			name: "format with no hint returns string",
			input: `
@schema Plain {
    value: int
}
let record = Plain({value: 42})
record.format("value")`,
			expected: "42",
		},
		{
			name: "format date from string",
			input: `
@schema Dates {
    created: string | {format: "date"}
}
let record = Dates({created: "2025-01-15"})
record.format("created")`,
			expected: "Jan 15, 2025",
		},
		{
			name: "format datetime from datetime literal",
			input: `
@schema Events {
    when: datetime | {format: "datetime"}
}
let record = Events({when: @2025-01-15T14:30:00})
record.format("when")`,
			expected: "Jan 15, 2025 2:30 PM",
		},
		{
			name: "format with unknown hint returns string",
			input: `
@schema Custom {
    value: int | {format: "unknown_format"}
}
let record = Custom({value: 42})
record.format("value")`,
			expected: "42",
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

// TestRecordEnumValuesMethods tests record.enumValues() shorthand
func TestRecordEnumValuesMethods(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "record.enumValues returns enum options",
			input: `
@schema Role {
    role: enum("admin", "user", "guest")
}
let record = Role({role: "admin"})
record.enumValues("role").length()`,
			expected: "3",
		},
		{
			name: "record.enumValues returns empty for non-enum",
			input: `
@schema Plain {
    name: string
}
let record = Plain({name: "test"})
record.enumValues("name").length()`,
			expected: "0",
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

// TestSchemaMethodsEdgeCases tests edge cases for schema methods
func TestSchemaMethodsEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "schema.title on non-existent field returns titlecase",
			input: `
@schema Empty {
    name: string
}
let record = Empty({name: "test"})
record.schema().title("non_existent_field")`,
			expected: "Non Existent Field",
		},
		{
			name: "schema.placeholder on non-existent field returns null",
			input: `
@schema Empty {
    name: string
}
let record = Empty({name: "test"})
record.schema().placeholder("non_existent") == null`,
			expected: "true",
		},
		{
			name: "schema.meta on non-existent field returns null",
			input: `
@schema Empty {
    name: string
}
let record = Empty({name: "test"})
record.schema().meta("non_existent", "title") == null`,
			expected: "true",
		},
		{
			name: "record.format on non-existent field returns null",
			input: `
@schema Empty {
    name: string
}
let record = Empty({name: "test"})
record.format("non_existent") == null`,
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

// =============================================================================
// Phase 3: Table Integration Tests
// =============================================================================

// TestSchemaArrayCreatesTable tests P3-001: Schema([...]) creates Table
func TestSchemaArrayCreatesTable(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "schema with array creates table",
			input: `
@schema User {
    name: string
    age: int
}
let users = User([
    {name: "Alice", age: 30},
    {name: "Bob", age: 25}
])
users.length`,
			expected: "2",
		},
		{
			name: "typed table has schema attached",
			input: `
@schema User {
    name: string
    age: int
}
let users = User([{name: "Alice", age: 30}])
users.schema != null`,
			expected: "true",
		},
		{
			name: "typed table schema name matches",
			input: `
@schema User {
    name: string
    age: int
}
let users = User([{name: "Alice", age: 30}])
users.schema.name`,
			expected: "User",
		},
		{
			name: "empty array creates empty table",
			input: `
@schema User {
    name: string
}
let users = User([])
users.length`,
			expected: "0",
		},
		{
			name: "typed table filters unknown fields",
			input: `
@schema User {
    name: string
}
let users = User([{name: "Alice", extra: "ignored"}])
let row = users[0]
row.name`,
			expected: "Alice",
		},
		{
			name: "typed table applies defaults",
			input: `
@schema User {
    name: string
    role: string = "member"
}
let users = User([{name: "Alice"}])
users[0].role`,
			expected: "member",
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

// TestTypedTableLiteral tests P3-002: @table(Schema) [...] literal
func TestTypedTableLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "table literal with schema",
			input: `
@schema Product {
    sku: string
    price: int
}
let products = @table(Product) [
    {sku: "A001", price: 100},
    {sku: "A002", price: 200}
]
products.length`,
			expected: "2",
		},
		{
			name: "table literal with schema has schema attached",
			input: `
@schema Product {
    sku: string
}
let products = @table(Product) [{sku: "A001"}]
products.schema.name`,
			expected: "Product",
		},
		{
			name: "table literal applies defaults",
			input: `
@schema Product {
    sku: string
    stock: int = 0
}
let products = @table(Product) [{sku: "A001"}]
products[0].stock`,
			expected: "0",
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

// TestDictAsSchemaCreatesRecord tests P3-003: {...}.as(Schema) creates Record
func TestDictAsSchemaCreatesRecord(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "dict.as(Schema) creates record",
			input: `
@schema User {
    name: string
    age: int
}
let data = {name: "Alice", age: 30}
let record = data.as(User)
record.name`,
			expected: "Alice",
		},
		{
			name: "dict.as(Schema) type is record",
			input: `
@schema User {
    name: string
}
let data = {name: "Alice"}
let record = data.as(User)
record.type()`,
			expected: "record",
		},
		{
			name: "dict.as(Schema) applies defaults",
			input: `
@schema User {
    name: string
    role: string = "guest"
}
let data = {name: "Alice"}
let record = data.as(User)
record.role`,
			expected: "guest",
		},
		{
			name: "dict.as(Schema) filters unknown fields",
			input: `
@schema User {
    name: string
}
let data = {name: "Alice", extra: "ignored"}
let record = data.as(User)
record.extra`,
			expected: "null",
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

// TestTableAsSchemaCreatesTypedTable tests P3-004: table(data).as(Schema) creates Table
func TestTableAsSchemaCreatesTypedTable(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "table.as(Schema) creates typed table",
			input: `
@schema User {
    name: string
    age: int
}
let data = @table [{name: "Alice", age: 30}, {name: "Bob", age: 25}]
let users = data.as(User)
users.schema.name`,
			expected: "User",
		},
		{
			name: "table.as(Schema) preserves row count",
			input: `
@schema User {
    name: string
}
let data = @table [{name: "Alice"}, {name: "Bob"}]
let users = data.as(User)
users.length`,
			expected: "2",
		},
		{
			name: "table.as(Schema) applies defaults",
			input: `
@schema User {
    name: string
    role: string = "member"
}
let data = @table [{name: "Alice"}]
let users = data.as(User)
users[0].role`,
			expected: "member",
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

// TestTableValidation tests P3-005, P3-006: table.validate() and table.isValid()
func TestTableValidation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "validate all rows",
			input: `
@schema User {
    name: string
    age: int
}
let users = User([
    {name: "Alice", age: 30},
    {age: 25}
])
let validated = users.validate()
validated.isValid()`,
			expected: "false",
		},
		{
			name: "all valid rows returns true",
			input: `
@schema User {
    name: string
}
let users = User([{name: "Alice"}, {name: "Bob"}])
let validated = users.validate()
validated.isValid()`,
			expected: "true",
		},
		{
			name: "unvalidated table with invalid data",
			input: `
@schema User {
    name: string
}
let users = User([{}])
users.isValid()`,
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

// TestTableErrors tests P3-007: table.errors() with row indices
func TestTableErrors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "errors returns array with row indices",
			input: `
@schema User {
    name: string
}
let users = User([
    {name: "Alice"},
    {},
    {name: "Carol"}
])
let validated = users.validate()
let errs = validated.errors()
errs.length()`,
			expected: "1",
		},
		{
			name: "error row index is zero-based",
			input: `
@schema User {
    name: string
}
let users = User([{name: "Alice"}, {}])
let validated = users.validate()
let errs = validated.errors()
errs[0].row`,
			expected: "1",
		},
		{
			name: "error contains field name",
			input: `
@schema User {
    name: string
}
let users = User([{}])
let validated = users.validate()
let errs = validated.errors()
errs[0].field`,
			expected: "name",
		},
		{
			name: "error contains code",
			input: `
@schema User {
    name: string
}
let users = User([{}])
let validated = users.validate()
let errs = validated.errors()
errs[0].code`,
			expected: "REQUIRED",
		},
		{
			name: "no errors returns empty array",
			input: `
@schema User {
    name: string
}
let users = User([{name: "Alice"}])
let validated = users.validate()
validated.errors().length()`,
			expected: "0",
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

// TestTableValidInvalidRows tests P3-008, P3-009: table.validRows() and table.invalidRows()
func TestTableValidInvalidRows(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "validRows returns only valid rows",
			input: `
@schema User {
    name: string
}
let users = User([{name: "Alice"}, {}, {name: "Carol"}])
let validated = users.validate()
validated.validRows().length`,
			expected: "2",
		},
		{
			name: "invalidRows returns only invalid rows",
			input: `
@schema User {
    name: string
}
let users = User([{name: "Alice"}, {}, {}])
let validated = users.validate()
validated.invalidRows().length`,
			expected: "2",
		},
		{
			name: "validRows on all-valid table",
			input: `
@schema User {
    name: string
}
let users = User([{name: "Alice"}, {name: "Bob"}])
let validated = users.validate()
validated.validRows().length`,
			expected: "2",
		},
		{
			name: "invalidRows on all-valid table",
			input: `
@schema User {
    name: string
}
let users = User([{name: "Alice"}])
let validated = users.validate()
validated.invalidRows().length`,
			expected: "0",
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

// TestTableSchemaMethod tests P3-010: table.schema()
func TestTableSchemaMethod(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "typed table has schema property",
			input: `
@schema User {
    name: string
}
let users = User([{name: "Alice"}])
users.schema.name`,
			expected: "User",
		},
		{
			name: "untyped table schema is null",
			input: `
let users = @table [{name: "Alice"}]
users.schema == null`,
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

// TestTableRowReturnsRecord tests P3-011: table[n] returns Record
func TestTableRowReturnsRecord(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "typed table row is a record",
			input: `
@schema User {
    name: string
}
let users = User([{name: "Alice"}])
let row = users[0]
row.type()`,
			expected: "record",
		},
		{
			name: "typed table row has schema",
			input: `
@schema User {
    name: string
}
let users = User([{name: "Alice"}])
let row = users[0]
row.schema().name`,
			expected: "User",
		},
		{
			name: "typed table row data access",
			input: `
@schema User {
    name: string
    age: int
}
let users = User([{name: "Alice", age: 30}])
users[0].name`,
			expected: "Alice",
		},
		{
			name: "negative indexing works for typed tables",
			input: `
@schema User {
    name: string
}
let users = User([{name: "Alice"}, {name: "Bob"}])
users[-1].name`,
			expected: "Bob",
		},
		{
			name: "validated row has errors accessible",
			input: `
@schema User {
    name: string
}
let users = User([{}])
let validated = users.validate()
let row = validated[0]
row.isValid()`,
			expected: "false",
		},
		{
			name: "untyped table row is dictionary",
			input: `
let users = @table [{name: "Alice"}]
let row = users[0]
row.type()`,
			expected: "dictionary",
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
// BUG-015: Schema column order preservation
// =============================================================================

func TestSchemaFieldOrderPreserved(t *testing.T) {
	// BUG-015: Applying a schema via .as() was sorting columns alphabetically
	// instead of preserving the schema's declaration order
	input := `
@schema Person {
    id: id
    Name: string
    Firstname: string
    Surname: string
    Birthday: date
    Age: int
}

let data = @table [
    {id: 1, Name: "Solly Phillips", Firstname: "Solly", Surname: "Phillips", Birthday: "2005-04-22", Age: 20}
]

let typed = data.as(Person)
typed.columns
`
	result := evalRecordTest(t, input)
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array, got %T: %s", result, result.Inspect())
	}

	// Verify column order matches schema declaration order
	expectedOrder := []string{"id", "Name", "Firstname", "Surname", "Birthday", "Age"}
	if len(arr.Elements) != len(expectedOrder) {
		t.Fatalf("expected %d columns, got %d", len(expectedOrder), len(arr.Elements))
	}

	for i, expected := range expectedOrder {
		actual := arr.Elements[i].Inspect()
		if actual != expected {
			t.Errorf("column %d: expected %q, got %q", i, expected, actual)
		}
	}
}

// TestSchemaRowFieldOrderPreserved verifies that accessing .rows preserves schema field order
func TestSchemaRowFieldOrderPreserved(t *testing.T) {
	// BUG-015 follow-up: The table.rows property was returning dictionaries
	// with alphabetical key order instead of schema declaration order
	input := `
@schema Person {
    id: id
    Name: string
    Firstname: string
    Surname: string
    Birthday: date
    Age: int
}

let data = @table [
    {id: 1, Name: "Solly Phillips", Firstname: "Solly", Surname: "Phillips", Birthday: "2005-04-22", Age: 20}
]

let typed = data.as(Person)
let rows = typed.rows
let row = rows[0]
row.keys()
`
	result := evalRecordTest(t, input)
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array, got %T: %s", result, result.Inspect())
	}

	// Verify key order matches schema declaration order
	expectedOrder := []string{"id", "Name", "Firstname", "Surname", "Birthday", "Age"}
	if len(arr.Elements) != len(expectedOrder) {
		t.Fatalf("expected %d keys, got %d: %s", len(expectedOrder), len(arr.Elements), result.Inspect())
	}

	for i, expected := range expectedOrder {
		actual := arr.Elements[i].Inspect()
		if actual != expected {
			t.Errorf("key %d: expected %q, got %q", i, expected, actual)
		}
	}
}