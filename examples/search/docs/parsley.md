---
title: Parsley Language Reference
tags: [language, reference, syntax]
date: 2026-01-09
---

# Parsley Language Reference

Parsley is a dynamic programming language designed for web development.

## Variables

Variables are dynamically typed:

```parsley
name = "Alice"
age = 30
active = true
```

## Functions

Define functions with the `fn` keyword:

```parsley
greet = fn(name) {
  "Hello, {name}!"
}
```

## Control Flow

### If Statements

```parsley
@if user.logged_in {
  <p>Welcome back!</p>
} @else {
  <p>Please log in</p>
}
```

### Loops

```parsley
@for item in items {
  <li>{item.name}</li>
}
```

## Built-ins

Basil provides powerful built-ins:

- `@DB` - Database connections
- `@SEARCH` - Full-text search
- `@FILE` - File operations
