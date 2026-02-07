// remote_write_test.go — Tests for remote write operators =/=> and =/=>>
// Covers spec FEAT-104 test plan layers 1–6

package tests

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// ============================================================================
// Layer 1: Lexer Tests
// ============================================================================

func TestRemoteWriteTokenise(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedType lexer.TokenType
		expectedLit  string
		skipToIndex  int // how many tokens to skip to reach the one we care about
	}{
		// L1.1 — Tokenise =/=>
		{
			name:         "L1.1a basic remote write",
			input:        "x =/=> y",
			expectedType: lexer.REMOTE_WRITE,
			expectedLit:  "=/=>",
			skipToIndex:  1,
		},
		// L1.2 — Tokenise =/=>>
		{
			name:         "L1.2a basic remote append",
			input:        "x =/=>> y",
			expectedType: lexer.REMOTE_APPEND,
			expectedLit:  "=/=>>",
			skipToIndex:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			var tok lexer.Token
			for i := 0; i <= tt.skipToIndex; i++ {
				tok = l.NextToken()
			}
			if tok.Type != tt.expectedType {
				t.Errorf("expected token type %s, got %s", tt.expectedType, tok.Type)
			}
			if tok.Literal != tt.expectedLit {
				t.Errorf("expected literal %q, got %q", tt.expectedLit, tok.Literal)
			}
		})
	}
}

func TestRemoteWriteNoAmbiguity(t *testing.T) {
	// L1.3 — Ensure existing tokens are unaffected
	tests := []struct {
		name         string
		input        string
		skipToIndex  int
		expectedType lexer.TokenType
		expectedLit  string
	}{
		{
			name:         "L1.3a plain assignment unchanged",
			input:        "x = y",
			skipToIndex:  1,
			expectedType: lexer.ASSIGN,
			expectedLit:  "=",
		},
		{
			name:         "L1.3b equality unchanged",
			input:        "x == y",
			skipToIndex:  1,
			expectedType: lexer.EQ,
			expectedLit:  "==",
		},
		{
			name:         "L1.3c file write unchanged",
			input:        "x ==> y",
			skipToIndex:  1,
			expectedType: lexer.WRITE_TO,
			expectedLit:  "==>",
		},
		{
			name:         "L1.3d file append unchanged",
			input:        "x ==>> y",
			skipToIndex:  1,
			expectedType: lexer.APPEND_TO,
			expectedLit:  "==>>",
		},
		{
			name:         "L1.3g both operators in same input — remote write",
			input:        "x =/=> y; a ==> b",
			skipToIndex:  1,
			expectedType: lexer.REMOTE_WRITE,
			expectedLit:  "=/=>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			var tok lexer.Token
			for i := 0; i <= tt.skipToIndex; i++ {
				tok = l.NextToken()
			}
			if tok.Type != tt.expectedType {
				t.Errorf("expected token type %s, got %s", tt.expectedType, tok.Type)
			}
			if tok.Literal != tt.expectedLit {
				t.Errorf("expected literal %q, got %q", tt.expectedLit, tok.Literal)
			}
		})
	}
}

func TestRemoteWriteBothOperatorsInSameInput(t *testing.T) {
	// L1.3g continued — verify second operator too
	input := "x =/=> y; a ==> b"
	l := lexer.New(input)

	// Collect all tokens
	var tokens []lexer.Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == lexer.EOF {
			break
		}
	}

	// Find REMOTE_WRITE and WRITE_TO
	foundRemoteWrite := false
	foundWriteTo := false
	for _, tok := range tokens {
		if tok.Type == lexer.REMOTE_WRITE {
			foundRemoteWrite = true
		}
		if tok.Type == lexer.WRITE_TO {
			foundWriteTo = true
		}
	}
	if !foundRemoteWrite {
		t.Error("expected REMOTE_WRITE token in input")
	}
	if !foundWriteTo {
		t.Error("expected WRITE_TO token in input")
	}
}

func TestRemoteWriteTokenPosition(t *testing.T) {
	// L1.4 — Token position tracking
	input := "x =/=> y"
	l := lexer.New(input)
	l.NextToken()        // x
	tok := l.NextToken() // =/=>

	if tok.Type != lexer.REMOTE_WRITE {
		t.Fatalf("expected REMOTE_WRITE, got %s", tok.Type)
	}
	// The =/=> starts at column 3 (1-based)
	if tok.Column != 3 {
		t.Errorf("expected column 3, got %d", tok.Column)
	}
}

// ============================================================================
// Layer 2: Parser Tests
// ============================================================================

func parseProgram(input string) *ast.Program {
	l := lexer.New(input)
	p := parser.New(l)
	return p.ParseProgram()
}

func TestRemoteWriteParsing(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantAppend bool
	}{
		// P2.1 — Basic =/=> parsing
		{"P2.1a simplest form", "x =/=> y", false},
		{"P2.1b dict as value", `{a: 1} =/=> target`, false},
		{"P2.1c string as value", `"hello" =/=> target`, false},
		{"P2.1d array as value", `[1, 2] =/=> target`, false},
		// P2.2 — Basic =/=>> parsing
		{"P2.2a simplest append form", "x =/=>> y", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := parseProgram(tt.input)
			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}

			rws, ok := program.Statements[0].(*ast.RemoteWriteStatement)
			if !ok {
				t.Fatalf("expected RemoteWriteStatement, got %T", program.Statements[0])
			}
			if rws.Append != tt.wantAppend {
				t.Errorf("expected Append=%v, got %v", tt.wantAppend, rws.Append)
			}
			if rws.Value == nil {
				t.Error("Value should not be nil")
			}
			if rws.Target == nil {
				t.Error("Target should not be nil")
			}
		})
	}
}

func TestRemoteWriteStringRoundTrip(t *testing.T) {
	// P2.3 — String() round-trip
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"P2.3a write", "x =/=> y", "x =/=> y;"},
		{"P2.3b append", "x =/=>> y", "x =/=>> y;"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := parseProgram(tt.input)
			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}
			rws, ok := program.Statements[0].(*ast.RemoteWriteStatement)
			if !ok {
				t.Fatalf("expected RemoteWriteStatement, got %T", program.Statements[0])
			}
			got := rws.String()
			if got != tt.expected {
				t.Errorf("expected String() = %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestRemoteWriteSemicolonHandling(t *testing.T) {
	// P2.4 — Semicolon handling
	tests := []struct {
		name  string
		input string
	}{
		{"P2.4a explicit semicolon", "x =/=> y;"},
		{"P2.4b no semicolon", "x =/=> y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}
			if len(program.Statements) != 1 {
				t.Fatalf("expected 1 statement, got %d", len(program.Statements))
			}
			if _, ok := program.Statements[0].(*ast.RemoteWriteStatement); !ok {
				t.Fatalf("expected RemoteWriteStatement, got %T", program.Statements[0])
			}
		})
	}
}

// ============================================================================
// Layer 3: Evaluator Tests — HTTP Remote Write
// ============================================================================

// testEvalRemoteWriteOp evaluates with write permissions (for tests that need it)
func testEvalRemoteWriteOp(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	env.Security = &evaluator.SecurityPolicy{
		AllowWriteAll: true,
	}
	return evaluator.Eval(program, env)
}

// newEchoServer creates a test HTTP server that echoes method, body, and headers as JSON
func newEchoServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		auth := r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := `{"method": "` + r.Method + `", "bodyLength": ` + itoa(len(body)) + `, "auth": "` + auth + `"}`
		w.Write([]byte(resp))
	}))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// extractResponseData extracts the __data field from a typed response dict, evaluating it.
func extractResponseData(t *testing.T, result evaluator.Object) evaluator.Object {
	t.Helper()

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary (typed response), got %T (%s)", result, result.Inspect())
	}

	dataExpr, exists := dict.Pairs["__data"]
	if !exists {
		t.Fatalf("response dict missing __data field")
	}

	env := evaluator.NewEnvironment()
	return evaluator.Eval(dataExpr, env)
}

// extractResponseMeta extracts a field from the __response sub-dict of a typed response dict.
func extractResponseMeta(t *testing.T, result evaluator.Object, field string) evaluator.Object {
	t.Helper()

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary (typed response), got %T (%s)", result, result.Inspect())
	}

	responseExpr, exists := dict.Pairs["__response"]
	if !exists {
		t.Fatalf("response dict missing __response field")
	}

	env := evaluator.NewEnvironment()
	responseObj := evaluator.Eval(responseExpr, env)
	responseDict, ok := responseObj.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("__response is not a Dictionary, got %T (%s)", responseObj, responseObj.Inspect())
	}

	fieldExpr, exists := responseDict.Pairs[field]
	if !exists {
		keys := make([]string, 0, len(responseDict.Pairs))
		for k := range responseDict.Pairs {
			keys = append(keys, k)
		}
		t.Fatalf("__response dict missing field %q, available: %v", field, keys)
	}

	return evaluator.Eval(fieldExpr, env)
}

// extractDataField extracts a field from the parsed JSON __data dict.
func extractDataField(t *testing.T, result evaluator.Object, field string) evaluator.Object {
	t.Helper()

	data := extractResponseData(t, result)
	dataDict, ok := data.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected __data to be Dictionary, got %T (%s)", data, data.Inspect())
	}

	fieldExpr, exists := dataDict.Pairs[field]
	if !exists {
		keys := make([]string, 0, len(dataDict.Pairs))
		for k := range dataDict.Pairs {
			keys = append(keys, k)
		}
		t.Fatalf("data dict missing field %q, available: %v", field, keys)
	}

	env := evaluator.NewEnvironment()
	return evaluator.Eval(fieldExpr, env)
}

func TestRemoteWriteHTTPPost(t *testing.T) {
	// E3.1 — HTTP POST (default method)
	server := newEchoServer()
	defer server.Close()

	tests := []struct {
		name        string
		input       string
		checkMethod string
	}{
		{
			name:        "E3.1a POST dict",
			input:       `{name: "Alice"} =/=> JSON(url("` + server.URL + `"))`,
			checkMethod: "POST",
		},
		{
			name:        "E3.1b POST string value",
			input:       `"hello" =/=> JSON(url("` + server.URL + `"))`,
			checkMethod: "POST",
		},
		{
			name:        "E3.1c POST array",
			input:       `[1, 2, 3] =/=> JSON(url("` + server.URL + `"))`,
			checkMethod: "POST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalHelper(tt.input)

			// Result is a typed response dict. The echo server returns JSON with "method" field.
			methodObj := extractDataField(t, result, "method")

			str, ok := methodObj.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String for method, got %T (%s)", methodObj, methodObj.Inspect())
			}
			if str.Value != tt.checkMethod {
				t.Errorf("expected method %q, got %q", tt.checkMethod, str.Value)
			}
		})
	}
}

func TestRemoteWriteHTTPPut(t *testing.T) {
	// E3.2 — HTTP PUT (via accessor)
	server := newEchoServer()
	defer server.Close()

	input := `{name: "Alice"} =/=> JSON(url("` + server.URL + `")).put`
	result := testEvalHelper(input)

	methodObj := extractDataField(t, result, "method")
	str, ok := methodObj.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T (%s)", methodObj, methodObj.Inspect())
	}
	if str.Value != "PUT" {
		t.Errorf("expected method PUT, got %q", str.Value)
	}
}

func TestRemoteWriteHTTPPatch(t *testing.T) {
	// E3.3 — HTTP PATCH (via accessor)
	server := newEchoServer()
	defer server.Close()

	input := `{age: 31} =/=> JSON(url("` + server.URL + `")).patch`
	result := testEvalHelper(input)

	methodObj := extractDataField(t, result, "method")
	str, ok := methodObj.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T (%s)", methodObj, methodObj.Inspect())
	}
	if str.Value != "PATCH" {
		t.Errorf("expected method PATCH, got %q", str.Value)
	}
}

func TestRemoteWriteHTTPPostExplicit(t *testing.T) {
	// E3.4 — HTTP POST explicit (via accessor)
	server := newEchoServer()
	defer server.Close()

	input := `{name: "Alice"} =/=> JSON(url("` + server.URL + `")).post`
	result := testEvalHelper(input)

	methodObj := extractDataField(t, result, "method")
	str, ok := methodObj.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T (%s)", methodObj, methodObj.Inspect())
	}
	if str.Value != "POST" {
		t.Errorf("expected method POST, got %q", str.Value)
	}
}

func TestRemoteWriteHTTPCustomHeaders(t *testing.T) {
	// E3.5 — HTTP with custom headers
	// The url() builtin only takes one arg; headers are passed via the format factory's options dict.
	server := newEchoServer()
	defer server.Close()

	input := `{data: 1} =/=> JSON(url("` + server.URL + `"), {headers: {Authorization: "Bearer token123"}})`
	result := testEvalHelper(input)

	authObj := extractDataField(t, result, "auth")
	str, ok := authObj.(*evaluator.String)
	if !ok {
		t.Fatalf("expected String, got %T (%s)", authObj, authObj.Inspect())
	}
	if str.Value != "Bearer token123" {
		t.Errorf("expected 'Bearer token123', got %q", str.Value)
	}
}

func TestRemoteWriteHTTPErrorHandling(t *testing.T) {
	// E3.6 — HTTP error handling

	// Server returning 500
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body)
		r.Body.Close()
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer errorServer.Close()

	t.Run("E3.6a server error", func(t *testing.T) {
		input := `{data: 1} =/=> JSON(url("` + errorServer.URL + `"))`
		result := testEvalHelper(input)

		switch r := result.(type) {
		case *evaluator.Dictionary:
			// Typed response dict — check that ok is false via __response
			okObj := extractResponseMeta(t, r, "ok")
			b, ok := okObj.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean for ok, got %T", okObj)
			}
			if b.Value != false {
				t.Errorf("expected ok=false for 500 response")
			}
		case *evaluator.Error:
			// Error object is also acceptable for server errors
		default:
			t.Fatalf("expected Dictionary or Error, got %T (%s)", result, result.Inspect())
		}
	})

	t.Run("E3.6b connection refused", func(t *testing.T) {
		input := `{data: 1} =/=> JSON(url("http://localhost:1"))`
		result := testEvalHelper(input)

		// Should be an error object
		if _, ok := result.(*evaluator.Error); !ok {
			// Could also be a response dict with error info — both acceptable
			if _, isDict := result.(*evaluator.Dictionary); !isDict {
				t.Fatalf("expected Error or Dictionary, got %T (%s)", result, result.Inspect())
			}
		}
	})
}

func TestRemoteWriteResponseFields(t *testing.T) {
	// E3.7 — Verify response dict has expected structure
	server := newEchoServer()
	defer server.Close()

	input := `{a: 1} =/=> JSON(url("` + server.URL + `"))`
	result := testEvalHelper(input)

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary response, got %T (%s)", result, result.Inspect())
	}

	// Response should have typed dict fields: __type, __format, __data, __response
	for _, field := range []string{"__type", "__format", "__data", "__response"} {
		if _, exists := dict.Pairs[field]; !exists {
			keys := make([]string, 0, len(dict.Pairs))
			for k := range dict.Pairs {
				keys = append(keys, k)
			}
			t.Errorf("response dict missing field %q (keys: %v)", field, keys)
		}
	}

	// Check status is 200 via __response
	statusObj := extractResponseMeta(t, result, "status")
	if num, ok := statusObj.(*evaluator.Integer); ok {
		if num.Value != 200 {
			t.Errorf("expected status 200, got %d", num.Value)
		}
	} else {
		t.Errorf("expected Integer status, got %T", statusObj)
	}

	// Check ok is true via __response
	okObj := extractResponseMeta(t, result, "ok")
	if b, ok := okObj.(*evaluator.Boolean); ok {
		if !b.Value {
			t.Errorf("expected ok=true, got false")
		}
	} else {
		t.Errorf("expected Boolean ok, got %T", okObj)
	}
}

// ============================================================================
// Layer 4: Evaluator Tests — Target Rejection by =/=>
// ============================================================================

func TestRemoteWriteRejectsLocalFiles(t *testing.T) {
	// E4.1 — Reject local file handles
	tmpDir, err := os.MkdirTemp("", "parsley_remote_write_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	localFile := filepath.Join(tmpDir, "local.json")

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "E4.1a JSON file target",
			input:    `"data" =/=> JSON("` + localFile + `")`,
			contains: []string{"network writes", "==>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalRemoteWriteOp(tt.input)

			errObj, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected Error, got %T (%s)", result, result.Inspect())
			}
			for _, substr := range tt.contains {
				if !strings.Contains(strings.ToLower(errObj.Message), strings.ToLower(substr)) {
					t.Errorf("error message should contain %q, got %q", substr, errObj.Message)
				}
			}
		})
	}
}

func TestRemoteWriteRejectsNonHandleTypes(t *testing.T) {
	// E4.2 — Reject non-handle types
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"E4.2a integer target", `"data" =/=> 123`, "requires an HTTP request handle or SFTP file handle"},
		{"E4.2b string target", `"data" =/=> "string"`, "requires an HTTP request handle or SFTP file handle"},
		{"E4.2c array target", `"data" =/=> [1, 2]`, "requires an HTTP request handle or SFTP file handle"},
		{"E4.2d plain dict target", `"data" =/=> {a: 1}`, "requires an HTTP request handle or SFTP file handle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalHelper(tt.input)

			errObj, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected Error, got %T (%s)", result, result.Inspect())
			}
			if !strings.Contains(strings.ToLower(errObj.Message), strings.ToLower(tt.contains)) {
				t.Errorf("error message should contain %q, got %q", tt.contains, errObj.Message)
			}
		})
	}
}

// ============================================================================
// Layer 5: Evaluator Tests — =/=>> (Remote Append)
// ============================================================================

func TestRemoteAppendRejectsHTTP(t *testing.T) {
	// E5.1 — Reject HTTP targets
	server := newEchoServer()
	defer server.Close()

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "E5.1a HTTP has no append",
			input:    `"data" =/=>> JSON(url("` + server.URL + `"))`,
			contains: []string{"not supported for HTTP", "no append semantic"},
		},
		{
			name:     "E5.1b even with explicit method",
			input:    `"data" =/=>> JSON(url("` + server.URL + `")).post`,
			contains: []string{"not supported for HTTP"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalHelper(tt.input)

			errObj, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected Error, got %T (%s)", result, result.Inspect())
			}
			for _, substr := range tt.contains {
				if !strings.Contains(strings.ToLower(errObj.Message), strings.ToLower(substr)) {
					t.Errorf("error message should contain %q, got %q", substr, errObj.Message)
				}
			}
		})
	}
}

func TestRemoteAppendRejectsLocalFiles(t *testing.T) {
	// E5.2 — Reject local file handles
	tmpDir, err := os.MkdirTemp("", "parsley_remote_append_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	localFile := filepath.Join(tmpDir, "local.txt")

	input := `"data" =/=>> text("` + localFile + `")`
	result := testEvalRemoteWriteOp(input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T (%s)", result, result.Inspect())
	}
	// Should mention remote appends and suggest the local operator ==>>
	msg := strings.ToLower(errObj.Message)
	if !strings.Contains(msg, "remote appends") && !strings.Contains(msg, "==>>") {
		t.Errorf("error message should mention remote appends or ==>>, got %q", errObj.Message)
	}
}

// ============================================================================
// Layer 6: Evaluator Tests — Breaking Change to ==>
// ============================================================================

func TestWriteOperatorRejectsHTTPRequests(t *testing.T) {
	// E6.1 — ==> rejects HTTP request dicts
	server := newEchoServer()
	defer server.Close()

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "E6.1a plain POST",
			input:    `{a: 1} ==> JSON(url("` + server.URL + `"))`,
			contains: []string{"operator ==> is for local file writes", "use =/=>"},
		},
		{
			name:     "E6.1b PUT variant",
			input:    `{a: 1} ==> JSON(url("` + server.URL + `")).put`,
			contains: []string{"operator ==> is for local file writes", "use =/=>"},
		},
		{
			name:     "E6.1c text format",
			input:    `"data" ==> text(url("` + server.URL + `"))`,
			contains: []string{"operator ==> is for local file writes", "use =/=>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalHelper(tt.input)

			errObj, ok := result.(*evaluator.Error)
			if !ok {
				t.Fatalf("expected Error, got %T (%s)", result, result.Inspect())
			}
			for _, substr := range tt.contains {
				if !strings.Contains(errObj.Message, substr) {
					t.Errorf("error message should contain %q, got %q", substr, errObj.Message)
				}
			}
		})
	}
}

func TestAppendOperatorRejectsHTTPRequests(t *testing.T) {
	// ==>> should also reject HTTP request dicts
	server := newEchoServer()
	defer server.Close()

	input := `{a: 1} ==>> JSON(url("` + server.URL + `"))`
	result := testEvalHelper(input)

	errObj, ok := result.(*evaluator.Error)
	if !ok {
		t.Fatalf("expected Error, got %T (%s)", result, result.Inspect())
	}
	if !strings.Contains(errObj.Message, "==>>") || !strings.Contains(errObj.Message, "=/=>>") {
		t.Errorf("error message should mention ==>> and =/=>>, got %q", errObj.Message)
	}
}

func TestWriteOperatorStillWorksForLocalFiles(t *testing.T) {
	// E6.4 — ==> still works for local files (regression)
	tmpDir, err := os.MkdirTemp("", "parsley_local_write_regression_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("E6.4a local text write", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "test_text.txt")
		code := `"hello" ==> text("` + filePath + `")`
		result := testEvalRemoteWriteOp(code)
		if result != nil && result.Type() == "ERROR" {
			t.Fatalf("unexpected error: %s", result.Inspect())
		}
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(content) != "hello" {
			t.Errorf("expected 'hello', got %q", string(content))
		}
	})

	t.Run("E6.4b local JSON write", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "test_json.json")
		code := `{a: 1} ==> JSON("` + filePath + `")`
		result := testEvalRemoteWriteOp(code)
		if result != nil && result.Type() == "ERROR" {
			t.Fatalf("unexpected error: %s", result.Inspect())
		}
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if !strings.Contains(string(content), `"a"`) {
			t.Errorf("expected JSON with 'a' key, got %q", string(content))
		}
	})

	t.Run("E6.4c local text append", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "test_append.txt")
		// Create initial file
		os.WriteFile(filePath, []byte("initial"), 0644)
		code := `"-appended" ==>> text("` + filePath + `")`
		result := testEvalRemoteWriteOp(code)
		if result != nil && result.Type() == "ERROR" {
			t.Fatalf("unexpected error: %s", result.Inspect())
		}
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(content) != "initial-appended" {
			t.Errorf("expected 'initial-appended', got %q", string(content))
		}
	})
}
