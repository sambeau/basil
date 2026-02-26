## 9. Comments

Parsley supports single-line comments only.

### Single-Line Comments

Use `//` for comments:

```parsley
// This is a comment
let x = 42  // Inline comment
```

### No Multi-Line Comments

Parsley does **not** support multi-line `/* ... */` comments. This syntax would conflict with regex literals.

```parsley
// ❌ This will NOT work as a comment
/* This is not
   a comment */

// ✅ Use multiple single-line comments instead
// Line one
// Line two
// Line three
```

### No Hash Comments

Python/Shell-style `#` comments are not supported (except within unit literals like `#5m`).

```parsley
// ❌ This will cause an error
# Not a comment

// ✅ Use // instead
// This is correct
```
