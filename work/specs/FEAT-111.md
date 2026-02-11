---
id: FEAT-111
title: "Declarative Method Registry"
status: draft
priority: high
created: 2025-01-15
author: "@human"
blocking: false
---

# FEAT-111: Declarative Method Registry

## Summary
Refactor `methods.go` to use a declarative method registry instead of switch statements. This creates a single source of truth for method dispatch AND introspection, eliminating the synchronization problem between `methods.go` and `introspect.go`.

## User Story
As a Parsley maintainer, I want method definitions to be self-documenting so that adding a new method automatically makes it discoverable via `describe()` and the help system.

As a Parsley developer, I want `describe(type)` to always show accurate, up-to-date method information so that I can trust the introspection output.

## Acceptance Criteria
- [ ] Method dispatch uses registry lookup instead of switch statements
- [ ] Each method entry includes: function reference, arity, and description
- [ ] `TypeMethods` map in `introspect.go` is removed (now redundant)
- [ ] Introspection reads directly from the method registries
- [ ] All existing method tests continue to pass
- [ ] No performance regression in method dispatch
- [ ] Adding a new method requires only one code change (registry entry)

## Design Decisions

- **Registry per type**: Each type has its own registry map (e.g., `StringMethods`, `ArrayMethods`) for locality and maintainability.

- **Map-based dispatch**: Replace switch statements with map lookups. Maps are O(1) and may be faster than large switches.

- **Keep method implementations separate**: The actual method functions remain as standalone functions; the registry just references them.

- **Backward compatible**: No changes to method call syntax or behavior from user perspective.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
| Component | Location | Changes |
|-----------|----------|---------|
| Method dispatch | `pkg/parsley/evaluator/methods.go` | Major refactor (~2000 lines) |
| Introspection | `pkg/parsley/evaluator/introspect.go` | Remove `TypeMethods`, read from registries |
| Tests | `pkg/parsley/evaluator/*_test.go` | Update if needed |

### Dependencies
- Depends on: FEAT-110 (validation tests provide safety net for refactor)
- Blocks: FEAT-112 (help system reads from registry)

### Effort Estimate
8-12 hours (significant refactoring)

---

## Implementation Plan

### Phase 1: Define Registry Structure

**New types in `methods.go`:**

```go
// MethodEntry defines a single method with its implementation and metadata
type MethodEntry struct {
    Fn          func(obj Object, args []Object, env *Environment) Object
    Arity       string // "0", "1", "0-1", "1+", "2"
    Description string
}

// MethodRegistry maps method names to their entries for a type
type MethodRegistry map[string]MethodEntry
```

### Phase 2: Create Registries for Each Type

**Example: String methods**

```go
var stringMethods = MethodRegistry{
    "toUpper": {
        Fn:          stringToUpper,
        Arity:       "0",
        Description: "Convert to uppercase",
    },
    "toLower": {
        Fn:          stringToLower,
        Arity:       "0",
        Description: "Convert to lowercase",
    },
    "trim": {
        Fn:          stringTrim,
        Arity:       "0",
        Description: "Remove leading and trailing whitespace",
    },
    "split": {
        Fn:          stringSplit,
        Arity:       "0-1",
        Description: "Split string by delimiter (default: whitespace)",
    },
    // ... all string methods
}

// Method implementation (existing code, minor signature change)
func stringToUpper(obj Object, args []Object, env *Environment) Object {
    s := obj.(*String)
    return &String{Value: strings.ToUpper(s.Value)}
}
```

**Types to create registries for:**
- `stringMethods`
- `arrayMethods`
- `dictionaryMethods`
- `integerMethods`
- `floatMethods`
- `datetimeMethods`
- `durationMethods`
- `moneyMethods`
- `regexMethods`
- `dbConnectionMethods`
- `fileMethods`
- `responseMethods`
- `requestMethods`
- `sftpConnectionMethods`
- `errorMethods`

### Phase 3: Refactor Method Dispatch

**Current dispatch (switch-based):**

```go
func evalMethod(obj Object, method string, args []Object, env *Environment) Object {
    switch obj := obj.(type) {
    case *String:
        switch method {
        case "toUpper":
            return &String{Value: strings.ToUpper(obj.Value)}
        case "toLower":
            return &String{Value: strings.ToLower(obj.Value)}
        // ... 30+ more cases
        }
    case *Array:
        // ... another large switch
    // ... many more type cases
    }
}
```

**New dispatch (registry-based):**

```go
// Master registry mapping type names to their method registries
var methodRegistries = map[string]MethodRegistry{
    "string":         stringMethods,
    "array":          arrayMethods,
    "dictionary":     dictionaryMethods,
    "integer":        integerMethods,
    "float":          floatMethods,
    "datetime":       datetimeMethods,
    "duration":       durationMethods,
    "money":          moneyMethods,
    "regex":          regexMethods,
    "db_connection":  dbConnectionMethods,
    "file":           fileMethods,
    "response":       responseMethods,
    "request":        requestMethods,
    "sftp_connection": sftpConnectionMethods,
    "error":          errorMethods,
}

func evalMethod(obj Object, method string, args []Object, env *Environment) Object {
    typeName := strings.ToLower(string(obj.Type()))
    
    registry, ok := methodRegistries[typeName]
    if !ok {
        return newError("type %s has no methods", obj.Type())
    }
    
    entry, ok := registry[method]
    if !ok {
        return newError("unknown method '%s' for type %s", method, obj.Type())
    }
    
    // Arity checking
    if !checkArity(entry.Arity, len(args)) {
        return newArityError(method, entry.Arity, len(args))
    }
    
    return entry.Fn(obj, args, env)
}

func checkArity(spec string, got int) bool {
    switch spec {
    case "0":
        return got == 0
    case "1":
        return got == 1
    case "2":
        return got == 2
    case "0-1":
        return got == 0 || got == 1
    case "1-2":
        return got == 1 || got == 2
    case "0-2":
        return got >= 0 && got <= 2
    case "1+":
        return got >= 1
    case "0+":
        return got >= 0
    default:
        return true // Unknown spec, allow
    }
}
```

### Phase 4: Update Introspection

**Remove from `introspect.go`:**
- `var TypeMethods = map[string][]MethodInfo{...}` (entire map, ~200 lines)

**Add helper function:**

```go
// GetMethodsForType returns method info for a type from the registry
func GetMethodsForType(typeName string) []MethodInfo {
    registry, ok := methodRegistries[typeName]
    if !ok {
        return nil
    }
    
    methods := make([]MethodInfo, 0, len(registry))
    for name, entry := range registry {
        methods = append(methods, MethodInfo{
            Name:        name,
            Arity:       entry.Arity,
            Description: entry.Description,
        })
    }
    
    // Sort alphabetically for consistent output
    sort.Slice(methods, func(i, j int) bool {
        return methods[i].Name < methods[j].Name
    })
    
    return methods
}
```

**Update `builtinDescribe()` to use `GetMethodsForType()`**

---

## Test Plan

| Test | Expected |
|------|----------|
| All existing method tests | Pass unchanged |
| `"hello".toUpper()` | Returns "HELLO" |
| `[1,2,3].map(fn(x) x*2)` | Returns [2,4,6] |
| `describe("hello")` | Shows all string methods |
| Benchmark: method dispatch | No significant regression |
| Unknown method error | Clear error message |
| Arity error | Clear error message with expected arity |

### Migration Verification

```bash
# Before refactor: run full test suite, save output
go test ./pkg/parsley/... > before.txt

# After refactor: run full test suite, compare
go test ./pkg/parsley/... > after.txt
diff before.txt after.txt  # Should be identical (or only timing diffs)
```

---

## Rollout Strategy

1. **Create registry structure** — Add types, don't change dispatch yet
2. **Populate registries** — Copy info from current switch statements
3. **Add parallel dispatch** — New registry-based path, enabled by flag
4. **Test thoroughly** — Ensure parity with switch-based dispatch  
5. **Switch over** — Make registry-based dispatch the default
6. **Remove old code** — Delete switch statements and `TypeMethods` map
7. **Update introspection** — Point to new registries

---

## Implementation Notes
*To be added during implementation*

## Related
- Report: `work/reports/PARSLEY-1.0-ALPHA-READINESS.md` (Section 2)
- Validation tests: FEAT-110 (provides safety net)
- Help system: FEAT-112 (consumes this registry)
- Current methods: `pkg/parsley/evaluator/methods.go`
- Current introspection: `pkg/parsley/evaluator/introspect.go`
