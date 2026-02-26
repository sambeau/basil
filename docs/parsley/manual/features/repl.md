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

Variables defined in one expression are available in subsequent expressions:

```
>> let name = "Alice"
"Alice"

>> let age = 30
30

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

View all variables currently in scope:

```
>> let x = 10
10

>> let name = "Bob"
"Bob"

>> :env
name = "Bob"
x = 10
```

### :clear

Reset the environment, removing all user-defined variables:

```
>> :clear
Environment cleared

>> :env
(empty)
```

### :raw

Toggle between PLN output and raw output mode:

```
>> "hello"
"hello"

>> :raw
Raw output mode enabled

:> "hello"
hello

:> :raw
PLN output mode enabled

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

The REPL remembers your command history within a session. Use the up and down arrow keys to navigate previous commands.

## Multiline Input

For multiline expressions, the REPL automatically detects incomplete input and waits for more:

```
>> let data = {
..   name: "Alice",
..   age: 30
.. }
{name: "Alice", age: 30}

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
Runtime error: division by zero

>> "hello".nonexistent()
Runtime error: Unknown method `nonexistent` for string

>> let x =
Parser error: unexpected end of input
```

You can continue entering new expressions after an error.

## Debugging Tips

### Inspect Values

Use `.inspect()` to see type information:

```
>> (42).inspect()
{__type: "integer", value: 42}

>> @now.inspect()
{__type: "datetime", ...}
```

### Check Types

Use `describe()` on a value to see its type and available methods:

```
>> describe([1, 2, 3])
"array with 3 elements\n\nMethods:\n  .filter(arg)..."
```

### Log Intermediate Values

Use `log()` to print debug output without affecting the result:

```
>> let double = fn(x) { log("input:", x); x * 2 }
fn(x)

>> double(21)
input: 21
42
```

## Limitations

- The REPL runs in a single environment — there's no module system or file imports
- Command history is not persisted between sessions
- Some features requiring file context (like relative imports) may not work

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