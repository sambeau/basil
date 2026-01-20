package server

import (
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// TestPartPropsRecordRoundTrip tests that a Record survives being passed through Part props
func TestPartPropsRecordRoundTrip(t *testing.T) {
	// Create a Record manually (simulating what happens in user's code)
	schema := &evaluator.DSLSchema{
		Name: "Person",
		Fields: map[string]*evaluator.DSLSchemaField{
			"Firstname": {Name: "Firstname", Type: "string"},
			"Surname":   {Name: "Surname", Type: "string"},
		},
	}

	env := evaluator.NewEnvironment()
	env.PLNSecret = "test-secret"

	record := &evaluator.Record{
		Schema: schema,
		Data: map[string]ast.Expression{
			"Firstname": &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: "John"}},
			"Surname":   &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: "Smith"}},
		},
		KeyOrder: []string{"Firstname", "Surname"},
		Env:      env,
	}

	t.Logf("Original record: %s", record.Inspect())

	// Check NeedsPLNSerialization
	needsPLN := evaluator.NeedsPLNSerialization(record)
	t.Logf("NeedsPLNSerialization: %v", needsPLN)
	if !needsPLN {
		t.Fatal("Record should need PLN serialization!")
	}

	// Now simulate Part props encoding - build a dictionary with the record
	propsDict := &evaluator.Dictionary{
		Pairs:    make(map[string]ast.Expression),
		KeyOrder: []string{"person"},
		Env:      env,
	}
	propsDict.Pairs["person"] = &ast.ObjectLiteralExpression{Obj: record}

	// Encode to JSON (this is what EncodePropsToJSON does)
	propsJSON := evaluator.EncodePropsToJSON(propsDict)
	t.Logf("Encoded props JSON: %s", propsJSON)

	// Check that PLN marker is present
	if !strings.Contains(propsJSON, "__pln") {
		t.Fatalf("Expected PLN marker in encoded props, got: %s", propsJSON)
	}

	// Now simulate the round-trip - JavaScript sends it back as URL-encoded
	// JavaScript would parse the JSON, then on refresh, send it as form/query params

	// The JavaScript sends: person={"__pln":"hmac:..."}
	// We need to URL-encode this
	jsonValue := url.QueryEscape(`{"__pln":"` + extractPLNValue(propsJSON) + `"}`)

	req := httptest.NewRequest("GET", "/?_view=test&person="+jsonValue, nil)

	h := &parsleyHandler{
		server: &Server{
			stderr: io.Discard,
		},
	}

	parsedEnv := evaluator.NewEnvironment()
	parsedEnv.PLNSecret = "test-secret"

	props, err := h.parsePartProps(req, parsedEnv)
	if err != nil {
		t.Fatalf("parsePartProps error: %v", err)
	}

	// Evaluate the person prop
	personObj := evaluator.Eval(props.Pairs["person"], props.Env)
	t.Logf("Parsed person type: %T", personObj)
	t.Logf("Parsed person value: %v", personObj.Inspect())

	// Check it's still a Record (not a String!)
	_, ok := personObj.(*evaluator.Record)
	if !ok {
		t.Fatalf("Expected Record after round-trip, got %T: %v", personObj, personObj.Inspect())
	}
}

// extractPLNValue extracts the PLN value from encoded props JSON
// Input: {"person":{"__pln":"hmac:..."}}
// Output: hmac:...
func extractPLNValue(propsJSON string) string {
	// Simple extraction - find __pln value
	start := strings.Index(propsJSON, `"__pln":"`)
	if start == -1 {
		return ""
	}
	start += len(`"__pln":"`)
	end := strings.Index(propsJSON[start:], `"`)
	if end == -1 {
		return ""
	}
	return propsJSON[start : start+end]
}
