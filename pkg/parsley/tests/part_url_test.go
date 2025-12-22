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

// TestPartURLGenerationSiteMode tests URL generation for Parts in site mode
// This addresses the bug where @~/parts/foo.part was generating /parts/parts/foo.part
func TestPartURLGenerationSiteMode(t *testing.T) {
	// Simulate a site mode setup:
	// Project root: /project
	// Site dir: /project/site
	// Handler: /project/site/admin/index.pars (route: /admin/)
	// Parts at project level: /project/parts/counter.part

	tests := []struct {
		name            string
		partPath        string // Path in Part src attribute
		expectedURLPart string // Expected substring in generated URL
		unexpectedPart  string // Should NOT appear in URL
	}{
		{
			name:            "project root part with @~/",
			partPath:        "@~/parts/counter.part",
			expectedURLPart: "/parts/counter.part",
			unexpectedPart:  "/admin/parts/", // Bug: was doubling the path
		},
		{
			name:            "relative part with @./",
			partPath:        "@./local.part",
			expectedURLPart: "/admin/local.part",
			unexpectedPart:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test environment simulating site mode
			// Project root
			tmpDir := t.TempDir()
			siteDir := filepath.Join(tmpDir, "site")
			adminDir := filepath.Join(siteDir, "admin")
			partsDir := filepath.Join(tmpDir, "parts")

			// Create directories
			if err := os.MkdirAll(adminDir, 0755); err != nil {
				t.Fatalf("Failed to create admin dir: %v", err)
			}
			if err := os.MkdirAll(partsDir, 0755); err != nil {
				t.Fatalf("Failed to create parts dir: %v", err)
			}

			// Create Part files
			counterPart := `export default = fn() { <div>"Counter"</div> }`
			if err := os.WriteFile(filepath.Join(partsDir, "counter.part"), []byte(counterPart), 0644); err != nil {
				t.Fatalf("Failed to write counter.part: %v", err)
			}

			localPart := `export default = fn() { <div>"Local"</div> }`
			if err := os.WriteFile(filepath.Join(adminDir, "local.part"), []byte(localPart), 0644); err != nil {
				t.Fatalf("Failed to write local.part: %v", err)
			}

			// Create handler file
			handlerPath := filepath.Join(adminDir, "index.pars")
			input := `<Part src={` + tt.partPath + `} view="default"/>`

			// Set up environment like site mode does
			l := lexer.New(input)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("Parse error: %s", p.Errors()[0])
			}

			env := evaluator.NewEnvironment()
			env.Filename = handlerPath
			env.RootPath = tmpDir       // Project root (parent of site/)
			env.HandlerPath = "/admin/" // Route path
			env.Security = &evaluator.SecurityPolicy{
				AllowExecuteAll: true,
			}

			// Evaluate
			result := evaluator.Eval(program, env)

			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("Eval error: %s", result.Inspect())
			}

			html := result.(*evaluator.String).Value
			t.Logf("Generated HTML: %s", html)

			// Check expected URL pattern
			if tt.expectedURLPart != "" && !strings.Contains(html, tt.expectedURLPart) {
				t.Errorf("Expected URL to contain %q, got: %s", tt.expectedURLPart, html)
			}

			// Check for bug regression (doubled paths)
			if tt.unexpectedPart != "" && strings.Contains(html, tt.unexpectedPart) {
				t.Errorf("URL should NOT contain %q (bug regression), got: %s", tt.unexpectedPart, html)
			}
		})
	}
}
