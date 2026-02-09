package pln

import (
	"sort"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// Helper to create a Dictionary from a map of Objects
func makeDict(pairs map[string]evaluator.Object) *evaluator.Dictionary {
	d := &evaluator.Dictionary{
		Pairs:    make(map[string]ast.Expression),
		KeyOrder: []string{},
	}
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := pairs[k]
		d.Pairs[k] = &ast.ObjectLiteralExpression{Obj: v}
		d.KeyOrder = append(d.KeyOrder, k)
	}
	return d
}

func TestSerializePrimitives(t *testing.T) {
	tests := []struct {
		name     string
		input    evaluator.Object
		expected string
	}{
		{"integer", &evaluator.Integer{Value: 42}, "42"},
		{"negative integer", &evaluator.Integer{Value: -17}, "-17"},
		{"float", &evaluator.Float{Value: 3.14}, "3.14"},
		{"float whole", &evaluator.Float{Value: 5.0}, "5.0"},
		{"string", &evaluator.String{Value: "hello"}, `"hello"`},
		{"string with quotes", &evaluator.String{Value: `say "hi"`}, `"say \"hi\""`},
		{"string with newline", &evaluator.String{Value: "line1\nline2"}, `"line1\nline2"`},
		{"true", &evaluator.Boolean{Value: true}, "true"},
		{"false", &evaluator.Boolean{Value: false}, "false"},
		{"null", &evaluator.Null{}, "null"},
		{"nil", nil, "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSerializer()
			result, err := s.Serialize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSerializeArrays(t *testing.T) {
	tests := []struct {
		name     string
		input    *evaluator.Array
		expected string
	}{
		{"empty", &evaluator.Array{Elements: []evaluator.Object{}}, "[]"},
		{"single", &evaluator.Array{Elements: []evaluator.Object{
			&evaluator.Integer{Value: 1},
		}}, "[1]"},
		{"multiple", &evaluator.Array{Elements: []evaluator.Object{
			&evaluator.Integer{Value: 1},
			&evaluator.Integer{Value: 2},
			&evaluator.Integer{Value: 3},
		}}, "[1, 2, 3]"},
		{"mixed", &evaluator.Array{Elements: []evaluator.Object{
			&evaluator.Integer{Value: 1},
			&evaluator.String{Value: "two"},
			&evaluator.Boolean{Value: true},
		}}, `[1, "two", true]`},
		{"nested", &evaluator.Array{Elements: []evaluator.Object{
			&evaluator.Array{Elements: []evaluator.Object{
				&evaluator.Integer{Value: 1},
				&evaluator.Integer{Value: 2},
			}},
			&evaluator.Array{Elements: []evaluator.Object{
				&evaluator.Integer{Value: 3},
				&evaluator.Integer{Value: 4},
			}},
		}}, "[[1, 2], [3, 4]]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSerializer()
			result, err := s.Serialize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSerializeDicts(t *testing.T) {
	tests := []struct {
		name     string
		input    *evaluator.Dictionary
		expected string
	}{
		{"empty", makeDict(map[string]evaluator.Object{}), "{}"},
		{"single", makeDict(map[string]evaluator.Object{
			"name": &evaluator.String{Value: "Alice"},
		}), `{name: "Alice"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSerializer()
			result, err := s.Serialize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSerializeDictMultipleKeys(t *testing.T) {
	// Keys are sorted alphabetically
	input := makeDict(map[string]evaluator.Object{
		"age":  &evaluator.Integer{Value: 30},
		"name": &evaluator.String{Value: "Alice"},
	})

	s := NewSerializer()
	result, err := s.Serialize(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `{age: 30, name: "Alice"}`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSerializeDictKeyNeedsQuoting(t *testing.T) {
	input := makeDict(map[string]evaluator.Object{
		"my-key": &evaluator.Integer{Value: 1},
	})

	s := NewSerializer()
	result, err := s.Serialize(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `{"my-key": 1}`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSerializeNested(t *testing.T) {
	input := makeDict(map[string]evaluator.Object{
		"items": &evaluator.Array{Elements: []evaluator.Object{
			makeDict(map[string]evaluator.Object{
				"id":   &evaluator.Integer{Value: 1},
				"name": &evaluator.String{Value: "Item 1"},
			}),
			makeDict(map[string]evaluator.Object{
				"id":   &evaluator.Integer{Value: 2},
				"name": &evaluator.String{Value: "Item 2"},
			}),
		}},
		"total": &evaluator.Integer{Value: 2},
	})

	s := NewSerializer()
	result, err := s.Serialize(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Keys are sorted, so items comes before total
	expected := `{items: [{id: 1, name: "Item 1"}, {id: 2, name: "Item 2"}], total: 2}`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSerializeRecord(t *testing.T) {
	schema := &evaluator.DSLSchema{Name: "Person"}
	input := &evaluator.Record{
		Schema: schema,
		Data: map[string]ast.Expression{
			"age":  &ast.ObjectLiteralExpression{Obj: &evaluator.Integer{Value: 30}},
			"name": &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: "Alice"}},
		},
	}

	s := NewSerializer()
	result, err := s.Serialize(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `@Person({age: 30, name: "Alice"})`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSerializeRecordWithErrors(t *testing.T) {
	schema := &evaluator.DSLSchema{Name: "Person"}
	input := &evaluator.Record{
		Schema: schema,
		Data: map[string]ast.Expression{
			"age":  &ast.ObjectLiteralExpression{Obj: &evaluator.Integer{Value: -5}},
			"name": &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: ""}},
		},
		Errors: map[string]*evaluator.RecordError{
			"age":  {Message: "must be positive"},
			"name": {Message: "required"},
		},
	}

	s := NewSerializer()
	result, err := s.Serialize(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `@Person({age: -5, name: ""}) @errors {age: "must be positive", name: "required"}`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSerializeRecordWithoutSchema(t *testing.T) {
	input := &evaluator.Record{
		Schema: nil,
		Data: map[string]ast.Expression{
			"name": &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: "Alice"}},
		},
	}

	s := NewSerializer()
	result, err := s.Serialize(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `@Record({name: "Alice"})`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSerializeDatetime(t *testing.T) {
	tests := []struct {
		name     string
		input    *evaluator.Dictionary
		expected string
	}{
		{
			"date",
			makeDict(map[string]evaluator.Object{
				"__type": &evaluator.String{Value: "datetime"},
				"kind":   &evaluator.String{Value: "date"},
				"year":   &evaluator.Integer{Value: 2024},
				"month":  &evaluator.Integer{Value: 1},
				"day":    &evaluator.Integer{Value: 15},
			}),
			"@2024-01-15",
		},
		{
			"datetime",
			makeDict(map[string]evaluator.Object{
				"__type": &evaluator.String{Value: "datetime"},
				"kind":   &evaluator.String{Value: "datetime"},
				"year":   &evaluator.Integer{Value: 2024},
				"month":  &evaluator.Integer{Value: 1},
				"day":    &evaluator.Integer{Value: 15},
				"hour":   &evaluator.Integer{Value: 14},
				"minute": &evaluator.Integer{Value: 30},
				"second": &evaluator.Integer{Value: 0},
			}),
			"@2024-01-15T14:30:00",
		},
		{
			"datetime with UTC",
			makeDict(map[string]evaluator.Object{
				"__type":   &evaluator.String{Value: "datetime"},
				"kind":     &evaluator.String{Value: "datetime"},
				"year":     &evaluator.Integer{Value: 2024},
				"month":    &evaluator.Integer{Value: 1},
				"day":      &evaluator.Integer{Value: 15},
				"hour":     &evaluator.Integer{Value: 14},
				"minute":   &evaluator.Integer{Value: 30},
				"second":   &evaluator.Integer{Value: 0},
				"timezone": &evaluator.String{Value: "UTC"},
			}),
			"@2024-01-15T14:30:00Z",
		},
		{
			"time",
			makeDict(map[string]evaluator.Object{
				"__type": &evaluator.String{Value: "datetime"},
				"kind":   &evaluator.String{Value: "time"},
				"hour":   &evaluator.Integer{Value: 14},
				"minute": &evaluator.Integer{Value: 30},
				"second": &evaluator.Integer{Value: 45},
			}),
			"@T14:30:45",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSerializer()
			result, err := s.Serialize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSerializePath(t *testing.T) {
	input := makeDict(map[string]evaluator.Object{
		"__type": &evaluator.String{Value: "path"},
		"value":  &evaluator.String{Value: "/home/user/data.txt"},
	})

	s := NewSerializer()
	result, err := s.Serialize(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "@/home/user/data.txt"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSerializeURL(t *testing.T) {
	input := makeDict(map[string]evaluator.Object{
		"__type": &evaluator.String{Value: "url"},
		"value":  &evaluator.String{Value: "https://example.com/api"},
	})

	s := NewSerializer()
	result, err := s.Serialize(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "@https://example.com/api"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSerializeErrorOnFunction(t *testing.T) {
	input := &evaluator.Function{}

	s := NewSerializer()
	_, err := s.Serialize(input)
	if err == nil {
		t.Error("expected error for function, got nil")
	}
	if !strings.Contains(err.Error(), "cannot serialize function") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSerializeErrorOnBuiltin(t *testing.T) {
	input := &evaluator.Builtin{}

	s := NewSerializer()
	_, err := s.Serialize(input)
	if err == nil {
		t.Error("expected error for builtin, got nil")
	}
	if !strings.Contains(err.Error(), "cannot serialize builtin") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSerializeCircularReference(t *testing.T) {
	// Create a circular array
	arr := &evaluator.Array{Elements: []evaluator.Object{}}
	arr.Elements = append(arr.Elements, arr) // circular!

	s := NewSerializer()
	_, err := s.Serialize(arr)
	if err == nil {
		t.Error("expected error for circular reference, got nil")
	}
	if !strings.Contains(err.Error(), "circular reference") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSerializePretty(t *testing.T) {
	input := makeDict(map[string]evaluator.Object{
		"items": &evaluator.Array{Elements: []evaluator.Object{
			&evaluator.Integer{Value: 1},
			&evaluator.Integer{Value: 2},
		}},
		"name": &evaluator.String{Value: "test"},
	})

	s := NewPrettySerializer("  ")
	result, err := s.Serialize(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that it has newlines and indentation
	if !strings.Contains(result, "\n") {
		t.Error("expected newlines in pretty output")
	}
	if !strings.Contains(result, "  ") {
		t.Error("expected indentation in pretty output")
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input evaluator.Object
	}{
		{"integer", &evaluator.Integer{Value: 42}},
		{"float", &evaluator.Float{Value: 3.14}},
		{"string", &evaluator.String{Value: "hello world"}},
		{"array", &evaluator.Array{Elements: []evaluator.Object{
			&evaluator.Integer{Value: 1},
			&evaluator.Integer{Value: 2},
			&evaluator.Integer{Value: 3},
		}}},
		{"dict", makeDict(map[string]evaluator.Object{
			"name": &evaluator.String{Value: "Alice"},
			"age":  &evaluator.Integer{Value: 30},
		})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			s := NewSerializer()
			serialized, err := s.Serialize(tt.input)
			if err != nil {
				t.Fatalf("serialize error: %v", err)
			}

			// Parse
			p := NewParser(serialized)
			parsed, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v (input: %q)", err, serialized)
			}

			// Serialize again
			s2 := NewSerializer()
			reserialized, err := s2.Serialize(parsed)
			if err != nil {
				t.Fatalf("reserialize error: %v", err)
			}

			// Should be equal
			if serialized != reserialized {
				t.Errorf("round-trip mismatch:\noriginal:     %q\nreserialized: %q", serialized, reserialized)
			}
		})
	}
}

func TestSerializeUnicode(t *testing.T) {
	input := &evaluator.String{Value: "Hello"}

	s := NewSerializer()
	result, err := s.Serialize(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Basic ASCII should not be escaped
	if result != `"Hello"` {
		t.Errorf("expected %q, got %q", `"Hello"`, result)
	}
}

func TestSerializeControlChars(t *testing.T) {
	input := &evaluator.String{Value: "a\x00b"} // null byte

	s := NewSerializer()
	result, err := s.Serialize(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Control chars should be escaped
	if !strings.Contains(result, "\\u0000") {
		t.Errorf("expected unicode escape for null byte, got %q", result)
	}
}
