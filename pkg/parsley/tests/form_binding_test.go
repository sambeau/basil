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
