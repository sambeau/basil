# Analysis Report: `let` and `const` in Parsley

**Date:** January 2025  
**Status:** Design Analysis → **Decision Made** → **[FEAT-122](../specs/FEAT-122.md)**  
**Author:** AI Assistant

---

## Executive Summary

This report analyzes whether Parsley should:
1. Add a `const` keyword for immutable bindings
2. Make `let` mandatory (currently optional)
3. Change the default mutability semantics

**Key Finding:** The pre-1.0 window is indeed the right time to address this. Post-1.0 changes to variable declaration syntax would require a breaking 2.0 release.

### Decision

**Adopt Swift-style semantics:**
- `let` = immutable (constant)
- `var` = mutable (variable)

This is the "correct" design, matching mathematical semantics, and we have a clean slate to do it right.

---

## 1. Current State in Parsley

### 1.1 How `let` Works Today

From `reference.md`:

```parsley
let x = 5
let name = "Alice"
```

> **Note**: `let` is technically optional for simple assignments (`x = 5` works), but using it is recommended for clarity. The keyword is reserved for potential future features like block-scoping or immutability.

**Current behavior:**
- `let x = 5` — declares and initializes
- `x = 5` — also works (implicit declaration)
- All variables are mutable
- Lexical scoping with closure semantics
- `let` is a reserved keyword

### 1.2 The "Optional `let`" Problem

Having optional `let` creates inconsistency:

```parsley
let name = "Alice"    // explicit
age = 25              // implicit — same effect

// Which style should users prefer?
// Documentation says "recommended" but why?
```

This ambiguity is confusing for users and makes the language feel unfinished.

---

## 2. Survey of Modern Languages

### 2.1 Languages WITH Immutability Keywords

| Language | Mutable | Immutable | Default | Notes |
|----------|---------|-----------|---------|-------|
| **JavaScript** | `let` | `const` | — | Both explicit; `const` preferred by linters |
| **TypeScript** | `let` | `const` | — | Same as JS; `const` strongly preferred |
| **Rust** | `let mut` | `let` | Immutable | Immutable by default; must opt-in to mutate |
| **Kotlin** | `var` | `val` | — | Both explicit; `val` preferred |
| **Swift** | `var` | `let` | — | Both explicit; `let` preferred |
| **Scala** | `var` | `val` | — | `val` strongly preferred; `var` discouraged |
| **Go** | `:=` / `var` | `const` | Mutable | `const` only for compile-time constants |
| **C#** | `var` | `const`/`readonly` | Mutable | `const` for compile-time; `readonly` for runtime |
| **Java** | — | `final` | Mutable | `final` is a modifier, not a declaration |
| **Dart** | `var` | `final`/`const` | — | `final` runtime, `const` compile-time |
| **Zig** | `var` | `const` | — | Both explicit |

### 2.2 Languages WITHOUT Immutability Keywords

| Language | Declaration | Notes |
|----------|-------------|-------|
| **Python** | `x = 5` | No const; convention is `UPPER_CASE` for "constants" |
| **Ruby** | `x = 5` | No const; `UPPER_CASE` triggers warning on reassign |
| **PHP** | `$x = 5` | `const` exists but only at class/global scope |
| **Lua** | `local x = 5` | No const (though Lua 5.4 added `<const>` attribute) |
| **Perl** | `my $x = 5` | No native const; `use constant` is a workaround |
| **Shell** | `x=5` | `readonly x` exists but rarely used |

### 2.3 Key Observations

1. **Almost all modern languages have const/immutability** — Python and Ruby are notable exceptions, but both are frequently criticized for this.

2. **The trend is toward immutable-by-default** — Rust, Scala, and functional languages make mutability opt-in.

3. **`let` means different things:**
   - JavaScript/Rust: mutable binding
   - Swift: immutable binding
   - Mathematics: immutable (from "let x = ...")
   
4. **Python's lack of const is seen as a weakness** — Common interview question: "How do you define constants in Python?" Answer: "You can't, really."

---

## 3. What Does `let` Actually Mean?

### 3.1 Etymology and Usage

The word "let" comes from mathematics: "Let x = 5" means "assume x has the value 5 for the following discussion." In math, this is inherently immutable — you don't reassign x mid-proof.

| Context | Meaning of `let` |
|---------|------------------|
| **Mathematics** | Immutable assumption |
| **JavaScript** | Mutable variable (contrast with `const`) |
| **Swift** | Immutable constant |
| **Rust** | Immutable by default, `let mut` for mutable |
| **Lisp/Scheme** | Local binding (typically immutable) |
| **Haskell** | Local binding (immutable, it's Haskell) |

**Verdict:** Outside JavaScript, `let` typically implies immutability or at least doesn't imply mutability.

### 3.2 The JavaScript Anomaly

JavaScript's `let` is mutable because of historical accident:
- `var` was the original (function-scoped, hoisted)
- `let` was added in ES6 for block-scoping
- `const` was added alongside `let`
- The names were chosen for contrast: `let` (changeable) vs `const` (fixed)

This is an artifact of JavaScript's evolution, not a principled design choice.

---

## 4. Design Options for Parsley

### Option A: Add `const`, Keep Optional `let`

```parsley
let x = 5       // mutable (explicit)
x = 5           // mutable (implicit) — still works
const y = 10    // immutable
```

**Pros:**
- Minimal change
- Backward compatible
- Familiar to JS developers

**Cons:**
- Optional `let` remains confusing
- Three ways to declare variables
- Doesn't address the core inconsistency

### Option B: Add `const`, Make `let` Required

```parsley
let x = 5       // mutable (required)
const y = 10    // immutable
// x = 5        // ERROR: must use let or const
```

**Pros:**
- Clear and consistent
- Familiar to JS/TS developers
- Forces explicit declaration

**Cons:**
- Breaking change (implicit declarations break)
- More verbose
- `let` meaning differs from Swift/Rust/math

### Option C: Immutable by Default (Rust-style)

```parsley
let x = 5       // immutable
var y = 10      // mutable
// x = 5        // ERROR: x is immutable
```

**Pros:**
- Aligns with modern best practices
- Encourages functional style
- `let` matches mathematical meaning
- Catches accidental mutation bugs

**Cons:**
- Significant breaking change
- Unfamiliar to JS developers
- More friction for simple scripts

### Option D: Swift-style (`let`/`var`)

```parsley
let x = 5       // immutable
var y = 10      // mutable
```

**Pros:**
- Clean, simple, symmetric
- Clear intent in both directions
- `let` matches mathematical meaning
- Well-proven in Swift

**Cons:**
- Breaking change
- `var` might confuse JS developers (where `var` is legacy)

### Option E: Kotlin-style (`val`/`var`)

```parsley
val x = 5       // immutable (value)
var y = 10      // mutable (variable)
```

**Pros:**
- Clean mnemonic: val=value, var=variable
- No overloaded meanings
- Well-proven in Kotlin

**Cons:**
- Breaking change
- `val` is not widely known outside Kotlin/Scala

### Option F: Status Quo + Require `let`

```parsley
let x = 5       // mutable (required)
// x = 5        // ERROR: must use let
// No const
```

**Pros:**
- Minimal change
- Consistent (one way to declare)
- Simple mental model

**Cons:**
- No immutability option
- May be seen as lacking compared to other languages
- Misses opportunity to add const

---

## 5. The "Do We Need `const`?" Question

### 5.1 Arguments FOR `const`

1. **Intent Documentation** — `const` signals "this will not change" to readers
2. **Bug Prevention** — Compiler catches accidental reassignment
3. **Optimization** — Interpreter can optimize immutable bindings
4. **Industry Expectation** — Most modern languages have it
5. **Best Practices** — "Prefer const" is common advice in JS/TS

### 5.2 Arguments AGAINST `const`

1. **Simplicity** — One less concept to learn
2. **Python/Ruby Success** — Major languages survive without it
3. **False Security** — `const obj = {}` doesn't prevent `obj.x = 1`
4. **Scripting Nature** — Parsley is for quick scripts, not large systems
5. **YAGNI** — Maybe users don't actually need it

### 5.3 The Python Counter-Example

Python lacks `const` and is extremely successful. However:
- Python developers frequently cite this as a pain point
- `UPPER_CASE` convention exists specifically to work around it
- Type checkers (mypy) added `Final` to address the gap
- Large Python codebases often have accidental mutation bugs

**Python proves you can succeed without `const`, but doesn't prove `const` is unnecessary.**

---

## 6. Mutability Default Analysis

### 6.1 "Declare Mutables" (Immutable by Default)

Used by: Rust, Scala, Haskell, F#, Clojure

```parsley
let x = 5       // can't reassign
var y = 10      // can reassign
y = 20          // OK
x = 15          // ERROR
```

**Philosophy:** Mutation is the exception, not the rule. Forces developers to think about what actually needs to change.

**Best for:** Functional programming, concurrent code, large systems

### 6.2 "Declare Immutables" (Mutable by Default)

Used by: JavaScript, Go, C#, Java

```parsley
let x = 5       // can reassign
const y = 10    // can't reassign
x = 15          // OK
y = 20          // ERROR
```

**Philosophy:** Mutability is normal; mark the special cases that shouldn't change.

**Best for:** Imperative programming, quick scripts, procedural code

### 6.3 Recommendation for Parsley

Parsley is a **scripting language** for **web development** and **quick tasks**. Users expect:
- Fast iteration
- Familiar syntax (JavaScript-adjacent)
- Low ceremony

**Immutable-by-default would be too much friction** for the target use case. A simple loop:

```parsley
// With immutable-by-default:
var sum = 0
for (n in [1,2,3,4,5]) {
    sum = sum + n   // Would need var
}

// With mutable-by-default:
let sum = 0
for (n in [1,2,3,4,5]) {
    sum = sum + n   // Just works
}
```

---

## 7. Recommendation

### ~~Primary Recommendation: Option B — Add `const`, Require `let`~~ SUPERSEDED

### Final Decision: Option D — Swift-style (`let`/`var`)

```parsley
let x = 5       // immutable (constant)
var y = 10      // mutable (variable)
y = 20          // OK - reassignment
// x = 15       // ERROR: cannot reassign immutable binding
```

**Rationale:**
1. **Semantic Correctness** — `let` matches mathematical meaning ("let x = 5")
2. **Clean Slate** — No legacy to maintain; do it right from the start
3. **Modern Design** — Aligns with Swift, Rust (partially), Kotlin (val/var)
4. **Clear Intent** — Both keywords explicitly communicate mutability
5. **Avoid JavaScript's Mistake** — JS would undo `let`=mutable if they could

### Migration Strategy

1. **Phase 1 (v0.x.x):**
   - Add `var` keyword for mutable bindings
   - Deprecate implicit declarations with warning
   - `let` continues to work as mutable (transitional)
   
2. **Phase 2 (v0.x+1.x):**
   - `let` becomes immutable
   - Code using `let` for mutated variables gets error
   - Provide migration tool: `pars --migrate-let-var`

3. **Phase 3 (v1.0):**
   - Clean semantics: `let`=immutable, `var`=mutable
   - Implicit declarations are errors

---

## 8. Implementation Considerations

### 8.1 What Immutability Means

**Shallow immutability** (binding, not contents):
```parsley
let arr = [1, 2, 3]
arr[0] = 99     // OK — mutating contents
arr = [4, 5]    // ERROR — reassigning binding

let obj = {x: 1}
obj.x = 2       // OK — mutating property
obj = {y: 3}    // ERROR — reassigning binding
```

**Rationale:** Deep immutability is complex and has performance implications. Shallow immutability is well-understood from Swift, JavaScript's `const`, etc.

### 8.2 Reserved Keywords

Current reserved keywords (from `lexer.go`):
```
fn, function, let, for, in, if, else, return, export, import,
try, check, stop, skip, true, false, null, and, or, as, via
```

**Action Required:** Add `var` to reserved keywords immediately.

Note: `const` is already in `ParsleyKeywords` in `errors.go` but not in lexer. This is fine since we're not using `const`.

### 8.3 Export Interaction

```parsley
export let PI = 3.14159         // Immutable export (constant)
export var counter = 0          // Mutable export (if needed)
```

### 8.4 Destructuring

```parsley
let [a, b] = [1, 2]             // a and b are immutable
var [x, y] = [1, 2]             // x and y are mutable

let {name, age} = person        // immutable bindings
var {name, age} = person        // mutable bindings
```

### 8.5 Codebase Impact Analysis

#### Files Affected

| Area | Count | Notes |
|------|-------|-------|
| `.pars` files total | 166 | Across entire codebase |
| Files using `let` | 133 | Need review for mutability |
| Total `let` usages | ~994 | Lines containing `let` |
| Reassignment patterns | ~6 | `x = x + n` style mutations |

#### Go Implementation Files

| File | Changes Needed |
|------|----------------|
| `lexer/lexer.go` | Add `VAR` token type, add `"var": VAR` to keywords map |
| `ast/ast.go` | Add `Mutable bool` field to `LetStatement`, rename to `BindingStatement`? |
| `parser/parser.go` | Handle `var` keyword, track mutability |
| `evaluator/evaluator.go` | Track immutable bindings, error on reassignment |
| `errors/errors.go` | Add error for immutable reassignment |

#### Environment Changes

Current `Environment` struct has:
```go
type Environment struct {
    store         map[string]Object
    letBindings   map[string]bool  // tracks 'let' declarations
    protected     map[string]bool  // tracks protected variables
    // ...
}
```

**Change to:**
```go
type Environment struct {
    store         map[string]Object
    immutable     map[string]bool  // tracks 'let' (immutable) bindings
    protected     map[string]bool  // tracks protected variables (imports, etc.)
    // ...
}
```

The `Update()` method already checks `IsProtected()`. We need to also check `IsImmutable()`:

```go
func (e *Environment) Update(name string, val Object) Object {
    // Check if variable is protected
    if e.IsProtected(name) {
        return &Error{Message: fmt.Sprintf("cannot reassign protected variable '%s'", name)}
    }
    // Check if variable is immutable (declared with 'let')
    if e.IsImmutable(name) {
        return &Error{Message: fmt.Sprintf("cannot reassign immutable binding '%s' (declared with 'let')", name)}
    }
    // ... rest of method
}
```

---

## 9. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| User confusion during migration | Medium | Low | Clear deprecation warnings, migration tool |
| Breaking existing scripts | High | Medium | Phased rollout, auto-migration tool |
| `let` meaning change | Medium | Medium | Clear messaging: "let now means constant" |
| Performance regression | Low | Low | Shallow immutability is trivial to check |
| Scope creep (deep immutability) | Medium | Low | Document that `let` is shallow binding immutability |
| JS developers expect `let`=mutable | Medium | Low | Documentation, error messages explain Swift model |

---

## 10. Conclusion

### Do We Need Immutability?

**Yes.** Almost all modern languages have it. The pre-1.0 window is the right time.

### What Should `let` Mean?

**Immutable.** This matches:
- Mathematics ("let x = 5" is an assumption, not a variable)
- Swift (major modern language)
- Rust's default `let` (before `mut`)
- Common functional programming convention

### What About Mutable Bindings?

**Use `var`.** Clear, explicit, well-understood:
- Swift uses `var`
- Kotlin uses `var` (with `val` for immutable)
- JavaScript's legacy `var` gives it familiarity (even if different semantics)

### Final Decision

**Implement Swift-style: `let`=immutable, `var`=mutable**

```parsley
let x = 5       // immutable (constant binding)
var y = 10      // mutable (variable binding)
```

This is the correct design. We have a clean slate — let's use it.

---

## 11. Implementation Roadmap

### Phase 1: Add `var` (Non-Breaking)

1. Add `VAR` token to lexer
2. Parse `var` as mutable binding
3. `let` continues to work as before (mutable, for compatibility)
4. Add deprecation warning for implicit declarations
5. Update documentation to recommend `var` for mutable bindings

### Phase 2: Flip `let` Semantics (Breaking)

1. `let` becomes immutable
2. Error on reassigning `let` bindings
3. Existing code using `let` for mutated vars breaks → use `var`
4. Provide `pars --migrate-let-var` tool to auto-fix

### Phase 3: Require Explicit Declaration (Breaking)

1. Remove implicit declaration support
2. All bindings must use `let` or `var`
3. Error messages guide users to correct syntax

### Estimated Effort

| Task | Effort | Risk |
|------|--------|------|
| Add `var` keyword (lexer, parser) | 2-3 hours | Low |
| Track mutability in AST | 1-2 hours | Low |
| Enforce immutability in evaluator | 2-3 hours | Medium |
| Migration tool | 4-6 hours | Low |
| Update all documentation | 4-8 hours | Low |
| Update test files (~133 files) | 2-4 hours | Low |
| **Total** | **~2-3 days** | |

---

## Appendix: Quick Reference of Other Languages

### JavaScript/TypeScript
```javascript
let x = 5;      // mutable
const y = 10;   // immutable (shallow)
var z = 15;     // legacy, avoid
```

### Rust
```rust
let x = 5;      // immutable
let mut y = 10; // mutable
```

### Swift
```swift
let x = 5       // immutable
var y = 10      // mutable
```

### Kotlin
```kotlin
val x = 5       // immutable
var y = 10      // mutable
```

### Go
```go
const x = 5     // compile-time constant only
y := 10         // mutable (short declaration)
var z = 15      // mutable (long declaration)
```

### Python
```python
x = 5           # mutable (no const)
X_CONSTANT = 5  # convention only
```

### Ruby
```ruby
x = 5           # mutable
X_CONSTANT = 5  # warning on reassign, but allowed
```

---

## Appendix B: Parsley Code Requiring Migration

### Files with Reassignment Patterns (Need `var`)

These files use `x = x + n` style mutations and will need `var`:

```
./contrib/zed-extension/test/sample.pars:27:    total = total + n
./examples/parsley/temp/counter.pars:3:         count = count + 1
./examples/parsley/temp/debug_counter.pars:4:   count = count + 1
./examples/parsley/temp/test_examples.pars:38:  total_age = total_age + u.age
./examples/parsley/sftp_demo.pars:225:          total = total + parseInt(row[1])
./examples/parsley/modules/arrays.pars:5:       total = total + item
```

### Files Using Implicit Declaration (Need `let` or `var`)

Files like `path_url_interpolation_demo.pars` use implicit declarations:
```parsley
name = "config"     // implicit — needs 'let name = "config"'
p = @(./data/{name}.json)
```

A migration tool can automatically prefix these with `let` (safe default since most are not reassigned).
