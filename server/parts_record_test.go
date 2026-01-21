package server

import (
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
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

// TestRecordInterpolationRoundTrip tests that a Record interpolated via @{row}
// survives the round-trip through Parts.refresh JavaScript call
func TestRecordInterpolationRoundTrip(t *testing.T) {
	// Create a Record with PLN secret
	schema := &evaluator.DSLSchema{
		Name: "Item",
		Fields: map[string]*evaluator.DSLSchemaField{
			"id":    {Name: "id", Type: "int"},
			"name":  {Name: "name", Type: "string"},
			"price": {Name: "price", Type: "int"},
		},
	}

	env := evaluator.NewEnvironment()
	env.PLNSecret = "test-secret"

	record := &evaluator.Record{
		Schema: schema,
		Data: map[string]ast.Expression{
			"id":    &ast.ObjectLiteralExpression{Obj: &evaluator.Integer{Value: 42}},
			"name":  &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: "Widget"}},
			"price": &ast.ObjectLiteralExpression{Obj: &evaluator.Integer{Value: 999}},
		},
		KeyOrder: []string{"id", "name", "price"},
		Env:      env,
	}

	// Simulate what @{row} produces when interpolated in a JavaScript context
	// This uses evaluator.objectToTemplateString, not server.objectToTemplateString
	// We need to use the actual Parsley evaluation path
	// For testing, we'll use EncodePropsToJSON which uses the same PLN encoding logic
	propsDict := &evaluator.Dictionary{
		Pairs:    map[string]ast.Expression{"record": &ast.ObjectLiteralExpression{Obj: record}},
		KeyOrder: []string{"record"},
		Env:      env,
	}
	interpolated := evaluator.EncodePropsToJSON(propsDict)
	t.Logf("Encoded Props JSON: %s", interpolated)

	// Extract just the record value from {"record":{...}}
	// The encoded JSON looks like: {"record":{"__pln":"..."}}
	// When JavaScript sends it back, it sends: record={"__pln":"..."}
	plnValue := extractPLNValue(interpolated)
	if plnValue == "" {
		t.Fatalf("Expected PLN encoding in Record, got: %s", interpolated)
	}

	// Now simulate JavaScript sending this back: record={"__pln":"..."}
	jsonValue := url.QueryEscape(`{"__pln":"` + plnValue + `"}`)
	req := httptest.NewRequest("GET", "/?_view=test&record="+jsonValue, nil)

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

	// Evaluate the record prop
	recordObj := evaluator.Eval(props.Pairs["record"], props.Env)
	t.Logf("Parsed record type: %T", recordObj)
	t.Logf("Parsed record value: %v", recordObj.Inspect())

	// Should be a Record
	parsedRecord, ok := recordObj.(*evaluator.Record)
	if !ok {
		t.Fatalf("Expected Record after round-trip, got %T: %v", recordObj, recordObj.Inspect())
	}

	// Verify field values
	id := parsedRecord.Get("id", parsedEnv)
	if idInt, ok := id.(*evaluator.Integer); !ok || idInt.Value != 42 {
		t.Errorf("Expected id=42, got %v", id)
	}

	name := parsedRecord.Get("name", parsedEnv)
	if nameStr, ok := name.(*evaluator.String); !ok || nameStr.Value != "Widget" {
		t.Errorf("Expected name=Widget, got %v", name)
	}

	price := parsedRecord.Get("price", parsedEnv)
	if priceInt, ok := price.(*evaluator.Integer); !ok || priceInt.Value != 999 {
		t.Errorf("Expected price=999, got %v", price)
	}
}

// TestPLNSecretPropagatesInImportedModules tests that PLNSecret propagates to
// functions defined in imported modules. This is important for Records created
// by functions like birthdayToPerson that are defined in helper modules.
func TestPLNSecretPropagatesInImportedModules(t *testing.T) {
	// Create temp directory for test module
	tmpDir := t.TempDir()

	// Write a helper module that defines a function returning a Record
	helperContent := `
@schema Person { Firstname: string, Surname: string }

export createPerson = fn(data) {
	data.as(Person)
}
`
	helperPath := tmpDir + "/helpers.pars"
	if err := os.WriteFile(helperPath, []byte(helperContent), 0644); err != nil {
		t.Fatalf("Failed to write helper module: %v", err)
	}

	// Main program that imports and uses the helper
	mainContent := `
let {createPerson} = import @(` + helperPath + `)

let person = createPerson({Firstname: "John", Surname: "Doe"})
person
`

	l := lexer.NewWithFilename(mainContent, tmpDir+"/main.pars")
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	// Create environment with PLNSecret set
	env := evaluator.NewEnvironment()
	env.Filename = tmpDir + "/main.pars"
	env.RootPath = tmpDir
	env.PLNSecret = "test-secret"
	env.Security = &evaluator.SecurityPolicy{
		AllowExecute: []string{tmpDir},
	}

	// Clear module cache to ensure fresh evaluation
	evaluator.ClearModuleCache()

	// Evaluate
	result := evaluator.Eval(program, env)
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("Evaluation error: %v", result.Inspect())
	}

	// Result should be a Record
	record, ok := result.(*evaluator.Record)
	if !ok {
		t.Fatalf("Expected Record, got %T: %v", result, result.Inspect())
	}

	// The Record's environment should have PLNSecret
	if record.Env == nil {
		t.Fatal("Record.Env is nil")
	}
	if record.Env.PLNSecret == "" {
		t.Fatal("Record.Env.PLNSecret is empty - PLNSecret did not propagate to imported module")
	}
	if record.Env.PLNSecret != "test-secret" {
		t.Errorf("Record.Env.PLNSecret = %q, want %q", record.Env.PLNSecret, "test-secret")
	}

	t.Logf("Record.Env.PLNSecret: %q (correctly propagated)", record.Env.PLNSecret)
}
