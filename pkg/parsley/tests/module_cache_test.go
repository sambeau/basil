package tests

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestModuleCacheClear tests that ClearModuleCache() clears all cached modules
func TestModuleCacheClear(t *testing.T) {
	// Create a temp module
	tmpDir := t.TempDir()
	modulePath := filepath.Join(tmpDir, "counter.pars")

	// First version of module
	err := os.WriteFile(modulePath, []byte(`export count = 1`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Import the module
	input := `let mod = import @` + modulePath + `; mod.count`

	env := evaluator.NewEnvironment()
	env.Filename = filepath.Join(tmpDir, "main.pars")
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	result := evaluator.Eval(program, env)

	intResult, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T: %s", result, result.Inspect())
	}
	if intResult.Value != 1 {
		t.Errorf("expected 1, got %d", intResult.Value)
	}

	// Update the module
	err = os.WriteFile(modulePath, []byte(`export count = 99`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Without clearing cache, should still get old value
	env2 := evaluator.NewEnvironment()
	env2.Filename = filepath.Join(tmpDir, "main.pars")
	env2.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	l2 := lexer.New(input)
	p2 := parser.New(l2)
	program2 := p2.ParseProgram()

	result2 := evaluator.Eval(program2, env2)

	intResult2, ok := result2.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T: %s", result2, result2.Inspect())
	}
	// Should be cached value (1), not new value (99)
	if intResult2.Value != 1 {
		t.Logf("Note: got %d (cache may have been cleared by another test)", intResult2.Value)
	}

	// Now clear the cache
	evaluator.ClearModuleCache()

	// Should get new value
	env3 := evaluator.NewEnvironment()
	env3.Filename = filepath.Join(tmpDir, "main.pars")
	env3.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	l3 := lexer.New(input)
	p3 := parser.New(l3)
	program3 := p3.ParseProgram()

	result3 := evaluator.Eval(program3, env3)

	intResult3, ok := result3.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T: %s", result3, result3.Inspect())
	}
	if intResult3.Value != 99 {
		t.Errorf("expected 99 after cache clear, got %d", intResult3.Value)
	}
}

// TestModuleCacheThreadSafety tests that module cache is thread-safe
func TestModuleCacheThreadSafety(t *testing.T) {
	// Create a temp module
	tmpDir := t.TempDir()
	modulePath := filepath.Join(tmpDir, "shared.pars")

	err := os.WriteFile(modulePath, []byte(`export value = 42`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Clear cache to start fresh
	evaluator.ClearModuleCache()

	// Import from multiple goroutines concurrently
	var wg sync.WaitGroup
	errors := make(chan error, 100)
	results := make(chan int64, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			input := `let mod = import @` + modulePath + `; mod.value`

			env := evaluator.NewEnvironment()
			env.Filename = filepath.Join(tmpDir, "test.pars")
			env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

			l := lexer.New(input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				errors <- &parseError{p.Errors()}
				return
			}

			result := evaluator.Eval(program, env)

			if errObj, ok := result.(*evaluator.Error); ok {
				errors <- &evalError{errObj.Message}
				return
			}

			if intResult, ok := result.(*evaluator.Integer); ok {
				results <- intResult.Value
			}
		}()
	}

	wg.Wait()
	close(errors)
	close(results)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent import error: %v", err)
	}

	// All results should be 42
	for val := range results {
		if val != 42 {
			t.Errorf("expected 42, got %d", val)
		}
	}
}

type parseError struct {
	errors []string
}

func (e *parseError) Error() string {
	return e.errors[0]
}

type evalError struct {
	message string
}

func (e *evalError) Error() string {
	return e.message
}

// TestModuleCacheConcurrentClearAndImport tests clearing cache while imports happen
func TestModuleCacheConcurrentClearAndImport(t *testing.T) {
	// Create a temp module
	tmpDir := t.TempDir()
	modulePath := filepath.Join(tmpDir, "cleartest.pars")

	err := os.WriteFile(modulePath, []byte(`export num = 100`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	evaluator.ClearModuleCache()

	var wg sync.WaitGroup

	// Goroutine that repeatedly clears cache
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			evaluator.ClearModuleCache()
		}
	}()

	// Goroutines that import
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				input := `let mod = import @` + modulePath + `; mod.num`

				env := evaluator.NewEnvironment()
				env.Filename = filepath.Join(tmpDir, "test.pars")
				env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

				l := lexer.New(input)
				p := parser.New(l)
				program := p.ParseProgram()

				if len(p.Errors()) == 0 {
					result := evaluator.Eval(program, env)
					// We just want to make sure it doesn't panic
					_ = result
				}
			}
		}()
	}

	wg.Wait()
	// Test passes if no panic/race condition
}

// TestModuleCacheIsolation tests that different modules are cached separately
func TestModuleCacheIsolation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two different modules
	mod1Path := filepath.Join(tmpDir, "mod1.pars")
	mod2Path := filepath.Join(tmpDir, "mod2.pars")

	os.WriteFile(mod1Path, []byte(`export val = "one"`), 0644)
	os.WriteFile(mod2Path, []byte(`export val = "two"`), 0644)

	evaluator.ClearModuleCache()

	// Import both
	input := `
		let m1 = import @` + mod1Path + `
		let m2 = import @` + mod2Path + `
		m1.val + "-" + m2.val
	`

	env := evaluator.NewEnvironment()
	env.Filename = filepath.Join(tmpDir, "main.pars")
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	result := evaluator.Eval(program, env)

	strResult, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}

	if strResult.Value != "one-two" {
		t.Errorf("expected 'one-two', got %q", strResult.Value)
	}
}

// TestDynamicAccessorInCachedModule tests that @basil/http imports at module scope
func TestDynamicAccessorInCachedModule(t *testing.T) {
	evaluator.ClearModuleCache()

	// Create a module that imports @basil/http at module scope
	tmpDir := t.TempDir()
	modulePath := filepath.Join(tmpDir, "handler.pars")

	moduleCode := `let {method} = import @basil/http
export let getMethod = fn() { method }
`
	err := os.WriteFile(modulePath, []byte(moduleCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	mainCode := `
let handler = import @` + modulePath + `
handler.getMethod()
`

	// Create first environment with method GET
	env1 := evaluator.NewEnvironment()
	env1.Filename = filepath.Join(tmpDir, "main.pars")
	env1.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}
	// Set up BasilCtx with request data (method defaults to GET)
	env1.BasilCtx = evaluator.BuildTestBasilContext(map[string]string{}, nil, nil)

	l1 := lexer.New(mainCode)
	p1 := parser.New(l1)
	program1 := p1.ParseProgram()
	if len(p1.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p1.Errors())
	}

	result1 := evaluator.Eval(program1, env1)
	if err, ok := result1.(*evaluator.Error); ok {
		t.Fatalf("first request error: %s", err.Inspect())
	}

	t.Logf("Result1 type: %T, value: %s", result1, result1.Inspect())

	str1, ok := result1.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result1, result1.Inspect())
	}

	// Verify first request got "GET"
	if str1.Value != "GET" {
		t.Errorf("first request: expected method='GET', got %s", str1.Value)
	}

	// Create second environment (simulating new request)
	// The module should be CACHED, but method should still be FRESH from current context
	env2 := evaluator.NewEnvironment()
	env2.Filename = filepath.Join(tmpDir, "main.pars")
	env2.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}
	env2.BasilCtx = evaluator.BuildTestBasilContext(map[string]string{}, nil, nil)

	l2 := lexer.New(mainCode)
	p2 := parser.New(l2)
	program2 := p2.ParseProgram()
	if len(p2.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p2.Errors())
	}

	result2 := evaluator.Eval(program2, env2)
	if err, ok := result2.(*evaluator.Error); ok {
		t.Fatalf("second request error: %s", err.Inspect())
	}

	t.Logf("Result2 type: %T, value: %s", result2, result2.Inspect())

	str2, ok := result2.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result2, result2.Inspect())
	}

	// Verify second request still got "GET" (showing dynamic accessor works even with cached module)
	if str2.Value != "GET" {
		t.Errorf("second request: expected method='GET', got %s (BUG-014: value cached from first request)", str2.Value)
	}
}

// TestParamsDynamicAccessorInModule tests that params from @basil/http
// resolves @params from the environment chain, not from a cached value.
func TestParamsDynamicAccessorInModule(t *testing.T) {
	evaluator.ClearModuleCache()

	// Create a module that imports params from @basil/http at module scope
	tmpDir := t.TempDir()
	modulePath := filepath.Join(tmpDir, "handler.pars")

	moduleCode := `let {params} = import @basil/http
export let getOrderBy = fn() { params.orderBy ?? "none" }
`
	err := os.WriteFile(modulePath, []byte(moduleCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	mainCode := `
let handler = import @` + modulePath + `
handler.getOrderBy()
`

	// Create first environment with @params containing orderBy=name
	env1 := evaluator.NewEnvironment()
	env1.Filename = filepath.Join(tmpDir, "main.pars")
	env1.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}
	env1.BasilCtx = evaluator.BuildTestBasilContext(map[string]string{}, nil, nil)
	// Set @params directly (as handler.go does)
	env1.Set("@params", &evaluator.Dictionary{
		Pairs: map[string]ast.Expression{
			"orderBy": &ast.StringLiteral{Value: "name"},
		},
	})

	l1 := lexer.New(mainCode)
	p1 := parser.New(l1)
	program1 := p1.ParseProgram()
	if len(p1.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p1.Errors())
	}

	result1 := evaluator.Eval(program1, env1)
	if err, ok := result1.(*evaluator.Error); ok {
		t.Fatalf("first request error: %s", err.Inspect())
	}

	str1, ok := result1.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result1, result1.Inspect())
	}

	if str1.Value != "name" {
		t.Errorf("first request: expected orderBy='name', got %s", str1.Value)
	}

	// Create second environment with different @params (orderBy=age)
	// The module should be CACHED, but params should be FRESH from current env
	env2 := evaluator.NewEnvironment()
	env2.Filename = filepath.Join(tmpDir, "main.pars")
	env2.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}
	env2.BasilCtx = evaluator.BuildTestBasilContext(map[string]string{}, nil, nil)
	env2.Set("@params", &evaluator.Dictionary{
		Pairs: map[string]ast.Expression{
			"orderBy": &ast.StringLiteral{Value: "age"},
		},
	})

	l2 := lexer.New(mainCode)
	p2 := parser.New(l2)
	program2 := p2.ParseProgram()
	if len(p2.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p2.Errors())
	}

	result2 := evaluator.Eval(program2, env2)
	if err, ok := result2.(*evaluator.Error); ok {
		t.Fatalf("second request error: %s", err.Inspect())
	}

	str2, ok := result2.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result2, result2.Inspect())
	}

	// Verify second request got "age" (not cached "name" from first request)
	if str2.Value != "age" {
		t.Errorf("second request: expected orderBy='age', got %s (params cached from first request)", str2.Value)
	}
}

// TestAtParamsDirectlyInModuleFunction tests that @params works directly
// inside a module function (without needing to import from @basil/http).
func TestAtParamsDirectlyInModuleFunction(t *testing.T) {
	evaluator.ClearModuleCache()

	// Create a module that uses @params directly in an exported function
	tmpDir := t.TempDir()
	modulePath := filepath.Join(tmpDir, "handler.pars")

	// Note: @params is used directly, not imported
	moduleCode := `export let getOrderBy = fn() { @params.orderBy ?? "none" }
`
	err := os.WriteFile(modulePath, []byte(moduleCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	mainCode := `
let handler = import @` + modulePath + `
handler.getOrderBy()
`

	// Create environment with @params
	env := evaluator.NewEnvironment()
	env.Filename = filepath.Join(tmpDir, "main.pars")
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}
	env.BasilCtx = evaluator.BuildTestBasilContext(map[string]string{}, nil, nil)
	env.Set("@params", &evaluator.Dictionary{
		Pairs: map[string]ast.Expression{
			"orderBy": &ast.StringLiteral{Value: "name"},
		},
	})

	l := lexer.New(mainCode)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	result := evaluator.Eval(program, env)
	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("error: %s", err.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}

	if str.Value != "name" {
		t.Errorf("expected orderBy='name', got %s (@params not available in module function)", str.Value)
	}
}

// TestAtParamsModuleScopeError tests that accessing @params at module scope
// produces a helpful error message explaining it's only available in functions.
func TestAtParamsModuleScopeError(t *testing.T) {
	evaluator.ClearModuleCache()

	// Create a module that tries to use @params at module scope
	tmpDir := t.TempDir()
	modulePath := filepath.Join(tmpDir, "bad_module.pars")

	// This should fail - @params at module scope
	moduleCode := `let order = @params.orderBy
export let getOrder = fn() { order }
`
	err := os.WriteFile(modulePath, []byte(moduleCode), 0644)
	if err != nil {
		t.Fatal(err)
	}

	mainCode := `
let handler = import @` + modulePath + `
handler.getOrder()
`

	// Create environment with @params set (simulating Basil server)
	env := evaluator.NewEnvironment()
	env.Filename = filepath.Join(tmpDir, "main.pars")
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}
	env.BasilCtx = evaluator.BuildTestBasilContext(map[string]string{}, nil, nil)
	env.Set("@params", &evaluator.Dictionary{
		Pairs: map[string]ast.Expression{
			"orderBy": &ast.StringLiteral{Value: "name"},
		},
	})

	l := lexer.New(mainCode)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	result := evaluator.Eval(program, env)

	// Should be an error
	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected error, got %T: %s", result, result.Inspect())
	}

	// Check error message mentions @params and module scope
	if errObj.Code != "UNDEF-0010" {
		t.Errorf("expected error code UNDEF-0010, got %s: %s", errObj.Code, errObj.Message)
	}

	if len(errObj.Hints) < 2 {
		t.Errorf("expected at least 2 hints, got %d", len(errObj.Hints))
	}
}
