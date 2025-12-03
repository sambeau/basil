package evaluator

import (
	"fmt"
	"strings"
)

// DevLogWriter is an interface for writing dev logs.
// This is implemented by server.DevLog but defined here to avoid import cycles.
type DevLogWriter interface {
	LogFromEvaluator(route, level, filename string, line int, callRepr, valueRepr string) error
	ClearLogs(route string) error
}

// DevModule provides dev tools functions (dev.log, dev.clearLog, etc.)
// It's callable as a namespace and has methods.
// Note: DevModule reads DevLog from the environment at call time, not at creation time.
// This allows modules imported at top-level to still log when called from handlers.
type DevModule struct {
	defaultRoute string
}

// NewDevModule creates a new DevModule.
// The actual DevLogWriter is read from env.DevLog at call time.
func NewDevModule() *DevModule {
	return &DevModule{
		defaultRoute: "",
	}
}

func (dm *DevModule) Type() ObjectType { return BUILTIN_OBJ }
func (dm *DevModule) Inspect() string  { return "dev" }

// loadDevModule returns the dev module for stdlib import
// The DevLogWriter is read from env.DevLog at call time (not import time).
// This allows modules to be imported at top-level and still log when called from handlers.
func loadDevModule(env *Environment) Object {
	devModule := NewDevModule()
	return &StdlibModuleDict{
		Exports: map[string]Object{
			"dev": devModule,
		},
	}
}

// evalDevModuleMethod handles method calls on the dev module
func evalDevModuleMethod(dm *DevModule, method string, args []Object, env *Environment) Object {
	switch method {
	case "log":
		return dm.evalLog(args, env)
	case "clearLog":
		return dm.evalClearLog(args, env)
	case "logPage":
		return dm.evalLogPage(args, env)
	case "setLogRoute":
		return dm.evalSetLogRoute(args, env)
	case "clearLogPage":
		return dm.evalClearLogPage(args, env)
	default:
		return newError("unknown method '%s' on dev module", method)
	}
}

// evalLog implements dev.log(value) and dev.log(label, value)
func (dm *DevModule) evalLog(args []Object, env *Environment) Object {
	// No-op in production mode (read from env at call time, not import time)
	if env.DevLog == nil {
		return NULL
	}

	if len(args) < 1 || len(args) > 3 {
		return newError("dev.log expects 1-3 arguments: dev.log(value) or dev.log(label, value) or dev.log(value, {level: \"warn\"})")
	}

	var label string
	var value Object
	var level string = "info"

	if len(args) == 1 {
		// dev.log(value)
		value = args[0]
	} else if len(args) == 2 {
		// Could be dev.log(label, value) or dev.log(value, options)
		if isOptionsDict(args[1]) {
			// dev.log(value, {level: "warn"})
			value = args[0]
			level = extractLevel(args[1], env)
		} else {
			// dev.log(label, value)
			label = devObjectToString(args[0])
			value = args[1]
		}
	} else if len(args) == 3 {
		// dev.log(label, value, options)
		label = devObjectToString(args[0])
		value = args[1]
		level = extractLevel(args[2], env)
	}

	// Build call representation
	callRepr := buildCallRepr("dev.log", label, value)

	// Get file/line info from environment
	filename := env.Filename
	line := 0
	if env.LastToken != nil {
		line = env.LastToken.Line
	}

	if err := env.DevLog.LogFromEvaluator(dm.defaultRoute, level, filename, line, callRepr, value.Inspect()); err != nil {
		// Don't fail the script, just log to stderr
		fmt.Printf("[WARN] dev.log failed: %v\n", err)
	}

	return NULL
}

// evalClearLog implements dev.clearLog()
func (dm *DevModule) evalClearLog(args []Object, env *Environment) Object {
	// No-op in production mode
	if env.DevLog == nil {
		return NULL
	}

	if len(args) != 0 {
		return newError("dev.clearLog expects 0 arguments")
	}

	if err := env.DevLog.ClearLogs(dm.defaultRoute); err != nil {
		return newError("dev.clearLog failed: %v", err)
	}

	return NULL
}

// evalLogPage implements dev.logPage(route, value) and dev.logPage(route, label, value)
func (dm *DevModule) evalLogPage(args []Object, env *Environment) Object {
	// No-op in production mode
	if env.DevLog == nil {
		return NULL
	}

	if len(args) < 2 || len(args) > 4 {
		return newError("dev.logPage expects 2-4 arguments: dev.logPage(route, value) or dev.logPage(route, label, value)")
	}

	route := devObjectToString(args[0])
	if !isValidRoute(route) {
		return newError("dev.logPage: invalid route '%s' (use alphanumeric, hyphens, underscores)", route)
	}

	var label string
	var value Object
	var level string = "info"

	if len(args) == 2 {
		// dev.logPage(route, value)
		value = args[1]
	} else if len(args) == 3 {
		// Could be dev.logPage(route, label, value) or dev.logPage(route, value, options)
		if isOptionsDict(args[2]) {
			// dev.logPage(route, value, {level: "warn"})
			value = args[1]
			level = extractLevel(args[2], env)
		} else {
			// dev.logPage(route, label, value)
			label = devObjectToString(args[1])
			value = args[2]
		}
	} else if len(args) == 4 {
		// dev.logPage(route, label, value, options)
		label = devObjectToString(args[1])
		value = args[2]
		level = extractLevel(args[3], env)
	}

	// Build call representation
	callRepr := buildCallRepr("dev.logPage", label, value)

	// Get file/line info from environment
	filename := env.Filename
	line := 0
	if env.LastToken != nil {
		line = env.LastToken.Line
	}

	if err := env.DevLog.LogFromEvaluator(route, level, filename, line, callRepr, value.Inspect()); err != nil {
		fmt.Printf("[WARN] dev.logPage failed: %v\n", err)
	}

	return NULL
}

// evalSetLogRoute implements dev.setLogRoute(route)
func (dm *DevModule) evalSetLogRoute(args []Object, env *Environment) Object {
	// No-op in production mode
	if env.DevLog == nil {
		return NULL
	}

	if len(args) != 1 {
		return newError("dev.setLogRoute expects 1 argument: dev.setLogRoute(route)")
	}

	route := devObjectToString(args[0])
	if route != "" && !isValidRoute(route) {
		return newError("dev.setLogRoute: invalid route '%s' (use alphanumeric, hyphens, underscores)", route)
	}

	dm.defaultRoute = route
	return NULL
}

// evalClearLogPage implements dev.clearLogPage(route)
func (dm *DevModule) evalClearLogPage(args []Object, env *Environment) Object {
	// No-op in production mode
	if env.DevLog == nil {
		return NULL
	}

	if len(args) != 1 {
		return newError("dev.clearLogPage expects 1 argument: dev.clearLogPage(route)")
	}

	route := devObjectToString(args[0])
	if !isValidRoute(route) {
		return newError("dev.clearLogPage: invalid route '%s' (use alphanumeric, hyphens, underscores)", route)
	}

	if err := env.DevLog.ClearLogs(route); err != nil {
		return newError("dev.clearLogPage failed: %v", err)
	}

	return NULL
}

// Helper functions

// devObjectToString converts an object to string, prioritizing String values
func devObjectToString(obj Object) string {
	if obj == nil {
		return ""
	}
	switch o := obj.(type) {
	case *String:
		return o.Value
	default:
		return obj.Inspect()
	}
}

func isOptionsDict(obj Object) bool {
	dict, ok := obj.(*Dictionary)
	if !ok {
		return false
	}
	// Check if it looks like an options dict (has "level" key)
	for key := range dict.Pairs {
		if key == "level" {
			return true
		}
	}
	return false
}

func extractLevel(obj Object, env *Environment) string {
	dict, ok := obj.(*Dictionary)
	if !ok {
		return "info"
	}

	levelExpr, exists := dict.Pairs["level"]
	if !exists {
		return "info"
	}

	// Evaluate the expression to get the value
	// Use the dictionary's environment for evaluation
	evalEnv := dict.Env
	if evalEnv == nil {
		evalEnv = env
	}
	levelVal := Eval(levelExpr, evalEnv)
	if str, ok := levelVal.(*String); ok {
		level := strings.ToLower(str.Value)
		if level == "warn" || level == "warning" {
			return "warn"
		}
	}

	return "info"
}

func buildCallRepr(fn string, label string, value Object) string {
	if label != "" {
		return fmt.Sprintf("%s(\"%s\", %s)", fn, label, truncateRepr(value.Inspect(), 50))
	}
	return fmt.Sprintf("%s(%s)", fn, truncateRepr(value.Inspect(), 50))
}

func truncateRepr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func isValidRoute(route string) bool {
	if route == "" {
		return true // Empty route is valid (means default)
	}
	for _, c := range route {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
