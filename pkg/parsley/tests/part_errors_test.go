package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// TestPartErrorHandling verifies that Part errors are handled gracefully
func TestPartErrorHandling(t *testing.T) {
	t.Run("missing view parameter", func(t *testing.T) {
		// Trying to use a non-existent view should return an error
		input := `
let html = <Part src={@./test_fixtures/parts/counter.part} view="nonexistent" count={0}/>
html
`

		result := evalModule(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")

		// Should get an error because the view doesn't exist
		if result.Type() != evaluator.ERROR_OBJ {
			t.Fatalf("Expected error for non-existent view, got: %s", result.Inspect())
		}

		errObj := result.(*evaluator.Error)
		if !strings.Contains(errObj.Message, "nonexistent") {
			t.Errorf("Expected error message to mention non-existent view, got: %s", errObj.Message)
		}
	})

	t.Run("runtime error in view function", func(t *testing.T) {
		// If a view function throws an error, it should be caught
		// In practice, this would happen on the server when handling the Part request
		input := `
let html = <Part src={@./test_fixtures/parts/counter.part} view="default" count={0}/>
html
`

		result := evalModule(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")

		if result.Type() == evaluator.ERROR_OBJ {
			t.Fatalf("Eval error: %s", result.Inspect())
		}

		// The Part component renders successfully
		// Runtime errors happen when the server executes the view function
		html := result.(*evaluator.String).Value
		if !strings.Contains(html, "data-part-src") {
			t.Errorf("Expected Part wrapper")
		}
	})

	t.Run("missing src attribute", func(t *testing.T) {
		// Missing src should return an error
		input := `
let html = <Part view="default"/>
html
`

		result := evalModule(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")

		if result.Type() != evaluator.ERROR_OBJ {
			t.Errorf("Expected error for missing src attribute")
		}

		errObj := result.(*evaluator.Error)
		if !strings.Contains(errObj.Message, "src") {
			t.Errorf("Expected error message to mention missing src, got: %s", errObj.Message)
		}
	})

	t.Run("non-part file", func(t *testing.T) {
		// Using a non-.part file should return an error
		input := `
let html = <Part src={@./test_fixtures/import_module.pars} view="default"/>
html
`

		result := evalModule(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")

		if result.Type() != evaluator.ERROR_OBJ {
			t.Errorf("Expected error for non-.part file")
		}

		errObj := result.(*evaluator.Error)
		if !strings.Contains(errObj.Message, ".part") {
			t.Errorf("Expected error message to mention .part extension, got: %s", errObj.Message)
		}
	})
}

// TestPartLoadingState documents the loading state behavior
func TestPartLoadingState(t *testing.T) {
	// This test documents how the JavaScript handles loading states
	// Actual testing would require a browser environment

	input := `
let html = <Part src={@./test_fixtures/parts/counter.part} view="default" count={0}/>
html
`

	result := evalModule(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("Eval error: %s", result.Inspect())
	}

	html := result.(*evaluator.String).Value

	// The Part wrapper should NOT have the loading class initially
	if strings.Contains(html, "part-loading") {
		t.Errorf("Part should not have loading class in initial render")
	}

	// JavaScript behavior (documented here, not testable in Go):
	// 1. On click/submit, JS adds 'part-loading' class
	// 2. On successful fetch, JS updates innerHTML and removes 'part-loading'
	// 3. On fetch error, JS logs error and removes 'part-loading' (leaves old content)
	// 4. CSS can style .part-loading to show loading indicator
}

// TestPartErrorRecovery documents error recovery behavior
func TestPartErrorRecovery(t *testing.T) {
	// This test documents how Parts recover from errors
	// Actual testing would require a browser environment

	input := `
let html = <Part src={@./test_fixtures/parts/counter.part} view="default" count={5}/>
html
`

	result := evalModule(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("Eval error: %s", result.Inspect())
	}

	html := result.(*evaluator.String).Value

	// Verify the initial content is present
	if !strings.Contains(html, "Count: 5") {
		t.Errorf("Expected initial content")
	}

	// JavaScript error recovery behavior (documented):
	//
	// If a Part update fails (network error, 404, 500, etc.):
	// 1. The old content remains visible (no blank screen)
	// 2. The loading class is removed
	// 3. An error is logged to console for debugging
	// 4. The user can try the action again
	//
	// This prevents broken UI states where content disappears on error.
	// Users see the last working state until a successful update.
}
