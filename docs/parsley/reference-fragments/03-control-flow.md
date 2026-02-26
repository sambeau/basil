## 3. Control Flow

Parsley's control flow constructs are expression-based — they return values.

### 3.1 If Expression

```parsley
// Compact form (ternary-style)
let status = if (age >= 18) "adult" else "minor"

// Block form
let result = if (condition) {
    "value if true"
} else {
    "value if false"
}

// If-else-if chain
if (score >= 90) {
    "A"
} else if (score >= 80) {
    "B"
} else {
    "C"
}
```

### 3.2 For Expression

For loops return arrays — they work like `map` in functional languages.

```parsley
// Basic iteration
for (x in [1, 2, 3]) { x * 2 }  // [2, 4, 6]

// With index
for (x, i in ["a", "b", "c"]) { `{i}: {x}` }  // ["0: a", "1: b", "2: c"]

// Filter pattern (null values are omitted)
for (x in [1, 2, 3, 4]) {
    if (x % 2 == 0) { x }  // [2, 4]
}

// Over range
for (n in 1..5) { n }  // [1, 2, 3, 4, 5]

// Over dictionary
for (k, v in {a: 1, b: 2}) { `{k}={v}` }  // ["a=1", "b=2"]
```

### 3.3 Loop Control

```parsley
// stop - exit loop early, return accumulated results
for (x in 1..100) {
    if (x > 3) stop
    x
}  // [1, 2, 3]

// skip - skip current iteration
for (x in 1..5) {
    if (x == 3) skip
    x
}  // [1, 2, 4, 5]
```

### 3.4 Try Expression

```parsley
// Capture result and error
let {data, error} = try {
    riskyOperation()
}

if (error) {
    `Error: {error.message}`
} else {
    data
}
```

### 3.5 Check Guard

```parsley
// Early exit on condition failure
check condition else { "fallback value" }

// With error handling
check user != null else { fail("User not found") }
```

### 3.6 With Expression

Scoped field access for cleaner code:

```parsley
let point = {x: 10, y: 20}

with (point) {
    x + y  // Can access x and y directly
}  // 30
```
