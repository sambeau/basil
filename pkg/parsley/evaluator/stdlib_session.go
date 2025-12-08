package evaluator

import (
	"fmt"
	"time"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// SESSION_MODULE_OBJ is the type identifier for session module
const SESSION_MODULE_OBJ = "SESSION_MODULE"

// SessionModule wraps session data and provides methods for Parsley scripts
type SessionModule struct {
	Data    map[string]interface{}
	Flash   map[string]string
	Dirty   bool
	Cleared bool
	MaxAge  time.Duration
}

func (sm *SessionModule) Type() ObjectType { return SESSION_MODULE_OBJ }
func (sm *SessionModule) Inspect() string  { return "<session>" }
func (sm *SessionModule) Eq(other Object) bool {
	if o, ok := other.(*SessionModule); ok {
		return sm == o
	}
	return false
}

// NewSessionModule creates a new session module with the given data
func NewSessionModule(data map[string]interface{}, flash map[string]string, maxAge time.Duration) *SessionModule {
	if data == nil {
		data = make(map[string]interface{})
	}
	if flash == nil {
		flash = make(map[string]string)
	}
	return &SessionModule{
		Data:   data,
		Flash:  flash,
		MaxAge: maxAge,
	}
}

// evalSessionMethod handles method calls on the session module
func evalSessionMethod(sm *SessionModule, method string, args []Object, env *Environment) Object {
	switch method {
	case "get":
		return sessionGet(sm, args)
	case "set":
		return sessionSet(sm, args)
	case "delete":
		return sessionDelete(sm, args)
	case "has":
		return sessionHas(sm, args)
	case "clear":
		return sessionClear(sm)
	case "all":
		return sessionAll(sm)
	case "flash":
		return sessionFlash(sm, args)
	case "getFlash":
		return sessionGetFlash(sm, args)
	case "getAllFlash":
		return sessionGetAllFlash(sm)
	case "hasFlash":
		return sessionHasFlash(sm)
	case "regenerate":
		return sessionRegenerate(sm)
	default:
		return unknownMethodError("session", method, []string{
			"get", "set", "delete", "has", "clear", "all",
			"flash", "getFlash", "getAllFlash", "hasFlash", "regenerate",
		})
	}
}

// sessionGet retrieves a value from the session
// Usage: basil.session.get("key") or basil.session.get("key", defaultValue)
func sessionGet(sm *SessionModule, args []Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return newArityErrorRange("session.get", len(args), 1, 2)
	}

	key, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "session.get", "string", args[0].Type())
	}

	value, exists := sm.Data[key.Value]
	if !exists {
		if len(args) == 2 {
			return args[1] // Return default value
		}
		return NULL
	}

	// Convert Go value back to Parsley object
	return sessionGoToObject(value)
}

// sessionSet stores a value in the session
// Usage: basil.session.set("key", value)
func sessionSet(sm *SessionModule, args []Object) Object {
	if len(args) != 2 {
		return newArityError("session.set", len(args), 2)
	}

	key, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "session.set", "string", args[0].Type())
	}

	// Convert Parsley object to Go value for storage
	value := objectToGo(args[1])
	sm.Data[key.Value] = value
	sm.Dirty = true

	return NULL
}

// sessionDelete removes a value from the session
// Usage: basil.session.delete("key")
func sessionDelete(sm *SessionModule, args []Object) Object {
	if len(args) != 1 {
		return newArityError("session.delete", len(args), 1)
	}

	key, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "session.delete", "string", args[0].Type())
	}

	delete(sm.Data, key.Value)
	sm.Dirty = true

	return NULL
}

// sessionHas checks if a key exists in the session
// Usage: basil.session.has("key")
func sessionHas(sm *SessionModule, args []Object) Object {
	if len(args) != 1 {
		return newArityError("session.has", len(args), 1)
	}

	key, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "session.has", "string", args[0].Type())
	}

	_, exists := sm.Data[key.Value]
	return nativeBoolToParsBoolean(exists)
}

// sessionClear removes all session data
// Usage: basil.session.clear()
func sessionClear(sm *SessionModule) Object {
	sm.Data = make(map[string]interface{})
	sm.Flash = make(map[string]string)
	sm.Dirty = true
	sm.Cleared = true
	return NULL
}

// sessionAll returns all session data as a dictionary
// Usage: basil.session.all()
func sessionAll(sm *SessionModule) Object {
	// Convert to Parsley dictionary
	pairs := make(map[string]ast.Expression)
	for k, v := range sm.Data {
		obj := sessionGoToObject(v)
		pairs[k] = &ast.ObjectLiteralExpression{Obj: obj}
	}
	return &Dictionary{Pairs: pairs}
}

// sessionFlash sets a flash message (one-time message)
// Usage: basil.session.flash("success", "Operation completed!")
func sessionFlash(sm *SessionModule, args []Object) Object {
	if len(args) != 2 {
		return newArityError("session.flash", len(args), 2)
	}

	key, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "session.flash", "string", args[0].Type())
	}

	msg, ok := args[1].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "session.flash", "string", args[1].Type())
	}

	sm.Flash[key.Value] = msg.Value
	sm.Dirty = true

	return NULL
}

// sessionGetFlash retrieves and removes a flash message
// Usage: basil.session.getFlash("success")
func sessionGetFlash(sm *SessionModule, args []Object) Object {
	if len(args) != 1 {
		return newArityError("session.getFlash", len(args), 1)
	}

	key, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "session.getFlash", "string", args[0].Type())
	}

	msg, exists := sm.Flash[key.Value]
	if !exists {
		return NULL
	}

	// Remove after reading (one-time message)
	delete(sm.Flash, key.Value)
	sm.Dirty = true

	return &String{Value: msg}
}

// sessionGetAllFlash retrieves and removes all flash messages
// Usage: basil.session.getAllFlash()
func sessionGetAllFlash(sm *SessionModule) Object {
	if len(sm.Flash) == 0 {
		return &Dictionary{Pairs: make(map[string]ast.Expression)}
	}

	// Convert to Parsley dictionary
	pairs := make(map[string]ast.Expression)
	for k, v := range sm.Flash {
		pairs[k] = &ast.ObjectLiteralExpression{Obj: &String{Value: v}}
	}

	// Clear flash messages
	sm.Flash = make(map[string]string)
	sm.Dirty = true

	return &Dictionary{Pairs: pairs}
}

// sessionHasFlash checks if any flash messages exist
// Usage: basil.session.hasFlash()
func sessionHasFlash(sm *SessionModule) Object {
	return nativeBoolToParsBoolean(len(sm.Flash) > 0)
}

// sessionRegenerate creates a new session while preserving data
// For cookie sessions, this just marks the session as dirty to get a new timestamp
// Usage: basil.session.regenerate()
func sessionRegenerate(sm *SessionModule) Object {
	sm.Dirty = true
	return NULL
}

// sessionGoToObject converts a Go value to a Parsley Object
// This is a local helper to avoid import cycles with parsley package
func sessionGoToObject(v interface{}) Object {
	if v == nil {
		return NULL
	}
	switch val := v.(type) {
	case bool:
		return nativeBoolToParsBoolean(val)
	case int:
		return &Integer{Value: int64(val)}
	case int64:
		return &Integer{Value: val}
	case float64:
		return &Float{Value: val}
	case string:
		return &String{Value: val}
	case []interface{}:
		elements := make([]Object, len(val))
		for i, elem := range val {
			elements[i] = sessionGoToObject(elem)
		}
		return &Array{Elements: elements}
	case map[string]interface{}:
		pairs := make(map[string]ast.Expression)
		for k, elem := range val {
			obj := sessionGoToObject(elem)
			pairs[k] = &ast.ObjectLiteralExpression{Obj: obj}
		}
		return &Dictionary{Pairs: pairs}
	default:
		return &String{Value: fmt.Sprintf("%v", val)}
	}
}
