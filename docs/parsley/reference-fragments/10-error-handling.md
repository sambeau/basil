## 10. Error Handling

Parsley provides structured error handling through the `try` expression and `fail` function.

### 10.1 The `try` Expression

The `try` expression catches certain errors and returns them as values instead of terminating execution. It wraps the result in a dictionary with `result` and `error` fields. The `error` slot is a dictionary (with at least `message` and `code`) on failure, or `null` on success.

**Syntax**: `try` only accepts **function calls** or **method calls**, not arbitrary expressions or blocks.

```parsley
let result = try someFunction(args)
// Returns: {result: value, error: null} on success
// Returns: {result: null, error: {message: "...", code: "..."}} on catchable error

let result2 = try obj.method(args)
// Same pattern for method calls
```

#### Success Case

When the function succeeds, `result` contains the return value and `error` is `null`:

```parsley
let add = fn(a, b) { a + b }
let res = try add(2, 3)
res.result                      // 5
res.error                       // null
```

#### Error Case

When a catchable error occurs, `result` is `null` and `error` is a dictionary:

```parsley
let validate = fn(x) {
    check x > 0 else fail("must be positive")
    x * 2
}

let res = try validate(-5)
res.result                      // null
res.error                       // {message: "must be positive", code: "USER-0001"}
res.error.message               // "must be positive"
```

#### Pattern: Check and Handle

Use destructuring with conditionals for clean error handling:

```parsley
let {result, error} = try riskyOperation()
if (error) {
    <div class="error">"Operation failed: " + error.message</div>
} else {
    <div class="success">"Result: " + toString(result)</div>
}
```

> 💡 String coercion: `"" + error` automatically yields `error.message`, so `"Failed: " + error` also works.

#### Pattern: Default with Null Coalescing

Use `??` to provide fallback values:

```parsley
let result = (try parseJSON(input)).result ?? {}
let data = (try loadConfig()).result ?? {default: true}
```

---

### 10.2 The `fail` Function

Use `fail(message)` or `fail(dict)` to create a catchable error. This is the primary way to signal errors in validation and business logic.

#### String Form

```parsley
let validateEmail = fn(email) {
    check email.includes("@") else fail("Invalid email format")
    check email.length() > 3 else fail("Email too short")
    email
}

let result = try validateEmail("bad")
result.error.message            // "Email too short"
```

#### Dictionary Form

Pass a dictionary with at least a `message` key (string). Optional `code` and `status` fields control error identity and HTTP status:

```parsley
fail({
    message: "Out of stock",
    code: "NO_STOCK",
    status: 400,
    product: "Widget"
})
```

All fields in the dictionary are preserved and available when caught by `try`.

**Important**: `fail()` creates **catchable** errors (class: "value"). They can be caught by `try` expressions.

---

### 10.3 Catchable vs Non-Catchable Errors

Not all errors can be caught by `try`. Parsley distinguishes between:

- **Catchable errors** — External/runtime errors that may occur despite correct code (network failures, invalid user input, file not found)
- **Non-catchable errors** — Developer errors that indicate bugs (type mismatches, wrong number of arguments, undefined variables)

#### Catchable Error Classes

These errors **can** be caught by `try`:

| Class | Examples |
|-------|----------|
| **Value** | Created by `fail()`, `api.*` helpers, empty required fields |
| **Format** | Invalid URL, malformed JSON, bad date string |
| **IO** | File not found, permission denied |
| **Network** | HTTP request failure, timeout |
| **Database** | Connection failed, query error |
| **Security** | Access denied, authentication required |

```parsley
// These CAN be caught:
try url("not a valid url")      // Format error - invalid URL
try fail("custom error")         // Value error
try readFile(@./missing.txt)    // IO error - file not found
```

#### Non-Catchable Errors

These errors **cannot** be caught by `try` — they propagate and terminate execution:

| Class | Examples |
|-------|----------|
| **Type** | Wrong type passed to function or method |
| **Arity** | Wrong number of function arguments |
| **Undefined** | Variable, function, or method not found |
| **Index** | Array index out of bounds |
| **Operator** | Invalid operation (e.g., adding incompatible types) |
| **State** | Invalid state transition |

```parsley
// These CANNOT be caught - they propagate:
try unknownFunction()           // Undefined error - propagates
try "text".split(123)           // Type error - propagates
try someFunc()                  // Arity error if wrong args - propagates
```

**Why?** Non-catchable errors indicate bugs in your code. They should fail loudly during development so you fix them, not be silently caught at runtime.

---

### 10.4 Error Prevention

#### Check Guards

Use `check...else` for validation with early returns:

```parsley
let processOrder = fn(order) {
    check order else fail("Order required")
    check order.items else fail("Order must have items")
    check order.total > 0 else fail("Order total must be positive")
    // Process order...
}
```

#### Absence never errors

Reaching into `null`, or asking for a missing dictionary key, returns `null` — so chains walk through missing data instead of crashing. Dot and bracket access behave identically:

```parsley
let user = {name: "Alice"}
user["email"]                   // null (missing key)
user.email                      // null (dot access, same)
user["profile"]["city"]         // null (chain through a missing intermediate)
null["anything"]                // null (indexing into null)
```

#### Optional Index Access

Only an out-of-range position on a *present* array or string is an error. `[?index]` opts into `null` instead:

```parsley
let arr = [1, 2, 3]
arr[?99]                        // null (no error)
arr[99]                         // Error: index out of bounds
```

Dictionaries have no positional range, so `[?]` is unnecessary there — `user[?"email"]` is identical to `user["email"]`.

#### Null Coalescing

Use `??` to provide default values:

```parsley
let name = user.name ?? "Anonymous"
let config = loadConfig() ?? {default: true}
```

#### Asserting presence

The strict counterpart to forgiving access: `?? fail("…")` stops with a clear, located error where a value is required, instead of letting `null` propagate silently:

```parsley
let dbUrl = config["database"]["url"] ?? fail("database url is required")
```

---

