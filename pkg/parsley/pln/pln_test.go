package pln

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

func TestSerializeAPI(t *testing.T) {
	obj := &evaluator.Integer{Value: 42}
	result, err := Serialize(obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "42" {
		t.Errorf("expected %q, got %q", "42", result)
	}
}

func TestSerializePrettyAPI(t *testing.T) {
	obj := &evaluator.Array{Elements: []evaluator.Object{
		&evaluator.Integer{Value: 1},
		&evaluator.Integer{Value: 2},
	}}
	result, err := SerializePretty(obj, "  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "\n") {
		t.Error("expected newlines in pretty output")
	}
}

func TestDeserializeAPI(t *testing.T) {
	obj, err := Deserialize(`{name: "Alice", age: 30}`, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dict, ok := obj.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T", obj)
	}

	if len(dict.Pairs) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(dict.Pairs))
	}
}

func TestParseAPI(t *testing.T) {
	obj, err := Parse(`[1, 2, 3]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	arr, ok := obj.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", obj)
	}

	if len(arr.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr.Elements))
	}
}

func TestMustParseAPI(t *testing.T) {
	// Should not panic for valid PLN
	obj := MustParse(`"hello"`)
	if str, ok := obj.(*evaluator.String); !ok || str.Value != "hello" {
		t.Errorf("expected String 'hello', got %v", obj)
	}
}

func TestMustParsePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid PLN")
		}
	}()
	MustParse(`{invalid`)
}

func TestValidateAPI(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid integer", "42", false},
		{"valid dict", `{a: 1}`, false},
		{"valid array", `[1, 2, 3]`, false},
		{"invalid", `{a: }`, true},
		{"expression", `1 + 1`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestIsValidPLNAPI(t *testing.T) {
	if !IsValidPLN(`42`) {
		t.Error("expected 42 to be valid PLN")
	}
	if IsValidPLN(`{invalid`) {
		t.Error("expected {invalid to be invalid PLN")
	}
}

func TestSchemaResolution(t *testing.T) {
	// Create a mock schema
	schema := &evaluator.DSLSchema{
		Name: "Person",
		Fields: map[string]*evaluator.DSLSchemaField{
			"name": {Name: "name", Type: "string"},
		},
	}

	// Schema resolver that returns our mock schema
	resolver := func(name string) *evaluator.DSLSchema {
		if name == "Person" {
			return schema
		}
		return nil
	}

	// Create an environment
	env := evaluator.NewEnvironment()

	// Deserialize with schema resolution
	obj, err := Deserialize(`@Person({name: "Alice"})`, resolver, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	record, ok := obj.(*evaluator.Record)
	if !ok {
		t.Fatalf("expected Record, got %T", obj)
	}

	if record.Schema != schema {
		t.Error("expected record to have resolved schema")
	}
}

func TestUnknownSchemaBecomesRecord(t *testing.T) {
	// Without a resolver, records still become Records with inferred schema
	// This allows Records to survive round-trips even without full schema definitions
	obj, err := Deserialize(`@UnknownType({x: 1})`, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	record, ok := obj.(*evaluator.Record)
	if !ok {
		t.Fatalf("expected Record for unknown schema, got %T", obj)
	}

	// Check schema was inferred
	if record.Schema == nil {
		t.Fatal("expected schema to be set")
	}
	if record.Schema.Name != "UnknownType" {
		t.Errorf("expected schema name 'UnknownType', got %q", record.Schema.Name)
	}

	// Check data field was preserved
	if _, hasX := record.Data["x"]; !hasX {
		t.Error("expected 'x' field in record data")
	}
}

func TestRoundTripAPI(t *testing.T) {
	// Create a complex object
	original := makeDict(map[string]evaluator.Object{
		"items": &evaluator.Array{Elements: []evaluator.Object{
			&evaluator.Integer{Value: 1},
			&evaluator.Integer{Value: 2},
		}},
		"name": &evaluator.String{Value: "test"},
	})

	// Serialize
	pln, err := Serialize(original)
	if err != nil {
		t.Fatalf("serialize error: %v", err)
	}

	// Deserialize
	parsed, err := Parse(pln)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Serialize again
	pln2, err := Serialize(parsed)
	if err != nil {
		t.Fatalf("reserialize error: %v", err)
	}

	// Should be identical
	if pln != pln2 {
		t.Errorf("round-trip mismatch:\noriginal:     %q\nreserialized: %q", pln, pln2)
	}
}
