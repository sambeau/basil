# tree-sitter-parsley

Tree-sitter grammar for the [Parsley](https://github.com/sambeau/basil) programming language.

## Features

- Full syntax highlighting for Parsley source files (`.pars`, `.part`)
- Support for all Parsley language features:
  - Statements: `let`, `export`, `import`, `return`, `check`, `for`, `if`/`else`, `try`
  - Expressions: arithmetic, comparison, logical, regex match, range, etc.
  - Literals: numbers, strings (with interpolation), templates, regex, money, booleans
  - At-literals: `@sqlite`, `@now`, `@std/...`, paths, URLs, durations, datetimes
  - Special operators: file I/O (`<==`, `==>`), database (`<=?=>`), Query DSL (`?->`, `|>`)
  - JSX-like tags with attributes, embedded expressions, and text content
  - Functions with parameters, defaults, and rest parameters
  - Array and dictionary destructuring

## Installation

### Zed

Zed automatically discovers grammars from the tree-sitter registry. Once this grammar is published, Parsley support should appear automatically.

### Neovim (nvim-treesitter)

Add to your nvim-treesitter configuration:

```lua
local parser_config = require("nvim-treesitter.parsers").get_parser_configs()
parser_config.parsley = {
  install_info = {
    url = "https://github.com/sambeau/tree-sitter-parsley",
    files = {"src/parser.c"},
  },
  filetype = "pars",
}
```

Then run `:TSInstall parsley`.

### Helix

Add to your `languages.toml`:

```toml
[[language]]
name = "parsley"
scope = "source.parsley"
file-types = ["pars", "part"]
roots = []
comment-token = "//"
indent = { tab-width = 2, unit = "  " }

[[grammar]]
name = "parsley"
source = { git = "https://github.com/sambeau/tree-sitter-parsley", rev = "main" }
```

Then run `hx --grammar fetch` and `hx --grammar build`.

## Development

### Prerequisites

- Node.js (for tree-sitter-cli)
- tree-sitter-cli: `npm install -g tree-sitter-cli`

### Building

```bash
# Generate the parser
tree-sitter generate

# Run tests
tree-sitter test

# Parse a sample file
tree-sitter parse path/to/file.pars

# Test highlighting
tree-sitter highlight path/to/file.pars
```

### Project Structure

```
tree-sitter-parsley/
├── grammar.js              # Main grammar definition
├── queries/
│   └── highlights.scm      # Syntax highlighting queries
├── test/
│   └── corpus/             # Test cases
│       ├── literals.txt
│       ├── statements.txt
│       ├── strings.txt
│       ├── expressions.txt
│       └── tags.txt
├── bindings/
│   ├── node/               # Node.js bindings
│   └── rust/               # Rust bindings
├── src/                    # Generated parser (after tree-sitter generate)
├── package.json
├── Cargo.toml
└── binding.gyp
```

### Testing

Test files in `test/corpus/` follow the tree-sitter corpus format:

```
================================================================================
Test name
================================================================================

source code here

--------------------------------------------------------------------------------

(expected_tree)
```

Run all tests with:

```bash
tree-sitter test
```

## License

MIT

## Related

- [Basil](https://github.com/sambeau/basil) - The Parsley web server
- [VS Code Extension](https://marketplace.visualstudio.com/items?itemName=sambeau.parsley-language) - VS Code support for Parsley
- [highlight.js grammar](https://github.com/sambeau/basil/tree/main/contrib/highlightjs) - highlight.js support for Parsley