package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestImportWithoutExecutePermission tests that local module imports work
// without execute permission (BUG-022 fix)
func TestImportWithoutExecutePermission(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple module
	modulePath := filepath.Join(tmpDir, "utils.pars")
	moduleCode := `export let greet = fn(name) { "Hello, " + name + "!" }`
	err := os.WriteFile(modulePath, []byte(moduleCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create main file that imports the module
	mainCode := `
let utils = import @` + modulePath + `
utils.greet("World")
`

	// Create environment WITHOUT execute permission
	env := evaluator.NewEnvironment()
	env.Filename = filepath.Join(tmpDir, "main.pars")
	env.Security = &evaluator.SecurityPolicy{
		AllowExecuteAll: false, // Explicitly no execute permission
	}

	l := lexer.New(mainCode)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	result := evaluator.Eval(program, env)

	// Should NOT be an error - imports should work without execute permission
	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("import should work without execute permission, got error: %s", errObj.Message)
	}

	// Verify the result is correct
	strResult, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}

	if strResult.Value != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", strResult.Value)
	}
}

// TestImportWithRelativePath tests that relative path imports work without execute permission
func TestImportWithRelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	libDir := filepath.Join(tmpDir, "lib")
	os.Mkdir(libDir, 0755)

	// Create a module in lib subdirectory
	modulePath := filepath.Join(libDir, "utils.pars")
	moduleCode := `export let add = fn(a, b) { a + b }`
	err := os.WriteFile(modulePath, []byte(moduleCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Use relative path import
	mainCode := `
let utils = import @./lib/utils.pars
utils.add(2, 3)
`

	// Create environment WITHOUT execute permission
	env := evaluator.NewEnvironment()
	env.Filename = filepath.Join(tmpDir, "main.pars")
	env.Security = &evaluator.SecurityPolicy{
		AllowExecuteAll: false,
	}

	l := lexer.New(mainCode)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	result := evaluator.Eval(program, env)

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("relative path import should work, got error: %s", errObj.Message)
	}

	intResult, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
	}

	if intResult.Value != 5 {
		t.Errorf("expected 5, got %d", intResult.Value)
	}
}

// TestShellStillRequiresExecutePermission verifies that @shell still requires
// execute permission (regression test - this should NOT change with BUG-022 fix)
func TestShellStillRequiresExecutePermission(t *testing.T) {
	// Try to execute a shell command without execute permission
	code := `@shell("echo", ["hello"]) <=#=> null`

	env := evaluator.NewEnvironment()
	env.Security = &evaluator.SecurityPolicy{
		AllowExecuteAll: false, // No execute permission
	}

	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	result := evaluator.Eval(program, env)

	// Shell commands return a dictionary with an error field when they fail
	dictResult, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary result from shell command, got: %T %s", result, result.Inspect())
	}

	// Check that there's an error field indicating security failure
	errorExpr, hasError := dictResult.Pairs["error"]
	if !hasError {
		t.Fatalf("expected error field in shell result, got: %s", result.Inspect())
	}

	// Evaluate the error expression
	errorObj := evaluator.Eval(errorExpr, env)
	errorStr, ok := errorObj.(*evaluator.String)
	if !ok || errorStr.Value == "" {
		t.Fatalf("expected non-empty error message in shell result, got: %s", result.Inspect())
	}

	// Verify the error message mentions security/permission
	if !containsAny(errorStr.Value, []string{"security", "not allowed", "denied"}) {
		t.Errorf("expected security-related error message, got: %s", errorStr.Value)
	}
}

// TestImportWithRestrictRead tests that RestrictRead policy blocks imports
func TestImportWithRestrictRead(t *testing.T) {
	tmpDir := t.TempDir()
	restrictedDir := filepath.Join(tmpDir, "restricted")
	os.Mkdir(restrictedDir, 0755)

	// Create a module in restricted directory
	modulePath := filepath.Join(restrictedDir, "secret.pars")
	moduleCode := `export let secret = "classified"`
	err := os.WriteFile(modulePath, []byte(moduleCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	mainCode := `import @` + modulePath

	// Create environment with read restriction on that directory
	env := evaluator.NewEnvironment()
	env.Filename = filepath.Join(tmpDir, "main.pars")
	env.Security = &evaluator.SecurityPolicy{
		RestrictRead: []string{restrictedDir},
	}

	l := lexer.New(mainCode)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	result := evaluator.Eval(program, env)

	// Should be an error - read is restricted
	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("import from restricted directory should fail, got: %T %s", result, result.Inspect())
	}

	// Should be a security error about read access (SEC-0001 or SEC-0002)
	if errObj.Code != "SEC-0001" && errObj.Code != "SEC-0002" {
		t.Errorf("expected security error, got %s: %s", errObj.Code, errObj.Message)
	}
}

// TestImportDoesNotRequireExecuteFlag tests that the -x flag is not needed for imports
func TestImportDoesNotRequireExecuteFlag(t *testing.T) {
	tmpDir := t.TempDir()

	// Create module
	modulePath := filepath.Join(tmpDir, "math.pars")
	moduleCode := `export let double = fn(x) { x * 2 }`
	err := os.WriteFile(modulePath, []byte(moduleCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	mainCode := `
let math = import @` + modulePath + `
math.double(21)
`

	// Simulate default security policy (no -x flag)
	// When Security is nil, execute is denied but read is allowed
	env := evaluator.NewEnvironment()
	env.Filename = filepath.Join(tmpDir, "main.pars")
	env.Security = nil // This simulates running without any security flags

	l := lexer.New(mainCode)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	result := evaluator.Eval(program, env)

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("import should work with nil security policy (no -x flag), got error: %s", errObj.Message)
	}

	intResult, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
	}

	if intResult.Value != 42 {
		t.Errorf("expected 42, got %d", intResult.Value)
	}
}

// TestNestedImportsWithoutExecute tests that nested imports work without execute permission
func TestNestedImportsWithoutExecute(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first module that imports another
	module2Path := filepath.Join(tmpDir, "module2.pars")
	module2Code := `export let value = 100`
	os.WriteFile(module2Path, []byte(module2Code), 0644)

	module1Path := filepath.Join(tmpDir, "module1.pars")
	module1Code := `
let mod2 = import @` + module2Path + `
export let getValue = fn() { mod2.value }
`
	os.WriteFile(module1Path, []byte(module1Code), 0644)

	// Main file imports module1
	mainCode := `
let mod1 = import @` + module1Path + `
mod1.getValue()
`

	// No execute permission
	env := evaluator.NewEnvironment()
	env.Filename = filepath.Join(tmpDir, "main.pars")
	env.Security = &evaluator.SecurityPolicy{
		AllowExecuteAll: false,
	}

	l := lexer.New(mainCode)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	result := evaluator.Eval(program, env)

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("nested imports should work without execute permission, got error: %s", errObj.Message)
	}

	intResult, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
	}

	if intResult.Value != 100 {
		t.Errorf("expected 100, got %d", intResult.Value)
	}
}

// Helper function to check if string contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
