package tests

import (
	"testing"
	"time"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestSessionModule_GetSet(t *testing.T) {
	sm := evaluator.NewSessionModule(nil, nil, time.Hour)

	env := evaluator.NewEnvironment()
	env.Set("session", sm)

	tests := []struct {
		name     string
		code     string
		expected any
	}{
		{
			name:     "set and get string",
			code:     `session.set("name", "Alice"); session.get("name")`,
			expected: "Alice",
		},
		{
			name:     "set and get number",
			code:     `session.set("count", 42); session.get("count")`,
			expected: int64(42),
		},
		{
			name:     "get nonexistent returns null",
			code:     `session.get("missing")`,
			expected: nil,
		},
		{
			name:     "get with default",
			code:     `session.get("missing", "default")`,
			expected: "default",
		},
		{
			name:     "has returns true for existing",
			code:     `session.set("key", "value"); session.has("key")`,
			expected: true,
		},
		{
			name:     "has returns false for missing",
			code:     `session.has("nonexistent")`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh session for each test
			sm := evaluator.NewSessionModule(nil, nil, time.Hour)
			env := evaluator.NewEnvironment()
			env.Set("session", sm)

			result := evaluateCode(tt.code, env)
			if err, ok := result.(*evaluator.Error); ok {
				t.Fatalf("unexpected error: %s", err.Inspect())
			}

			got := toGoValue(result)
			if got != tt.expected {
				t.Errorf("expected %v (%T), got %v (%T)", tt.expected, tt.expected, got, got)
			}
		})
	}
}

func TestSessionModule_Delete(t *testing.T) {
	sm := evaluator.NewSessionModule(nil, nil, time.Hour)
	sm.Data["key"] = "value"

	env := evaluator.NewEnvironment()
	env.Set("session", sm)

	// Delete and verify
	result := evaluateCode(`session.delete("key"); session.has("key")`, env)
	if toGoValue(result) != false {
		t.Error("expected has() to return false after delete")
	}

	if !sm.Dirty {
		t.Error("expected session to be dirty after delete")
	}
}

func TestSessionModule_Clear(t *testing.T) {
	sm := evaluator.NewSessionModule(nil, nil, time.Hour)
	sm.Data["key1"] = "value1"
	sm.Data["key2"] = "value2"
	sm.Flash["msg"] = "hello"

	env := evaluator.NewEnvironment()
	env.Set("session", sm)

	evaluateCode(`session.clear()`, env)

	if len(sm.Data) != 0 {
		t.Errorf("expected empty data after clear, got %v", sm.Data)
	}
	if len(sm.Flash) != 0 {
		t.Errorf("expected empty flash after clear, got %v", sm.Flash)
	}
	if !sm.Dirty {
		t.Error("expected session to be dirty after clear")
	}
	if !sm.Cleared {
		t.Error("expected Cleared flag to be set")
	}
}

func TestSessionModule_All(t *testing.T) {
	sm := evaluator.NewSessionModule(nil, nil, time.Hour)
	sm.Data["name"] = "Alice"
	sm.Data["count"] = int64(42)

	env := evaluator.NewEnvironment()
	env.Set("session", sm)

	result := evaluateCode(`session.all()`, env)
	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T", result)
	}

	if len(dict.Pairs) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(dict.Pairs))
	}
}

func TestSessionModule_Flash(t *testing.T) {
	sm := evaluator.NewSessionModule(nil, nil, time.Hour)

	env := evaluator.NewEnvironment()
	env.Set("session", sm)

	// Set flash
	evaluateCode(`session.flash("success", "Operation completed!")`, env)

	// Verify flash was set
	if sm.Flash["success"] != "Operation completed!" {
		t.Errorf("expected flash to be set, got %v", sm.Flash)
	}
	if !sm.Dirty {
		t.Error("expected session to be dirty after flash")
	}
}

func TestSessionModule_GetFlash(t *testing.T) {
	sm := evaluator.NewSessionModule(nil, nil, time.Hour)
	sm.Flash["success"] = "Operation completed!"

	env := evaluator.NewEnvironment()
	env.Set("session", sm)

	// Get flash - should return the message
	result := evaluateCode(`session.getFlash("success")`, env)
	if str, ok := result.(*evaluator.String); !ok || str.Value != "Operation completed!" {
		t.Errorf("expected 'Operation completed!', got %v", result)
	}

	// Flash should be cleared
	if _, exists := sm.Flash["success"]; exists {
		t.Error("expected flash to be cleared after getFlash")
	}

	// Getting again should return null
	result = evaluateCode(`session.getFlash("success")`, env)
	if result != evaluator.NULL {
		t.Errorf("expected NULL on second getFlash, got %v", result)
	}
}

func TestSessionModule_GetAllFlash(t *testing.T) {
	sm := evaluator.NewSessionModule(nil, nil, time.Hour)
	sm.Flash["success"] = "Saved!"
	sm.Flash["info"] = "Note this"

	env := evaluator.NewEnvironment()
	env.Set("session", sm)

	result := evaluateCode(`session.getAllFlash()`, env)
	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T", result)
	}

	if len(dict.Pairs) != 2 {
		t.Errorf("expected 2 flash messages, got %d", len(dict.Pairs))
	}

	// Flash should be cleared
	if len(sm.Flash) != 0 {
		t.Error("expected flash to be cleared after getAllFlash")
	}
}

func TestSessionModule_HasFlash(t *testing.T) {
	sm := evaluator.NewSessionModule(nil, nil, time.Hour)

	env := evaluator.NewEnvironment()
	env.Set("session", sm)

	// Initially no flash
	result := evaluateCode(`session.hasFlash()`, env)
	if toGoValue(result) != false {
		t.Error("expected hasFlash() to return false initially")
	}

	// Add flash
	sm.Flash["test"] = "value"

	result = evaluateCode(`session.hasFlash()`, env)
	if toGoValue(result) != true {
		t.Error("expected hasFlash() to return true after adding flash")
	}
}

func TestSessionModule_Regenerate(t *testing.T) {
	sm := evaluator.NewSessionModule(nil, nil, time.Hour)

	env := evaluator.NewEnvironment()
	env.Set("session", sm)

	evaluateCode(`session.regenerate()`, env)

	if !sm.Dirty {
		t.Error("expected session to be dirty after regenerate")
	}
}

func TestSessionModule_MethodErrors(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{"get no args", `session.get()`},
		{"get too many args", `session.get("a", "b", "c")`},
		{"get non-string key", `session.get(123)`},
		{"set wrong arity", `session.set("key")`},
		{"set non-string key", `session.set(123, "value")`},
		{"delete no args", `session.delete()`},
		{"delete non-string", `session.delete(123)`},
		{"has no args", `session.has()`},
		{"has non-string", `session.has(123)`},
		{"flash wrong arity", `session.flash("key")`},
		{"flash non-string key", `session.flash(123, "msg")`},
		{"flash non-string msg", `session.flash("key", 123)`},
		{"getFlash no args", `session.getFlash()`},
		{"getFlash non-string", `session.getFlash(123)`},
		{"unknown method", `session.unknownMethod()`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := evaluator.NewSessionModule(nil, nil, time.Hour)
			env := evaluator.NewEnvironment()
			env.Set("session", sm)

			result := evaluateCode(tt.code, env)
			if _, ok := result.(*evaluator.Error); !ok {
				t.Errorf("expected error, got %T: %v", result, result)
			}
		})
	}
}

func TestSessionModule_DataTypes(t *testing.T) {
	sm := evaluator.NewSessionModule(nil, nil, time.Hour)
	env := evaluator.NewEnvironment()
	env.Set("session", sm)

	// Store various types
	evaluateCode(`session.set("bool", true)`, env)
	evaluateCode(`session.set("float", 3.14)`, env)
	evaluateCode(`session.set("array", [1, 2, 3])`, env)
	evaluateCode(`session.set("dict", {"a": 1, "b": 2})`, env)

	// Verify bool
	result := evaluateCode(`session.get("bool")`, env)
	if toGoValue(result) != true {
		t.Errorf("expected true, got %v", toGoValue(result))
	}

	// Verify float
	result = evaluateCode(`session.get("float")`, env)
	if toGoValue(result) != 3.14 {
		t.Errorf("expected 3.14, got %v", toGoValue(result))
	}

	// Verify array
	result = evaluateCode(`session.get("array")`, env)
	if arr, ok := result.(*evaluator.Array); !ok || len(arr.Elements) != 3 {
		t.Errorf("expected array with 3 elements, got %v", result)
	}

	// Verify dict
	result = evaluateCode(`session.get("dict")`, env)
	if dict, ok := result.(*evaluator.Dictionary); !ok || len(dict.Pairs) != 2 {
		t.Errorf("expected dict with 2 pairs, got %v", result)
	}
}

// Helper to evaluate Parsley code
func evaluateCode(code string, env *evaluator.Environment) evaluator.Object {
	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()
	return evaluator.Eval(program, env)
}

// Helper to convert Parsley object to Go value
func toGoValue(obj evaluator.Object) any {
	switch o := obj.(type) {
	case *evaluator.Null:
		return nil
	case *evaluator.Boolean:
		return o.Value
	case *evaluator.Integer:
		return o.Value
	case *evaluator.Float:
		return o.Value
	case *evaluator.String:
		return o.Value
	default:
		return obj.Inspect()
	}
}
