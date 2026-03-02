// Package evaluator provides session and dev module method implementations via declarative registry.
// This file migrates session and dev module methods from switch-based dispatch (FEAT-137 Tier 3).
package evaluator

// SessionMethodRegistry defines all methods available on session objects.
var SessionMethodRegistry MethodRegistry

// DevModuleMethodRegistry defines all methods available on the dev module.
var DevModuleMethodRegistry MethodRegistry

func init() {
	SessionMethodRegistry = MethodRegistry{
		"get": {
			Fn:          sessionMethodGet,
			Arity:       "1-2",
			Description: "Get session value (key, default?)",
		},
		"set": {
			Fn:          sessionMethodSet,
			Arity:       "2",
			Description: "Set session value (key, value)",
		},
		"delete": {
			Fn:          sessionMethodDelete,
			Arity:       "1",
			Description: "Delete session key",
		},
		"has": {
			Fn:          sessionMethodHas,
			Arity:       "1",
			Description: "Check if key exists",
		},
		"clear": {
			Fn:          sessionMethodClear,
			Arity:       "0",
			Description: "Clear all session data",
		},
		"all": {
			Fn:          sessionMethodAll,
			Arity:       "0",
			Description: "Get all session data as dictionary",
		},
		"flash": {
			Fn:          sessionMethodFlash,
			Arity:       "2",
			Description: "Set flash message (key, value)",
		},
		"getFlash": {
			Fn:          sessionMethodGetFlash,
			Arity:       "1",
			Description: "Get and clear flash message",
		},
		"getAllFlash": {
			Fn:          sessionMethodGetAllFlash,
			Arity:       "0",
			Description: "Get all flash messages",
		},
		"hasFlash": {
			Fn:          sessionMethodHasFlash,
			Arity:       "0",
			Description: "Check if any flash messages exist",
		},
		"regenerate": {
			Fn:          sessionMethodRegenerate,
			Arity:       "0",
			Description: "Regenerate session ID",
		},
	}
	RegisterMethodRegistry("session", SessionMethodRegistry)

	DevModuleMethodRegistry = MethodRegistry{
		"log": {
			Fn:          devMethodLog,
			Arity:       "1-3",
			Description: "Log value to dev panel",
		},
		"clearLog": {
			Fn:          devMethodClearLog,
			Arity:       "0",
			Description: "Clear dev log",
		},
		"logPage": {
			Fn:          devMethodLogPage,
			Arity:       "2-4",
			Description: "Log value for a specific route",
		},
		"setLogRoute": {
			Fn:          devMethodSetLogRoute,
			Arity:       "1",
			Description: "Set default log route",
		},
		"clearLogPage": {
			Fn:          devMethodClearLogPage,
			Arity:       "1",
			Description: "Clear page log for a route",
		},
	}
	RegisterMethodRegistry("dev", DevModuleMethodRegistry)
}

// ============================================================================
// Session method registry wrappers
// ============================================================================

func sessionMethodGet(receiver Object, args []Object, env *Environment) Object {
	return sessionGet(receiver.(*SessionModule), args)
}

func sessionMethodSet(receiver Object, args []Object, env *Environment) Object {
	return sessionSet(receiver.(*SessionModule), args)
}

func sessionMethodDelete(receiver Object, args []Object, env *Environment) Object {
	return sessionDelete(receiver.(*SessionModule), args)
}

func sessionMethodHas(receiver Object, args []Object, env *Environment) Object {
	return sessionHas(receiver.(*SessionModule), args)
}

func sessionMethodClear(receiver Object, args []Object, env *Environment) Object {
	return sessionClear(receiver.(*SessionModule))
}

func sessionMethodAll(receiver Object, args []Object, env *Environment) Object {
	return sessionAll(receiver.(*SessionModule))
}

func sessionMethodFlash(receiver Object, args []Object, env *Environment) Object {
	return sessionFlash(receiver.(*SessionModule), args)
}

func sessionMethodGetFlash(receiver Object, args []Object, env *Environment) Object {
	return sessionGetFlash(receiver.(*SessionModule), args)
}

func sessionMethodGetAllFlash(receiver Object, args []Object, env *Environment) Object {
	return sessionGetAllFlash(receiver.(*SessionModule))
}

func sessionMethodHasFlash(receiver Object, args []Object, env *Environment) Object {
	return sessionHasFlash(receiver.(*SessionModule))
}

func sessionMethodRegenerate(receiver Object, args []Object, env *Environment) Object {
	return sessionRegenerate(receiver.(*SessionModule))
}

// ============================================================================
// Dev module method registry wrappers
// ============================================================================

func devMethodLog(receiver Object, args []Object, env *Environment) Object {
	return receiver.(*DevModule).evalLog(args, env)
}

func devMethodClearLog(receiver Object, args []Object, env *Environment) Object {
	return receiver.(*DevModule).evalClearLog(args, env)
}

func devMethodLogPage(receiver Object, args []Object, env *Environment) Object {
	return receiver.(*DevModule).evalLogPage(args, env)
}

func devMethodSetLogRoute(receiver Object, args []Object, env *Environment) Object {
	return receiver.(*DevModule).evalSetLogRoute(args, env)
}

func devMethodClearLogPage(receiver Object, args []Object, env *Environment) Object {
	return receiver.(*DevModule).evalClearLogPage(args, env)
}
