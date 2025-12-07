package evaluator

import (
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
		for k, v := range f.Options.Pairs {
			pairs[k] = v
		}
	}

	return &Dictionary{Pairs: pairs}
}

// =============================================================================
// Error Helpers
// =============================================================================

// apiNotFound creates a 404 Not Found error
func apiNotFound(args ...Object) Object {
	msg := "Not found"
	if len(args) == 1 {
		if str, ok := args[0].(*String); ok {
			msg = str.Value
		}
	}
	return &APIError{Code: "HTTP-404", Message: msg, Status: 404}
}

// apiForbidden creates a 403 Forbidden error
func apiForbidden(args ...Object) Object {
	msg := "Forbidden"
	if len(args) == 1 {
		if str, ok := args[0].(*String); ok {
			msg = str.Value
		}
	}
	return &APIError{Code: "HTTP-403", Message: msg, Status: 403}
}

// apiBadRequest creates a 400 Bad Request error
func apiBadRequest(args ...Object) Object {
	msg := "Bad request"
	if len(args) == 1 {
		if str, ok := args[0].(*String); ok {
			msg = str.Value
		}
	}
	return &APIError{Code: "HTTP-400", Message: msg, Status: 400}
}

// apiUnauthorized creates a 401 Unauthorized error
func apiUnauthorized(args ...Object) Object {
	msg := "Unauthorized"
	if len(args) == 1 {
		if str, ok := args[0].(*String); ok {
			msg = str.Value
		}
	}
	return &APIError{Code: "HTTP-401", Message: msg, Status: 401}
}

// apiConflict creates a 409 Conflict error
func apiConflict(args ...Object) Object {
	msg := "Conflict"
	if len(args) == 1 {
		if str, ok := args[0].(*String); ok {
			msg = str.Value
		}
	}
	return &APIError{Code: "HTTP-409", Message: msg, Status: 409}
}

// apiServerError creates a 500 Internal Server Error
func apiServerError(args ...Object) Object {
	msg := "Internal server error"
	if len(args) == 1 {
		if str, ok := args[0].(*String); ok {
			msg = str.Value
		}
	}
	return &APIError{Code: "HTTP-500", Message: msg, Status: 500}
}

// APIError represents an API error with HTTP status
type APIError struct {
	Code    string
	Message string
	Status  int
	Field   string // Optional field name for validation errors
}

func (e *APIError) Type() ObjectType { return API_ERROR_OBJ }
func (e *APIError) Inspect() string  { return e.Code + ": " + e.Message }

// ToDict converts the error to a dictionary for JSON response
func (e *APIError) ToDict() *Dictionary {
	pairs := make(map[string]ast.Expression)
	errorPairs := make(map[string]ast.Expression)

	errorPairs["code"] = objectToExpression(&String{Value: e.Code})
	errorPairs["message"] = objectToExpression(&String{Value: e.Message})
	if e.Field != "" {
		errorPairs["field"] = objectToExpression(&String{Value: e.Field})
	}

	pairs["error"] = objectToExpression(&Dictionary{Pairs: errorPairs})

	return &Dictionary{Pairs: pairs}
}
