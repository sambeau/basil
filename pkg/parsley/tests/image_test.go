package tests

import (
	"fmt"
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
	// configurable source dimensions for srcset tests
	srcWidth  int
	srcHeight int
}

type mockTransformCall struct {
	SourcePath string
	Opts       map[string]any
}

func newMockImageRegistry() *mockImageRegistry {
	return &mockImageRegistry{
		srcWidth:  800,
		srcHeight: 600,
	}
}

func (m *mockImageRegistry) Transform(sourcePath string, opts map[string]any) (string, error) {
	m.transformCalls = append(m.transformCalls, mockTransformCall{
		SourcePath: sourcePath,
		Opts:       opts,
	})
	// Return a deterministic URL based on the path and width
	ext := ".jpg"
	if idx := strings.LastIndex(sourcePath, "."); idx != -1 {
		ext = sourcePath[idx:]
	}
	// Include width in hash for srcset tests
	width := 0
	if w, ok := opts["width"]; ok {
		switch v := w.(type) {
		case int:
			width = v
		case int64:
			width = int(v)
		case float64:
			width = int(v)
		}
	}
	return fmt.Sprintf("/__img/abc%d%s", width, ext), nil
}

func (m *mockImageRegistry) Info(sourcePath string) (map[string]any, error) {
	m.infoCalls = append(m.infoCalls, sourcePath)
	return map[string]any{
		"width":       int64(m.srcWidth),
		"height":      int64(m.srcHeight),
		"format":      "jpeg",
		"orientation": "landscape",
	}, nil
}

func (m *mockImageRegistry) BlurPlaceholder(sourcePath string) (string, error) {
	// Return a deterministic data URI for testing
	return "data:image/jpeg;base64,/9j/4AAQSkZJRg==", nil
}

// evalWithImage sets up environment with image() and imageInfo() functions and image registry
func evalWithImage(t *testing.T, input, filename, rootPath string) (evaluator.Object, *mockImageRegistry) {
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

	// Inject image, imageInfo, imageBlur, and imageSrcset functions
	env.SetProtected("image", evaluator.NewImageBuiltin())
	env.SetProtected("imageInfo", evaluator.NewImageInfoBuiltin())
	env.SetProtected("imageBlur", evaluator.NewImageBlurBuiltin())
	env.SetProtected("imageSrcset", evaluator.NewImageSrcsetBuiltin())

	return evaluator.Eval(program, env), registry
}

func TestImageBasic(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(testFile, []byte("fake-jpeg-data"), 0o644); err != nil {
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
	if err := os.WriteFile(testFile, []byte("fake-jpeg-data"), 0o644); err != nil {
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

func TestImageWithSmartCropAndFocal(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(testFile, []byte("fake-jpeg-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := `image(@./photo.jpg, {width: 400, height: 300, crop: "smart", focal: {x: 0.5, y: 0.5}})`
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

	// Should have called Transform with options including focal
	if len(registry.transformCalls) != 1 {
		t.Fatalf("expected 1 Transform call, got %d", len(registry.transformCalls))
	}

	call := registry.transformCalls[0]

	// Check crop is "smart"
	crop, exists := call.Opts["crop"]
	if !exists {
		t.Fatal("expected 'crop' in options")
	}
	if crop != "smart" {
		t.Errorf("expected crop='smart', got %v", crop)
	}

	// Check focal is a map with x and y
	focal, exists := call.Opts["focal"]
	if !exists {
		t.Fatal("expected 'focal' in options")
	}
	focalMap, ok := focal.(map[string]any)
	if !ok {
		t.Fatalf("expected focal to be map[string]any, got %T", focal)
	}
	if focalMap["x"] != 0.5 {
		t.Errorf("expected focal.x=0.5, got %v", focalMap["x"])
	}
	if focalMap["y"] != 0.5 {
		t.Errorf("expected focal.y=0.5, got %v", focalMap["y"])
	}
}

func TestImageWithStringPath(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(testFile, []byte("fake-jpeg-data"), 0o644); err != nil {
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
	if err := os.MkdirAll(handlerDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a file outside handler directory
	secretFile := filepath.Join(tmpDir, "secret.jpg")
	if err := os.WriteFile(secretFile, []byte("secret"), 0o644); err != nil {
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
		if err := os.WriteFile(testFile, []byte("fake"), 0o644); err != nil {
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
	if err := os.WriteFile(testFile, []byte("fake-jpeg-data"), 0o644); err != nil {
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
		if err := os.WriteFile(testFile, []byte("fake"), 0o644); err != nil {
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

// === imageBlur() tests ===

func TestImageBlurBasic(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test image file
	testFile := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(testFile, []byte("fake image data"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := `imageBlur(@./photo.jpg)`
	result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	strObj, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T (%v)", result, result)
	}

	// Should return a data URI
	if !strings.HasPrefix(strObj.Value, "data:image/jpeg;base64,") {
		t.Errorf("expected data URI, got: %s", strObj.Value)
	}
}

func TestImageBlurNotInHandler(t *testing.T) {
	l := lexer.New(`imageBlur(@./photo.jpg)`)
	p := parser.New(l)
	program := p.ParseProgram()

	env := evaluator.NewEnvironment()
	// No ImageRegistry set

	env.SetProtected("imageBlur", evaluator.NewImageBlurBuiltin())

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

func TestImageBlurWrongArity(t *testing.T) {
	tmpDir := t.TempDir()

	// No arguments
	t.Run("no args", func(t *testing.T) {
		input := `imageBlur()`
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
		if err := os.WriteFile(testFile, []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}

		input := `imageBlur(@./a.jpg, "extra")`
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

func TestImageBlurSecurityPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()

	// Try to access a path outside the root
	input := `imageBlur(@./../../../etc/passwd)`
	result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error for path traversal, got %T (%v)", result, result)
	}

	if errObj.Class != errors.ClassSecurity {
		t.Errorf("expected security error class, got: %s", errObj.Class)
	}
}

// === imageSrcset() tests ===

func TestImageSrcsetBasic(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test image file
	testFile := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(testFile, []byte("fake image data"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := `imageSrcset(@./photo.jpg, {}, [200, 400, 600])`
	result, registry := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T (%v)", result, result)
	}

	// Check required keys exist
	requiredKeys := []string{"src", "srcset", "width", "height"}
	for _, key := range requiredKeys {
		if _, exists := dict.Pairs[key]; !exists {
			t.Errorf("expected key %q in result dictionary", key)
		}
	}

	// Check srcset contains width descriptors
	srcsetExpr := dict.Pairs["srcset"]
	srcsetObj := evaluator.Eval(srcsetExpr, dict.Env)
	srcsetStr, ok := srcsetObj.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String for srcset, got %T", srcsetObj)
	}

	// Should contain width descriptors
	if !strings.Contains(srcsetStr.Value, "w") {
		t.Errorf("srcset should contain width descriptors, got: %s", srcsetStr.Value)
	}

	// Should have called Transform 3 times (one for each width)
	if len(registry.transformCalls) != 3 {
		t.Errorf("expected 3 Transform calls, got %d", len(registry.transformCalls))
	}
}

func TestImageSrcsetDensityMode(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "icon.jpg")
	if err := os.WriteFile(testFile, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := `imageSrcset(@./icon.jpg, {width: 64}, [1, 2, 3], "x")`
	result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T (%v)", result, result)
	}

	// Check srcset contains density descriptors
	srcsetExpr := dict.Pairs["srcset"]
	srcsetObj := evaluator.Eval(srcsetExpr, dict.Env)
	srcsetStr, ok := srcsetObj.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String for srcset, got %T", srcsetObj)
	}

	// Should contain density descriptors (1x, 2x, 3x)
	if !strings.Contains(srcsetStr.Value, "1x") || !strings.Contains(srcsetStr.Value, "2x") {
		t.Errorf("srcset should contain density descriptors, got: %s", srcsetStr.Value)
	}
}

func TestImageSrcsetDensityModeRequiresWidth(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(testFile, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Density mode without style.width should error
	input := `imageSrcset(@./photo.jpg, {}, [1, 2, 3], "x")`
	result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T (%v)", result, result)
	}

	if !strings.Contains(errObj.Message, "width") {
		t.Errorf("error should mention width requirement, got: %s", errObj.Message)
	}
}

func TestImageSrcsetEmptyWidths(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(testFile, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := `imageSrcset(@./photo.jpg, {}, [])`
	result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error for empty widths, got %T (%v)", result, result)
	}

	if !strings.Contains(errObj.Message, "empty") {
		t.Errorf("error should mention empty array, got: %s", errObj.Message)
	}
}

func TestImageSrcsetNonPositiveWidth(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(testFile, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := `imageSrcset(@./photo.jpg, {}, [400, -100, 800])`
	result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error for non-positive width, got %T (%v)", result, result)
	}

	if !strings.Contains(errObj.Message, "positive") {
		t.Errorf("error should mention positive requirement, got: %s", errObj.Message)
	}
}

func TestImageSrcsetInvalid4thArg(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(testFile, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := `imageSrcset(@./photo.jpg, {width: 100}, [1, 2], "invalid")`
	result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error for invalid 4th arg, got %T (%v)", result, result)
	}

	if !strings.Contains(errObj.Message, "x") {
		t.Errorf("error should mention 'x', got: %s", errObj.Message)
	}
}

func TestImageSrcsetNotInHandler(t *testing.T) {
	l := lexer.New(`imageSrcset(@./photo.jpg, {}, [400, 800])`)
	p := parser.New(l)
	program := p.ParseProgram()

	env := evaluator.NewEnvironment()
	// No ImageRegistry set

	env.SetProtected("imageSrcset", evaluator.NewImageSrcsetBuiltin())

	result := evaluator.Eval(program, env)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T (%v)", result, result)
	}

	if errObj.Class != errors.ClassState {
		t.Errorf("expected state error class, got: %s", errObj.Class)
	}
}

func TestImageSrcsetWrongArity(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("too few args", func(t *testing.T) {
		input := `imageSrcset(@./photo.jpg, {})`
		result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

		errObj, ok := result.(*evaluator.Error)
		if !ok {
			t.Fatalf("expected Error, got %T", result)
		}

		if errObj.Class != errors.ClassArity {
			t.Errorf("expected ArityError, got %s", errObj.Class)
		}
	})

	t.Run("too many args", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "a.jpg")
		if err := os.WriteFile(testFile, []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}

		input := `imageSrcset(@./a.jpg, {}, [400], "x", "extra")`
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

func TestImageSrcsetSecurityPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()

	input := `imageSrcset(@./../../../etc/passwd, {}, [400, 800])`
	result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error for path traversal, got %T (%v)", result, result)
	}

	if errObj.Class != errors.ClassSecurity {
		t.Errorf("expected security error class, got: %s", errObj.Class)
	}
}

func TestImageSrcsetResultDimensions(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(testFile, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := `imageSrcset(@./photo.jpg, {}, [400, 800])`
	result, _ := evalWithImage(t, input, filepath.Join(tmpDir, "test.pars"), tmpDir)

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T (%v)", result, result)
	}

	// Check width is an integer
	widthExpr := dict.Pairs["width"]
	widthObj := evaluator.Eval(widthExpr, dict.Env)
	_, ok = widthObj.(*evaluator.Integer)
	if !ok {
		t.Errorf("expected Integer for width, got %T", widthObj)
	}

	// Check height is an integer
	heightExpr := dict.Pairs["height"]
	heightObj := evaluator.Eval(heightExpr, dict.Env)
	_, ok = heightObj.(*evaluator.Integer)
	if !ok {
		t.Errorf("expected Integer for height, got %T", heightObj)
	}
}

func TestImageSrcsetDensityModeClampedDuplicates(t *testing.T) {
	// Regression test: when source is small (100px wide) and baseWidth=64,
	// scales [1,2,3] produce pixel widths [64, 128→100, 192→100].
	// After clamping, 2x and 3x both map to 100px. The srcset must still
	// contain all three descriptors (1x, 2x, 3x) with correct labels.
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "small.jpg")
	if err := os.WriteFile(testFile, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	registry := newMockImageRegistry()
	registry.srcWidth = 100
	registry.srcHeight = 75

	input := `imageSrcset(@./small.jpg, {width: 64}, [1, 2, 3], "x")`

	l := lexer.NewWithFilename(input, filepath.Join(tmpDir, "test.pars"))
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.Filename = filepath.Join(tmpDir, "test.pars")
	env.RootPath = tmpDir
	env.ImageRegistry = registry
	env.SetProtected("imageSrcset", evaluator.NewImageSrcsetBuiltin())

	result := evaluator.Eval(program, env)

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T (%v)", result, result)
	}

	srcsetExpr := dict.Pairs["srcset"]
	srcsetObj := evaluator.Eval(srcsetExpr, dict.Env)
	srcsetStr, ok := srcsetObj.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String for srcset, got %T", srcsetObj)
	}

	// Must contain all three density descriptors
	if !strings.Contains(srcsetStr.Value, "1x") {
		t.Errorf("srcset missing 1x descriptor, got: %s", srcsetStr.Value)
	}
	if !strings.Contains(srcsetStr.Value, "2x") {
		t.Errorf("srcset missing 2x descriptor, got: %s", srcsetStr.Value)
	}
	if !strings.Contains(srcsetStr.Value, "3x") {
		t.Errorf("srcset missing 3x descriptor, got: %s", srcsetStr.Value)
	}

	// Only 2 unique widths (64 and 100), so only 2 Transform calls
	if len(registry.transformCalls) != 2 {
		t.Errorf("expected 2 Transform calls (deduped), got %d", len(registry.transformCalls))
	}
}
