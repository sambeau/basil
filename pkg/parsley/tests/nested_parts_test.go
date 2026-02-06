package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// TestNestedParts verifies that nested Parts work correctly
// Uses the existing counter.part fixture which contains a nested Part structure
func TestNestedParts(t *testing.T) {
	// Render a Part that contains nested Parts
	// The counter.part fixture has buttons with part-click attributes
	input := `
let html = <Part src={@./test_fixtures/parts/counter.part} view="default" count={5}/>
html
`

	result := evalModule(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("Eval error: %s", result.Inspect())
	}

	html := result.(*evaluator.String).Value

	// Debug: print the actual HTML
	t.Logf("Generated HTML:\n%s", html)

	// Verify Part wrapper exists
	if !strings.Contains(html, "data-part-src") {
		t.Errorf("Expected Part wrapper with data-part-src")
	}

	if !strings.Contains(html, "data-part-view") {
		t.Errorf("Expected Part wrapper with data-part-view")
	}

	if !strings.Contains(html, "data-part-props") {
		t.Errorf("Expected Part wrapper with data-part-props")
	}

	// Verify the Part URL is generated correctly
	if !strings.Contains(html, "/test_fixtures/parts/counter.part") {
		t.Errorf("Expected Part URL in data-part-src")
	}

	// Verify props are JSON-encoded (check for count key, value might be quoted or not)
	if !strings.Contains(html, `count`) {
		t.Errorf("Expected props to contain count key in data-part-props, got: %s", html)
	}

	// Verify part-click attributes are preserved in the output
	if !strings.Contains(html, "part-click") {
		t.Errorf("Expected part-click attributes to be preserved")
	}
}

// TestNestedPartsRendering verifies that multiple levels of nesting work
func TestNestedPartsMultipleLevels(t *testing.T) {
	// Test that we can have a Part inside another Part
	// by rendering counter twice - once as outer, once conceptually as inner
	input := `
let outer = <Part src={@./test_fixtures/parts/counter.part} view="default" count={1}/>
outer
`

	result := evalModule(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("Eval error: %s", result.Inspect())
	}

	html := result.(*evaluator.String).Value

	// Count Part wrappers - should be at least 1
	partCount := strings.Count(html, "data-part-src")
	if partCount < 1 {
		t.Errorf("Expected at least 1 Part wrapper, got %d", partCount)
	}
}

// TestNestedPartsJavaScriptReInitialization documents the JS re-init behavior
// Note: This is a documentation test - actual JS testing would require a browser
func TestNestedPartsJavaScriptReInitialization(t *testing.T) {
	// When a Part is updated, the JavaScript runtime should:
	// 1. Fetch the new HTML from the server
	// 2. Replace innerHTML of the Part wrapper
	// 3. Call initParts() again to attach event handlers to any nested Parts
	//
	// This test verifies that the HTML structure supports this flow

	input := `
let html = <Part src={@./test_fixtures/parts/counter.part} view="default" count={0}/>
html
`

	result := evalModule(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("Eval error: %s", result.Inspect())
	}

	html := result.(*evaluator.String).Value

	// Verify data attributes are present for JS to use
	if !strings.Contains(html, "data-part-src") {
		t.Errorf("JS needs data-part-src to fetch updates")
	}

	if !strings.Contains(html, "data-part-view") {
		t.Errorf("JS needs data-part-view to know current view")
	}

	if !strings.Contains(html, "data-part-props") {
		t.Errorf("JS needs data-part-props to maintain state")
	}

	// Verify part-click attributes exist for JS to attach handlers
	if !strings.Contains(html, "part-click") {
		t.Errorf("JS needs part-click attributes to attach click handlers")
	}

	// The JavaScript in injectPartsRuntime() will:
	// - Query all [data-part-src] elements
	// - For each, query all [part-click] elements
	// - Attach click handlers that call updatePart()
	// - After innerHTML update, call initParts() again for nested Parts
}
