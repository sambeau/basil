package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
)

// Phase 4: Form Binding Tests (FEAT-091)

// evalFormTest helper that evaluates Parsley code using the full evaluator
func evalFormTest(t *testing.T, input string) evaluator.Object {
	t.Helper()
	result, err := parsley.Eval(input)
	if err != nil {
		t.Fatalf("evaluation error: %v", err)
	}
	if result == nil || result.Value == nil {
		t.Fatal("result is nil")
	}
	return result.Value
}

// TEST-FORM-001: @record establishes context
func TestFormRecordContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "form with @record renders without @record attribute",
			input: `
				@schema User {
					name: string
					email: email
				}
				let user = User({name: "Alice", email: "alice@example.com"})
				<form @record={user}>
					"content"
				</form>
			`,
			expected: `<form>content</form>`,
		},
		{
			name: "form with @record and other attributes",
			input: `
				@schema User {
					name: string
				}
				let user = User({name: "Alice"})
				<form @record={user} method="post" action="/save">
					"content"
				</form>
			`,
			expected: `<form method="post" action="/save">content</form>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value
			// Normalize whitespace for comparison
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
			}
		})
	}
}

// TEST-FORM-002: @field binds value and adds attributes
func TestFormFieldBinding(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "input @field adds name and value",
			input: `
				@schema User {
					name: string
				}
				let user = User({name: "Alice"})
				<form @record={user}>
					<input @field="name"/>
				</form>
			`,
			contains: []string{`name="name"`, `value="Alice"`},
		},
		{
			name: "input @field adds required attribute",
			input: `
				@schema User {
					name: string
				}
				let user = User({name: ""})
				<form @record={user}>
					<input @field="name"/>
				</form>
			`,
			contains: []string{`name="name"`, `required`, `aria-required="true"`},
		},
		{
			name: "input @field derives email type",
			input: `
				@schema User {
					email: email
				}
				let user = User({email: "alice@example.com"})
				<form @record={user}>
					<input @field="email"/>
				</form>
			`,
			contains: []string{`type="email"`, `name="email"`, `value="alice@example.com"`},
		},
		{
			name: "input @field with explicit type overrides derived",
			input: `
				@schema User {
					phone: phone
				}
				let user = User({phone: "+1234567890"})
				<form @record={user}>
					<input type="text" @field="phone"/>
				</form>
			`,
			contains: []string{`type="text"`, `name="phone"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q\ngot:\n%s", s, result)
				}
			}
		})
	}
}

// TEST-FORM-003: ARIA attributes
func TestFormAriaAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "validated record with error has aria-invalid",
			input: `
				@schema User {
					name: string(min: 3)
				}
				let user = User({name: "AB"}).validate()
				<form @record={user}>
					<input @field="name"/>
				</form>
			`,
			contains: []string{`aria-invalid="true"`, `aria-describedby="name-error"`},
		},
		{
			name: "validated record without error has aria-invalid false",
			input: `
				@schema User {
					name: string
				}
				let user = User({name: "Alice"}).validate()
				<form @record={user}>
					<input @field="name"/>
				</form>
			`,
			contains: []string{`aria-invalid="false"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q\ngot:\n%s", s, result)
				}
			}
		})
	}
}

// TEST-FORM-004: Checkbox binding
func TestFormCheckboxBinding(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name: "checkbox checked when value is true",
			input: `
				@schema User {
					active: bool
				}
				let user = User({active: true})
				<form @record={user}>
					<input type="checkbox" @field="active"/>
				</form>
			`,
			contains: []string{`type="checkbox"`, `name="active"`, `checked`},
		},
		{
			name: "checkbox unchecked when value is false",
			input: `
				@schema User {
					active: bool
				}
				let user = User({active: false})
				<form @record={user}>
					<input type="checkbox" @field="active"/>
				</form>
			`,
			contains: []string{`type="checkbox"`, `name="active"`},
			excludes: []string{`checked`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q\ngot:\n%s", s, result)
				}
			}

			for _, s := range tt.excludes {
				if strings.Contains(result, s) {
					t.Errorf("expected result NOT to contain %q\ngot:\n%s", s, result)
				}
			}
		})
	}
}

// TEST-FORM-005: Radio binding
func TestFormRadioBinding(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name: "radio checked when value matches",
			input: `
				@schema User {
					status: enum["active", "inactive"]
				}
				let user = User({status: "active"})
				<form @record={user}>
					<input type="radio" @field="status" value="active"/>
					<input type="radio" @field="status" value="inactive"/>
				</form>
			`,
			contains: []string{
				`value="active" checked`,
				`value="inactive"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q\ngot:\n%s", s, result)
				}
			}
		})
	}
}

// TEST-FORM-006: Label component
func TestFormLabelComponent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "self-closing Label renders title",
			input: `
				@schema User {
					firstName: string | {title: "First Name"}
				}
				let user = User({firstName: "Alice"})
				<form @record={user}>
					<Label @field="firstName"/>
				</form>
			`,
			expected: `<form><label for="firstName">First Name</label></form>`,
		},
		{
			name: "Label generates title from field name",
			input: `
				@schema User {
					firstName: string
				}
				let user = User({firstName: "Alice"})
				<form @record={user}>
					<Label @field="firstName"/>
				</form>
			`,
			expected: `<form><label for="firstName">First Name</label></form>`,
		},
		{
			name: "Label tag pair includes children",
			input: `
				@schema User {
					name: string | {title: "Name"}
				}
				let user = User({name: "Alice"})
				<form @record={user}>
					<Label @field="name">" (required)"</Label>
				</form>
			`,
			expected: `<form><label for="name">Name (required)</label></form>`,
		},
		{
			name: "Label with @tag override",
			input: `
				@schema User {
					name: string | {title: "Name"}
				}
				let user = User({name: "Alice"})
				<form @record={user}>
					<Label @field="name" @tag="span"/>
				</form>
			`,
			expected: `<form><span>Name</span></form>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
			}
		})
	}
}

// TEST-FORM-007: Error component
func TestFormErrorComponent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		isEmpty  bool
	}{
		{
			name: "Error renders message when field has error",
			input: `
				@schema User {
					name: string(min: 3)
				}
				let user = User({name: "AB"}).validate()
				<form @record={user}>
					<Error @field="name"/>
				</form>
			`,
			contains: []string{`id="name-error"`, `role="alert"`, `class="error"`},
		},
		{
			name: "Error renders nothing when field is valid",
			input: `
				@schema User {
					name: string
				}
				let user = User({name: "Alice"}).validate()
				<form @record={user}>
					<Error @field="name"/>
				</form>
			`,
			isEmpty: true,
		},
		{
			name: "Error with @tag override",
			input: `
				@schema User {
					name: string(min: 3)
				}
				let user = User({name: "AB"}).validate()
				<form @record={user}>
					<Error @field="name" @tag="div"/>
				</form>
			`,
			contains: []string{`<div id="name-error"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value

			if tt.isEmpty {
				// Check that Error rendered nothing (just form wrapper)
				if !strings.Contains(result, "<form></form>") {
					t.Errorf("expected empty error but got:\n%s", result)
				}
				return
			}

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q\ngot:\n%s", s, result)
				}
			}
		})
	}
}

// TEST-FORM-008: Meta component
func TestFormMetaComponent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Meta renders metadata value",
			input: `
				@schema User {
					email: email | {help: "Your email address"}
				}
				let user = User({email: "alice@example.com"})
				<form @record={user}>
					<Meta @field="email" @key="help"/>
				</form>
			`,
			expected: `<form><span>Your email address</span></form>`,
		},
		{
			name: "Meta with @tag override",
			input: `
				@schema User {
					email: email | {help: "Your email address"}
				}
				let user = User({email: "alice@example.com"})
				<form @record={user}>
					<Meta @field="email" @key="help" @tag="p"/>
				</form>
			`,
			expected: `<form><p>Your email address</p></form>`,
		},
		{
			name: "Meta renders nothing when key not present",
			input: `
				@schema User {
					email: email
				}
				let user = User({email: "alice@example.com"})
				<form @record={user}>
					<Meta @field="email" @key="help"/>
				</form>
			`,
			expected: `<form></form>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
			}
		})
	}
}

// TEST-FORM-009: Select component
func TestFormSelectComponent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "Select renders options from enum",
			input: `
				@schema User {
					status: enum["active", "pending", "inactive"]
				}
				let user = User({status: "pending"})
				<form @record={user}>
					<Select @field="status"/>
				</form>
			`,
			contains: []string{
				`<select`,
				`name="status"`,
				`<option value="active">active</option>`,
				`<option value="pending" selected>pending</option>`,
				`<option value="inactive">inactive</option>`,
			},
		},
		{
			name: "Select with placeholder",
			input: `
				@schema User {
					status: enum["active", "inactive"]
				}
				let user = User({status: ""})
				<form @record={user}>
					<Select @field="status" placeholder="Choose status"/>
				</form>
			`,
			contains: []string{
				`<option value="">Choose status</option>`,
			},
		},
		{
			name: "Select with required field",
			input: `
				@schema User {
					status: enum["active", "inactive"]
				}
				let user = User({status: ""})
				<form @record={user}>
					<Select @field="status"/>
				</form>
			`,
			contains: []string{`required`, `aria-required="true"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q\ngot:\n%s", s, result)
				}
			}
		})
	}
}

// TEST-FORM-010: Error handling for @field outside form
func TestFormFieldOutsideForm(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		errText string // Error message should contain this text
	}{
		{
			name: "input @field outside form is error",
			input: `
				@schema User {
					name: string
				}
				<input @field="name"/>
			`,
			errText: "must be inside a <form @record=",
		},
		{
			name: "Label outside form is error",
			input: `
				@schema User {
					name: string
				}
				<Label @field="name"/>
			`,
			errText: "must be inside a <form @record=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsley.Eval(tt.input)

			// Check for error in result value
			if err == nil && result != nil && result.Value != nil {
				errObj, isErr := result.Value.(*evaluator.Error)
				if isErr {
					if !strings.Contains(errObj.Message, tt.errText) {
						t.Errorf("expected error message to contain %q but got: %s", tt.errText, errObj.Message)
					}
					return
				}
				// Not an error at all
				t.Fatalf("expected error but got: %s", result.Value.Inspect())
			}

			// Check for parsing error
			if err != nil {
				if !strings.Contains(err.Error(), tt.errText) {
					t.Errorf("expected error message to contain %q but got: %v", tt.errText, err)
				}
			}
		})
	}
}

// TEST-FORM-011: Type derivation
func TestFormTypeDerivation(t *testing.T) {
	tests := []struct {
		name         string
		schemaType   string
		expectedType string
	}{
		{"email derives email", "email", "email"},
		{"url derives url", "url", "url"},
		{"phone derives tel", "phone", "tel"},
		{"int derives number", "int", "number"},
		{"date derives date", "date", "date"},
		{"datetime derives datetime-local", "datetime", "datetime-local"},
		{"time derives time", "time", "time"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `
				@schema Test {
					field: ` + tt.schemaType + `?
				}
				let data = Test({field: ""})
				<form @record={data}>
					<input @field="field"/>
				</form>
			`

			evaluated := evalFormTest(t, input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value
			expected := `type="` + tt.expectedType + `"`

			if !strings.Contains(result, expected) {
				t.Errorf("expected result to contain %q\ngot:\n%s", expected, result)
			}
		})
	}
}

// TEST-FORM-012: Length constraints
func TestFormLengthConstraints(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "minLength constraint",
			input: `
				@schema User {
					name: string(min: 3)
				}
				let user = User({name: ""})
				<form @record={user}>
					<input @field="name"/>
				</form>
			`,
			contains: []string{`minlength="3"`},
		},
		{
			name: "maxLength constraint",
			input: `
				@schema User {
					name: string(max: 50)
				}
				let user = User({name: ""})
				<form @record={user}>
					<input @field="name"/>
				</form>
			`,
			contains: []string{`maxlength="50"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q\ngot:\n%s", s, result)
				}
			}
		})
	}
}

// TEST-FORM-013: Numeric constraints
func TestFormNumericConstraints(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "min value constraint on number",
			input: `
				@schema Product {
					price: int(min: 0)
				}
				let product = Product({price: 100})
				<form @record={product}>
					<input @field="price"/>
				</form>
			`,
			contains: []string{`min="0"`},
		},
		{
			name: "max value constraint on number",
			input: `
				@schema Product {
					qty: int(max: 999)
				}
				let product = Product({qty: 10})
				<form @record={product}>
					<input @field="qty"/>
				</form>
			`,
			contains: []string{`max="999"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q\ngot:\n%s", s, result)
				}
			}
		})
	}
}

// TEST-FORM-014: Placeholder from metadata
func TestFormPlaceholderMetadata(t *testing.T) {
	input := `
		@schema User {
			name: string | {placeholder: "Enter your name"}
		}
		let user = User({name: ""})
		<form @record={user}>
			<input @field="name"/>
		</form>
	`

	evaluated := evalFormTest(t, input)
	if errObj, isErr := evaluated.(*evaluator.Error); isErr {
		t.Fatalf("unexpected error: %s", errObj.Message)
	}

	result := evaluated.(*evaluator.String).Value

	if !strings.Contains(result, `placeholder="Enter your name"`) {
		t.Errorf("expected result to contain placeholder\ngot:\n%s", result)
	}
}

// TEST-FORM-015: Textarea binding
func TestFormTextareaBinding(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "textarea renders value as content",
			input: `
				@schema Post {
					body: text?
				}
				let post = Post({body: "Hello world"})
				<form @record={post}>
					<textarea @field="body"/>
				</form>
			`,
			contains: []string{`<textarea`, `name="body"`, `>Hello world</textarea>`},
		},
		{
			name: "textarea with validation attributes",
			input: `
				@schema Post {
					body: text(min: 10, max: 1000)
				}
				let post = Post({body: ""})
				<form @record={post}>
					<textarea @field="body"/>
				</form>
			`,
			contains: []string{`minlength="10"`, `maxlength="1000"`, `required`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q\ngot:\n%s", s, result)
				}
			}
		})
	}
}

// TEST-FORM-016: Record shorthand methods work in form context
func TestFormRecordShorthandMethods(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "record.title() shorthand",
			input: `
				@schema User {
					firstName: string | {title: "First Name"}
				}
				let user = User({firstName: "Alice"})
				user.title("firstName")
			`,
			expected: "First Name",
		},
		{
			name: "record.placeholder() shorthand",
			input: `
				@schema User {
					email: email | {placeholder: "you@example.com"}
				}
				let user = User({email: ""})
				user.placeholder("email")
			`,
			expected: "you@example.com",
		},
		{
			name: "record.meta() shorthand",
			input: `
				@schema User {
					email: email | {help: "Your work email"}
				}
				let user = User({email: ""})
				user.meta("email", "help")
			`,
			expected: "Your work email",
		},
		{
			name: "record.enumValues() shorthand",
			input: `
				@schema User {
					status: enum["active", "pending"]
				}
				let user = User({status: "active"})
				user.enumValues("status")
			`,
			expected: `[active, pending]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.Inspect()

			if result != tt.expected {
				t.Errorf("expected %s but got %s", tt.expected, result)
			}
		})
	}
}

// =============================================================================
// FEAT-094: Auto Field Form Handling Tests (SPEC-ID-007, SPEC-ID-008)
// =============================================================================

// TestAutoFieldsExcludedFromVisibleFields tests SPEC-ID-008
func TestAutoFieldsExcludedFromVisibleFields(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "SPEC-ID-008: visibleFields excludes auto fields",
			input: `
				@schema User {
					id: ulid(auto)
					name: string
					email: email
				}
				User.visibleFields()
			`,
			expected: `[name, email]`,
		},
		{
			name: "visibleFields excludes multiple auto fields",
			input: `
				@schema Article {
					id: id(auto)
					title: string
					createdAt: datetime(auto)
					updatedAt: datetime(auto)
				}
				Article.visibleFields()
			`,
			expected: `[title]`,
		},
		{
			name: "visibleFields excludes both auto and hidden",
			input: `
				@schema Product {
					id: int(auto)
					name: string
					internalCode: string | {hidden: true}
					price: money
				}
				Product.visibleFields()
			`,
			expected: `[name, price]`,
		},
		{
			name: "fields() includes auto fields (not visibleFields)",
			input: `
				@schema User {
					id: ulid(auto)
					name: string
				}
				User.fields()
			`,
			expected: `[id, name]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.Inspect()
			if result != tt.expected {
				t.Errorf("expected %s but got %s", tt.expected, result)
			}
		})
	}
}

// TestAutoFieldRendersAsHidden tests SPEC-ID-007
func TestAutoFieldRendersAsHidden(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "SPEC-ID-007: auto field renders as hidden input",
			input: `
				@schema User {
					id: ulid(auto)
					name: string
				}
				let user = User({id: "01ARZ3NDEKTSV4RRFFQ69G5FAV", name: "Alice"})
				<form @record={user}>
					<input @field="id"/>
					<input @field="name"/>
				</form>
			`,
			contains: []string{`type="hidden"`, `name="id"`, `readonly`, `name="name"`, `value="Alice"`},
		},
		{
			name: "auto field includes value when present",
			input: `
				@schema Article {
					id: int(auto)
					title: string
				}
				let article = Article({id: 42, title: "Hello"})
				<form @record={article}>
					<input @field="id"/>
				</form>
			`,
			contains: []string{`type="hidden"`, `name="id"`, `value="42"`, `readonly`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected result to contain %q\ngot: %s", expected, result)
				}
			}
		})
	}
}

// TestFormPatternAttribute tests SPEC-PAT-008, SPEC-PAT-009
func TestFormPatternAttribute(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "SPEC-PAT-008: pattern generates HTML pattern attribute",
			input: `
				@schema User {
					username: string(pattern: /^[a-z][a-z0-9_]*$/)
				}
				let user = User({username: ""})
				<form @record={user}>
					<input @field="username"/>
				</form>
			`,
			contains: []string{`pattern="^[a-z][a-z0-9_]*$"`, `name="username"`},
		},
		{
			name: "pattern with length constraints",
			input: `
				@schema Registration {
					code: string(min: 4, max: 8, pattern: /^[A-Z0-9]+$/)
				}
				let reg = Registration({code: ""})
				<form @record={reg}>
					<input @field="code"/>
				</form>
			`,
			contains: []string{`pattern="^[A-Z0-9]+$"`, `minlength="4"`, `maxlength="8"`},
		},
		{
			name: "complex pattern - slug format",
			input: `
				@schema Post {
					slug: string(pattern: /^[a-z0-9]+(?:-[a-z0-9]+)*$/)
				}
				let post = Post({slug: ""})
				<form @record={post}>
					<input @field="slug"/>
				</form>
			`,
			contains: []string{`pattern="^[a-z0-9]+(?:-[a-z0-9]+)*$"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected result to contain %q\ngot: %s", expected, result)
				}
			}
		})
	}
}

// TEST-FORM-015: Autocomplete attribute derivation (FEAT-097)
func TestFormAutocomplete(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name: "email type derives autocomplete email",
			input: `
				@schema User {
					email: email
				}
				let user = User({email: ""})
				<form @record={user}>
					<input @field="email"/>
				</form>
			`,
			contains: []string{`autocomplete="email"`},
		},
		{
			name: "firstName field name derives autocomplete",
			input: `
				@schema User {
					firstName: string
				}
				let user = User({firstName: ""})
				<form @record={user}>
					<input @field="firstName"/>
				</form>
			`,
			contains: []string{`autocomplete="given-name"`},
		},
		{
			name: "password field derives current-password",
			input: `
				@schema Login {
					password: string
				}
				let login = Login({password: ""})
				<form @record={login}>
					<input @field="password" type="password"/>
				</form>
			`,
			contains: []string{`autocomplete="current-password"`},
		},
		{
			name: "newPassword field derives new-password",
			input: `
				@schema Registration {
					newPassword: string
				}
				let reg = Registration({newPassword: ""})
				<form @record={reg}>
					<input @field="newPassword" type="password"/>
				</form>
			`,
			contains: []string{`autocomplete="new-password"`},
		},
		{
			name: "explicit metadata override",
			input: `
				@schema Shipping {
					street: string | {autocomplete: "shipping street-address"}
				}
				let shipping = Shipping({street: ""})
				<form @record={shipping}>
					<input @field="street"/>
				</form>
			`,
			contains: []string{`autocomplete="shipping street-address"`},
		},
		{
			name: "autocomplete off via metadata",
			input: `
				@schema Form {
					captcha: string | {autocomplete: "off"}
				}
				let form = Form({captcha: ""})
				<form @record={form}>
					<input @field="captcha"/>
				</form>
			`,
			contains: []string{`autocomplete="off"`},
		},
		{
			name: "unknown field has no autocomplete",
			input: `
				@schema Form {
					favoriteColor: string
				}
				let form = Form({favoriteColor: ""})
				<form @record={form}>
					<input @field="favoriteColor"/>
				</form>
			`,
			excludes: []string{`autocomplete=`},
		},
		{
			name: "phone type derives tel",
			input: `
				@schema Contact {
					phone: phone
				}
				let contact = Contact({phone: ""})
				<form @record={contact}>
					<input @field="phone"/>
				</form>
			`,
			contains: []string{`autocomplete="tel"`},
		},
		{
			name: "address fields derive correct values",
			input: `
				@schema Address {
					street: string
					city: string
					state: string
					zipCode: string
					country: string
				}
				let addr = Address({street: "", city: "", state: "", zipCode: "", country: ""})
				<form @record={addr}>
					<input @field="street"/>
					<input @field="city"/>
					<input @field="state"/>
					<input @field="zipCode"/>
					<input @field="country"/>
				</form>
			`,
			contains: []string{
				`autocomplete="street-address"`,
				`autocomplete="address-level2"`,
				`autocomplete="address-level1"`,
				`autocomplete="postal-code"`,
				`autocomplete="country-name"`,
			},
		},
		{
			name: "select with autocomplete",
			input: `
				@schema Checkout {
					country: string | {autocomplete: "country-name"} = "US" | "CA" | "UK"
				}
				let checkout = Checkout({country: "US"})
				<form @record={checkout}>
					<select @field="country"/>
				</form>
			`,
			contains: []string{`autocomplete="country-name"`},
		},
		{
			name: "textarea with autocomplete",
			input: `
				@schema Profile {
					address: string | {autocomplete: "street-address"}
				}
				let profile = Profile({address: ""})
				<form @record={profile}>
					<textarea @field="address"/>
				</form>
			`,
			contains: []string{`autocomplete="street-address"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected result to contain %q\ngot: %s", expected, result)
				}
			}

			for _, excluded := range tt.excludes {
				if strings.Contains(result, excluded) {
					t.Errorf("expected result NOT to contain %q\ngot: %s", excluded, result)
				}
			}
		})
	}
}

// TEST-FORM-012: Form @record auto-inserts hidden id field (FEAT-098)
func TestFormHiddenIdField(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name: "form with id inserts hidden input",
			input: `
				@schema User {
					id: int(auto)
					name: string
				}
				let user = User({id: 42, name: "Alice"})
				<form @record={user}>
					<input @field="name"/>
				</form>
			`,
			contains: []string{
				`<input type="hidden" name="id" value="42"/>`,
				`name="name"`,
			},
		},
		{
			name: "form without id field does not insert hidden input",
			input: `
				@schema User {
					name: string
				}
				let user = User({name: "Alice"})
				<form @record={user}>
					<input @field="name"/>
				</form>
			`,
			excludes: []string{`type="hidden"`},
		},
		{
			name: "form with null id does not insert hidden input",
			input: `
				@schema User {
					id: int(auto)
					name: string
				}
				let user = User({name: "Alice"})
				<form @record={user}>
					<input @field="name"/>
				</form>
			`,
			excludes: []string{`type="hidden"`},
		},
		{
			name: "form with uuid id",
			input: `
				@schema Product {
					id: uuid(auto)
					name: string
				}
				let product = Product({id: "550e8400-e29b-41d4-a716-446655440000", name: "Widget"})
				<form @record={product}>
					<input @field="name"/>
				</form>
			`,
			contains: []string{`<input type="hidden" name="id" value="550e8400-e29b-41d4-a716-446655440000"/>`},
		},
		{
			name: "hidden id appears before form contents",
			input: `
				@schema User {
					id: int(auto)
					name: string
				}
				let user = User({id: 99, name: "Bob"})
				<form @record={user}>
					<input @field="name"/>
				</form>
			`,
			// Hidden input should come right after opening tag
			contains: []string{`<form><input type="hidden" name="id" value="99"/>`},
		},
		{
			name: "hidden id escapes special characters",
			input: `
				@schema Doc {
					id: string
					title: string
				}
				let doc = Doc({id: "a&b<c>d\"e", title: "Test"})
				<form @record={doc}>
					<input @field="title"/>
				</form>
			`,
			contains: []string{`value="a&amp;b&lt;c&gt;d&quot;e"`},
		},
		{
			name: "form with other attributes and id",
			input: `
				@schema User {
					id: int(auto)
					name: string
				}
				let user = User({id: 1, name: "Test"})
				<form @record={user} method="post" action="/save">
					<input @field="name"/>
				</form>
			`,
			contains: []string{
				`method="post"`,
				`action="/save"`,
				`<input type="hidden" name="id" value="1"/>`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := evalFormTest(t, tt.input)
			if errObj, isErr := evaluated.(*evaluator.Error); isErr {
				t.Fatalf("unexpected error: %s", errObj.Message)
			}

			result := evaluated.(*evaluator.String).Value

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected result to contain %q\ngot: %s", expected, result)
				}
			}

			for _, excluded := range tt.excludes {
				if strings.Contains(result, excluded) {
					t.Errorf("expected result NOT to contain %q\ngot: %s", excluded, result)
				}
			}
		})
	}
}
