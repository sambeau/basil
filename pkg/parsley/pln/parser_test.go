package pln

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

func TestParsePrimitives(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"42", int64(42)},
		{"-42", int64(-42)},
		{"3.14", 3.14},
		{"-3.14", -3.14},
		{`"hello"`, "hello"},
		{`"hello\nworld"`, "hello\nworld"},
		{"true", true},
		{"false", false},
		{"null", nil},
	}

	for _, tt := range tests {
		p := NewParser(tt.input)
		obj, err := p.Parse()
		if err != nil {
			t.Errorf("input %q: unexpected error: %v", tt.input, err)
			continue
		}

		switch expected := tt.expected.(type) {
		case int64:
			intObj, ok := obj.(*evaluator.Integer)
			if !ok {
				t.Errorf("input %q: expected Integer, got %T", tt.input, obj)
				continue
			}
			if intObj.Value != expected {
				t.Errorf("input %q: expected %d, got %d", tt.input, expected, intObj.Value)
			}
		case float64:
			floatObj, ok := obj.(*evaluator.Float)
			if !ok {
				t.Errorf("input %q: expected Float, got %T", tt.input, obj)
				continue
			}
			if floatObj.Value != expected {
				t.Errorf("input %q: expected %f, got %f", tt.input, expected, floatObj.Value)
			}
		case string:
			strObj, ok := obj.(*evaluator.String)
			if !ok {
				t.Errorf("input %q: expected String, got %T", tt.input, obj)
				continue
			}
			if strObj.Value != expected {
				t.Errorf("input %q: expected %q, got %q", tt.input, expected, strObj.Value)
			}
		case bool:
			boolObj, ok := obj.(*evaluator.Boolean)
			if !ok {
				t.Errorf("input %q: expected Boolean, got %T", tt.input, obj)
				continue
			}
			if boolObj.Value != expected {
				t.Errorf("input %q: expected %v, got %v", tt.input, expected, boolObj.Value)
			}
		case nil:
			_, ok := obj.(*evaluator.Null)
			if !ok {
				t.Errorf("input %q: expected Null, got %T", tt.input, obj)
			}
		}
	}
}

func TestParseArrays(t *testing.T) {
	tests := []struct {
		input    string
		expected []int64
	}{
		{"[]", []int64{}},
		{"[1]", []int64{1}},
		{"[1, 2, 3]", []int64{1, 2, 3}},
		{"[1, 2, 3,]", []int64{1, 2, 3}}, // trailing comma
	}

	for _, tt := range tests {
		p := NewParser(tt.input)
		obj, err := p.Parse()
		if err != nil {
			t.Errorf("input %q: unexpected error: %v", tt.input, err)
			continue
		}

		arr, ok := obj.(*evaluator.Array)
		if !ok {
			t.Errorf("input %q: expected Array, got %T", tt.input, obj)
			continue
		}

		if len(arr.Elements) != len(tt.expected) {
			t.Errorf("input %q: expected %d elements, got %d", tt.input, len(tt.expected), len(arr.Elements))
			continue
		}

		for i, exp := range tt.expected {
			intObj, ok := arr.Elements[i].(*evaluator.Integer)
			if !ok {
				t.Errorf("input %q element %d: expected Integer, got %T", tt.input, i, arr.Elements[i])
				continue
			}
			if intObj.Value != exp {
				t.Errorf("input %q element %d: expected %d, got %d", tt.input, i, exp, intObj.Value)
			}
		}
	}
}

func TestParseDicts(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]int64
	}{
		{"{}", map[string]int64{}},
		{"{a: 1}", map[string]int64{"a": 1}},
		{"{a: 1, b: 2}", map[string]int64{"a": 1, "b": 2}},
		{`{"a": 1, "b": 2}`, map[string]int64{"a": 1, "b": 2}},
		{"{a: 1,}", map[string]int64{"a": 1}}, // trailing comma
	}

	for _, tt := range tests {
		p := NewParser(tt.input)
		obj, err := p.Parse()
		if err != nil {
			t.Errorf("input %q: unexpected error: %v", tt.input, err)
			continue
		}

		dict, ok := obj.(*evaluator.Dictionary)
		if !ok {
			t.Errorf("input %q: expected Dictionary, got %T", tt.input, obj)
			continue
		}

		if len(dict.Pairs) != len(tt.expected) {
			t.Errorf("input %q: expected %d pairs, got %d", tt.input, len(tt.expected), len(dict.Pairs))
			continue
		}

		for key, exp := range tt.expected {
			expr, ok := dict.Pairs[key]
			if !ok {
				t.Errorf("input %q: missing key %q", tt.input, key)
				continue
			}
			objLit, ok := expr.(*ast.ObjectLiteralExpression)
			if !ok {
				t.Errorf("input %q key %q: expected ObjectLiteralExpression, got %T", tt.input, key, expr)
				continue
			}
			intObj, ok := objLit.Obj.(*evaluator.Integer)
			if !ok {
				t.Errorf("input %q key %q: expected Integer, got %T", tt.input, key, objLit.Obj)
				continue
			}
			if intObj.Value != exp {
				t.Errorf("input %q key %q: expected %d, got %d", tt.input, key, exp, intObj.Value)
			}
		}
	}
}

func TestParseNested(t *testing.T) {
	input := `{a: [1, {b: 2}]}`

	p := NewParser(input)
	obj, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dict, ok := obj.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T", obj)
	}

	aExpr, ok := dict.Pairs["a"]
	if !ok {
		t.Fatal("missing key 'a'")
	}

	aObjLit := aExpr.(*ast.ObjectLiteralExpression)
	aArr, ok := aObjLit.Obj.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array for 'a', got %T", aObjLit.Obj)
	}

	if len(aArr.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(aArr.Elements))
	}

	// First element is 1
	intObj, ok := aArr.Elements[0].(*evaluator.Integer)
	if !ok || intObj.Value != 1 {
		t.Errorf("expected first element to be 1, got %v", aArr.Elements[0])
	}

	// Second element is {b: 2}
	innerDict, ok := aArr.Elements[1].(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary for second element, got %T", aArr.Elements[1])
	}

	bExpr, ok := innerDict.Pairs["b"]
	if !ok {
		t.Fatal("missing key 'b' in inner dict")
	}
	bObjLit := bExpr.(*ast.ObjectLiteralExpression)
	bInt, ok := bObjLit.Obj.(*evaluator.Integer)
	if !ok || bInt.Value != 2 {
		t.Errorf("expected b=2, got %v", bObjLit.Obj)
	}
}

func TestParseRecordWithoutSchema(t *testing.T) {
	input := `@Person({name: "Alice", age: 30})`

	p := NewParser(input)
	obj, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Without a schema resolver, should still return a Record with an inferred schema
	record, ok := obj.(*evaluator.Record)
	if !ok {
		t.Fatalf("expected Record, got %T", obj)
	}

	if record.Schema == nil {
		t.Fatal("expected schema to be set")
	}
	if record.Schema.Name != "Person" {
		t.Errorf("expected schema name 'Person', got %q", record.Schema.Name)
	}
}

func TestParseRecordWithErrors(t *testing.T) {
	input := `@Person({name: "", email: "bad"}) @errors {name: "Required", email: "Invalid"}`

	p := NewParser(input)
	obj, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Without resolver, should still return a Record with errors
	record, ok := obj.(*evaluator.Record)
	if !ok {
		t.Fatalf("expected Record, got %T", obj)
	}

	if record.Schema == nil || record.Schema.Name != "Person" {
		t.Error("expected schema name 'Person'")
	}

	// Check errors were parsed
	if record.Errors == nil || len(record.Errors) != 2 {
		t.Errorf("expected 2 errors, got %v", record.Errors)
	}
}

func TestParseDatetime(t *testing.T) {
	tests := []struct {
		input        string
		expectedKind string
	}{
		{"@2024-01-20", "date"},
		{"@2024-01-20T10:30:00Z", "datetime"},
		{"@10:30:00", "time"},
	}

	for _, tt := range tests {
		p := NewParser(tt.input)
		obj, err := p.Parse()
		if err != nil {
			t.Errorf("input %q: unexpected error: %v", tt.input, err)
			continue
		}

		dict, ok := obj.(*evaluator.Dictionary)
		if !ok {
			t.Errorf("input %q: expected Dictionary, got %T", tt.input, obj)
			continue
		}

		// Check __type is datetime
		typeExpr, ok := dict.Pairs["__type"]
		if !ok {
			t.Errorf("input %q: missing __type", tt.input)
			continue
		}
		typeObjLit := typeExpr.(*ast.ObjectLiteralExpression)
		typeStr, ok := typeObjLit.Obj.(*evaluator.String)
		if !ok || typeStr.Value != "datetime" {
			t.Errorf("input %q: expected __type='datetime', got %v", tt.input, typeObjLit.Obj)
		}

		// Check kind
		kindExpr, ok := dict.Pairs["kind"]
		if !ok {
			t.Errorf("input %q: missing kind", tt.input)
			continue
		}
		kindObjLit := kindExpr.(*ast.ObjectLiteralExpression)
		kindStr, ok := kindObjLit.Obj.(*evaluator.String)
		if !ok || kindStr.Value != tt.expectedKind {
			t.Errorf("input %q: expected kind=%q, got %v", tt.input, tt.expectedKind, kindObjLit.Obj)
		}
	}
}

func TestParsePath(t *testing.T) {
	input := "@/path/to/file"

	p := NewParser(input)
	obj, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dict, ok := obj.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T", obj)
	}

	typeExpr := dict.Pairs["__type"].(*ast.ObjectLiteralExpression)
	typeStr := typeExpr.Obj.(*evaluator.String)
	if typeStr.Value != "path" {
		t.Errorf("expected __type='path', got %q", typeStr.Value)
	}

	valueExpr := dict.Pairs["value"].(*ast.ObjectLiteralExpression)
	valueStr := valueExpr.Obj.(*evaluator.String)
	if valueStr.Value != "/path/to/file" {
		t.Errorf("expected value='/path/to/file', got %q", valueStr.Value)
	}
}

func TestParseURL(t *testing.T) {
	input := "@https://example.com/api"

	p := NewParser(input)
	obj, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dict, ok := obj.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T", obj)
	}

	typeExpr := dict.Pairs["__type"].(*ast.ObjectLiteralExpression)
	typeStr := typeExpr.Obj.(*evaluator.String)
	if typeStr.Value != "url" {
		t.Errorf("expected __type='url', got %q", typeStr.Value)
	}

	valueExpr := dict.Pairs["value"].(*ast.ObjectLiteralExpression)
	valueStr := valueExpr.Obj.(*evaluator.String)
	if valueStr.Value != "https://example.com/api" {
		t.Errorf("expected value='https://example.com/api', got %q", valueStr.Value)
	}
}

func TestParseComments(t *testing.T) {
	input := `// This is a comment
{
    // name field
    name: "Alice",
    age: 30  // inline comment (should not work, but we're lenient)
}`

	p := NewParser(input)
	obj, err := p.Parse()
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

func TestParseDeepNesting(t *testing.T) {
	// Create deeply nested structure
	input := ""
	for i := 0; i < MaxNestingDepth+5; i++ {
		input += "["
	}
	input += "1"
	for i := 0; i < MaxNestingDepth+5; i++ {
		input += "]"
	}

	p := NewParser(input)
	_, err := p.Parse()
	if err == nil {
		t.Error("expected error for deep nesting, got nil")
	}
}

func TestParseErrorOnExpression(t *testing.T) {
	// PLN should not allow expressions
	tests := []string{
		"1 + 1",
		"{name: x}",
	}

	for _, input := range tests {
		p := NewParser(input)
		_, err := p.Parse()
		if err == nil {
			t.Errorf("input %q: expected error, got nil", input)
		}
	}
}
