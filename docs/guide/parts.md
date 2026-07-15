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
2. Fetches `/counter.part?_view=increment&count=5`
3. Updates the Part's HTML

**Note:** `part-click` does *not* merge the container's props — view switches start
fresh, so the view receives exactly the `part-*` props written on the element.
Pass along any state the next view needs (or better, store it in the database
and pass only an id).

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
2. Merges it on top of the container's current props
3. POSTs to `/article.part?_view=save` with the fields in the body
4. Updates the Part's HTML

Unlike `part-click`, form submits *do* carry the Part's existing props forward —
form fields supplement the Part's current state.

#### `part-*` Custom Props

Any attribute starting with `part-` becomes a prop:

```parsley
<button part-click="delete" part-id={item.id} part-confirm="true">
    Delete
</button>
```

The `delete` view receives: `{id: 123, confirm: true}` (values are type-coerced,
and only the button's own `part-*` props are sent)

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

### Complex Props (Records, DateTime, etc.)

Parts can pass complex types like Records, DateTimes, and nested dictionaries. These are automatically serialized to PLN (Parsley Literal Notation) and HMAC-signed for security.

```parsley
schema User {
    name: String
    email: String
}

let user = User({ name: "Alice", email: "alice@example.com" })

<Part 
    src={@./profile.part} 
    view="show" 
    user={user}              // Record - auto-serialized to PLN
    lastLogin={@now}         // DateTime - auto-serialized
    settings={{ theme: "dark", notifications: true }}  // Dictionary - auto-serialized
/>
```

In the Part view function, these props are automatically deserialized back to their original types:

```parsley
// profile.part
export show = fn(props) {
    <div>
        <h1>{props.user.name}</h1>           // user is a User record
        <p>Last login: {props.lastLogin.format("date")}</p>  // DateTime
        <p>Theme: {props.settings.theme}</p>  // Dictionary
    </div>
}
```

**Security**: Complex props are HMAC-signed using the session secret. Tampered props are rejected, preventing injection attacks. The signature is transparent - you don't need to do anything special.

**Supported types for auto-serialization**:
- Records (with or without validation errors)
- DateTime values
- Paths and URLs
- Nested dictionaries and arrays containing any of the above

**Not supported** (will cause an error):
- Functions
- Database connections
- File handles

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
- Immediate async load with `part-load="view"` (for slow data)
- Lazy loading with `part-lazy="view"` (+ optional `part-lazy-threshold={px}`)
- Cross-Part targeting: `part-target="id"` lets any element on the page drive a Part by id
- A scripting API: `window.Parts` — `refresh()` (with debounce), `get()`, and `on()`/`off()` lifecycle events (`beforeRefresh`, `afterRefresh`, `error`); see the [Parts JavaScript API](../basil/manual/parts-js.md)

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
- Minimum interval: 100ms (anything lower is clamped to 100ms)
- Uses the latest `data-part-props` and `data-part-view` for each refresh
- Stops if the Part is removed from the DOM
- Keeps using `part-loading` during fetch

### Immediate Async Load (`part-load`)

Fetch a view immediately after page load. Use for slow data (API calls, database queries) where you want to show a placeholder first.

```parsley
<Part 
    src={@./profile.part}
    view="placeholder"            # initial server render
    part-load="loaded"            # view to fetch immediately
/>
```

Details:
- Renders placeholder view server-side
- Fetches specified view immediately after page loads
- Use when data is slow but should start loading right away
- Auto-refresh (if configured) starts after the load completes

### Lazy Loading (`part-lazy`, `part-lazy-threshold`)

Defer loading a view until the Part is scrolled near the viewport. Use a placeholder view for initial render.

```parsley
<Part 
    src={@./heavy-chart.part}
    view="placeholder"            # initial server render / placeholder
    part-lazy="loaded"            # view to load when visible
    part-lazy-threshold={200}     # start loading 200px before entering viewport (optional)
/>
```

Details:
- Uses Intersection Observer for efficient visibility detection
- Loads only once; does not reload on re-entry
- `part-lazy-threshold` defaults to `0` if omitted
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

Props accumulate across *form submits* (fields merge on top of the Part's
current props), which is what makes multi-step forms work. Clicks don't
accumulate — a `part-click` sends only the props written on the element:

```parsley
// Initial render — the Part's props are {step: 1}
<Part src={@./form.part} view="default" step={1}/>

// A form submit merges its fields with {step: 1}
<form part-submit="next">
    <input name="name"/>
    <button>Next</button>
</form>

// View receives {step: 1, name: "..."}
export next = fn(props) {
    ...
}

// A click sends ONLY its own part-* props
<button part-click="next" part-step={2}>Next</button>
// View receives {step: 2} — nothing else
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

The todo list lives in the database — each view re-reads it, so the Part is
always correct, even across page refreshes or multiple tabs. (Carrying a
growing list through props doesn't survive repeated updates; store real state
and pass ids.)

```parsley
export default = fn(props) {
    let todos = @DB <=??=> "SELECT * FROM todos ORDER BY id"

    <div>
        <h2>Todos</h2>
        <form part-submit="add">
            <input name="text" placeholder="New todo"/>
            <button>Add</button>
        </form>
        <ul>
            {for (todo in todos) {
                <li>{todo.text}
                    <button part-click="remove" part-id={todo.id}>×</button>
                </li>
            }}
        </ul>
    </div>
}

export add = fn(props) {
    @DB <=!=> "INSERT INTO todos (text) VALUES (?)" <- [props.text]
    default({})
}

export remove = fn(props) {
    @DB <=!=> "DELETE FROM todos WHERE id = ?" <- [props.id]
    default({})
}
```

**File: `index.pars`**

```parsley
<html>
<body>
    <h1>My Todos</h1>
    <Part src={@./todo.part} view="default"/>
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

- [The Parts Guide](../basil/manual/parts-guide.md) - Patterns in depth: nesting, lazy loading, loading states, error handling
- [Parts JavaScript API](../basil/manual/parts-js.md) - `window.Parts`, events, and cross-Part targeting
- [Parsley CHEATSHEET.md](../parsley/CHEATSHEET.md) - Quick Parts syntax reference
- [Parsley reference.md](../parsley/reference.md) - Complete Parts specification
- [examples/parts/](../../examples/parts/) - Working example with counter
- [FAQ](faq.md) - Common Parts questions
