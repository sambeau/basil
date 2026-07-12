---
id: man-bas-parts
title: "Parts"
system: basil
type: feature
name: parts
created: 2026-07-12
version: 1.0.0-alpha.3
author: "@sam"
keywords:
  - parts
  - components
  - interactive
  - part-click
  - part-submit
  - views
  - fragments
  - htmx
---

# Parts

Parts are interactive components that update themselves without a full page reload — counters, inline editors, live forms. No JavaScript framework: you write view functions in Parsley, and Basil swaps the HTML over the wire.

> **Basil-only.** Parts require the Basil server to serve view updates.

## Quick Example

A Part file (`counter.part`) exports one function per view:

```parsley
export default = fn(props) {
    <div>
        "Count: " + props.count
        <button part-click="increment" part-count={props.count}>"+"</button>
    </div>
}

export increment = fn(props) {
    let newCount = props.count + 1
    <div>
        "Count: " + newCount
        <button part-click="increment" part-count={newCount}>"+"</button>
    </div>
}
```

Use it from any handler:

```parsley
<Part src={@./counter.part} view="default" count={0}/>
```

Click the button; the counter updates in place.

## How It Works

- **`.part` files are view modules.** They may *only* export functions. Each export is a view — a state the component can be in.
- **`part-click="name"`** on any element calls the `name` view on click, with the element's `part-*` attributes as props.
- **`part-submit="name"`** on a `<form>` calls the view on submit, with the form fields as props.
- The server renders the view function and the component's HTML is replaced with the result.

A display/edit/save cycle is three views:

```parsley
export default = fn(props) {
    <div>props.name <button part-click="edit" part-name={props.name}>"Edit"</button></div>
}

export edit = fn(props) {
    <form part-submit="save">
        <input name="name" value={props.name}/>
        <button>"Save"</button>
    </form>
}

export save = fn(props) {
    // props.name comes from the form field
    <div>props.name <button part-click="edit" part-name={props.name}>"Edit"</button></div>
}
```

## The `<Part/>` Component

| Attribute | Description |
|-----------|-------------|
| `src` | Path to the `.part` file |
| `view` | Initial view to render (default: `"default"`) |
| *anything else* | Passed to the view as props |

## Routing Parts

The `.part` file must be reachable by URL so the browser can request view updates. In site mode, put it under your site directory. In routes mode, add a route:

```yaml
routes:
  - path: /
    handler: ./handlers/index.pars
  - path: /counter.part
    handler: ./handlers/counter.part
```

## See Also

- [Routing](routing.md) — how Part URLs resolve
- The [Parts guide](https://github.com/sambeau/basil/blob/main/docs/guide/parts.md) — nesting, lazy loading, loading states, and error handling
