package evaluator

import (
	"sort"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/format"
)

// Format accessor methods for integration with the format package.
// These methods provide a clean interface for the formatter to access
// object internals without exposing implementation details.

// GetElements returns array elements as typed objects for the formatter.
// This satisfies format.ArrayAccessor.
func (a *Array) GetElements() []format.TypedObject {
	result := make([]format.TypedObject, len(a.Elements))
	for i, elem := range a.Elements {
		result[i] = wrapObject(elem)
	}
	return result
}

// objectWrapper adapts an evaluator Object to format.TypedObject
type objectWrapper struct {
	obj Object
}

func (w *objectWrapper) Type() string {
	if w.obj == nil {
		return "NULL"
	}
	return string(w.obj.Type())
}

func (w *objectWrapper) Inspect() string {
	if w.obj == nil {
		return "null"
	}
	return w.obj.Inspect()
}

// wrapObject wraps an evaluator Object as format.TypedObject
func wrapObject(obj Object) format.TypedObject {
	if obj == nil {
		return &objectWrapper{nil}
	}
	// If it's an array, wrap it so GetElements can be called
	if arr, ok := obj.(*Array); ok {
		return &arrayWrapper{arr}
	}
	// If it's a dictionary, wrap it so accessor methods work
	if dict, ok := obj.(*Dictionary); ok {
		return &dictWrapper{dict}
	}
	// If it's a function, wrap it so accessor methods work
	if fn, ok := obj.(*Function); ok {
		return &functionWrapper{fn}
	}
	// If it's a record, wrap it so accessor methods work
	if rec, ok := obj.(*Record); ok {
		return &recordWrapper{rec}
	}
	// Default wrapper for simple types
	return &objectWrapper{obj}
}

// arrayWrapper adapts Array to format.ArrayAccessor
type arrayWrapper struct {
	arr *Array
}

func (w *arrayWrapper) Type() string    { return string(w.arr.Type()) }
func (w *arrayWrapper) Inspect() string { return w.arr.Inspect() }
func (w *arrayWrapper) GetElements() []format.TypedObject {
	return w.arr.GetElements()
}

// dictWrapper adapts Dictionary to format.DictionaryAccessor
type dictWrapper struct {
	dict *Dictionary
}

func (w *dictWrapper) Type() string    { return string(w.dict.Type()) }
func (w *dictWrapper) Inspect() string { return w.dict.Inspect() }
func (w *dictWrapper) GetKeys() []string {
	return w.dict.GetKeys()
}
func (w *dictWrapper) GetValueObject(key string) format.TypedObject {
	return w.dict.GetValueObject(key)
}

// functionWrapper adapts Function to format.FunctionAccessor
type functionWrapper struct {
	fn *Function
}

func (w *functionWrapper) Type() string              { return string(w.fn.Type()) }
func (w *functionWrapper) Inspect() string           { return w.fn.Inspect() }
func (w *functionWrapper) GetParamStrings() []string { return w.fn.GetParamStrings() }
func (w *functionWrapper) GetBodyString() string     { return w.fn.GetBodyString() }

// recordWrapper adapts Record to format.RecordAccessor
type recordWrapper struct {
	rec *Record
}

func (w *recordWrapper) Type() string           { return string(w.rec.Type()) }
func (w *recordWrapper) Inspect() string        { return w.rec.Inspect() }
func (w *recordWrapper) GetSchemaName() string  { return w.rec.GetSchemaName() }
func (w *recordWrapper) GetFieldKeys() []string { return w.rec.GetFieldKeys() }
func (w *recordWrapper) GetFieldObject(key string) format.TypedObject {
	return w.rec.GetFieldObject(key)
}

// GetKeys returns dictionary keys in order.
// This satisfies format.DictionaryAccessor.
func (d *Dictionary) GetKeys() []string {
	return d.Keys()
}

// GetValueObject returns the value for a key as a format.TypedObject.
// This satisfies format.DictionaryAccessor.
func (d *Dictionary) GetValueObject(key string) format.TypedObject {
	expr, ok := d.Pairs[key]
	if !ok {
		return nil
	}
	// Wrap the expression value for the formatter
	return wrapExpr(expr)
}

// wrapExpr wraps an expression as a format.TypedObject for display
func wrapExpr(expr ast.Expression) format.TypedObject {
	if expr == nil {
		return &objectWrapper{nil}
	}
	// Handle object literals specially - these contain evaluated objects
	if objLit, ok := expr.(*ast.ObjectLiteralExpression); ok {
		if objLit.Obj != nil {
			if obj, ok := objLit.Obj.(Object); ok {
				return wrapObject(obj)
			}
		}
		return &objectWrapper{nil}
	}
	// For other expressions, create a string wrapper with the expr representation
	return &exprWrapper{expr}
}

// exprWrapper wraps an AST expression for display
type exprWrapper struct {
	expr ast.Expression
}

func (w *exprWrapper) Type() string    { return "EXPRESSION" }
func (w *exprWrapper) Inspect() string { return w.expr.String() }

// GetParamStrings returns function parameter names as strings.
// This satisfies format.FunctionAccessor.
func (f *Function) GetParamStrings() []string {
	params := make([]string, len(f.Params))
	for i, p := range f.Params {
		params[i] = p.String()
	}
	return params
}

// GetBodyString returns the function body as a string.
// This satisfies format.FunctionAccessor.
func (f *Function) GetBodyString() string {
	if f.Body == nil {
		return ""
	}
	return f.Body.String()
}

// GetSchemaName returns the record's schema name.
// This satisfies format.RecordAccessor.
func (r *Record) GetSchemaName() string {
	if r.Schema != nil {
		return r.Schema.Name
	}
	return "?"
}

// GetFieldKeys returns the record's field keys in order.
// This satisfies format.RecordAccessor.
func (r *Record) GetFieldKeys() []string {
	keys := r.KeyOrder
	if len(keys) == 0 && len(r.Data) > 0 {
		keys = make([]string, 0, len(r.Data))
		for key := range r.Data {
			keys = append(keys, key)
		}
		sort.Strings(keys)
	}
	return keys
}

// GetFieldObject returns the value for a field as a format.TypedObject.
// This satisfies format.RecordAccessor.
func (r *Record) GetFieldObject(key string) format.TypedObject {
	expr, ok := r.Data[key]
	if !ok {
		return nil
	}
	return wrapExpr(expr)
}

// FormatObject formats an evaluator Object using the pretty-printer.
// This is the primary entry point for REPL and documentation output.
func FormatObject(obj Object) string {
	if obj == nil {
		return "null"
	}
	wrapped := wrapObject(obj)
	return format.FormatObject(wrapped)
}
