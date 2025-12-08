package server

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
)

// TestMatch_BasicParameter tests basic :param capture
func TestMatch_BasicParameter(t *testing.T) {
	code := `match("/users/123", "/users/:id")`
	result, err := parsley.Eval(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Value == nil || result.Value.Type() != evaluator.DICTIONARY_OBJ {
		t.Fatalf("expected dictionary, got %s", result.Value.Type())
	}

	dict := result.Value.(*evaluator.Dictionary)
	idExpr, ok := dict.Pairs["id"]
	if !ok {
		t.Fatal("expected 'id' key in result")
	}

	idVal := evaluator.Eval(idExpr, evaluator.NewEnvironment())
	if idVal.Inspect() != "123" {
		t.Errorf("expected id='123', got %q", idVal.Inspect())
	}
}

// TestMatch_MultipleParameters tests multiple :param captures
func TestMatch_MultipleParameters(t *testing.T) {
	code := `match("/users/42/posts/99", "/users/:userId/posts/:postId")`
	result, err := parsley.Eval(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dict := result.Value.(*evaluator.Dictionary)

	// Check userId
	userIdExpr, ok := dict.Pairs["userId"]
	if !ok {
		t.Fatal("expected 'userId' key")
	}
	userIdVal := evaluator.Eval(userIdExpr, evaluator.NewEnvironment())
	if userIdVal.Inspect() != "42" {
		t.Errorf("expected userId='42', got %q", userIdVal.Inspect())
	}

	// Check postId
	postIdExpr, ok := dict.Pairs["postId"]
	if !ok {
		t.Fatal("expected 'postId' key")
	}
	postIdVal := evaluator.Eval(postIdExpr, evaluator.NewEnvironment())
	if postIdVal.Inspect() != "99" {
		t.Errorf("expected postId='99', got %q", postIdVal.Inspect())
	}
}

// TestMatch_GlobCapture tests *name glob capture
func TestMatch_GlobCapture(t *testing.T) {
	code := `match("/files/docs/2025/report.pdf", "/files/*path")`
	result, err := parsley.Eval(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dict := result.Value.(*evaluator.Dictionary)
	pathExpr, ok := dict.Pairs["path"]
	if !ok {
		t.Fatal("expected 'path' key")
	}

	pathVal := evaluator.Eval(pathExpr, evaluator.NewEnvironment())
	arr, ok := pathVal.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected array, got %s", pathVal.Type())
	}

	expected := []string{"docs", "2025", "report.pdf"}
	if len(arr.Elements) != len(expected) {
		t.Fatalf("expected %d elements, got %d", len(expected), len(arr.Elements))
	}

	for i, exp := range expected {
		if arr.Elements[i].Inspect() != exp {
			t.Errorf("expected element[%d]=%q, got %q", i, exp, arr.Elements[i].Inspect())
		}
	}
}

// TestMatch_LiteralOnly tests pattern with no parameters
func TestMatch_LiteralOnly(t *testing.T) {
	code := `match("/users", "/users")`
	result, err := parsley.Eval(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Value.Type() != evaluator.DICTIONARY_OBJ {
		t.Fatalf("expected dictionary, got %s", result.Value.Type())
	}

	dict := result.Value.(*evaluator.Dictionary)
	if len(dict.Pairs) != 0 {
		t.Errorf("expected empty dictionary, got %d keys", len(dict.Pairs))
	}
}

// TestMatch_NoMatch tests pattern that doesn't match
func TestMatch_NoMatch(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{"wrong path prefix", `match("/posts/123", "/users/:id")`},
		{"missing segment", `match("/users", "/users/:id")`},
		{"extra segment", `match("/users/123/extra", "/users/:id")`},
		{"case sensitive", `match("/Users/123", "/users/:id")`},
		{"empty id segment", `match("/users/", "/users/:id")`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsley.Eval(tt.code)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Value.Type() != evaluator.NULL_OBJ {
				t.Errorf("expected null, got %s: %s", result.Value.Type(), result.Value.Inspect())
			}
		})
	}
}

// TestMatch_TrailingSlash tests trailing slash handling
func TestMatch_TrailingSlash(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{"path with trailing slash", `match("/users/123/", "/users/:id")`, true},
		{"pattern with trailing slash", `match("/users/123", "/users/:id/")`, true},
		{"both with trailing slash", `match("/users/123/", "/users/:id/")`, true},
		{"literal with trailing slash", `match("/users/", "/users")`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsley.Eval(tt.code)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			isMatch := result.Value.Type() == evaluator.DICTIONARY_OBJ
			if isMatch != tt.expected {
				t.Errorf("expected match=%v, got %v (result: %s)", tt.expected, isMatch, result.Value.Inspect())
			}
		})
	}
}

// TestMatch_GlobWithParam tests mixed :param and *glob
func TestMatch_GlobWithParam(t *testing.T) {
	code := `match("/users/123/files/a/b/c", "/users/:id/files/*rest")`
	result, err := parsley.Eval(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dict := result.Value.(*evaluator.Dictionary)

	// Check id
	idExpr, ok := dict.Pairs["id"]
	if !ok {
		t.Fatal("expected 'id' key")
	}
	idVal := evaluator.Eval(idExpr, evaluator.NewEnvironment())
	if idVal.Inspect() != "123" {
		t.Errorf("expected id='123', got %q", idVal.Inspect())
	}

	// Check rest (glob)
	restExpr, ok := dict.Pairs["rest"]
	if !ok {
		t.Fatal("expected 'rest' key")
	}
	restVal := evaluator.Eval(restExpr, evaluator.NewEnvironment())
	arr, ok := restVal.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected array for rest, got %s", restVal.Type())
	}

	expected := []string{"a", "b", "c"}
	if len(arr.Elements) != len(expected) {
		t.Fatalf("expected %d elements, got %d", len(expected), len(arr.Elements))
	}
}

// TestMatch_EmptyGlob tests glob that matches no segments
func TestMatch_EmptyGlob(t *testing.T) {
	code := `match("/api", "/api/*rest")`
	result, err := parsley.Eval(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dict := result.Value.(*evaluator.Dictionary)
	restExpr, ok := dict.Pairs["rest"]
	if !ok {
		t.Fatal("expected 'rest' key")
	}

	restVal := evaluator.Eval(restExpr, evaluator.NewEnvironment())
	arr, ok := restVal.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected array, got %s", restVal.Type())
	}

	if len(arr.Elements) != 0 {
		t.Errorf("expected empty array, got %d elements", len(arr.Elements))
	}
}

// TestMatch_RootPath tests matching root path
func TestMatch_RootPath(t *testing.T) {
	code := `match("/", "/")`
	result, err := parsley.Eval(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Value.Type() != evaluator.DICTIONARY_OBJ {
		t.Fatalf("expected dictionary, got %s", result.Value.Type())
	}
}

// TestMatch_CatchAllGlob tests /*all pattern
func TestMatch_CatchAllGlob(t *testing.T) {
	code := `match("/any/path/here", "/*all")`
	result, err := parsley.Eval(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dict := result.Value.(*evaluator.Dictionary)
	allExpr, ok := dict.Pairs["all"]
	if !ok {
		t.Fatal("expected 'all' key")
	}

	allVal := evaluator.Eval(allExpr, evaluator.NewEnvironment())
	arr, ok := allVal.(*evaluator.Array)
	if !ok {
		t.Fatalf("expected array, got %s", allVal.Type())
	}

	expected := []string{"any", "path", "here"}
	if len(arr.Elements) != len(expected) {
		t.Fatalf("expected %d elements, got %d", len(expected), len(arr.Elements))
	}
}

// TestMatch_WithDestructuring tests match result usage
func TestMatch_WithDestructuring(t *testing.T) {
	code := `
let params = match("/users/42/posts/99", "/users/:userId/posts/:postId")
params.userId + "-" + params.postId
`
	result, err := parsley.Eval(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Value.Inspect() != "42-99" {
		t.Errorf("expected '42-99', got %q", result.Value.Inspect())
	}
}

// TestMatch_InvalidArguments tests error handling
func TestMatch_InvalidArguments(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{"no arguments", `match()`},
		{"one argument", `match("/users")`},
		{"three arguments", `match("/users", "/users", "/users")`},
		{"number as path", `match(123, "/users")`},
		{"number as pattern", `match("/users", 123)`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsley.Eval(tt.code)
			// Either err is returned, or result is an error object
			if err == nil && (result.Value == nil || result.Value.Type() != evaluator.ERROR_OBJ) {
				t.Errorf("expected error, got %s: %s", result.Value.Type(), result.Value.Inspect())
			}
		})
	}
}

// TestMatch_ThreeParameters tests pattern with three parameters
func TestMatch_ThreeParameters(t *testing.T) {
	code := `match("/a/b/c", "/:x/:y/:z")`
	result, err := parsley.Eval(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dict := result.Value.(*evaluator.Dictionary)

	for _, key := range []string{"x", "y", "z"} {
		if _, ok := dict.Pairs[key]; !ok {
			t.Errorf("expected '%s' key in result", key)
		}
	}
}

// TestMatch_SpecialCharactersInSegment tests URL-encoded paths
func TestMatch_SpecialCharactersInSegment(t *testing.T) {
	code := `match("/users/john%20doe", "/users/:name")`
	result, err := parsley.Eval(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dict := result.Value.(*evaluator.Dictionary)
	nameExpr, ok := dict.Pairs["name"]
	if !ok {
		t.Fatal("expected 'name' key")
	}

	nameVal := evaluator.Eval(nameExpr, evaluator.NewEnvironment())
	// URL encoding is preserved (decoding is caller's responsibility)
	if nameVal.Inspect() != "john%20doe" {
		t.Errorf("expected 'john%%20doe', got %q", nameVal.Inspect())
	}
}
