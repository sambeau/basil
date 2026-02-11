package help

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestDescribeType tests type topic resolution
func TestDescribeType(t *testing.T) {
	tests := []struct {
		topic       string
		wantKind    string
		wantName    string
		wantMethods bool // expect at least one method
	}{
		{"string", "type", "string", true},
		{"integer", "type", "integer", true},
		{"float", "type", "float", true},
		{"money", "type", "money", true},
		{"array", "type", "array", true},
		{"dictionary", "type", "dictionary", true},
		{"datetime", "type", "datetime", true},
		{"duration", "type", "duration", true},
		{"path", "type", "path", true},
		{"url", "type", "url", true},
		{"table", "type", "table", true},
		{"regex", "type", "regex", true},
	}

	for _, tt := range tests {
		t.Run(tt.topic, func(t *testing.T) {
			result, err := DescribeTopic(tt.topic)
			if err != nil {
				t.Fatalf("DescribeTopic(%q) returned error: %v", tt.topic, err)
			}

			if result.Kind != tt.wantKind {
				t.Errorf("Kind = %q, want %q", result.Kind, tt.wantKind)
			}

			if result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}

			if tt.wantMethods && len(result.Methods) == 0 {
				t.Errorf("expected methods for type %q, got none", tt.topic)
			}
		})
	}
}

// TestDescribeTypeCaseInsensitive tests that type lookup is case insensitive
func TestDescribeTypeCaseInsensitive(t *testing.T) {
	tests := []string{"String", "STRING", "string", "Array", "ARRAY"}

	for _, topic := range tests {
		t.Run(topic, func(t *testing.T) {
			result, err := DescribeTopic(topic)
			if err != nil {
				t.Fatalf("DescribeTopic(%q) returned error: %v", topic, err)
			}

			if result.Kind != "type" {
				t.Errorf("Kind = %q, want 'type'", result.Kind)
			}
		})
	}
}

// TestDescribeModule tests module topic resolution
func TestDescribeModule(t *testing.T) {
	tests := []struct {
		topic           string
		wantName        string
		wantDescription bool // expect non-empty description
		wantExports     bool // expect some exports
	}{
		{"@std/math", "@std/math", true, true},
		{"@std/table", "@std/table", true, true},
		{"@std/valid", "@std/valid", true, true},
		{"@basil/http", "@basil/http", true, false}, // basil modules may not have documented exports
		{"@basil/auth", "@basil/auth", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.topic, func(t *testing.T) {
			result, err := DescribeTopic(tt.topic)
			if err != nil {
				t.Fatalf("DescribeTopic(%q) returned error: %v", tt.topic, err)
			}

			if result.Kind != "module" {
				t.Errorf("Kind = %q, want 'module'", result.Kind)
			}

			if result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}

			if tt.wantDescription && result.Description == "" {
				t.Error("expected non-empty description")
			}

			if tt.wantExports && len(result.Exports) == 0 {
				t.Error("expected some exports")
			}
		})
	}
}

// TestDescribeUnknownModule tests error handling for unknown modules
func TestDescribeUnknownModule(t *testing.T) {
	tests := []string{"@std/unknown", "@basil/nonexistent", "@other/module"}

	for _, topic := range tests {
		t.Run(topic, func(t *testing.T) {
			_, err := DescribeTopic(topic)
			if err == nil {
				t.Errorf("DescribeTopic(%q) should return error for unknown module", topic)
			}
		})
	}
}

// TestDescribeSpecialTopics tests the special keywords: builtins, operators, types
func TestDescribeSpecialTopics(t *testing.T) {
	t.Run("builtins", func(t *testing.T) {
		result, err := DescribeTopic("builtins")
		if err != nil {
			t.Fatalf("DescribeTopic('builtins') returned error: %v", err)
		}

		if result.Kind != "builtin-list" {
			t.Errorf("Kind = %q, want 'builtin-list'", result.Kind)
		}

		if len(result.Builtins) == 0 {
			t.Error("expected at least one builtin")
		}

		// Check that some known builtins are present
		found := make(map[string]bool)
		for _, b := range result.Builtins {
			found[b.Name] = true
		}

		expected := []string{"JSON", "CSV", "print", "fail"}
		for _, name := range expected {
			if !found[name] {
				t.Errorf("expected builtin %q to be in list", name)
			}
		}
	})

	t.Run("operators", func(t *testing.T) {
		result, err := DescribeTopic("operators")
		if err != nil {
			t.Fatalf("DescribeTopic('operators') returned error: %v", err)
		}

		if result.Kind != "operator-list" {
			t.Errorf("Kind = %q, want 'operator-list'", result.Kind)
		}

		if len(result.Operators) == 0 {
			t.Error("expected at least one operator")
		}

		// Check that some known operators are present
		found := make(map[string]bool)
		for _, op := range result.Operators {
			found[op.Symbol] = true
		}

		expected := []string{"+", "-", "*", "/", "==", "!=", "&&", "||", "++", "in"}
		for _, sym := range expected {
			if !found[sym] {
				t.Errorf("expected operator %q to be in list", sym)
			}
		}
	})

	t.Run("types", func(t *testing.T) {
		result, err := DescribeTopic("types")
		if err != nil {
			t.Fatalf("DescribeTopic('types') returned error: %v", err)
		}

		if result.Kind != "type-list" {
			t.Errorf("Kind = %q, want 'type-list'", result.Kind)
		}

		if len(result.TypeNames) == 0 {
			t.Error("expected at least one type name")
		}

		// Check that some known types are present
		found := make(map[string]bool)
		for _, name := range result.TypeNames {
			found[name] = true
		}

		expected := []string{"string", "integer", "float", "array", "dictionary"}
		for _, name := range expected {
			if !found[name] {
				t.Errorf("expected type %q to be in list", name)
			}
		}
	})
}

// TestDescribeBuiltinByName tests looking up specific builtins
func TestDescribeBuiltinByName(t *testing.T) {
	tests := []struct {
		topic        string
		wantName     string
		wantCategory string
	}{
		{"JSON", "JSON", "file"},
		{"CSV", "CSV", "file"},
		{"now", "now", "time"},
		{"fail", "fail", "control"},
		{"print", "print", "output"},
		{"toInt", "toInt", "conversion"},
	}

	for _, tt := range tests {
		t.Run(tt.topic, func(t *testing.T) {
			result, err := DescribeTopic(tt.topic)
			if err != nil {
				t.Fatalf("DescribeTopic(%q) returned error: %v", tt.topic, err)
			}

			if result.Kind != "builtin" {
				t.Errorf("Kind = %q, want 'builtin'", result.Kind)
			}

			if result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}

			if result.Category != tt.wantCategory {
				t.Errorf("Category = %q, want %q", result.Category, tt.wantCategory)
			}

			if result.Description == "" {
				t.Error("expected non-empty description")
			}
		})
	}
}

// TestDescribeUnknownTopic tests error handling for unknown topics
func TestDescribeUnknownTopic(t *testing.T) {
	tests := []string{"nonexistent", "foobar", "xyz123", "unknownType"}

	for _, topic := range tests {
		t.Run(topic, func(t *testing.T) {
			_, err := DescribeTopic(topic)
			if err == nil {
				t.Errorf("DescribeTopic(%q) should return error for unknown topic", topic)
			}

			// Error message should be helpful
			errMsg := err.Error()
			if !strings.Contains(errMsg, "unknown topic") {
				t.Errorf("error message should contain 'unknown topic', got: %s", errMsg)
			}
		})
	}
}

// TestDescribeEmptyTopic tests error handling for empty topic
func TestDescribeEmptyTopic(t *testing.T) {
	_, err := DescribeTopic("")
	if err == nil {
		t.Error("DescribeTopic('') should return error")
	}

	_, err = DescribeTopic("   ")
	if err == nil {
		t.Error("DescribeTopic('   ') should return error")
	}
}

// TestFormatTextType tests text formatting for type results
func TestFormatTextType(t *testing.T) {
	result, err := DescribeTopic("string")
	if err != nil {
		t.Fatalf("DescribeTopic('string') returned error: %v", err)
	}

	output := FormatText(result, 80)

	// Check for expected sections
	if !strings.Contains(output, "Type: string") {
		t.Error("output should contain 'Type: string'")
	}

	if !strings.Contains(output, "Methods:") {
		t.Error("output should contain 'Methods:' section")
	}

	// Check for a known method
	if !strings.Contains(output, "toUpper") {
		t.Error("output should contain 'toUpper' method")
	}
}

// TestFormatTextModule tests text formatting for module results
func TestFormatTextModule(t *testing.T) {
	result, err := DescribeTopic("@std/math")
	if err != nil {
		t.Fatalf("DescribeTopic('@std/math') returned error: %v", err)
	}

	output := FormatText(result, 80)

	// Check for expected sections
	if !strings.Contains(output, "Module: @std/math") {
		t.Error("output should contain 'Module: @std/math'")
	}

	if !strings.Contains(output, "Exports:") {
		t.Error("output should contain 'Exports:' section")
	}

	// Check for known exports
	if !strings.Contains(output, "PI") {
		t.Error("output should contain 'PI' constant")
	}
}

// TestFormatTextBuiltinList tests text formatting for builtin list
func TestFormatTextBuiltinList(t *testing.T) {
	result, err := DescribeTopic("builtins")
	if err != nil {
		t.Fatalf("DescribeTopic('builtins') returned error: %v", err)
	}

	output := FormatText(result, 80)

	// Check for expected structure
	if !strings.Contains(output, "Builtin Functions") {
		t.Error("output should contain 'Builtin Functions' header")
	}

	// Check for category headers
	if !strings.Contains(output, "File/Data Loading:") {
		t.Error("output should contain 'File/Data Loading:' category")
	}
}

// TestFormatTextOperatorList tests text formatting for operator list
func TestFormatTextOperatorList(t *testing.T) {
	result, err := DescribeTopic("operators")
	if err != nil {
		t.Fatalf("DescribeTopic('operators') returned error: %v", err)
	}

	output := FormatText(result, 80)

	// Check for expected structure
	if !strings.Contains(output, "Operators") {
		t.Error("output should contain 'Operators' header")
	}

	// Check for category headers
	if !strings.Contains(output, "Arithmetic:") {
		t.Error("output should contain 'Arithmetic:' category")
	}

	if !strings.Contains(output, "Comparison:") {
		t.Error("output should contain 'Comparison:' category")
	}
}

// TestFormatTextTypeList tests text formatting for type list
func TestFormatTextTypeList(t *testing.T) {
	result, err := DescribeTopic("types")
	if err != nil {
		t.Fatalf("DescribeTopic('types') returned error: %v", err)
	}

	output := FormatText(result, 80)

	// Check for expected structure
	if !strings.Contains(output, "Available Types") {
		t.Error("output should contain 'Available Types' header")
	}

	// Check for category groups
	if !strings.Contains(output, "Primitives:") {
		t.Error("output should contain 'Primitives:' group")
	}

	if !strings.Contains(output, "Collections:") {
		t.Error("output should contain 'Collections:' group")
	}
}

// TestFormatTextBuiltin tests text formatting for a single builtin
func TestFormatTextBuiltin(t *testing.T) {
	result, err := DescribeTopic("JSON")
	if err != nil {
		t.Fatalf("DescribeTopic('JSON') returned error: %v", err)
	}

	output := FormatText(result, 80)

	// Check for expected content
	if !strings.Contains(output, "JSON(") {
		t.Error("output should contain 'JSON(' signature")
	}

	if !strings.Contains(output, "Arity:") {
		t.Error("output should contain 'Arity:' field")
	}

	if !strings.Contains(output, "Category:") {
		t.Error("output should contain 'Category:' field")
	}
}

// TestFormatJSON tests JSON output formatting
func TestFormatJSON(t *testing.T) {
	tests := []string{"string", "builtins", "operators", "types", "@std/math", "JSON"}

	for _, topic := range tests {
		t.Run(topic, func(t *testing.T) {
			result, err := DescribeTopic(topic)
			if err != nil {
				t.Fatalf("DescribeTopic(%q) returned error: %v", topic, err)
			}

			jsonBytes, err := FormatJSON(result)
			if err != nil {
				t.Fatalf("FormatJSON returned error: %v", err)
			}

			// Verify it's valid JSON
			var parsed map[string]any
			if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
				t.Errorf("FormatJSON produced invalid JSON: %v", err)
			}

			// Verify expected fields
			if _, ok := parsed["kind"]; !ok {
				t.Error("JSON should contain 'kind' field")
			}

			if _, ok := parsed["name"]; !ok {
				t.Error("JSON should contain 'name' field")
			}
		})
	}
}

// TestFormatJSONRoundTrip tests that JSON output can be unmarshalled back
func TestFormatJSONRoundTrip(t *testing.T) {
	result, err := DescribeTopic("string")
	if err != nil {
		t.Fatalf("DescribeTopic('string') returned error: %v", err)
	}

	jsonBytes, err := FormatJSON(result)
	if err != nil {
		t.Fatalf("FormatJSON returned error: %v", err)
	}

	var parsed TopicResult
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if parsed.Kind != result.Kind {
		t.Errorf("Round-trip Kind mismatch: got %q, want %q", parsed.Kind, result.Kind)
	}

	if parsed.Name != result.Name {
		t.Errorf("Round-trip Name mismatch: got %q, want %q", parsed.Name, result.Name)
	}

	if len(parsed.Methods) != len(result.Methods) {
		t.Errorf("Round-trip Methods count mismatch: got %d, want %d", len(parsed.Methods), len(result.Methods))
	}
}

// TestModuleScopedDescriptions verifies that shared export names (e.g. "uuid", "string")
// get the correct module-scoped description, not a flat-map collision.
func TestModuleScopedDescriptions(t *testing.T) {
	tests := []struct {
		module     string
		export     string
		wantSubstr string // substring that must appear in the export's description
	}{
		{"@std/id", "uuid", "Generate UUID"},
		{"@std/valid", "uuid", "Check UUID format"},
		{"@std/schema", "string", "schema validator"},
		{"@std/valid", "string", "Check if value is string"},
		{"@std/schema", "email", "schema validator"},
		{"@std/valid", "email", "Check email format"},
	}

	for _, tt := range tests {
		t.Run(tt.module+"/"+tt.export, func(t *testing.T) {
			result, err := DescribeTopic(tt.module)
			if err != nil {
				t.Fatalf("DescribeTopic(%q) returned error: %v", tt.module, err)
			}

			var found bool
			for _, exp := range result.Exports {
				if exp.Name == tt.export {
					found = true
					if !strings.Contains(exp.Description, tt.wantSubstr) {
						t.Errorf("%s export %q description = %q, want substring %q",
							tt.module, tt.export, exp.Description, tt.wantSubstr)
					}
					break
				}
			}
			if !found {
				t.Errorf("%s missing export %q", tt.module, tt.export)
			}
		})
	}
}

// TestBasilModuleExports verifies that @basil modules report their exports
func TestBasilModuleExports(t *testing.T) {
	tests := []struct {
		module      string
		wantExports []string
	}{
		{"@basil/http", []string{"params", "request", "response", "route", "method"}},
		{"@basil/auth", []string{"session", "auth", "user"}},
	}

	for _, tt := range tests {
		t.Run(tt.module, func(t *testing.T) {
			result, err := DescribeTopic(tt.module)
			if err != nil {
				t.Fatalf("DescribeTopic(%q) returned error: %v", tt.module, err)
			}

			exportNames := make(map[string]bool)
			for _, exp := range result.Exports {
				exportNames[exp.Name] = true
			}

			for _, want := range tt.wantExports {
				if !exportNames[want] {
					t.Errorf("%s missing expected export %q", tt.module, want)
				}
			}
		})
	}
}

// TestMigratedVsUnmigratedTypes tests that both migrated and unmigrated types work
func TestMigratedVsUnmigratedTypes(t *testing.T) {
	// Migrated types (use registries)
	migrated := []string{"string", "integer", "float", "money"}

	// Unmigrated types (use TypeMethods)
	unmigrated := []string{"array", "dictionary", "datetime", "duration"}

	for _, typeName := range append(migrated, unmigrated...) {
		t.Run(typeName, func(t *testing.T) {
			result, err := DescribeTopic(typeName)
			if err != nil {
				t.Fatalf("DescribeTopic(%q) returned error: %v", typeName, err)
			}

			if result.Kind != "type" {
				t.Errorf("Kind = %q, want 'type'", result.Kind)
			}

			if len(result.Methods) == 0 {
				t.Errorf("expected methods for type %q", typeName)
			}
		})
	}
}
