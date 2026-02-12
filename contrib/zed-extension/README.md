# Parsley Language Extension for Zed

Syntax highlighting and language support for [Parsley](https://github.com/sambeau/basil) in the Zed editor.

## Features

- **Syntax Highlighting**: Full syntax highlighting for Parsley source files
- **Bracket Matching**: Intelligent bracket pairing for arrays, dictionaries, functions, and tags
- **Code Outline**: Navigate your code structure with function and export definitions
- **Auto-Indentation**: Smart indentation for nested blocks and expressions
- **File Type Detection**: Automatic recognition of `.pars` and `.part` files

## Supported Syntax

- **Statements**: `let`, `export`, `return`, `check`, `for`, `if`, `try`
- **Expressions**: Arithmetic, comparison, logical, regex matching, ranges
- **Literals**: Numbers, strings (with interpolation), templates, regex, money, booleans
- **At-literals**: `@sqlite`, `@now`, `@std/...`, paths, URLs, durations, datetimes
- **Operators**: File I/O (`<==`, `==>`), database (`<=?=>`), Query DSL (`?->`, `|>`)
- **Tags**: JSX-like tags with attributes and embedded expressions
- **Functions**: With parameters, defaults, and rest parameters
- **Destructuring**: Array and dictionary patterns

## Installation

### From Zed Extensions (Recommended)

Once published, install directly from Zed:

1. Open Zed
2. Press `Cmd+Shift+P` (Mac) or `Ctrl+Shift+P` (Linux/Windows)
3. Type "extensions" and select "zed: extensions"
4. Search for "Parsley"
5. Click "Install"

### Development Installation

To test or develop this extension locally:

1. Clone this repository
2. Open Zed
3. Press `Cmd+Shift+P` and select "zed: install dev extension"
4. Navigate to this directory and select it
5. Open a `.pars` file to test

## Usage

Simply open any `.pars` or `.part` file in Zed. The extension will automatically:

- Apply syntax highlighting
- Enable bracket matching
- Populate the outline view with your functions and exports
- Provide smart indentation as you type

## Example

```parsley
// Define a function
greet = fn(name) {
  return "Hello, {name}!"
}

// Export a value
export message = greet("World")

// Work with data
users = [
  {name: "Alice", age: 30},
  {name: "Bob", age: 25}
]

// Use at-literals
db = @sqlite("./data.db")
now = @now
config = @std/env
```

## About Parsley

Parsley is a modern programming language designed for web development with the Basil framework. It features:

- Clean, expressive syntax
- First-class support for databases and file I/O
- JSX-like templating for HTML generation
- Built-in time and money types
- Powerful string interpolation

Learn more at [github.com/sambeau/basil](https://github.com/sambeau/basil)

## Development

This extension uses the [Tree-sitter grammar for Parsley](https://github.com/sambeau/basil/tree/main/contrib/tree-sitter-parsley).

### Building

No build step is required. Zed automatically compiles the Tree-sitter grammar when the extension is installed.

### Testing

1. Make changes to query files (`.scm`)
2. Reload the extension: `Cmd+Shift+P` → "zed: reload extensions"
3. Test with sample Parsley files in `test/`

### Contributing

Contributions are welcome! Please:

1. Fork this repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) for details

## Links

- [Parsley Language](https://github.com/sambeau/basil)
- [Tree-sitter Grammar](https://github.com/sambeau/basil/tree/main/contrib/tree-sitter-parsley)
- [Zed Editor](https://zed.dev)
- [Report Issues](https://github.com/sambeau/parsley-zed/issues)

## Credits

Developed by the Basil Contributors. Tree-sitter grammar and language design by the Parsley team.