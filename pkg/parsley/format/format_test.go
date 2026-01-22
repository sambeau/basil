package format

import (
	"testing"
)

// Mock types for testing

type mockObject struct {
	typ     string
	inspect string
}

func (m *mockObject) Type() string    { return m.typ }
func (m *mockObject) Inspect() string { return m.inspect }

type mockArray struct {
	mockObject
	elements []TypedObject
}

func (m *mockArray) GetElements() []TypedObject { return m.elements }

type mockDict struct {
	mockObject
	keys   []string
	values map[string]TypedObject
}

func (m *mockDict) GetKeys() []string                     { return m.keys }
func (m *mockDict) GetValueObject(key string) TypedObject { return m.values[key] }

type mockFunction struct {
	mockObject
	params []string
	body   string
}

func (m *mockFunction) GetParamStrings() []string { return m.params }
func (m *mockFunction) GetBodyString() string     { return m.body }

type mockRecord struct {
	mockObject
	schemaName string
	keys       []string
	fields     map[string]TypedObject
}

func (m *mockRecord) GetSchemaName() string                 { return m.schemaName }
func (m *mockRecord) GetFieldKeys() []string                { return m.keys }
func (m *mockRecord) GetFieldObject(key string) TypedObject { return m.fields[key] }

// Tests

func TestFormatInteger(t *testing.T) {
	obj := &mockObject{typ: "INTEGER", inspect: "42"}
	result := formatTypedObject(obj)
	if result != "42" {
		t.Errorf("Expected '42', got '%s'", result)
	}
}

func TestFormatBoolean(t *testing.T) {
	obj := &mockObject{typ: "BOOLEAN", inspect: "true"}
	result := formatTypedObject(obj)
	if result != "true" {
		t.Errorf("Expected 'true', got '%s'", result)
	}
}

func TestFormatString(t *testing.T) {
	obj := &mockObject{typ: "STRING", inspect: "hello"}
	result := formatTypedObject(obj)
	if result != `"hello"` {
		t.Errorf("Expected '\"hello\"', got '%s'", result)
	}
}

func TestFormatStringWithQuotes(t *testing.T) {
	obj := &mockObject{typ: "STRING", inspect: `hello "world"`}
	result := formatTypedObject(obj)
	expected := `"hello \"world\""`
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatNull(t *testing.T) {
	obj := &mockObject{typ: "NULL", inspect: "null"}
	result := formatTypedObject(obj)
	if result != "null" {
		t.Errorf("Expected 'null', got '%s'", result)
	}
}

func TestFormatArrayInline(t *testing.T) {
	elements := []TypedObject{
		&mockObject{typ: "INTEGER", inspect: "1"},
		&mockObject{typ: "INTEGER", inspect: "2"},
		&mockObject{typ: "INTEGER", inspect: "3"},
	}
	arr := &mockArray{
		mockObject: mockObject{typ: "ARRAY", inspect: "[1, 2, 3]"},
		elements:   elements,
	}
	result := formatTypedObject(arr)
	expected := "[1, 2, 3]"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatArrayEmpty(t *testing.T) {
	arr := &mockArray{
		mockObject: mockObject{typ: "ARRAY", inspect: "[]"},
		elements:   []TypedObject{},
	}
	result := formatTypedObject(arr)
	if result != "[]" {
		t.Errorf("Expected '[]', got '%s'", result)
	}
}

func TestFormatArrayMultiline(t *testing.T) {
	// Create array that exceeds threshold
	elements := []TypedObject{
		&mockObject{typ: "STRING", inspect: "this is a very long string"},
		&mockObject{typ: "STRING", inspect: "another very long string here"},
		&mockObject{typ: "STRING", inspect: "and one more long string"},
	}
	arr := &mockArray{
		mockObject: mockObject{typ: "ARRAY"},
		elements:   elements,
	}
	result := formatTypedObject(arr)
	// Should be multiline
	if result[0] != '[' {
		t.Errorf("Expected '[' at start, got '%c'", result[0])
	}
	if result[len(result)-1] != ']' {
		t.Errorf("Expected ']' at end, got '%c'", result[len(result)-1])
	}
	// Should contain newlines
	if len(result) < 50 {
		t.Errorf("Expected multiline output, got short: '%s'", result)
	}
}

func TestFormatDictInline(t *testing.T) {
	dict := &mockDict{
		mockObject: mockObject{typ: "DICTIONARY"},
		keys:       []string{"a", "b"},
		values: map[string]TypedObject{
			"a": &mockObject{typ: "INTEGER", inspect: "1"},
			"b": &mockObject{typ: "INTEGER", inspect: "2"},
		},
	}
	result := formatTypedObject(dict)
	expected := "{a: 1, b: 2}"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatDictEmpty(t *testing.T) {
	dict := &mockDict{
		mockObject: mockObject{typ: "DICTIONARY"},
		keys:       []string{},
		values:     map[string]TypedObject{},
	}
	result := formatTypedObject(dict)
	if result != "{}" {
		t.Errorf("Expected '{}', got '%s'", result)
	}
}

func TestFormatDictKeyNeedsQuotes(t *testing.T) {
	dict := &mockDict{
		mockObject: mockObject{typ: "DICTIONARY"},
		keys:       []string{"foo-bar", "normal"},
		values: map[string]TypedObject{
			"foo-bar": &mockObject{typ: "INTEGER", inspect: "1"},
			"normal":  &mockObject{typ: "INTEGER", inspect: "2"},
		},
	}
	result := formatTypedObject(dict)
	expected := `{"foo-bar": 1, normal: 2}`
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatFunctionInline(t *testing.T) {
	fn := &mockFunction{
		mockObject: mockObject{typ: "FUNCTION"},
		params:     []string{"x"},
		body:       "x * 2",
	}
	result := formatTypedObject(fn)
	expected := "fn(x) { x * 2 }"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatFunctionMultipleParams(t *testing.T) {
	fn := &mockFunction{
		mockObject: mockObject{typ: "FUNCTION"},
		params:     []string{"a", "b", "c"},
		body:       "a + b + c",
	}
	result := formatTypedObject(fn)
	expected := "fn(a, b, c) { a + b + c }"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatRecordInline(t *testing.T) {
	rec := &mockRecord{
		mockObject: mockObject{typ: "RECORD"},
		schemaName: "User",
		keys:       []string{"name", "age"},
		fields: map[string]TypedObject{
			"name": &mockObject{typ: "STRING", inspect: "Alice"},
			"age":  &mockObject{typ: "INTEGER", inspect: "30"},
		},
	}
	result := formatTypedObject(rec)
	expected := `User{name: "Alice", age: 30}`
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"foo", true},
		{"fooBar", true},
		{"foo_bar", true},
		{"_foo", true},
		{"foo123", true},
		{"123foo", false},
		{"foo-bar", false},
		{"foo bar", false},
		{"", false},
	}

	for _, tt := range tests {
		result := isValidIdentifier(tt.input)
		if result != tt.expected {
			t.Errorf("isValidIdentifier(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestFitsInThreshold(t *testing.T) {
	tests := []struct {
		input     string
		threshold int
		expected  bool
	}{
		{"hello", 10, true},
		{"hello world", 10, false},
		{"hello", 5, true},
		{"hello", 4, false},
		{"hello\nworld", 20, false}, // Contains newline
	}

	for _, tt := range tests {
		result := fitsInThreshold(tt.input, tt.threshold)
		if result != tt.expected {
			t.Errorf("fitsInThreshold(%q, %d) = %v, want %v", tt.input, tt.threshold, result, tt.expected)
		}
	}
}
