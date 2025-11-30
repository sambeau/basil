# Parsley Language Reference for AI Agents

When writing Parsley code in Basil handlers, tests, or examples, consult this reference.

Full documentation: https://github.com/sambeau/parsley/blob/main/docs/CHEATSHEET.md

## 🚨 Critical: HTML/XML Tag Rules

**Tags don't need quotes** - they are first-class syntax:
```parsley
// ✅ CORRECT - Tags are native syntax
<p>Hello</p>
<div class="card">Content</div>

// ❌ WRONG - Don't quote tags
"<p>Hello</p>"
```

**Singleton tags MUST be self-closing**:
```parsley
// ✅ CORRECT
<br/>
<hr/>
<img src="photo.jpg"/>
<link rel="stylesheet" href="/style.css"/>
<input type="text" name="email"/>

// ❌ WRONG - Will error
<br>
<hr>
<img src="photo.jpg">
<link rel="stylesheet" href="/style.css">
```

## 🚨 Major Gotchas

### Output is `log()`, not `print()`
```parsley
// ✅ CORRECT
log("hello")
logLine("debug info")  // Includes line number

// ❌ WRONG
print("hello")
console.log("hello")
```

### Comments are `//`, not `#`
```parsley
// ✅ CORRECT
// This is a comment
/* Multi-line comment */

// ❌ WRONG - Will error
# This is not a comment
```

### String interpolation uses `{var}`, not `${var}`
```parsley
// ✅ CORRECT
let name = "Alice"
"Hello, {name}!"

// ❌ WRONG
"Hello, ${name}!"
```

### `for` returns an array (like map)
```parsley
// For is an expression that returns an array
let doubled = for (n in [1,2,3]) { n * 2 }  // [2, 4, 6]

// Filter pattern - if returns null, item is omitted
let evens = for (n in [1,2,3,4]) {
    if (n % 2 == 0) { n }  // [2, 4]
}
```

### `if` is an expression (returns a value)
```parsley
let status = if (age >= 18) "adult" else "minor"
```

### Path literals use `@`
```parsley
// ✅ CORRECT
let path = @./config.json
let url = @https://example.com

// ❌ WRONG - Just a string, not a path
let path = "./config.json"
```

## Component Pattern

```parsley
// Define a component (function returning tags)
let Page = fn({title, contents}) {
    <html>
    <head>
        <title>{title}</title>
    </head>
    <body>
        {contents}
    </body>
    </html>
}

// Use it (note: self-closing OR with children)
<Page title="Home">
    <h1>Welcome!</h1>
</Page>
```

## Module Pattern

```parsley
// In Page.pars - export the component
export Page = fn({title, contents}) {
    <html>
    <head><title>{title}</title></head>
    <body>{contents}</body>
    </html>
}

// In index.pars - import and use
{Page} = import(@./Page.pars)

<Page title="Home">
    <h1>Welcome!</h1>
</Page>
```

## Response Types in Basil Handlers

**HTML** (auto-detected by leading `<`):
```parsley
<html><body>Hello!</body></html>
```

**JSON** (return a dictionary):
```parsley
{
    message: "Hello!",
    timestamp: now()
}
```

**Custom response** (full control):
```parsley
{
    status: 201,
    headers: {"X-Custom": "value"},
    body: "Created!"
}
```

## Quick Syntax Reference

| Feature | Parsley |
|---------|---------|
| Variable | `let x = 5` |
| Function | `fn(x) { x * 2 }` |
| If expr | `if (x) "yes" else "no"` |
| For loop | `for (x in arr) { x * 2 }` |
| Dictionary | `{x: 1, y: 2}` |
| String interp | `"Hello {name}"` |
| Null | `null` |
| Comment | `// comment` |
