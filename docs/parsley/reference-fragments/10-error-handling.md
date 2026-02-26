## 10. Error Handling

Parsley provides structured error handling through `try` expressions and the `fail` function.

### 10.1 The `try` Expression

Wrap potentially failing operations in `try` to capture errors:

```parsley
let {data, error} = try {
    JSON(<== @./config.json)
}

if (error) {
    log("Failed to load config:", error.message)
    {}  // Return empty dict as fallback
} else {
    data
}
```

The result is always a dictionary with:
- `data` — The result if successful, or `null` if an error occurred
- `error` — `null` if successful, or an error dictionary if failed

### 10.2 Error Dictionary

When an error occurs, `error` is a dictionary containing:

| Key | Type | Description |
|-----|------|-------------|
| `message` | string | Human-readable error message |
| `code` | string | Error code (e.g., `"io_error"`, `"parse_error"`) |
| `details` | dict | Additional error-specific information |

```parsley
let {data, error} = try { JSON(<== @./missing.json) }

if (error) {
    error.message  // "file not found: ./missing.json"
    error.code     // "io_error"
}
```

### 10.3 The `fail` Function

Explicitly raise an error:

```parsley
// Simple string message
fail("Something went wrong")

// Structured error with code
fail({
    message: "Validation failed",
    code: "validation_error",
    details: {field: "email", reason: "invalid format"}
})
```

### 10.4 Catchable vs Non-Catchable Errors

**Catchable** (can be caught with `try`):
- I/O errors (file not found, network failure)
- Parse errors (invalid JSON, malformed data)
- Validation errors (schema mismatch)
- Explicit `fail()` calls

**Non-Catchable** (program bugs, always fatal):
- Syntax errors
- Type errors
- Undefined variable access
- Division by zero

### 10.5 Null Coalescing for Simple Cases

For simple default values, use `??` instead of `try`:

```parsley
// Instead of try/catch for optional file
let config = (try { JSON(<== @./config.json) }).data ?? {}

// With optional chaining
let name = user?.profile?.name ?? "Anonymous"
```

### 10.6 Check Guards

Use `check` for early exits on validation failure:

```parsley
fn processUser(user) {
    check user != null else { fail("User required") }
    check user.email != null else { fail("Email required") }
    
    // Continue with valid user
    sendWelcomeEmail(user.email)
}
```
