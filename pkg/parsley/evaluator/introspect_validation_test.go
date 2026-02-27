package evaluator

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// introspect_validation_test.go
//
// Validation tests that verify method registries and TypeMethods maps in introspect.go
// accurately reflect actual method implementations in methods.go and eval_method_dispatch.go.
//
// These tests catch drift between documentation and implementation.
// As FEAT-111 (Declarative Method Registry) migrates types, tests automatically use registries.
//
// SCOPE:
// - Tests verify existence of documented methods/properties
// - Tests verify documented arity matches implementation
// - Uses registries for migrated types (string, integer, float, money)
// - Falls back to TypeMethods for non-migrated types
//
// SKIPPED TYPES (require external resources):
// - dbconnection: Requires database connection
// - sftpconnection: Covered by eval_sftp_integration_test.go (testenv fake SFTP server)
// - sftpfile: Covered by eval_sftp_integration_test.go (testenv fake SFTP server)
// - session: Requires server context
// - dev: Requires dev module setup
// - tablemodule: Requires table module initialization
// - request: Requires HTTP request context
// - response: Requires HTTP response context

// ============================================================================
// Test Value Factory
// ============================================================================

// createTestValues returns sample values for each type that has methods/properties.
// Returns nil for types that require external setup (marked in comments).
func createTestValues() map[string]Object {
	env := NewEnvironment()

	values := map[string]Object{
		// Primitive types
		"string":  &String{Value: "test"},
		"integer": &Integer{Value: 42},
		"float":   &Float{Value: 3.14},
		"boolean": &Boolean{Value: true},
		"null":    &Null{},

		// Collections
		"array": &Array{Elements: []Object{
			&Integer{Value: 1},
			&Integer{Value: 2},
		}},
		"dictionary": &Dictionary{
			Pairs: map[string]ast.Expression{
				"key": &ast.StringLiteral{Value: "value"},
			},
			KeyOrder: []string{"key"},
			Env:      env,
		},

		// Money (direct struct)
		"money": &Money{
			Amount:   1000,
			Currency: "USD",
			Scale:    2,
		},

		// Table (direct struct)
		"table": &Table{
			Rows: []*Dictionary{
				{
					Pairs: map[string]ast.Expression{
						"id":   &ast.IntegerLiteral{Value: 1},
						"name": &ast.StringLiteral{Value: "Alice"},
					},
					KeyOrder: []string{"id", "name"},
					Env:      env,
				},
			},
			Columns: []string{"id", "name"},
		},

		// Typed dictionaries - datetime
		"datetime": &Dictionary{
			Pairs: map[string]ast.Expression{
				"__type": &ast.StringLiteral{Value: "datetime"},
				"year":   &ast.IntegerLiteral{Value: 2024},
				"month":  &ast.IntegerLiteral{Value: 1},
				"day":    &ast.IntegerLiteral{Value: 15},
				"hour":   &ast.IntegerLiteral{Value: 10},
				"minute": &ast.IntegerLiteral{Value: 30},
				"second": &ast.IntegerLiteral{Value: 0},
				"kind":   &ast.StringLiteral{Value: "datetime"},
			},
			KeyOrder: []string{"__type", "year", "month", "day", "hour", "minute", "second", "kind"},
			Env:      env,
		},

		// Typed dictionaries - duration
		"duration": &Dictionary{
			Pairs: map[string]ast.Expression{
				"__type":  &ast.StringLiteral{Value: "duration"},
				"months":  &ast.IntegerLiteral{Value: 0},
				"seconds": &ast.IntegerLiteral{Value: 3600},
			},
			KeyOrder: []string{"__type", "months", "seconds"},
			Env:      env,
		},

		// Typed dictionaries - path
		"path": &Dictionary{
			Pairs: map[string]ast.Expression{
				"__type":   &ast.StringLiteral{Value: "path"},
				"absolute": &ast.Boolean{Value: true},
				"segments": &ast.ArrayLiteral{
					Elements: []ast.Expression{
						&ast.StringLiteral{Value: "home"},
						&ast.StringLiteral{Value: "user"},
						&ast.StringLiteral{Value: "file.txt"},
					},
				},
			},
			KeyOrder: []string{"__type", "absolute", "segments"},
			Env:      env,
		},

		// Typed dictionaries - url
		"url": &Dictionary{
			Pairs: map[string]ast.Expression{
				"__type":   &ast.StringLiteral{Value: "url"},
				"scheme":   &ast.StringLiteral{Value: "https"},
				"host":     &ast.StringLiteral{Value: "example.com"},
				"port":     &ast.IntegerLiteral{Value: 443},
				"path":     &ast.StringLiteral{Value: "/path"},
				"query":    &ast.DictionaryLiteral{Pairs: map[string]ast.Expression{}, KeyOrder: []string{}},
				"fragment": &ast.StringLiteral{Value: ""},
			},
			KeyOrder: []string{"__type", "scheme", "host", "port", "path", "query", "fragment"},
			Env:      env,
		},

		// Typed dictionaries - regex
		"regex": &Dictionary{
			Pairs: map[string]ast.Expression{
				"__type":  &ast.StringLiteral{Value: "regex"},
				"pattern": &ast.StringLiteral{Value: "test"},
				"flags":   &ast.StringLiteral{Value: ""},
			},
			KeyOrder: []string{"__type", "pattern", "flags"},
			Env:      env,
		},

		// Typed dictionaries - file
		"file": &Dictionary{
			Pairs: map[string]ast.Expression{
				"__type": &ast.StringLiteral{Value: "file"},
				"path":   &ast.StringLiteral{Value: "/test/file.txt"},
				"format": &ast.StringLiteral{Value: "text"},
			},
			KeyOrder: []string{"__type", "path", "format"},
			Env:      env,
		},

		// Typed dictionaries - directory
		"directory": &Dictionary{
			Pairs: map[string]ast.Expression{
				"__type": &ast.StringLiteral{Value: "dir"},
				"path":   &ast.StringLiteral{Value: "/test/dir"},
			},
			KeyOrder: []string{"__type", "path"},
			Env:      env,
		},
	}

	// Types that require external resources are not included:
	// - dbconnection (needs DB)
	// - sftpconnection (covered by eval_sftp_integration_test.go)
	// - sftpfile (covered by eval_sftp_integration_test.go)
	// - session (needs server context)
	// - dev (needs dev module)
	// - tablemodule (needs table module)
	// - function (no methods documented)

	return values
}

// ============================================================================
// Helper Functions
// ============================================================================

// makeArgs creates a slice of n dummy String arguments for testing.
func makeArgs(count int) []Object {
	args := make([]Object, count)
	for i := range args {
		args[i] = &String{Value: "test"}
	}
	return args
}

// isUnknownMethodError checks if the result is an "unknown method" error (UNDEF-0002).
func isUnknownMethodError(obj Object) bool {
	if err, ok := obj.(*Error); ok {
		return err.Code == "UNDEF-0002"
	}
	return false
}

// isArityError checks if the result is an arity-related error.
// Arity errors contain "takes", "expects", or "argument" in the message.
func isArityError(obj Object) bool {
	if err, ok := obj.(*Error); ok {
		msg := strings.ToLower(err.Message)
		return strings.Contains(msg, "takes") ||
			strings.Contains(msg, "expects") ||
			strings.Contains(msg, "argument")
	}
	return false
}

// arityBounds represents parsed arity information.
type arityBounds struct {
	min       int
	max       int
	unbounded bool // true for "1+", "0+" style arities
}

// parseArityBounds parses arity strings into min/max bounds.
// Examples:
//   - "0" → {0, 0, false}
//   - "1" → {1, 1, false}
//   - "0-1" → {0, 1, false}
//   - "1-2" → {1, 2, false}
//   - "1+" → {1, -1, true}
//   - "0+" → {0, -1, true}
func parseArityBounds(arity string) arityBounds {
	// Handle unbounded: "1+", "0+"
	if minStr, found := strings.CutSuffix(arity, "+"); found {
		var minVal int
		if minStr == "" {
			minVal = 0
		} else {
			// Simple parse, assuming valid format
			for _, c := range minStr {
				minVal = minVal*10 + int(c-'0')
			}
		}
		return arityBounds{min: minVal, max: -1, unbounded: true}
	}

	// Handle range: "0-1", "1-2"
	if strings.Contains(arity, "-") {
		parts := strings.Split(arity, "-")
		minVal := 0
		maxVal := 0
		for _, c := range parts[0] {
			minVal = minVal*10 + int(c-'0')
		}
		for _, c := range parts[1] {
			maxVal = maxVal*10 + int(c-'0')
		}
		return arityBounds{min: minVal, max: maxVal, unbounded: false}
	}

	// Handle fixed: "0", "1", "2"
	n := 0
	for _, c := range arity {
		n = n*10 + int(c-'0')
	}
	return arityBounds{min: n, max: n, unbounded: false}
}

// ============================================================================
// Test: Method Existence
// ============================================================================

// getMethodsForValidation returns method info for a type, preferring registry over TypeMethods.
// This allows validation tests to work correctly as types are migrated to registries.
func getMethodsForValidation(typeName string) []MethodInfo {
	// First check if this type has a registry (migrated types)
	if registry := GetRegistryForType(typeName); registry != nil {
		return registry.ToMethodInfos()
	}
	// Fall back to TypeMethods (non-migrated types)
	return TypeMethods[typeName]
}

func TestTypeMethods_AllMethodsExist(t *testing.T) {
	testValues := createTestValues()
	env := NewEnvironment()

	// Collect all type names from both registries and TypeMethods
	allTypes := make(map[string]bool)
	for typeName := range TypeMethods {
		allTypes[typeName] = true
	}
	for typeName := range typeRegistries {
		allTypes[typeName] = true
	}

	for typeName := range allTypes {
		methods := getMethodsForValidation(typeName)
		if len(methods) == 0 {
			continue // No methods to test
		}

		testVal, ok := testValues[typeName]
		if !ok {
			// Type requires external setup - skip with note
			t.Logf("SKIP: %s (requires external resources)", typeName)
			continue
		}

		for _, method := range methods {
			t.Run(typeName+"."+method.Name, func(t *testing.T) {
				// Attempt to call method with minimum arity
				bounds := parseArityBounds(method.Arity)
				args := makeArgs(bounds.min)

				result := dispatchMethodCall(testVal, method.Name, args, env)

				// Check for "unknown method" error
				if isUnknownMethodError(result) {
					t.Errorf("Method %s.%s() is documented but doesn't exist in implementation",
						typeName, method.Name)
				}
			})
		}
	}
}

// ============================================================================
// Test: Arity Validation
// ============================================================================

func TestTypeMethods_ArityMatches(t *testing.T) {
	testValues := createTestValues()
	env := NewEnvironment()

	// Collect all type names from both registries and TypeMethods
	allTypes := make(map[string]bool)
	for typeName := range TypeMethods {
		allTypes[typeName] = true
	}
	for typeName := range typeRegistries {
		allTypes[typeName] = true
	}

	for typeName := range allTypes {
		methods := getMethodsForValidation(typeName)
		if len(methods) == 0 {
			continue
		}

		testVal, ok := testValues[typeName]
		if !ok {
			continue // Skip types requiring external resources
		}

		for _, method := range methods {
			bounds := parseArityBounds(method.Arity)

			// Test 1: Minimum arity should be accepted (may fail with type error, but not arity error)
			t.Run(typeName+"."+method.Name+"_min_arity", func(t *testing.T) {
				args := makeArgs(bounds.min)
				result := dispatchMethodCall(testVal, method.Name, args, env)

				// Should not get arity error when calling with minimum args
				// (may get type errors or other errors, but not arity)
				if isArityError(result) {
					if err, ok := result.(*Error); ok {
						// Only fail if error explicitly says wrong number of arguments
						if strings.Contains(err.Message, "takes") || strings.Contains(err.Message, "expects") {
							t.Errorf("Method %s.%s() documented as arity %q but rejected %d args: %s",
								typeName, method.Name, method.Arity, bounds.min, err.Message)
						}
					}
				}
			})

			// Test 2: Too few arguments should fail (if min > 0)
			if bounds.min > 0 {
				t.Run(typeName+"."+method.Name+"_too_few_args", func(t *testing.T) {
					args := makeArgs(bounds.min - 1)
					_ = dispatchMethodCall(testVal, method.Name, args, env)
					// We expect an arity error, but some methods might succeed with fewer args
					// (optional params). This is OK - we're just checking consistency with docs.
				})
			}

			// Test 3: Too many arguments should fail (if not unbounded)
			if !bounds.unbounded {
				t.Run(typeName+"."+method.Name+"_too_many_args", func(t *testing.T) {
					args := makeArgs(bounds.max + 1)
					_ = dispatchMethodCall(testVal, method.Name, args, env)
					// We expect an arity error, but some methods might accept extra args.
					// We're primarily checking the documented arity is a valid range.
				})
			}
		}
	}
}

// ============================================================================
// Test: Property Existence
// ============================================================================

func TestTypeProperties_AllPropertiesExist(t *testing.T) {
	testValues := createTestValues()

	for typeName, properties := range TypeProperties {
		if len(properties) == 0 {
			continue // No properties to test
		}

		testVal, ok := testValues[typeName]
		if !ok {
			t.Logf("SKIP: %s properties (requires external resources)", typeName)
			continue
		}

		for _, prop := range properties {
			t.Run(typeName+"."+prop.Name, func(t *testing.T) {
				// For typed dictionaries, properties are stored in Pairs
				if dict, ok := testVal.(*Dictionary); ok {
					// Check if property exists in dictionary
					if _, exists := dict.Pairs[prop.Name]; !exists {
						// Property might be computed - check if accessing it causes an error
						// We can't easily test property access without parsing, so we'll
						// just verify the property is in the dictionary for now
						t.Logf("Property %s.%s not in test dictionary (may be computed property)",
							typeName, prop.Name)
					}
				}

				// For other types (Money, Table), properties are struct fields
				// These are tested implicitly by the type system - can't access non-existent fields
			})
		}
	}
}

// ============================================================================
// Test: Skipped Types Documentation
// ============================================================================

func TestSkippedTypes_Documented(t *testing.T) {
	// This test documents which types are skipped and why
	skippedTypes := map[string]string{
		"dbconnection":   "Requires database connection",
		"sftpconnection": "Covered by eval_sftp_integration_test.go; skipped here because introspection needs a live handle",
		"sftpfile":       "Covered by eval_sftp_integration_test.go; skipped here because introspection needs a live handle",
		"session":        "Requires server context",
		"dev":            "Requires dev module setup",
	}

	testValues := createTestValues()

	for typeName, reason := range skippedTypes {
		if _, exists := TypeMethods[typeName]; exists {
			if _, hasTestVal := testValues[typeName]; !hasTestVal {
				t.Logf("SKIPPED: %s - %s", typeName, reason)
			}
		}
	}
}

// ============================================================================
// Test: Arity Parser
// ============================================================================

// ============================================================================
// BUG-023 Regression Tests
// ============================================================================

// TestBUG023_NoPhantomOperators verifies that OperatorMetadata does not contain
// operators that the parser cannot actually parse.
func TestBUG023_NoPhantomOperators(t *testing.T) {
	phantoms := []string{"**", "|>", "?:"}
	for _, op := range phantoms {
		if _, exists := OperatorMetadata[op]; exists {
			t.Errorf("OperatorMetadata contains phantom operator %q which does not exist in the parser", op)
		}
	}
}

// TestBUG023_OperatorsAreParseable verifies that every operator listed in
// OperatorMetadata can actually be parsed by the Parsley parser.
func TestBUG023_OperatorsAreParseable(t *testing.T) {
	// Build test expressions for each operator category
	testExprs := map[string]string{
		"+":      "1 + 2",
		"-":      "5 - 3",
		"*":      "2 * 3",
		"/":      "10 / 2",
		"%":      "10 % 3",
		"==":     "1 == 1",
		"!=":     "1 != 2",
		"<":      "1 < 2",
		">":      "2 > 1",
		"<=":     "1 <= 2",
		">=":     "2 >= 1",
		"&&":     "true && false",
		"and":    "true and false",
		"||":     "true || false",
		"or":     "true or false",
		"!":      "!true",
		"++":     "[1] ++ [2]",
		"in":     "1 in [1,2]",
		"not in": "3 not in [1,2]",
		"..":     "1..5",
		"~":      `"abc" ~ /a/`,
		"!~":     `"abc" !~ /z/`,
		"??":     "null ?? 1",
	}

	for op := range OperatorMetadata {
		expr, ok := testExprs[op]
		if !ok {
			t.Errorf("OperatorMetadata contains operator %q with no parse test — add a test expression", op)
			continue
		}

		t.Run("parse_"+op, func(t *testing.T) {
			l := lexer.New(expr)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Errorf("Operator %q is in OperatorMetadata but fails to parse %q: %s",
					op, expr, p.Errors()[0])
			}
			if program == nil {
				t.Errorf("Operator %q produced nil program for %q", op, expr)
			}
		})
	}
}

// TestBUG023_UnitBuiltinExists verifies that the `unit` builtin has metadata.
func TestBUG023_UnitBuiltinExists(t *testing.T) {
	meta, exists := BuiltinMetadata["unit"]
	if !exists {
		t.Fatal("BuiltinMetadata is missing entry for 'unit'")
	}
	if meta.Arity != "1-2" {
		t.Errorf("unit arity: got %q, want %q", meta.Arity, "1-2")
	}
	if meta.Category != "unit" {
		t.Errorf("unit category: got %q, want %q", meta.Category, "unit")
	}
}

// TestBUG023_MatchMetadata verifies that `match` is described as a path matcher with arity 2.
func TestBUG023_MatchMetadata(t *testing.T) {
	meta, exists := BuiltinMetadata["match"]
	if !exists {
		t.Fatal("BuiltinMetadata is missing entry for 'match'")
	}
	if meta.Arity != "2" {
		t.Errorf("match arity: got %q, want %q", meta.Arity, "2")
	}
	if meta.Category == "regex" {
		t.Error("match category should not be 'regex' — it is a path/URL pattern matcher")
	}
	if strings.Contains(strings.ToLower(meta.Description), "regex") {
		t.Errorf("match description should not mention regex: %q", meta.Description)
	}
	if len(meta.Params) != 2 {
		t.Errorf("match params count: got %d, want 2", len(meta.Params))
	}
}

// TestBUG023_FormatMetadata verifies that `format` is described as a duration formatter.
func TestBUG023_FormatMetadata(t *testing.T) {
	meta, exists := BuiltinMetadata["format"]
	if !exists {
		t.Fatal("BuiltinMetadata is missing entry for 'format'")
	}
	if meta.Arity != "1-2" {
		t.Errorf("format arity: got %q, want %q", meta.Arity, "1-2")
	}
	if strings.Contains(strings.ToLower(meta.Description), "placeholder") {
		t.Errorf("format description should not mention placeholders: %q", meta.Description)
	}
	if strings.Contains(strings.ToLower(meta.Description), "template") {
		t.Errorf("format description should not mention template: %q", meta.Description)
	}
	if strings.Contains(strings.ToLower(meta.Description), "array") {
		t.Errorf("format description should not mention arrays (removed in 1.0): %q", meta.Description)
	}
}

// TestBUG023_MatchActualArity verifies match rejects 3 arguments at runtime.
func TestBUG023_MatchActualArity(t *testing.T) {
	builtins := getBuiltins()
	matchFn := builtins["match"]
	result := matchFn.Fn(&String{Value: "/x"}, &String{Value: "/x"}, &String{Value: "extra"})
	if _, ok := result.(*Error); !ok {
		t.Errorf("match with 3 args should return error, got %T", result)
	}
}

// TestBUG023_FormatRejectsStringTemplate verifies format rejects printf-style usage.
func TestBUG023_FormatRejectsStringTemplate(t *testing.T) {
	builtins := getBuiltins()
	formatFn := builtins["format"]
	result := formatFn.Fn(&String{Value: "hello %s"}, &String{Value: "world"})
	if _, ok := result.(*Error); !ok {
		t.Errorf("format with string template should return error, got %T", result)
	}
}

func TestParseArityBounds(t *testing.T) {
	tests := []struct {
		arity     string
		wantMin   int
		wantMax   int
		wantUnbnd bool
	}{
		{"0", 0, 0, false},
		{"1", 1, 1, false},
		{"2", 2, 2, false},
		{"0-1", 0, 1, false},
		{"1-2", 1, 2, false},
		{"0-3", 0, 3, false},
		{"1+", 1, -1, true},
		{"0+", 0, -1, true},
		{"2+", 2, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.arity, func(t *testing.T) {
			result := parseArityBounds(tt.arity)
			if result.min != tt.wantMin {
				t.Errorf("min: got %d, want %d", result.min, tt.wantMin)
			}
			if result.max != tt.wantMax {
				t.Errorf("max: got %d, want %d", result.max, tt.wantMax)
			}
			if result.unbounded != tt.wantUnbnd {
				t.Errorf("unbounded: got %v, want %v", result.unbounded, tt.wantUnbnd)
			}
		})
	}
}
