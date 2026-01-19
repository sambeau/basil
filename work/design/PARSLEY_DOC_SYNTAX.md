# Parsley Documentation System Design

**Status:** Proposal  
**Date:** 2026-01-19  
**Author:** AI Assistant with human review

## Overview

This document proposes a documentation system for Parsley, inspired by Go's godoc. The system uses `@doc { }` blocks to attach documentation to declarations, and `pars doc` to generate human-readable output.

## Goals

1. **Simple syntax** - Easy to write, minimal ceremony
2. **Readable in source** - Documentation enhances code readability
3. **Machine-parseable** - Enables tooling (doc generation, IDE hints)
4. **Self-documenting schemas** - Schema definitions speak for themselves
5. **Forcing function** - Undocumented items don't appear in generated docs

## Non-Goals

1. Complex markup languages (no full HTML)
2. Inline documentation extraction from comments
3. Type inference documentation (Parsley is dynamic)
4. API documentation for internal/private code

---

## Syntax Design

### Basic Form

```parsley
@doc {
  Documentation content goes here.
  
  Supports a markdown subset for formatting.
}
let myFunction = fn(x, y) {
  ...
}
```

### Placement Rules

1. `@doc` block must immediately precede a declaration
2. Valid targets: `let`, `fn`, `export`, `schema`
3. No blank lines between `@doc` and declaration
4. Multiple `@doc` blocks not allowed (error)

### Valid Examples

```parsley
@doc {
  Calculates the factorial of a number.
  
  Returns 1 for n <= 1.
}
let factorial = fn(n) {
  if n <= 1 { return 1 }
  return n * factorial(n - 1)
}

@doc {
  User schema for the application.
}
schema User {
  id: int(auto, readOnly)
  name: string(required)
  email: string(required, unique)
}

@doc {
  The default greeting message.
}
export let greeting = "Hello, World!"
```

### Invalid Examples

```parsley
// ERROR: blank line between @doc and declaration
@doc { Description }

let x = 1

// ERROR: @doc on non-declaration
@doc { Not valid }
if true { ... }

// ERROR: @doc inside expression
let x = @doc { nope } fn() { }
```

---

## Content Format

### Markdown Subset

The `@doc` content supports a limited markdown subset:

| Feature | Syntax | Supported |
|---------|--------|-----------|
| Paragraphs | Blank lines | ✓ |
| Bold | `**text**` | ✓ |
| Italic | `*text*` | ✓ |
| Code | `` `code` `` | ✓ |
| Code blocks | ``` | ✓ |
| Links | `[text](url)` | ✓ |
| Lists | `- item` | ✓ |
| Numbered lists | `1. item` | ✓ |
| Headings | `## Heading` | ✗ (use paragraphs) |
| Images | `![alt](url)` | ✗ |
| Tables | `| a | b |` | ✗ |
| HTML | `<tag>` | ✗ (escaped) |

### Rationale

- **No headings:** Documentation should be concise; structure comes from code
- **No images:** Documentation is text-focused
- **No tables:** Keep it simple; use code blocks for tabular data
- **No HTML:** Security and simplicity

### Code Blocks

Fenced code blocks with optional language hint:

```parsley
@doc {
  Formats a date for display.
  
  Example:
  ```parsley
  formatDate(now())  // "January 19, 2026"
  ```
}
let formatDate = fn(date) { ... }
```

### No Interpolation

Content is **literal text**, no variable interpolation:

```parsley
let version = "1.0"

@doc {
  Current version is {version}.   // Literal "{version}", NOT "1.0"
}
export let VERSION = version
```

**Rationale:** Documentation should be static and predictable. Dynamic docs would complicate tooling and create confusing behavior.

---

## Semantic Conventions

### First Sentence Rule

The first sentence (up to first period) is the **summary**. Used in:
- Index listings
- IDE hover hints
- Short-form documentation

```parsley
@doc {
  Validates an email address. Returns true if the format is valid,
  false otherwise. Does not verify the domain exists.
}
let validateEmail = fn(email) { ... }
```

Summary: "Validates an email address."

### Parameter Documentation

Use a simple list format:

```parsley
@doc {
  Sends an email to the specified recipient.
  
  - to: The recipient email address
  - subject: Email subject line
  - body: HTML body content
  
  Returns true if sent successfully.
}
let sendEmail = fn(to, subject, body) { ... }
```

### Schema Documentation

Schemas are **self-documenting** through their field definitions and constraints. The `@doc` block provides context, not field descriptions:

```parsley
@doc {
  Represents a blog post in the system.
  
  Posts are created in draft status and must be published
  explicitly via the publish() function.
}
schema Post {
  id: int(auto, readOnly)
  title: string(required, max: 200)
  slug: string(required, unique)           // URL-friendly identifier
  content: text                            // Markdown content
  status: string(default: "draft")         // draft, published, archived
  publishedAt: datetime
  author: User                             // Foreign key to User
  createdAt: datetime(auto)
  updatedAt: datetime(auto)
}
```

**Key insight:** Field comments (`//`) describe individual fields. The `@doc` block describes the schema's purpose and behavior.

---

## Grammar

### Lexer

New token:

```
DOC_BLOCK    @doc { ... }
```

The lexer captures everything between `@doc {` and the matching `}`, handling nested braces.

### Parser

```
docBlock     := "@doc" "{" docContent "}"
docContent   := <any text until matching "}">

declaration  := docBlock? (letStatement | fnStatement | exportStatement | schemaStatement)
```

### AST

```go
type DocBlock struct {
    Content string      // Raw content (whitespace preserved)
    Line    int         // Line number for error reporting
}

type LetStatement struct {
    Doc   *DocBlock     // nil if no documentation
    Name  *Identifier
    Value Expression
}

// Similar for FnStatement, ExportStatement, SchemaStatement
```

---

## Documentation Generation

### `pars doc` Command

```
pars doc [options] <path>

Options:
  --format=md|html|json    Output format (default: md)
  --out=<dir>              Output directory (default: stdout)
  --private                Include non-exported items
  --index                  Generate index file
```

### Output Structure

For a file `handlers/users.pars`:

```
docs/
  handlers/
    users.md           # Documentation for users.pars
  index.md             # Index of all documented items
```

### Markdown Output Example

```markdown
# users.pars

User management handlers.

## Functions

### createUser

```parsley
let createUser = fn(name, email)
```

Creates a new user account.

- name: The user's display name
- email: The user's email address

Returns the created User record.

---

### deleteUser

```parsley
let deleteUser = fn(id)
```

Deletes a user by ID. Requires admin role.

## Schemas

### User

```parsley
schema User {
  id: int(auto, readOnly)
  name: string(required)
  email: string(required, unique)
}
```

Represents a user in the system.
```

### JSON Output (for tooling)

```json
{
  "file": "handlers/users.pars",
  "doc": "User management handlers.",
  "items": [
    {
      "name": "createUser",
      "kind": "function",
      "signature": "fn(name, email)",
      "summary": "Creates a new user account.",
      "doc": "Creates a new user account.\n\n- name: The user's display name\n- email: The user's email address\n\nReturns the created User record.",
      "line": 15,
      "exported": true
    }
  ]
}
```

---

## File-Level Documentation

A `@doc` block at the top of a file (before any declarations) documents the file itself:

```parsley
@doc {
  User management handlers.
  
  This module provides CRUD operations for user accounts,
  including authentication and profile management.
}

import { db } from "@/lib/database"

@doc {
  Creates a new user account.
}
export let createUser = fn(name, email) {
  ...
}
```

---

## Edge Cases

### Multiple Exports

When re-exporting, documentation attaches to the export:

```parsley
@doc {
  Re-exported utility functions.
}
export { formatDate, parseDate } from "./dates"
```

### Computed Exports

Documentation is optional but allowed:

```parsley
@doc {
  Dynamic configuration based on environment.
}
export computed config
```

### Conditional Declarations

Documentation attaches to the declaration, not the condition:

```parsley
// This works
@doc { Production logger. }
let logger = if ENV == "prod" {
  createProdLogger()
} else {
  createDevLogger()
}
```

### Anonymous Functions

Cannot be documented directly (no declaration to attach to):

```parsley
// Documentation goes on the binding, not the function
@doc { Event handler for clicks. }
let onClick = fn(event) { ... }

// Not valid - no target
items.map(@doc { nope } fn(x) { x * 2 })
```

---

## Implementation Plan

### Phase 1: Lexer & Parser
1. Add `DOC_BLOCK` token to lexer
2. Handle nested brace counting
3. Add `DocBlock` AST node
4. Attach to declaration nodes
5. Parser tests

### Phase 2: `pars doc` Command
1. Add subcommand infrastructure to `pars`
2. Implement doc extraction from AST
3. Markdown output generator
4. Basic CLI interface

### Phase 3: Enhanced Output
1. HTML output format
2. JSON output format
3. Index generation
4. Cross-reference linking

### Phase 4: Integration
1. IDE hover hints (LSP)
2. REPL `help()` function
3. Online documentation site

---

## Examples

### Library Module

```parsley
@doc {
  String manipulation utilities.
  
  Provides common string operations not available in the
  standard library.
}

@doc {
  Capitalizes the first letter of each word.
  
  Example:
  ```parsley
  titleCase("hello world")  // "Hello World"
  ```
}
export let titleCase = fn(s) {
  s.split(" ")
   .map(fn(word) { word[0].upper() + word[1:] })
   .join(" ")
}

@doc {
  Truncates a string to the specified length.
  
  - s: The string to truncate
  - maxLen: Maximum length (default: 100)
  - suffix: Truncation indicator (default: "...")
  
  If the string is shorter than maxLen, returns it unchanged.
}
export let truncate = fn(s, maxLen = 100, suffix = "...") {
  if s.len() <= maxLen { return s }
  return s[0:maxLen - suffix.len()] + suffix
}
```

### Handler Module

```parsley
@doc {
  Blog post API handlers.
}

import { Post } from "@/schemas/post"
import { auth } from "@/lib/auth"

@doc {
  Lists all published posts.
  
  Supports pagination via `page` and `limit` query parameters.
  Returns posts ordered by publish date, newest first.
}
export let listPosts = fn() {
  let page = params.page ?? 1
  let limit = params.limit ?? 20
  
  Post.where(status: "published")
      .order(publishedAt: "desc")
      .paginate(page, limit)
}

@doc {
  Creates a new blog post.
  
  Requires authentication. The post is created in draft status.
}
export let createPost = fn() {
  auth.require()
  
  Post.create(
    title: form.title,
    content: form.content,
    author: @user,
    status: "draft"
  )
}
```

---

## Open Questions

1. **Should we support `@param` and `@returns` annotations?**
   - Pro: Structured, tooling-friendly
   - Con: More syntax to learn, Go manages without them
   - Recommendation: No, use prose lists

2. **Should undocumented exports appear in generated docs?**
   - Pro: Complete API surface
   - Con: Encourages documentation
   - Recommendation: No, forcing function for docs

3. **Should `@doc` support escape sequences for `}`?**
   - Use case: Documentation containing `}`
   - Option A: `\}` escape
   - Option B: Count braces (current proposal)
   - Recommendation: Brace counting (simpler)

4. **Should we support `@example` blocks separately?**
   - Pro: Structured examples, could be tested
   - Con: More complexity
   - Recommendation: No, use code blocks in `@doc`

5. **Maximum doc length?**
   - Recommendation: No hard limit, but lint warning over 500 words

---

## References

- [Go Doc Comments](https://go.dev/doc/comment)
- [Rust Documentation](https://doc.rust-lang.org/rustdoc/)
- [JSDoc](https://jsdoc.app/)
- [Python Docstrings](https://peps.python.org/pep-0257/)
