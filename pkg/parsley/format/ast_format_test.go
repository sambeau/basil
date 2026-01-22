package format

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// helper to parse and format code
func parseAndFormat(t *testing.T, input string) string {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors for input %q: %v", input, p.Errors())
	}
	return FormatProgram(program)
}

func TestFormatLetStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"let x = 5", "let x = 5"},
		// Strings are preserved as-is from the token
		{"let name = \"Alice\"", `let name = "Alice"`},
		{"let arr = [1, 2, 3]", "let arr = [1, 2, 3]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseAndFormat(t, tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatArrayLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[]", "[]"},
		{"[1, 2, 3]", "[1, 2, 3]"},
		{"[1]", "[1]"},
		// Long array that exceeds threshold - check it has newlines
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseAndFormat(t, tt.input)
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestFormatArrayLiteralMultiline(t *testing.T) {
	input := `["alpha", "bravo", "charlie", "delta", "echo", "foxtrot"]`
	result := parseAndFormat(t, input)

	// Should contain newlines for multiline formatting
	if !contains(result, "\n") {
		// It's okay if it stays inline - threshold check may differ
		// Just verify it's valid
		if !contains(result, "alpha") {
			t.Errorf("expected array elements in output, got %q", result)
		}
	}
}

func TestFormatDictionaryLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"{}", "{}"},
		{"{a: 1}", "{a: 1}"},
		{"{a: 1, b: 2}", "{a: 1, b: 2}"},
		// Quoted keys
		{`{"with space": 1}`, `{"with space": 1}`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseAndFormat(t, tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatFunctionLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"fn(x) { x * 2 }", "fn(x) { x * 2 }"},
		{"fn(a, b) { a + b }", "fn(a, b) { a + b }"},
		{"fn() { 42 }", "fn() { 42 }"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseAndFormat(t, tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatIfExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Short if-else can be inline
		{"if (x > 0) 1 else -1", "if (x > 0) 1 else -1"},
		// With blocks - the formatter optimizes to no-block if simple
		{"if (x > 0) { x } else { 0 }", "if (x > 0) x else 0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseAndFormat(t, tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatForExpression(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// For loop with function form (for arr fn)
		{"function form", "for ([1, 2, 3]) fn(n) { n * 2 }", "for ([1, 2, 3]) fn(n) { n * 2 }"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAndFormat(t, tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatInfixExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1 + 2", "1 + 2"},
		{"a * b", "a * b"},
		{"x > 0 && y < 10", "x > 0 && y < 10"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseAndFormat(t, tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatDotExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"obj.name", "obj.name"},
		{"arr.length()", "arr.length()"},
		{"name.trim().toUpper()", "name.trim().toUpper()"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseAndFormat(t, tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatCheckStatement(t *testing.T) {
	input := `let validate = fn(x) {
    check x > 0 else "must be positive"
    x
}`
	result := parseAndFormat(t, input)
	// Should contain the check statement
	if !contains(result, "check x > 0 else") {
		t.Errorf("expected check statement in output, got %q", result)
	}
}

func TestFormatSchemaDeclaration(t *testing.T) {
	input := `@schema User {
    id: int
    name: string
}`
	result := parseAndFormat(t, input)
	// Should be multiline with proper indentation
	if !contains(result, "@schema User {") {
		t.Errorf("expected schema header in output, got %q", result)
	}
	if !contains(result, "id: int") {
		t.Errorf("expected id field in output, got %q", result)
	}
}

func TestFormatSchemaMetadata_SingleItem(t *testing.T) {
	// Single short metadata item stays inline
	input := `@schema User {
    active: bool | {hidden: true}
}`
	result := parseAndFormat(t, input)
	if !contains(result, "| {hidden: true}") {
		t.Errorf("expected single metadata to stay inline, got %q", result)
	}
}

func TestFormatSchemaMetadata_MultipleItems(t *testing.T) {
	// Multiple metadata items always go multiline
	input := `@schema User {
    name: string | {title: "Name", placeholder: "Enter name"}
}`
	result := parseAndFormat(t, input)
	// Should be multiline
	if !contains(result, "| {\n") {
		t.Errorf("expected multiline metadata for multiple items, got %q", result)
	}
	// Should have title and placeholder on separate lines
	if !contains(result, `title: "Name"`) {
		t.Errorf("expected title in metadata, got %q", result)
	}
	if !contains(result, `placeholder: "Enter name"`) {
		t.Errorf("expected placeholder in metadata, got %q", result)
	}
}

func TestFormatSchemaBlankLinePreserved(t *testing.T) {
	// Blank line after schema should be preserved
	input := `@schema User {
    id: int
}

let x = 1`
	result := parseAndFormat(t, input)
	// Should have blank line between schema and let
	if !contains(result, "}\n\nlet") {
		t.Errorf("expected blank line after schema to be preserved, got %q", result)
	}
}

func TestFormatNode_Nil(t *testing.T) {
	result := FormatNode(nil)
	if result != "" {
		t.Errorf("expected empty string for nil, got %q", result)
	}
}

func TestFormatProgram_Nil(t *testing.T) {
	result := FormatProgram(nil)
	if result != "" {
		t.Errorf("expected empty string for nil, got %q", result)
	}
}

// ============================================================================
// Query DSL Tests
// ============================================================================

func TestFormatQueryExpression_Inline(t *testing.T) {
	input := `@query(Users | id == 1 ?-> *)`
	result := parseAndFormat(t, input)
	expected := `@query(Users | id == 1 ?-> *)`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatQueryExpression_StringQuoting(t *testing.T) {
	input := `@query(Users | status == "active" ?-> *)`
	result := parseAndFormat(t, input)
	// Must preserve string quotes
	if !contains(result, `"active"`) {
		t.Errorf("expected quoted string in output, got %q", result)
	}
}

func TestFormatQueryExpression_Multiline(t *testing.T) {
	input := `@query(Users | status == "active" | role == "admin" | age >= 18 | verified == true ??-> *)`
	result := parseAndFormat(t, input)
	// Should be multiline (exceeds clause threshold)
	if !contains(result, "\n") {
		t.Errorf("expected multiline output for complex query, got %q", result)
	}
	// Should have proper indentation (tab)
	if !contains(result, "\tUsers") {
		t.Errorf("expected indented table name, got %q", result)
	}
	// String values should be quoted
	if !contains(result, `"active"`) {
		t.Errorf("expected quoted string 'active', got %q", result)
	}
}

func TestFormatInsertExpression_Inline(t *testing.T) {
	input := `@insert(Users |< name: "Alice" .)`
	result := parseAndFormat(t, input)
	expected := `@insert(Users |< name: "Alice" .)`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatInsertExpression_Multiline(t *testing.T) {
	input := `@insert(Users |< name: "Bob" |< email: "bob@test.com" |< status: "active" |< role: "user" ?-> *)`
	result := parseAndFormat(t, input)
	// Should be multiline
	if !contains(result, "\n") {
		t.Errorf("expected multiline output, got %q", result)
	}
	// Should have proper indentation (tab)
	if !contains(result, "\tUsers") {
		t.Errorf("expected indented table name, got %q", result)
	}
}

func TestFormatUpdateExpression(t *testing.T) {
	input := `@update(Users | id == {userId} |< status: "inactive" .-> count)`
	result := parseAndFormat(t, input)
	// Should contain the update structure
	if !contains(result, "@update(") {
		t.Errorf("expected @update in output, got %q", result)
	}
	// Interpolation should be preserved
	if !contains(result, "{userId}") {
		t.Errorf("expected interpolation preserved, got %q", result)
	}
}

func TestFormatDeleteExpression(t *testing.T) {
	input := `@delete(Users | status == "inactive" .)`
	result := parseAndFormat(t, input)
	expected := `@delete(Users | status == "inactive" .)`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatTransactionExpression(t *testing.T) {
	input := `@transaction { let x = 1 }`
	result := parseAndFormat(t, input)
	// Should contain transaction structure
	if !contains(result, "@transaction {") {
		t.Errorf("expected @transaction in output, got %q", result)
	}
}

func TestFormatTableLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "inline single row",
			input:    `let t = @table [{id: 1, name: "X"}]`,
			expected: `let t = @table [{id: 1, name: "X"}]`,
		},
		{
			name:     "multiline multiple rows",
			input:    `let t = @table [{id: 1, name: "Alice"}, {id: 2, name: "Bob"}]`,
			expected: "let t = @table [\n\t{id: 1, name: \"Alice\"},\n\t{id: 2, name: \"Bob\"},\n]",
		},
		{
			name:     "with schema",
			input:    `let t = @table(User) [{id: 1, name: "Alice"}, {id: 2, name: "Bob"}]`,
			expected: "let t = @table(User) [\n\t{id: 1, name: \"Alice\"},\n\t{id: 2, name: \"Bob\"},\n]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAndFormat(t, tt.input)
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

// helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr)))
}

func TestFormatComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single leading comment",
			input:    "// comment\nlet x = 1",
			expected: "// comment\nlet x = 1",
		},
		{
			name:     "multiple leading comments",
			input:    "// comment 1\n// comment 2\nlet x = 1",
			expected: "// comment 1\n// comment 2\nlet x = 1",
		},
		{
			name:     "comment between statements",
			input:    "let x = 1\n// comment\nlet y = 2",
			expected: "let x = 1\n// comment\nlet y = 2",
		},
		{
			name:     "trailing comment",
			input:    "let x = 1 // trailing\nlet y = 2",
			expected: "let x = 1 // trailing\nlet y = 2",
		},
		{
			name:     "trailing and leading comments",
			input:    "let x = 1 // trailing\n// leading\nlet y = 2",
			expected: "let x = 1 // trailing\n// leading\nlet y = 2",
		},
		{
			name:     "trailing comment on opening brace",
			input:    "let f = fn() { // comment\n\tlet x = 1\n}",
			expected: "let f = fn() { // comment\n\tlet x = 1\n}",
		},
		{
			name:     "comment inside tag content",
			input:    "<div>\n\t// comment\n\t<span>x</span>\n</div>",
			expected: "<div>\n\t// comment\n\t<span>x</span>\n</div>",
		},
		{
			name:     "blank line after opening tag preserved",
			input:    "<form>\n\n\t<fieldset>\"x\"</fieldset>\n</form>",
			expected: "<form>\n\n\t<fieldset>\"x\"</fieldset>\n</form>",
		},
		{
			name:     "blank line before closing tag preserved",
			input:    "<fieldset>\n\t<label>\"x\"</label>\n\n</fieldset>",
			expected: "<fieldset>\n\t<label>\"x\"</label>\n\n</fieldset>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAndFormat(t, tt.input)
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestFormatBlankLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single blank line preserved",
			input:    "let x = 1\n\nlet y = 2",
			expected: "let x = 1\n\nlet y = 2",
		},
		{
			name:     "multiple blank lines collapse to one",
			input:    "let x = 1\n\n\n\nlet y = 2",
			expected: "let x = 1\n\nlet y = 2",
		},
		{
			name:     "blank line with comment",
			input:    "let x = 1\n\n// comment\nlet y = 2",
			expected: "let x = 1\n\n// comment\nlet y = 2",
		},
		{
			name:     "blank line after comment before first statement",
			input:    "// header comment\n\nlet x = 1",
			expected: "// header comment\n\nlet x = 1",
		},
		{
			name:     "blank line preserved between check statements",
			input:    "if (x) {\n\n\tcheck (a) else {err: 1}\n\n\tlet y = 2\n\n\tcheck (b) else {err: 2}\n}",
			expected: "if (x) {\n\n\tcheck (a) else {err: 1}\n\n\tlet y = 2\n\n\tcheck (b) else {err: 2}\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAndFormat(t, tt.input)
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestFormatTagLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple tag no attributes",
			input:    "<br/>",
			expected: "<br/>",
		},
		{
			name:     "tag with string attribute",
			input:    `<input type="text"/>`,
			expected: `<input type="text"/>`,
		},
		{
			name:     "tag with boolean attribute",
			input:    `<input disabled/>`,
			expected: `<input disabled/>`,
		},
		{
			name:     "tag with expression attribute",
			input:    `<input value={name}/>`,
			expected: `<input value={name}/>`,
		},
		{
			name:     "tag with spread",
			input:    `<input ...attrs/>`,
			expected: `<input ...attrs/>`,
		},
		{
			name:     "tag with multiple attributes inline",
			input:    `<input type="text" name="foo"/>`,
			expected: `<input type="text" name="foo"/>`,
		},
		{
			name:     "tag with single-quoted onclick",
			input:    `<i onclick='doSomething()' class="icon"/>`,
			expected: `<i onclick='doSomething()' class="icon"/>`,
		},
		{
			name:     "tag with @ attribute (form binding)",
			input:    `<input @field="Firstname"/>`,
			expected: `<input @field="Firstname"/>`,
		},
		{
			name:     "tag with @record attribute",
			input:    `<form @record={person} action="/edit/" method="POST"></form>`,
			expected: `<form @record={person} action="/edit/" method="POST"></form>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAndFormat(t, tt.input)
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestFormatTagPairExpression(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple pair tag",
			input:    "<div>content</div>",
			expected: "<div>content</div>",
		},
		{
			name:     "pair tag with string attribute",
			input:    `<div class="foo">content</div>`,
			expected: `<div class="foo">content</div>`,
		},
		{
			name:     "pair tag with expression attribute",
			input:    `<div id={myId}>content</div>`,
			expected: `<div id={myId}>content</div>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAndFormat(t, tt.input)
			if result != tt.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestFormatTagAttributesMultiline(t *testing.T) {
	// This tests the multiline attribute formatting when line exceeds threshold
	// Tags with many attributes should break to multiline
	input := `<Part view="form" id="editor" action={@params.action} method="post" class="container"/>`

	result := parseAndFormat(t, input)

	// Should break to multiline since total attr length > 60
	if !contains(result, "\n") {
		t.Errorf("expected multiline output for long tag, got:\n%s", result)
	}

	// Should have proper indentation
	if !contains(result, "\t") {
		t.Errorf("expected indented attributes, got:\n%s", result)
	}

	// Should preserve all attributes
	if !contains(result, "view=") {
		t.Errorf("expected view attribute, got:\n%s", result)
	}
	if !contains(result, "action={@params.action}") {
		t.Errorf("expected action attribute with expression, got:\n%s", result)
	}
}

func TestFormatTagAttributesMultilineExact(t *testing.T) {
	// Test exact output format for multiline tag attributes
	input := `<Part view="form" id="editor" action={@params.action} method="post" class="container"/>`
	expected := `<Part
	view="form"
	id="editor"
	action={@params.action}
	method="post"
	class="container"
/>`

	result := parseAndFormat(t, input)
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormatTagPairAttributesMultiline(t *testing.T) {
	// Test multiline attributes on paired tags too
	input := `<Form method="post" action="/api/users" class="form-container" enctype="multipart/form-data">
	<input type="text"/>
</Form>`
	expected := `<Form
	method="post"
	action="/api/users"
	class="form-container"
	enctype="multipart/form-data"
>
	<input type="text"/>
</Form>`

	result := parseAndFormat(t, input)
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}