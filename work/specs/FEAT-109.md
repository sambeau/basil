---
id: FEAT-109
title: "Tree-sitter Grammar and Registry Submissions"
status: implemented
priority: high
created: 2025-01-15
author: "@human"
blocking: true
---

# FEAT-109: Tree-sitter Grammar and Registry Submissions

## Summary
Create a Tree-sitter grammar for Parsley to enable syntax highlighting in modern editors (Zed, Nova, Neovim, Helix) and on GitHub. Submit the grammar to the tree-sitter registry and GitHub linguist for `.pars` file recognition.

## User Story
As a Parsley developer using a modern editor like Zed, Neovim, or Helix, I want my code to be syntax highlighted so that I can read and write Parsley code effectively.

As a potential Parsley user browsing code on GitHub, I want `.pars` files to be recognized and syntax highlighted so that I can evaluate the language by reading real code.

## Acceptance Criteria

> **Note:** See [FEAT-109-ZED-INVESTIGATION.md](FEAT-109-ZED-INVESTIGATION.md) for detailed analysis of creating a Zed Editor extension (Feb 2025).

### Tree-sitter Grammar
- [x] Grammar covers all Parsley syntax (keywords, literals, operators, strings, comments, tags)
- [x] Includes `highlights.scm` query file for syntax highlighting
- [x] Passes all corpus tests (129/129)
- [x] Works in Zed editor (extension implemented in contrib/zed-extension/)
- [ ] Works in Neovim with tree-sitter plugin
- [ ] Works in Helix editor

### Registry Submissions
- [x] Grammar published as standalone `tree-sitter-parsley` repository
- [ ] Submitted to tree-sitter grammar registry
- [x] Submitted to GitHub linguist for `.pars` file recognition (PR awaiting review)
- [ ] `.pars` files highlighted on github.com (pending linguist merge)

## Design Decisions

- **Standalone repository**: Tree-sitter grammars are conventionally published as separate repositories (`tree-sitter-{language}`). Initial development will be in `contrib/tree-sitter-parsley/`, then published separately.

- **Focus on highlighting first**: Tree-sitter can power more than highlighting (code navigation, refactoring), but the initial grammar focuses on syntax highlighting. Advanced features can be added later.

- **External scanner for edge cases**: Some Parsley constructs (like JSX-style tags with embedded expressions) may require an external scanner written in C.

- **Test-driven development**: Build the grammar incrementally using tree-sitter's corpus test format to ensure correctness.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Affected Components
| Component | Location | Notes |
|-----------|----------|-------|
| Grammar source | `contrib/tree-sitter-parsley/` | Initial development |
| Standalone repo | `github.com/sambeau/tree-sitter-parsley` | For registry submission |
| GitHub linguist | External PR | For `.pars` recognition |

### Dependencies
- Depends on: FEAT-108 (grammar updates inform what syntax to support)
- Blocks: None (but should be done before 1.0 Alpha)

### Effort Estimate
- Grammar development: 2-4 days
- Testing and refinement: 1-2 days
- Registry submissions: 1-2 hours (but may take time to be accepted)

---

## Implementation Plan

### Phase 1: Project Setup

**Directory structure:**
```
contrib/tree-sitter-parsley/
├── grammar.js           # Main grammar definition
├── src/
│   ├── parser.c         # Generated parser
│   └── scanner.c        # External scanner (if needed)
├── queries/
│   ├── highlights.scm   # Syntax highlighting
│   ├── injections.scm   # Language injections (optional)
│   └── locals.scm       # Local variable scoping (optional)
├── bindings/
│   ├── node/            # Node.js bindings
│   │   ├── binding.cc
│   │   └── index.js
│   └── rust/            # Rust bindings
│       ├── lib.rs
│       └── build.rs
├── test/
│   └── corpus/          # Test cases
│       ├── literals.txt
│       ├── expressions.txt
│       ├── statements.txt
│       └── tags.txt
├── package.json
├── binding.gyp
├── Cargo.toml
└── README.md
```

### Phase 2: Core Grammar

**grammar.js skeleton:**
```javascript
module.exports = grammar({
  name: 'parsley',

  extras: $ => [/\s/, $.comment],

  conflicts: $ => [
    // Handle ambiguities
  ],

  rules: {
    source_file: $ => repeat($._statement),

    // === STATEMENTS ===
    _statement: $ => choice(
      $.let_statement,
      $.function_definition,
      $.export_statement,
      $.import_statement,
      $.for_statement,
      $.if_statement,
      $.try_statement,
      $.return_statement,
      $.check_statement,
      $.expression_statement,
    ),

    let_statement: $ => seq('let', $.identifier, '=', $._expression),
    
    function_definition: $ => seq(
      $.identifier, '=', 'fn', $.parameter_list, $._expression
    ),

    export_statement: $ => seq(
      'export', optional('computed'), $.identifier, '=', $._expression
    ),

    import_statement: $ => seq('import', $._expression, optional(seq('as', $.identifier))),

    // === EXPRESSIONS ===
    _expression: $ => choice(
      $._literal,
      $.identifier,
      $.binary_expression,
      $.unary_expression,
      $.call_expression,
      $.index_expression,
      $.member_expression,
      $.array,
      $.dictionary,
      $.function_expression,
      $.if_expression,
      $.for_expression,
      $.parenthesized_expression,
      $.tag_expression,
    ),

    // === LITERALS ===
    _literal: $ => choice(
      $.number,
      $.string,
      $.template_string,
      $.raw_string,
      $.regex,
      $.boolean,
      $.null,
      $.at_literal,
      $.money,
    ),

    at_literal: $ => choice(
      $.datetime,
      $.duration,
      $.path,
      $.url,
      $.connection,
      $.schema,
      $.table,
      $.query,
      $.context,
      $.stdlib,
    ),

    // Connection literals: @sqlite, @postgres, @mysql, @sftp, @shell
    connection: $ => /@(sqlite|postgres|mysql|sftp|shell|DB)\b/,

    // Schema/table/query DSL
    schema: $ => /@schema\b/,
    table: $ => /@table\b/,
    query: $ => /@(query|insert|update|delete|transaction)\b/,

    // Context literals: @env, @args, @params, @SEARCH
    context: $ => /@(env|args|params|SEARCH)\b/,

    // Standard library: @std/module
    stdlib: $ => /@std\/[a-zA-Z]+/,

    // Time literals
    datetime: $ => /@\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}(:\d{2})?)?/,
    duration: $ => /@-?\d+[yMwdhms]([0-9yMwdhms]|mo)*/,

    // Path and URL literals
    path: $ => /@(\.\.?\/|\/|~\/)[^\s<>"{}|\\^`\[\]]*/,
    url: $ => /@https?:\/\/[^\s<>"{}|\\^`\[\]]*/,

    // Money: $100, £50, EUR#100
    money: $ => /([$£€¥]|[A-Z]{3}#)\d+(\.\d{1,2})?/,

    // === OPERATORS ===
    binary_expression: $ => choice(
      // Arithmetic
      prec.left(1, seq($._expression, choice('+', '-'), $._expression)),
      prec.left(2, seq($._expression, choice('*', '/', '%'), $._expression)),
      // Comparison
      prec.left(0, seq($._expression, choice('==', '!=', '<', '>', '<=', '>='), $._expression)),
      // Logical
      prec.left(-1, seq($._expression, choice('and', 'or', '&&', '||'), $._expression)),
      // Regex match
      prec.left(0, seq($._expression, choice('~', '!~'), $._expression)),
      // Concatenation
      prec.left(1, seq($._expression, '++', $._expression)),
      // Nullish coalescing
      prec.right(-2, seq($._expression, '??', $._expression)),
      // Range
      prec.left(0, seq($._expression, '..', $._expression)),
      // I/O operators
      prec.left(0, seq($._expression, choice('<==', '==>', '==>>', '<=?=>', '<=??=>', '<=!=>', '<=#=>'), $._expression)),
      // Query DSL operators
      prec.left(0, seq($._expression, choice('|>', '|<', '?->', '??->', '.->', '<-'), $._expression)),
    ),

    // === TAGS (JSX-like) ===
    tag_expression: $ => choice(
      $.self_closing_tag,
      seq($.open_tag, repeat($._tag_child), $.close_tag),
    ),

    open_tag: $ => seq('<', $.tag_name, repeat($.attribute), '>'),
    close_tag: $ => seq('</', $.tag_name, '>'),
    self_closing_tag: $ => seq('<', $.tag_name, repeat($.attribute), '/>'),

    tag_name: $ => /[a-zA-Z][a-zA-Z0-9-]*/,
    attribute: $ => seq($.attribute_name, optional(seq('=', $._attribute_value))),
    attribute_name: $ => /[a-zA-Z][a-zA-Z0-9_-]*/,
    _attribute_value: $ => choice($.string, $.embedded_expression),

    _tag_child: $ => choice(
      $.tag_expression,
      $.embedded_expression,
      $.text,
    ),

    embedded_expression: $ => seq('{', $._expression, '}'),

    // === BASIC TOKENS ===
    identifier: $ => /[a-zA-Z_][a-zA-Z0-9_]*/,
    number: $ => /\d+(\.\d+)?/,
    boolean: $ => choice('true', 'false'),
    null: $ => 'null',

    string: $ => seq('"', repeat(choice(/[^"\\{}]+/, $.escape, $.interpolation)), '"'),
    template_string: $ => seq('`', repeat(choice(/[^`\\{}]+/, $.escape, $.interpolation)), '`'),
    raw_string: $ => seq("'", repeat(choice(/[^'\\@]+/, $.escape, $.raw_interpolation)), "'"),

    interpolation: $ => seq('{', $._expression, '}'),
    raw_interpolation: $ => seq('@{', $._expression, '}'),
    escape: $ => /\\./,

    regex: $ => seq('/', /[^/\n]+/, '/', optional(/[gimsuvy]+/)),

    comment: $ => /\/\/.*/,
  }
});
```

### Phase 3: Highlighting Queries

**queries/highlights.scm:**
```scheme
; Keywords
["let" "fn" "function" "if" "else" "for" "in" "return" "export" "import" "try" "check" "stop" "skip" "as"] @keyword
["computed"] @keyword
["and" "or" "not"] @keyword.operator

; Literals
(number) @number
(money) @number
(string) @string
(template_string) @string
(raw_string) @string
(regex) @string.regexp
(boolean) @constant.builtin
(null) @constant.builtin
(escape) @string.escape

; At-literals
(datetime) @number
(duration) @number
(path) @string.special.path
(url) @string.special.url
(connection) @function.builtin
(schema) @type
(table) @type
(query) @function.builtin
(context) @variable.builtin
(stdlib) @module

; Operators
["+" "-" "*" "/" "%" "==" "!=" "<" ">" "<=" ">=" "~" "!~" "++" "??" ".." "=" ] @operator
["<==" "==>" "==>>" "<=?=>" "<=??=>" "<=!=>" "<=#=>"] @operator
["|>" "|<" "?->" "??->" ".->" "<-"] @operator
["&&" "||" "!"] @operator

; Punctuation
["(" ")" "[" "]" "{" "}"] @punctuation.bracket
["," ";" ":" "."] @punctuation.delimiter

; Tags
(tag_name) @tag
(attribute_name) @attribute

; Functions
(function_definition name: (identifier) @function)
(call_expression function: (identifier) @function.call)

; Comments
(comment) @comment

; Identifiers
(identifier) @variable
```

### Phase 4: Test Corpus

**test/corpus/literals.txt:**
```
================================================================================
At-literals: connections
================================================================================

@sqlite("test.db")
@postgres("postgres://localhost/db")
@mysql("user:pass@tcp(localhost)/db")

--------------------------------------------------------------------------------

(source_file
  (expression_statement (call_expression (connection) (arguments (string))))
  (expression_statement (call_expression (connection) (arguments (string))))
  (expression_statement (call_expression (connection) (arguments (string)))))

================================================================================
At-literals: datetime and duration
================================================================================

@2024-01-15
@2024-01-15T10:30:00
@2h30m
@-7d

--------------------------------------------------------------------------------

(source_file
  (expression_statement (datetime))
  (expression_statement (datetime))
  (expression_statement (duration))
  (expression_statement (duration)))

================================================================================
Query DSL literals
================================================================================

@schema
@table
@query
@insert
@update
@delete
@transaction

--------------------------------------------------------------------------------

(source_file
  (expression_statement (schema))
  (expression_statement (table))
  (expression_statement (query))
  (expression_statement (query))
  (expression_statement (query))
  (expression_statement (query))
  (expression_statement (query)))
```

### Phase 5: Registry Submissions

**Tree-sitter Registry:**
1. Create standalone repository `github.com/sambeau/tree-sitter-parsley`
2. Ensure CI passes (grammar generation, tests, bindings)
3. Add to tree-sitter wiki: https://github.com/tree-sitter/tree-sitter/wiki/List-of-parsers

**GitHub Linguist:**
1. Fork https://github.com/github/linguist
2. Add entry to `lib/linguist/languages.yml`:
```yaml
Parsley:
  type: programming
  color: "#4B8BBE"
  extensions:
    - ".pars"
  tm_scope: source.parsley
  ace_mode: text
  language_id: XXXXXX  # Assigned by linguist maintainers
```
3. Add sample files to `samples/Parsley/`
4. Reference tree-sitter grammar in `grammars.yml`
5. Submit PR

---

## Test Plan

| Test | Expected |
|------|----------|
| `tree-sitter generate` | No errors |
| `tree-sitter test` | All corpus tests pass |
| `tree-sitter parse sample.pars` | Valid AST, no ERROR nodes |
| `tree-sitter highlight sample.pars` | Correct highlighting output |
| Open `.pars` in Zed | Syntax highlighted |
| Open `.pars` in Neovim (tree-sitter) | Syntax highlighted |
| Open `.pars` in Helix | Syntax highlighted |
| View `.pars` on GitHub (after linguist merge) | Syntax highlighted |

---

## Editor Integration Notes

### Zed
Zed automatically discovers grammars from the tree-sitter registry. Once published, Parsley support should appear automatically.

### Neovim
Users can add Parsley support via nvim-treesitter:
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

### Helix
Add to `languages.toml`:
```toml
[[language]]
name = "parsley"
scope = "source.parsley"
file-types = ["pars"]
roots = []

[[grammar]]
name = "parsley"
source = { git = "https://github.com/sambeau/tree-sitter-parsley", rev = "main" }
```

---

## Implementation Notes

### Zed Extension (Feb 2026)
- **Status**: ✅ Implemented
- **Location**: `contrib/zed-extension/`
- **Investigation**: See `FEAT-109-ZED-INVESTIGATION.md` for detailed feasibility analysis
- **Implementation Plan**: `PLAN-090.md`
- **Features**:
  - Full syntax highlighting (using `highlights.scm` from tree-sitter grammar)
  - Bracket matching for all Parsley bracket types
  - Code outline with functions and exports
  - Smart auto-indentation
  - Support for `.pars` and `.part` files
- **Testing**: Ready for local testing via Zed's dev extension feature (see `contrib/zed-extension/TESTING.md`)
- **Next Steps**:
  1. User validates extension locally in Zed
  2. Create standalone repository `github.com/sambeau/parsley-zed`
  3. Submit PR to `zed-industries/extensions` registry

### Tree-sitter Grammar
- Grammar repository: `contrib/tree-sitter-parsley/`
- 129/129 corpus tests passing
- Published as part of Basil monorepo at `https://github.com/sambeau/basil`
- Zed extension references: `contrib/tree-sitter-parsley` subdirectory

## Related
- Report: `work/reports/PARSLEY-1.0-ALPHA-READINESS.md` (Section 5)
- Grammar updates: FEAT-108 (VS Code and highlight.js)
- Lexer (source of truth): `pkg/parsley/lexer/lexer.go`
- Tree-sitter docs: https://tree-sitter.github.io/tree-sitter/
- Linguist: https://github.com/github/linguist