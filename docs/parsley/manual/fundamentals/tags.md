---
id: man-pars-tags
title: Tags
system: parsley
type: fundamentals
name: tags
created: 2026-02-06
version: 0.2.0
author: Basil Team
keywords:
  - tag
  - html
  - xml
  - component
  - attribute
  - spread
  - self-closing
  - form binding
  - SQL tag
  - Cache tag
  - Part tag
---

# Tags

Tags are first-class syntax in Parsley — not strings. They render to HTML and are the primary way to build web pages. Unlike JSX, attribute values don't need `{...}` wrappers for simple strings, and text content inside tags must be quoted.

## Self-Closing Tags

Self-closing tags **must** use `/>`. Omitting the slash is a parse error:

```parsley
<br/>
<hr/>
<img src="photo.jpg" alt="A photo"/>
<input type="text" name="email"/>
```

> ⚠️ `<br>` is invalid in Parsley. Always write `<br/>`. This applies to all void elements (`img`, `input`, `link`, `meta`, etc.).

## Pair Tags

Opening tag, content, closing tag. Text content must be quoted — unquoted words are treated as variable references:

```parsley
<p>"Hello, World!"</p>          // literal string
<h1>"Welcome"</h1>              // literal string

let message = "Dynamic content"
<p>message</p>                   // variable reference → <p>Dynamic content</p>
```

## Attributes

### String Attributes

Simple string values — no braces needed:

```parsley
<div class="container">"Content"</div>
<a href="/about">"About Us"</a>
```

### Expression Attributes

Wrap expressions in `{...}`:

```parsley
let cls = "active"
<div class={cls}>"Content"</div>
// <div class="active">Content</div>

let count = 5
<div data-count={count}>"Items"</div>
// <div data-count="5">Items</div>

<div class={"item-" + toString(count)}>"test"</div>
// <div class="item-5">test</div>
```

### Boolean Attributes

Pass `true` to include an attribute, `false` to omit it entirely:

```parsley
<input type="text" required={true}/>
// <input type="text" required />

<input type="text" required={false}/>
// <input type="text" />
```

### Spread Attributes

Expand a dictionary into attributes with `...`:

```parsley
let attrs = {class: "btn", id: "submit"}
<button ...attrs>"Submit"</button>
// <button class="btn" id="submit">Submit</button>
```

## Content

Tag content can be any mix of literal strings, variables, expressions, and nested tags.

### Literal Text

Must be quoted:

```parsley
<p>"This is literal text."</p>
```

### Variables and Expressions

Unquoted identifiers are variable lookups. Method calls and expressions work too:

```parsley
let name = "alice"
<span>name</span>               // <span>alice</span>
<span>name.toTitle()</span>      // <span>Alice</span>
```

### Control Flow

`if`, `for`, `let`, and function calls all work inside tags:

```parsley
let items = ["apple", "banana", "cherry"]
<ul>
    for (item in items) {
        <li>item</li>
    }
</ul>
// <ul><li>apple</li><li>banana</li><li>cherry</li></ul>
```

Conditional rendering:

```parsley
let show = true
<div>
    if (show) {
        <p>"visible"</p>
    }
</div>
// <div><p>visible</p></div>
```

Conditional attributes:

```parsley
let active = true
<div class={if (active) "on" else "off"}>"test"</div>
// <div class="on">test</div>
```

## Script and Style Tags

Content inside `<script>` and `<style>` tags uses **raw string rules** — the same rules as single-quoted strings (`'...'`). Braces `{` and `}` are **literal characters**, and interpolation uses `@{expr}`. This is by design: CSS and JavaScript both use `{` `}` as core syntax, so treating them as literal avoids conflicts.

```parsley
<style>
    .card { border: 1px solid #ccc; }
    .card:hover { background: lightblue; }
</style>

<script>
    document.querySelectorAll(".tab").forEach(function(el) {
        el.addEventListener("click", function() { el.classList.toggle("active"); });
    });
</script>
```

Use `@{expr}` to interpolate dynamic values:

```parsley
let accent = "tomato"
<style>
    .highlight { color: @{accent}; }
</style>

let endpoint = "/api/data"
<script>
    fetch("@{endpoint}").then(function(r) { return r.json(); });
</script>
```

> ⚠️ **`{` is literal inside `<script>` and `<style>`** — don't use `{expr}` (template string syntax) here. Use `@{expr}` (raw string syntax) instead. See [Strings](../builtins/strings.md) for more on the three string types.

## Nested Tags

Tags nest naturally:

```parsley
<div class="card">
    <h2>"Title"</h2>
    <p>"Body text"</p>
</div>
```

A realistic table example with loops:

```parsley
let rows = [{name: "Alice", age: 30}, {name: "Bob", age: 25}]
<table>
    <thead>
        <tr>
            for (k, _ in rows[0]) {
                <th>k.toTitle()</th>
            }
        </tr>
    </thead>
    <tbody>
        for (row in rows) {
            <tr>
                for (_, v in row) {
                    <td>v</td>
                }
            </tr>
        }
    </tbody>
</table>
```

## Components

Components are functions that return tags. Uppercase names distinguish components from standard HTML tags.

### Self-Closing Component

Props are passed as a dictionary:

```parsley
let Card = fn(props) {
    <div class="card">
        <h3>props.title</h3>
        <p>props.body</p>
    </div>
}
<Card title="Hello" body="World"/>
// <div class="card"><h3>Hello</h3><p>World</p></div>
```

### Component with Children

Use tag-pair syntax. Children are passed as `contents`:

```parsley
let Card = fn({title, contents}) {
    <div class="card">
        <h3>title</h3>
        contents
    </div>
}
<Card title="Hello"><p>"World"</p></Card>
// <div class="card"><h3>Hello</h3><p>World</p></div>
```

### Destructured Props

Components commonly destructure props for cleaner access:

```parsley
let Button = fn({label, type}) {
    <button class={"btn btn-" + type}>label</button>
}
<Button label="Save" type="primary"/>
```

### Layout Pattern

Components compose for layouts:

```parsley
let Page = fn({title, contents}) {
    <html>
    <head><title>title</title></head>
    <body>contents</body>
    </html>
}

<Page title="Home">
    <h1>"Welcome!"</h1>
    <p>"This is the home page."</p>
</Page>
```

## Special Tags

> Some special tags are **Basil-only** — they require the Basil web server and are not available in standalone `pars` scripts. These are marked below.

### SQL

The `<SQL>` tag builds parameterized queries. Content is the SQL text; parameters are passed as attributes:

```parsley
<SQL name="alice">
    SELECT * FROM users WHERE name = @name
</SQL>
```

Parameters use `@name` syntax inside the SQL. This prevents SQL injection — values are bound as parameters, never interpolated.

### Cache <small>(Basil only)</small>

The `<Cache>` tag caches rendered fragments by key. Requires the Basil server — in standalone `pars`, the tag still renders its content but caching is a no-op.

```parsley
<Cache key="sidebar" maxAge={300}>
    <nav>
        // ... expensive rendering ...
    </nav>
</Cache>
```

| Attribute | Type | Description |
|---|---|---|
| `key` | string | Cache key (required) |
| `maxAge` | integer | TTL in seconds (required) |
| `enabled` | boolean | Enable/disable caching (default: `true`) |

### Part <small>(Basil only)</small>

The `<Part>` tag creates an AJAX-loadable fragment. Requires the Basil server and a route configured in `basil.yaml`.

```parsley
<Part src={@./sidebar.part} view="default"/>
```

Parts are loaded from `.part` files — modules where all exports are view functions. Attributes include `src` (required), `view`, `refresh`, `lazy`, and `id`.

## tag() Builtin

Create tags programmatically when the tag name or structure is dynamic:

```parsley
tag("div", {class: "box"}, "Hello")
// Returns a tag dictionary: {__type: "tag", name: "div", attrs: {...}, contents: "Hello"}

tag("img", {src: "photo.jpg"})
// Self-closing when contents is null
```

`tag()` returns a tag dictionary (not an HTML string). It renders to HTML when used as output or inside other tags.

## Form Binding

Parsley provides special `@`-prefixed attributes that bind form elements to schema-validated records.

### @record

Establishes a form context. The attribute is removed from output:

```parsley
<form @record={userRecord} method="POST">
    // Form elements can now use @field
</form>
```

### @field

Binds an input to a schema field. Automatically sets `name`, `value`, `type`, constraint attributes (`required`, `minlength`, etc.), accessibility attributes, and `autocomplete`:

```parsley
<form @record={form} method="POST">
    <div class="field">
        <label @field="email"/>
        <input @field="email"/>
        <error @field="email"/>
    </div>
</form>
```

### Form Binding Elements

| Element | Purpose |
|---|---|
| `<input @field="name"/>` | Text input bound to field — sets name, value, type, constraints |
| `<label @field="name"/>` | Label from field metadata |
| `<error @field="name"/>` | Validation error message (renders nothing if valid) |
| `<select @field="status"/>` | Dropdown for enum fields — auto-generates `<option>` elements |
| `<val @field="name" @key="help"/>` | Metadata value (help text, hints) |

Use `@tag` to change the rendered element type:

```parsley
<label @field="email" @tag="span"/>  // Renders <span>Email</span>
<error @field="email" @tag="div"/>   // Renders <div class="error">...</div>
```

## Key Differences from Other Languages

- **Tags are syntax, not strings** — `<p>"Hello"</p>` is a Parsley expression, not a quoted string. Don't write `"<p>Hello</p>"`.
- **Self-closing slash is mandatory** — `<br/>` not `<br>`. This is stricter than HTML5.
- **Text content must be quoted** — `<p>"text"</p>` not `<p>text</p>`. Unquoted words are variable lookups.
- **No JSX-style `{...}` for simple attribute strings** — write `class="active"` not `class={"active"}`. Use `{...}` only for expressions.
- **Boolean attributes use `{true}`/`{false}`** — `{true}` includes the attribute, `{false}` omits it entirely.
- **Components are just functions** — any uppercase-named function can be used as a tag. No class syntax, no `render()` method, no hooks.
- **`contents` is the children prop** — when using tag-pair syntax on a component, children are passed as `contents`.

## See Also

- [Functions](functions.md) — components are functions returning tags
- [Modules](modules.md) — importing components from other files
- [Control Flow](control-flow.md) — `if` and `for` inside tag content
- [Data Model](data-model.md) — schema and records used in form binding