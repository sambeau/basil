package evaluator

import (
	"fmt"
	"maps"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// loadAPIModule returns the api module as a StdlibModuleDict
func loadAPIModule(env *Environment) Object {
	return &StdlibModuleDict{
		Exports: map[string]Object{
			// Auth wrappers - use StdlibBuiltin so they can be called
			"public":    &StdlibBuiltin{Fn: apiPublic},
			"adminOnly": &StdlibBuiltin{Fn: apiAdminOnly},
			"roles":     &StdlibBuiltin{Fn: apiRoles},
			"auth":      &StdlibBuiltin{Fn: apiAuth},

			// Error helpers - Builtin is fine for these (no function args)
			"notFound":     &Builtin{Fn: apiNotFound},
			"forbidden":    &Builtin{Fn: apiForbidden},
			"badRequest":   &Builtin{Fn: apiBadRequest},
			"unauthorized": &Builtin{Fn: apiUnauthorized},
			"conflict":     &Builtin{Fn: apiConflict},
			"serverError":  &Builtin{Fn: apiServerError},

			// Redirect helper
			"redirect": &Builtin{Fn: apiRedirect},
		},
	}
}

// =============================================================================
// Auth Wrappers
// =============================================================================

// apiPublic wraps a function to mark it as publicly accessible (no auth)
func apiPublic(args []Object, env *Environment) Object {
	// public(fn) or public({options}, fn)
	var fn Object
	var options *Dictionary

	if len(args) == 1 {
		fn = args[0]
	} else if len(args) == 2 {
		if opts, ok := args[0].(*Dictionary); ok {
			options = opts
			fn = args[1]
		} else {
			return newTypeError("TYPE-0001", "public", "dictionary", args[0].Type())
		}
	} else {
		return newArityError("public", len(args), 1)
	}

	// Ensure it's a function
	if !isCallable(fn) {
		return newTypeError("TYPE-0001", "public", "function", fn.Type())
	}

	return &AuthWrappedFunction{
		Inner:    fn,
		AuthType: "public",
		Options:  options,
	}
}

// apiAdminOnly wraps a function to require admin role
func apiAdminOnly(args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("adminOnly", len(args), 1)
	}

	fn := args[0]

	// Ensure it's a function
	if !isCallable(fn) {
		return newTypeError("TYPE-0001", "adminOnly", "function", fn.Type())
	}

	return &AuthWrappedFunction{
		Inner:    fn,
		AuthType: "admin",
		Roles:    []string{"admin"},
	}
}

// apiRoles wraps a function to require specific roles
func apiRoles(args []Object, env *Environment) Object {
	if len(args) != 2 {
		return newArityError("roles", len(args), 2)
	}

	rolesArr, ok := args[0].(*Array)
	if !ok {
		return newTypeError("TYPE-0001", "roles", "array", args[0].Type())
	}

	fn := args[1]

	// Ensure it's a function
	if !isCallable(fn) {
		return newTypeError("TYPE-0001", "roles", "function", fn.Type())
	}

	// Extract role strings
	var roles []string
	for _, elem := range rolesArr.Elements {
		if str, ok := elem.(*String); ok {
			roles = append(roles, str.Value)
		}
	}

	return &AuthWrappedFunction{
		Inner:    fn,
		AuthType: "roles",
		Roles:    roles,
	}
}

// apiAuth wraps a function with custom auth options
func apiAuth(args []Object, env *Environment) Object {
	// auth(fn) with optional first argument for options
	var fn Object
	var options *Dictionary

	if len(args) == 1 {
		fn = args[0]
	} else if len(args) == 2 {
		if opts, ok := args[0].(*Dictionary); ok {
			options = opts
			fn = args[1]
		} else {
			return newTypeError("TYPE-0001", "auth", "dictionary", args[0].Type())
		}
	} else {
		return newArityError("auth", len(args), 1)
	}

	// Ensure it's a function
	if !isCallable(fn) {
		return newTypeError("TYPE-0001", "auth", "function", fn.Type())
	}

	return &AuthWrappedFunction{
		Inner:    fn,
		AuthType: "auth",
		Options:  options,
	}
}

// isCallable checks if an object can be called as a function
func isCallable(obj Object) bool {
	switch obj.(type) {
	case *Function, *Builtin, *StdlibBuiltin, *AuthWrappedFunction:
		return true
	default:
		return false
	}
}

// AuthWrappedFunction represents a function with auth metadata
type AuthWrappedFunction struct {
	Inner    Object      // The wrapped function
	AuthType string      // "public", "admin", "roles", "auth"
	Roles    []string    // Required roles (for "roles" and "admin")
	Options  *Dictionary // Additional options
}

func (f *AuthWrappedFunction) Type() ObjectType { return FUNCTION_OBJ }
func (f *AuthWrappedFunction) Inspect() string {
	return f.AuthType + "(" + f.Inner.Inspect() + ")"
}

// GetAuthMetadata returns the auth metadata as a dictionary
func (f *AuthWrappedFunction) GetAuthMetadata() *Dictionary {
	pairs := make(map[string]ast.Expression)
	pairs["__auth__"] = objectToExpression(&String{Value: f.AuthType})

	if len(f.Roles) > 0 {
		roleObjs := make([]Object, len(f.Roles))
		for i, r := range f.Roles {
			roleObjs[i] = &String{Value: r}
		}
		pairs["roles"] = objectToExpression(&Array{Elements: roleObjs})
	}

	if f.Options != nil {
		maps.Copy(pairs, f.Options.Pairs)
	}

	return &Dictionary{Pairs: pairs}
}

// =============================================================================
// Error Helpers
// =============================================================================

// apiFailError builds a unified *Error with a UserDict containing code, message, and status.
func apiFailError(code string, message string, status int) *Error {
	pairs := make(map[string]ast.Expression)
	pairs["code"] = objectToExpression(&String{Value: code})
	pairs["message"] = objectToExpression(&String{Value: message})
	pairs["status"] = objectToExpression(&Integer{Value: int64(status)})
	dict := &Dictionary{
		Pairs:    pairs,
		KeyOrder: []string{"code", "message", "status"},
	}
	return &Error{
		Class:    ClassValue,
		Code:     code,
		Message:  message,
		UserDict: dict,
	}
}

// ApiFailError is the exported version of apiFailError for use by the server package.
func ApiFailError(code string, message string, status int) *Error {
	return apiFailError(code, message, status)
}

// apiNotFound creates a 404 Not Found error
func apiNotFound(args ...Object) Object {
	msg := "Not found"
	if len(args) == 1 {
		if str, ok := args[0].(*String); ok {
			msg = str.Value
		}
	}
	return apiFailError("HTTP-404", msg, 404)
}

// apiForbidden creates a 403 Forbidden error
func apiForbidden(args ...Object) Object {
	msg := "Forbidden"
	if len(args) == 1 {
		if str, ok := args[0].(*String); ok {
			msg = str.Value
		}
	}
	return apiFailError("HTTP-403", msg, 403)
}

// apiBadRequest creates a 400 Bad Request error
func apiBadRequest(args ...Object) Object {
	msg := "Bad request"
	if len(args) == 1 {
		if str, ok := args[0].(*String); ok {
			msg = str.Value
		}
	}
	return apiFailError("HTTP-400", msg, 400)
}

// apiUnauthorized creates a 401 Unauthorized error
func apiUnauthorized(args ...Object) Object {
	msg := "Unauthorized"
	if len(args) == 1 {
		if str, ok := args[0].(*String); ok {
			msg = str.Value
		}
	}
	return apiFailError("HTTP-401", msg, 401)
}

// apiConflict creates a 409 Conflict error
func apiConflict(args ...Object) Object {
	msg := "Conflict"
	if len(args) == 1 {
		if str, ok := args[0].(*String); ok {
			msg = str.Value
		}
	}
	return apiFailError("HTTP-409", msg, 409)
}

// apiServerError creates a 500 Internal Server Error
func apiServerError(args ...Object) Object {
	msg := "Internal server error"
	if len(args) == 1 {
		if str, ok := args[0].(*String); ok {
			msg = str.Value
		}
	}
	return apiFailError("HTTP-500", msg, 500)
}

// =============================================================================
// Redirect Helper
// =============================================================================

// Redirect represents an HTTP redirect response
type Redirect struct {
	URL    string
	Status int
}

func (r *Redirect) Type() ObjectType { return REDIRECT_OBJ }
func (r *Redirect) Inspect() string  { return fmt.Sprintf("redirect(%s, %d)", r.URL, r.Status) }

// apiRedirect creates an HTTP redirect response
// redirect(url) - 302 Found (default)
// redirect(url, status) - custom status (must be 3xx)
func apiRedirect(args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return newArityErrorRange("redirect", len(args), 1, 2)
	}

	// Extract URL from first argument
	var url string
	switch u := args[0].(type) {
	case *String:
		url = u.Value
	case *Dictionary:
		// Check if it's a path object (has __type: "path")
		if typeExpr, ok := u.Pairs["__type"]; ok {
			if strLit, ok := typeExpr.(*ast.StringLiteral); ok && strLit.Value == "path" {
				// Convert path to string
				url = pathDictToString(u)
			}
		}
		if url == "" {
			return newTypeError("TYPE-0001", "redirect", "string or path", u.Type())
		}
	default:
		return newTypeError("TYPE-0001", "redirect", "string or path", args[0].Type())
	}

	// Validate URL is not empty
	if url == "" {
		return &Error{
			Message: "redirect URL cannot be empty",
			Class:   ClassValue,
			Code:    "VALUE-0001",
			Hints:   []string{"provide a valid URL or path"},
		}
	}

	// Default status is 302 Found
	status := 302

	// Check for optional status code
	if len(args) == 2 {
		switch s := args[1].(type) {
		case *Integer:
			status = int(s.Value)
		default:
			return newTypeError("TYPE-0001", "redirect", "integer", args[1].Type())
		}

		// Validate status is a 3xx redirect code
		if status < 300 || status > 399 {
			return &Error{
				Message: fmt.Sprintf("redirect status must be 3xx, got %d", status),
				Class:   ClassValue,
				Code:    "VALUE-0002",
				Hints:   []string{"use 301 (permanent), 302 (found), 303 (see other), 307 (temporary), or 308 (permanent)"},
			}
		}
	}

	return &Redirect{URL: url, Status: status}
}
