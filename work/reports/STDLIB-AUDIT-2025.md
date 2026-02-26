# Parsley Standard Library Audit Report

**Date:** 2025-02-26  
**Purpose:** Assess standard library fitness for v1.0 release  
**Status:** Complete — Ready for feature specification

---

## Executive Summary

This audit reviewed the Parsley standard library modules to assess their fitness for v1.0 release. The primary concerns were:

1. Overlap between standard library functions and builtin types/methods
2. Confusion between Parsley (`@std/`) and Basil (`@basil/`) namespaces
3. Whether the Query DSL makes some functionality obsolete
4. Gaps in validation coverage (particularly ID types)
5. Comparison with Python/Ruby/Rails standard libraries for missing functionality

**Overall recommendation:** Remove/deprecate redundant modules, slim down `@std/valid` to genuinely useful validators, ensure we can validate all ID types we can generate, move all server-specific modules to the `@basil/` namespace, add a new `@std/hash` module, and add missing string methods for base64/case conversion/truncation.

---

## Namespace Principle

- **`@std/`** — Pure Parsley functionality (works without Basil server)
- **`@basil/`** — Server-specific functionality (requires Basil runtime context)

Parsley should remain "pure" — a general-purpose language that can theoretically be used with other servers or environments. All HTTP and server-specific functionality belongs in `@basil/`.

---

## Module Decisions

### `@std/math` — ✅ KEEP (No changes)

**Quality: 9/10**

Well-designed, focused module following standard library conventions. No overlap with builtins or Query DSL.

**Contents:** Constants (PI, E, TAU), rounding, aggregation, statistics, random, transcendental functions, geometry/interpolation.

---

### `@std/id` — ✅ KEEP (No changes)

**Quality: 9/10**

Clean, focused module for ID generation.

**Contents:** `new()` (ULID), `uuid()`/`uuidv4()`, `uuidv7()`, `nanoid(len?)`, `cuid()`

---

### `@std/mdDoc` — ✅ KEEP (No changes)

**Quality: 7/10**

Clean API for markdown document parsing and querying. No overlap with other functionality.

---

### `@std/schema` — ❌ DEPRECATE

**Quality: 4/10**

**Reason:** Completely superseded by the `@schema` DSL syntax.

| `@std/schema` (verbose) | `@schema` DSL (native) |
|-------------------------|------------------------|
| `schema.string({minLength: 1, maxLength: 50})` | `string(1..50)` |
| `schema.email()` | `email` |
| `schema.define("User", {...})` | `@schema User {...}` |
| `schema.validate(User, data)` | `record.validate()` |

The DSL schema provides additional features not available in `@std/schema`:
- Relations (`User via user_id`)
- Field metadata pipe syntax (`| title: "..."`)
- Auto fields, nullable types
- Database binding integration
- Query DSL integration

**Action:** Add deprecation error with migration guide pointing to `@schema` syntax.

---

### `@std/valid` — 🔄 SLIM DOWN SIGNIFICANTLY

**Quality: 5/10 (current), targeting 8/10 (after changes)**

The module currently contains many redundant functions. After cleanup, it should contain only validators that:
- Are commonly needed in web development
- Don't make sense as schema storage types
- Cannot be easily expressed with native Parsley features

The module serves as the dedicated home for validation predicates — a cross-cutting concern used in form handlers and API endpoints. Keeping validators together (rather than splitting ID validators into `@std/id`) provides:
- A single import for validation code
- A natural home for future validators (IBAN, VAT numbers, etc.)
- Clear separation between generation (`@std/id`) and validation (`@std/valid`)

#### Functions to REMOVE:

| Function | Reason | Alternative |
|----------|--------|-------------|
| `string(x)` | Redundant | `inspect(x).type == "string"` |
| `number(x)` | Redundant | `inspect(x).type == "integer" or "float"` |
| `integer(x)` | Redundant | `inspect(x).type == "integer"` |
| `boolean(x)` | Redundant | `inspect(x).type == "boolean"` |
| `array(x)` | Redundant | `inspect(x).type == "array"` |
| `dict(x)` | Redundant | `inspect(x).type == "dictionary"` |
| `minLen(str, n)` | Redundant | Schema constraint `string(n..)` |
| `maxLen(str, n)` | Redundant | Schema constraint `string(..n)` |
| `length(str, min, max)` | Redundant | Schema constraint `string(min..max)` |
| `min(num, n)` | Redundant | Schema constraint `int(n..)` |
| `max(num, n)` | Redundant | Schema constraint `int(..n)` |
| `between(num, lo, hi)` | Redundant | Schema constraint `int(lo..hi)` |
| `positive(n)` | Redundant | `n > 0` |
| `negative(n)` | Redundant | `n < 0` |
| `email(str)` | Redundant | Schema type `email` |
| `url(str)` | Redundant | Schema type `url` |
| `phone(str)` | Redundant | Schema type `phone` |
| `matches(str, pattern)` | Redundant | `str ~ /pattern/` |
| `alpha(str)` | Redundant | `str ~ /^[a-zA-Z]+$/` |
| `alphanumeric(str)` | Redundant | `str ~ /^[a-zA-Z0-9]+$/` |
| `numeric(str)` | Redundant | `str ~ /^[0-9]+$/` or `try { float(str) }` |
| `empty(str)` | Redundant | `str.trim() == ""` |
| `contains(arr, val)` | Redundant | `val in arr` |
| `oneOf(val, arr)` | Redundant | `val in arr` |
| `date(str, locale?)` | Redundant | `datetime()` constructor handles this |
| `time(str)` | Redundant | `datetime()` constructor handles this |
| `parseDate(str, locale)` | Redundant | `datetime(str, {locale: "..."})` |

#### Functions to KEEP:

| Function | Reason |
|----------|--------|
| `uuid(str)` | Validate UUID format (v4/v7) |
| `creditCard(str)` | Luhn algorithm validation; credit cards need validation but shouldn't be stored |
| `luhn(str)` | Generic Luhn check; useful beyond credit cards |
| `postalCode(str, locale)` | Locale-aware validation (US, GB, CA); awkward as schema type |

#### Functions to ADD:

| Function | Reason |
|----------|--------|
| `ulid(str)` | Validate ULID format; parity with `uuid()` |
| `nanoid(str, len?)` | Validate NanoID format; `len` defaults to 21 |
| `cuid(str)` | Validate CUID2 format |

**Critical requirement:** We must be able to validate all ID types that `@std/id` can generate.

#### Final `@std/valid` API:

```parsley
import @std/valid

// ID validators
valid.uuid(str)           // UUID v4/v7 format
valid.ulid(str)           // ULID format (26 chars, Crockford Base32)
valid.nanoid(str, len?)   // NanoID format (default len=21)
valid.cuid(str)           // CUID2 format (25 chars, c prefix)

// Financial validators
valid.creditCard(str)     // Credit card number (Luhn check)
valid.luhn(str)           // Generic Luhn algorithm

// Locale-aware validators
valid.postalCode(str, locale)  // Postal code (US, GB, CA)
```

---

### `@std/hash` — 🆕 NEW MODULE

**Reason:** Hashing is a common need for checksums, cache keys, ETags, and non-security hashing. Python (`hashlib`), Ruby (`digest`), and most standard libraries include this.

#### API:

```parsley
import @std/hash

hash.md5("hello")       // "5d41402abc4b2a76b9719d911017c592"
hash.sha1("hello")      // "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d"
hash.sha256("hello")    // "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
hash.sha512("hello")    // (128-char hex string)
```

**Note:** These are for checksums and non-security purposes. For password hashing, use Basil's auth system.

---

### `@std/api` — 🔄 MOVE TO `@basil/api`

**Quality: 6/10**

This module provides HTTP API utilities (auth wrappers, error helpers, redirects) that are entirely Basil server-specific. Parsley does not currently support HTTP outside the Basil server environment.

**Contents:** Auth wrappers (`public`, `adminOnly`, `roles`, `auth`), error helpers (`notFound`, `forbidden`, `badRequest`, `unauthorized`, `conflict`, `serverError`), redirect helper (`redirect`)

**Action:** Move to `@basil/api` with deprecation alias for `@std/api`. Functionality review deferred to Basil 1.0 readiness.

---

### `@std/dev` — 🔄 MOVE TO `@basil/log`

**Quality: 7/10**

Development logging functionality that requires Basil server context. Already identified in backlog (#57).

**Contents:** `dev.log()`, `dev.clearLog()`, `dev.logPage()`, `dev.setLogRoute()`, `dev.clearLogPage()`

**Action:** Move to `@basil/log` with deprecation alias for `@std/dev`.

---

### `@std/html` — 🔄 MOVE TO `@basil/html`

**Quality: 7/10**

Entirely Basil-dependent (requires prelude loader). All components are `.pars` files loaded from Basil's prelude directory. HTML utility functions (`htmlEncode`, `htmlDecode`, `stripHtml`, etc.) are already available as string methods in the core language, so there's no functionality to split out.

**Contents:** Layout (`Page`, `Head`), Form (`TextField`, `Form`, etc.), Navigation (`Nav`, `Breadcrumb`), Media (`Img`, `Figure`), Utility (`SrOnly`, `A`, `Icon`), Time (`LocalTime`, `RelativeTime`), Table (`DataTable`)

**Action:** Move to `@basil/html` with deprecation alias for `@std/html`.

---

### `@std/table` — ✅ ALREADY DEPRECATED (Correct)

The code already returns an error pointing to `@table` literal syntax. No action needed.

---

## String Methods to Add

Comparison with Python/Ruby/Rails identified missing string functionality that is commonly needed in web development, particularly when Parsley sits between files, databases, and JavaScript.

### Base64 Encoding/Decoding

```parsley
"hello".toBase64()           // "aGVsbG8="
"aGVsbG8=".fromBase64()      // "hello"
```

**Use cases:** Encoding binary data for JSON/URLs, basic auth headers, data URIs.

### Case Conversion

```parsley
"hello_world".toCamel()      // "helloWorld"
"hello_world".toPascal()     // "HelloWorld"
"HelloWorld".toSnake()       // "hello_world"
"HelloWorld".toKebab()       // "hello-world"
```

**Use cases:** API field name conversion, JavaScript interop, database column mapping.

### Truncation

```parsley
"Hello world".truncate(8)        // "Hello..."
"Hello world".truncate(8, "…")   // "Hello…"
"Hi".truncate(8)                 // "Hi" (no change if shorter)
```

**Use cases:** Display truncation, preview text, UI constraints.

**Note:** While `if text.length() > 8 then text[:8]+"…" else text` works, the method form is cleaner for a common operation.

---

## Summary Table

| Module | Decision | Rationale |
|--------|----------|-----------|
| `@std/math` | ✅ Keep | Excellent, canonical |
| `@std/id` | ✅ Keep | Excellent, focused |
| `@std/mdDoc` | ✅ Keep | Good, no overlap |
| `@std/valid` | 🔄 Slim down | Remove 27 redundant functions, add 3 ID validators |
| `@std/hash` | 🆕 Add | New module for checksums (md5, sha1, sha256, sha512) |
| `@std/schema` | ❌ Deprecate | Superseded by `@schema` DSL |
| `@std/api` | 🔄 Move to `@basil/api` | Server-specific; review deferred to Basil 1.0 |
| `@std/dev` | 🔄 Move to `@basil/log` | Server-specific |
| `@std/html` | 🔄 Move to `@basil/html` | Server-specific; HTML utils already in string methods |
| `@std/table` | ✅ Already deprecated | No action needed |
| String methods | 🆕 Add | `toBase64`, `fromBase64`, `toCamel`, `toPascal`, `toSnake`, `toKebab`, `truncate` |

---

## Final `@std/` Namespace (Post-cleanup)

```
@std/
├── math      # Mathematical functions and constants
├── id        # ID generation (ULID, UUID, NanoID, CUID)
├── valid     # Validation predicates (IDs, credit cards, postal codes)
├── hash      # Hashing functions (MD5, SHA1, SHA256, SHA512)
└── mdDoc     # Markdown document parsing and querying
```

---

## Final `@basil/` Namespace (Post-cleanup)

```
@basil/
├── api       # HTTP API utilities (auth wrappers, error helpers, redirects)
├── auth      # Authentication context (session, auth, user)
├── http      # HTTP request context (params, request, response, route, method)
├── log       # Development logging (formerly @std/dev)
└── html      # Pre-built HTML components (formerly @std/html)
```

---

## ID Type Validation Coverage

### Final State:

| ID Type | Can Generate | DSL Schema | @std/valid | Status |
|---------|-------------|------------|------------|--------|
| ULID | ✅ `id.new()` | ✅ `ulid` | 🆕 `valid.ulid()` | ✅ Complete |
| UUID v4 | ✅ `id.uuid()` | ✅ `uuid` | ✅ `valid.uuid()` | ✅ Complete |
| UUID v7 | ✅ `id.uuidv7()` | ✅ `uuid` | ✅ `valid.uuid()` | ✅ Complete |
| NanoID | ✅ `id.nanoid()` | ❌ | 🆕 `valid.nanoid()` | ✅ Complete |
| CUID | ✅ `id.cuid()` | ❌ | 🆕 `valid.cuid()` | ✅ Complete |

### Validation Patterns:

| ID Type | Regex/Rule |
|---------|------------|
| ULID | `^[0-9A-HJKMNP-TV-Z]{26}$` (Crockford Base32) |
| UUID | `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$` |
| NanoID | Characters from `0-9A-Za-z_-`, configurable length (default 21) |
| CUID2 | `^c[0-9a-z]{24}$` (starts with `c`, 25 chars total, base36) |

---

## Implementation Checklist

### Phase 1: Deprecations

- [ ] Add deprecation error for `@std/schema` with migration guide to `@schema` syntax

### Phase 2: `@std/valid` Changes

**Remove (27 functions):**
- [ ] Type validators: `string`, `number`, `integer`, `boolean`, `array`, `dict`
- [ ] Constraint validators: `minLen`, `maxLen`, `length`, `min`, `max`, `between`, `positive`, `negative`
- [ ] Format validators covered by schema: `email`, `url`, `phone`
- [ ] String validators: `matches`, `alpha`, `alphanumeric`, `numeric`, `empty`
- [ ] Collection validators: `contains`, `oneOf`
- [ ] Date validators: `date`, `time`, `parseDate`

**Keep (4 functions):**
- [ ] `uuid`, `creditCard`, `luhn`, `postalCode`

**Add (3 functions):**
- [ ] `ulid` — ULID format validation
- [ ] `nanoid` — NanoID format validation with optional length
- [ ] `cuid` — CUID2 format validation

### Phase 3: New `@std/hash` Module

- [ ] Create `stdlib_hash.go`
- [ ] Implement `hash.md5(str)`
- [ ] Implement `hash.sha1(str)`
- [ ] Implement `hash.sha256(str)`
- [ ] Implement `hash.sha512(str)`
- [ ] Register module in `loadStdlibModule()`

### Phase 4: New String Methods

- [ ] `str.toBase64()` — Base64 encode
- [ ] `str.fromBase64()` — Base64 decode
- [ ] `str.toCamel()` — Convert to camelCase
- [ ] `str.toPascal()` — Convert to PascalCase
- [ ] `str.toSnake()` — Convert to snake_case
- [ ] `str.toKebab()` — Convert to kebab-case
- [ ] `str.truncate(len, suffix?)` — Truncate with suffix (default "...")

### Phase 5: Namespace Moves

- [ ] Move `@std/api` to `@basil/api` with deprecation alias
- [ ] Move `@std/dev` to `@basil/log` with deprecation alias
- [ ] Move `@std/html` to `@basil/html` with deprecation alias

### Phase 6: Documentation

- [ ] Update `docs/parsley/reference.md` to reflect changes
- [ ] Document namespace distinction (`@std/` vs `@basil/`)
- [ ] Add migration guide for deprecated/moved modules
- [ ] Document new `@std/hash` module
- [ ] Document new string methods

### Phase 7: Testing

- [ ] Add tests for new `valid.ulid()`, `valid.nanoid()`, `valid.cuid()` functions
- [ ] Add tests for `@std/hash` functions
- [ ] Add tests for new string methods
- [ ] Verify deprecation errors display correctly with helpful messages
- [ ] Verify deprecation aliases work for moved modules

---

## Risk Assessment

**Risk of shipping v1.0 without these changes:**

| Risk | Level | Impact |
|------|-------|--------|
| Maintaining two validation systems forever | High | Technical debt, user confusion |
| User confusion about validation approach | Medium | Support burden, inconsistent code |
| Namespace confusion (`@std/` vs `@basil/`) | Medium | Conceptual muddle |
| Missing ID validators | Low | Incomplete but fixable |
| Missing hash/base64/case conversion | Low | Users work around it |

**Risk of making these changes:**

| Risk | Level | Mitigation |
|------|-------|------------|
| Breaking changes | Low | Acceptable pre-v1.0 |
| Removed functions | Low | Clear alternatives documented |
| Scope creep | Low | Well-defined additions |

---

## Technical Implementation Notes

### Deprecation Error Pattern

For deprecated modules, return a helpful error on import:

```go
// In loadStdlibModule()
if name == "schema" {
    return newImportError("IMPORT-0006", map[string]any{
        "Module":      "schema",
        "Replacement": "Use @schema { ... } syntax instead. See docs/parsley/reference.md#schema-literals",
    })
}
```

### Deprecation Alias Pattern

For moved modules, support both old and new paths during transition:

```go
// In loadStdlibModule()
if name == "api" || name == "dev" || name == "html" {
    // Log deprecation warning
    fmt.Fprintf(os.Stderr, "Warning: @std/%s is deprecated, use @basil/%s instead\n", name, newName)
    // Still load the module (from basil namespace)
    return loadBasilModule(newName, env)
}
```

### `@std/hash` Implementation

```go
var hashModuleMeta = ModuleMeta{
    Description: "Cryptographic hash functions for checksums and non-security hashing",
    Exports: map[string]ExportMeta{
        "md5":    {Kind: "function", Arity: "1", Description: "MD5 hash (hex string)"},
        "sha1":   {Kind: "function", Arity: "1", Description: "SHA1 hash (hex string)"},
        "sha256": {Kind: "function", Arity: "1", Description: "SHA256 hash (hex string)"},
        "sha512": {Kind: "function", Arity: "1", Description: "SHA512 hash (hex string)"},
    },
}

func loadHashModule(env *Environment) Object {
    return &StdlibModuleDict{
        Meta: &hashModuleMeta,
        Exports: map[string]Object{
            "md5":    &Builtin{Fn: hashMD5},
            "sha1":   &Builtin{Fn: hashSHA1},
            "sha256": &Builtin{Fn: hashSHA256},
            "sha512": &Builtin{Fn: hashSHA512},
        },
    }
}
```

### Final `@std/valid` Implementation

```go
var validModuleMeta = ModuleMeta{
    Description: "Validation predicates for IDs, financial data, and locale-specific formats",
    Exports: map[string]ExportMeta{
        // ID validators
        "uuid":       {Kind: "function", Arity: "1", Description: "Check UUID v4/v7 format"},
        "ulid":       {Kind: "function", Arity: "1", Description: "Check ULID format"},
        "nanoid":     {Kind: "function", Arity: "1-2", Description: "Check NanoID format (length?)"},
        "cuid":       {Kind: "function", Arity: "1", Description: "Check CUID2 format"},
        // Financial validators
        "creditCard": {Kind: "function", Arity: "1", Description: "Check credit card number (Luhn)"},
        "luhn":       {Kind: "function", Arity: "1", Description: "Check Luhn algorithm"},
        // Locale-aware validators
        "postalCode": {Kind: "function", Arity: "2", Description: "Check postal code format (locale)"},
    },
}

// Regex patterns
var (
    uuidRegex   = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
    ulidRegex   = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
    nanoidRegex = regexp.MustCompile(`^[0-9A-Za-z_-]+$`)
    cuidRegex   = regexp.MustCompile(`^c[0-9a-z]{24}$`)
)
```

### String Method Implementations

```go
// In StringMethodRegistry

"toBase64": {
    Fn:          stringToBase64,
    Arity:       "0",
    Description: "Encode as Base64",
},
"fromBase64": {
    Fn:          stringFromBase64,
    Arity:       "0",
    Description: "Decode from Base64",
},
"toCamel": {
    Fn:          stringToCamel,
    Arity:       "0",
    Description: "Convert to camelCase",
},
"toPascal": {
    Fn:          stringToPascal,
    Arity:       "0",
    Description: "Convert to PascalCase",
},
"toSnake": {
    Fn:          stringToSnake,
    Arity:       "0",
    Description: "Convert to snake_case",
},
"toKebab": {
    Fn:          stringToKebab,
    Arity:       "0",
    Description: "Convert to kebab-case",
},
"truncate": {
    Fn:          stringTruncate,
    Arity:       "1-2",
    Description: "Truncate to length with suffix (default '...')",
},
```

---

## Estimated Effort

| Phase | Estimated Time | Complexity |
|-------|----------------|------------|
| Phase 1: Deprecations | 1 hour | Low |
| Phase 2: @std/valid changes | 2-3 hours | Medium |
| Phase 3: @std/hash module | 1-2 hours | Low |
| Phase 4: String methods | 3-4 hours | Medium |
| Phase 5: Namespace moves | 2 hours | Low |
| Phase 6: Documentation | 2-3 hours | Low |
| Phase 7: Testing | 3-4 hours | Medium |
| **Total** | **14-19 hours** | **Medium** |

---

## Deferred Items

The following were considered but deferred:

| Feature | Reason | When to Revisit |
|---------|--------|-----------------|
| Pluralization (`pluralize`, `singularize`) | English-specific, requires inflection database, complex edge cases | When addressing i18n/l10n as a whole |
| Additional locales for `postalCode` | Current support (US, GB, CA) covers common cases | Based on user demand |
| HMAC signing | Used internally but niche for user code | If users request it |

---

## References

- `pkg/parsley/evaluator/stdlib_valid.go` — Current @std/valid implementation
- `pkg/parsley/evaluator/stdlib_schema.go` — Current @std/schema implementation (to deprecate)
- `pkg/parsley/evaluator/stdlib_table.go` — Module loading logic
- `pkg/parsley/evaluator/stdlib_id.go` — ID generation (reference for validation patterns)
- `pkg/parsley/evaluator/stdlib_dsl_schema.go` — DSL schema validation patterns
- `pkg/parsley/evaluator/methods_string.go` — String method registry
- `docs/parsley/reference.md` — Language reference (needs updating)
- `work/BACKLOG.md` — Item #57 (rename @std/dev)