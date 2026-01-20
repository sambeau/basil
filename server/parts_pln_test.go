package server

import (
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// testServer creates a Server suitable for testing with stderr set
func testServer() *Server {
	return &Server{
		stderr: io.Discard, // Prevent nil pointer on logError
	}
}

func TestParsePartPropsSimple(t *testing.T) {
	// Create a mock handler
	h := &parsleyHandler{
		server: testServer(),
	}

	// Create request with simple props
	req := httptest.NewRequest("GET", "/?_view=test&name=Alice&count=42", nil)
	env := evaluator.NewEnvironment()

	props, err := h.parsePartProps(req, env)
	if err != nil {
		t.Fatalf("parsePartProps failed: %v", err)
	}

	// Check name prop
	nameObj := evaluator.Eval(props.Pairs["name"], env)
	if strObj, ok := nameObj.(*evaluator.String); ok {
		if strObj.Value != "Alice" {
			t.Errorf("expected name='Alice', got %q", strObj.Value)
		}
	} else {
		t.Errorf("expected String for name, got %T", nameObj)
	}

	// Check count prop (should be coerced to integer)
	countObj := evaluator.Eval(props.Pairs["count"], env)
	if intObj, ok := countObj.(*evaluator.Integer); ok {
		if intObj.Value != 42 {
			t.Errorf("expected count=42, got %d", intObj.Value)
		}
	} else {
		t.Errorf("expected Integer for count, got %T", countObj)
	}
}

func TestParsePartPropsBoolean(t *testing.T) {
	h := &parsleyHandler{
		server: testServer(),
	}

	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"on", true},
		{"off", false},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/?_view=test&flag="+tt.value, nil)
		env := evaluator.NewEnvironment()

		props, err := h.parsePartProps(req, env)
		if err != nil {
			t.Fatalf("parsePartProps failed for %q: %v", tt.value, err)
		}

		flagObj := evaluator.Eval(props.Pairs["flag"], env)
		if boolObj, ok := flagObj.(*evaluator.Boolean); ok {
			if boolObj.Value != tt.expected {
				t.Errorf("for %q: expected %v, got %v", tt.value, tt.expected, boolObj.Value)
			}
		} else {
			t.Errorf("expected Boolean for %q, got %T", tt.value, flagObj)
		}
	}
}

func TestParsePartPropsJSON(t *testing.T) {
	h := &parsleyHandler{
		server: testServer(),
	}

	// JSON-encoded dict
	jsonValue := url.QueryEscape(`{"x":1,"y":"test"}`)
	req := httptest.NewRequest("GET", "/?_view=test&data="+jsonValue, nil)
	env := evaluator.NewEnvironment()

	props, err := h.parsePartProps(req, env)
	if err != nil {
		t.Fatalf("parsePartProps failed: %v", err)
	}

	dataObj := evaluator.Eval(props.Pairs["data"], env)
	if dict, ok := dataObj.(*evaluator.Dictionary); ok {
		// Check x
		xObj := evaluator.Eval(dict.Pairs["x"], dict.Env)
		if intObj, ok := xObj.(*evaluator.Integer); ok {
			if intObj.Value != 1 {
				t.Errorf("expected x=1, got %d", intObj.Value)
			}
		} else {
			t.Errorf("expected Integer for x, got %T", xObj)
		}

		// Check y
		yObj := evaluator.Eval(dict.Pairs["y"], dict.Env)
		if strObj, ok := yObj.(*evaluator.String); ok {
			if strObj.Value != "test" {
				t.Errorf("expected y='test', got %q", strObj.Value)
			}
		} else {
			t.Errorf("expected String for y, got %T", yObj)
		}
	} else {
		t.Errorf("expected Dictionary for data, got %T", dataObj)
	}
}

func TestParsePartPropsPLN(t *testing.T) {
	// The PLN signing functions are registered by server/pln_hooks.go init()
	// Verify they're registered
	if evaluator.DeserializePLNProp == nil {
		t.Fatal("DeserializePLNProp not registered - pln_hooks.go init() not running")
	}

	h := &parsleyHandler{
		server: testServer(),
	}

	// Create a signed PLN value
	secret := "test-secret"
	plnVal := `{name: "Alice", age: 30}`
	signedPLN := SignPLN(plnVal, secret)

	// JSON-encode the PLN marker
	jsonMarker := `{"__pln":"` + signedPLN + `"}`
	jsonValue := url.QueryEscape(jsonMarker)

	req := httptest.NewRequest("GET", "/?_view=test&person="+jsonValue, nil)

	env := evaluator.NewEnvironment()
	env.PLNSecret = secret

	props, err := h.parsePartProps(req, env)
	if err != nil {
		t.Fatalf("parsePartProps failed: %v", err)
	}

	personObj := evaluator.Eval(props.Pairs["person"], env)
	if dict, ok := personObj.(*evaluator.Dictionary); ok {
		// Check name
		nameObj := evaluator.Eval(dict.Pairs["name"], dict.Env)
		if strObj, ok := nameObj.(*evaluator.String); ok {
			if strObj.Value != "Alice" {
				t.Errorf("expected name='Alice', got %q", strObj.Value)
			}
		} else {
			t.Errorf("expected String for name, got %T", nameObj)
		}

		// Check age
		ageObj := evaluator.Eval(dict.Pairs["age"], dict.Env)
		if intObj, ok := ageObj.(*evaluator.Integer); ok {
			if intObj.Value != 30 {
				t.Errorf("expected age=30, got %d", intObj.Value)
			}
		} else {
			t.Errorf("expected Integer for age, got %T", ageObj)
		}
	} else {
		t.Errorf("expected Dictionary for person, got %T", personObj)
	}
}

func TestParsePartPropsPLNTampered(t *testing.T) {
	h := &parsleyHandler{
		server: testServer(),
	}

	// Create a signed PLN value with one secret
	pln := `{name: "Alice"}`
	signedPLN := SignPLN(pln, "secret1")

	// JSON-encode the PLN marker
	jsonValue := url.QueryEscape(`{"__pln":"` + signedPLN + `"}`)
	req := httptest.NewRequest("GET", "/?_view=test&person="+jsonValue, nil)

	// Use different secret - should fail verification
	env := evaluator.NewEnvironment()
	env.PLNSecret = "secret2"

	props, err := h.parsePartProps(req, env)
	if err != nil {
		t.Fatalf("parsePartProps failed: %v", err)
	}

	// Should fall back to the raw JSON object (with __pln key)
	personObj := evaluator.Eval(props.Pairs["person"], env)
	if dict, ok := personObj.(*evaluator.Dictionary); ok {
		// Should still have __pln key (verification failed, fell back to JSON)
		if _, hasPLN := dict.Pairs["__pln"]; !hasPLN {
			t.Error("expected __pln key in fallback dict")
		}
	} else {
		t.Errorf("expected Dictionary for person, got %T", personObj)
	}
}

func TestParsePartPropsPost(t *testing.T) {
	h := &parsleyHandler{
		server: testServer(),
	}

	// Create POST request with form data
	form := url.Values{}
	form.Set("name", "Bob")
	form.Set("count", "99")

	req := httptest.NewRequest("POST", "/?_view=test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	env := evaluator.NewEnvironment()

	props, err := h.parsePartProps(req, env)
	if err != nil {
		t.Fatalf("parsePartProps failed: %v", err)
	}

	// Check name prop
	nameObj := evaluator.Eval(props.Pairs["name"], env)
	if strObj, ok := nameObj.(*evaluator.String); ok {
		if strObj.Value != "Bob" {
			t.Errorf("expected name='Bob', got %q", strObj.Value)
		}
	} else {
		t.Errorf("expected String for name, got %T", nameObj)
	}
}

func TestJSONToObject(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		checkFn func(evaluator.Object) bool
	}{
		{
			"null",
			nil,
			func(o evaluator.Object) bool {
				_, ok := o.(*evaluator.Null)
				return ok
			},
		},
		{
			"boolean true",
			true,
			func(o evaluator.Object) bool {
				b, ok := o.(*evaluator.Boolean)
				return ok && b.Value == true
			},
		},
		{
			"integer",
			float64(42), // JSON numbers are float64
			func(o evaluator.Object) bool {
				i, ok := o.(*evaluator.Integer)
				return ok && i.Value == 42
			},
		},
		{
			"float",
			3.14,
			func(o evaluator.Object) bool {
				f, ok := o.(*evaluator.Float)
				return ok && f.Value == 3.14
			},
		},
		{
			"string",
			"hello",
			func(o evaluator.Object) bool {
				s, ok := o.(*evaluator.String)
				return ok && s.Value == "hello"
			},
		},
		{
			"array",
			[]interface{}{float64(1), float64(2), float64(3)},
			func(o evaluator.Object) bool {
				a, ok := o.(*evaluator.Array)
				return ok && len(a.Elements) == 3
			},
		},
		{
			"object",
			map[string]interface{}{"a": float64(1)},
			func(o evaluator.Object) bool {
				d, ok := o.(*evaluator.Dictionary)
				return ok && len(d.Pairs) == 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jsonToObject(tt.input)
			if !tt.checkFn(result) {
				t.Errorf("unexpected result for %s: %v (%T)", tt.name, result, result)
			}
		})
	}
}

func TestNeedsPLNSerialization(t *testing.T) {
	env := evaluator.NewEnvironment()

	tests := []struct {
		name     string
		obj      evaluator.Object
		expected bool
	}{
		{
			"integer",
			&evaluator.Integer{Value: 42},
			false,
		},
		{
			"string",
			&evaluator.String{Value: "hello"},
			false,
		},
		{
			"simple dict",
			&evaluator.Dictionary{
				Pairs: map[string]ast.Expression{
					"name": &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: "Alice"}},
				},
				Env: env,
			},
			false,
		},
		{
			"record",
			&evaluator.Record{
				Schema: &evaluator.DSLSchema{Name: "Person"},
				Data:   map[string]ast.Expression{},
			},
			true,
		},
		{
			"datetime dict",
			&evaluator.Dictionary{
				Pairs: map[string]ast.Expression{
					"__type": &ast.ObjectLiteralExpression{Obj: &evaluator.String{Value: "datetime"}},
					"year":   &ast.ObjectLiteralExpression{Obj: &evaluator.Integer{Value: 2024}},
				},
				Env: env,
			},
			true,
		},
		{
			"nested record",
			&evaluator.Dictionary{
				Pairs: map[string]ast.Expression{
					"user": &ast.ObjectLiteralExpression{
						Obj: &evaluator.Record{
							Schema: &evaluator.DSLSchema{Name: "User"},
							Data:   map[string]ast.Expression{},
						},
					},
				},
				Env: env,
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluator.NeedsPLNSerialization(tt.obj)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestParsePartPropsNestedDict tests that nested dictionaries are properly
// deserialized and can be accessed with dot notation
func TestParsePartPropsNestedDict(t *testing.T) {
	h := &parsleyHandler{
		server: testServer(),
	}

	// Simulate what JavaScript sends: a nested object JSON-stringified
	jsonValue := url.QueryEscape(`{"Firstname":"John","Surname":"Smith"}`)
	req := httptest.NewRequest("GET", "/?_view=test&person="+jsonValue, nil)
	env := evaluator.NewEnvironment()

	props, err := h.parsePartProps(req, env)
	if err != nil {
		t.Fatalf("parsePartProps failed: %v", err)
	}

	// Check that person is a dictionary
	personExpr := props.Pairs["person"]
	if personExpr == nil {
		t.Fatal("person prop not found")
	}

	personObj := evaluator.Eval(personExpr, env)
	t.Logf("personObj type: %T", personObj)

	personDict, ok := personObj.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary for person, got %T: %v", personObj, personObj)
	}

	// Check that we can access nested properties
	firstnameExpr := personDict.Pairs["Firstname"]
	if firstnameExpr == nil {
		t.Fatal("Firstname not found in person dict")
	}

	firstnameObj := evaluator.Eval(firstnameExpr, personDict.Env)
	if strObj, ok := firstnameObj.(*evaluator.String); ok {
		if strObj.Value != "John" {
			t.Errorf("expected Firstname='John', got %q", strObj.Value)
		}
	} else {
		t.Errorf("expected String for Firstname, got %T", firstnameObj)
	}
}

// TestParsePartPropsNestedDictDotNotation tests full evaluation with dot notation
func TestParsePartPropsNestedDictDotNotation(t *testing.T) {
	h := &parsleyHandler{
		server: testServer(),
	}

	// Simulate what JavaScript sends
	jsonValue := url.QueryEscape(`{"Firstname":"John","Surname":"Smith"}`)
	req := httptest.NewRequest("GET", "/?_view=test&person="+jsonValue, nil)
	env := evaluator.NewEnvironment()

	props, err := h.parsePartProps(req, env)
	if err != nil {
		t.Fatalf("parsePartProps failed: %v", err)
	}

	// Set props in environment so we can use it in eval
	env.Set("props", props)

	// Try to evaluate props.person.Firstname using Parsley
	code := `props.person.Firstname`
	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error: %v", p.Errors())
	}

	result := evaluator.Eval(program, env)
	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("eval error: %s", errObj.Message)
	}

	if strObj, ok := result.(*evaluator.String); ok {
		if strObj.Value != "John" {
			t.Errorf("expected 'John', got %q", strObj.Value)
		}
	} else {
		t.Errorf("expected String, got %T: %v", result, result)
	}
}
