package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestRootPathImport(t *testing.T) {
	// Create temp directory structure:
	// tmpdir/
	//   main.pars
	//   components/
	//     header.pars
	//   utils/
	//     deep/
	//       nested.pars

	tmpDir := t.TempDir()

	// Create directories
	componentsDir := filepath.Join(tmpDir, "components")
	utilsDeepDir := filepath.Join(tmpDir, "utils", "deep")
	if err := os.MkdirAll(componentsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(utilsDeepDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create header.pars
	headerContent := `export title = "Header Component"`
	if err := os.WriteFile(filepath.Join(componentsDir, "header.pars"), []byte(headerContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create nested.pars - imports from root using ~/
	nestedContent := `{title} = import(@~/components/header.pars)
export message = "Nested says: " + title`
	if err := os.WriteFile(filepath.Join(utilsDeepDir, "nested.pars"), []byte(nestedContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main.pars - imports nested module
	mainContent := `{message} = import(@./utils/deep/nested.pars)
message`
	mainPath := filepath.Join(tmpDir, "main.pars")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Parse and evaluate main.pars with RootPath set
	l := lexer.New(mainContent)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.Filename = mainPath
	env.RootPath = tmpDir // Set the root path

	// Allow reading/executing from tmpDir
	env.Security = &evaluator.SecurityPolicy{
		AllowExecute: []string{tmpDir},
	}

	result := evaluator.Eval(program, env)

	if result == nil {
		t.Fatal("Eval returned nil")
	}

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("Eval returned error: %s", errObj.Message)
	}

	expected := "Nested says: Header Component"
	if result.Inspect() != expected {
		t.Errorf("expected %q, got %q", expected, result.Inspect())
	}
}

func TestRootPathWithoutRootSet(t *testing.T) {
	// Test that ~/ falls back to home directory when RootPath is not set
	// This maintains backward compatibility for standalone pars usage

	// Create a temp file in home directory simulation isn't practical,
	// so we just test that parsing works and doesn't error
	input := `let path = @~/some/module.pars
path.path` // Access the path property of the path dictionary

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	// Don't set RootPath - simulating standalone pars CLI

	result := evaluator.Eval(program, env)

	if result == nil {
		t.Fatal("Eval returned nil")
	}

	// Should not error - ~/ should expand to home directory
	if errObj, ok := result.(*evaluator.Error); ok {
		// Only fail if it's NOT a "file not found" error (which is expected)
		if !containsSubstr(errObj.Message, "not found") && !containsSubstr(errObj.Message, "no such file") {
			t.Fatalf("unexpected error: %s", errObj.Message)
		}
	}

	// The path should contain the expanded home directory
	pathStr := result.Inspect()
	if containsSubstr(pathStr, "~") {
		t.Errorf("expected ~ to be expanded, got: %s", pathStr)
	}
}

func TestRootPathRead(t *testing.T) {
	// Test that ~/ works with file read operations
	tmpDir := t.TempDir()

	// Create a data file
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	dataContent := `{"name": "test"}`
	if err := os.WriteFile(filepath.Join(dataDir, "config.json"), []byte(dataContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Script that reads from ~/data/config.json
	input := `let data <== JSON(@~/data/config.json)
data.name`

	mainPath := filepath.Join(tmpDir, "main.pars")

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	env.Filename = mainPath
	env.RootPath = tmpDir

	// Allow reading from tmpDir
	env.Security = &evaluator.SecurityPolicy{
		NoRead: false,
	}

	result := evaluator.Eval(program, env)

	if result == nil {
		t.Fatal("Eval returned nil")
	}

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("Eval returned error: %s", errObj.Message)
	}

	if result.Inspect() != "test" {
		t.Errorf("expected \"test\", got %q", result.Inspect())
	}
}
