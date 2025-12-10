# Parts - Interactive HTML Components

Parts enable interactive HTML components that update without full page reloads. They're perfect for counters, forms, live data updates, and any UI that needs to respond to user actions.

## Quick Example

**Create a Part file** (`counter.part`):

```parsley
export default = fn(props) {
    <div>
        Count: {props.count}
        <button part-click="increment" part-count={props.count}>+</button>
    </div>
}

export increment = fn(props) {
    let newCount = props.count + 1
    <div>
        Count: {newCount}
        <button part-click="increment" part-count={newCount}>+</button>
    </div>
}
```

**Use it in a handler** (`index.pars`):

```parsley
<Part src={@./counter.part} view="default" count={0}/>
```

**Configure the route** (`basil.yaml`):

```yaml
routes:
  - path: /
    handler: ./handlers/index.pars
  
  - path: /counter.part
    handler: ./handlers/counter.part
```

That's it! Click the button and the counter updates without a page reload.

## How It Works

### 1. Part Files (`.part`)

Part files are special Parsley modules that can **only export functions**. Each exported function is a "view" that returns HTML for a specific state.

```parsley
// Valid - only functions
export default = fn(props) { <div>Hello</div> }
export edit = fn(props) { <form>...</form> }

// Invalid - no variables allowed
export count = 0  // ERROR!
```

### 2. View Functions

Each view function:
- Receives a `props` dictionary as its parameter
- Returns HTML (tags or strings)
- Represents a different state or interaction mode

```parsley
export default = fn(props) {
    // Display mode
    <div>{props.name}
        <button part-click="edit">Edit</button>
    </div>
}

export edit = fn(props) {
    // Edit mode
    <form part-submit="save">
        <input name="name" value={props.name}/>
        <button>Save</button>
    </form>
}

export save = fn(props) {
    // Save and return to display
    // props.name contains the form value
    <div>{props.name}
        <button part-click="edit">Edit</button>
    </div>
}
```

### 3. The `<Part/>` Tag

The built-in `<Part/>` component embeds a Part in your page:

```parsley
<Part src={@./counter.part} view="default" count={0}/>
```

**Attributes:**
- `src` - Path to the `.part` file (must be a path literal with `@`)
- `view` - Which view function to call (defaults to `"default"`)
- Any other attributes become props passed to the view function

**Rendered Output:**

```html
<div data-part-src="/counter.part" 
     data-part-view="default" 
     data-part-props='{"count":0}'>
    <!-- View function's HTML output here -->
</div>
```

### 4. Interactive Attributes

Inside a Part's HTML, use special attributes to trigger view updates:

#### `part-click="viewName"`

Calls a view when the element is clicked:

```parsley
<button part-click="increment" part-count={count}>+</button>
```

When clicked:
1. Collects all `part-*` attributes from the button
2. Merges with container's props
3. Fetches `/counter.part?_view=increment&count=5`
4. Updates the Part's HTML

#### `part-submit="viewName"`

Calls a view when a form is submitted:

```parsley
<form part-submit="save">
    <input name="title" value={props.title}/>
    <input name="body" value={props.body}/>
    <button>Save</button>
</form>
```

When submitted:
1. Collects form data
2. Merges with container's props
3. Fetches `/article.part?_view=save&title=Hello&body=World`
4. Updates the Part's HTML

#### `part-*` Custom Props

Any attribute starting with `part-` becomes a prop:

```parsley
<button part-click="delete" part-id={item.id} part-confirm="true">
    Delete
</button>
```

The `delete` view receives: `{id: "123", confirm: "true", ...otherProps}`

## Props and Type Coercion

Props are passed via query parameters and automatically coerced:

| Query String | Coerced Value | Type |
|--------------|---------------|------|
| `count=42` | `42` | Integer |
| `price=3.14` | `3.14` | Float |
| `active=true` | `true` | Boolean |
| `active=false` | `false` | Boolean |
| `name=Alice` | `"Alice"` | String |

```parsley
export increment = fn(props) {
    // props.count is an Integer, not a String
    let newCount = props.count + 1  // Works correctly
    ...
}
```

## Routing Configuration

Parts need routes in `basil.yaml`:

```yaml
routes:
  - path: /
    handler: ./handlers/index.pars
  
  # Part files need explicit routes
  - path: /counter.part
    handler: ./handlers/counter.part
  
  - path: /article.part
    handler: ./handlers/article.part
```

The route path should match the URL generated from the Part's location:
- Handler at route `/` with Part `./counter.part` → Route `/counter.part`
- Handler at route `/admin` with Part `./widgets/todo.part` → Route `/admin/widgets/todo.part`

## JavaScript Runtime

When you use a `<Part/>` tag, Basil automatically injects JavaScript before `</body>`:

```html
<script>
(function() {
  // Handles part-click and part-submit events
  // Fetches new views from server
  // Updates Part innerHTML
})();
</script>
```

**Features:**
- Automatic initialization on page load
- Re-initializes after each Part update (for nested Parts)
- Graceful error handling (keeps old content on failure)
- Loading class (`part-loading`) during fetch
- Auto-refresh with `part-refresh={ms}`
- Lazy loading with `part-load="view"` (+ optional `part-load-threshold={px}`)

**CSS Hook:**

```css
[data-part-src].part-loading {
    opacity: 0.5;
    pointer-events: none;
}
```

## Advanced Patterns

### Auto-Refresh (`part-refresh`)

Refresh a Part on an interval (milliseconds). The timer resets after manual interactions and pauses when the tab is hidden.

```parsley
<Part src={@./clock.part} part-refresh={1000}/>
```

Details:
- Minimum interval: 100ms (anything lower is ignored)
- Uses the latest `data-part-props` and `data-part-view` for each refresh
- Stops if the Part is removed from the DOM
- Keeps using `part-loading` during fetch

### Lazy Loading (`part-load`, `part-load-threshold`)

Defer loading a view until the Part is near the viewport. Use a placeholder view for initial render.

```parsley
<Part 
    src={@./profile.part}
    view="placeholder"            # initial server render / placeholder
    part-load="loaded"            # view to load when visible
    part-load-threshold={200}      # start loading 200px before entering viewport (optional)
/>
```

Details:
- Uses Intersection Observer for efficient visibility detection
- Loads only once; does not reload on re-entry
- `part-load-threshold` defaults to `0` if omitted
- Auto-refresh (if configured) starts after the lazy load completes

### Nested Parts

Parts can contain other Parts:

```parsley
// dashboard.part
export default = fn(props) {
    <div>
        <h1>Dashboard</h1>
        <Part src={@./counter.part} view="default" count={0}/>
        <Part src={@./timer.part} view="default" seconds={60}/>
    </div>
}
```

Each Part maintains its own state and updates independently.

### Conditional Views

Use props to control which view is shown:

```parsley
export default = fn(props) {
    if (props.editing == "true") {
        // Show edit form
        <form part-submit="save">
            <input name="text" value={props.text}/>
        </form>
    } else {
        // Show display mode
        <div>{props.text}
            <button part-click="default" part-editing="true">Edit</button>
        </div>
    }
}
```

### State Accumulation

Props accumulate across interactions:

```parsley
// Initial render
<Part src={@./form.part} view="default" step={1}/>

// After clicking next, step=2 is added
<button part-click="next" part-step={2}>Next</button>

// View receives {step: 2, ...originalProps}
export next = fn(props) {
    // props.step is now 2
    ...
}
```

### Multi-Step Forms

```parsley
export default = fn(props) {
    let step = props.step ?? 1
    
    if (step == 1) {
        <form part-submit="step2">
            <input name="name" placeholder="Your name"/>
            <button>Next</button>
        </form>
    } else if (step == 2) {
        <form part-submit="step3">
            <p>Hello, {props.name}!</p>
            <input name="email" placeholder="Your email"/>
            <button>Next</button>
        </form>
    } else {
        <div>
            <p>Name: {props.name}</p>
            <p>Email: {props.email}</p>
            <button part-click="default" part-step={1}>Start Over</button>
        </div>
    }
}

export step2 = fn(props) {
    // Add step marker and re-render
    default({name: props.name, step: 2})
}

export step3 = fn(props) {
    default({name: props.name, email: props.email, step: 3})
}
```

## Error Handling

### Server Errors

If a view function fails or returns an error:
- Server responds with 400/404/500
- JavaScript logs error to console
- Old content remains visible (no blank/broken state)

### Client Errors

If the fetch fails (network error, timeout):
- JavaScript logs error to console
- Old content remains visible
- `part-loading` class is removed

### Debugging

Check the browser console for Part-related errors:

```
Failed to update Part: Error: HTTP 404
Failed to parse Part props: SyntaxError
```

The server logs show:
- Part file path
- View function name
- Props received
- Any execution errors

## Example: Todo List

**File: `todo.part`**

```parsley
export default = fn(props) {
    let todos = props.todos ?? []
    
    <div>
        <h2>Todos</h2>
        <form part-submit="add">
            <input name="text" placeholder="New todo"/>
            <button>Add</button>
        </form>
        <ul>
            {for (todo in todos) {
                <li>{todo}
                    <button part-click="remove" part-text={todo}>×</button>
                </li>
            }}
        </ul>
    </div>
}

export add = fn(props) {
    let todos = props.todos ?? []
    let newTodos = todos ++ [props.text]
    default({todos: newTodos})
}

export remove = fn(props) {
    let todos = props.todos ?? []
    let filtered = for (todo in todos) {
        if (todo != props.text) { todo }
    }
    default({todos: filtered})
}
```

**File: `index.pars`**

```parsley
<html>
<body>
    <h1>My Todos</h1>
    <Part src={@./todo.part} view="default" todos={["Buy milk", "Walk dog"]}/>
</body>
</html>
```

**File: `basil.yaml`**

```yaml
routes:
  - path: /
    handler: ./handlers/index.pars
  - path: /todo.part
    handler: ./handlers/todo.part
```

## See Also

- [Parsley CHEATSHEET.md](../parsley/CHEATSHEET.md) - Quick Parts syntax reference
- [Parsley reference.md](../parsley/reference.md) - Complete Parts specification
- [examples/parts/](../../examples/parts/) - Working example with counter
- [FAQ](faq.md) - Common Parts questions
