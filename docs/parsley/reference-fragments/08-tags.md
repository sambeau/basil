## 8. Tags (HTML/XML)

Parsley treats HTML/XML tags as first-class syntax, making it natural to generate markup.

### 8.1 Self-Closing Tags

Self-closing tags **must** use the `/>` syntax:

```parsley
<br/>
<hr/>
<input type="text" name="email"/>
<img src="photo.jpg" alt="A photo"/>
```

### 8.2 Pair Tags

```parsley
<div>"content"</div>
<p>"Hello, World!"</p>
<section>
    <h1>"Title"</h1>
    <p>"Body text"</p>
</section>
```

### 8.3 Attributes

#### String Attributes

Unquoted attribute values are treated as strings:

```parsley
<div class="container" id="main">"content"</div>
```

#### Expression Attributes

Use `{expression}` for dynamic attribute values:

```parsley
<div class={className} id={`item-{id}`}>"content"</div>
<input value={user.name} disabled={isDisabled}/>
```

### 8.4 Content

Content between tags can be:

```parsley
// Literal strings (must be quoted)
<p>"Hello, World!"</p>

// Variables
<p>userName</p>

// Expressions
<p>user.name.toUpper()</p>

// Interpolated strings
<p>`Welcome, {user.name}!`</p>

// Other tags
<div>
    <span>"nested"</span>
</div>

// For loops
<ul>
    for (item in items) {
        <li>item.name</li>
    }
</ul>

// Conditionals
<div>
    if (loggedIn) {
        <span>"Welcome back!"</span>
    } else {
        <a href="/login">"Log in"</a>
    }
</div>
```

### 8.5 Spread Attributes

Spread a dictionary into attributes:

```parsley
let attrs = {class: "btn", disabled: true}
<button ...attrs>"Click"</button>
// → <button class="btn" disabled>Click</button>
```

Boolean handling:
- `true` → attribute present (e.g., `disabled`)
- `false` or `null` → attribute omitted

### 8.6 Components

Components are functions that return markup:

```parsley
let Button = fn(props) {
    <button class={props.class ?? "btn"}>
        props.children
    </button>
}

// Usage
<Button class="primary">"Submit"</Button>
```
