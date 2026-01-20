# Parsley Literal Notation (PLN)

## Overview

PLN is a data serialization format for Parsley. It uses a safe subset of Parsley syntax to represent data values—including records with their schemas and validation errors—without allowing code execution.

Think of PLN as "JSON for Parsley"—but with types, dates, comments, and schema-aware records.

## Motivation

### The Problem

Passing complex data between components is awkward:

```parsley
// You have a validated record with errors
let person = Person({name: "", email: "bad"}).validate()

// You want to pass it to a Part... but how?
<Part src={@~/parts/editor.part} person={???}/>
```

Currently, you'd have to:
1. Convert to JSON: `person.data().toJSON()`
2. Lose the schema binding
3. Lose the validation errors
4. Manually reconstruct in the Part

### The Solution

With PLN, complex objects serialize naturally:

```parsley
// Just pass it
<Part src={@~/parts/editor.part} person={person}/>

// In the Part, it arrives intact
let {person} = @props  // Still a Person record, errors preserved
```

## What PLN Looks Like

PLN uses familiar Parsley syntax, restricted to pure values:

### Primitives

```parsley
42                          // Integer
3.14                        // Float
"Hello, world"              // String
true                        // Boolean
false
null
```

### Collections

```parsley
// Arrays
[1, 2, 3]
["apple", "banana", "cherry"]

// Dictionaries
{name: "Alice", age: 30}
{
    title: "My Post",
    tags: ["parsley", "web"],
    published: true
}
```

### Records (Schema-Bound Data)

This is where PLN shines—it preserves the schema association:

```parsley
@Person({
    name: "Alice Smith",
    email: "alice@example.com",
    age: 30
})
```

When deserialized, this becomes a proper `Person` record, not just a dictionary.

### Records with Validation Errors

PLN can preserve validation state for round-tripping form data:

```parsley
@Person({
    name: "",
    email: "not-an-email"
}) @errors {
    name: "Name is required",
    email: "Invalid email format"
}
```

### Dates and Times

Native datetime support (no more ISO strings that might be dates or might be text):

```parsley
@2024-01-20                     // Date
@2024-01-20T10:30:00Z           // DateTime (UTC)
@2024-01-20T10:30:00+05:30      // DateTime (with timezone)
@10:30:00                       // Time
```

### Paths and URLs

```parsley
@/path/to/file.txt              // File path
@./relative/path.txt            // Relative path
@https://example.com/api        // URL
```

### Comments

Unlike JSON, PLN supports comments:

```parsley
// Customer record for order #12345
@Customer({
    name: "Alice Smith",      // Primary account holder
    email: "alice@example.com",
    
    // Shipping address
    address: @Address({
        street: "123 Main St",
        city: "Springfield",
        zip: "12345"          // Validated against USPS
    })
})
```

## What PLN Cannot Express

PLN is deliberately limited to **values**. These are not valid PLN:

```parsley
// ❌ No expressions
1 + 1

// ❌ No function calls
Person({name: "Alice"})      // Note: this is a call
@Person({name: "Alice"})     // ✓ This is PLN (@ prefix)

// ❌ No variables
{name: userName}

// ❌ No functions
fn(x) { x * 2 }

// ❌ No control flow
if (x) { "yes" } else { "no" }
```

This is intentional. PLN is data, not code.

## The API

### Serializing

```parsley
let person = Person({name: "Alice", email: "alice@example.com"})

// Serialize to PLN string
let pln = serialize(person)
// Result: '@Person({name: "Alice", email: "alice@example.com"})'

// Serialize with validation errors
let invalid = Person({name: "", email: "bad"}).validate()
let pln = serialize(invalid)
// Result: '@Person({name: "", email: "bad"}) @errors {name: "Name is required", ...}'
```

### Deserializing

```parsley
// Deserialize back to a Record
let person = deserialize('@Person({name: "Alice"})')
// Returns: Person record, properly bound to schema

// Schema must be available in scope, or:
let data = deserialize('@UnknownSchema({x: 1})')
// Returns: Dictionary with {x: 1, __schema: "UnknownSchema"}
// Warning in dev mode: "Unknown schema 'UnknownSchema'"
```

### What Can Be Serialized

| Type | Serializable | PLN Representation |
|------|--------------|-------------------|
| Integer | ✓ | `42` |
| Float | ✓ | `3.14` |
| String | ✓ | `"hello"` |
| Boolean | ✓ | `true`, `false` |
| Null | ✓ | `null` |
| Array | ✓ | `[1, 2, 3]` |
| Dictionary | ✓ | `{a: 1, b: 2}` |
| Record | ✓ | `@Schema({...})` |
| DateTime | ✓ | `@2024-01-20T10:30:00Z` |
| Path | ✓ | `@/path/to/file` |
| URL | ✓ | `@https://example.com` |
| Function | ✗ | Cannot serialize code |
| DB Connection | ✗ | Cannot serialize handles |
| File Handle | ✗ | Cannot serialize handles |

Attempting to serialize a non-serializable value produces an error:

```parsley
serialize({callback: fn(x) { x }})
// Error: Cannot serialize function
```

## Automatic Serialization in Parts

When passing props to Parts, serialization happens automatically:

```parsley
// In your page
let person = Person({name: "Alice"}).validate()
<Part src={@~/parts/editor.part} person={person} count={42}/>

// Basil automatically:
// - Detects 'person' is a Record → serializes to PLN
// - Detects 'count' is an Integer → passes as-is
// - Signs serialized data with HMAC for security

// In the Part
let {person, count} = @props
// 'person' is a Person record (deserialized)
// 'count' is 42 (passed directly)
```

## Security

### HMAC Signing

When PLN is transmitted (e.g., in Part props), it's signed:

```
SIGNATURE:BASE64_ENCODED_PLN
```

- Prevents tampering in transit
- Server validates signature before deserializing
- Uses application's secret key

### No Code Execution

PLN's restricted grammar means:
- No function calls can be injected
- No variable access (can't leak `admin_password`)
- No expressions (can't compute `1+1` let alone `delete_all()`)
- Parsing uses a dedicated PLN parser, not `eval`

### Schema Validation

On deserialization, if a schema is specified:
- Data is validated against the schema
- Invalid data can be rejected or flagged
- Provides defense-in-depth beyond HMAC

## Use Cases

### 1. Part Communication

Pass rich data between page and Parts:

```parsley
// Page detects validation error
let result = person.validate()
<Part src={@~/parts/form.part} person={result}/>

// Part receives record with errors intact
let {person} = @props
if (not person.isValid()) {
    // Show errors next to fields
}
```

### 2. Data Files

Store typed data in files:

```parsley
// config.pln
{
    // Application settings
    appName: "My App",
    maxUsers: 100,
    
    // Feature flags
    features: {
        darkMode: true,
        betaFeatures: false
    },
    
    // Launch date
    launchDate: @2024-06-01
}
```

### 3. Test Fixtures

Create readable test data:

```parsley
// fixtures/users.pln
[
    // Admin user for permission tests
    @User({
        id: 1,
        name: "Admin",
        role: "admin"
    }),
    
    // Regular user
    @User({
        id: 2,
        name: "Alice",
        role: "user"
    }),
    
    // User with validation errors (for error display tests)
    @User({
        id: 3,
        name: "",
        email: "invalid"
    }) @errors {
        name: "Required",
        email: "Invalid format"
    }
]
```

### 4. API Responses

Return structured, typed data:

```parsley
// Response from /api/user/123
@User({
    id: 123,
    name: "Alice",
    createdAt: @2024-01-15T09:30:00Z,
    profile: @Profile({
        bio: "Software developer",
        links: ["https://github.com/alice"]
    })
})
```

## Comparison with JSON

| Feature | JSON | PLN |
|---------|------|-----|
| Primitives | ✓ | ✓ |
| Arrays | ✓ | ✓ |
| Objects/Dicts | ✓ | ✓ |
| Comments | ✗ | ✓ |
| Trailing commas | ✗ | ✓ |
| Unquoted keys | ✗ | ✓ |
| Native dates | ✗ | ✓ |
| Native paths/URLs | ✗ | ✓ |
| Schema/type tags | ✗ | ✓ |
| Validation errors | ✗ | ✓ |
| Human readable | Mostly | ✓ |
| Widely supported | ✓ | Parsley only |

PLN is not a JSON replacement for interchange with other systems. It's a Parsley-native format for Parsley-to-Parsley communication.

## Grammar Summary

```
value       = primitive | array | dict | record | datetime | path
primitive   = INTEGER | FLOAT | STRING | BOOLEAN | NULL
array       = '[' (value (',' value)* ','?)? ']'
dict        = '{' (pair (',' pair)* ','?)? '}'
pair        = (IDENT | STRING) ':' value
record      = '@' IDENT '(' dict ')' errors?
errors      = '@errors' dict
datetime    = '@' ISO_DATE_TIME
path        = '@' PATH_LITERAL
comment     = '//' .* NEWLINE
```

## File Extension

PLN data files use the `.pln` extension:

```
config.pln
fixtures/users.pln
data/products.pln
```

## Summary

PLN gives Parsley a native serialization format that:

- **Preserves types** — Records stay records, dates stay dates
- **Preserves schemas** — `@Person({...})` deserializes to a Person record
- **Preserves errors** — Validation state survives round-trips
- **Stays readable** — Comments, clean syntax, familiar to Parsley users
- **Stays safe** — No code execution, HMAC signed in transit
- **Integrates seamlessly** — Auto-serialization in Parts

It's the missing piece for rich component communication in Basil applications.
