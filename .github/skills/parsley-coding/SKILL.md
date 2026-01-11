---
name: parsley-coding
description: How to write Parsley code for tests, examples and documentation.
updated: 2026-01-11
---

# Writing Parsley Code

This skill helps you write and debug Parsley code.

## When to use this skill

Use this skill when you need to:
- Create new tests in Parsley code
- When debugging functionality during development
- Creating examples of how to use a feature
- Discussing new features and how they might fit into Parsley/Basil ecosystem
- Creating workflow docs: designs, specs, plans, bugs, reports
- Creating documentation that has Parsley code examples

## Writing code

1. Read ``docs/parsley/CHEATSHEET.md`` to learn Parsley code especially common mistakes
2. Use ``pars`` to test and validate Parsley code snippets

### Notes:

- Parsely code looks like Javascript, but it is expression-based
- Parsely code looks like React/JSX code, but tags and code co-exist — code is not interpolated inside { } brackets
- ``if`` and ``for`` are expressions that return values
- functions (``fn``) return everything by default, so do not need ``return`` statement 
- Parsley has rich pseudo-types for dates, times, money with their own literals with `@` constructors
- Parsley has a standard library ``import @std/…``
- By convention, Parsley files use ``.pars`` file-ending

## Common Pitfalls
(see docs/parsley/CHEATSHEET.md for more Major Gotchas (Common Mistakes))

### 1. Code within tag pairs needs no { and } brackets

```parsley
export Table = fn({rows, hidden, ...props}){
	<table ...props>
		<thead>
			<tr>
				for (k,_ in rows[0]){
					if (k not in hidden)
						<th class={"th-"+k}>k.toTitle()</th>
				}
			</tr>
		</thead>
		<tbody>
			for (row in rows){
				<tr>
				for (k,v in row){
					if (k not in hidden)
						<td class={"td-"+k}>v</td>
				}
				</tr>
			}
		</tbody>
	</table>
}
```

### 2. Import syntax uses path literals and destructuring

```parsley
let {query} = import @basil/http
let {session} = import @basil/auth
let {Page as BasilPage} = import @std/html
let {People} = import @~/schema/birthdays.pars
```

### 3. ``for`` returns an array of values, making it more like map and filter

```parsley
let doubled = for (n in [1,2,3]) { n * 2 }  // [2, 4, 6]

// Filter pattern - if returns null, omitted from result
let evens = for (n in [1,2,3,4]) {
    if (n % 2 == 0) { n }  // [2, 4]
}
```

### 4. If  Parentheses are optional but recommended

```parsley
if age >= 18 { "adult" }
if (age >= 18) { "adult" }
let status = if (age >= 18) "adult" else "minor"
```

### 5. Literals Use @ (Path, Date, Time, Duration, ...)

```parsley
// ✅ CORRECT
let path = @./config.json          // Relative to current file
let rootPath = @~/components/nav   // Relative to project root (Basil)

// ❌ WRONG
let path = "./config.json"  // This is just a string

// Other litreals
let url = @https://example.com
let date = @2024-11-29
let time = @14:30
let duration = @1d
```

### 6. No Arrow Functions - Use fn() { }

```parsley
// ❌ WRONG (JavaScript arrow functions)
arr.map(x => x * 2)

// ✅ CORRECT - Use fn() { } syntax
arr.map(fn(x) { x * 2 })
arr.filter(fn(x) { x > 0 })

// Named functions
let double = fn(x) { x * 2 }
```

### 7. Strings in HTML Must Be Quoted

```parsley
<h3>Welcome to Parsley</h3>  // ❌ WRONG - bare text in tags

<h3>"Welcome to Parsley"</h3>  // ✅ CORRECT - strings need quotes

<h3>`Welcome to {name}`</h3>  // Template literal style also works
```

### 8. Tag Attributes Don't Need Quotes for Simple Values

```parsley
// ✅ CORRECT - no quotes needed for simple identifiers
<div class=container id=main disabled=true>
<button type=submit disabled={isDisabled}>

// ✅ ALSO CORRECT - quotes when you need them
<div class="user-profile" id={userId}>
<a href="/about">

// Use quotes for:
// - Multi-word values: class="nav item"
// - Values with special chars: onclick="alert('hi')"
// - String literals vs expressions
```

### 9. Tag Attributes: Strings vs Expressions

```parsley
// Tag attributes have THREE forms:

// 1. Double-quoted strings - literal, no interpolation
<button onclick="alert('hello')">
<a href="/about">

// 2. Single-quoted strings - RAW, for embedding JavaScript
<button onclick='Parts.refresh("editor", {id: 1})'>
// ^ Double quotes and braces stay literal - perfect for JS!
// Use @{} for dynamic values:
<button onclick='Parts.refresh("editor", {id: @{myId}})'>

// 3. Expression braces - Parsley code
<div class={`user-{id}`}>              // Template string for dynamic class
<button disabled={!isValid}>           // Boolean expression
<img width={width} height={height}>

// ❌ WRONG - interpolation in quoted strings
<div class="user-{id}">               // {id} is literal text

// ✅ CORRECT - use expression braces with template string
<div class={`user-{id}`}>
```

### 10. Single-Quoted Raw Strings (JavaScript Embedding)

```parsley
// Single quotes create raw strings - braces stay literal
let js = 'Parts.refresh("editor", {id: 1})'
let regex = '\d+\.\d+'              // Backslashes stay literal

// Use @{} for interpolation inside raw strings
let id = 42
let js = 'Parts.refresh("editor", {id: @{id}})'  // id interpolated

// Perfect for onclick handlers with dynamic values:
let myId = 5
<button onclick='Parts.refresh("editor", {id: @{myId}, view: "delete"})'/>

// Static JS (no interpolation needed):
<button onclick='Parts.refresh("editor", {id: 1, view: "delete"})'/>

// Escape @ with \@ if you need a literal @
'email: user\@domain.com'          // literal @
```

### 11. Singleton Tags MUST Self-Close, Paired Tags Can Be Empty

```parsley
// ❌ WRONG - not self-closing
<br>
<img src="photo.jpg">
<Part src={@./foo.part}>

// ✅ CORRECT - self-closing tags need />
<br/>
<img src="photo.jpg"/>
<Part src={@./foo.part}/>
```

### 13. Use .length() to find the length of something

```parsley
[1,2,3].length() // => 3
"ABC".length() // => 3
```

### 14. Use .type() to get the type name of a value (as a string)

```parsley
123.type() // => "integer"
[1,2,3].type() // => "array"
"hello".type() // => "string"
@1968-11-21.type() // => "datetime"
$5.type() // => "money"
@now.type() // => "datetime"
```

### 15. Module System: let vs export

```parsley
// Export functions/values to make them available to importers
export myFunc = fn(x) { x * 2 }
export myVar = 42

// Or export multiple at once
export {myFunc, myVar}

// let without export is file-local (private)
let private = 123        // Not available to importers
export public = 456      // Available to importers

// Default export
export default = fn() { "I'm the default" }
```

### 16. Standard Library Modules (Current)

```parsley
// Standard library modules (as of 2026-01-11)
import @std/mdDoc        // ✅ Markdown (mdDoc pseudo-type)
import @std/table        // Table DSL
import @std/math         // Math functions
import @std/valid        // Validation
import @std/schema       // Schema validation
import @std/id           // ID generation (UUID, ULID, etc)
import @std/api          // API helpers (redirect, notFound, etc)
import @std/html         // HTML components
import @std/dev          // Dev tools

// Basil context (available in handlers)
import @basil/http       // request, response, query, route
import @basil/auth       // db, session, auth, user

// ❌ DEPRECATED/REMOVED - don't use these:
// @std/markdown - removed, use @std/mdDoc
// now() builtin - removed, use @now magic variable
```

### 17. Magic Variables with @

```parsley
// Current datetime
@now                     // Current datetime
@now.year                // 2026
@now.format()            // "2026-01-11 14:30:00"

// Environment variables
@env.HOME                // "/Users/username"
@env.PATH                // System PATH

// Special values
@params                  // URL/form parameters (in Basil handlers)
```

## Running pars

To test Parsley code locally:

```bash
./pars filename.pars              # Run a script
./pars -e "1 + 2"                 # Evaluate expression
./pars -w script.pars              # Allow file writes
./pars -x script.pars              # Allow imports
```

## Testing Context

When writing tests and examples:

- **Unit tests**: Place in `pkg/parsley/tests/*.pars` or `*_test.go`
- **Integration tests**: Use Go test files with Parsley evaluation
- **Examples**: Create in `examples/*/handlers/*.pars`
- **Quick validation**: Use `./pars` to test snippets before documenting
- **Handler testing**: Run `./basil --dev` and test in browser

## Best practices

- Consult cheatsheet
- Run code snippets in ``pars`` before using in documentation
- Run all examples to be sure they are valid code