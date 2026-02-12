---
id: FEAT-114
title: "Tree-sitter Grammar: Direct Translation from Go Source"
status: draft
priority: high
created: 2026-02-12
author: "@human"
---

# FEAT-114: Tree-sitter Grammar — Direct Translation from Go Source

## Summary

Create a tree-sitter grammar for Parsley by directly translating the Go lexer and parser source code into tree-sitter's JavaScript DSL. The Go source IS the grammar — every rule, token, precedence level, and structural choice should mirror the Go implementation.

## Strategy

**Bottom-up translation.** Start with terminals (tokens), build up through literals, expressions, statements, and finally tags. Each step produces a working grammar with passing corpus tests for everything translated so far.

**This is a translation, not a design exercise.** The Go parser defines what Parsley is. The tree-sitter grammar should read like a 1:1 structural mirror of the Go code. When in doubt, read the Go function — it IS the spec.

## Source Files

The two source files that define the grammar:

| File | Role |
|------|------|
| `pkg/parsley/lexer/lexer.go` | Token types, keywords, operator patterns, string/regex/money lexing |
| `pkg/parsley/parser/parser.go` | Grammar rules, precedence, all `parse*` functions |

## Output Files

| File | Content |
|------|---------|
| `contrib/tree-sitter-parsley/grammar.js` | New grammar (replace entirely) |
| `contrib/tree-sitter-parsley/test/corpus/*.txt` | New test corpus (replace entirely) |
| `contrib/tree-sitter-parsley/src/scanner.c` | External scanner for style/script tags (Phase 2) |
| `contrib/zed-extension/languages/parsley/highlights.scm` | Updated for new node names |
| `contrib/zed-extension/languages/parsley/brackets.scm` | Updated |
| `contrib/zed-extension/languages/parsley/outline.scm` | Updated |
| `contrib/zed-extension/languages/parsley/indents.scm` | Updated |
| `contrib/zed-extension/test/sample.pars` | Rewritten with valid Parsley code |

## Real Parsley Code for Reference

Use working code from `/Users/samphillips/Dev/bofdi/` and `pkg/parsley/tests/*.go` for examples and corpus tests. These files demonstrate every major language feature in context:

- `bofdi/components/unsafeTable.pars` — Tags with code content, `for`, `if`, `not in`, destructuring, spread attributes
- `bofdi/components/page.pars` — Imports, arrays, `<style>` raw text, component pattern
- `bofdi/site/index.pars` — Full page with tags, nested components, string content
- `bofdi/site/edit/edit.pars` — `@query`, `@params`, `check...else`, `@schema` usage, datetime templates, complex control flow
- `bofdi/schema/person.pars` — `@schema` declaration with types, options, metadata
- `bofdi/schema/birthday.pars` — `export computed`, `@query`, `.map()`, `.as()`, datetime arithmetic

---

## Precedence Table

Direct from `parser.go` L17-29 and the `var precedences` map at L33-67:

```
Go Constant     Value   Tree-sitter Name    Operators
─────────────   ─────   ────────────────    ─────────
LOWEST          0       (default)
COMMA_PREC      1       COMMA               , ==> ==>> =/=> =/=>>
LOGIC_OR        2       OR                  or | || ??
LOGIC_AND       3       AND                 and & &&
EQUALS          4       EQUALS              == != ~ !~ in is <=?=> <=??=> <=!=> <=#=>
LESSGREATER     5       COMPARE             < > <= >=
SUM             6       SUM                 + - ..
CONCAT          7       CONCAT              ++
PRODUCT         8       PRODUCT             * / %
PREFIX          9       PREFIX              -x !x not x
INDEX           10      INDEX               a[i] a.b
CALL            11      CALL                f(x)
```

## Keywords

Direct from `var keywords` map at `lexer.go` L395-420:

```
Keyword      Token        Notes
────────     ─────        ─────
fn           FUNCTION
function     FUNCTION     alias
let          LET
for          FOR
in           IN
as           AS
true         TRUE
false        FALSE
if           IF
else         ELSE
return       RETURN
export       EXPORT
and          AND
or           OR
not          BANG         prefix operator
try          TRY
import       IMPORT
check        CHECK
stop         STOP
skip         SKIP
via          VIA          schema relations
is           IS           schema checking
computed     COMPUTED     computed exports
```

---

## Translation Plan

Each step translates one section of the Go source. After each step: `tree-sitter generate && tree-sitter test`.

### Step 1 — Scaffold and basic tokens

**Source:** `lexer.go` L10-160 (token type constants), L395-427 (keywords)

Create `grammar.js` with:
- `source_file` → `repeat($._statement)`
- `identifier` → `/[a-zA-Z_][a-zA-Z0-9_]*/`
- `comment` → `token(seq("//", /.*/))`
- `word: ($) => $.identifier`
- `extras: ($) => [/\s/, $.comment]`
- `PREC` constants matching the table above

Corpus: identifier, comment.

### Step 2 — Number literals

**Source:** `lexer.go` L1257-1272 (`readNumber`), `parser.go` L1071-1095 (`parseIntegerLiteral`, `parseFloatLiteral`)

- `number` → `/\d+(\.\d+)?/` (covers INT and FLOAT)

Corpus: integer, float.

### Step 3 — String literals

**Source:** `lexer.go` L1570-1664 (`readString`, `readRawString`, `readTemplate`)
**Source:** `parser.go` L1097-1107 (`parseStringLiteral`, `parseTemplateLiteral`, `parseRawTemplateLiteral`)

- `string` → `seq('"', repeat(choice($.escape_sequence, $.interpolation, /[^"\\{]+/)), '"')`
- `template_string` → `seq('`', repeat(choice($.escape_sequence, $.interpolation, /[^`\\{]+/)), '`')`
- `raw_string` → `seq("'", repeat(choice($.escape_sequence, $.raw_interpolation, $._raw_string_content)), "'")`
- `escape_sequence` → `/\\./`
- `interpolation` → `seq('{', $._expression, '}')`
- `raw_interpolation` → `seq('@{', $._expression, '}')`

Corpus: plain strings, strings with interpolation, template strings, raw strings with `@{}`.

### Step 4 — Boolean and null

**Source:** `parser.go` L1834-1836 (`parseBoolean`), `evaluator/eval_infix.go` L809-819 (`evalIdentifier`)

- `boolean` → `choice('true', 'false')`

`null` is NOT a keyword — it's a regular identifier that the evaluator special-cases (`evalIdentifier` returns `NULL` when `node.Value == "null"`). It parses as `$.identifier` like any other variable. Handle it in `highlights.scm` by matching the identifier text `"null"` for syntax highlighting.

Corpus: `true`, `false`.

### Step 5 — Regex literals

**Source:** `lexer.go` L2293-2327 (`readRegex`), `parser.go` L1109-1137 (`parseRegexLiteral`)

- `regex` → `token(seq('/', /[^\/\n]+/, '/', optional(/[gimsuvy]+/)))`

Note: `/` ambiguity (division vs regex) is handled by the Go lexer's `shouldTreatAsRegex()` which checks `lastTokenType`. Tree-sitter handles this structurally — regex can only appear where a prefix expression is expected.

Corpus: `/pattern/`, `/pattern/gi`.

### Step 6 — Money literals

**Source:** `lexer.go` L1275-1565 (`readMoneyLiteral`, `isCurrencyCodeStart`, `isCompoundCurrencySymbol`)

- `money` → covers `$`, `£`, `€`, `¥`, compound (`CA$`, `AU$`, `HK$`, `S$`, `CN¥`), and CODE# format (`USD#`, `GBP#`, etc.)

Pattern: `/([$£€¥]|[A-Z]{1,2}[$£€¥]|[A-Z]{3}#)\d+(\.\d{1,2})?/`

Corpus: `$100`, `£50.99`, `€25`, `¥1000`, `CA$50`, `USD#100.00`.

### Step 7 — @-literals (non-template)

**Source:** `lexer.go` L2474-2655 (`detectAtLiteralType`), L2659-2874 (individual read functions)
**Source:** `parser.go` prefix registrations L108-140

Each `@`-literal type from `detectAtLiteralType`:

- `datetime_literal` → `@2024-01-15`, `@2024-01-15T10:30:00Z`, `@12:30:00`
- `time_now_literal` → `@now`, `@today`, `@timeNow`, `@dateNow`
- `duration_literal` → `@2h30m`, `@-7d`, `@1y6mo`
- `connection_literal` → `@sqlite`, `@postgres`, `@mysql`, `@sftp`, `@shell`, `@DB`
- `context_literal` → `@SEARCH`, `@env`, `@args`, `@params`
- `schema_literal` → `@schema`
- `table_literal` → `@table`
- `query_literal` → `@query`
- `mutation_literal` → `@insert`, `@update`, `@delete`, `@transaction`
- `stdlib_import` → `@std/math`, `@basil/http`, `@basil/auth`
- `stdio_literal` → `@-`, `@stdin`, `@stdout`, `@stderr`
- `path_literal` → `@./file`, `@../dir`, `@/usr/local`, `@~/home`
- `url_literal` → `@https://example.com`

Corpus: one test per literal type, taken from bofdi code examples.

### Step 8 — @-template literals

**Source:** `lexer.go` L2900-3032 (`detectTemplateAtLiteralType`, `readPathTemplate`, `readUrlTemplate`, `readDatetimeTemplate`)
**Source:** `parser.go` L1330-1352

- `path_template` → `seq('@(', repeat(choice(/[^{}()]+/, $.interpolation)), ')')`
- `url_template` → same structure with URL prefix
- `datetime_template` → same structure with datetime prefix

Corpus: `@(./path/{name}/file)`, `@({year}-{month}-{day})`.

### Step 9 — Array literals

**Source:** `parser.go` L2139-2169 (`parseSquareBracketArrayLiteral`)

- `array_literal` → `seq('[', commaSep($._expression), ']')`

Note: No spread in arrays — the parser just calls `parseExpression` for each element.

Corpus: `[]`, `[1, 2, 3]`, `[1, "two", true]`, nested arrays.

### Step 10 — Dictionary literals

**Source:** `parser.go` L2988-3079 (`parseDictionaryLiteral`)

- `dictionary_literal` → `seq('{', commaSep(choice($.pair, $.computed_property)), '}')`
- `pair` → `seq(field('key', choice($.identifier, $.string)), ':', field('value', $._expression))`
- `computed_property` → `seq('[', field('key', $._expression), ']', ':', field('value', $._expression))`

Note: No spread in dictionaries and no shorthand properties — the parser always expects `key: value` pairs or computed `[key]: value`.

Corpus: `{}`, `{a: 1}`, `{a: 1, b: 2}`, computed keys, string keys.

### Step 11 — Primary expressions and expression framework

**Source:** `parser.go` prefix registrations L97-155, `parseExpression` L957-1012

Wire up `_expression` and `_primary_expression`:

- `_primary_expression` → `choice($.identifier, $._literal, $.array_literal, $.dictionary_literal)`
- `_literal` → `choice($.number, $.string, $.template_string, $.raw_string, $.regex, $.boolean, $.null, $.money, $._at_literal)`
- `_expression` → all expression types (built incrementally in following steps)

Corpus: expression statements with each literal type.

### Step 12 — Prefix expressions

**Source:** `parser.go` L1963-2000 (`parsePrefixExpression`, `parseReadExpression`, `parseFetchExpression`)

- Unary: `-expr`, `!expr`, `not expr`
- Read prefix: `<== expr`
- Fetch prefix: `<=/= expr`

Corpus: `-x`, `!flag`, `not condition`, `<== @./file.txt`.

### Step 13 — Infix expressions (binary operators)

**Source:** `parser.go` L2002-2014 (`parseInfixExpression`), infix registrations L158-183, precedence table L33-67

All operators registered via `registerInfix` that use `parseInfixExpression`:

```
Arithmetic:    +  -  *  /  %
Comparison:    ==  !=  <  >  <=  >=
Logical:       and  or  (also && ||)
Regex:         ~  !~
Membership:    in
Concatenation: ++
Range:         ..
Nullish:       ??
File I/O:      ==>  ==>>  =/=>  =/=>>
Database:      <=?=>  <=??=>  <=!=>
Process:       <=#=>
```

Each with correct precedence and associativity from the table.

Corpus: one test per operator, a precedence test showing `1 + 2 * 3` groups correctly.

### Step 14 — Special infix operators

**Source:** `parser.go` L2019-2064 (`parseNotInExpression`, `parseIsExpression`)

- `not in` → compound operator: `expr not in expr` (registered as BANG infix, checks for following `in`)
- `is` / `is not` → `expr is expr`, `expr is not expr`

Corpus: `x not in list`, `value is Schema`, `value is not Schema`.

### Step 15 — Call expressions

**Source:** `parser.go` L2498-2577 (`parseCallExpression`, `parseExpressionList`)

- `call_expression` → `prec(CALL, seq(field('function', $._expression), field('arguments', $.arguments)))`
- `arguments` → `seq('(', commaSep($._expression), ')')`

Note: Only certain expression types are callable (identifiers, member access, index, calls, function literals, connection literals, grouped). Tree-sitter doesn't enforce this — it's a semantic check.

Note: No spread `...` in call arguments.

Corpus: `f()`, `f(1, 2)`, `obj.method()`, chained calls.

### Step 16 — Index and slice expressions

**Source:** `parser.go` L2579-2637 (`parseIndexOrSliceExpression`, `parseSliceExpression`)

- `index_expression` → `prec(INDEX, seq(field('object', $._expression), '[', field('index', $._expression), ']'))`
- `slice_expression` → `prec(INDEX, seq(field('object', $._expression), '[', optional(field('start', $._expression)), ':', optional(field('end', $._expression)), ']'))`

Corpus: `arr[0]`, `arr[1:3]`, `arr[:5]`, `arr[2:]`.

### Step 17 — Member (dot) expressions

**Source:** `parser.go` L3082-3097 (`parseDotExpression`)

- `member_expression` → `prec(INDEX, seq(field('object', $._expression), '.', field('property', $.identifier)))`

Corpus: `obj.name`, `a.b.c`, `arr.length`.

### Step 18 — Parenthesized expressions

**Source:** `parser.go` L2079-2137 (`parseGroupedExpression`)

- `parenthesized_expression` → `seq('(', $._expression, ')')`

Corpus: `(1 + 2) * 3`.

### Step 19 — Function expressions

**Source:** `parser.go` L2303-2385 (`parseFunctionLiteral`, `parseFunctionParametersNew`, `parseFunctionParameter`)

- `function_expression` → `seq(choice('fn', 'function'), optional(field('parameters', $.parameter_list)), field('body', $.block))`
- `parameter_list` → `seq('(', commaSep(choice($.identifier, $._destructuring_param, $._default_param, $._rest_param)), ')')`
- `_default_param` → `seq($.identifier, '=', $._expression)`
- `_rest_param` → `seq('...', $.identifier)`
- `_destructuring_param` → array or dictionary destructuring pattern

Body is ALWAYS a block `{...}`. No expression bodies.

`fn {}` (no parameters) is also valid.

Corpus: `fn(x) { x * 2 }`, `fn(a, b) { a + b }`, `fn(x, y = 0) { x + y }`, `fn(...rest) { rest }`, `fn({a, b}) { a + b }`, `fn {} `.

### Step 20 — Block

**Source:** `parser.go` L2286-2301 (`parseBlockStatement`)

- `block` → `seq('{', repeat($._statement), '}')`

Corpus: `{ let x = 1; x + 2 }`.

### Step 21 — If expression

**Source:** `parser.go` L2171-2284 (`parseIfExpression`)

Two forms depending on whether parens are present:

**With parens** — `if (cond)` — consequence can be a block OR a single expression:
- `if (cond) { body }` — block form
- `if (cond) expr` — compact form (single expression, no braces)
- `if (cond) expr else expr` — compact ternary-style
- `if (cond) return expr` — return in compact form

**Without parens** — `if cond` — consequence MUST be a block:
- `if cond { body }` — condition is parsed until `{` is reached
- `if cond expr` — ERROR: "if without parentheses requires braces"

Both forms support optional `else`:
- `else { body }` — block
- `else expr` — single expression (compact)
- `else if ...` — chained

The tree-sitter rule needs to handle both forms. Simplest approach: treat parens as optional grouping, consequence as `choice($.block, $._expression)`.

Corpus: `if (x > 0) { "positive" }`, `if (x > 0) { "yes" } else { "no" }`, `if (x > 0) "yes" else "no"`, `if x > 0 { "yes" } else { "no" }`, `if (a) { } else if (b) { } else { }`.

### Step 22 — For expression

**Source:** `parser.go` L2390-2496 (`parseForExpression`)

Two forms:

**Iteration form** — `for var in iterable { body }`:
- With parens: `for (x in arr) { body }` — iterable parsed as full expression, then expect `)`
- Without parens: `for x in arr { body }` — iterable parsed until `{` is reached
- With key,value: `for (k, v in dict) { body }`
- Body is ALWAYS a block `{...}` — `for (x in arr) expr` is an error

**Mapping form** — `for (iterable) function`:
- REQUIRES parens: `for (arr) fn(x) { x * 2 }`
- Without parens is an error: parser can't determine where iterable ends
- `for (arr) { ... }` is also an error (ambiguous — use iteration form instead)

Note: the `for item in list` without-parens form works because the parser checks `peekTokenIs(lexer.IN)` to detect the iteration pattern, then uses `parseExpressionUntilBrace()` for the iterable.

Corpus: `for (x in arr) { x * 2 }`, `for (k, v in dict) { k }`, `for x in arr { x }`, `for (arr) fn(x) { x * 2 }`.

### Step 23 — Try expression

**Source:** `parser.go` L1838-1863 (`parseTryExpression`)

- `try_expression` → `seq('try', $.call_expression)`

Body must be a call expression (function call or method call). Not any arbitrary expression.

Corpus: `try fetchData()`, `try obj.load()`.

### Step 24 — Import expression

**Source:** `parser.go` L1871-1931 (`parseImportExpression`)

- `import_expression` → `seq('import', field('source', $._expression), optional(seq('as', field('alias', $.identifier))))`

Corpus: `import @std/math`, `import @./utils.pars as utils`, `let {Page} = import @std/html`.

### Step 25 — Patterns (destructuring)

**Source:** `parser.go` L3100-3250 (`parseDictDestructuringPattern`, `parseArrayDestructuringPattern`)

- `_pattern` → `choice($.identifier, $.array_pattern, $.dictionary_pattern, '_')`
- `array_pattern` → `seq('[', commaSep(choice($._pattern, seq('...', optional($.identifier)))), ']')`
- `dictionary_pattern` → `seq('{', commaSep(choice($.identifier, seq(field('key', $.identifier), ':', field('value', $._pattern)), seq('...', optional($.identifier)))), '}')`

Corpus: `let [a, b, c] = arr`, `let {name, age} = person`, `let [first, ...rest] = arr`, `{a, ...rest} = dict`.

### Step 26 — Statements: let

**Source:** `parser.go` L499-669 (`parseLetStatement`)

- `let_statement` → `seq('let', field('pattern', $._pattern), '=', field('value', $._expression))`

Corpus: `let x = 5`, `let [a, b] = arr`, `let {name} = person`.

### Step 27 — Statements: assignment (bare, no let)

**Source:** `parser.go` L292-370 (`parseStatement` — IDENT followed by ASSIGN case), L673-730 (`parseAssignmentStatement`)

- `assignment_statement` → `seq(field('left', choice($.identifier, $.member_expression, $.index_expression)), '=', field('right', $._expression))`

Also: dictionary destructuring assignment `{a, b} = expr` (parser.go L733-790).

Note: The parser checks for `IDENT` followed by `ASSIGN` in `parseStatement` to distinguish from expression statements.

Corpus: `x = 5`, `obj.name = "Alice"`, `arr[0] = 99`, `{a, b} = expr`.

### Step 28 — Statements: export

**Source:** `parser.go` L379-496 (`parseExportStatement`, `parseComputedExportStatement`)

- `export_statement` → `seq('export', optional('computed'), field('name', $.identifier), '=', field('value', $._expression))`

Also: `export @schema` and `export @table` — export followed by an @-literal expression.

Corpus: `export greeting = "Hello"`, `export computed total = a + b`.

### Step 29 — Statements: return, check, stop, skip

**Source:** `parser.go` L793-860

- `return_statement` → `seq('return', optional($._expression))`
- `check_statement` → `seq('check', field('condition', $._expression), 'else', field('fallback', $._expression))`
- `stop_statement` → `'stop'`
- `skip_statement` → `'skip'`

Check REQUIRES `else` followed by a fallback expression.

Corpus: `return x + 1`, `return`, `check x > 0 else "negative"`, `check data else {error: "no data"}`, `stop`, `skip`.

### Step 30 — Statements: expression statement

**Source:** `parser.go` L863-945 (`parseExpressionStatement`)

- `expression_statement` → `$._expression`

This is the fallback — any expression is a valid statement.

Corpus: `greet("World")`, `x + y`.

### Step 31 — Statement dispatch

**Source:** `parser.go` L292-370 (`parseStatement`)

Wire up `_statement`:

```
_statement → choice(
  $.let_statement,
  $.assignment_statement,
  $.export_statement,
  $.return_statement,
  $.check_statement,
  $.stop_statement,
  $.skip_statement,
  $.expression_statement,
)
```

The `parseStatement` switch at L292 defines the priority:
1. `EXPORT` → export
2. `LET` → let
3. `RETURN` → return
4. `CHECK` → check
5. `STOP` → stop
6. `SKIP` → skip
7. `LBRACE` → try dict destructuring assignment, fall back to expression
8. `IDENT` followed by `ASSIGN` → assignment
9. default → expression statement

Corpus: a multi-statement program exercising each statement type.

### Step 32 — Tags: self-closing and open/close

**Source:** `parser.go` L1354-1365 (`parseTagLiteral`), L1367-1528 (`parseTagPair`, `parseTagContents`)
**Source:** `lexer.go` L1802-1923 (`readTagStartOrSingleton`)

- `tag_expression` → `choice($.self_closing_tag, seq($.open_tag, repeat($._statement), $.close_tag))`
- `self_closing_tag` → `seq('<', field('name', $.tag_name), repeat($.tag_attribute), '/>')`
- `open_tag` → `seq('<', field('name', $.tag_name), repeat($.tag_attribute), '>')`
- `close_tag` → `seq('</', field('name', $.tag_name), '>')`
- `tag_name` → `/[a-zA-Z][a-zA-Z0-9-]*/`

**Critical:** Tag content is `repeat($._statement)` — full Parsley code. NOT text with interpolation.

This is because the Go lexer does NOT enter a special mode for normal tags. After `<div>`, `NextToken()` continues in normal mode. The parser's `parseTagContents()` calls `parseStatement()` for each token (L1508-1522).

Grouping tags: `<>...</>` (empty tag name).

Corpus: `<br/>`, `<div>"Hello"</div>`, `<p>name</p>` (variable reference), `<div><span>"nested"</span></div>`.

### Step 33 — Tag attributes

**Source:** `parser.go` L1645-1814 (`parseTagAttributes`), `lexer.go` tag attribute handling

- `tag_attribute` → `choice(seq(field('name', $.attribute_name), '=', field('value', choice($.string, $.tag_expression_attribute))), field('name', $.attribute_name), $.tag_spread_attribute)`
- `attribute_name` → `/[a-zA-Z@][a-zA-Z0-9_-]*/` (note: allows `@field`, `@record` etc.)
- `tag_expression_attribute` → `seq('{', $._expression, '}')`
- `tag_spread_attribute` → `seq('...', $.identifier)`

Note: Expression attributes use `{expr}` — this is the ONE place where `{expr}` is JSX-like. Inside tag CONTENT, `{` starts a dictionary. In tag ATTRIBUTES, `{expr}` is an expression value.

Corpus: `<div class="container">`, `<div class={expr}>`, `<button ...props>`, `<input @field="name"/>`.

### Step 34 — Tags with code content

**Source:** `parser.go` L1421-1528 (`parseTagContents`)

Since tag content is `repeat($._statement)`, all Parsley code works inside tags. Test with real examples from bofdi:

Corpus:
```
<ul>
  for (item in items) {
    <li>item</li>
  }
</ul>
```

```
<div>
  if (condition) {
    <p>"yes"</p>
  } else {
    <p>"no"</p>
  }
</div>
```

```
<table>
  <thead>
    <tr>
      for (k, _ in rows[0]) {
        if (k not in hidden)
          <th class={"th-" + k}>k.toTitle()</th>
      }
    </tr>
  </thead>
</table>
```

### Step 35 — Schema declarations (Phase 2)

**Source:** `parser.go` L3253-3410 (`parseSchemaDeclaration`, `parseSchemaField`)

- `schema_declaration` → `seq('@schema', field('name', $.identifier), '{', repeat($.schema_field), '}')`
- `schema_field` → `seq(field('name', $.identifier), ':', field('type', $.identifier), optional('?'), optional($.enum_values), optional($.type_options), optional(seq('=', field('default', $._expression))), optional(seq('|', field('metadata', $.dictionary_literal))), optional(seq('via', field('foreign_key', $.identifier))))`

Note: Can appear after `export`: `export @schema Person { ... }`

Corpus: from `bofdi/schema/person.pars`.

### Step 36 — Table literals (Phase 2)

**Source:** `parser.go` L3413-3523 (`parseTableLiteral`)

- `table_literal` → `seq('@table', optional(seq('(', field('schema', $.identifier), ')')), '[', commaSep($.dictionary_literal), ']')`

Corpus: `@table [{name: "Alice"}, {name: "Bob"}]`.

### Step 37 — Query DSL (Phase 2)

**Source:** `parser.go` L3611-3764 (`parseQueryExpression`), L4588-4800 (insert/update/delete/transaction)

The query DSL is a complex sub-grammar with its own parsing rules:

- `query_expression` → `seq('@query', '(', ...query clauses..., ')')`
- Query clauses: source, optional alias (`as`), conditions (`| field op value`), modifiers (`| order`, `| limit`), group by (`+ by`), terminal (`?->`, `??->`, `*`, field projections)

Also: `@insert(...)`, `@update(...)`, `@delete(...)`, `@transaction(...)`

Corpus: from `bofdi/schema/birthday.pars`:
```
@query(People ??-> *)
@query(
  People
  | Firstname == {person.Firstname}
  | Surname == {person.Surname}
  ?-> count
)
```

### Step 38 — Style/script raw text tags (Phase 2)

**Source:** `lexer.go` L1926-2160 (`nextTagContentToken` — raw text mode)

Requires a tree-sitter **external scanner** (`src/scanner.c`) to handle:
- Content between `<style>...</style>` and `<script>...</script>` as raw text
- `@{expr}` interpolation within raw text
- Everything else is literal text (including `{` and `}`)

Corpus: from `bofdi/components/page.pars`:
```
<style>
  :root {
    --pico-font-size: 1rem;
  }
</style>
```

### Step 39 — Highlights and queries

Update Zed extension query files to match the new node names:

- `highlights.scm` — keywords, operators, literals, types
- `brackets.scm` — matching pairs
- `outline.scm` — let, export, function definitions
- `indents.scm` — blocks, tags, arrays, dictionaries

### Step 40 — Sample file

Replace `contrib/zed-extension/test/sample.pars` with a file containing valid Parsley code. Use a condensed version of the bofdi code that exercises all major features.

---

## Phase 2: External Scanner and Remaining Work

Phase 2 addresses the remaining gaps that require context-sensitive tokenization beyond what tree-sitter's pure JS grammar can express. These items were deferred from Phase 1 because they require a C external scanner or sub-grammar work.

**Implementation plan:** PLAN-092

### Phase 2 Items

#### 1. External scanner for multi-sibling tag children (High Priority)

**Problem:** Inside a tag like `<html>`, children are full Parsley code. After parsing `</head>`, tree-sitter encounters `<body>` — the `<` is ambiguous between less-than (infix) and the start of a new sibling tag. The Go lexer solves this with stateful `tagDepth` tracking and lookahead, but tree-sitter's pure JS grammar has no such state. This causes most real handler files (which use multiple sibling tags) to produce parse errors.

**Solution:** A C external scanner (`src/scanner.c`) that maintains a tag stack. When inside tag children and encountering `<`, it checks whether the next characters form `</tagname>` (close tag) or `<tagname` (new open tag), and emits the correct token type. Estimated ~150–300 lines of C.

**Outcome:** All real handler files with multiple sibling tags parse correctly. This is the single most impactful Phase 2 item.

#### 2. Style/script raw text tags with `@{}` interpolation (High Priority)

**Problem:** In the Go lexer, `<style>` and `<script>` are special — their contents are raw text (not Parsley code), and only `@{...}` triggers interpolation back into Parsley. The current grammar treats their children as Parsley code, so CSS/JS content would be parsed as broken Parsley expressions.

**Solution:** Extend the external scanner (from item 1) to recognize `<style>`/`<script>` tags, switch into raw text mode, and emit everything as opaque `raw_text` tokens. The scanner breaks out only for `@{` (interpolation) or `</style>`/`</script>` (close tag). Adds ~100–200 lines to the C scanner.

**Outcome:** The parse tree for style/script tags produces `raw_text` nodes for literal CSS/JS content and `interpolation` nodes for `@{expr}` expressions:

```
(tag_expression
  (open_tag (tag_name))          ; <style>
  (raw_text)                     ; ".container { background-color: "
  (interpolation ...)            ; @{theme.bg}
  (raw_text)                     ; "; font-size: 14px; }"
  (close_tag (tag_name)))        ; </style>
```

#### 3. Language injection queries (Low Priority — trivial addition)

**Problem:** Once raw text nodes exist in style/script tags (item 2), editors could provide CSS/JS syntax highlighting inside those tags using tree-sitter's language injection feature. This is optional polish — treating raw content as opaque strings with highlighted `@{}` interpolations is sufficient for v1.

**Solution:** Write an `injections.scm` query file (~10–15 lines) that tells the editor to parse `raw_text` nodes inside `<style>` tags with the CSS tree-sitter grammar, and `<script>` tags with the JavaScript grammar. The `@{}` interpolation nodes become "holes" in the injected parse that keep their Parsley highlighting.

**Outcome:** Full CSS syntax highlighting inside `<style>` tags and JS highlighting inside `<script>` tags in any editor that supports tree-sitter injection (Zed, Neovim, Helix, Emacs). No grammar or scanner changes needed beyond what item 2 provides.

#### 4. Regex vs division external scanner (Medium Priority)

**Problem:** `/pattern/flags` (regex) vs `a / b` (division) are ambiguous to a context-free parser. The Go lexer uses `shouldTreatAsRegex()` which checks the previous token type. The current grammar uses a `token(...)` rule that works for common cases but can misparse complex expressions.

**Solution:** Add logic to the external scanner that tracks the previous token type and decides whether `/` should start a regex or be a division operator, mirroring the Go lexer's `shouldTreatAsRegex`. Adds ~50–100 lines to the C scanner.

**Outcome:** Regex literals and division are always parsed correctly regardless of context.

#### 5. Query DSL sub-grammar (Low Priority)

**Problem:** `@query(...)` contains a SQL-like DSL that is its own mini-language. Currently the grammar captures balanced parentheses as opaque tokens — it won't break the parse, but it won't structure the query content.

**Solution:** Define grammar rules for the query DSL's actual syntax (clauses, conditions, table references, `{expr}` interpolation). Replace the balanced-paren stub with real nodes.

**Outcome:** Query expressions have structured parse trees with named nodes for clauses, conditions, field references, enabling proper syntax highlighting of queries. Without this, `@query(...)` bodies show as unstructured text — functional but not pretty. This is the lowest priority item.

#### 6. Zed extension queries — highlights, brackets, outline, indents (Medium Priority)

**Problem:** Tree-sitter grammars are useless in an editor without `.scm` query files that map parse tree nodes to editor features. These queries reference node names, so they break if node names change. Blocked on node name stabilization.

**Solution:** Write/update `highlights.scm`, `brackets.scm`, `outline.scm`, and `indents.scm` with patterns referencing the final node names from the rewritten grammar.

**Outcome:** Full Parsley editing support in Zed (and any tree-sitter-capable editor): syntax coloring, auto-indent, bracket matching, code outline. This is where manual testing becomes possible.

### External Scanner — Design Notes

**What is it?** An external scanner is a standard, first-class feature of tree-sitter itself — not Zed-specific. It's a hand-written C file (`scanner.c`) that tree-sitter calls during parsing for context-sensitive tokenization.

**How it works:**
1. In `grammar.js`, declare `externals: $ => [$.raw_text, ...]` — token types the external scanner can produce.
2. During parsing, tree-sitter calls the external scanner first. It can claim a token (returning true) or decline (returning false), letting the internal lexer handle it.
3. The scanner has `serialize`/`deserialize` for state persistence across tree-sitter's GLR backtracking.

**Precedent:** Many mainstream grammars use external scanners — HTML (raw text in `<script>`/`<style>`), JavaScript (regex vs division, template strings, automatic semicolons), Python (indentation), Ruby, TypeScript, etc.

**Portability:** The scanner ships as part of the grammar. Any tool using `tree-sitter-parsley` gets it automatically: Zed, Neovim, Emacs, Helix, GitHub (linguist), `ast-grep`, language servers, linters — anything using tree-sitter as a library. It imposes no constraints on downstream consumers.

### What Becomes Possible with a Complete Grammar

- **Full editor support** — syntax highlighting, indentation, bracket matching, code outline, code folding in any tree-sitter-capable editor
- **Language injection** — CSS highlighting inside `<style>`, JS inside `<script>` via injection queries
- **Structural code search** — tools like `ast-grep` can search/refactor Parsley by structure, not text
- **GitHub code navigation** — if submitted to the tree-sitter grammar registry
- **Linting/formatting foundations** — parse tree usable for building tools without the Go runtime
- **LSP foundation** — a future language server could use tree-sitter for fast, incremental parsing

### Current Limitations (Phase 1 Only)

Without Phase 2, the grammar has these practical gaps:
- **Multi-sibling tags produce errors** — most real handler files won't parse cleanly (item 1)
- **`<style>`/`<script>` contents are wrong** — parsed as broken Parsley instead of raw text (item 2)
- **Some regex literals may misparse** — edge cases where `/` is ambiguous (item 4)
- **Queries are opaque** — `@query(...)` bodies lack internal structure (item 5)

The grammar is usable for pure-expression Parsley (scripts without tags), but tag issues make it unreliable for web handlers/templates — the most common use case.

## Conflicts and Ambiguities

Tree-sitter's GLR parser handles ambiguity via `conflicts` declarations. Known cases from the Go parser:

1. **`<` — tag start vs less-than.** The Go lexer uses `peekChar()` — `<` followed by letter/`>` is a tag, otherwise comparison. Tree-sitter handles this structurally since `open_tag` and `self_closing_tag` have distinct shapes.

2. **`/` — division vs regex.** The Go lexer uses `shouldTreatAsRegex()` checking `lastTokenType`. Tree-sitter handles this by regex only appearing where a prefix is expected.

3. **`{` — dictionary vs block.** The Go parser uses backtracking for dict destructuring assignment. Tree-sitter needs a `conflicts` declaration.

4. **`not` — prefix vs `not in` infix.** The Go parser registers `BANG` as both prefix and infix, checking for following `in` in the infix case. Tree-sitter may need a compound `not_in` operator or conflict resolution.

5. **Statement vs expression in tag content.** Since tag content is `repeat($._statement)` and `_statement` includes `expression_statement`, this should work naturally.

## Validation

After each step:
1. `tree-sitter generate` — grammar compiles
2. `tree-sitter test` — corpus passes
3. `tree-sitter parse <file>` — test against real .pars files from bofdi/

Final validation: `tree-sitter parse` on all bofdi .pars files should produce clean trees with no ERROR nodes.

## Related

- Go lexer: `pkg/parsley/lexer/lexer.go`
- Go parser: `pkg/parsley/parser/parser.go`
- Real Parsley code: `/Users/samphillips/Dev/bofdi/`
- Parsley tests: `pkg/parsley/tests/*.go` (124 files with valid Parsley input strings)
- Tree-sitter output: `contrib/tree-sitter-parsley/`
- Zed extension: `contrib/zed-extension/`
- Feature: FEAT-109 (Zed Extension for Parsley)