---
id: man-bas-parts-guide
title: "The Parts Guide"
system: basil
type: guide
name: parts-guide
created: 2026-07-14
version: 1.0.0-alpha.4
author: "@sam"
keywords:
  - parts
  - guide
  - nesting
  - lazy loading
  - part-load
  - part-lazy
  - part-refresh
  - part-target
  - loading states
  - error handling
  - patterns
---

# The Parts Guide

The [Parts page](parts.md) covers the essentials: `.part` files, views, `part-click`, and `part-submit`. This guide is the deep dive — the patterns that come up when you build real things with Parts, and the details that make them work.

## Thinking in Views

A Part is a little state machine. Each exported function is one state the component can be in, and interactions move between them. When you're sketching a Part, start by listing its states:

- An inline editor: `default` (display), `edit` (form), `save` (write, then display)
- A modal editor: `new` (an add button), `form` (the modal), `delete` (a confirm dialog)
- A slow panel: `placeholder` (skeleton), `loaded` (the real content)

Views are just functions, so they can share implementation. Aliasing an export is the usual trick when two states render the same thing:

```parsley
// counter.part
let render = fn(props) {
    <div>
        "Count: " + props.count
        <button part-click="step" part-count={props.count + 1}>"+"</button>
        <button part-click="step" part-count={props.count - 1}>"−"</button>
    </div>
}

export default = render
export step = render
```

And views can call each other — a `save` view typically does its work, then hands off to `default` to render the result:

```parsley
export save = fn(props) {
    @DB <=!=> <SQL text={props.text} id={props.id}>UPDATE todos SET text = :text WHERE id = :id</SQL>
    default(props)
}
```

View functions receive one props dictionary, so destructuring keeps them tidy:

```parsley
export edit = fn({id, text}) {
    <form part-submit="save">
        <input type="hidden" name="id" value={id}/>
        <input name="text" value={text} autofocus/>
        <button>"Save"</button>
    </form>
}
```

## Where State Lives

Parts have no memory between requests. Whatever a view needs must arrive in its props — or come from somewhere durable, like the database. The two triggers handle props differently, and the difference is worth internalizing:

- **`part-click` starts fresh.** The view receives *only* the `part-*` props written on the clicked element. If the next view needs a value, pass it explicitly: `part-id={props.id}`.
- **`part-submit` carries state forward.** Form fields are merged on top of the Part's current props, so everything the Part already knew survives the submit.

Fresh clicks sound like a limitation, but they keep Parts honest: a button says exactly what the next view will receive, and stale state can't sneak through. For anything bigger than a few scalars, don't ferry data through props at all — store it and re-read it:

```parsley
// todo-item.part

export default = fn({id}) {
    let todo = @DB <=?=> <SQL id={id}>SELECT * FROM todos WHERE id = :id</SQL>
    <div>
        todo.text
        <button part-click="edit" part-id={id}>"Edit"</button>
    </div>
}

export edit = fn({id}) {
    let todo = @DB <=?=> <SQL id={id}>SELECT * FROM todos WHERE id = :id</SQL>
    <form part-submit="save">
        <input type="hidden" name="id" value={id}/>
        <input name="text" value={todo.text} autofocus/>
        <button>"Save"</button>
        <button type="button" part-click="default" part-id={id}>"Cancel"</button>
    </form>
}

export save = fn({id, text}) {
    @DB <=!=> <SQL text={text} id={id}>UPDATE todos SET text = :text WHERE id = :id</SQL>
    default({id: id})
}
```

The only prop that ever travels is `id`. The database is the source of truth, and every view renders from it — which also means the Part is always correct after a refresh, a lost update, or a second browser tab.

### Multi-Step Forms

Because form submits merge into the Part's props, multi-step wizards accumulate answers naturally:

```parsley
// signup.part

export default = fn(props) {
    <form part-submit="step2">
        <input name="name" placeholder="Your name"/>
        <button>"Next"</button>
    </form>
}

export step2 = fn(props) {
    // props.name arrived from step one and will be carried
    // through the next submit automatically
    <form part-submit="finish">
        <p>`Hello, {props.name}!`</p>
        <input name="email" placeholder="Your email"/>
        <button>"Sign up"</button>
    </form>
}

export finish = fn({name, email}) {
    @DB <=!=> <SQL name={name} email={email}>INSERT INTO signups (name, email) VALUES (:name, :email)</SQL>
    <p>`Thanks, {name} — check {email} for a confirmation.`</p>
}
```

Each submit adds its fields to the accumulated props, so `finish` receives both `name` and `email` without any hidden inputs.

## Nesting Parts

Parts can contain other Parts. Each one gets its own wrapper, its own props, and updates independently — clicking inside one Part never touches its siblings:

```parsley
// dashboard.part
export default = fn({userId}) {
    <div class="dashboard">
        <Part src={@./clock.part} part-refresh={1000}/>
        <Part src={@./notifications.part} userId={userId} part-refresh={30000}/>
        <Part src={@./activity.part} userId={userId} view="placeholder" part-lazy="loaded"/>
    </div>
}
```

When a parent Part re-renders, its children come back with it (they're part of the parent's HTML) and the runtime re-initializes them automatically, including their timers and lazy-loading observers.

## Loading States

Every fetch adds a `part-loading` class to the Part's wrapper and removes it when the new HTML lands (or the fetch fails). Style it however you like — it does nothing until you do:

```css
[data-part-src].part-loading {
    opacity: 0.5;
    pointer-events: none;
}
```

Or add a spinner:

```css
[data-part-src] {
    position: relative;
}
[data-part-src].part-loading::after {
    content: "";
    position: absolute;
    inset: 0;
    background: url(/spinner.svg) center no-repeat;
}
```

For content that hasn't arrived yet (see `part-load` and `part-lazy` below), the usual pattern is a skeleton view — same shape as the real content, greyed out:

```parsley
export placeholder = fn(props) {
    <div class="card skeleton">
        <p>"Loading…"</p>
    </div>
}
```

## Slow Data: `part-load`

Some views are slow — an API call, a heavy query. Rather than hold up the whole page, render a placeholder and let the Part fetch its real content the moment the page has loaded:

```parsley
<Part
    src={@./profile.part}
    view="placeholder"       // rendered into the page by the server
    part-load="loaded"       // fetched immediately after page load
    userId={userId}
/>
```

```parsley
// profile.part

export placeholder = fn(props) {
    <div class="skeleton">"Loading profile…"</div>
}

export loaded = fn({userId}) {
    let user <== fetch(`https://api.example.com/users/{userId}`)
    <div class="profile">
        <h2>user.name</h2>
        <p>user.email</p>
    </div>
}
```

The page arrives instantly with the skeleton in place; the profile fills in as soon as it's ready. The Part's props (here `userId`) are sent with the load request.

## Lazy Loading: `part-lazy`

`part-lazy` is `part-load` for content the visitor might never scroll to — heavy charts, comment sections, image grids. The view is fetched when the Part approaches the viewport, and only once:

```parsley
<Part
    src={@./heavy-chart.part}
    view="placeholder"
    part-lazy="loaded"
    part-lazy-threshold={200}    // start 200px before it's visible (optional)
/>
```

Details worth knowing:

- Visibility is detected with an `IntersectionObserver` — no scroll handlers, no jank.
- `part-lazy-threshold` defaults to `0`; it's the distance in pixels below the fold at which loading starts.
- The view loads once. Scrolling away and back does not reload it.
- If the Part has no height (an empty placeholder, say), it can't intersect anything — the runtime notices and loads it immediately instead. Give your placeholder some size if you want genuine laziness.

## Live Data: `part-refresh`

`part-refresh` re-fetches the current view on an interval, in milliseconds. The classic demo is a clock:

```parsley
// clock.part
export default = fn(props) {
    <span class="clock">@now.time</span>
}
```

```parsley
<Part src={@./clock.part} part-refresh={1000}/>
```

Every second, the Part re-requests its current view with its current props and swaps in the answer. The same mechanism drives notification badges, job-status panels, and dashboards — pair it with a database query and the page stays live without a line of JavaScript.

The runtime is polite about it:

- Intervals below 100ms are clamped to 100ms.
- Refreshing pauses while the tab is hidden and resumes when it's visible again.
- The timer resets after any manual interaction, so a click never races a refresh.
- If the Part leaves the DOM, its timer stops.
- With `part-load` or `part-lazy`, auto-refresh starts *after* the initial load completes.

Note that "current view" means whatever view the Part showed last. If a click has switched a Part with `part-refresh` into an `edit` view, it's the `edit` view that refreshes — usually not what you want while someone is typing, so keep auto-refreshing Parts read-only, or nest one (a display Part with `part-refresh` inside an editor Part without).

## Controlling a Part from Outside

Everything so far lives *inside* the Part: its own buttons change its own views. `part-target` breaks that seal — any element on the page can drive any Part, by id:

```parsley
<Part src={@./editor.part} id="editor"/>

<table>
    for (person in people) {
        <tr>
            <td>person.name</td>
            <td>
                <button part-target="editor" part-click="edit" part-id={person.id}>"Edit"</button>
                <button part-target="editor" part-click="confirmDelete" part-id={person.id}>"Delete"</button>
            </td>
        </tr>
    }
</table>
```

Clicking a row's Edit button switches the *editor* Part to its `edit` view with that row's id. This is the backbone of the modal-editor pattern: one Part renders the popup, and buttons scattered through a table summon it.

`part-target` works on forms too — put it on the form (or its submit button) and the form's fields are sent to the target Part's view:

```parsley
<form part-target="results" part-submit="search">
    <input name="q" placeholder="Search…"/>
    <button>"Go"</button>
</form>

<Part src={@./results.part} id="results"/>
```

Unlike a plain `part-click`, targeted interactions merge their props *on top of the target's current props* — the target keeps its state and receives your additions.

### A Modal Editor

The pattern in full, condensed from a real app. The Part idles as an "add" button and becomes a modal on demand:

```parsley
// editor.part — four states: new, form, edit, confirmDelete

export new = fn(props) {
    <button part-click="form">"＋ Add person"</button>
}

export form = fn({person, error}) {
    <dialog open>
        <form part-submit="save">
            <input name="name" value={person?.name ?? ""}/>
            <input name="email" value={person?.email ?? ""}/>
            if (error) <p class="error">error</p>
            <button>"Save"</button>
            <button type="button" part-click="new">"Cancel"</button>
        </form>
    </dialog>
}

export edit = fn({id}) {
    let person = @DB <=?=> <SQL id={id}>SELECT * FROM people WHERE id = :id</SQL>
    form({person: person})
}

export confirmDelete = fn({id}) {
    let person = @DB <=?=> <SQL id={id}>SELECT * FROM people WHERE id = :id</SQL>
    <dialog open>
        <p>`Delete {person.name}?`</p>
        <button part-click="doDelete" part-id={id}>"Delete"</button>
        <button part-click="new">"Cancel"</button>
    </dialog>
}

export doDelete = fn({id}) {
    @DB <=!=> <SQL id={id}>DELETE FROM people WHERE id = :id</SQL>
    new({})
}

export default = new
```

Every state renders from the database by id, so the modal can be summoned from anywhere with two attributes and never shows stale data.

## Error Handling

Parts fail safe: **if a view request fails, the old content stays put.** No blank boxes, no broken half-rendered states — the visitor keeps what they had, and the failure is reported where developers look:

- The browser console logs `Failed to update Part:` with the status and the server's error message.
- The Part fires an `error` event you can subscribe to from JavaScript — see the [JavaScript API](parts-js.md#events).
- In dev mode, view errors appear in the [dev log panel](dev-tools.md) as `Part error: file.part → view()`.

The server responds with a meaningful status when something is wrong with the request itself:

| Status | Meaning |
|--------|---------|
| 404 | The Part was requested without a view (direct access is blocked), or the view doesn't exist |
| 400 | The props couldn't be parsed |
| 500 | The view function raised an error |

For errors that are part of the interaction — validation, mostly — don't rely on any of this. Render them. An error state is just another thing a view can show:

```parsley
export save = fn({id, text}) {
    if (text.trim() == "") {
        edit({id: id, error: "Text can't be empty"})
    } else {
        @DB <=!=> <SQL text={text} id={id}>UPDATE todos SET text = :text WHERE id = :id</SQL>
        default({id: id})
    }
}
```

## Debugging

- **Watch the wire.** Every interaction is a plain HTTP request — open the network tab and look for requests with a `_view` query parameter. The props are right there in the query string or form body, and the response is the HTML that replaced the Part.
- **Read the console.** Failed updates, unparseable props, and missing `part-target` ids are all logged.
- **Check the dev log.** In dev mode, `dev.log()` works inside view functions, and Part errors are reported with file and view names.
- **Inspect the wrapper.** The Part's `<div>` carries its state in plain sight: `data-part-src`, `data-part-view`, and `data-part-props` always reflect the last render.

## Organizing Part Files

`.part` files can live anywhere your handlers can reach — Basil doesn't impose a layout. Pick whichever reads best for your project:

```
# Parts alongside the pages that use them
site/
  dashboard.pars
  dashboard-clock.part

# A dedicated parts directory
site/
  dashboard.pars
  parts/
    clock.part

# Feature folders
site/
  dashboard/
    index.pars
    clock.part
```

The `src` attribute takes file-relative paths (`@./clock.part`) or project-root paths (`@~/parts/clock.part`), just like `import`.

## See Also

- [Parts](parts.md) — the essentials
- [Parts JavaScript API](parts-js.md) — `window.Parts`, events, and scripting
- [Database](database.md) — `@DB` and the query operators used in these examples
- [Dev Tools](dev-tools.md) — the dev log panel
