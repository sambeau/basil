---
id: man-pars-std-markdown
title: "@std/markdown"
system: parsley
type: stdlib
name: "@std/markdown"
created: 2024-12-13
version: 0.1.0
author: "@sam"
keywords: std, markdown, ast, parse, html, toc, documentation
---

# @std/markdown (Deprecated)

> **⚠️ Deprecated:** This module is deprecated in favor of `@std/mdDoc` which provides a cleaner pseudo-type API. See [Migration Guide](#migration-to-stdmddoc) below.

The `@std/markdown` module provides tools for parsing markdown into an AST (Abstract Syntax Tree), manipulating it programmatically, and rendering it back to markdown or HTML.

```parsley
let {md} = import @std/markdown

let doc = md.parse("# Hello\n\nThis is **bold** text.")
print(md.toHTML(doc))
```

## Importing

```parsley
let {md} = import @std/markdown
```

## Methods

### parse()

#### Usage: `md.parse(source)`

Parses a markdown string and returns an AST dictionary representing the document structure. Each node in the AST contains a `type` field and type-specific properties.

```parsley
let {md} = import @std/markdown

let source = `
# Introduction

This is a paragraph with **bold** and *italic* text.

## Getting Started

1. First step
2. Second step
`

let ast = md.parse(source)
print(ast.type)  // "document"
print(ast.children.length())  // number of top-level elements
```

The AST can also be parsed from a file dictionary:

```parsley
let content <== text(@./README.md)
let ast = md.parse(content)
```

### toMarkdown()

#### Usage: `md.toMarkdown(ast)`

Renders an AST back to markdown text. This is useful after programmatically modifying the AST.

```parsley
let {md} = import @std/markdown

let ast = md.parse("# Hello\n\nWorld")
let output = md.toMarkdown(ast)
print(output)
// # Hello
//
// World
```

### toHTML()

#### Usage: `md.toHTML(ast)`

Renders an AST to HTML. Headings automatically include `id` attributes generated from their text for anchor linking.

```parsley
let {md} = import @std/markdown

let ast = md.parse("# Hello World\n\nA paragraph.")
let html = md.toHTML(ast)
print(html)
// <h1 id="hello-world">Hello World</h1>
// <p>A paragraph.</p>
```

## AST Node Types

Each node in the AST is a dictionary with a `type` field. Here are all supported node types:

### Document Structure

| Type | Description | Properties |
|------|-------------|------------|
| `document` | Root node | `children` |
| `heading` | Heading (h1-h6) | `level`, `text`, `id`, `children` |
| `paragraph` | Paragraph | `children` |
| `blockquote` | Block quote | `children` |
| `thematic_break` | Horizontal rule (`---`) | — |

### Inline Content

| Type | Description | Properties |
|------|-------------|------------|
| `text` | Plain text | `value`, `softBreak`, `hardBreak` |
| `emphasis` | Italic or bold | `level` (1=italic, 2=bold), `children` |
| `code_span` | Inline code | `code` |
| `link` | Hyperlink | `url`, `title`, `children` |
| `image` | Image | `url`, `alt`, `title` |
| `autolink` | Auto-detected link | `url`, `protocol` |
| `strikethrough` | ~~Strikethrough~~ | `children` |

### Code Blocks

| Type | Description | Properties |
|------|-------------|------------|
| `code_block` | Indented code | `code` |
| `fenced_code_block` | Fenced code (```) | `language`, `code` |

### Lists

| Type | Description | Properties |
|------|-------------|------------|
| `list` | Ordered or unordered list | `ordered`, `start`, `tight`, `children` |
| `list_item` | List item | `offset`, `children` |
| `task_checkbox` | Task list checkbox | `checked` |

### Tables (GFM)

| Type | Description | Properties |
|------|-------------|------------|
| `table` | Table container | `children` |
| `table_header` | Header row | `children` |
| `table_row` | Body row | `children` |
| `table_cell` | Cell | `alignment`, `children` |

### Raw Content

| Type | Description | Properties |
|------|-------------|------------|
| `html_block` | Block-level HTML | `html` |
| `raw_html` | Inline HTML | `html` |

## Examples

### Generating a Table of Contents

Extract all headings and build a linked TOC:

```parsley
let {md} = import @std/markdown

let source = `
# Introduction

Some intro text.

## Getting Started

### Installation

Install instructions.

### Configuration

Config details.

## Advanced Topics

### Performance

Performance tips.

## Conclusion

Final thoughts.
`

let ast = md.parse(source)

// Recursive function to find all headings
let findHeadings = fn(node, results) {
  if node.type == "heading" {
    results = results ++ [{
      level: node.level,
      text: node.text,
      id: node.id
    }]
  }
  if node.children {
    for child in node.children {
      results = findHeadings(child, results)
    }
  }
  results
}

let headings = findHeadings(ast, [])

// Build markdown TOC
let toc = headings.map(fn(h) {
  let indent = "  ".repeat(h.level - 1)
  indent + "- [" + h.text + "](#" + h.id + ")"
}).join("\n")

print(toc)
```

Output:
```
- [Introduction](#introduction)
  - [Getting Started](#getting-started)
    - [Installation](#installation)
    - [Configuration](#configuration)
  - [Advanced Topics](#advanced-topics)
    - [Performance](#performance)
  - [Conclusion](#conclusion)
```

### Building an HTML TOC Navigation

```parsley
let {md} = import @std/markdown

let ast = md.parse(source)
let headings = findHeadings(ast, [])

let htmlToc = `<nav class="toc"><ul>`
for h in headings {
  htmlToc = htmlToc + `<li class="level-` + h.level + `">`
  htmlToc = htmlToc + `<a href="#` + h.id + `">` + h.text + `</a></li>`
}
htmlToc = htmlToc + `</ul></nav>`

print(htmlToc)
```

### Extracting All Links

Find all links in a document:

```parsley
let {md} = import @std/markdown

let findLinks = fn(node, results) {
  if node.type == "link" {
    results = results ++ [{
      url: node.url,
      title: node.title or ""
    }]
  }
  if node.children {
    for child in node.children {
      results = findLinks(child, results)
    }
  }
  results
}

let doc = md.parse(`
Check out [Parsley](https://parsley.dev) and 
[Basil](https://basil.dev "The Basil Server").
`)

let links = findLinks(doc, [])
for link in links {
  print(link.url)
}
// https://parsley.dev
// https://basil.dev
```

### Extracting Code Blocks by Language

```parsley
let {md} = import @std/markdown

let findCodeBlocks = fn(node, lang, results) {
  if node.type == "fenced_code_block" and node.language == lang {
    results = results ++ [node.code]
  }
  if node.children {
    for child in node.children {
      results = findCodeBlocks(child, lang, results)
    }
  }
  results
}

let doc = md.parse(`
Here's some JavaScript:

\`\`\`javascript
console.log("hello")
\`\`\`

And some Parsley:

\`\`\`parsley
print("hello")
\`\`\`
`)

let parsleyBlocks = findCodeBlocks(doc, "parsley", [])
print(parsleyBlocks)
// ["print(\"hello\")\n"]
```

### Document Title Extraction

Get the first h1 heading as the document title:

```parsley
let {md} = import @std/markdown

let getTitle = fn(node) {
  if node.type == "heading" and node.level == 1 {
    return node.text
  }
  if node.children {
    for child in node.children {
      let title = getTitle(child)
      if title { return title }
    }
  }
  null
}

let doc = md.parse("# My Amazing Document\n\nContent here...")
let title = getTitle(doc)
print(title)  // "My Amazing Document"
```

### Converting Markdown to HTML with Custom Processing

```parsley
let {md} = import @std/markdown

// Parse, inspect, and render
let source = `
# Welcome

This is a **test** document with:

- Item one
- Item two
- Item three

> A blockquote for emphasis.

\`\`\`parsley
let x = 42
\`\`\`
`

let ast = md.parse(source)
let html = md.toHTML(ast)

// Wrap in a template
let page = `<!DOCTYPE html>
<html>
<head><title>Document</title></head>
<body>
` + html + `
</body>
</html>`

print(page)
```

## GFM Extensions

The module supports GitHub Flavored Markdown extensions:

### Tables

```parsley
let {md} = import @std/markdown

let doc = md.parse(`
| Name | Age |
|------|-----|
| Alice | 30 |
| Bob | 25 |
`)

print(md.toHTML(doc))
// <table>
// <thead><tr><th>Name</th><th>Age</th></tr></thead>
// <tr><td>Alice</td><td>30</td></tr>
// <tr><td>Bob</td><td>25</td></tr>
// </table>
```

### Task Lists

```parsley
let {md} = import @std/markdown

let doc = md.parse(`
- [x] Complete task
- [ ] Pending task
- [ ] Another pending
`)

print(md.toHTML(doc))
// Renders with checkbox inputs
```

### Strikethrough

```parsley
let {md} = import @std/markdown

let doc = md.parse("This is ~~deleted~~ text.")
print(md.toHTML(doc))
// <p>This is <del>deleted</del> text.</p>
```

## Tips

1. **Heading IDs are auto-generated** from the heading text, converted to lowercase with spaces replaced by hyphens. Use these for anchor links.

2. **The AST is a regular Parsley dictionary** — you can inspect it with `print()` or iterate over it with standard dictionary/array methods.

3. **Round-trip fidelity**: `md.toMarkdown(md.parse(source))` produces semantically equivalent markdown, though formatting may differ slightly.

4. **Use for documentation systems**: Parse markdown files, extract metadata, build indexes, generate navigation, and render to HTML — all within Parsley.

## Migration to @std/mdDoc

The `@std/mdDoc` module provides a cleaner API where methods are called directly on the document object rather than passing the AST as a parameter.

### Before (deprecated)

```parsley
let {md} = import @std/markdown
let ast = md.parse("# Hello")
md.title(ast)
md.toHTML(ast)
md.headings(ast)
md.findAll(ast, "link")
```

### After (recommended)

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("# Hello")
doc.title()
doc.toHTML()
doc.headings()
doc.findAll("link")
```

Key changes:
- `md.parse(text)` → `mdDoc(text)`
- `md.method(ast, ...)` → `doc.method(...)`
- Transform methods (`map`, `filter`) return new `mdDoc` objects

See the [@std/mdDoc documentation](std-mdDoc.md) for full details.

