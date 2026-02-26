## 1. Literals

Parsley supports a variety of literal types for representing data directly in code.

### 1.1 Numbers

#### Integers

```parsley
42
-15
0
```

#### Floats

```parsley
3.14159
-2.718
0.5
```

### 1.2 Strings

Parsley has three string types:

- **Double-quoted strings** (`"..."`) — Standard strings with escape sequences, no interpolation
- **Template strings** (`` `...` ``) — Support `{expression}` interpolation
- **Raw strings** (`'...'`) — No escape processing, useful for regex and JavaScript

```parsley
"Hello, World!"           // Double-quoted
`Hello, {name}!`          // Template with interpolation
'no\escapes\here'         // Raw string
```

### 1.3 Booleans and Null

```parsley
true
false
null
```

### 1.4 Arrays

```parsley
[1, 2, 3]
["a", "b", "c"]
[]                        // Empty array
```

### 1.5 Dictionaries

```parsley
{name: "Alice", age: 30}
{key: value}
{}                        // Empty dictionary
```

### 1.6 DateTime Literals

```parsley
@2024-12-25               // Date
@14:30                    // Time
@2024-12-25T14:30         // DateTime
@now                      // Current datetime
@today                    // Current date
```

### 1.7 Duration Literals

```parsley
@2h30m                    // 2 hours, 30 minutes
@1d                       // 1 day
@1y2mo                    // 1 year, 2 months
```

### 1.8 Money Literals

```parsley
$99.99                    // US Dollars
€50.00                    // Euros
£25.00                    // British Pounds
USD#100.00                // CODE# format
```

### 1.9 Unit Literals

```parsley
#5m                       // 5 meters
#10kg                     // 10 kilograms
#3/8in                    // Fractional inches
#72°F                     // Temperature
```

### 1.10 Path and URL Literals

```parsley
@./local/file.txt         // Relative path
@~/home/file.txt          // Home-relative path
@https://example.com/api  // URL
```
