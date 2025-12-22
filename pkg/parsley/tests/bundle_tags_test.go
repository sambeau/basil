package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// mockAssetBundle implements evaluator.AssetBundler for testing
type mockAssetBundle struct {
	cssURL string
	jsURL  string
}

func (m *mockAssetBundle) CSSUrl() string {
	return m.cssURL
}

func (m *mockAssetBundle) JSUrl() string {
	return m.jsURL
}

func evalWithAssetBundle(t *testing.T, input string, bundle *mockAssetBundle) evaluator.Object {
	t.Helper()

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.AssetBundle = bundle

	return evaluator.Eval(program, env)
}

func TestCssTag_EmitsLink(t *testing.T) {
	bundle := &mockAssetBundle{
		cssURL: "/__site.css?v=abc12345",
	}

	result := evalWithAssetBundle(t, "<CSS/>", bundle)

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	expected := `<link rel="stylesheet" href="/__site.css?v=abc12345">`
	if str.Value != expected {
		t.Errorf("Expected %q, got %q", expected, str.Value)
	}
}

func TestScriptTag_EmitsScript(t *testing.T) {
	bundle := &mockAssetBundle{
		jsURL: "/__site.js?v=def67890",
	}

	result := evalWithAssetBundle(t, "<Javascript/>", bundle)

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	expected := `<script src="/__site.js?v=def67890"></script>`
	if str.Value != expected {
		t.Errorf("Expected %q, got %q", expected, str.Value)
	}
}

func TestCssTag_NoBundle(t *testing.T) {
	// No bundle set
	l := lexer.New("<CSS/>")
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	// env.AssetBundle is nil

	result := evaluator.Eval(program, env)

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	if str.Value != "" {
		t.Errorf("Expected empty string when no bundle, got %q", str.Value)
	}
}

func TestCssTag_EmptyBundle(t *testing.T) {
	bundle := &mockAssetBundle{
		cssURL: "", // No CSS files
	}

	result := evalWithAssetBundle(t, "<CSS/>", bundle)

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	if str.Value != "" {
		t.Errorf("Expected empty string for empty bundle, got %q", str.Value)
	}
}

func TestScriptTag_EmptyBundle(t *testing.T) {
	bundle := &mockAssetBundle{
		jsURL: "", // No JS files
	}

	result := evalWithAssetBundle(t, "<Javascript/>", bundle)

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	if str.Value != "" {
		t.Errorf("Expected empty string for empty bundle, got %q", str.Value)
	}
}

func TestCssAndScriptTags_InTemplate(t *testing.T) {
	bundle := &mockAssetBundle{
		cssURL: "/__site.css?v=abc123",
		jsURL:  "/__site.js?v=def456",
	}

	input := `
<html>
<head>
  <CSS/>
</head>
<body>
  <h1>"Test"</h1>
  <Javascript/>
</body>
</html>
`

	result := evalWithAssetBundle(t, input, bundle)

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	// Check that both tags are rendered
	if !contains(str.Value, `<link rel="stylesheet" href="/__site.css?v=abc123">`) {
		t.Error("Output should contain CSS link tag")
	}
	if !contains(str.Value, `<script src="/__site.js?v=def456"></script>`) {
		t.Error("Output should contain Javascript tag")
	}
}

func TestBasilJSTag_EmitsScript(t *testing.T) {
	l := lexer.New("<BasilJS/>")
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.BasilJSURL = "/__/js/basil.abc1234.js"

	result := evaluator.Eval(program, env)

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	expected := `<script src="/__/js/basil.abc1234.js"></script>`
	if str.Value != expected {
		t.Errorf("Expected %q, got %q", expected, str.Value)
	}
}

func TestBasilJSTag_NoURL(t *testing.T) {
	l := lexer.New("<BasilJS/>")
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	// env.BasilJSURL is empty string

	result := evaluator.Eval(program, env)

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	if str.Value != "" {
		t.Errorf("Expected empty string when no BasilJSURL, got %q", str.Value)
	}
}
