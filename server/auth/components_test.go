package auth

import (
	"strings"
	"testing"
)

func TestComponentExpander_ExpandRegister(t *testing.T) {
	ResetUniqueIDCounter()
	expander := NewComponentExpander()

	tests := []struct {
		name     string
		input    string
		contains []string
		notIn    []string
	}{
		{
			name:  "basic register component",
			input: `<basil.auth.Register/>`,
			contains: []string{
				`class="basil-auth-register"`,
				`placeholder="Your name"`,
				`placeholder="you@example.com"`,
				`>Create account</button>`,
				`/__auth/register/begin`,
				`/__auth/register/finish`,
			},
		},
		{
			name:  "register with custom button text",
			input: `<basil.auth.Register button_text="Sign up now"/>`,
			contains: []string{
				`>Sign up now</button>`,
			},
		},
		{
			name:  "register with custom placeholders",
			input: `<basil.auth.Register name_placeholder="Enter name" email_placeholder="Enter email"/>`,
			contains: []string{
				`placeholder="Enter name"`,
				`placeholder="Enter email"`,
			},
		},
		{
			name:  "register with prefilled values",
			input: `<basil.auth.Register name="Sam" email="sam@example.com"/>`,
			contains: []string{
				`value="Sam"`,
				`value="sam@example.com"`,
			},
		},
		{
			name:  "register with custom class",
			input: `<basil.auth.Register class="my-form"/>`,
			contains: []string{
				`class="basil-auth-register my-form"`,
			},
		},
		{
			name:  "register with redirect",
			input: `<basil.auth.Register redirect="/dashboard"/>`,
			contains: []string{
				`window.location.href = '/dashboard'`,
			},
		},
		{
			name:  "register with recovery_page",
			input: `<basil.auth.Register recovery_page="/recovery-codes"/>`,
			contains: []string{
				`sessionStorage.setItem('basil_recovery_codes'`,
				`window.location.href = '/recovery-codes'`,
			},
			notIn: []string{
				`alert(`,
			},
		},
		{
			name:  "component in HTML context",
			input: `<div><basil.auth.Register/></div>`,
			contains: []string{
				`<div><form id=`,
				`</script></div>`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expander.ExpandComponents(tt.input)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("expected output to contain %q\nGot:\n%s", want, result)
				}
			}

			for _, notWant := range tt.notIn {
				if strings.Contains(result, notWant) {
					t.Errorf("expected output NOT to contain %q\nGot:\n%s", notWant, result)
				}
			}
		})
	}
}

func TestComponentExpander_ExpandLogin(t *testing.T) {
	ResetUniqueIDCounter()
	expander := NewComponentExpander()

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "basic login component",
			input: `<basil.auth.Login/>`,
			contains: []string{
				`class="basil-auth-login"`,
				`>Sign in</button>`,
				`/__auth/login/begin`,
				`/__auth/login/finish`,
			},
		},
		{
			name:  "login with custom button text",
			input: `<basil.auth.Login button_text="Log in with passkey"/>`,
			contains: []string{
				`>Log in with passkey</button>`,
			},
		},
		{
			name:  "login with custom class",
			input: `<basil.auth.Login class="login-btn"/>`,
			contains: []string{
				`class="basil-auth-login login-btn"`,
			},
		},
		{
			name:  "login with redirect",
			input: `<basil.auth.Login redirect="/home"/>`,
			contains: []string{
				`window.location.href = '/home'`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expander.ExpandComponents(tt.input)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("expected output to contain %q\nGot:\n%s", want, result)
				}
			}
		})
	}
}

func TestComponentExpander_ExpandLogout(t *testing.T) {
	ResetUniqueIDCounter()
	expander := NewComponentExpander()

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "basic logout component",
			input: `<basil.auth.Logout/>`,
			contains: []string{
				`class="basil-auth-logout`,
				`>Sign out</button>`,
				`/__auth/logout`,
			},
		},
		{
			name:  "logout with custom text",
			input: `<basil.auth.Logout text="Log out"/>`,
			contains: []string{
				`>Log out</button>`,
			},
		},
		{
			name:  "logout as link",
			input: `<basil.auth.Logout method="link"/>`,
			contains: []string{
				`<a id=`,
				`>Sign out</a>`,
			},
		},
		{
			name:  "logout with custom class",
			input: `<basil.auth.Logout class="nav-link"/>`,
			contains: []string{
				`class="basil-auth-logout nav-link`,
			},
		},
		{
			name:  "logout with redirect",
			input: `<basil.auth.Logout redirect="/login"/>`,
			contains: []string{
				`window.location.href = '/login'`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expander.ExpandComponents(tt.input)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("expected output to contain %q\nGot:\n%s", want, result)
				}
			}
		})
	}
}

func TestComponentExpander_MultipleComponents(t *testing.T) {
	ResetUniqueIDCounter()
	expander := NewComponentExpander()

	input := `<div>
		<basil.auth.Login/>
		<basil.auth.Logout/>
	</div>`

	result := expander.ExpandComponents(input)

	// Should contain both components
	if !strings.Contains(result, "basil-auth-login") {
		t.Error("expected login component")
	}
	if !strings.Contains(result, "basil-auth-logout") {
		t.Error("expected logout component")
	}

	// Each should have unique IDs
	if strings.Count(result, "basil-login-") != 2 { // ID appears twice: div id and getElementById
		t.Error("expected unique login ID")
	}
	if strings.Count(result, "basil-logout-") != 2 {
		t.Error("expected unique logout ID")
	}
}

func TestComponentExpander_NoComponents(t *testing.T) {
	expander := NewComponentExpander()

	input := `<div><p>Hello world</p></div>`
	result := expander.ExpandComponents(input)

	if result != input {
		t.Errorf("expected unchanged output for HTML without components\nGot: %s", result)
	}
}

// TestComponentExpander_ShortPrefix tests that auth.Register/Login/Logout work
// without the basil. prefix (this is what Parsley code like `<auth.Register/>` produces)
func TestComponentExpander_ShortPrefix(t *testing.T) {
	ResetUniqueIDCounter()
	expander := NewComponentExpander()

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "register without basil prefix",
			input: `<auth.Register button_text="Sign up"/>`,
			contains: []string{
				`class="basil-auth-register"`,
				`>Sign up</button>`,
				`/__auth/register/begin`,
			},
		},
		{
			name:  "login without basil prefix",
			input: `<auth.Login button_text="Sign in"/>`,
			contains: []string{
				`class="basil-auth-login"`,
				`>Sign in</button>`,
				`/__auth/login/begin`,
			},
		},
		{
			name:  "logout without basil prefix",
			input: `<auth.Logout text="Log out"/>`,
			contains: []string{
				`class="basil-auth-logout`,
				`>Log out</button>`,
				`/__auth/logout`,
			},
		},
		{
			name: "register with multiline attributes (Parsley output)",
			input: `<auth.Register                                                                  name_placeholder="Your name"
      email_placeholder="Email (optional)"
      button_text="Create account"
      recovery_page="/recovery-codes"
      class="auth-form" />`,
			contains: []string{
				`class="basil-auth-register auth-form"`,
				`placeholder="Your name"`,
				`placeholder="Email (optional)"`,
				`>Create account</button>`,
				`window.location.href = '/recovery-codes'`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expander.ExpandComponents(tt.input)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("expected output to contain %q\nGot:\n%s", want, result)
				}
			}
		})
	}
}

func TestComponentExpander_EscapesHTML(t *testing.T) {
	ResetUniqueIDCounter()
	expander := NewComponentExpander()

	// Test that special characters are escaped
	input := `<basil.auth.Register button_text="Click &amp; go"/>`
	result := expander.ExpandComponents(input)

	// The &amp; should remain escaped in output
	if !strings.Contains(result, "&amp;") {
		t.Error("expected HTML entities to be preserved in button text")
	}
}

func TestParseAttributes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name:  "empty",
			input: "",
			want:  map[string]string{},
		},
		{
			name:  "single attribute double quotes",
			input: ` name="Sam"`,
			want:  map[string]string{"name": "Sam"},
		},
		{
			name:  "single attribute single quotes",
			input: ` name='Sam'`,
			want:  map[string]string{"name": "Sam"},
		},
		{
			name:  "multiple attributes",
			input: ` name="Sam" email="sam@example.com" class="form"`,
			want:  map[string]string{"name": "Sam", "email": "sam@example.com", "class": "form"},
		},
		{
			name:  "attributes with underscores",
			input: ` button_text="Click me" name_placeholder="Enter name"`,
			want:  map[string]string{"button_text": "Click me", "name_placeholder": "Enter name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAttributes(tt.input)

			if len(got) != len(tt.want) {
				t.Errorf("parseAttributes() got %d attrs, want %d", len(got), len(tt.want))
				return
			}

			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parseAttributes()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}
