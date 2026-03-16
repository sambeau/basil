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

// mockImageRegistry implements evaluator.ImageRegistrar for testing
type mockImageRegistry struct {
	transformCalls []mockTransformCall
	infoCalls      []string
}

type mockTransformCall struct {
	SourcePath string
	Opts       map[string]any
}

func newMockImageRegistry() *mockImageRegistry {
	return &mockImageRegistry{}
}

func (m *mockImageRegistry) Transform(sourcePath string, opts map[string]any) (string, error) {
	m.transformCalls = append(m.transformCalls, mockTransformCall{
		SourcePath: sourcePath,
		Opts:       opts,
	})
	// Return a deterministic URL based on the path
	ext := ".jpg"
	if idx := strings.LastIndex(sourcePath, "."); idx != -1 {
		ext = sourcePath[idx:]
	}
	return "/__img/abc123def45678" + ext, nil
}

func (m *mockImageRegistry) Info(sourcePath string) (map[string]any, error) {
	m.infoCalls = append(m.infoCalls, sourcePath)
	return map[string]any{
		"width":       int64(800),
		"height":      int64(600),
		"format":      "jpeg",
		"orientation": "landscape",
	}, nil
}

// evalWithImage sets up environment with image() and imageInfo() functions and image registry
func evalWithImage(t *testing.T, input string, filename string, rootPath string) (evaluator.Object, *mockImageRegistry) {
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

	// Set up image registry
	registry := newMockImageRegistry()
	env.ImageRegistry = registry

	// Inject image and imageInfo functions
	env.SetProtected("image", evaluator.NewImageBuiltin())
	env.SetProtected("imageInfo", evaluator.NewImageInfoBuiltin())

	return evaluator.Eval(program, env), registry
}

func TestImageBasic(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(testFile, []byte("fake-jpeg-data"), 0644); err != nil {
		t.Fatal(err)
	}

	input := `image(@./photo.jpg)`
	result, registry := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	// Should return a string URL
	str, ok := result.(*evaluator.String)
	if !ok {
		if errObj, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("got error: %s", errObj.Message)
		}
		t.Fatalf("expected String, got %T (%v)", result, result)
	}

	// Should be an image URL
	if !strings.HasPrefix(str.Value, "/__img/") {
		t.Errorf("expected URL to start with /__img/, got: %s", str.Value)
	}
	if !strings.HasSuffix(str.Value, ".jpg") {
		t.Errorf("expected URL to end with .jpg, got: %s", str.Value)
	}

	// Should have called Transform
	if len(registry.transformCalls) != 1 {
		t.Errorf("expected 1 Transform call, got %d", len(registry.transformCalls))
	}
}

func TestImageWithOptions(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(testFile, []byte("fake-jpeg-data"), 0644); err != nil {
		t.Fatal(err)
	}

	input := `image(@./photo.jpg, {width: 300})`
	result, registry := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	str, ok := result.(*evaluator.String)
	if !ok {
		if errObj, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("got error: %s", errObj.Message)
		}
		t.Fatalf("expected String, got %T (%v)", result, result)
	}

	if !strings.HasPrefix(str.Value, "/__img/") {
		t.Errorf("expected URL to start with /__img/, got: %s", str.Value)
	}

	// Should have called Transform with options
	if len(registry.transformCalls) != 1 {
		t.Fatalf("expected 1 Transform call, got %d", len(registry.transformCalls))
	}

	call := registry.transformCalls[0]
	width, exists := call.Opts["width"]
	if !exists {
		t.Fatal("expected 'width' in options")
	}
	if width != int64(300) {
		t.Errorf("expected width=300, got %v (%T)", width, width)
	}
}

func TestImageWithStringPath(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(testFile, []byte("fake-jpeg-data"), 0644); err != nil {
		t.Fatal(err)
	}

	input := `image("./photo.jpg")`
	result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	str, ok := result.(*evaluator.String)
	if !ok {
		if errObj, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("got error: %s", errObj.Message)
		}
		t.Fatalf("expected String, got %T", result)
	}

	if !strings.HasPrefix(str.Value, "/__img/") {
		t.Errorf("expected URL to start with /__img/, got: %s", str.Value)
	}
}

func TestImageNotInHandler(t *testing.T) {
	// Test when no image registry is set (not in handler context)
	l := lexer.New(`image(@./photo.jpg)`)
	p := parser.New(l)
	program := p.ParseProgram()

	env := evaluator.NewEnvironment()
	// No ImageRegistry set - simulating non-handler context

	// Inject image but without registry
	env.SetProtected("image", evaluator.NewImageBuiltin())

	result := evaluator.Eval(program, env)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T (%v)", result, result)
	}

	// Should mention it's only available in handlers
	msg := strings.ToLower(errObj.Message)
	if !strings.Contains(msg, "handler") && !strings.Contains(msg, "server") {
		t.Errorf("error should mention handler/server context, got: %s", errObj.Message)
	}
}

func TestImageSecurityCheck(t *testing.T) {
	tmpDir := t.TempDir()
	handlerDir := filepath.Join(tmpDir, "handlers")
	if err := os.MkdirAll(handlerDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file outside handler directory
	secretFile := filepath.Join(tmpDir, "secret.jpg")
	if err := os.WriteFile(secretFile, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	input := `image(@../secret.jpg)`
	result, _ := evalWithImage(t, input, filepath.Join(handlerDir, "test.pars"), handlerDir)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error for path traversal, got %T (%v)", result, result)
	}

	if errObj.Class != "security" {
		t.Errorf("expected security error class, got: %s", errObj.Class)
	}
}

func TestImageWrongArgumentType(t *testing.T) {
	tmpDir := t.TempDir()

	input := `image(123)`
	result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T", result)
	}

	if errObj.Class != errors.ClassType {
		t.Errorf("expected TypeError, got %s", errObj.Class)
	}
}

func TestImageWrongArity(t *testing.T) {
	tmpDir := t.TempDir()

	// No arguments
	t.Run("no args", func(t *testing.T) {
		input := `image()`
		result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

		errObj, ok := result.(*evaluator.Error)
		if !ok {
			t.Fatalf("expected Error, got %T", result)
		}

		if errObj.Class != errors.ClassArity {
			t.Errorf("expected ArityError, got %s", errObj.Class)
		}
	})

	// Too many arguments
	t.Run("too many args", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "a.jpg")
		if err := os.WriteFile(testFile, []byte("fake"), 0644); err != nil {
			t.Fatal(err)
		}

		input := `image(@./a.jpg, {}, "extra")`
		result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

		errObj, ok := result.(*evaluator.Error)
		if !ok {
			t.Fatalf("expected Error, got %T", result)
		}

		if errObj.Class != errors.ClassArity {
			t.Errorf("expected ArityError, got %s", errObj.Class)
		}
	})
}

func TestImageInfoBasic(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(testFile, []byte("fake-jpeg-data"), 0644); err != nil {
		t.Fatal(err)
	}

	input := `imageInfo(@./photo.jpg)`
	result, registry := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		if errObj, isErr := result.(*evaluator.Error); isErr {
			t.Fatalf("got error: %s", errObj.Message)
		}
		t.Fatalf("expected Dictionary, got %T (%v)", result, result)
	}

	// Should have called Info
	if len(registry.infoCalls) != 1 {
		t.Errorf("expected 1 Info call, got %d", len(registry.infoCalls))
	}

	// Check width
	widthExpr, exists := dict.Pairs["width"]
	if !exists {
		t.Fatal("expected 'width' key in result dictionary")
	}
	widthObj := evaluator.Eval(widthExpr, dict.Env)
	widthInt, ok := widthObj.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer for width, got %T (%v)", widthObj, widthObj)
	}
	if widthInt.Value != 800 {
		t.Errorf("expected width=800, got %d", widthInt.Value)
	}

	// Check height
	heightExpr, exists := dict.Pairs["height"]
	if !exists {
		t.Fatal("expected 'height' key in result dictionary")
	}
	heightObj := evaluator.Eval(heightExpr, dict.Env)
	heightInt, ok := heightObj.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer for height, got %T (%v)", heightObj, heightObj)
	}
	if heightInt.Value != 600 {
		t.Errorf("expected height=600, got %d", heightInt.Value)
	}

	// Check format
	formatExpr, exists := dict.Pairs["format"]
	if !exists {
		t.Fatal("expected 'format' key in result dictionary")
	}
	formatObj := evaluator.Eval(formatExpr, dict.Env)
	formatStr, ok := formatObj.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String for format, got %T (%v)", formatObj, formatObj)
	}
	if formatStr.Value != "jpeg" {
		t.Errorf("expected format=jpeg, got %s", formatStr.Value)
	}

	// Check orientation
	orientExpr, exists := dict.Pairs["orientation"]
	if !exists {
		t.Fatal("expected 'orientation' key in result dictionary")
	}
	orientObj := evaluator.Eval(orientExpr, dict.Env)
	orientStr, ok := orientObj.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String for orientation, got %T (%v)", orientObj, orientObj)
	}
	if orientStr.Value != "landscape" {
		t.Errorf("expected orientation=landscape, got %s", orientStr.Value)
	}
}

func TestImageInfoNotInHandler(t *testing.T) {
	l := lexer.New(`imageInfo(@./photo.jpg)`)
	p := parser.New(l)
	program := p.ParseProgram()

	env := evaluator.NewEnvironment()
	// No ImageRegistry set

	env.SetProtected("imageInfo", evaluator.NewImageInfoBuiltin())

	result := evaluator.Eval(program, env)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T (%v)", result, result)
	}

	if errObj.Class != errors.ClassState {
		t.Errorf("expected state error class, got: %s", errObj.Class)
	}

	msg := strings.ToLower(errObj.Message)
	if !strings.Contains(msg, "handler") && !strings.Contains(msg, "server") {
		t.Errorf("error should mention handler/server context, got: %s", errObj.Message)
	}
}

func TestImageInfoWrongArity(t *testing.T) {
	tmpDir := t.TempDir()

	// No arguments
	t.Run("no args", func(t *testing.T) {
		input := `imageInfo()`
		result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

		errObj, ok := result.(*evaluator.Error)
		if !ok {
			t.Fatalf("expected Error, got %T", result)
		}

		if errObj.Class != errors.ClassArity {
			t.Errorf("expected ArityError, got %s", errObj.Class)
		}
	})

	// Too many arguments
	t.Run("too many args", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "a.jpg")
		if err := os.WriteFile(testFile, []byte("fake"), 0644); err != nil {
			t.Fatal(err)
		}

		input := `imageInfo(@./a.jpg, "extra")`
		result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

		errObj, ok := result.(*evaluator.Error)
		if !ok {
			t.Fatalf("expected Error, got %T", result)
		}

		if errObj.Class != errors.ClassArity {
			t.Errorf("expected ArityError, got %s", errObj.Class)
		}
	})
}
