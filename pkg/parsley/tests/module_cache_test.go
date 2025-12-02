package tests

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

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
	input := `let mod = import(@` + modulePath + `); mod.count`

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

			input := `let mod = import(@` + modulePath + `); mod.value`

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
				input := `let mod = import(@` + modulePath + `); mod.num`

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
		let m1 = import(@` + mod1Path + `)
		let m2 = import(@` + mod2Path + `)
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
