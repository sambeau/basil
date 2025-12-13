---
id: man-pars-std-mddoc
title: "@std/mdDoc"
system: parsley
type: stdlib
name: "@std/mdDoc"
created: 2024-12-13
version: 0.15.3
author: "@sam"
keywords: std, markdown, ast, parse, html, toc, documentation, mdDoc
---

# @std/mdDoc

The `@std/mdDoc` module provides the `mdDoc` pseudo-type for parsing markdown into an AST (Abstract Syntax Tree), manipulating it programmatically, and rendering it back to markdown or HTML. 

An `mdDoc` wraps a markdown document's AST and provides convenient methods for querying and transforming the document. This is useful for generating tables of contents, extracting headings, transforming documents, and building documentation systems.

```parsley
let {mdDoc} = import @std/mdDoc

let doc = mdDoc("# Hello\n\nThis is **bold** text.")
doc.title()    // "Hello"
doc.toHTML()   // "<h1 id="hello">Hello</h1>\n<p>This is <strong>bold</strong> text.</p>\n"
```

## Importing

```parsley
let {mdDoc} = import @std/mdDoc
```

## Constructor

### mdDoc()

#### Usage: `mdDoc(source)` or `mdDoc(astDict)`

Creates an `mdDoc` from a markdown string or wraps an existing AST dictionary.

```parsley
let {mdDoc} = import @std/mdDoc

// From markdown text
let doc = mdDoc("# Welcome\n\nHello world!")

// From an existing AST dictionary  
let ast = {type: "document", children: [...]}
let doc2 = mdDoc(ast)
```

## Methods

### ast()

#### Usage: `doc.ast()`

Returns the underlying AST dictionary. Useful when you need direct access to the raw AST structure.

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("# Hello")
doc.ast()  // {type: "document", children: [...]}
```

### codeBlocks()

#### Usage: `doc.codeBlocks()`

Extracts all code blocks with their metadata.

```parsley
let {mdDoc} = import @std/mdDoc

let source = `
Here is some code:

\`\`\`go
func main() {}
\`\`\`
`

let doc = mdDoc(source)
doc.codeBlocks()  // [{language: "go", code: "func main() {}\n"}]
```

### filter()

#### Usage: `doc.filter(fn)`

Creates a new `mdDoc` containing only nodes where the predicate function returns true.

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("# Keep This\n\nRemove paragraph\n\n## Keep This Too")

// Keep only headings
let headingsOnly = doc.filter(fn(node) {
    node.type == "document" || node.type == "heading"
})
```

### findAll()

#### Usage: `doc.findAll(type)` or `doc.findAll([types])`

Finds all nodes of the given type(s) in the document.

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("# Title\n\nParagraph\n\n## Subtitle")

doc.findAll("heading")  // Returns array of all heading nodes
doc.findAll(["heading", "paragraph"])  // Returns headings and paragraphs
```

### findFirst()

#### Usage: `doc.findFirst(type)`

Finds the first node of the given type, or `null` if not found.

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("Some text\n\n# First Heading\n\n## Second")

doc.findFirst("heading")  // Returns the first heading node
```

### headings()

#### Usage: `doc.headings()`

Extracts all headings with their metadata.

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("# Title\n\n## Chapter 1\n\n## Chapter 2")

doc.headings()
// [
//   {level: 1, text: "Title", id: "title"},
//   {level: 2, text: "Chapter 1", id: "chapter-1"},
//   {level: 2, text: "Chapter 2", id: "chapter-2"}
// ]
```

### images()

#### Usage: `doc.images()`

Extracts all images with their metadata.

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("![Alt text](image.png \"Title\")")

doc.images()  // [{url: "image.png", alt: "Alt text", title: "Title"}]
```

### links()

#### Usage: `doc.links()`

Extracts all links with their metadata.

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("Check out [Basil](https://example.com \"A framework\")")

doc.links()  // [{url: "https://example.com", title: "A framework", text: "Basil"}]
```

### map()

#### Usage: `doc.map(fn)`

Transforms nodes by applying a function. Returns a new `mdDoc` with transformed nodes.

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("# hello world")

// Capitalize heading text
let transformed = doc.map(fn(node) {
    if node.type == "heading" {
        {type: node.type, level: node.level, text: upper(node.text), id: node.id, children: node.children}
    } else {
        node
    }
})
```

### text()

#### Usage: `doc.text()`

Extracts all plain text content from the document, stripping formatting.

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("# Hello\n\nThis is **bold** text.")

doc.text()  // "Hello This is bold text."
```

### title()

#### Usage: `doc.title()`

Returns the document title (first h1 heading), or `null` if none exists.

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("# My Document\n\nSome content...")

doc.title()  // "My Document"
```

### toc()

#### Usage: `doc.toc()` or `doc.toc({minLevel: n, maxLevel: n})`

Generates a table of contents from the document headings.

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("# Title\n\n## Chapter 1\n\n### Section 1.1\n\n## Chapter 2")

doc.toc()
// [
//   {level: 1, text: "Title", id: "title", indent: 0},
//   {level: 2, text: "Chapter 1", id: "chapter-1", indent: 1},
//   {level: 3, text: "Section 1.1", id: "section-11", indent: 2},
//   {level: 2, text: "Chapter 2", id: "chapter-2", indent: 1}
// ]

// Filter to only h2 and h3
doc.toc({minLevel: 2, maxLevel: 3})
```

### toHTML()

#### Usage: `doc.toHTML()`

Renders the document to HTML.

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("# Hello\n\nWorld")

doc.toHTML()  // "<h1 id=\"hello\">Hello</h1>\n<p>World</p>\n"
```

### toMarkdown()

#### Usage: `doc.toMarkdown()`

Renders the document back to markdown text.

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("# Hello\n\n**Bold** text")

doc.toMarkdown()  // "# Hello\n\n**Bold** text"
```

### walk()

#### Usage: `doc.walk(fn)`

Walks the document tree, calling the function on each node. Return value is ignored.

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("# Hello\n\n## World")

doc.walk(fn(node) {
    if node.type == "heading" {
        print("Found heading:", node.text)
    }
})
```

### wordCount()

#### Usage: `doc.wordCount()`

Returns the word count of the document.

```parsley
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("# Hello World\n\nThis is a test document.")

doc.wordCount()  // 7
```

## AST Node Types

The mdDoc AST uses these node types:

| Type | Description | Properties |
|------|-------------|------------|
| `document` | Root node | `children` |
| `heading` | Heading (h1-h6) | `level`, `text`, `id`, `children` |
| `paragraph` | Paragraph | `children` |
| `text` | Plain text | `value`, `softBreak`, `hardBreak` |
| `emphasis` | Italic or bold | `level` (1=italic, 2=bold), `children` |
| `code_span` | Inline code | `code` |
| `link` | Hyperlink | `url`, `title`, `children` |
| `image` | Image | `url`, `alt`, `title` |
| `code_block` | Indented code | `code` |
| `fenced_code_block` | Fenced code | `language`, `code` |
| `list` | List | `ordered`, `tight`, `start`, `children` |
| `list_item` | List item | `offset`, `children` |
| `blockquote` | Block quote | `children` |
| `thematic_break` | Horizontal rule | - |
| `table` | GFM table | `children` |
| `table_header` | Table header row | `children` |
| `table_row` | Table row | `children` |
| `table_cell` | Table cell | `alignment`, `children` |
| `strikethrough` | GFM strikethrough | `children` |
| `task_checkbox` | GFM task checkbox | `checked` |

## Migration from @std/markdown

If you were using `@std/markdown`, here's how to migrate:

```parsley
// Old way
let {md} = import @std/markdown
let ast = md.parse("# Hello")
md.title(ast)
md.toHTML(ast)

// New way
let {mdDoc} = import @std/mdDoc
let doc = mdDoc("# Hello")
doc.title()
doc.toHTML()
```

The key differences:
- Methods are now called on the `mdDoc` object, not passed the AST as first argument
- The constructor `mdDoc()` replaces `md.parse()`
- Transform methods (`map`, `filter`) return new `mdDoc` objects
