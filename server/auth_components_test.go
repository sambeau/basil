package server

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// evalWithPrelude evaluates Parsley source with the prelude loaded, so
// @basil/auth's Register/Login/Logout components are available.
func evalWithPrelude(t *testing.T, input string) string {
	t.Helper()
	if err := initPrelude("test"); err != nil {
		t.Fatalf("initPrelude: %v", err)
	}
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	result := evaluator.Eval(program, evaluator.NewEnvironment())
	switch v := result.(type) {
	case *evaluator.String:
		return v.Value
	case *evaluator.Array:
		// A component with sibling top-level tags evaluates to an array of
		// HTML strings; the server concatenates these when responding.
		var sb strings.Builder
		for _, el := range v.Elements {
			str, ok := el.(*evaluator.String)
			if !ok {
				t.Fatalf("expected array of Strings, got %T: %v", el, el.Inspect())
			}
			sb.WriteString(str.Value)
		}
		return sb.String()
	default:
		t.Fatalf("expected String or Array, got %T: %v", result, result.Inspect())
		return ""
	}
}

func TestAuthRegisterComponent(t *testing.T) {
	html := evalWithPrelude(t, `
let {Register} = import @basil/auth
<Register button_text="Sign up now" redirect="/dashboard" recovery_page="/recovery-codes" class="my-form"/>`)

	for _, want := range []string{
		`class="basil-auth-register my-form"`,
		`data-redirect="/dashboard"`,
		`data-recovery-page="/recovery-codes"`,
		`placeholder="Your name"`,
		`placeholder="you@example.com"`,
		`>Sign up now</button>`,
		`class="basil-auth-error"`,
		`<script src="/__/js/auth.js"></script>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("Register output missing %q in:\n%s", want, html)
		}
	}
}

func TestAuthRegisterDefaults(t *testing.T) {
	html := evalWithPrelude(t, `
let {Register} = import @basil/auth
<Register/>`)

	for _, want := range []string{
		`class="basil-auth-register"`,
		`data-redirect="/"`,
		`>Create account</button>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("Register defaults missing %q in:\n%s", want, html)
		}
	}
	if strings.Contains(html, "data-recovery-page") {
		t.Errorf("Register without recovery_page should omit data-recovery-page:\n%s", html)
	}
}

func TestAuthRegisterPrefill(t *testing.T) {
	html := evalWithPrelude(t, `
let {Register} = import @basil/auth
<Register name="Sam" email="sam@example.com"/>`)

	for _, want := range []string{
		`value="Sam"`,
		`value="sam@example.com"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("Register prefill missing %q in:\n%s", want, html)
		}
	}
}

func TestAuthLoginComponent(t *testing.T) {
	html := evalWithPrelude(t, `
let {Login} = import @basil/auth
<Login button_text="Log in with passkey" redirect="/home"/>`)

	for _, want := range []string{
		`class="basil-auth-login"`,
		`data-redirect="/home"`,
		`>Log in with passkey</button>`,
		`<script src="/__/js/auth.js"></script>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("Login output missing %q in:\n%s", want, html)
		}
	}
}

func TestAuthLogoutButtonAndLink(t *testing.T) {
	button := evalWithPrelude(t, `
let {Logout} = import @basil/auth
<Logout text="Sign out" redirect="/bye"/>`)
	if !strings.Contains(button, `<button`) || !strings.Contains(button, `class="basil-auth-logout basil-auth-button"`) {
		t.Errorf("Logout default should render a button:\n%s", button)
	}
	if !strings.Contains(button, `data-redirect="/bye"`) {
		t.Errorf("Logout missing data-redirect:\n%s", button)
	}

	link := evalWithPrelude(t, `
let {Logout} = import @basil/auth
<Logout method="link"/>`)
	if !strings.Contains(link, `<a href="#"`) || !strings.Contains(link, `class="basil-auth-logout"`) {
		t.Errorf("Logout method=link should render a link:\n%s", link)
	}
	if !strings.Contains(link, `>Sign out</a>`) {
		t.Errorf("Logout link missing default text:\n%s", link)
	}
}

func TestAuthComponentRenameOnImport(t *testing.T) {
	html := evalWithPrelude(t, `
let {Login as PasskeyLogin} = import @basil/auth
<PasskeyLogin button_text="In you go"/>`)

	if !strings.Contains(html, `>In you go</button>`) {
		t.Errorf("renamed import should render the component:\n%s", html)
	}
}
