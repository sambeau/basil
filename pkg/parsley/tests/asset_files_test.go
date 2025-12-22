package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// evalWithFilesContext creates an environment suitable for testing fileList() and asset()
func evalWithFilesContext(t *testing.T, input string, cwd string, rootPath string, publicDir string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.Filename = filepath.Join(cwd, "test.pars")
	env.RootPath = rootPath
	env.Security = &evaluator.SecurityPolicy{}

	if publicDir != "" {
		basilDict := &evaluator.Dictionary{
			Pairs: map[string]ast.Expression{
				"public_dir": &ast.StringLiteral{Value: publicDir},
			},
			Env: env,
		}
		env.SetProtected("basil", basilDict)
	}

	return evaluator.Eval(program, env)
}

// TestAssetAcceptsFileDictionaries tests that asset() works with file dictionaries from fileList()
func TestAssetAcceptsFileDictionaries(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	publicDir := filepath.Join(tmpDir, "public")
	imagesDir := filepath.Join(publicDir, "images")

	os.MkdirAll(imagesDir, 0755)

	// Create test image files
	os.WriteFile(filepath.Join(imagesDir, "a.png"), []byte("img"), 0644)
	os.WriteFile(filepath.Join(imagesDir, "b.png"), []byte("img"), 0644)

	// Test: fileList() returns file dicts, asset() transforms them to web URLs
	// Using absolute paths
	input := `
		let images = fileList(@` + imagesDir + `/*.png)
		for (img in images) { asset(img) }
	`

	result := evalWithFilesContext(t, input, tmpDir, tmpDir, "./public")

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}

	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array, got %T: %s", result, result.Inspect())
	}

	// Should have 2 results
	if len(arr.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(arr.Elements))
	}

	// Each should be a web URL starting with /images/
	for _, elem := range arr.Elements {
		path := elem.Inspect()
		if path != "/images/a.png" && path != "/images/b.png" {
			t.Errorf("expected /images/*.png web URL, got %q", path)
		}
	}
}

// TestAssetWithFileDictInHTML tests asset() with file dicts in HTML context
func TestAssetWithFileDictInHTML(t *testing.T) {
	tmpDir := t.TempDir()
	publicDir := filepath.Join(tmpDir, "public")
	imagesDir := filepath.Join(publicDir, "img")

	os.MkdirAll(imagesDir, 0755)
	os.WriteFile(filepath.Join(imagesDir, "photo.jpg"), []byte("img"), 0644)

	// Using absolute path
	input := `
		let photos = fileList(@` + imagesDir + `/*.jpg)
		let photo = photos[0]
		<img src={asset(photo)}/>
	`

	result := evalWithFilesContext(t, input, tmpDir, tmpDir, "./public")

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}

	expected := `<img src="/img/photo.jpg" />`
	if result.Inspect() != expected {
		t.Errorf("expected %q, got %q", expected, result.Inspect())
	}
}

// TestFileListEnvironmentPreservation tests that fileList() preserves environment for asset()
func TestFileListEnvironmentPreservation(t *testing.T) {
	// This tests the fix where fileList() would lose the caller's environment
	// (including public_dir) when the file dictionaries were used with asset()

	tmpDir := t.TempDir()
	publicDir := filepath.Join(tmpDir, "public")
	cssDir := filepath.Join(publicDir, "css")

	os.MkdirAll(cssDir, 0755)
	os.WriteFile(filepath.Join(cssDir, "style.css"), []byte("body{}"), 0644)

	// The key test: in a for loop, the file dicts from fileList() should retain
	// enough context for asset() to work correctly
	// Using absolute path pattern since relative paths resolve against cwd
	input := `
		let styles = fileList(@` + publicDir + `/css/*.css)
		for (css in styles) {
			<link rel=stylesheet href={asset(css)}/>
		}
	`

	result := evalWithFilesContext(t, input, tmpDir, tmpDir, "./public")

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}

	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array, got %T: %s", result, result.Inspect())
	}

	if len(arr.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(arr.Elements))
	}

	expected := `<link rel=stylesheet href="/css/style.css" />`
	if arr.Elements[0].Inspect() != expected {
		t.Errorf("expected %q, got %q", expected, arr.Elements[0].Inspect())
	}
}

// TestAssetWithPathLiteral tests asset() with direct path literals
func TestAssetWithPathLiteral(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		publicDir string
		expected  string
	}{
		{
			name:      "path_under_public_dir",
			input:     `asset(@./public/js/app.js)`,
			publicDir: "./public",
			expected:  `/js/app.js`,
		},
		{
			name:      "path_not_under_public_dir",
			input:     `asset(@./data/config.json)`,
			publicDir: "./public",
			expected:  `./data/config.json`,
		},
		{
			name:      "path_without_public_dir",
			input:     `asset(@./public/style.css)`,
			publicDir: "",
			expected:  `./public/style.css`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			result := evalWithFilesContext(t, tt.input, tmpDir, tmpDir, tt.publicDir)

			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestAssetWithStringPath tests asset() with string paths
// Note: asset() with string paths works differently than with path literals
func TestAssetWithStringPath(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		publicDir string
		expected  string
	}{
		{
			name:      "string_path_outside_public",
			input:     `asset("./private/secret.txt")`,
			publicDir: "./public",
			expected:  `./private/secret.txt`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			result := evalWithFilesContext(t, tt.input, tmpDir, tmpDir, tt.publicDir)

			if result.Inspect() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.Inspect())
			}
		})
	}
}

// TestFileListWithAssetPipeline tests a realistic pipeline: fileList() -> map with asset()
func TestFileListWithAssetPipeline(t *testing.T) {
	tmpDir := t.TempDir()
	publicDir := filepath.Join(tmpDir, "public")
	assetsDir := filepath.Join(publicDir, "assets")

	os.MkdirAll(assetsDir, 0755)
	os.WriteFile(filepath.Join(assetsDir, "main.js"), []byte("js"), 0644)
	os.WriteFile(filepath.Join(assetsDir, "vendor.js"), []byte("js"), 0644)

	// Realistic use case: collect files and generate script tags
	// Using absolute path pattern
	input := `
		let scripts = fileList(@` + publicDir + `/assets/*.js)
		for (script in scripts) {
			<script src={asset(script)}/>
		}
	`

	result := evalWithFilesContext(t, input, tmpDir, tmpDir, "./public")

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}

	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array, got %T: %s", result, result.Inspect())
	}

	if len(arr.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(arr.Elements))
	}

	// Check that each element is a proper script tag with web URL
	for _, elem := range arr.Elements {
		tag := elem.Inspect()
		// Should be <script src="/assets/xxx.js" />
		if tag != `<script src="/assets/main.js" />` && tag != `<script src="/assets/vendor.js" />` {
			t.Errorf("unexpected tag: %q", tag)
		}
	}
}

// TestAssetWithDirectoryDict tests asset() with directory dictionaries
func TestAssetWithDirectoryDict(t *testing.T) {
	tmpDir := t.TempDir()
	publicDir := filepath.Join(tmpDir, "public")
	subDir := filepath.Join(publicDir, "subdir")

	os.MkdirAll(subDir, 0755)

	// Test asset() on a directory path
	input := `asset(@./public/subdir)`

	result := evalWithFilesContext(t, input, tmpDir, tmpDir, "./public")

	expected := `/subdir`
	if result.Inspect() != expected {
		t.Errorf("expected %q, got %q", expected, result.Inspect())
	}
}
