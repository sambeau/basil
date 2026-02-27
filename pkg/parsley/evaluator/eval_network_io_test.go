package evaluator

import (
	"fmt"
	"testing"

	"github.com/sambeau/basil/testenv"
)

func TestFetch_JSON(t *testing.T) {
	env := testenv.Start(t, testenv.WithHTTPS())
	testHTTPClient = env.HTTPSClient
	t.Cleanup(func() { testHTTPClient = nil })

	env.ServeJSON("/data.json", map[string]any{"name": "basil", "version": 1})

	src := fmt.Sprintf(`
let {data, error} = <=/= JSON(url("%s/data.json"))
data
`, env.HTTPSURL)

	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}

	dict, ok := result.(*Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary, got %T: %s", result, result.Inspect())
	}

	nameExpr, exists := dict.Pairs["name"]
	if !exists {
		t.Fatal("expected 'name' key in result")
	}
	nameObj := Eval(nameExpr, NewEnvironment())
	nameStr, ok := nameObj.(*String)
	if !ok {
		t.Fatalf("expected string for 'name', got %T", nameObj)
	}
	if nameStr.Value != "basil" {
		t.Errorf("expected name=%q, got %q", "basil", nameStr.Value)
	}
}

func TestFetch_Text(t *testing.T) {
	env := testenv.Start(t, testenv.WithHTTPS())
	testHTTPClient = env.HTTPSClient
	t.Cleanup(func() { testHTTPClient = nil })

	env.ServeText("/hello.txt", "hello basil")

	src := fmt.Sprintf(`
let {data, error} = <=/= text(url("%s/hello.txt"))
data
`, env.HTTPSURL)

	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}

	str, ok := result.(*String)
	if !ok {
		t.Fatalf("expected String, got %T: %s", result, result.Inspect())
	}
	if str.Value != "hello basil" {
		t.Errorf("expected %q, got %q", "hello basil", str.Value)
	}
}

func TestFetch_404_StatusCode(t *testing.T) {
	env := testenv.Start(t, testenv.WithHTTPS())
	testHTTPClient = env.HTTPSClient
	t.Cleanup(func() { testHTTPClient = nil })

	env.ServeError("/missing", 404)

	// In Parsley's fetch semantics, {data, error} captures network-level failures
	// only. An HTTP 404 is a valid response — data holds the body and error is null.
	// Use {data, error, status} and assert on status to check HTTP error codes.
	src := fmt.Sprintf(`
let {data, error, status} = <=/= text(url("%s/missing"))
status
`, env.HTTPSURL)

	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected hard evaluator error: %s", result.(*Error).Message)
	}
	num, ok := result.(*Integer)
	if !ok {
		t.Fatalf("expected Integer for status, got %T: %s", result, result.Inspect())
	}
	if num.Value != 404 {
		t.Errorf("expected status 404, got %d", num.Value)
	}
}

func TestFetch_ErrorCapture_Success(t *testing.T) {
	env := testenv.Start(t, testenv.WithHTTPS())
	testHTTPClient = env.HTTPSClient
	t.Cleanup(func() { testHTTPClient = nil })

	env.ServeJSON("/ok.json", map[string]any{"ok": true})

	// On a successful fetch the error field should be null.
	src := fmt.Sprintf(`
let {data, error} = <=/= JSON(url("%s/ok.json"))
error
`, env.HTTPSURL)

	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected evaluator error: %s", result.(*Error).Message)
	}
	if result != NULL {
		t.Errorf("expected error to be null on success, got %s", result.Inspect())
	}
}

func TestFetch_InvalidURL(t *testing.T) {
	// No testenv needed — port 1 on loopback always refuses connections.
	// A network-level failure may surface as a hard evaluator Error or as a
	// captured error string in the {data, error} pattern — both are acceptable.
	src := `
let {data, error} = <=/= text(url("https://127.0.0.1:1/bad"))
error
`
	result := testEval(src)
	// Accept either a captured error string or a hard evaluator Error.
	switch r := result.(type) {
	case *String:
		if r.Value == "" {
			t.Error("expected non-empty error string for refused connection")
		}
	case *Error:
		// Hard error is acceptable for connection-level failures.
	default:
		t.Errorf("expected String or Error for refused connection, got %T: %s", result, result.Inspect())
	}
}

func TestFetch_Redirect(t *testing.T) {
	env := testenv.Start(t, testenv.WithHTTPS())
	testHTTPClient = env.HTTPSClient
	t.Cleanup(func() { testHTTPClient = nil })

	env.ServeText("/final", "landed")
	env.ServeRedirect("/redir", "/final", 302)

	src := fmt.Sprintf(`
let {data, error} = <=/= text(url("%s/redir"))
data
`, env.HTTPSURL)

	result := testEval(src)
	if isError(result) {
		t.Fatalf("unexpected error: %s", result.(*Error).Message)
	}
	str, ok := result.(*String)
	if !ok {
		t.Fatalf("expected String after redirect, got %T: %s", result, result.Inspect())
	}
	if str.Value != "landed" {
		t.Errorf("expected body %q after redirect, got %q", "landed", str.Value)
	}
}
