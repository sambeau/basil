package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// isPartRequest checks if the request is for a Part view
// Part requests have a _view query parameter
func isPartRequest(r *http.Request) bool {
	return r.URL.Query().Get("_view") != ""
}

// handlePartRequest handles requests for Part views
// Returns HTML fragment (no wrapper div - JS will replace innerHTML)
func (h *parsleyHandler) handlePartRequest(w http.ResponseWriter, r *http.Request, scriptPath string, env *evaluator.Environment) {
	// Extract view name from query params
	viewName := r.URL.Query().Get("_view")
	if viewName == "" {
		http.Error(w, "Missing _view parameter", http.StatusBadRequest)
		return
	}

	// Read and parse the Part file
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read Part file: %v", err), http.StatusInternalServerError)
		return
	}

	l := lexer.NewWithFilename(string(content), scriptPath)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		http.Error(w, fmt.Sprintf("Parse error: %s", p.Errors()[0]), http.StatusInternalServerError)
		return
	}

	// Execute the Part file to get exports
	result := evaluator.Eval(program, env)
	if result.Type() == evaluator.ERROR_OBJ {
		errObj := result.(*evaluator.Error)
		http.Error(w, errObj.Message, http.StatusInternalServerError)
		return
	}

	// Get the view function from the environment (exports are in env.store)
	viewObj, hasView := env.Get(viewName)
	if !hasView {
		http.Error(w, fmt.Sprintf("Part does not export view '%s'", viewName), http.StatusNotFound)
		return
	}

	fnObj, ok := viewObj.(*evaluator.Function)
	if !ok {
		http.Error(w, fmt.Sprintf("Part view '%s' is not a function", viewName), http.StatusInternalServerError)
		return
	}

	// Parse props from query params and form body
	props, err := h.parsePartProps(r, env)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse props: %v", err), http.StatusBadRequest)
		return
	}

	// Call the view function with props using ApplyFunctionWithEnv
	// This properly handles all parameter types including destructuring patterns like fn({width})
	result = evaluator.ApplyFunctionWithEnv(fnObj, []evaluator.Object{props}, env)

	// Unwrap return values
	if retVal, ok := result.(*evaluator.ReturnValue); ok {
		result = retVal.Value
	}

	if result.Type() == evaluator.ERROR_OBJ {
		errObj := result.(*evaluator.Error)
		http.Error(w, errObj.Message, http.StatusInternalServerError)
		return
	}

	// Convert result to HTML string
	html := objectToTemplateString(result)

	// Return the HTML fragment (no wrapper - JS will handle that)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// parsePartProps parses props from query parameters and form body
// Applies the same type coercion as form handling
func (h *parsleyHandler) parsePartProps(r *http.Request, env *evaluator.Environment) (*evaluator.Dictionary, error) {
	props := make(map[string]ast.Expression)

	// Parse query params first
	for key, values := range r.URL.Query() {
		if key == "_view" {
			continue // Skip the special _view parameter
		}
		if len(values) > 0 {
			props[key] = createLiteralFromValue(coerceFormValue(values[0]))
		}
	}

	// For POST requests, parse form body and merge
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			return nil, err
		}

		for key, values := range r.PostForm {
			if len(values) > 0 {
				props[key] = createLiteralFromValue(coerceFormValue(values[0]))
			}
		}
	}

	return &evaluator.Dictionary{Pairs: props, Env: env}, nil
}

// coerceFormValue applies type coercion to form values
// Same logic as form handling: numbers, booleans, etc.
func coerceFormValue(value string) evaluator.Object {
	// Try boolean
	if value == "true" || value == "on" {
		return &evaluator.Boolean{Value: true}
	}
	if value == "false" || value == "off" || value == "" {
		return &evaluator.Boolean{Value: false}
	}

	// Try integer
	if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
		return &evaluator.Integer{Value: intVal}
	}

	// Try float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return &evaluator.Float{Value: floatVal}
	}

	// Default to string
	return &evaluator.String{Value: value}
}

// createLiteralFromValue converts an evaluator.Object to an ast.Expression literal
func createLiteralFromValue(obj evaluator.Object) ast.Expression {
	switch v := obj.(type) {
	case *evaluator.Integer:
		return &ast.IntegerLiteral{Value: v.Value}
	case *evaluator.Float:
		return &ast.FloatLiteral{Value: v.Value}
	case *evaluator.Boolean:
		return &ast.Boolean{Value: v.Value}
	case *evaluator.String:
		return &ast.StringLiteral{Value: v.Value}
	default:
		return &ast.StringLiteral{Value: obj.Inspect()}
	}
}

// objectToTemplateString converts an evaluator object to a string for HTML output
func objectToTemplateString(obj evaluator.Object) string {
	switch v := obj.(type) {
	case *evaluator.String:
		return v.Value
	case *evaluator.Integer:
		return fmt.Sprintf("%d", v.Value)
	case *evaluator.Float:
		return fmt.Sprintf("%g", v.Value)
	case *evaluator.Boolean:
		if v.Value {
			return "true"
		}
		return "false"
	case *evaluator.Array:
		// Concatenate array elements (for multiple expressions in block)
		var result string
		for _, elem := range v.Elements {
			result += objectToTemplateString(elem)
		}
		return result
	default:
		return obj.Inspect()
	}
}
