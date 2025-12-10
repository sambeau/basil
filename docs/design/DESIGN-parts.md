# Design Document: Parts

**Status:** Draft  
**Created:** 2024-12-10  
**Author:** Discussion between human and AI

## Overview

A **Part** is a special kind of Parsley module for Basil that represents a reloadable HTML fragment with multiple views.

### Core Characteristics

1. Returns a fragment of HTML
2. Has one or more views (states) it can display
3. Can be dynamically replaced in the page via JavaScript
4. Server-side rendered initially (for SEO), then interactive

## Motivation

Parts enable rich, interactive UX patterns without heavy client-side frameworks:

- **Editable forms** that refresh on submit
- **Error states** that appear/clear dynamically
- **Periodic refresh** for live data (clocks, notifications)
- **Lazy loading** for expensive data fetches
- **Responsive alternatives** for different viewports

Inspired by [Hotwire Spark](https://github.com/hotwired/spark) and similar "HTML-over-the-wire" approaches.

## Design Principles

1. **No special YAML config** — Basil handles `.part` files automatically
2. **Parts are isolated** — Props passed at creation, no knowledge of parent page
3. **Server-side initial render** — For SEO, then JS makes them interactive
4. **Parts can contain Parts** — Nested composition
5. **Auth inherited** — Parts get the session context automatically
6. **Minimal JavaScript** — ~30 lines, no framework dependencies

---

## File Format

Parts use the `.part` file extension and export view functions:

```parsley
# counter.part

let render = fn(count) {
    <div>
        <span>{count}</span>
        <button part-click="increment" part-count={count + 1}>+</button>
    </div>
}

export default = render
export increment = render
```

Each export is a **view** — a function that returns HTML. Views can share implementation (as above) or be completely different.

### Conventions

- `default` — The initial view rendered when no view is specified
- View functions receive props as parameters
- Views return HTML fragments

---

## The `<Part/>` Component

Pages include Parts using the `<Part/>` component:

```parsley
<Part 
    src=@./counter.part
    view="default"           # optional, defaults to "default"
    count={0}                # props passed to the view function
/>
```

### Props

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `src` | path | yes | Path to the `.part` file (uses `@` path literal) |
| `view` | string | no | Which view to render (default: `"default"`) |
| `*` | any | no | All other props passed to the view function |

### Generated HTML

The `<Part/>` component renders to:

```html
<div data-part-src="/parts/counter" 
     data-part-view="default"
     data-part-props='{"count":0}'>
    <!-- Server-rendered initial view -->
    <div>
        <span>0</span>
        <button part-click="increment" part-count="1">+</button>
    </div>
</div>
```

The wrapper div:
- Identifies this as a Part (`data-part-src`)
- Stores the current view and props for JS
- Contains the server-rendered HTML

---

## Part Attributes

Attributes control how Parts reload themselves.

### `part-click`

Switches view on click:

```parsley
<button part-click="edit" part-id={id}>Edit</button>
```

### `part-submit`

Switches view on form submission:

```parsley
<form part-submit="save">
    <input name="text" value={text}/>
    <button type="submit">Save</button>
</form>
```

Form data becomes props for the target view.

### `part-{propname}`

Pass props when switching views:

```parsley
<button part-click="increment" part-count={count + 1}>+</button>
```

Props are extracted from attributes matching `part-*` (excluding reserved names).

### `part-refresh` (v1.1)

Auto-refresh on interval:

```parsley
<div part-refresh={1000}>
    {time.now().format("HH:mm:ss")}
</div>
```

### `part-load` (v1.1)

Immediately load a different view after mount (for lazy loading):

```parsley
<div part-load="loaded" part-userId={userId}>
    Loading...
</div>
```

---

## Examples

### Editable Todo Item

**Embedding in a page:**
```parsley
# In todo-list.pars
<ul>
    for todo in todos {
        <li>
            <Part src=@./parts/todo-item.part id={todo.id} text={todo.text} done={todo.done}/>
        </li>
    }
</ul>
```

**The Part:**
```parsley
# todo-item.part

export default = fn(id, text, done) {
    <div>
        <span class={done ? "done" : ""}>{text}</span>
        <button part-click="edit" part-id={id} part-text={text}>Edit</button>
    </div>
}

export edit = fn(id, text) {
    <form part-submit="save">
        <input type="hidden" name="id" value={id}/>
        <input name="text" value={text} autofocus/>
        <button type="submit">Save</button>
        <button type="button" part-click="default" part-id={id} part-text={text}>Cancel</button>
    </form>
}

export save = fn(id, text) {
    # Save to database
    let db = @sqlite("app.db")
    db.exec("UPDATE todos SET text = ? WHERE id = ?", text, id)
    
    # Return to default view
    default(id, text, false)
}
```

### Auto-Refresh Clock

**Embedding in a page:**
```parsley
# In dashboard.pars
<header>
    <h1>Dashboard</h1>
    <Part src=@./parts/clock.part/>
</header>
```

**The Part:**
```parsley
# clock.part

export default = fn() {
    let now = @time.now()
    <div part-refresh={1000}>
        {now.format("HH:mm:ss")}
    </div>
}
```

### Lazy-Loaded Profile

**Embedding in a page:**
```parsley
# In profile.pars
<main>
    <Part src=@./parts/user-profile.part userId={request.params.id}/>
</main>
```

**The Part:**
```parsley
# user-profile.part

export default = fn(userId) {
    <div part-load="loaded" part-userId={userId}>
        <div class="skeleton">Loading profile...</div>
    </div>
}

export loaded = fn(userId) {
    let user <== fetch("/api/users/{userId}")
    <div class="profile">
        <h2>{user.name}</h2>
        <p>{user.email}</p>
    </div>
}
```

### Nested Parts

**Embedding in a page:**
```parsley
# In app.pars
<main>
    <Part src=@./parts/dashboard.part userId={session.userId}/>
</main>
```

**The Part (containing other Parts):**
```parsley
# dashboard.part

export default = fn(userId) {
    <div>
        <Part src=@./user-header.part userId={userId}/>
        <Part src=@./notifications.part userId={userId}/>
        <Part src=@./recent-activity.part userId={userId}/>
    </div>
}
```

---

## Server Routing

### File Location

`.part` files can live anywhere in your project — next to your `.pars` files, in a dedicated `parts/` directory, or any other location accessible to the calling script. The developer chooses the organization that suits their project.

```
# Option A: Parts alongside pages
handlers/
  dashboard.pars
  dashboard-clock.part
  todo-list.pars
  todo-item.part

# Option B: Dedicated parts directory
handlers/
  dashboard.pars
  todo-list.pars
  parts/
    clock.part
    todo-item.part

# Option C: Mixed / feature-based
handlers/
  dashboard/
    index.pars
    clock.part
  todos/
    list.pars
    item.part
```

The `src` attribute in `<Part/>` uses file-relative paths, just like `import`:

```parsley
# From handlers/dashboard.pars
<Part src=@./dashboard-clock.part/>      # Option A
<Part src=@./parts/clock.part/>          # Option B
<Part src=@./dashboard/clock.part/>      # Option C
```

### Parts in Non-File-Routed Sites

Parts work equally well in single-handler mode or custom routing configurations. The Basil server uses the `.part` extension to distinguish Parts from regular modules — no special import syntax required.

### Direct Access Prevention

**Basil server will never serve a `.part` file directly.** Parts are only accessible through the `<Part/>` component or via the internal Part API (with `_view` parameter). Direct requests to `.part` files return 404.

This ensures:
- Parts can't be accessed outside their intended context
- Auth/session handling is always applied via the parent page
- No accidental exposure of partial HTML fragments

### Request Format

When the JS runtime requests a Part view:

```
GET /parts/counter?_view=increment&count=5
POST /parts/todo-item?_view=save
    Body: id=123&text=Updated+text
```

- `_view` parameter selects which view function to call
- All other parameters become props
- POST body (form data) merged with query params as props
- Auth/session inherited from cookies

---

## JavaScript Runtime

Basil auto-injects this script when a page contains `<Part/>` components:

```javascript
(function() {
    function refresh(part, view, props) {
        const src = part.dataset.partSrc;
        const url = new URL(src, location.origin);
        if (view) url.searchParams.set('_view', view);
        Object.entries(props || {}).forEach(([k, v]) => 
            url.searchParams.set(k, v));
        
        fetch(url, { credentials: 'same-origin' })
            .then(r => r.text())
            .then(html => { 
                part.innerHTML = html; 
                init(part); 
            });
    }
    
    function getProps(el) {
        const props = {};
        for (const attr of el.attributes) {
            if (attr.name.startsWith('part-') && 
                !['part-click','part-submit','part-load','part-refresh'].includes(attr.name)) {
                props[attr.name.slice(5)] = attr.value;
            }
        }
        return props;
    }
    
    function init(root) {
        root.querySelectorAll('[part-click]').forEach(el => {
            el.onclick = () => {
                const part = el.closest('[data-part-src]');
                refresh(part, el.getAttribute('part-click'), getProps(el));
            };
        });
        
        root.querySelectorAll('form[part-submit]').forEach(form => {
            form.onsubmit = (e) => {
                e.preventDefault();
                const part = form.closest('[data-part-src]');
                const props = Object.fromEntries(new FormData(form));
                refresh(part, form.getAttribute('part-submit'), props);
            };
        });
        
        root.querySelectorAll('[part-refresh]').forEach(el => {
            const ms = parseInt(el.getAttribute('part-refresh'));
            const part = el.closest('[data-part-src]');
            setInterval(() => refresh(part), ms);
        });
        
        root.querySelectorAll('[part-load]').forEach(el => {
            const part = el.closest('[data-part-src]');
            refresh(part, el.getAttribute('part-load'), getProps(el));
        });
    }
    
    document.querySelectorAll('[data-part-src]').forEach(init);
})();
```

---

## Versioned Scope

### V1 (MVP)

- `.part` files with exported view functions
- `<Part src="..." view="..." props.../>` component
- `part-click` for click-triggered view changes
- `part-submit` for form-triggered view changes
- `part-{prop}` for passing props
- Server-side initial render
- Self-contained JS runtime
- Parts can contain Parts
- Auth inherited from session

### V1.1

- `part-refresh={ms}` for polling/auto-refresh
- `part-load="view"` for lazy loading

### V1.2 (Future)

- Responsive Parts with media query mapping:
  ```parsley
  <Part 
      src=@./nav.part
      responsive={{
          "(max-width: 640px)": "mobile",
          "(max-width: 1024px)": "tablet",
          "default": "desktop"
      }}
  />
  ```
- Target other Parts on the page (not just self-replacement)

---

## Design Decisions

### Why "view" not "state"?

"State" is overloaded (React state, app state, server state). "View" clearly describes what you're looking at — which visual representation of the Part is displayed.

### Why exports not a dictionary?

Using `export` feels more like regular Parsley code. The alternative (returning a dictionary of views) was considered but felt less natural:

```parsley
# Alternative rejected:
{
    default: fn(count) { ... },
    increment: fn(count) { ... }
}
```

### Why `part-*` attributes not `data-part-*`?

`part-click` is cleaner to write. The output could transform to `data-part-click` for HTML validity if needed, but for ergonomics we use the shorter form in Parsley source.

### Why server-side initial render?

- SEO: Search engines see the content
- Performance: No flash of loading state
- Progressive enhancement: Works without JS (initial view)

### Why inherit auth?

Parts are fragments of the same page — they should have the same permissions. The `credentials: 'same-origin'` in fetch ensures cookies are sent.

### Why coerce prop types?

Props arriving via query params or form data are strings in HTTP. Parts use the same type coercion as Basil forms:

- `"true"`/`"false"` → boolean
- Numeric strings (`"5"`, `"3.14"`) → number
- Empty string → `""` (not null)
- Everything else → string

This keeps behavior consistent — developers don't need to learn different rules for Parts vs regular form handlers. Props are coerced before being passed to the view function.

Note: Props set via `part-{name}={expression}` in Parsley source are evaluated server-side and retain their original types.

---

## Resolved Design Questions

### Error Handling

**Decision: Leave old content on fetch failure (V1)**

If a Part fetch fails (network error, 500, etc.), the old content remains visible. This is the most graceful degradation — the user sees stale but valid HTML rather than a broken UI.

Developers who need custom error states can handle errors within their Part views, or we may add an `export error = fn(message) { ... }` convention in a future version.

### Loading Indicators

**Decision: Add `part-loading` class during fetch**

The JS runtime adds a `part-loading` class to the Part wrapper while fetching:

```javascript
function refresh(part, view, props) {
    part.classList.add('part-loading');
    
    fetch(url, { credentials: 'same-origin' })
        .then(r => r.text())
        .then(html => { 
            part.innerHTML = html; 
            init(part); 
        })
        .finally(() => {
            part.classList.remove('part-loading');
        });
}
```

Developers can style this as needed:

```css
[data-part-src].part-loading {
    opacity: 0.5;
    pointer-events: none;
}

/* Or add a spinner */
[data-part-src].part-loading::after {
    content: '';
    position: absolute;
    /* spinner styles */
}
```

This is minimal, opt-in (does nothing if you don't style it), and gives developers full control.

### Caching

**Decision: Defer (no caching in V1)**

Caching HTML fragments is tricky:
- Auth state changes → stale content
- Form data → stale inputs  
- Database changes → outdated data

For V1, every Part request hits the server fresh. If performance becomes an issue, future versions could add:
- `Cache-Control` headers on Part responses
- `part-cache={seconds}` attribute
- ETag/conditional requests

### Animation

**Decision: CSS class hooks for enter/exit transitions**

The JS runtime adds transition classes that developers can style:

```javascript
function refresh(part, view, props) {
    part.classList.add('part-leave');
    
    fetch(url)
        .then(r => r.text())
        .then(html => {
            // Wait for leave animation (or skip if no transition defined)
            const duration = getTransitionDuration(part);
            
            setTimeout(() => {
                part.innerHTML = html;
                part.classList.remove('part-leave');
                part.classList.add('part-enter');
                
                // Remove enter class after animation completes
                requestAnimationFrame(() => {
                    requestAnimationFrame(() => {
                        part.classList.remove('part-enter');
                    });
                });
                
                init(part);
            }, duration);
        });
}

function getTransitionDuration(el) {
    const style = getComputedStyle(el);
    const duration = parseFloat(style.transitionDuration) * 1000;
    return duration || 0;  // 0 = no animation, swap immediately
}
```

Example CSS for fade animation:

```css
/* Opt-in: only Parts with transitions animate */
.animated-part {
    transition: opacity 0.2s ease;
}

.animated-part.part-leave {
    opacity: 0;
}

.animated-part.part-enter {
    opacity: 0;
    animation: part-fade-in 0.2s ease forwards;
}

@keyframes part-fade-in {
    to { opacity: 1; }
}
```

This approach:
- Has zero overhead if you don't use it (no transition = instant swap)
- Is fully customizable via CSS
- Requires minimal additional JS (~10 lines)
- Needs no new attributes to learn

Future versions could add an `animate` attribute for preset animations or view-to-view transition maps.

---

## Prior Art

- [Hotwire Spark](https://github.com/hotwired/spark) — Ruby on Rails hot reloading
- [HTMX](https://htmx.org/) — HTML attributes for AJAX (`hx-get`, `hx-swap`)
- [Turbo Frames](https://turbo.hotwired.dev/handbook/frames) — Hotwire's `<turbo-frame>` elements
- [Livewire](https://laravel-livewire.com/) — Laravel's full-component server state
- [Alpine.js](https://alpinejs.dev/) — Client-side reactivity, pairs with server fragments
- [Unpoly](https://unpoly.com/) — `[up-follow]`, `[up-target]` for fragment updates

HTMX is the closest spiritual match — minimal JS, server-rendered HTML fragments, declarative attributes.
