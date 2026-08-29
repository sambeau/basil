---
id: man-bas-parts-js
title: "Parts JavaScript API"
system: basil
type: reference
name: parts-js
created: 2026-07-14
version: 1.0.0-alpha.4
author: "@sam"
keywords:
  - parts
  - javascript
  - window.Parts
  - refresh
  - events
  - beforeRefresh
  - afterRefresh
  - part-target
  - debounce
---

# Parts JavaScript API

Parts work without writing any JavaScript — but when you want to script them, the runtime exposes a small API: refresh any Part from your own code, read its state, and subscribe to its lifecycle. This is how Parts talk to the rest of your front end: websockets, keyboard shortcuts, third-party widgets, analytics.

The runtime is injected automatically on any page that renders a `<Part/>`. There is nothing to include; `window.Parts` is just there.

## Give Your Parts Ids

Everything on this page addresses Parts by element id:

```parsley
<Part src={@./results.part} id="results"/>
```

A Part without an `id` still works normally — it just can't be scripted or targeted, and it fires no events.

## `Parts.refresh(id, props, options)`

Fetch a view and update the Part — the programmatic equivalent of a `part-click`, but with merging: `props` are merged *on top of the Part's current props*, and the merged set becomes the Part's new state.

```js
// Re-fetch the current view with current props
Parts.refresh("results");

// Merge in new props
Parts.refresh("results", { q: "basil" });

// Switch view, debounce, or POST
Parts.refresh("results", { q: "basil" }, {
    view: "search",     // default: the Part's current view
    debounce: 300,      // ms — collapse rapid calls into one (per Part)
    method: "POST"      // default: "GET"
});
```

| Option | Default | Description |
|--------|---------|-------------|
| `view` | current view | Which view function to fetch |
| `debounce` | none | Wait this many ms after the *last* call before fetching |
| `method` | `"GET"` | `"GET"` sends props as query params, `"POST"` as a form body |

Debouncing is per-Part: rapid calls to the same id collapse into one request. That makes live search a one-liner:

```parsley
<input oninput='Parts.refresh("results", {q: this.value}, {view: "search", debounce: 300})'
       placeholder="Search…"/>

<Part src={@./results.part} id="results" view="search" q=""/>
```

Because the input lives *outside* the Part, it never reloads — focus and cursor position survive while the results update beneath it. That's the pattern to reach for whenever some of the UI must persist (a search box, a filter bar, a video player) while the rest refreshes: master-detail lists, dashboard filters, live markdown previews.

If the id doesn't exist (or isn't a Part), `refresh` logs a console warning and does nothing.

Note that `refresh` returns immediately — it doesn't return a promise, so there's nothing to `await`. To run code once the update has landed, listen for [`afterRefresh`](#events).

## `Parts.get(id)`

Read a Part's current state. Returns `null` if the id isn't a Part.

```js
const part = Parts.get("results");
// {
//     id: "results",
//     view: "search",         // current view name
//     props: { q: "basil" },  // current props
//     element: <div>,          // the wrapper DOM element
//     loading: false           // true while a fetch is in flight
// }
```

Handy for guards (`if (!Parts.get("editor").loading) …`) and for building on top of a Part's state without duplicating it.

## `Parts.on(id, event, callback)`

Subscribe to a Part's lifecycle. Returns an unsubscribe function.

```js
const off = Parts.on("results", "afterRefresh", (detail) => {
    console.log(`results now showing ${detail.view}`, detail.props);
});

off();  // unsubscribe
```

Pass `"*"` as the id to hear from every Part on the page:

```js
Parts.on("*", "error", (detail) => {
    toast(`Something went wrong updating ${detail.id}`);
});
```

Listeners are registered by id, not by element — they survive refreshes, even though the Part's inner HTML is replaced each time.

### Events

| Event | Fires | Detail |
|-------|-------|--------|
| `beforeRefresh` | Just before a fetch starts | `{ id, view, props }` |
| `afterRefresh` | After new HTML is in the DOM | `{ id, view, props }` |
| `error` | When a fetch or view fails (old content is kept) | `{ id, view, props, error }` |

All updates fire events, whatever triggered them: clicks, submits, auto-refresh, lazy loads, `part-target`, or `Parts.refresh`. Events only fire for Parts that have an id.

`afterRefresh` is the hook for work that needs the new DOM — syntax highlighting, focusing a field, re-attaching a third-party widget:

```js
Parts.on("editor", "afterRefresh", () => {
    document.querySelector("#editor input[autofocus]")?.focus();
});
```

And events make Parts coordinate without knowing about each other:

```js
// When the cart Part updates, refresh the little badge in the header
Parts.on("cart", "afterRefresh", () => Parts.refresh("cart-badge"));
```

## `Parts.off(id)`

Remove *all* listeners for a Part in one go:

```js
Parts.off("results");
```

For removing a single listener, use the function returned by `Parts.on`.

## Declarative Targeting: `part-target`

You often don't need JavaScript at all — `part-target` lets any element on the page drive a Part by id, straight from HTML. See [Controlling a Part from Outside](parts-guide.md#controlling-a-part-from-outside) in the Parts Guide:

```parsley
<button part-target="editor" part-click="edit" part-id={person.id}>"Edit"</button>
```

Rule of thumb: reach for `part-target` first, and for `window.Parts` when you need timing (debounce), conditions, or a non-click trigger (websocket message, timer, keyboard shortcut):

```js
const socket = new WebSocket("wss://example.com/updates");
socket.onmessage = () => Parts.refresh("notifications");
```

## How the Runtime Works

Useful when debugging, or if you're curious what the runtime actually does.

### The wrapper element

Every `<Part/>` renders as a `<div>` that carries the Part's state in data attributes:

```html
<div id="results"
     data-part-src="/parts/results.part"
     data-part-view="search"
     data-part-props='{"q":"basil"}'>
    …current HTML…
</div>
```

| Attribute | Meaning |
|-----------|---------|
| `data-part-src` | URL the runtime fetches views from |
| `data-part-view` | The view currently displayed |
| `data-part-props` | The current props, as JSON |
| `data-part-refresh` | Auto-refresh interval (ms), if set |
| `data-part-load` | View to fetch on page load, if set |
| `data-part-lazy` / `data-part-lazy-threshold` | Lazy-load view and distance, if set |

After every update the runtime writes the new view name and props back to these attributes, so the DOM always tells the truth about a Part's state.

### The wire format

A view request is an ordinary HTTP request:

```
GET  /parts/results.part?_view=search&q=basil     ← clicks, refreshes
POST /parts/results.part?_view=save               ← form submits
     body: id=7&text=Hello
```

`_view` names the view function; everything else becomes props, coerced back to types server-side (numbers, booleans, signed complex values). Requests without `_view` get a 404 — Parts can't be fetched directly. During every fetch the wrapper has a `part-loading` class; on failure the old content is kept and an `error` event fires.

### Reserved attribute names

`part-click`, `part-submit`, `part-load`, `part-lazy`, `part-lazy-threshold`, `part-refresh`, `part-target`, and `part-form` are runtime instructions — every *other* `part-*` attribute becomes a prop.

## See Also

- [Parts](parts.md) — the essentials
- [The Parts Guide](parts-guide.md) — patterns: nesting, lazy loading, loading states, error handling
