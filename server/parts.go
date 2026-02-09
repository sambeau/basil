package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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

// logPartError logs a Part error to the dev log if available
func (h *parsleyHandler) logPartError(scriptPath, viewName, errMsg string) {
	if h.server.devLog == nil {
		return
	}
	filename := filepath.Base(scriptPath)
	callRepr := fmt.Sprintf("Part error: %s → %s()", filename, viewName)
	_ = h.server.devLog.LogFromEvaluator("", "warn", scriptPath, 0, callRepr, errMsg)
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
		errMsg := fmt.Sprintf("Failed to read Part file: %v", err)
		h.logPartError(scriptPath, viewName, errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	l := lexer.NewWithFilename(string(content), scriptPath)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		errMsg := fmt.Sprintf("Parse error: %s", p.Errors()[0])
		h.logPartError(scriptPath, viewName, errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	// Execute the Part file to get exports
	result := evaluator.Eval(program, env)
	if result.Type() == evaluator.ERROR_OBJ {
		errObj := result.(*evaluator.Error)
		h.logPartError(scriptPath, viewName, errObj.Message)
		http.Error(w, errObj.Message, http.StatusInternalServerError)
		return
	}

	// Get the view function from the environment (exports are in env.store)
	viewObj, hasView := env.Get(viewName)
	if !hasView {
		errMsg := fmt.Sprintf("Part does not export view '%s'", viewName)
		h.logPartError(scriptPath, viewName, errMsg)
		http.Error(w, errMsg, http.StatusNotFound)
		return
	}

	fnObj, ok := viewObj.(*evaluator.Function)
	if !ok {
		errMsg := fmt.Sprintf("Part view '%s' is not a function", viewName)
		h.logPartError(scriptPath, viewName, errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	// Parse props from query params and form body
	props, err := h.parsePartProps(r, env)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to parse props: %v", err)
		h.logPartError(scriptPath, viewName, errMsg)
		http.Error(w, errMsg, http.StatusBadRequest)
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
		h.logPartError(scriptPath, viewName, errObj.Message)
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
// Applies the same type coercion as form handling, with PLN deserialization for complex types
func (h *parsleyHandler) parsePartProps(r *http.Request, env *evaluator.Environment) (*evaluator.Dictionary, error) {
	props := make(map[string]ast.Expression)

	// Parse query params first
	for key, values := range r.URL.Query() {
		if key == "_view" {
			continue // Skip the special _view parameter
		}
		if len(values) > 0 {
			obj := h.parsePartPropValue(values[0], env)
			props[key] = createLiteralFromValue(obj)
		}
	}

	// For POST requests, parse form body and merge
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			return nil, err
		}

		for key, values := range r.PostForm {
			if len(values) > 0 {
				obj := h.parsePartPropValue(values[0], env)
				props[key] = createLiteralFromValue(obj)
			}
		}
	}

	// Build KeyOrder for iteration support
	keyOrder := make([]string, 0, len(props))
	for key := range props {
		keyOrder = append(keyOrder, key)
	}
	sort.Strings(keyOrder) // Deterministic order

	return &evaluator.Dictionary{Pairs: props, KeyOrder: keyOrder, Env: env}, nil
}

// parsePartPropValue parses a single prop value, handling PLN-encoded complex types
func (h *parsleyHandler) parsePartPropValue(value string, env *evaluator.Environment) evaluator.Object {
	// Check if value looks like JSON (starts with { or [)
	if len(value) > 0 && (value[0] == '{' || value[0] == '[') {
		// Try to parse as JSON
		var jsonValue any
		if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
			// Check if it's a PLN marker: {"__pln": "signed_pln_string"}
			if obj, ok := jsonValue.(map[string]any); ok {
				if plnSigned, ok := obj["__pln"].(string); ok {
					// Deserialize PLN prop
					if evaluator.DeserializePLNProp != nil && env.PLNSecret != "" {
						plnObj, err := evaluator.DeserializePLNProp(plnSigned, env.PLNSecret, env)
						if err == nil {
							return plnObj
						}
						// Log error but continue with string fallback
						h.server.logError("PLN deserialization failed: %v", err)
					}
				}
			}
			// Not a PLN marker - convert JSON back to Parsley object
			return jsonToObject(jsonValue)
		}
	}

	// Fall back to standard coercion
	return coerceFormValue(value)
}

// jsonToObject converts a JSON-parsed value to a Parsley object
func jsonToObject(value any) evaluator.Object {
	if value == nil {
		return &evaluator.Null{}
	}
	switch v := value.(type) {
	case bool:
		return &evaluator.Boolean{Value: v}
	case float64:
		// JSON numbers are float64
		if v == float64(int64(v)) {
			return &evaluator.Integer{Value: int64(v)}
		}
		return &evaluator.Float{Value: v}
	case string:
		return &evaluator.String{Value: v}
	case []any:
		elements := make([]evaluator.Object, len(v))
		for i, elem := range v {
			elements[i] = jsonToObject(elem)
		}
		return &evaluator.Array{Elements: elements}
	case map[string]any:
		pairs := make(map[string]ast.Expression)
		keyOrder := make([]string, 0, len(v))
		for key, val := range v {
			pairs[key] = &ast.ObjectLiteralExpression{Obj: jsonToObject(val)}
			keyOrder = append(keyOrder, key)
		}
		sort.Strings(keyOrder)
		return &evaluator.Dictionary{Pairs: pairs, KeyOrder: keyOrder}
	default:
		return &evaluator.String{Value: fmt.Sprintf("%v", value)}
	}
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
	case *evaluator.Null:
		return &ast.ObjectLiteralExpression{Obj: &evaluator.Null{}}
	case *evaluator.Dictionary, *evaluator.Array, *evaluator.Record, *evaluator.Money:
		// Complex types need to be wrapped in ObjectLiteralExpression
		return &ast.ObjectLiteralExpression{Obj: obj}
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
		var result strings.Builder
		for _, elem := range v.Elements {
			result.WriteString(objectToTemplateString(elem))
		}
		return result.String()
	default:
		return obj.Inspect()
	}
}
