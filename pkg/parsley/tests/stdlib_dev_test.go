package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// mockDevLogWriter is a test double for DevLogWriter
type mockDevLogWriter struct {
	entries []mockLogEntry
	cleared []string
}

type mockLogEntry struct {
	route     string
	level     string
	filename  string
	line      int
	callRepr  string
	valueRepr string
}

func (m *mockDevLogWriter) LogFromEvaluator(route, level, filename string, line int, callRepr, valueRepr string) error {
	m.entries = append(m.entries, mockLogEntry{
		route:     route,
		level:     level,
		filename:  filename,
		line:      line,
		callRepr:  callRepr,
		valueRepr: valueRepr,
	})
	return nil
}

func (m *mockDevLogWriter) ClearLogs(route string) error {
	m.cleared = append(m.cleared, route)
	return nil
}

func TestDevLog(t *testing.T) {
	tests := []struct {
		input         string
		expectedRepr  string
		expectedLabel string
	}{
		{`dev.log(42)`, "42", ""},
		{`dev.log("hello")`, `hello`, ""}, // String.Inspect() returns value without quotes
		{`dev.log([1, 2, 3])`, "[1, 2, 3]", ""},
		{`dev.log({a: 1, b: 2})`, "", ""}, // Dict order may vary
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mock := &mockDevLogWriter{}
			env := evaluator.NewEnvironment()
			env.Filename = "test.pars"
			env.DevLog = mock // Set on environment - DevModule reads at call time
			devModule := evaluator.NewDevModule()
			env.Set("dev", devModule)

			result := testEval(tt.input, env)
			if isError(result) {
				t.Fatalf("got error: %s", result.Inspect())
			}

			if len(mock.entries) != 1 {
				t.Fatalf("expected 1 log entry, got %d", len(mock.entries))
			}

			entry := mock.entries[0]
			if entry.level != "info" {
				t.Errorf("expected level 'info', got '%s'", entry.level)
			}
			if entry.filename != "test.pars" {
				t.Errorf("expected filename 'test.pars', got '%s'", entry.filename)
			}
			if tt.expectedRepr != "" && entry.valueRepr != tt.expectedRepr {
				t.Errorf("expected valueRepr '%s', got '%s'", tt.expectedRepr, entry.valueRepr)
			}
		})
	}
}

func TestDevLogWithLabel(t *testing.T) {
	mock := &mockDevLogWriter{}
	env := evaluator.NewEnvironment()
	env.Filename = "test.pars"
	env.DevLog = mock
	devModule := evaluator.NewDevModule()
	env.Set("dev", devModule)

	result := testEval(`dev.log("users", [1, 2, 3])`, env)
	if isError(result) {
		t.Fatalf("got error: %s", result.Inspect())
	}

	if len(mock.entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(mock.entries))
	}

	entry := mock.entries[0]
	if entry.valueRepr != "[1, 2, 3]" {
		t.Errorf("expected valueRepr '[1, 2, 3]', got '%s'", entry.valueRepr)
	}
	// Call repr should include the label
	if entry.callRepr == "" {
		t.Error("expected non-empty callRepr")
	}
}

func TestDevLogWithLevel(t *testing.T) {
	tests := []struct {
		input         string
		expectedLevel string
	}{
		{`dev.log(42)`, "info"},
		{`dev.log(42, {level: "info"})`, "info"},
		{`dev.log(42, {level: "warn"})`, "warn"},
		{`dev.log(42, {level: "warning"})`, "warn"},
		{`dev.log("label", 42, {level: "warn"})`, "warn"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mock := &mockDevLogWriter{}
			env := evaluator.NewEnvironment()
			env.Filename = "test.pars"
			env.DevLog = mock
			devModule := evaluator.NewDevModule()
			env.Set("dev", devModule)

			result := testEval(tt.input, env)
			if isError(result) {
				t.Fatalf("got error: %s", result.Inspect())
			}

			if len(mock.entries) != 1 {
				t.Fatalf("expected 1 log entry, got %d", len(mock.entries))
			}

			entry := mock.entries[0]
			if entry.level != tt.expectedLevel {
				t.Errorf("expected level '%s', got '%s'", tt.expectedLevel, entry.level)
			}
		})
	}
}

func TestDevClearLog(t *testing.T) {
	mock := &mockDevLogWriter{}
	env := evaluator.NewEnvironment()
	env.DevLog = mock
	devModule := evaluator.NewDevModule()
	env.Set("dev", devModule)

	result := testEval(`dev.clearLog()`, env)
	if isError(result) {
		t.Fatalf("got error: %s", result.Inspect())
	}

	if len(mock.cleared) != 1 {
		t.Fatalf("expected 1 clear call, got %d", len(mock.cleared))
	}

	if mock.cleared[0] != "" {
		t.Errorf("expected default route (empty string), got '%s'", mock.cleared[0])
	}
}

func TestDevNoOpInProduction(t *testing.T) {
	// Create dev module with nil DevLog in env (production mode)
	env := evaluator.NewEnvironment()
	// env.DevLog is nil - simulates production mode
	devModule := evaluator.NewDevModule()
	env.Set("dev", devModule)

	tests := []string{
		`dev.log(42)`,
		`dev.log("label", 42)`,
		`dev.clearLog()`,
		`dev.logPage("users", 42)`,
		`dev.setLogRoute("users")`,
		`dev.clearLogPage("users")`,
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			result := testEval(tt, env)
			// Should not error, just return null
			if isError(result) {
				t.Errorf("expected no error in production mode for %s, got: %s", tt, result.Inspect())
			}
			if result.Type() != evaluator.NULL_OBJ {
				t.Errorf("expected NULL in production mode for %s, got: %s", tt, result.Type())
			}
		})
	}
}

func TestDevLogPage(t *testing.T) {
	mock := &mockDevLogWriter{}
	env := evaluator.NewEnvironment()
	env.Filename = "test.pars"
	env.DevLog = mock
	devModule := evaluator.NewDevModule()
	env.Set("dev", devModule)

	result := testEval(`dev.logPage("users", [1, 2, 3])`, env)
	if isError(result) {
		t.Fatalf("got error: %s", result.Inspect())
	}

	if len(mock.entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(mock.entries))
	}

	entry := mock.entries[0]
	if entry.route != "users" {
		t.Errorf("expected route 'users', got '%s'", entry.route)
	}
	if entry.valueRepr != "[1, 2, 3]" {
		t.Errorf("expected valueRepr '[1, 2, 3]', got '%s'", entry.valueRepr)
	}
}

func TestDevSetLogRoute(t *testing.T) {
	mock := &mockDevLogWriter{}
	env := evaluator.NewEnvironment()
	env.Filename = "test.pars"
	env.DevLog = mock
	devModule := evaluator.NewDevModule()
	env.Set("dev", devModule)

	// Set route then log
	result := testEval(`
		dev.setLogRoute("orders")
		dev.log(42)
	`, env)
	if isError(result) {
		t.Fatalf("got error: %s", result.Inspect())
	}

	if len(mock.entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(mock.entries))
	}

	entry := mock.entries[0]
	if entry.route != "orders" {
		t.Errorf("expected route 'orders', got '%s'", entry.route)
	}
}

func TestDevClearLogPage(t *testing.T) {
	mock := &mockDevLogWriter{}
	env := evaluator.NewEnvironment()
	env.DevLog = mock
	devModule := evaluator.NewDevModule()
	env.Set("dev", devModule)

	result := testEval(`dev.clearLogPage("users")`, env)
	if isError(result) {
		t.Fatalf("got error: %s", result.Inspect())
	}

	if len(mock.cleared) != 1 {
		t.Fatalf("expected 1 clear call, got %d", len(mock.cleared))
	}

	if mock.cleared[0] != "users" {
		t.Errorf("expected route 'users', got '%s'", mock.cleared[0])
	}
}

func TestDevRouteValidation(t *testing.T) {
	mock := &mockDevLogWriter{}
	env := evaluator.NewEnvironment()
	env.DevLog = mock
	devModule := evaluator.NewDevModule()
	env.Set("dev", devModule)

	tests := []struct {
		input       string
		shouldError bool
	}{
		{`dev.logPage("valid-route", 1)`, false},
		{`dev.logPage("valid_route", 1)`, false},
		{`dev.logPage("validRoute123", 1)`, false},
		{`dev.logPage("invalid/route", 1)`, true},
		{`dev.logPage("invalid route", 1)`, true},
		{`dev.logPage("invalid.route", 1)`, true},
		{`dev.setLogRoute("valid")`, false},
		{`dev.setLogRoute("invalid/route")`, true},
		{`dev.clearLogPage("valid")`, false},
		{`dev.clearLogPage("invalid/route")`, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// Reset mock
			mock.entries = nil
			mock.cleared = nil

			result := testEval(tt.input, env)
			gotError := isError(result)

			if tt.shouldError && !gotError {
				t.Errorf("expected error for %s", tt.input)
			}
			if !tt.shouldError && gotError {
				t.Errorf("unexpected error for %s: %s", tt.input, result.Inspect())
			}
		})
	}
}

func TestDevStdlibImport(t *testing.T) {
	mock := &mockDevLogWriter{}
	env := evaluator.NewEnvironment()
	env.Filename = "test.pars"
	env.DevLog = mock // Set on environment, not directly injected

	// Import dev module via stdlib
	result := testEval(`
		let {dev} = import @std/dev
		dev.log("imported", 42)
		42
	`, env)

	if isError(result) {
		t.Fatalf("got error: %s", result.Inspect())
	}

	if len(mock.entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(mock.entries))
	}

	entry := mock.entries[0]
	if entry.valueRepr != "42" {
		t.Errorf("expected valueRepr '42', got '%s'", entry.valueRepr)
	}
}

func TestDevStdlibImportNoOpInProduction(t *testing.T) {
	// Environment without DevLog set (production mode)
	env := evaluator.NewEnvironment()
	env.Filename = "test.pars"

	// Import dev module via stdlib - should work but be a no-op
	result := testEval(`
		let {dev} = import @std/dev
		dev.log("test", 123)
		"success"
	`, env)

	if isError(result) {
		t.Fatalf("expected no error in production mode, got: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok || str.Value != "success" {
		t.Errorf("expected 'success', got %s", result.Inspect())
	}
}

// Helper to evaluate with a custom environment
func testEval(input string, env *evaluator.Environment) evaluator.Object {
	l := lexer.NewWithFilename(input, "test.pars")
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return &evaluator.Error{Message: p.Errors()[0]}
	}
	return evaluator.Eval(program, env)
}
