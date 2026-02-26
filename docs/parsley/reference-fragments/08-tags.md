## 8. Tags (HTML/XML)

Tags are first-class values that render to HTML strings. Unlike JSX (React), Parsley tags do not require quotes around attribute values for simple strings, and string content inside tags must be quoted.

**Key differences from JSX/React:**
- Attribute values don't need `{...}` for simple strings: `class="container"` not `class={"container"}`
- String content must be quoted: `<p>"Hello"</p>` not `<p>Hello</p>`
- Self-closing tags MUST use `/>`: `<br/>` not `<br>`

### 8.1 Self-Closing Tags

**Must use `/>` syntax** (unlike HTML5 where `<br>` is valid):

```parsley
<br/>
<hr/>
<img src="photo.jpg" alt="A photo"/>
<input type="text" name="email"/>
```

---

### 8.2 Pair Tags

Text content must be quoted. Unquoted text is interpreted as variable references:

```parsley
<p>"Hello, World!"</p>         // Literal string
<h1>"Welcome"</h1>             // Literal string

let message = "Dynamic content"
<p>message</p>                  // Variable reference
```

---

### 8.3 Attributes

#### String Attributes

```parsley
<div class="container">"Content"</div>
<a href="/about">"About Us"</a>
```

#### Expression Attributes

```parsley
let className = "active"
<div class={className}>"Dynamic class"</div>

let isDisabled = true
<button disabled={isDisabled}>"Click"</button>
```

---

### 8.4 Content

#### Variable Content

```parsley
let message = "Hello from variable"
<p>message</p>
```

#### Method Calls

```parsley
let name = "alice"
<span>name.toTitle()</span>
```

#### Parsley Code as Content

All Parsley code works inside tags—`let` statements, `for` loops, `if` expressions, function calls, etc.:

```parsley
<table>
    <thead>
        <tr>
            for (k,_ in rows[0]){
                if (k not in hidden) {
                    let title = k.toTitle()
                    <th class={"th-"+k}>
                        <a href={"?orderBy=" + title}>
                            title
                        </a>
                    </th>
                }
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
```

#### Expression Attributes

Expressions with operators work in attribute values using `{...}`:

```parsley
let count = 5
<div class={"item-" + toString(count)}>"test"</div>
<th class={"th-" + key}>key.toTitle()</th>
```

---

### 8.5 Nested Tags

```parsley
<div class="card">
    <h2>"Title"</h2>
    <p>"Body text"</p>
</div>
```

---

### 8.6 Spread Attributes

```parsley
let attrs = {class: "btn", id: "submit"}
<button ...attrs>"Submit"</button>
```

---

### 8.7 Components

Components are functions that return tags:

```parsley
let Card = fn(props) {
    let title = props.title
    let body = props.body
    <div class="card">
        <h3>title</h3>
        <p>body</p>
    </div>
}
<Card title="My Card" body="Card content"/>
```

#### Tag Pair Syntax with Contents

Components can also use tag pair syntax. Content is passed via `contents`:

```parsley
let Card = fn({title, contents}) {
    <div class="card">
        <h3>title</h3>
        <p>contents</p>
    </div>
}
<Card title="My Card">"Card content"</Card>
```

---

### 8.8 Form Binding

Parsley provides special attributes to bind HTML form elements to schema-validated records.

#### Form Context with `@record`

The `@record` attribute establishes a form context:

```parsley
<form @record={userRecord} method="POST">
    // Form elements can now use @field binding
</form>
```

The `@record` attribute is removed from output — it's a compile-time directive.

#### Input Binding with `@field`

The `@field` attribute binds an input to a schema field:

```parsley
<form @record={form} method="POST">
    <input @field="name"/>
    <input @field="email"/>
</form>
```

This automatically sets: `name`, `value`, `type` (from schema), constraint attributes (`required`, `minlength`, etc.), accessibility attributes (`aria-invalid`, `aria-describedby`), and `autocomplete` (derived from type/field name or metadata).

#### Autocomplete Derivation

The `autocomplete` attribute is automatically derived:

- **By type**: `email` → `"email"`, `phone` → `"tel"`, `url` → `"url"`
- **By field name**: `firstName` → `"given-name"`, `password` → `"current-password"`, etc.
- **By metadata**: Override with `| {autocomplete: "shipping street-address"}`

#### Form Binding Elements

| Element | Purpose | Example |
|---------|---------|---------|
| `<input @field="name"/>` | Text input bound to field | Sets name, value, type, constraints |
| `<label @field="name"/>` | Label from field metadata | Renders `<label for="name">Full Name</label>` |
| `<error @field="name"/>` | Validation error (if any) | Renders `<span class="error">...</span>` or nothing |
| `<select @field="status"/>` | Dropdown for enum fields | Auto-generates `<option>` elements |
| `<val @field="name" @key="help"/>` | Metadata value | Renders help text, hints, etc. |

**Example:**

```parsley
<form @record={form} method="POST">
    <div class="field">
        <label @field="email"/>
        <input @field="email"/>
        <error @field="email"/>
        <val @field="email" @key="help" @tag="small"/>
    </div>
</form>
```

Use `@tag` to change the output element type:

```parsley
<label @field="email" @tag="span"/>   // Renders <span>Email</span>
<error @field="email" @tag="div"/>    // Renders <div class="error">...</div>
```

See the [Record manual page](manual/builtins/record.md#form-binding) for complete documentation.

---

