# Parts: Interactive Components

> **Note**: For database operations and file I/O used within Parts, see `docs/basil/reference.md`.

Parts are server-rendered HTML fragments that can update dynamically without full page reloads. They're similar to Hotwire Turbo Frames or HTMX fragments.

## Key Concepts

1. **Parts are `.part` files** - Not regular `.pars` handlers
2. **Parts export functions** - Cannot export variables
3. **Parts need routes** - Must be configured in `basil.yaml`
4. **Parts update via HTTP** - Server renders new HTML, client swaps it in

## Creating a Part

### Basic Part

```parsley
// parts/counter.part
export default = fn(props) {
  let count = props.count ?? 0
  
  <div id="counter">
    <p>`Count: {count}`</p>
    <button part-click="increment" part-count={count + 1}>"Increment"</button>
    <button part-click="decrement" part-count={count - 1}>"Decrement"</button>
    <button part-click="reset" part-count={0}>"Reset"</button>
  </div>
}

export increment = fn(props) {
  default(props)  // Re-render with new props
}

export decrement = fn(props) {
  default(props)
}

export reset = fn(props) {
  default(props)
}
```

**Key points:**
- `default` export is the initial render
- Other exports are "states" or "actions"
- `part-click` attributes trigger state transitions
- `part-*` attributes become props for the next render


## Using Parts in Handlers

### Basic Usage

```parsley
// handlers/index.pars
<html>
<body>
  <h1>"My App"</h1>
  
  <!-- Include a part -->
  <Part src={@~/parts/counter.part} count={0}/>
</body>
</html>
```

### With ID (for targeted updates)

```parsley
<Part src={@~/parts/search-results.part} id="results" q={@params.q}/>
```

### With Auto-Refresh

```parsley
<!-- Refresh every 1000ms (1 second) -->
<Part src={@~/parts/clock.part} part-refresh={1000}/>

<!-- Live stats that update every 5 seconds -->
<Part src={@~/parts/stats.part} part-refresh={5000}/>
```

## Configuring Part Routes

Parts MUST be routed in `basil.yaml`:

```yaml
routes:
  # Individual parts
  - path: /parts/counter.part
    handler: ./parts/counter.part
  
  - path: /parts/todos.part
    handler: ./parts/todos.part
  
  # Or use wildcard for all parts
  - path: /parts/*.part
    handler: ./parts/{}.part
```

## Part Attributes

### Event Handlers

- `part-click="action"` - Click event triggers `action` export
- `part-submit="action"` - Form submit triggers `action` export  
- `part-change="action"` - Input change triggers `action` export

### Props

Any `part-*` attribute becomes a prop:

```parsley
<button part-click="increment" part-count={5} part-step={2}>"+2"</button>
```

In the `increment` function:
```parsley
export increment = fn(props) {
  props.count  // 5
  props.step   // 2
}
```

### Auto-Refresh

```parsley
<Part src={@~/parts/clock.part} part-refresh={1000}/>
```

Automatically re-renders every 1000ms.

## Common Patterns

### Toggle State

```parsley
// parts/toggle.part
export default = fn(props) {
  let open = props.open ?? false
  
  if (open) {
    <div>
      <p>"Content is visible"</p>
      <button part-click="toggle" part-open={false}>"Hide"</button>
    </div>
  } else {
    <button part-click="toggle" part-open={true}>"Show"</button>
  }
}

export toggle = fn(props) {
  default(props)
}
```

### Loading State

```parsley
// parts/search.part
export default = fn(props) {
  <form part-submit="search">
    <input name="q" placeholder="Search..."/>
    <button>"Search"</button>
  </form>
}

export search = fn(props) {
  let results = @DB <=??=> `SELECT * FROM items WHERE name LIKE '%{props.q}%'`
  
  <div>
    <form part-submit="search">
      <input name="q" value={props.q}/>
      <button>"Search"</button>
    </form>
    <div class="results">
      for (item in results) {
        <div>item.name</div>
      }
    </div>
  </div>
}
```

### Multiple Parts Interaction

```parsley
// handlers/todos.pars
<html>
<body>
  <h1>"Todo App"</h1>
  
  <!-- Form to add todos -->
  <Part src={@~/parts/add-todo.part}/>
  
  <!-- List of todos (could auto-refresh) -->
  <Part src={@~/parts/todos.part} part-refresh={5000}/>
</body>
</html>
```

## Common Pitfalls

### 1. Parts Must Have Routes

```yaml
# ❌ WRONG - Part won't work
<Part src={@~/parts/counter.part}/>

# ✅ CORRECT - Add route first
routes:
  - path: /parts/counter.part
    handler: ./parts/counter.part
```

### 2. Parts Cannot Export Variables

```parsley
// ❌ WRONG
export counter = 0           // Error!

// ✅ CORRECT
export default = fn(props) {
  let counter = props.count ?? 0
  // ...
}
```

### 3. Part Functions Must Return HTML

```parsley
// ❌ WRONG
export increment = fn(props) {
  props.count + 1  // Returns number, not HTML
}

// ✅ CORRECT
export increment = fn(props) {
  default(props)   // Returns HTML
}
```

### 4. Props Are Strings

All part-* attributes become string props:

```parsley
<button part-click="inc" part-count={5}>"++"</button>

export inc = fn(props) {
  props.count  // "5" (string, not number!)
  
  // Convert if needed:
  let count = toInt(props.count)
  count + 1
}
```

## Debugging Parts

1. **Check routes** - Ensure part is routed in basil.yaml
2. **Check browser network tab** - Part updates are HTTP requests
3. **Use dev tools** - Check `/__dev/log` for errors
4. **Test part URL directly** - Visit `/parts/counter.part?count=5`
5. **Check function exports** - Ensure all `part-click` actions have matching exports

## See Also

- `docs/guide/parts.md` - Parts user guide
- `docs/basil/reference.md` - Database and file I/O operations for use in Parts
- Examples: `examples/parts/` - Example part implementations
