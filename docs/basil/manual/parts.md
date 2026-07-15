---
id: man-bas-parts
title: "Parts"
system: basil
type: feature
name: parts
created: 2026-07-12
version: 1.0.0-alpha.4
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

Parts are interactive components that update themselves without a full page reload — counters, inline editors, live clocks, forms that validate as you go. No JavaScript framework: you write view functions in Parsley, and Basil swaps the HTML over the wire.

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

- **`.part` files are view modules.** They may *only* export functions. Each export is a view — a state the component can be in. (Imports and local `let`s are fine; they just can't be exported.)
- **The server renders the initial view** into the page, wrapped in a `<div>` that records the Part's source, current view, and props.
- **Interactions fetch a new view.** `part-click` and `part-submit` ask the server to render a different view function, and the Part's HTML is replaced with the result.
- **A small JavaScript runtime is injected automatically** — only on pages that contain a `<Part/>`. There is nothing to install or import.

Because the initial view is rendered server-side, Parts are SEO-friendly and show real content before any JavaScript runs.

## The `<Part/>` Component

```parsley
<Part src={@./clock.part} id="clock" part-refresh={1000}/>
```

| Attribute | Description |
|-----------|-------------|
| `src` | Path to the `.part` file (required) |
| `view` | Initial view to render (default: `"default"`) |
| `id` | Element id — needed for [`part-target`](parts-guide.md#controlling-a-part-from-outside) and the [JavaScript API](parts-js.md) |
| `part-refresh` | Auto-refresh interval in milliseconds (minimum 100) |
| `part-load` | View to fetch immediately after page load (show a placeholder first) |
| `part-lazy` | View to fetch when the Part scrolls into the viewport |
| `part-lazy-threshold` | Start lazy-loading this many pixels before the Part is visible |
| *anything else* | Passed to the view function as props |

## Triggering Views

Inside a Part's HTML, two attributes trigger view changes:

**`part-click="name"`** on any element calls the `name` view on click. It sends *only* the `part-*` props written on that element — view switches start fresh, so pass along any state you want to keep:

```parsley
<button part-click="edit" part-id={props.id} part-name={props.name}>"Edit"</button>
```

**`part-submit="name"`** on a `<form>` calls the view on submit. Form fields are merged *on top of* the Part's current props, so existing state carries through automatically:

```parsley
<form part-submit="save">
    <input name="name" value={props.name}/>
    <button>"Save"</button>
</form>
```

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

## Props

Props travel as query parameters (clicks) or form data (submits), and Basil coerces them back to sensible types before your view function sees them:

| Sent value | Received as |
|------------|-------------|
| `42` | Integer |
| `3.14` | Float |
| `true` / `false` | Boolean |
| `on` / `off` | Boolean (checkbox values) |
| anything else | String |

Complex values work too. Records, datetimes, paths, and URLs passed to a `<Part/>` — including inside dictionaries and arrays — are serialized automatically and HMAC-signed with the session secret, so they arrive in your view function as their original types and can't be tampered with in the browser:

```parsley
<Part src={@./profile.part} user={user} lastLogin={@now}/>
```

```parsley
// profile.part
export default = fn(props) {
    <div>
        <h1>props.user.name</h1>          // still a Record
        <p>"Last login: " + props.lastLogin.time</p>  // still a datetime
    </div>
}
```

Functions, database connections, and file handles can't be passed as props.

## Routing Parts

The `.part` file must be reachable by URL so the browser can request view updates. In site mode, put it anywhere under your site directory. In routes mode, add a route:

```yaml
routes:
  - path: /
    handler: ./handlers/index.pars
  - path: /counter.part
    handler: ./handlers/counter.part
```

Parts can't be fetched directly: a request for a `.part` URL without a view returns 404. Views are only served through the runtime's requests (or your own, via the [JavaScript API](parts-js.md)).

## There's More

This page covers the essentials. Parts can also:

- **Nest** — Parts inside Parts, each updating independently
- **Lazy-load** — render a placeholder, fetch the real view when scrolled into sight
- **Auto-refresh** — live clocks and dashboards with `part-refresh`
- **Show loading states** — a `part-loading` CSS class during every fetch
- **Be controlled from outside** — buttons elsewhere on the page can drive a Part with `part-target`, and scripts can use `window.Parts`

## See Also

- [The Parts Guide](parts-guide.md) — patterns and deep dives: nesting, lazy loading, loading states, live data, and error handling
- [Parts JavaScript API](parts-js.md) — `window.Parts`, events, and cross-Part targeting
- [Routing](routing.md) — how Part URLs resolve
