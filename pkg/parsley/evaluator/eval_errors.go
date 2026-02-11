// eval_errors.go - Error creation helpers for the Parsley evaluator
//
// This file contains helper functions for creating standardized error objects
// with proper error codes, classes, hints, and structured data.
// All functions return *Error objects that can be used directly in evaluation.

package evaluator

import (
	"fmt"

	perrors "github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// newErrorWithClass creates a simple error with a class (no error code or catalog).
func newErrorWithClass(class ErrorClass, format string, a ...any) *Error {
	return &Error{
		Class:   class,
		Message: fmt.Sprintf(format, a...),
	}
}

// newErrorWithClassAndPos creates an error with class and position information.
func newErrorWithClassAndPos(class ErrorClass, tok lexer.Token, format string, a ...any) *Error {
	return &Error{
		Class:   class,
		Message: fmt.Sprintf(format, a...),
		Line:    tok.Line,
		Column:  tok.Column,
	}
}

// newStructuredError creates a structured error from the catalog.
func newStructuredError(code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newStructuredErrorWithPosAndFile creates a structured error with position and file information.
func newStructuredErrorWithPosAndFile(code string, tok lexer.Token, env *Environment, data map[string]any) *Error {
	perr := perrors.New(code, data)
	file := ""
	if env != nil {
		file = env.Filename
	}
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Line:    tok.Line,
		Column:  tok.Column,
		File:    file,
		Data:    perr.Data,
	}
}

// newSecurityError creates a structured security error from a checkPathAccess error.
// The operation should be "read", "write", or "execute".
// We preserve the original error message for specificity (e.g., "file read restricted: /path")
// but add structured metadata for programmatic handling.
func newSecurityError(operation string, err error) *Error {
	// Map operation to error code
	var code string
	switch operation {
	case "read":
		code = "SEC-0002"
	case "write":
		code = "SEC-0003"
	case "execute":
		code = "SEC-0004"
	default:
		code = "SEC-0001"
	}

	// Get the catalog entry for hints
	perr := perrors.New(code, map[string]any{
		"Operation": operation,
	})

	// Use original error message for specificity, but add structured metadata
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: "security: " + err.Error(), // Preserve original specific message
		Hints:   perr.Hints,
		Data: map[string]any{
			"Operation": operation,
			"GoError":   err.Error(),
		},
	}
}

// newDatabaseError creates a structured database error.
func newDatabaseError(code string, err error) *Error {
	perr := perrors.New(code, map[string]any{
		"GoError": err.Error(),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newDatabaseStateError creates a structured database state error (no Go error).
func newDatabaseStateError(code string) *Error {
	perr := perrors.New(code, nil)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
	}
}

// newDatabaseErrorWithDriver creates a structured database error with driver info.
func newDatabaseErrorWithDriver(code, driver string, err error) *Error {
	perr := perrors.New(code, map[string]any{
		"Driver":  driver,
		"GoError": err.Error(),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newTypeError creates a structured type error for function arguments.
// code should be TYPE-0001 (general), TYPE-0005 (first arg), or TYPE-0006 (second arg).
func newTypeError(code, function, expected string, got ObjectType) *Error {
	perr := perrors.New(code, map[string]any{
		"Function": function,
		"Expected": expected,
		"Got":      perrors.TypeName(string(got)),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newIndexTypeError creates a structured error for unsupported index operations.
func newIndexTypeError(tok lexer.Token, left, index ObjectType) *Error {
	perr := perrors.New("TYPE-0013", map[string]any{
		"Left":  perrors.TypeName(string(left)),
		"Right": perrors.TypeName(string(index)),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
		Line:    tok.Line,
		Column:  tok.Column,
	}
}

// newSliceTypeError creates a structured error for unsupported slice operations.
func newSliceTypeError(left ObjectType) *Error {
	perr := perrors.New("TYPE-0014", map[string]any{
		"Type": perrors.TypeName(string(left)),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newArityError creates a structured error for wrong number of arguments (exact count).
func newArityError(function string, got, want int) *Error {
	perr := perrors.New("ARITY-0001", map[string]any{
		"Function": function,
		"Got":      got,
		"Want":     want,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newArityErrorRange creates a structured error for wrong number of arguments (range).
func newArityErrorRange(function string, got, min, max int) *Error {
	perr := perrors.New("ARITY-0004", map[string]any{
		"Function": function,
		"Got":      got,
		"Min":      min,
		"Max":      max,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newArityErrorExact creates a structured error for exactly X or Y arguments.
func newArityErrorExact(function string, got, choice1, choice2 int) *Error {
	perr := perrors.New("ARITY-0006", map[string]any{
		"Function": function,
		"Got":      got,
		"Choice1":  choice1,
		"Choice2":  choice2,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newArityErrorMin creates a structured error for minimum argument count (variadic).
func newArityErrorMin(function string, got, minArgs int) *Error {
	perr := perrors.New("ARITY-0005", map[string]any{
		"Function": function,
		"Got":      got,
		"Min":      minArgs,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newIOError creates a structured error for I/O operations.
func newIOError(code string, path string, err error) *Error {
	perr := perrors.New(code, map[string]any{
		"Path":    path,
		"GoError": err.Error(),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newFormatError creates a structured error for format/parsing issues.
func newFormatError(code string, err error) *Error {
	perr := perrors.New(code, map[string]any{
		"GoError": err.Error(),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newValueError creates a structured error for invalid values (empty arrays, domain errors, etc.)
func newValueError(code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newUndefinedMethodError creates a structured error for unknown methods.
func newUndefinedMethodError(method string, typeName string) *Error {
	perr := perrors.New("UNDEF-0002", map[string]any{
		"Method": method,
		"Type":   typeName,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newStateError creates a structured error for state-related issues.
func newStateError(code string) *Error {
	perr := perrors.New(code, nil)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newUndefinedComponentError creates a structured error for undefined components.
func newUndefinedComponentError(name string) *Error {
	perr := perrors.New("UNDEF-0003", map[string]any{
		"Name": name,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newUndefinedError creates a structured error for undefined properties/methods.
func newUndefinedError(code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newOperatorError creates a structured error for operator errors.
func newOperatorError(code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newOperatorErrorWithPos creates a structured error for operator errors with position info.
func newOperatorErrorWithPos(tok lexer.Token, code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
		Line:    tok.Line,
		Column:  tok.Column,
	}
}

// newLocaleError creates a structured error for invalid locale.
func newLocaleError(locale string) *Error {
	perr := perrors.New("FMT-0008", map[string]any{
		"Locale": locale,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newFormatErrorWithPos creates a structured format error with position info.
func newFormatErrorWithPos(code string, tok lexer.Token, err error) *Error {
	perr := perrors.New(code, map[string]any{
		"GoError": err.Error(),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
		Line:    tok.Line,
		Column:  tok.Column,
	}
}

// newParseError creates a structured parse error for template syntax issues.
func newParseError(code string, context string, err error) *Error {
	data := map[string]any{
		"Context": context,
	}
	if err != nil {
		data["GoError"] = err.Error()
	}
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newConversionError creates a structured type error for value conversion failures.
func newConversionError(code string, value string) *Error {
	perr := perrors.New(code, map[string]any{
		"Value": value,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newNetworkError creates a structured network error.
func newNetworkError(code string, err error) *Error {
	perr := perrors.New(code, map[string]any{
		"GoError": err.Error(),
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newSliceIndexTypeError creates a structured type error for slice index type issues.
func newSliceIndexTypeError(position string, got string) *Error {
	perr := perrors.New("TYPE-0018", map[string]any{
		"Position": position,
		"Got":      got,
	})
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newIndexError creates a structured index error.
func newIndexError(code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newIndexErrorWithPos creates a structured index error with position info.
func newIndexErrorWithPos(tok lexer.Token, code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
		Line:    tok.Line,
		Column:  tok.Column,
	}
}

// newCommandError creates a structured command/exec error.
func newCommandError(code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newLoopErrorWithPos creates a structured loop/iteration error with position info.
func newLoopErrorWithPos(tok lexer.Token, code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
		Line:    tok.Line,
		Column:  tok.Column,
	}
}

// newImportError creates a structured import/module error.
func newImportError(code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newCallError creates a structured call error.
func newCallError(code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newDestructuringError creates a structured destructuring error.
func newDestructuringError(code string, val Object) *Error {
	data := map[string]any{}
	if val != nil {
		data["Got"] = string(val.Type())
	}
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newHTTPErrorMessage creates an HTTP error with a custom message.
func newHTTPErrorMessage(code string, message string) *Error {
	data := map[string]any{"Error": message}
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newSQLError creates a structured SQL error.
func newSQLError(code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newStdioError creates a structured stdio error.
func newStdioError(code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newSFTPError creates a structured SFTP error.
func newSFTPError(code string, err error) *Error {
	data := map[string]any{}
	if err != nil {
		data["GoError"] = err.Error()
	}
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newFileOpError creates a structured file operator error.
func newFileOpError(code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newValidationError creates a structured validation error.
func newValidationError(code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}

// newInternalError creates a structured internal/context error.
func newInternalError(code string, data map[string]any) *Error {
	perr := perrors.New(code, data)
	return &Error{
		Class:   ErrorClass(perr.Class),
		Code:    perr.Code,
		Message: perr.Message,
		Hints:   perr.Hints,
		Data:    perr.Data,
	}
}
