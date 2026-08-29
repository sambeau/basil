---
id: man-pars-repl
title: Interactive REPL
system: parsley
type: features
name: repl
created: 2026-02-26
version: 0.2.0
author: Basil Team
keywords:
  - repl
  - interactive
  - shell
  - console
  - debugger
---

# Interactive REPL

The Parsley REPL (Read-Eval-Print Loop) provides an interactive environment for exploring the language, testing expressions, and debugging code.

## Starting the REPL

Run `pars` with no arguments:

```bash
pars
```

You'll see a welcome banner and prompt:

```
█▀█ ▄▀█ █▀█ █▀ █░░ █▀▀ █▄█
█▀▀ █▀█ █▀▄ ▄█ █▄▄ ██▄ ░█░ v 0.15.3

Type 'exit' or Ctrl+D to quit
Use Tab for completion, ↑↓ for history
Type ':help' for REPL commands

>>
```

## Basic Usage

Type expressions at the `>>` prompt:

```
>> 1 + 2
3

>> "hello".toUpper()
"HELLO"

>> [1, 2, 3].map(fn(x) { x * 2 })
[2, 4, 6]
```

Results are displayed in PLN (Parsley Literal Notation) format — strings are quoted, arrays use brackets, etc.

## Variables Persist

Variables defined in one expression are available in subsequent expressions. A
declaration prints `OK` rather than the value it bound:

```
>> let name = "Alice"
OK

>> let age = 30
OK

>> `{name} is {age}`
"Alice is 30"
```

## REPL Commands

Commands start with `:` and are not Parsley code:

| Command | Short | Description |
|---------|-------|-------------|
| `:help` | `:h`, `:?` | Show help for REPL commands |
| `:describe <topic>` | `:d <topic>` | Show documentation for a type, builtin, or module |
| `:env` | | Show all variables in scope |
| `:clear` | | Clear all user-defined variables |
| `:raw` | | Toggle raw output mode |
| `exit`, `quit` | | Exit the REPL |

### :describe

Get documentation for any type, builtin, module, or operator:

```
>> :d string
Type: string

Methods:
  .collapse()              Collapse whitespace to single spaces
  .digits()                Extract only digits
  ...

>> :d JSON
JSON(source, options?)

Load JSON from path or URL

Arity: 1-2
Category: file

>> :d @std/math
Module: @std/math
...
```

### :env

View all variables currently in scope, with their types:

```
>> let x = 10
OK

>> let name = "Bob"
OK

>> :env
  @args: ARRAY = []
  @env: DICTIONARY = {HOME: /Users/you, PATH: ...}
  name: STRING = Bob
  x: INTEGER = 10
```

### :clear

Reset the environment, removing all user-defined variables. The globals `@args`
and `@env` remain:

```
>> :clear
Environment cleared

>> :env
  @args: ARRAY = []
  @env: DICTIONARY = {HOME: /Users/you, PATH: ...}
```

### :raw

Toggle between PLN output and raw output mode:

```
>> "hello"
"hello"

>> :raw
Raw output mode ON (script-style output)

:> "hello"
hello

:> :raw
Raw output mode OFF (Parsley literal output)

>> "hello"
"hello"
```

Notice the prompt changes from `>>` to `:>` in raw mode.

## Output Modes

| Mode | Prompt | String Output | Array Output |
|------|--------|---------------|--------------|
| PLN (default) | `>>` | `"hello"` | `[1, 2, 3]` |
| Raw | `:>` | `hello` | `123` |

PLN mode shows the exact Parsley value — useful for debugging. Raw mode shows output as it would appear when running a script — useful for seeing rendered HTML or concatenated strings.

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `↑` / `↓` | Navigate command history |
| `Tab` | Autocomplete |
| `Ctrl+C` | Cancel current input |
| `Ctrl+D` | Exit REPL |
| `Ctrl+L` | Clear screen |

## History

The REPL remembers your command history and writes it to `.parsley_history` in the system temporary directory on exit, so it is restored in the next session. Use the up and down arrow keys to navigate previous commands.

## Multiline Input

For multiline expressions, the REPL automatically detects incomplete input and waits for more:

```
>> let data = {
..   name: "Alice",
..   age: 30
.. }
OK

>> for (i in 1..3) {
..   i * 10
.. }
[10, 20, 30]
```

The `..` prompt indicates continuation lines.

## Error Handling

Errors are displayed with context but don't exit the REPL:

```
>> 1 / 0
Runtime error: line 1, column 3
  Division by zero
  hint: Check if the divisor is zero before dividing

>> "hello".nonexistent()
Runtime error: line 1, column 8
  Unknown method `nonexistent` for string

>> let x =
..
Parser error: line 1, column 8
  unexpected 'end of file'
```

You can continue entering new expressions after an error.

## Debugging Tips

### Inspect Values

Use `.inspect()` to see type information:

```
>> (42).inspect()
{__type: "integer", value: 42}

>> @now.inspect().__type
"datetime"
```

Some types (dates, money, paths) print back as their own literal form even after
`.inspect()`, so read a field such as `.__type` to see the underlying shape.

### Check Types

Use `describe()` on a value to see its type and available methods:

```
>> describe([1, 2, 3])
"Type: array\n\nMethods:\n  .filter(arg)           - Filter by predicate\n..."
```

### Log Intermediate Values

Use `log()` to print debug output without affecting the result:

```
>> let double = fn(x) { log("input:", x); x * 2 }
OK

>> double(21)
input: 21
42
```

## Limitations

- The REPL runs in a single, flat environment — every line shares one scope
- `:clear` removes user variables but leaves the globals `@args` and `@env` in place
- Relative paths in `import` and file handles resolve against the directory you started `pars` in, not against any script file

## Use Cases

**Quick calculations:**
```
>> 365 * 24 * 60 * 60
31536000
```

**Exploring the API:**
```
>> :d array
>> [1, 2, 3].reverse()
```

**Testing transformations:**
```
>> "Hello World".toSnake()
"hello_world"
```

**Prototyping functions:**
```
>> let greet = fn(name) { `Hello, {name}!` }
>> greet("Alice")
"Hello, Alice!"
```

## See Also

- [CLI](cli.md) — Command-line options and script execution
- [PLN](../pln.md) — Parsley Literal Notation format
- [Getting Started](../getting-started.md) — Introduction to Parsley