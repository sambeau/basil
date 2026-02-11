---
id: man-pars-index
title: "Parsley Language Manual"
system: parsley
type: tutorial
name: index
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - manual
  - index
  - table of contents
  - reference
  - guide
---

# Parsley Language Manual

The complete reference for the Parsley programming language. Parsley is a modern, expression-oriented language designed for building web applications with the [Basil](../../guide/README.md) server.

> **New to Parsley?** Start with the [Getting Started Tutorial](getting-started.md) for a hands-on introduction, then explore the topics below.

---

## Language Fundamentals

Core language concepts — start here to learn how Parsley works.

| Page | Description |
|------|-------------|
| [Comments](fundamentals/comments.md) | Single-line and block comments |
| [Variables & Binding](fundamentals/variables.md) | `let`, destructuring, scope, and reassignment |
| [Types](fundamentals/types.md) | All types, coercion rules, `typeof`, and `is` |
| [Operators](fundamentals/operators.md) | Arithmetic, comparison, logical, and special operators |
| [Functions](fundamentals/functions.md) | Function expressions, closures, destructuring, and `this` binding |
| [Control Flow](fundamentals/control-flow.md) | `if`/`else`, `for`, `while`, and `check` |
| [Error Handling](fundamentals/errors.md) | `try`/`catch`, `check`, `fail`, and error objects |
| [Modules](fundamentals/modules.md) | `import`, `export`, and module resolution |
| [Tags](fundamentals/tags.md) | HTML/XML tag syntax, attributes, children, and components |
| [Data Model](fundamentals/data-model.md) | Schemas, records, and tables — how structured data fits together |

## Built-in Types

Reference pages for each of Parsley's built-in types.

| Page | Description |
|------|-------------|
| [Booleans & Null](builtins/booleans.md) | `true`, `false`, `null`, and truthiness |
| [Numbers](builtins/numbers.md) | Integers, floats, arithmetic, and formatting |
| [Strings](builtins/strings.md) | String literals, interpolation, methods, and formatting |
| [Arrays](builtins/array.md) | Ordered collections with map, filter, reduce, sort, and more |
| [Dictionaries](builtins/dictionary.md) | Key-value pairs, destructuring, `this` binding, and iteration |
| [Schemas](builtins/schema.md) | Declaring data shapes with types, constraints, and metadata |
| [Records](builtins/record.md) | Schema-bound dictionaries with validation and form binding |
| [Tables](builtins/table.md) | Typed tabular data with query, aggregation, and export methods |
| [Regex](builtins/regex.md) | Regular expressions — literals, operators, and methods |
| [Paths](builtins/paths.md) | File path literals, interpolation, properties, and methods |
| [URLs](builtins/urls.md) | URL literals, interpolation, properties, and methods |
| [DateTime](builtins/datetime.md) | Date and time values, formatting, and arithmetic |
| [Duration](builtins/duration.md) | Time durations and date arithmetic |
| [Money](builtins/money.md) | Currency values, arithmetic, and formatting |

## Features

Parsley's I/O and integration capabilities.

| Page | Description |
|------|-------------|
| [File I/O](features/file-io.md) | Reading and writing files with operators and handles |
| [Database](features/database.md) | Database connections, SQL tag, transactions, and table bindings |
| [Query DSL](features/query-dsl.md) | Declarative queries — `@query`, `@insert`, `@update`, `@delete` |
| [Data Formats](features/data-formats.md) | Parsing and generating Markdown, CSV, and JSON |
| [HTTP & Networking](features/network.md) | `fetch` operator, HTTP methods, and SFTP |
| [Shell Commands](features/commands.md) | `@shell` and the execute operator |
| [Security Model](features/security.md) | Security policy, file restrictions, and injection prevention |
| [PLN](pln.md) | Parsley Literal Notation — safe data serialization format |

## Standard Library

Importable modules providing higher-level functionality.

| Module | Description |
|--------|-------------|
| [@std/math](stdlib/math.md) | Constants, rounding, statistics, random, trigonometry, and geometry |
| [@std/valid](stdlib/valid.md) | Validation predicates for types, strings, numbers, and formats |
| [@std/id](stdlib/id.md) | ID generation — ULID, UUID v4/v7, NanoID, CUID |
| [@std/table](stdlib/table.md) | SQL-like data manipulation for arrays of dictionaries |
| [@std/api](stdlib/api.md) | Auth wrappers, error helpers, and redirect for Basil handlers |
| [@std/mdDoc](stdlib/mddoc.md) | Markdown document analysis — headings, links, TOC, and transforms |
| [@std/dev](stdlib/dev.md) | Development logging utilities for Basil handlers |
| [@std/session](stdlib/session.md) | Session management — key-value storage and flash messages |

---

## Quick Reference by Topic

### Working with Text
[Strings](builtins/strings.md) · [Regex](builtins/regex.md) · [Data Formats](features/data-formats.md) · [@std/valid](stdlib/valid.md)

### Working with Data
[Arrays](builtins/array.md) · [Dictionaries](builtins/dictionary.md) · [Tables](builtins/table.md) · [@std/table](stdlib/table.md) · [Schemas](builtins/schema.md) · [Records](builtins/record.md) · [Data Model](fundamentals/data-model.md)

### Building Web Pages
[Tags](fundamentals/tags.md) · [Modules](fundamentals/modules.md) · [Strings](builtins/strings.md) (interpolation) · [Control Flow](fundamentals/control-flow.md) (for loops)

### Database Access
[Database](features/database.md) · [Query DSL](features/query-dsl.md) · [Schemas](builtins/schema.md) · [Tables](builtins/table.md)

### Building APIs
[@std/api](stdlib/api.md) · [HTTP & Networking](features/network.md) · [@std/session](stdlib/session.md) · [Error Handling](fundamentals/errors.md)

### Files & External Systems
[File I/O](features/file-io.md) · [Paths](builtins/paths.md) · [URLs](builtins/urls.md) · [HTTP & Networking](features/network.md) · [Shell Commands](features/commands.md) · [PLN](pln.md)

### Dates, Numbers & Money
[Numbers](builtins/numbers.md) · [DateTime](builtins/datetime.md) · [Duration](builtins/duration.md) · [Money](builtins/money.md) · [@std/math](stdlib/math.md)

---

## Suggested Reading Order

If you're learning Parsley from scratch, we recommend this path:

1. **[Getting Started](getting-started.md)** — Your first Parsley program
2. **[Variables](fundamentals/variables.md)** and **[Types](fundamentals/types.md)** — The basics
3. **[Strings](builtins/strings.md)** and **[Arrays](builtins/array.md)** — Working with data
4. **[Functions](fundamentals/functions.md)** and **[Control Flow](fundamentals/control-flow.md)** — Logic and structure
5. **[Tags](fundamentals/tags.md)** — Building HTML
6. **[Modules](fundamentals/modules.md)** — Organizing code
7. **[Error Handling](fundamentals/errors.md)** — Robust programs
8. **Features** — Pick the topics relevant to your project
