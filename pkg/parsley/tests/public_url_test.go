package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// mockAssetRegistry implements evaluator.AssetRegistrar for testing
type mockAssetRegistry struct {
	registered map[string]string // filepath -> url
}

func newMockAssetRegistry() *mockAssetRegistry {
	return &mockAssetRegistry{
		registered: make(map[string]string),
	}
}

func (m *mockAssetRegistry) Register(filepath string) (string, error) {
	// Simple mock: return a deterministic URL based on the path
	hash := "abc123def456" // Mock hash
	ext := ""
	if idx := strings.LastIndex(filepath, "."); idx != -1 {
		ext = filepath[idx:]
	}
	url := "/__p/" + hash + ext
	m.registered[filepath] = url
	return url, nil
}

// evalWithPublicUrl sets up environment with publicUrl() function and asset registry
func evalWithPublicUrl(t *testing.T, input string, filename string, rootPath string) (evaluator.Object, *mockAssetRegistry) {
	t.Helper()

	l := lexer.NewWithFilename(input, filename)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.Filename = filename
	env.RootPath = rootPath

	// Set up asset registry
	registry := newMockAssetRegistry()
	env.AssetRegistry = registry

	// Inject publicUrl function
	env.SetProtected("publicUrl", evaluator.NewPublicURLBuiltin())

	return evaluator.Eval(program, env), registry
}

func TestPublicUrlBasic(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "icon.svg")
	if err := os.WriteFile(testFile, []byte("<svg/>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test basic publicUrl with path literal
	input := `publicUrl(@./icon.svg)`
	result, registry := evalWithPublicUrl(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	// Should return a string URL
	str, ok := result.(*evaluator.String)
	if !ok {
		if errObj, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("got error: %s", errObj.Message)
		}
		t.Fatalf("expected String, got %T (%v)", result, result)
	}

	// Should be a public URL
	if !strings.HasPrefix(str.Value, "/__p/") {
		t.Errorf("expected URL to start with /__p/, got: %s", str.Value)
	}
	if !strings.HasSuffix(str.Value, ".svg") {
		t.Errorf("expected URL to end with .svg, got: %s", str.Value)
	}

	// Should have registered the file
	if len(registry.registered) != 1 {
		t.Errorf("expected 1 registered file, got %d", len(registry.registered))
	}
}

func TestPublicUrlWithStringPath(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "style.css")
	if err := os.WriteFile(testFile, []byte("body{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test with string path
	input := `publicUrl("./style.css")`
	result, _ := evalWithPublicUrl(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	str, ok := result.(*evaluator.String)
	if !ok {
		if errObj, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("got error: %s", errObj.Message)
		}
		t.Fatalf("expected String, got %T", result)
	}

	if !strings.HasPrefix(str.Value, "/__p/") {
		t.Errorf("expected URL to start with /__p/, got: %s", str.Value)
	}
}

func TestPublicUrlNotInHandler(t *testing.T) {
	// Test when no asset registry is set (not in handler context)
	l := lexer.New(`publicUrl(@./icon.svg)`)
	p := parser.New(l)
	program := p.ParseProgram()

	env := evaluator.NewEnvironment()
	// No AssetRegistry set - simulating non-handler context

	// Inject publicUrl but without registry
	env.SetProtected("publicUrl", evaluator.NewPublicURLBuiltin())

	result := evaluator.Eval(program, env)

	// Should return an error
	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T (%v)", result, result)
	}

	// Should mention it's only available in handlers
	if !strings.Contains(strings.ToLower(errObj.Message), "handler") {
		t.Errorf("error should mention handler context, got: %s", errObj.Message)
	}
}

func TestPublicUrlSecurityCheck(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	handlerDir := filepath.Join(tmpDir, "handlers")
	if err := os.MkdirAll(handlerDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file outside handler directory
	secretFile := filepath.Join(tmpDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test trying to access file outside handler root
	input := `publicUrl(@../secret.txt)`
	result, _ := evalWithPublicUrl(t, input, filepath.Join(handlerDir, "test.pars"), handlerDir)

	// Should return an error for security violation
	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error for path traversal, got %T (%v)", result, result)
	}

	// Error should indicate path is outside allowed directory
	if errObj.Class != "security" {
		t.Errorf("expected security error class, got: %s", errObj.Class)
	}
}

func TestPublicUrlWrongArgumentType(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with wrong argument type
	input := `publicUrl(123)`
	result, _ := evalWithPublicUrl(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	// Should return a type error
	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T", result)
	}

	if errObj.Class != errors.ClassType {
		t.Errorf("expected TypeError, got %s", errObj.Class)
	}
}

func TestPublicUrlWrongArity(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with wrong number of arguments
	input := `publicUrl(@./a.svg, @./b.svg)`
	result, _ := evalWithPublicUrl(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	// Should return an arity error
	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T", result)
	}

	if errObj.Class != errors.ClassArity {
		t.Errorf("expected ArityError, got %s", errObj.Class)
	}
}

func TestPublicUrlInSubdirectory(t *testing.T) {
	// Create nested directory structure
	tmpDir := t.TempDir()
	handlersDir := filepath.Join(tmpDir, "handlers")
	componentsDir := filepath.Join(handlersDir, "components")
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create an asset in the components directory
	iconFile := filepath.Join(componentsDir, "button-icon.svg")
	if err := os.WriteFile(iconFile, []byte("<svg/>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test from a file in the components directory
	input := `publicUrl(@./button-icon.svg)`
	result, registry := evalWithPublicUrl(t, input, filepath.Join(componentsDir, "Button.pars"), handlersDir)

	str, ok := result.(*evaluator.String)
	if !ok {
		if errObj, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("got error: %s", errObj.Message)
		}
		t.Fatalf("expected String, got %T", result)
	}

	if !strings.HasPrefix(str.Value, "/__p/") {
		t.Errorf("expected URL to start with /__p/, got: %s", str.Value)
	}

	// Verify the full path was registered (including components dir)
	if len(registry.registered) != 1 {
		t.Errorf("expected 1 registered file, got %d", len(registry.registered))
	}

	// The registered path should be the absolute path to button-icon.svg
	for path := range registry.registered {
		if !strings.HasSuffix(path, "button-icon.svg") {
			t.Errorf("expected registered path to end with button-icon.svg, got: %s", path)
		}
	}
}
