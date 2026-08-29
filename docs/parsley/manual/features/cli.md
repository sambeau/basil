---
id: man-pars-cli
title: Command Line Interface
system: parsley
type: features
name: cli
created: 2026-02-26
version: 0.2.0
author: Basil Team
keywords:
  - cli
  - command line
  - pars
  - interpreter
  - repl
  - eval
  - format
  - describe
---

# Command Line Interface

The `pars` command is the Parsley interpreter. It can run scripts, evaluate expressions, format code, and provide introspection into the language.

## Quick Reference

```bash
pars                      # Start interactive REPL
pars script.pars          # Run a script
pars -e "1 + 2"           # Evaluate expression
pars fmt script.pars      # Format code
pars describe string      # Show type documentation
```

## Running Scripts

Run a `.pars` file by passing it as an argument:

```bash
pars myscript.pars
```

The script's output (the value of its last expression) is printed to stdout.

### Command-Line Arguments

Pass arguments after the script name. Access them via the [`@args` global](../fundamentals/globals.md#args):

```bash
pars greet.pars Alice Bob
```

```parsley
// greet.pars
let names = @args           // ["Alice", "Bob"]
for (name in names) {
    `Hello, {name}!`
}
```

## Evaluating Expressions

The `-e` or `--eval` flag evaluates a code string:

```bash
pars -e "1 + 2"             # 3
pars -e "[1, 2, 3]"         # [1, 2, 3]
pars -e "{a: 1, b: 2}"      # {a: 1, b: 2}
```

Output is in PLN (Parsley Literal Notation) format by default — strings are quoted, arrays use brackets, etc. This is useful for debugging and seeing exact values.

### Raw Output Mode

Use `-r` or `--raw` for script-like output (unquoted strings, concatenated arrays):

```bash
pars -e '"hello"'           # "hello" (PLN format)
pars -e '"hello"' --raw     # hello (raw output)
pars -e '[1, 2, 3]' --raw   # 123 (concatenated)
```

### Arguments with -e

Pass arguments after the expression:

```bash
pars -e '@args' foo bar     # ["foo", "bar"]
```

## Syntax Checking

The `-c` or `--check` flag validates syntax without executing:

```bash
pars -c script.pars         # Check one file
pars -c *.pars              # Check multiple files
```

Returns exit code 0 if all files are valid, non-zero if any have errors.

## Formatting Code

The `fmt` subcommand formats Parsley source code:

```bash
pars fmt script.pars        # Print formatted code to stdout
pars fmt -w script.pars     # Format in place (overwrite file)
pars fmt *.pars             # Format multiple files (stdout)
pars fmt -w *.pars          # Format multiple files in place
```

## Introspection with `describe`

The `describe` subcommand shows documentation for types, builtins, modules, and operators:

```bash
pars describe string        # String type and methods
pars describe array         # Array type and methods
pars describe integer       # Integer type and methods
pars describe float         # Float type and methods

pars describe builtins      # List all builtin functions
pars describe operators     # List all operators
pars describe types         # List all types

pars describe @std/math     # Module documentation
pars describe JSON          # Builtin function documentation
```

### JSON Output

Add `--json` for machine-readable output:

```bash
pars describe string --json
pars describe builtins --json
pars describe all --json    # Complete API schema
```

The `--json` output is useful for tooling, IDE integrations, and AI agents.

## Display Options

| Flag | Description |
|------|-------------|
| `-h`, `--help` | Show help message |
| `-v`, `-V`, `--version` | Show version information |
| `-pp`, `--pretty` | Pretty-print HTML output with indentation |
| `-q`, `--quiet` | Suppress hints and non-essential output |
| `--format <fmt>` | Output format: `text` (default), `json`, `pln` |
| `--machine` | Machine-readable JSON output (for AI/scripts) |

### Pretty-Printing HTML

When a script outputs HTML, use `-pp` to format it readably:

```bash
pars -pp page.pars          # Indented HTML output
```

### Machine Output

The `--machine` flag outputs JSON suitable for programmatic consumption:

```bash
pars --machine -e '1 + 2'
```

```json
{
  "ok": true,
  "type": "integer",
  "value": 3
}
```

## Security Options

Control file system and command execution access:

| Flag | Description |
|------|-------------|
| `--no-read` | Deny all file reads |
| `--no-write` | Deny all file writes |
| `--restrict-read=PATHS` | Deny reads from specific paths |
| `--restrict-write=PATHS` | Deny writes to specific paths |
| `--allow-execute=PATHS` | Allow executing binaries from paths |
| `--allow-execute-all`, `-x` | Allow all command execution |

### Examples

```bash
# Run script with no file write access
pars --no-write script.pars

# Restrict reads from /etc
pars --restrict-read=/etc script.pars

# Allow command execution
pars -x script.pars
pars --allow-execute=/usr/bin script.pars
```

By default, file reads and writes are allowed, but command execution is restricted. In production (Basil server), the security policy controls these permissions.

## Interactive REPL

Running `pars` with no arguments starts the interactive REPL:

```bash
pars
```

See [REPL](repl.md) for details on REPL commands and features.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (parse, runtime, or I/O) when running a script; syntax errors under `-c` |
| 2 | Parse error under `-e`; a file `-c` could not read |

## Examples

```bash
# Run a web page handler and pretty-print
pars -pp routes/index.pars

# Quick calculation
pars -e "365 * 24 * 60"     # Minutes in a year

# Process JSON from stdin
echo '{"name": "Alice"}' | pars -e 'let data <== JSON(@/dev/stdin); data.name'

# Validate all Parsley files in a project
pars -c **/*.pars

# Get method list for arrays as JSON
pars describe array --json | jq '.methods'

# Format all files in src/
pars fmt -w src/*.pars
```

## See Also

- [Globals](../fundamentals/globals.md) — `@args` and `@env`
- [REPL](repl.md) — Interactive mode commands
- [Security Model](security.md) — File and execution permissions
- [PLN](../pln.md) — Parsley Literal Notation format