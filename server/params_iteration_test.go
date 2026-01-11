package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// evalWithParams evaluates code with @params populated from an HTTP request
func evalWithParams(input string, req *http.Request) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return &evaluator.Error{Message: p.Errors()[0]}
	}

	env := evaluator.NewEnvironment()
	params := buildParams(req, env)
	env.Set("@params", params)

	return evaluator.Eval(program, env)
}

func TestParamsIteration(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		contentType string
		queryParams url.Values
		body        string
		script      string
		validate    func(t *testing.T, result evaluator.Object)
	}{
		{
			name:   "iterate over query params",
			method: "GET",
			queryParams: url.Values{
				"name": []string{"Alice"},
				"age":  []string{"30"},
				"city": []string{"NYC"},
			},
			script: `for(k,v in @params){ k + "=" + v }`,
			validate: func(t *testing.T, result evaluator.Object) {
				arr, ok := result.(*evaluator.Array)
				if !ok {
					t.Fatalf("expected Array, got %T", result)
				}
				// Should have 3 elements (keys are sorted: age, city, name)
				if len(arr.Elements) != 3 {
					t.Errorf("expected 3 elements, got %d", len(arr.Elements))
				}
				// Check first element
				if str, ok := arr.Elements[0].(*evaluator.String); ok {
					if str.Value != "age=30" {
						t.Errorf("expected 'age=30', got %q", str.Value)
					}
				}
			},
		},
		{
			name:        "count form params",
			method:      "POST",
			contentType: "application/x-www-form-urlencoded",
			body:        "username=bob&password=secret&remember=true",
			script:      `let count = 0; for(k,v in @params){ count = count + 1; null }; count`,
			validate: func(t *testing.T, result evaluator.Object) {
				arr, ok := result.(*evaluator.Array)
				if !ok {
					t.Fatalf("expected Array, got %T", result)
				}
				// Get the last element (count)
				if len(arr.Elements) == 0 {
					t.Fatalf("expected array with elements, got empty array")
				}
				num, ok := arr.Elements[len(arr.Elements)-1].(*evaluator.Integer)
				if !ok {
					t.Fatalf("expected last element to be Integer, got %T", arr.Elements[len(arr.Elements)-1])
				}
				if num.Value != 3 {
					t.Errorf("expected count of 3, got %d", num.Value)
				}
			},
		},
		{
			name:   "get values from params",
			method: "GET",
			queryParams: url.Values{
				"q":    []string{"search"},
				"page": []string{"1"},
			},
			script: `for(v in @params){ v }`,
			validate: func(t *testing.T, result evaluator.Object) {
				arr, ok := result.(*evaluator.Array)
				if !ok {
					t.Fatalf("expected Array, got %T", result)
				}
				// Should have 2 elements (values in key-sorted order: page->1, q->search)
				if len(arr.Elements) != 2 {
					t.Errorf("expected 2 elements, got %d", len(arr.Elements))
				}
				// Check values are from sorted keys (page=1, q=search)
				if str, ok := arr.Elements[0].(*evaluator.String); ok {
					if str.Value != "1" {
						t.Errorf("expected first value '1', got %q", str.Value)
					}
				}
				if str, ok := arr.Elements[1].(*evaluator.String); ok {
					if str.Value != "search" {
						t.Errorf("expected second value 'search', got %q", str.Value)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, "/?"+tt.queryParams.Encode(), strings.NewReader(tt.body))
				if tt.contentType != "" {
					req.Header.Set("Content-Type", tt.contentType)
				}
			} else {
				req = httptest.NewRequest(tt.method, "/?"+tt.queryParams.Encode(), nil)
			}

			// Execute script
			result := evalWithParams(tt.script, req)

			// Check for errors
			if err, ok := result.(*evaluator.Error); ok {
				t.Fatalf("Script evaluation failed: %v", err.Message)
			}

			tt.validate(t, result)
		})
	}
}

func TestParamsIterationEmpty(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	script := `let count = 0; for(k in @params){ count = count + 1; null }; count`

	result := evalWithParams(script, req)

	// Check for errors
	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("Script evaluation failed: %v", err.Message)
	}

	// For loop returns an array
	arr, ok := result.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	// Get the last element (count)
	if len(arr.Elements) == 0 {
		t.Fatalf("expected array with elements, got empty array")
	}

	num, ok := arr.Elements[len(arr.Elements)-1].(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected last element to be Integer, got %T", arr.Elements[len(arr.Elements)-1])
	}

	if num.Value != 0 {
		t.Errorf("Empty params iteration: got count %d, want 0", num.Value)
	}
}
