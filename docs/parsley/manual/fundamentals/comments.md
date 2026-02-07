---
id: man-pars-comments
title: Comments
system: parsley
type: fundamental
name: comments
created: 2026-02-05
version: 0.2.0
author: Basil Team
keywords:
  - comment
  - annotation
  - documentation
  - code comment
---

# Comments

Comments in Parsley are annotations in your source code that are ignored by the interpreter. They exist solely for human readers — to explain intent, document behaviour, or temporarily disable code. Parsley supports **single-line comments only**, using the `//` prefix.

```parsley
// This is a comment
```

## Syntax

### Single-line comments

A comment begins with `//` and continues to the end of the line. Everything after `//` is ignored.

```parsley
// Calculate the total price including tax
let total = price * 1.2
```

### Inline comments

Comments can appear at the end of a line, after the code:

```parsley
let rate = 0.05  // 5% interest rate
let years = 10   // Investment period
```

### Multiple comment lines

To write multi-line commentary, use `//` on each line:

```parsley
// This function calculates compound interest.
// It takes a principal amount, an annual rate,
// and the number of years.
let compound = fn(principal, rate, years) {
    principal * (1 + rate)  // simplified formula
}
```

## Comments in Different Contexts

Comments work anywhere a line break is valid — at the top of a file, between statements, inside blocks, and alongside tag content:

```parsley
// Top-level comment
let name = "Basil"

let page = fn() {
    // Inside a function body
    <div>
        // Between tags
        <h1>name</h1>
    </div>
}
```

## No Multi-line Comments

Parsley does **not** support block comments (`/* ... */`). If you're coming from C, Java, JavaScript, or Go, this is a deliberate simplification. Use multiple `//` lines instead.

```parsley
// ✅ Correct — multiple single-line comments
// This is a longer explanation
// that spans several lines.

// ❌ Wrong — block comments are not supported
// /* This will cause a syntax error */
```

## Key Differences from Other Languages

- **No block comments:** There is no `/* ... */` syntax — use multiple `//` lines
- **No doc-comments:** There is no `///` or `/** */` convention — just use `//`
- **No `#` comments:** Unlike Python, Ruby, or shell scripts, `#` is not a comment character and will cause a parse error

## See Also

- [Variables](variables.md) — declaring and binding values
- [Functions](functions.md) — defining functions