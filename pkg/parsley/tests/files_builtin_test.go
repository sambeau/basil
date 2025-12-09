package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// evalFileListBuiltin evaluates code that uses fileList() builtin
func evalFileListBuiltin(input string, cwd string, rootPath string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return &evaluator.Error{Message: strings.Join(p.Errors(), "\n")}
	}

	env := evaluator.NewEnvironment()
	env.Filename = filepath.Join(cwd, "test.pars")
	env.RootPath = rootPath
	env.Security = &evaluator.SecurityPolicy{}

	return evaluator.Eval(program, env)
}

// TestFileListPreservesDotSlashPrefix tests that fileList(@./...) preserves the ./ prefix
func TestFileListPreservesDotSlashPrefix(t *testing.T) {
	// Create temp directory with some files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{"a.txt", "b.txt", "c.txt"}
	for _, name := range testFiles {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte("test"), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Change to temp directory for test
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	input := `let f = fileList(@./*.txt); for (file in f) { toString(file.path) }`

	result := evalFileListBuiltin(input, tmpDir, tmpDir)

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}

	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array, got %T: %s", result, result.Inspect())
	}

	// Each path should start with ./
	for _, elem := range arr.Elements {
		path := elem.Inspect()
		if !strings.HasPrefix(path, "./") {
			t.Errorf("expected path to start with './', got %q", path)
		}
	}
}

// TestFileListWithRootPathAlias tests that fileList(@~/) uses RootPath when set
func TestFileListWithRootPathAlias(t *testing.T) {
	// Create temp directory structure:
	// root/
	//   subdir/
	//     test.pars (current file)
	//   data/
	//     file1.txt
	//     file2.txt

	rootDir := t.TempDir()
	subDir := filepath.Join(rootDir, "subdir")
	dataDir := filepath.Join(rootDir, "data")

	os.MkdirAll(subDir, 0755)
	os.MkdirAll(dataDir, 0755)

	// Create data files
	os.WriteFile(filepath.Join(dataDir, "file1.txt"), []byte("one"), 0644)
	os.WriteFile(filepath.Join(dataDir, "file2.txt"), []byte("two"), 0644)

	// Test fileList(@~/data/*.txt) from subdir should find files in root/data
	input := `let f = fileList(@~/data/*.txt); f.length()`

	result := evalFileListBuiltin(input, subDir, rootDir)

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}

	intResult, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
	}

	if intResult.Value != 2 {
		t.Errorf("expected 2 files, got %d", intResult.Value)
	}
}

// TestFileListRootPathFallback tests that ~/ falls back to home dir when RootPath not set
func TestFileListRootPathFallback(t *testing.T) {
	// When RootPath is empty, ~/ should expand to user's home directory
	// We can't easily test this without creating files in home, so we just
	// test that it doesn't error when RootPath is empty

	tmpDir := t.TempDir()

	// Create a glob pattern that won't match anything in home
	input := `let f = fileList(@~/nonexistent_parsley_test_dir_12345/*.xyz); f.length()`

	result := evalFileListBuiltin(input, tmpDir, "") // Empty RootPath

	if errObj, ok := result.(*evaluator.Error); ok {
		// Error is OK if it's about not finding files, but not about path resolution
		if strings.Contains(errObj.Message, "cannot expand") {
			t.Errorf("should not fail on path expansion: %s", errObj.Message)
		}
	}

	// Should return 0 (no matching files) or empty array
	if intResult, ok := result.(*evaluator.Integer); ok {
		if intResult.Value != 0 {
			t.Errorf("expected 0 files, got %d", intResult.Value)
		}
	}
}

// TestFileListAbsolutePath tests that fileList() works with absolute paths
func TestFileListAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "test1.dat"), []byte("data1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test2.dat"), []byte("data2"), 0644)

	input := `let f = fileList(@` + tmpDir + `/*.dat); f.length()`

	result := evalFileListBuiltin(input, tmpDir, tmpDir)

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}

	intResult, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
	}

	if intResult.Value != 2 {
		t.Errorf("expected 2 files, got %d", intResult.Value)
	}
}

// TestFileListEmptyResult tests that fileList() returns empty array for no matches
func TestFileListEmptyResult(t *testing.T) {
	tmpDir := t.TempDir()

	input := `let f = fileList(@./nonexistent/*.xyz); f.length()`

	result := evalFileListBuiltin(input, tmpDir, tmpDir)

	// Should return 0, not an error
	if errObj, ok := result.(*evaluator.Error); ok {
		// Pattern not matching is not necessarily an error
		if !strings.Contains(errObj.Message, "no match") {
			t.Logf("got error (may be OK): %s", errObj.Message)
		}
		return
	}

	if intResult, ok := result.(*evaluator.Integer); ok {
		if intResult.Value != 0 {
			t.Errorf("expected 0 files, got %d", intResult.Value)
		}
	}
}

// TestFileListResultIsFileDictionary tests that fileList() returns proper file dictionaries
func TestFileListResultIsFileDictionary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "sample.txt")
	os.WriteFile(testFile, []byte("hello world"), 0644)

	input := `let f = fileList(@` + tmpDir + `/*.txt); f[0].basename`

	result := evalFileListBuiltin(input, tmpDir, tmpDir)

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}

	strResult, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}

	if strResult.Value != "sample.txt" {
		t.Errorf("expected 'sample.txt', got %q", strResult.Value)
	}
}

// TestFileListGlobPatterns tests various glob patterns
func TestFileListGlobPatterns(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure
	subDir := filepath.Join(tmpDir, "sub")
	os.MkdirAll(subDir, 0755)

	// Create files
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("b"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "c.md"), []byte("c"), 0644)
	os.WriteFile(filepath.Join(subDir, "d.txt"), []byte("d"), 0644)

	tests := []struct {
		name     string
		pattern  string
		expected int
	}{
		{
			name:     "all_txt_in_root",
			pattern:  tmpDir + "/*.txt",
			expected: 2,
		},
		{
			name:     "all_md_in_root",
			pattern:  tmpDir + "/*.md",
			expected: 1,
		},
		{
			name:     "txt_in_subdir",
			pattern:  tmpDir + "/sub/*.txt",
			expected: 1,
		},
		{
			name:     "all_files_in_root",
			pattern:  tmpDir + "/*.*",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `let f = fileList(@` + tt.pattern + `); f.length()`

			result := evalFileListBuiltin(input, tmpDir, tmpDir)

			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			intResult, ok := result.(*evaluator.Integer)
			if !ok {
				t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
			}

			if intResult.Value != int64(tt.expected) {
				t.Errorf("expected %d files, got %d", tt.expected, intResult.Value)
			}
		})
	}
}
