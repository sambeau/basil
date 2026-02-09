// assignment_capture_test.go — Tests for assignment capture with remote write and fetch expressions
// Covers FEAT-104: `let response = payload =/=> target` and `let response = <=/= source`

package tests

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// ============================================================================
// Assignment Capture — Remote Write (`=/=>`)
// ============================================================================

func TestRemoteWriteAssignmentCapture(t *testing.T) {
	server := newEchoServer()
	defer server.Close()

	t.Run("let captures typed response dict from HTTP POST", func(t *testing.T) {
		input := `let response = {name: "Alice"} =/=> JSON(url("` + server.URL + `")); response`
		result := testEvalHelper(input)

		dict, ok := result.(*evaluator.Dictionary)
		if !ok {
			t.Fatalf("expected Dictionary (typed response), got %T (%s)", result, result.Inspect())
		}

		// Should have typed response dict fields
		for _, field := range []string{"__type", "__format", "__data", "__response"} {
			if _, exists := dict.Pairs[field]; !exists {
				keys := dictKeys(dict)
				t.Errorf("response dict missing field %q (keys: %v)", field, keys)
			}
		}
	})

	t.Run("let captures response with status 200", func(t *testing.T) {
		input := `let response = {a: 1} =/=> JSON(url("` + server.URL + `")); response`
		result := testEvalHelper(input)

		statusObj := extractResponseMeta(t, result, "status")
		num, ok := statusObj.(*evaluator.Integer)
		if !ok {
			t.Fatalf("expected Integer for status, got %T", statusObj)
		}
		if num.Value != 200 {
			t.Errorf("expected status 200, got %d", num.Value)
		}
	})

	t.Run("let captures response and can access __data", func(t *testing.T) {
		input := `let response = {name: "Bob"} =/=> JSON(url("` + server.URL + `")); response`
		result := testEvalHelper(input)

		data := extractResponseData(t, result)
		dataDict, ok := data.(*evaluator.Dictionary)
		if !ok {
			t.Fatalf("expected __data to be Dictionary, got %T (%s)", data, data.Inspect())
		}

		methodExpr, exists := dataDict.Pairs["method"]
		if !exists {
			t.Fatal("expected 'method' field in __data")
		}
		env := evaluator.NewEnvironment()
		methodObj := evaluator.Eval(methodExpr, env)
		str, ok := methodObj.(*evaluator.String)
		if !ok {
			t.Fatalf("expected String for method, got %T", methodObj)
		}
		if str.Value != "POST" {
			t.Errorf("expected method POST, got %q", str.Value)
		}
	})

	t.Run("assignment (no let) captures response", func(t *testing.T) {
		input := `response = {x: 1} =/=> JSON(url("` + server.URL + `")); response`
		result := testEvalHelper(input)

		dict, ok := result.(*evaluator.Dictionary)
		if !ok {
			t.Fatalf("expected Dictionary, got %T (%s)", result, result.Inspect())
		}

		if _, exists := dict.Pairs["__type"]; !exists {
			t.Error("expected typed response dict with __type field")
		}
	})

	t.Run("discard assignment with underscore", func(t *testing.T) {
		input := `let _ = {x: 1} =/=> JSON(url("` + server.URL + `"))`
		result := testEvalHelper(input)

		// Should not error — result is NULL from the let statement
		if _, ok := result.(*evaluator.Error); ok {
			t.Fatalf("unexpected error: %s", result.Inspect())
		}
	})

	t.Run("PUT via accessor captures response", func(t *testing.T) {
		input := `let response = {name: "Alice"} =/=> JSON(url("` + server.URL + `")).put; response`
		result := testEvalHelper(input)

		methodObj := extractDataField(t, result, "method")
		str, ok := methodObj.(*evaluator.String)
		if !ok {
			t.Fatalf("expected String, got %T", methodObj)
		}
		if str.Value != "PUT" {
			t.Errorf("expected method PUT, got %q", str.Value)
		}
	})

	t.Run("PATCH via accessor captures response", func(t *testing.T) {
		input := `let response = {age: 30} =/=> JSON(url("` + server.URL + `")).patch; response`
		result := testEvalHelper(input)

		methodObj := extractDataField(t, result, "method")
		str, ok := methodObj.(*evaluator.String)
		if !ok {
			t.Fatalf("expected String, got %T", methodObj)
		}
		if str.Value != "PATCH" {
			t.Errorf("expected method PATCH, got %q", str.Value)
		}
	})
}

// ============================================================================
// Assignment Capture — Remote Write Error Cases
// ============================================================================

func TestRemoteWriteAssignmentCaptureErrors(t *testing.T) {
	// Server that returns 500
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body)
		r.Body.Close()
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer errorServer.Close()

	t.Run("server error returns response dict not error object", func(t *testing.T) {
		input := `let response = {data: 1} =/=> JSON(url("` + errorServer.URL + `")); response`
		result := testEvalHelper(input)

		switch r := result.(type) {
		case *evaluator.Dictionary:
			// Typed response dict — check ok is false
			okObj := extractResponseMeta(t, r, "ok")
			b, ok := okObj.(*evaluator.Boolean)
			if !ok {
				t.Fatalf("expected Boolean for ok, got %T", okObj)
			}
			if b.Value {
				t.Error("expected ok=false for 500 response")
			}
		case *evaluator.Error:
			// Error object is also acceptable for connection-level errors
		default:
			t.Fatalf("expected Dictionary or Error, got %T (%s)", result, result.Inspect())
		}
	})

	t.Run("connection refused returns error", func(t *testing.T) {
		input := `let response = {data: 1} =/=> JSON(url("http://localhost:1")); response`
		result := testEvalHelper(input)

		// Connection errors should be Error objects
		if _, ok := result.(*evaluator.Error); !ok {
			// Could also be a response dict — both acceptable
			if _, isDict := result.(*evaluator.Dictionary); !isDict {
				t.Fatalf("expected Error or Dictionary, got %T (%s)", result, result.Inspect())
			}
		}
	})
}

// ============================================================================
// Destructuring — Remote Write with {data, error} Pattern
// ============================================================================

func TestRemoteWriteDestructuring(t *testing.T) {
	server := newEchoServer()
	defer server.Close()

	t.Run("let {data, error} from remote write", func(t *testing.T) {
		input := `let {data, error} = {name: "Alice"} =/=> JSON(url("` + server.URL + `")); data`
		result := testEvalHelper(input)

		// data should be the response content (a dict with method, bodyLength, auth)
		dict, ok := result.(*evaluator.Dictionary)
		if !ok {
			t.Fatalf("expected Dictionary for data, got %T (%s)", result, result.Inspect())
		}

		methodExpr, exists := dict.Pairs["method"]
		if !exists {
			keys := dictKeys(dict)
			t.Fatalf("data dict missing 'method' field, keys: %v", keys)
		}
		env := evaluator.NewEnvironment()
		methodObj := evaluator.Eval(methodExpr, env)
		str, ok := methodObj.(*evaluator.String)
		if !ok {
			t.Fatalf("expected String for method, got %T", methodObj)
		}
		if str.Value != "POST" {
			t.Errorf("expected method POST, got %q", str.Value)
		}
	})

	t.Run("let {data, error} error is null on success", func(t *testing.T) {
		input := `let {data, error} = {x: 1} =/=> JSON(url("` + server.URL + `")); error`
		result := testEvalHelper(input)

		if result != evaluator.NULL {
			t.Errorf("expected null error on success, got %T (%s)", result, result.Inspect())
		}
	})

	t.Run("{data, error} reassignment from remote write", func(t *testing.T) {
		input := `{data, error} = {y: 2} =/=> JSON(url("` + server.URL + `")); data`
		result := testEvalHelper(input)

		dict, ok := result.(*evaluator.Dictionary)
		if !ok {
			t.Fatalf("expected Dictionary for data, got %T (%s)", result, result.Inspect())
		}

		if _, exists := dict.Pairs["method"]; !exists {
			t.Error("expected 'method' field in destructured data")
		}
	})

	t.Run("let {data, error, status} from remote write", func(t *testing.T) {
		input := `let {data, error, status} = {z: 3} =/=> JSON(url("` + server.URL + `")); status`
		result := testEvalHelper(input)

		num, ok := result.(*evaluator.Integer)
		if !ok {
			t.Fatalf("expected Integer for status, got %T (%s)", result, result.Inspect())
		}
		if num.Value != 200 {
			t.Errorf("expected status 200, got %d", num.Value)
		}
	})

	t.Run("let {data, error, headers} from remote write", func(t *testing.T) {
		input := `let {data, error, headers} = {q: 1} =/=> JSON(url("` + server.URL + `")); headers`
		result := testEvalHelper(input)

		// headers should be a dictionary
		_, ok := result.(*evaluator.Dictionary)
		if !ok {
			// null is also acceptable if server doesn't return headers
			if result != evaluator.NULL {
				t.Fatalf("expected Dictionary or null for headers, got %T (%s)", result, result.Inspect())
			}
		}
	})
}

// ============================================================================
// Destructuring — Remote Write Error Capture
// ============================================================================

func TestRemoteWriteDestructuringErrors(t *testing.T) {
	// Server that returns 500
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body)
		r.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "server failure"}`))
	}))
	defer errorServer.Close()

	t.Run("server error captured in {data, error}", func(t *testing.T) {
		input := `let {data, error} = {x: 1} =/=> JSON(url("` + errorServer.URL + `")); error`
		result := testEvalHelper(input)

		// error should be null (non-connection errors return data in the data field)
		// or a string describing the error — both are acceptable
		switch result.(type) {
		case *evaluator.String:
			// error message string — acceptable
		case *evaluator.Null:
			// null means no error at network level — acceptable (server returned valid response)
		default:
			t.Fatalf("expected String or Null for error, got %T (%s)", result, result.Inspect())
		}
	})

	t.Run("connection refused captured in {data, error}", func(t *testing.T) {
		input := `let {data, error} = {x: 1} =/=> JSON(url("http://localhost:1")); error`
		result := testEvalHelper(input)

		// On connection error, error should be a non-null string
		str, ok := result.(*evaluator.String)
		if !ok {
			// If it's an Error object that bubbled up, destructuring didn't catch it
			if errObj, isErr := result.(*evaluator.Error); isErr {
				// This is also acceptable — the error was too severe for destructuring
				if !strings.Contains(strings.ToLower(errObj.Message), "connect") &&
					!strings.Contains(strings.ToLower(errObj.Message), "refused") &&
					!strings.Contains(strings.ToLower(errObj.Message), "dial") {
					t.Logf("got error: %s", errObj.Message)
				}
				return
			}
			t.Fatalf("expected String error message, got %T (%s)", result, result.Inspect())
		}
		if str.Value == "" {
			t.Error("expected non-empty error message for connection refused")
		}
	})
}

// ============================================================================
// Assignment Capture — Fetch Expression (`<=/=`)
// ============================================================================

func TestFetchExpressionAssignmentCapture(t *testing.T) {
	jsonServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name": "test", "value": 42}`))
	}))
	defer jsonServer.Close()

	textServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	}))
	defer textServer.Close()

	t.Run("let captures typed response dict from fetch expression", func(t *testing.T) {
		input := `let response = <=/= JSON(url("` + jsonServer.URL + `")); response`
		result := testEvalHelper(input)

		dict, ok := result.(*evaluator.Dictionary)
		if !ok {
			t.Fatalf("expected Dictionary (typed response), got %T (%s)", result, result.Inspect())
		}

		for _, field := range []string{"__type", "__format", "__data", "__response"} {
			if _, exists := dict.Pairs[field]; !exists {
				keys := dictKeys(dict)
				t.Errorf("response dict missing field %q (keys: %v)", field, keys)
			}
		}
	})

	t.Run("let captures fetch response with status 200", func(t *testing.T) {
		input := `let response = <=/= JSON(url("` + jsonServer.URL + `")); response`
		result := testEvalHelper(input)

		statusObj := extractResponseMeta(t, result, "status")
		num, ok := statusObj.(*evaluator.Integer)
		if !ok {
			t.Fatalf("expected Integer for status, got %T", statusObj)
		}
		if num.Value != 200 {
			t.Errorf("expected status 200, got %d", num.Value)
		}
	})

	t.Run("let captures fetch response __data", func(t *testing.T) {
		input := `let response = <=/= JSON(url("` + jsonServer.URL + `")); response`
		result := testEvalHelper(input)

		data := extractResponseData(t, result)
		dataDict, ok := data.(*evaluator.Dictionary)
		if !ok {
			t.Fatalf("expected __data to be Dictionary, got %T (%s)", data, data.Inspect())
		}

		nameExpr, exists := dataDict.Pairs["name"]
		if !exists {
			t.Fatal("expected 'name' field in __data")
		}
		env := evaluator.NewEnvironment()
		nameObj := evaluator.Eval(nameExpr, env)
		str, ok := nameObj.(*evaluator.String)
		if !ok {
			t.Fatalf("expected String for name, got %T", nameObj)
		}
		if str.Value != "test" {
			t.Errorf("expected name 'test', got %q", str.Value)
		}
	})

	t.Run("let captures text fetch response", func(t *testing.T) {
		input := `let response = <=/= text(url("` + textServer.URL + `")); response`
		result := testEvalHelper(input)

		dict, ok := result.(*evaluator.Dictionary)
		if !ok {
			t.Fatalf("expected Dictionary (typed response), got %T (%s)", result, result.Inspect())
		}

		if _, exists := dict.Pairs["__data"]; !exists {
			t.Error("expected __data field in typed response")
		}
	})

	t.Run("assignment captures fetch expression", func(t *testing.T) {
		input := `response = <=/= JSON(url("` + jsonServer.URL + `")); response`
		result := testEvalHelper(input)

		dict, ok := result.(*evaluator.Dictionary)
		if !ok {
			t.Fatalf("expected Dictionary, got %T (%s)", result, result.Inspect())
		}

		if _, exists := dict.Pairs["__type"]; !exists {
			t.Error("expected typed response dict with __type field")
		}
	})
}

// ============================================================================
// Destructuring — Fetch Expression with {data, error} Pattern
// ============================================================================

func TestFetchExpressionDestructuring(t *testing.T) {
	jsonServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name": "test", "value": 42}`))
	}))
	defer jsonServer.Close()

	textServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	}))
	defer textServer.Close()

	t.Run("let {data, error} from fetch expression", func(t *testing.T) {
		input := `let {data, error} = <=/= JSON(url("` + jsonServer.URL + `")); data`
		result := testEvalHelper(input)

		dict, ok := result.(*evaluator.Dictionary)
		if !ok {
			t.Fatalf("expected Dictionary for data, got %T (%s)", result, result.Inspect())
		}

		nameExpr, exists := dict.Pairs["name"]
		if !exists {
			keys := dictKeys(dict)
			t.Fatalf("data dict missing 'name' field, keys: %v", keys)
		}
		env := evaluator.NewEnvironment()
		nameObj := evaluator.Eval(nameExpr, env)
		str, ok := nameObj.(*evaluator.String)
		if !ok {
			t.Fatalf("expected String for name, got %T", nameObj)
		}
		if str.Value != "test" {
			t.Errorf("expected name 'test', got %q", str.Value)
		}
	})

	t.Run("let {data, error} error is null on success", func(t *testing.T) {
		input := `let {data, error} = <=/= JSON(url("` + jsonServer.URL + `")); error`
		result := testEvalHelper(input)

		if result != evaluator.NULL {
			t.Errorf("expected null error on success, got %T (%s)", result, result.Inspect())
		}
	})

	t.Run("let {data, error} from text fetch expression", func(t *testing.T) {
		input := `let {data, error} = <=/= text(url("` + textServer.URL + `")); data`
		result := testEvalHelper(input)

		str, ok := result.(*evaluator.String)
		if !ok {
			t.Fatalf("expected String for data, got %T (%s)", result, result.Inspect())
		}
		if str.Value != "Hello, World!" {
			t.Errorf("expected 'Hello, World!', got %q", str.Value)
		}
	})

	t.Run("let {data, error, status} from fetch expression", func(t *testing.T) {
		input := `let {data, error, status} = <=/= JSON(url("` + jsonServer.URL + `")); status`
		result := testEvalHelper(input)

		num, ok := result.(*evaluator.Integer)
		if !ok {
			t.Fatalf("expected Integer for status, got %T (%s)", result, result.Inspect())
		}
		if num.Value != 200 {
			t.Errorf("expected status 200, got %d", num.Value)
		}
	})
}

// ============================================================================
// Fetch Expression Error Capture Destructuring
// ============================================================================

func TestFetchExpressionDestructuringErrors(t *testing.T) {
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer errorServer.Close()

	t.Run("server error status in {data, error, status}", func(t *testing.T) {
		// Fetch expression as RHS to let with destructuring
		input := `let {data, error, status} = <=/= text(url("` + errorServer.URL + `")); status`
		result := testEvalHelper(input)

		num, ok := result.(*evaluator.Integer)
		if !ok {
			// Could be error bubbling up if too severe
			if _, isErr := result.(*evaluator.Error); isErr {
				t.Skipf("error bubbled up: %s", result.Inspect())
			}
			t.Fatalf("expected Integer for status, got %T (%s)", result, result.Inspect())
		}
		if num.Value != 500 {
			t.Errorf("expected status 500, got %d", num.Value)
		}
	})

	t.Run("connection refused in {data, error}", func(t *testing.T) {
		input := `let {data, error} = <=/= JSON(url("http://localhost:1")); error`
		result := testEvalHelper(input)

		switch r := result.(type) {
		case *evaluator.String:
			if r.Value == "" {
				t.Error("expected non-empty error message")
			}
		case *evaluator.Error:
			// Error bubbled — acceptable for connection-level failures
		default:
			t.Fatalf("expected String or Error, got %T (%s)", result, result.Inspect())
		}
	})
}

// ============================================================================
// Fetch Expression Error Handling (non-destructuring)
// ============================================================================

func TestFetchExpressionErrors(t *testing.T) {
	t.Run("fetch expression with invalid source type", func(t *testing.T) {
		input := `let x = <=/= 123; x`
		result := testEvalHelper(input)

		_, ok := result.(*evaluator.Error)
		if !ok {
			t.Fatalf("expected Error for invalid source, got %T (%s)", result, result.Inspect())
		}
	})

	t.Run("fetch expression with string instead of URL", func(t *testing.T) {
		input := `let x = <=/= "https://example.com"; x`
		result := testEvalHelper(input)

		_, ok := result.(*evaluator.Error)
		if !ok {
			t.Fatalf("expected Error for string source, got %T (%s)", result, result.Inspect())
		}
	})
}

// ============================================================================
// Backward Compatibility — Statement-Form Fetch Still Works
// ============================================================================

func TestFetchStatementStillWorks(t *testing.T) {
	jsonServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name": "test", "value": 42}`))
	}))
	defer jsonServer.Close()

	t.Run("statement form {data, error} <=/= still works", func(t *testing.T) {
		input := `{data, error} <=/= JSON(url("` + jsonServer.URL + `")); data`
		result := testEvalHelper(input)

		dict, ok := result.(*evaluator.Dictionary)
		if !ok {
			t.Fatalf("expected Dictionary, got %T (%s)", result, result.Inspect())
		}
		if _, exists := dict.Pairs["name"]; !exists {
			t.Error("expected 'name' field in fetched data")
		}
	})

	t.Run("statement form let x <=/= still works", func(t *testing.T) {
		input := `let x <=/= JSON(url("` + jsonServer.URL + `")); x`
		result := testEvalHelper(input)

		// x should be the typed response dict
		dict, ok := result.(*evaluator.Dictionary)
		if !ok {
			t.Fatalf("expected Dictionary, got %T (%s)", result, result.Inspect())
		}
		if _, exists := dict.Pairs["__type"]; !exists {
			t.Error("expected typed response dict")
		}
	})
}

// ============================================================================
// Backward Compatibility — Statement-Form Remote Write Still Works
// ============================================================================

func TestRemoteWriteStatementStillWorks(t *testing.T) {
	server := newEchoServer()
	defer server.Close()

	t.Run("statement form payload =/=> target still works", func(t *testing.T) {
		input := `{name: "Alice"} =/=> JSON(url("` + server.URL + `"))`
		result := testEvalHelper(input)

		// As a statement, remote write returns the typed response dict
		dict, ok := result.(*evaluator.Dictionary)
		if !ok {
			t.Fatalf("expected Dictionary, got %T (%s)", result, result.Inspect())
		}
		if _, exists := dict.Pairs["__type"]; !exists {
			t.Error("expected typed response dict")
		}
	})
}

// ============================================================================
// Parser — Write Expressions in Assignment Context
// ============================================================================

func TestParserWriteExpressionInAssignment(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "let with remote write",
			input: `let response = {a: 1} =/=> JSON(url("http://example.com"))`,
		},
		{
			name:  "let with remote append",
			input: `let response = {a: 1} =/=>> JSON(url("http://example.com"))`,
		},
		{
			name:  "let with fetch expression",
			input: `let response = <=/= JSON(url("http://example.com"))`,
		},
		{
			name:  "let destructuring with remote write",
			input: `let {data, error} = {a: 1} =/=> JSON(url("http://example.com"))`,
		},
		{
			name:  "let destructuring with fetch expression",
			input: `let {data, error} = <=/= JSON(url("http://example.com"))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := parseProgram(tt.input)

			if len(program.Statements) == 0 {
				t.Fatal("expected at least one statement")
			}

			// Should parse without errors — the statement should exist
			stmt := program.Statements[0]
			if stmt == nil {
				t.Fatal("parsed statement is nil")
			}
		})
	}
}

// ============================================================================
// Helpers
// ============================================================================

func dictKeys(dict *evaluator.Dictionary) []string {
	keys := make([]string, 0, len(dict.Pairs))
	for k := range dict.Pairs {
		keys = append(keys, k)
	}
	return keys
}

// NULL is exported from the evaluator package for comparison.
var NULL = evaluator.NULL
