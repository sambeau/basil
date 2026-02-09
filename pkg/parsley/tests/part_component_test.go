package tests

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestPartComponentBasic(t *testing.T) {
	input := `<Part src={@./test_fixtures/parts/counter.part}/>`

	l := lexer.NewWithFilename(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.Filename = "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars"
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	result := evaluator.Eval(program, env)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	// Should contain wrapper div
	if !strings.Contains(str.Value, "<div data-part-src=") {
		t.Errorf("expected wrapper div with data-part-src, got: %s", str.Value)
	}

	// Should contain default view name
	if !strings.Contains(str.Value, `data-part-view="default"`) {
		t.Errorf("expected data-part-view='default', got: %s", str.Value)
	}

	// Should contain props (even if empty)
	if !strings.Contains(str.Value, "data-part-props=") {
		t.Errorf("expected data-part-props attribute, got: %s", str.Value)
	}

	// Should mark environment as containing Parts
	if !env.ContainsParts {
		t.Errorf("expected env.ContainsParts to be true")
	}
}

func TestPartComponentWithView(t *testing.T) {
	input := `<Part src={@./test_fixtures/parts/counter.part} view="increment"/>`

	l := lexer.NewWithFilename(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.Filename = "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars"
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	result := evaluator.Eval(program, env)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	// Should use increment view
	if !strings.Contains(str.Value, `data-part-view="increment"`) {
		t.Errorf("expected data-part-view='increment', got: %s", str.Value)
	}
}

func TestPartComponentWithProps(t *testing.T) {
	input := `<Part src={@./test_fixtures/parts/counter.part} count={10}/>`

	l := lexer.NewWithFilename(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.Filename = "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars"
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	result := evaluator.Eval(program, env)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	// Should contain count value in rendered HTML
	if !strings.Contains(str.Value, "10") {
		t.Errorf("expected HTML to contain count value '10', got: %s", str.Value)
	}

	// Extract and verify props JSON
	propsStart := strings.Index(str.Value, "data-part-props='") + len("data-part-props='")
	propsEnd := strings.Index(str.Value[propsStart:], "'") + propsStart
	propsJSON := str.Value[propsStart:propsEnd]

	// Unescape HTML entities
	propsJSON = strings.ReplaceAll(propsJSON, "&quot;", "\"")
	propsJSON = strings.ReplaceAll(propsJSON, "&#39;", "'")
	propsJSON = strings.ReplaceAll(propsJSON, "&amp;", "&")
	propsJSON = strings.ReplaceAll(propsJSON, "&lt;", "<")
	propsJSON = strings.ReplaceAll(propsJSON, "&gt;", ">")

	var props map[string]any
	if err := json.Unmarshal([]byte(propsJSON), &props); err != nil {
		t.Fatalf("failed to parse props JSON: %v (JSON: %s)", err, propsJSON)
	}

	if count, ok := props["count"].(float64); !ok || count != 10 {
		t.Errorf("expected props.count to be 10, got: %v", props["count"])
	}
}

func TestPartComponentWithRefreshAndLoad(t *testing.T) {
	// Test immediate load (part-load) - fetches view right away
	input := `<Part src={@./test_fixtures/parts/counter.part} part-refresh={5000} part-load="loaded"/>`

	l := lexer.NewWithFilename(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.Filename = "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars"
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	result := evaluator.Eval(program, env)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	html := str.Value

	if !strings.Contains(html, `data-part-refresh="5000"`) {
		t.Errorf("expected data-part-refresh attribute, got: %s", html)
	}

	if !strings.Contains(html, `data-part-load="loaded"`) {
		t.Errorf("expected data-part-load attribute, got: %s", html)
	}

	// Ensure config attributes are not passed to view props
	propsStart := strings.Index(html, "data-part-props='") + len("data-part-props='")
	propsEnd := strings.Index(html[propsStart:], "'") + propsStart
	propsJSON := html[propsStart:propsEnd]
	propsJSON = strings.ReplaceAll(propsJSON, "&quot;", "\"")
	propsJSON = strings.ReplaceAll(propsJSON, "&#39;", "'")
	propsJSON = strings.ReplaceAll(propsJSON, "&amp;", "&")
	propsJSON = strings.ReplaceAll(propsJSON, "&lt;", "<")
	propsJSON = strings.ReplaceAll(propsJSON, "&gt;", ">")

	var props map[string]any
	if err := json.Unmarshal([]byte(propsJSON), &props); err != nil {
		t.Fatalf("failed to parse props JSON: %v (JSON: %s)", err, propsJSON)
	}

	if _, hasRefresh := props["part-refresh"]; hasRefresh {
		t.Errorf("expected part-refresh not to be passed to view props")
	}
	if _, hasLoad := props["part-load"]; hasLoad {
		t.Errorf("expected part-load not to be passed to view props")
	}
}

func TestPartComponentWithLazy(t *testing.T) {
	// Test lazy loading (part-lazy) - fetches view when scrolled into viewport
	input := `<Part src={@./test_fixtures/parts/counter.part} part-lazy="loaded" part-lazy-threshold={200}/>`

	l := lexer.NewWithFilename(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.Filename = "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars"
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	result := evaluator.Eval(program, env)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	html := str.Value

	if !strings.Contains(html, `data-part-lazy="loaded"`) {
		t.Errorf("expected data-part-lazy attribute, got: %s", html)
	}

	if !strings.Contains(html, `data-part-lazy-threshold="200"`) {
		t.Errorf("expected data-part-lazy-threshold attribute, got: %s", html)
	}

	// Ensure config attributes are not passed to view props
	propsStart := strings.Index(html, "data-part-props='") + len("data-part-props='")
	propsEnd := strings.Index(html[propsStart:], "'") + propsStart
	propsJSON := html[propsStart:propsEnd]
	propsJSON = strings.ReplaceAll(propsJSON, "&quot;", "\"")
	propsJSON = strings.ReplaceAll(propsJSON, "&#39;", "'")
	propsJSON = strings.ReplaceAll(propsJSON, "&amp;", "&")
	propsJSON = strings.ReplaceAll(propsJSON, "&lt;", "<")
	propsJSON = strings.ReplaceAll(propsJSON, "&gt;", ">")

	var props map[string]any
	if err := json.Unmarshal([]byte(propsJSON), &props); err != nil {
		t.Fatalf("failed to parse props JSON: %v (JSON: %s)", err, propsJSON)
	}

	if _, hasLazy := props["part-lazy"]; hasLazy {
		t.Errorf("expected part-lazy not to be passed to view props")
	}
	if _, hasThreshold := props["part-lazy-threshold"]; hasThreshold {
		t.Errorf("expected part-lazy-threshold not to be passed to view props")
	}
}

func TestPartComponentMissingSrc(t *testing.T) {
	input := `<Part/>`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	result := evaluator.Eval(program, env)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Fatalf("expected error for missing src, got: %s", result.Inspect())
	}

	err := result.(*evaluator.Error)
	if err.Code != "PART-0002" {
		t.Errorf("expected error code PART-0002, got: %s", err.Code)
	}
}

func TestPartComponentNonPartFile(t *testing.T) {
	input := `<Part src="./test.pars"/>`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	result := evaluator.Eval(program, env)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Fatalf("expected error for non-.part file, got: %s", result.Inspect())
	}

	err := result.(*evaluator.Error)
	if err.Code != "PART-0004" {
		t.Errorf("expected error code PART-0004, got: %s", err.Code)
	}
}

func TestPartComponentMissingView(t *testing.T) {
	input := `<Part src={@./test_fixtures/parts/counter.part} view="nonexistent"/>`

	l := lexer.NewWithFilename(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.Filename = "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars"
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	result := evaluator.Eval(program, env)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Fatalf("expected error for missing view, got: %s", result.Inspect())
	}

	err := result.(*evaluator.Error)
	if err.Code != "PART-0007" {
		t.Errorf("expected error code PART-0007, got: %s", err.Code)
	}
}

func TestPartComponentWithId(t *testing.T) {
	// Test that the id attribute is passed through to the wrapper div
	input := `<Part src={@./test_fixtures/parts/counter.part} id="search-results"/>`

	l := lexer.NewWithFilename(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.Filename = "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars"
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	result := evaluator.Eval(program, env)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	html := str.Value

	// Should have id as first attribute after opening div
	if !strings.Contains(html, `<div id="search-results"`) {
		t.Errorf("expected wrapper div to have id attribute, got: %s", html)
	}

	// Ensure id is not passed to view props
	propsStart := strings.Index(html, "data-part-props='") + len("data-part-props='")
	propsEnd := strings.Index(html[propsStart:], "'") + propsStart
	propsJSON := html[propsStart:propsEnd]
	propsJSON = strings.ReplaceAll(propsJSON, "&quot;", "\"")
	propsJSON = strings.ReplaceAll(propsJSON, "&#39;", "'")
	propsJSON = strings.ReplaceAll(propsJSON, "&amp;", "&")
	propsJSON = strings.ReplaceAll(propsJSON, "&lt;", "<")
	propsJSON = strings.ReplaceAll(propsJSON, "&gt;", ">")

	var props map[string]any
	if err := json.Unmarshal([]byte(propsJSON), &props); err != nil {
		t.Fatalf("failed to parse props JSON: %v (JSON: %s)", err, propsJSON)
	}

	if _, hasId := props["id"]; hasId {
		t.Errorf("expected id not to be passed to view props")
	}
}

func TestPartComponentWithIdAndProps(t *testing.T) {
	// Test that id works together with other props
	input := `<Part src={@./test_fixtures/parts/counter.part} id="counter-1" count={5}/>`

	l := lexer.NewWithFilename(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.Filename = "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars"
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	result := evaluator.Eval(program, env)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T", result)
	}

	html := str.Value

	// Should have id attribute
	if !strings.Contains(html, `id="counter-1"`) {
		t.Errorf("expected wrapper div to have id attribute, got: %s", html)
	}

	// Should contain count prop (HTML-escaped in data attribute)
	if !strings.Contains(html, `&quot;count&quot;:5`) {
		t.Errorf("expected props to contain count, got: %s", html)
	}
}
