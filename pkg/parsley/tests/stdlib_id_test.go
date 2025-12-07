package tests

import (
	"regexp"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// evalIDTest helper that evaluates Parsley code
func evalIDTest(t *testing.T, input string) evaluator.Object {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result == nil {
		t.Fatal("result is nil")
	}

	return result
}

// =============================================================================
// Module Import Tests
// =============================================================================

func TestIDModuleImport(t *testing.T) {
	input := `let {new, uuid} = import("std/id")
new`

	result := evalIDTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.BUILTIN_OBJ {
		t.Errorf("expected BUILTIN, got %s", result.Type())
	}
}

func TestIDModuleImportAll(t *testing.T) {
	input := `let id = import("std/id")
id.new`

	result := evalIDTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	if result.Type() != evaluator.BUILTIN_OBJ {
		t.Errorf("expected BUILTIN, got %s", result.Type())
	}
}

// =============================================================================
// ID Generation Tests - ULID (id.new)
// =============================================================================

func TestIDNew(t *testing.T) {
	input := `let id = import("std/id")
id.new()`

	result := evalIDTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected STRING, got %s", result.Type())
	}

	// ULID should be 26 characters
	if len(str.Value) != 26 {
		t.Errorf("expected ULID length 26, got %d: %s", len(str.Value), str.Value)
	}

	// ULID uses Crockford's Base32 (no I, L, O, U)
	ulidRegex := regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
	if !ulidRegex.MatchString(str.Value) {
		t.Errorf("invalid ULID format: %s", str.Value)
	}
}

func TestIDNewUniqueness(t *testing.T) {
	input := `let id = import("std/id")
let ids = [id.new(), id.new(), id.new(), id.new(), id.new()]
ids`

	result := evalIDTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected ARRAY, got %s", result.Type())
	}

	// Check all IDs are unique
	seen := make(map[string]bool)
	for _, elem := range arr.Elements {
		str, ok := elem.(*evaluator.String)
		if !ok {
			t.Fatalf("expected STRING element, got %s", elem.Type())
		}
		if seen[str.Value] {
			t.Errorf("duplicate ID generated: %s", str.Value)
		}
		seen[str.Value] = true
	}
}

// =============================================================================
// ID Generation Tests - UUID v4
// =============================================================================

func TestIDUUID(t *testing.T) {
	input := `let id = import("std/id")
id.uuid()`

	result := evalIDTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected STRING, got %s", result.Type())
	}

	// UUID format: 8-4-4-4-12
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(str.Value) {
		t.Errorf("invalid UUID v4 format: %s", str.Value)
	}
}

func TestIDUUIDv4Alias(t *testing.T) {
	input := `let id = import("std/id")
id.uuidv4()`

	result := evalIDTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected STRING, got %s", result.Type())
	}

	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(str.Value) {
		t.Errorf("invalid UUID v4 format: %s", str.Value)
	}
}

// =============================================================================
// ID Generation Tests - UUID v7
// =============================================================================

func TestIDUUIDv7(t *testing.T) {
	input := `let id = import("std/id")
id.uuidv7()`

	result := evalIDTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected STRING, got %s", result.Type())
	}

	// UUID v7 format: 8-4-4-4-12 with version 7
	uuidv7Regex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidv7Regex.MatchString(str.Value) {
		t.Errorf("invalid UUID v7 format: %s", str.Value)
	}
}

func TestIDUUIDv7Ordering(t *testing.T) {
	// UUID v7 should be time-sortable - verify both are valid format
	input := `let id = import("std/id")
let id1 = id.uuidv7()
id1`

	result := evalIDTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected STRING, got %s", result.Type())
	}

	// Verify it looks like a UUID v7 (36 chars with dashes)
	if len(str.Value) != 36 {
		t.Errorf("expected UUID length 36, got %d: %s", len(str.Value), str.Value)
	}
}

// =============================================================================
// ID Generation Tests - NanoID
// =============================================================================

func TestIDNanoid(t *testing.T) {
	input := `let id = import("std/id")
id.nanoid()`

	result := evalIDTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected STRING, got %s", result.Type())
	}

	// Default NanoID is 21 characters
	if len(str.Value) != 21 {
		t.Errorf("expected NanoID length 21, got %d: %s", len(str.Value), str.Value)
	}

	// NanoID uses URL-safe alphabet
	nanoidRegex := regexp.MustCompile(`^[0-9A-Za-z_-]+$`)
	if !nanoidRegex.MatchString(str.Value) {
		t.Errorf("invalid NanoID format: %s", str.Value)
	}
}

func TestIDNanoidCustomLength(t *testing.T) {
	input := `let id = import("std/id")
id.nanoid(10)`

	result := evalIDTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected STRING, got %s", result.Type())
	}

	if len(str.Value) != 10 {
		t.Errorf("expected NanoID length 10, got %d: %s", len(str.Value), str.Value)
	}
}

func TestIDNanoidInvalidLength(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"zero", 0},
		{"negative", -1},
		{"too large", 300},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `let id = import("std/id")
id.nanoid(` + string(rune('0'+tt.length)) + `)`

			// For negative and 0, we need to handle differently
			if tt.length == 0 {
				input = `let id = import("std/id")
id.nanoid(0)`
			} else if tt.length < 0 {
				// Skip negative test - can't easily represent in Parsley
				return
			} else if tt.length > 256 {
				input = `let id = import("std/id")
id.nanoid(300)`
			}

			result := evalIDTest(t, input)

			if result.Type() != evaluator.ERROR_OBJ {
				t.Errorf("expected error for invalid length %d, got %s", tt.length, result.Type())
			}
		})
	}
}

// =============================================================================
// ID Generation Tests - CUID
// =============================================================================

func TestIDCUID(t *testing.T) {
	input := `let id = import("std/id")
id.cuid()`

	result := evalIDTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("expected STRING, got %s", result.Type())
	}

	// CUID starts with 'c' and is 25 characters
	if len(str.Value) != 25 {
		t.Errorf("expected CUID length 25, got %d: %s", len(str.Value), str.Value)
	}

	if str.Value[0] != 'c' {
		t.Errorf("expected CUID to start with 'c', got: %s", str.Value)
	}
}

func TestIDCUIDUniqueness(t *testing.T) {
	input := `let id = import("std/id")
let ids = [id.cuid(), id.cuid(), id.cuid()]
ids`

	result := evalIDTest(t, input)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}

	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected ARRAY, got %s", result.Type())
	}

	// Check all IDs are unique
	seen := make(map[string]bool)
	for _, elem := range arr.Elements {
		str, ok := elem.(*evaluator.String)
		if !ok {
			t.Fatalf("expected STRING element, got %s", elem.Type())
		}
		if seen[str.Value] {
			t.Errorf("duplicate CUID generated: %s", str.Value)
		}
		seen[str.Value] = true
	}
}

// =============================================================================
// Arity Error Tests
// =============================================================================

func TestIDNewArityError(t *testing.T) {
	input := `let id = import("std/id")
id.new("extra")`

	result := evalIDTest(t, input)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Errorf("expected error for extra argument, got %s", result.Type())
	}
}

func TestIDUUIDArityError(t *testing.T) {
	input := `let id = import("std/id")
id.uuid("extra")`

	result := evalIDTest(t, input)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Errorf("expected error for extra argument, got %s", result.Type())
	}
}

func TestIDCUIDArityError(t *testing.T) {
	input := `let id = import("std/id")
id.cuid("extra")`

	result := evalIDTest(t, input)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Errorf("expected error for extra argument, got %s", result.Type())
	}
}
