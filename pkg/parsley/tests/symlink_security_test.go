package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestSymlinkSecurityRead tests that symlinks are properly resolved for read access
func TestSymlinkSecurityRead(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require admin on Windows")
	}

	// Create directory structure:
	// tmpdir/
	//   allowed/
	//     real.txt
	//   link_to_allowed -> allowed/
	//   restricted/
	//     secret.txt

	tmpDir := t.TempDir()

	allowedDir := filepath.Join(tmpDir, "allowed")
	restrictedDir := filepath.Join(tmpDir, "restricted")
	linkDir := filepath.Join(tmpDir, "link_to_allowed")

	os.MkdirAll(allowedDir, 0755)
	os.MkdirAll(restrictedDir, 0755)

	// Create files
	realFile := filepath.Join(allowedDir, "real.txt")
	secretFile := filepath.Join(restrictedDir, "secret.txt")
	os.WriteFile(realFile, []byte("allowed content"), 0644)
	os.WriteFile(secretFile, []byte("secret content"), 0644)

	// Create symlink
	err := os.Symlink(allowedDir, linkDir)
	if err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	// Test 1: Reading through symlink should work if target is allowed
	t.Run("read_through_symlink_allowed", func(t *testing.T) {
		// Use the <== read operator to read the file
		input := `content <== text("` + filepath.Join(linkDir, "real.txt") + `")`

		env := evaluator.NewEnvironment()
		env.Security = &evaluator.SecurityPolicy{}

		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		result := evaluator.Eval(program, env)

		if errObj, ok := result.(*evaluator.Error); ok {
			t.Fatalf("unexpected error: %s", errObj.Message)
		}

		strResult, ok := result.(*evaluator.String)
		if !ok {
			t.Fatalf("expected String, got %T: %s", result, result.Inspect())
		}

		if strResult.Value != "allowed content" {
			t.Errorf("expected 'allowed content', got %q", strResult.Value)
		}
	})
}

// TestSymlinkSecurityWrite tests that symlinks are properly resolved for write access
func TestSymlinkSecurityWrite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require admin on Windows")
	}

	tmpDir := t.TempDir()

	allowedDir := filepath.Join(tmpDir, "writable")
	linkDir := filepath.Join(tmpDir, "link_to_writable")

	os.MkdirAll(allowedDir, 0755)

	// Create symlink
	err := os.Symlink(allowedDir, linkDir)
	if err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	// Test: Writing through symlink should work if target is in AllowWrite
	t.Run("write_through_symlink", func(t *testing.T) {
		targetFile := filepath.Join(linkDir, "output.txt")
		input := `"hello via symlink" ==> text("` + targetFile + `")`

		env := evaluator.NewEnvironment()
		env.Security = &evaluator.SecurityPolicy{
			AllowWrite: []string{allowedDir}, // Allow the real directory
		}

		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		result := evaluator.Eval(program, env)

		if errObj, ok := result.(*evaluator.Error); ok {
			t.Fatalf("unexpected error: %s", errObj.Message)
		}

		// Verify file was written
		content, err := os.ReadFile(filepath.Join(allowedDir, "output.txt"))
		if err != nil {
			t.Fatalf("file not created: %v", err)
		}

		if string(content) != "hello via symlink" {
			t.Errorf("expected 'hello via symlink', got %q", string(content))
		}
	})
}

// TestSymlinkSecurityExecute tests that symlinks are properly resolved for script execution
func TestSymlinkSecurityExecute(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require admin on Windows")
	}

	tmpDir := t.TempDir()

	scriptsDir := filepath.Join(tmpDir, "scripts")
	linkDir := filepath.Join(tmpDir, "link_to_scripts")

	os.MkdirAll(scriptsDir, 0755)

	// Create a module
	modulePath := filepath.Join(scriptsDir, "helper.pars")
	os.WriteFile(modulePath, []byte(`export greeting = "Hello from symlinked module"`), 0644)

	// Create symlink
	err := os.Symlink(scriptsDir, linkDir)
	if err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	evaluator.ClearModuleCache()

	// Test: Importing through symlink should work if target is allowed
	t.Run("import_through_symlink", func(t *testing.T) {
		linkedModule := filepath.Join(linkDir, "helper.pars")
		input := `let m = import @` + linkedModule + `; m.greeting`

		env := evaluator.NewEnvironment()
		env.Filename = filepath.Join(tmpDir, "main.pars")
		env.Security = &evaluator.SecurityPolicy{
			AllowExecute: []string{scriptsDir}, // Allow real directory
		}

		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		result := evaluator.Eval(program, env)

		if errObj, ok := result.(*evaluator.Error); ok {
			t.Fatalf("unexpected error: %s", errObj.Message)
		}

		strResult, ok := result.(*evaluator.String)
		if !ok {
			t.Fatalf("expected String, got %T: %s", result, result.Inspect())
		}

		if strResult.Value != "Hello from symlinked module" {
			t.Errorf("expected 'Hello from symlinked module', got %q", strResult.Value)
		}
	})
}

// TestMacOSVarSymlink tests the specific macOS /var -> /private/var case
func TestMacOSVarSymlink(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-specific test")
	}

	// macOS uses /var which is a symlink to /private/var
	// Both paths should be treated as equivalent for security purposes

	// Create a temp file - os.TempDir() on macOS returns /var/... path
	tmpDir := t.TempDir() // This might be /var/folders/... or /private/var/folders/...

	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	// Get the "real" path by resolving symlinks
	realPath, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// If they're different, we have a symlink situation to test
	if realPath != tmpDir {
		t.Logf("Testing symlink: %s -> %s", tmpDir, realPath)

		// Test that we can read using the symlink path when real path is allowed
		t.Run("read_via_symlink_path", func(t *testing.T) {
			// Use the <== read operator with text()
			input := `content <== text("` + testFile + `")`

			env := evaluator.NewEnvironment()
			env.Security = &evaluator.SecurityPolicy{}

			l := lexer.New(input)
			p := parser.New(l)
			program := p.ParseProgram()

			result := evaluator.Eval(program, env)

			if errObj, ok := result.(*evaluator.Error); ok {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			// The result should be the string content
			strResult, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String, got %T", result)
			}

			if strResult.Value != "test content" {
				t.Errorf("expected 'test content', got %q", strResult.Value)
			}
		})
	} else {
		t.Log("No symlink difference found in temp dir path")
	}
}

// TestSymlinkRestriction tests that restricted paths work through symlinks
func TestSymlinkRestriction(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require admin on Windows")
	}

	tmpDir := t.TempDir()

	sensitiveDir := filepath.Join(tmpDir, "sensitive")
	linkToSensitive := filepath.Join(tmpDir, "innocent_link")

	os.MkdirAll(sensitiveDir, 0755)
	os.WriteFile(filepath.Join(sensitiveDir, "password.txt"), []byte("secret123"), 0644)

	// Create symlink trying to bypass restriction
	err := os.Symlink(sensitiveDir, linkToSensitive)
	if err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	// Test: Reading through symlink should fail if target is restricted
	t.Run("restricted_via_symlink", func(t *testing.T) {
		input := `read(@` + filepath.Join(linkToSensitive, "password.txt") + `)`

		env := evaluator.NewEnvironment()
		env.Security = &evaluator.SecurityPolicy{
			RestrictRead: []string{sensitiveDir}, // Restrict the real directory
		}

		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("parser errors: %v", p.Errors())
		}

		result := evaluator.Eval(program, env)

		// Should get an error about restricted access
		errObj, ok := result.(*evaluator.Error)
		if !ok {
			t.Fatalf("expected error for restricted path, got %T: %s", result, result.Inspect())
		}

		if errObj.Message == "" {
			t.Error("expected non-empty error message")
		}
	})
}
