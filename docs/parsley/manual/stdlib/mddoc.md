---
id: man-pars-std-mddoc
title: "@std/mdDoc"
system: parsley
type: stdlib
name: mddoc
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - markdown
  - document
  - parsing
  - headings
  - links
  - images
  - code blocks
  - AST
  - table of contents
---

# @std/mdDoc

Markdown document analysis and manipulation. Parse a Markdown string into a queryable document object that provides structured access to headings, links, images, code blocks, and the full AST.

```parsley
let mdDoc = import @std/mdDoc
```

## Constructor

| Function | Args | Returns | Description |
|---|---|---|---|
| `mdDoc(markdown)` | string | MdDoc | Parse a Markdown string into a document object |

```parsley
let doc = mdDoc.mdDoc("# Hello\n\nSome **bold** text.")
```

## Rendering Methods

| Method | Returns | Description |
|---|---|---|
| `.toMarkdown()` | string | Render back to Markdown (reformatted) |
| `.toHTML()` | string | Render to HTML |

```parsley
let doc = mdDoc.mdDoc("# Hello\n\nA paragraph.")
doc.toHTML()                     // "<h1 id=\"hello\">Hello</h1>\n<p>A paragraph.</p>\n"
doc.toMarkdown()                 // "# Hello\n\nA paragraph.\n"
```

## Query Methods

| Method | Args | Returns | Description |
|---|---|---|---|
| `.findAll(type)` | string or array | array | Find all nodes of the given type(s) |
| `.findFirst(type)` | string | node or null | Find the first node of the given type |
| `.headings()` | none | array | All headings as `{level, text, id}` |
| `.links()` | none | array | All links as `{url, title, text}` |
| `.images()` | none | array | All images as `{url, alt, title}` |
| `.codeBlocks()` | none | array | All code blocks as `{language, code}` |

```parsley
let markdown = `# Welcome

This has [a link](https://example.com) and ![an image](photo.png "A photo").

## Section One

Some content here.

## Section Two

More content.
`

let doc = mdDoc.mdDoc(markdown)

doc.headings()
// [
//   {level: 1, text: "Welcome", id: "welcome"},
//   {level: 2, text: "Section One", id: "section-one"},
//   {level: 2, text: "Section Two", id: "section-two"}
// ]

doc.links()
// [{url: "https://example.com", title: "", text: "a link"}]

doc.images()
// [{url: "photo.png", alt: "an image", title: "A photo"}]
```

### Code Blocks

```parsley
let md = "# Code\n\n```parsley\nlet x = 42\n```\n\n```js\nconsole.log(1)\n```"
let doc = mdDoc.mdDoc(md)

doc.codeBlocks()
// [
//   {language: "parsley", code: "let x = 42\n"},
//   {language: "js", code: "console.log(1)\n"}
// ]
```

## Convenience Methods

| Method | Returns | Description |
|---|---|---|
| `.title()` | string or null | Text of the first h1 heading |
| `.toc()` | array | Table of contents entries |
| `.text()` | string | Plain text content (all markup stripped) |
| `.wordCount()` | integer | Word count of plain text content |

```parsley
let doc = mdDoc.mdDoc("# My Doc\n\nThis is a short document.\n\n## Sub-section\n\nMore words here.")

doc.title()                      // "My Doc"
doc.wordCount()                  // 9
doc.text()                       // "My Doc This is a short document. Sub-section More words here."

doc.toc()
// [{level: 1, text: "My Doc", id: "my-doc"}, {level: 2, text: "Sub-section", id: "sub-section"}]
```

## Transform Methods

| Method | Args | Returns | Description |
|---|---|---|---|
| `.walk(fn)` | function | null | Visit each AST node |
| `.map(fn)` | function | MdDoc | Transform nodes, return new document |
| `.filter(fn)` | function | MdDoc | Keep only nodes matching predicate |

```parsley
// Count all nodes
let count = 0
doc.walk(fn(node) {
    count = count + 1
})

// Remove all images
let noImages = doc.filter(fn(node) {
    node.type != "image"
})
```

## AST Access

The `.ast` property exposes the raw abstract syntax tree as a dictionary:

```parsley
let doc = mdDoc.mdDoc("# Hello\n\nA paragraph.")
doc.ast                          // {type: "document", children: [...]}
```

The AST structure follows the CommonMark specification. Each node has a `type` field and type-specific properties. Use `.findAll()` and `.findFirst()` for structured queries instead of walking the AST manually.

## Common Patterns

### Generate Table of Contents

```parsley
let doc = mdDoc.mdDoc(content)
let toc = for (h in doc.toc()) {
    let indent = "  " * (h.level - 1)
    `{indent}- [{h.text}](#{h.id})`
}
toc.join("\n")
```

### Extract All External Links

```parsley
let doc = mdDoc.mdDoc(content)
let external = for (link in doc.links()) {
    if (link.url.includes("://")) { link }
}
```

### Documentation Index

```parsley
let files = fileList(@./docs, "*.md")
for (f in files) {
    let content <== text(f)
    let doc = mdDoc.mdDoc(content)
    {file: f, title: doc.title(), words: doc.wordCount()}
}
```

## See Also

- [Data Formats](../features/data-formats.md) — Markdown parsing with `.parseMarkdown()`
- [Strings](../builtins/strings.md) — string methods
- [File I/O](../features/file-io.md) — reading Markdown files with `markdown()` and `MD()` handles