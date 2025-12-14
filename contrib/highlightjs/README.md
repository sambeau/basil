# Parsley Syntax Highlighting for Highlight.js

This directory contains a [highlight.js](https://highlightjs.org/) language definition for the Parsley programming language.

## Features

- **Keywords**: `fn`, `let`, `if`, `else`, `for`, `in`, `return`, `export`, `import`, `try`, `and`, `or`, `not`
- **At-literals**: DateTime (`@2024-12-25T14:30:00Z`, `@now`), Duration (`@2h30m`), Paths (`@./config`, `@/usr/local`), URLs (`@https://example.com`), Stdlib imports (`@std/table`)
- **Money literals**: `$12.34`, `£99.99`, `EUR#50.00`
- **Template strings**: `` `Hello ${name}` ``
- **Regex literals**: `/pattern/flags`
- **JSX-like tags**: `<Component prop={value}>...</Component>`
- **Special operators**: File I/O (`<==`, `==>`), Database (`<=?=>`, `<=??=>`), etc.

## Installation

### Option 1: CDN (Not Yet Available)

Once published to CDN, you can use:

```html
<link rel="stylesheet" href="https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@latest/build/styles/default.min.css">
<script src="https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@latest/build/highlight.min.js"></script>
<script src="https://cdn.jsdelivr.net/gh/sambeau/basil@latest/contrib/highlightjs/parsley.min.js"></script>
```

### Option 2: Local Installation

1. Copy `parsley.js` to your project
2. Register the language with highlight.js:

```javascript
import hljs from 'highlight.js';
import parsley from './path/to/parsley.js';

hljs.registerLanguage('parsley', parsley);
```

### Option 3: Build from Source

If you're building highlight.js from source:

1. Copy `parsley.js` to `src/languages/` in your highlight.js directory
2. Rebuild highlight.js with Parsley included:

```bash
node tools/build.js -t cdn parsley
```

## Usage

### In HTML

```html
<pre><code class="language-parsley">
let greeting = "Hello, World!"
print(greeting)
</code></pre>

<script>
hljs.highlightAll();
</script>
```

### Programmatic

```javascript
import hljs from 'highlight.js';
import parsley from './parsley.js';

hljs.registerLanguage('parsley', parsley);

const code = `
let {table} = import @std/table
let data = table([{name: "Alice", age: 30}])
print(data.toHTML())
`;

const highlighted = hljs.highlight(code, { language: 'parsley' }).value;
```

## Examples

### Basic Syntax

```parsley
// Variables and functions
let name = "Alice"
let greet = fn(person) {
  "Hello, " + person
}

print(greet(name))
```

### At-Literals

```parsley
// DateTime
let now = @now
let meeting = @2024-12-25T14:30:00Z

// Duration
let timeout = @5m30s
let age = @30y

// Paths and URLs
let config = @./config.yaml
let api = @https://api.example.com/data

// Stdlib imports
let {table} = import @std/table
```

### Money

```parsley
let price = $19.99
let euro = EUR#50.00
let total = price + money(5.00, "USD")
```

### Templates

```parsley
let name = "World"
let message = `Hello, ${name}!`
```

### JSX-like Tags

```parsley
<Page title="Home">
  <h1>{title}</h1>
  <p>Welcome to {name}</p>
</Page>
```

### File I/O

```parsley
// Read from file
let content <== text(@./data.txt)

// Write to file
"Hello, World!" ==> text(@./output.txt)

// Append to file
"Additional line" ==>> text(@./log.txt)
```

### Database

```parsley
// Query one row
let user <=?=> {sql: "SELECT * FROM users WHERE id = ?", params: [1]}

// Query many rows
let users <=??=> {sql: "SELECT * FROM users"}

// Execute statement
<=!=> {sql: "INSERT INTO users (name) VALUES (?)", params: ["Alice"]}
```

## Language Reference

See the [Parsley documentation](../../docs/parsley/) for complete language reference.

## Contributing

To improve the syntax highlighting:

1. Edit `parsley.js`
2. Test with various Parsley code samples
3. Submit a pull request

## License

MIT License - see [LICENSE](../../LICENSE) file for details
