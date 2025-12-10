---
id: FEAT-061
title: "Parts: Reloadable HTML Fragments"
status: done
priority: high
created: 2025-12-10
completed: 2025-12-10
author: "@human + AI"
---

# FEAT-061: Parts: Reloadable HTML Fragments

## Summary

Parts are reloadable HTML fragments with multiple views. A Part is a `.part` file that exports view functions, embedded in pages via `<Part src=@./path.part/>`. Parts enable rich, interactive UX patterns (editable forms, live updates, lazy loading) without heavy client-side frameworks — just ~50 lines of auto-injected JavaScript.

## User Story

As a web developer, I want to create interactive page fragments that can reload themselves without full page refreshes, so that I can build responsive, app-like experiences while keeping my code simple and server-rendered.

## Acceptance Criteria

### Core Functionality
- [x] `.part` files are recognized as Part modules
- [x] Parts export view functions (e.g., `export default = fn(props) { ... }`)
- [x] `<Part src=@./path.part props.../>` component renders Parts
- [x] Initial render is server-side (for SEO)
- [x] `part-click` attribute triggers view change on click
- [x] `part-submit` attribute triggers view change on form submit
- [x] `part-{propname}` attributes pass props to target view
- [x] Props are type-coerced (same rules as Basil forms)

### Server Behavior
- [x] `.part` files return 404 when accessed directly (without _view)
- [x] Part requests use `_view` query param to select view
- [x] GET requests for `part-click`, POST for `part-submit`
- [x] Auth/session inherited from cookies
- [x] Parts can be located anywhere accessible to calling script

### JavaScript Runtime
- [x] Auto-injected when page contains `<Part/>` components
- [x] `data-part-loading` class added during fetch (CSS hook: `part-loading`)
- [ ] `part-leave`/`part-enter` classes for CSS animations (deferred - V1.1)
- [x] On fetch error, old content remains visible
- [x] Nested Parts re-initialized after parent refresh

### Composition
- [x] Parts can contain other Parts
- [x] Multiple instances of same Part are independent

## Design Decisions

- **"View" not "state"**: "State" is overloaded; "view" clearly describes which visual representation is displayed
- **Exports not dictionary**: Using `export` feels more like regular Parsley code
- **`part-*` not `data-part-*`**: Cleaner to write; output can transform if needed
- **Server-side initial render**: SEO, performance, progressive enhancement
- **Inherit auth**: Parts are page fragments — same permissions as parent
- **Type coercion**: Same as Basil forms for consistency
- **No caching (V1)**: Avoids stale content issues; defer to future version
- **CSS animation hooks**: Zero overhead if unused; fully customizable

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Specification

### File Format (`.part`)

Parts are Parsley files with the `.part` extension that export view functions:

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

**Conventions:**
- `default` — Initial view when no view specified
- View functions receive props as parameters
- Views return HTML fragments

### `<Part/>` Component

```parsley
<Part 
    src=@./counter.part
    view="default"           # optional, defaults to "default"
    count={0}                # props passed to view function
/>
```

**Props:**

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `src` | path | yes | Path to `.part` file (`@` path literal) |
| `view` | string | no | Which view to render (default: `"default"`) |
| `*` | any | no | All other props passed to view function |

**Generated HTML:**

```html
<div data-part-src="/parts/counter" 
     data-part-view="default"
     data-part-props='{"count":0}'>
    <!-- Server-rendered view content -->
</div>
```

### Part Attributes

| Attribute | Trigger | HTTP Method | Description |
|-----------|---------|-------------|-------------|
| `part-click="view"` | click | GET | Switch to named view |
| `part-submit="view"` | form submit | POST | Switch to named view |
| `part-{name}={value}` | — | — | Pass prop to target view |

**Reserved attribute names:** `part-click`, `part-submit`, `part-load`, `part-refresh`

### Type Coercion

Props from query params and form data are coerced before passing to view:

| Input | Output Type | Example |
|-------|-------------|---------|
| `"true"` | boolean | `true` |
| `"false"` | boolean | `false` |
| Numeric string | number | `"5"` → `5`, `"3.14"` → `3.14` |
| Empty string | string | `""` |
| Other | string | `"hello"` → `"hello"` |

Props set via `part-{name}={expression}` in source retain original types.

### Server Request Format

```
GET /_parts/counter?_view=increment&count=5
POST /_parts/todo-item?_view=save
    Content-Type: application/x-www-form-urlencoded
    Body: id=123&text=Updated+text
```

- `_view` param selects view function
- All other params become props (type-coerced)
- POST body merged with query params
- Cookies sent for auth/session

### JavaScript Runtime

Injected automatically when page contains `<Part/>`:

```javascript
(function() {
    function refresh(part, view, props, method) {
        part.classList.add('data-part-loading');
        part.classList.add('part-leave');
        
        const src = part.dataset.partSrc;
        const url = new URL(src, location.origin);
        if (view) url.searchParams.set('_view', view);
        
        const options = { credentials: 'same-origin' };
        
        if (method === 'POST') {
            options.method = 'POST';
            options.headers = { 'Content-Type': 'application/x-www-form-urlencoded' };
            options.body = new URLSearchParams(props).toString();
        } else {
            Object.entries(props || {}).forEach(([k, v]) => 
                url.searchParams.set(k, v));
        }
        
        const duration = getTransitionDuration(part);
        
        fetch(url, options)
            .then(r => r.text())
            .then(html => {
                setTimeout(() => {
                    part.innerHTML = html;
                    part.classList.remove('part-leave');
                    part.classList.add('part-enter');
                    requestAnimationFrame(() => {
                        requestAnimationFrame(() => {
                            part.classList.remove('part-enter');
                        });
                    });
                    init(part);
                }, duration);
            })
            .finally(() => {
                part.classList.remove('data-part-loading');
            });
    }
    
    function getProps(el) {
        const props = {};
        const reserved = ['part-click', 'part-submit', 'part-load', 'part-refresh'];
        for (const attr of el.attributes) {
            if (attr.name.startsWith('part-') && !reserved.includes(attr.name)) {
                props[attr.name.slice(5)] = attr.value;
            }
        }
        return props;
    }
    
    function getTransitionDuration(el) {
        const style = getComputedStyle(el);
        const duration = parseFloat(style.transitionDuration) * 1000;
        return duration || 0;
    }
    
    function init(root) {
        root.querySelectorAll('[part-click]').forEach(el => {
            el.onclick = (e) => {
                e.preventDefault();
                const part = el.closest('[data-part-src]');
                refresh(part, el.getAttribute('part-click'), getProps(el), 'GET');
            };
        });
        
        root.querySelectorAll('form[part-submit]').forEach(form => {
            form.onsubmit = (e) => {
                e.preventDefault();
                const part = form.closest('[data-part-src]');
                const props = Object.fromEntries(new FormData(form));
                refresh(part, form.getAttribute('part-submit'), props, 'POST');
            };
        });
    }
    
    document.querySelectorAll('[data-part-src]').forEach(init);
})();
```

### CSS Classes

| Class | When Applied | Purpose |
|-------|--------------|---------|
| `data-part-loading` | During fetch | Style loading state |
| `part-leave` | Before content swap | Exit animation |
| `part-enter` | After content swap | Enter animation |

### Server Implementation

**Affected Components:**

- `server/handler.go` — Detect `.part` files, handle `_view` param, return 404 on direct access
- `server/parts.go` (new) — Part request handler, view dispatch, prop coercion
- `pkg/parsley/evaluator/` — `<Part/>` component implementation
- `pkg/parsley/evaluator/` — Part module loading (`.part` extension)

**Request Flow:**

1. JS sends request to Part URL with `_view` and props
2. Server resolves `.part` file path
3. Server loads Part module, finds exported view function
4. Server coerces props to appropriate types
5. Server calls view function with props
6. Server returns HTML fragment (no wrapper)

**Security:**

- Direct requests to `.part` URLs return 404
- Part requests only valid with `_view` param or via internal API
- Auth/session validated same as parent page
- CSRF not required (Parts inherit parent's CSRF context)

### Edge Cases

1. **Missing view**: If `_view` names non-existent export → 404
2. **Missing `default`**: Part without `default` export → error on initial render
3. **Nested Part refresh**: When parent refreshes, child Parts re-initialize
4. **Same Part multiple times**: Each instance is independent (own wrapper div)
5. **Part in Part**: Works — nested Parts get own `data-part-src` wrapper
6. **Network error**: Old content remains, `data-part-loading` removed

## Versioned Scope

### V1 (This Spec)

- `.part` files with exported view functions
- `<Part src=@... view="..." props.../>` component
- `part-click` (GET) and `part-submit` (POST)
- `part-{prop}` for passing props
- Server-side initial render
- JS runtime with loading/animation classes
- Nested Parts support
- Auth inheritance

### V1.1 (Future)

- `part-refresh={ms}` for auto-refresh
- `part-load="view"` for lazy loading

### V1.2 (Future)

- Responsive Parts with media query mapping
- Target other Parts on page

## Related

- Design: `docs/design/DESIGN-parts.md`
- Plan: `docs/plans/FEAT-061-plan.md` ✅ Complete
- Documentation:
  - `docs/parsley/CHEATSHEET.md` - Parts syntax and gotchas
  - `docs/parsley/reference.md` - Complete Parts reference
  - `docs/guide/parts.md` - Comprehensive guide with examples
  - `docs/guide/faq.md` - Parts FAQ entries
- Example: `examples/parts/` - Working counter demo

---

## Implementation Notes

**Completed:** 2025-12-10

### Key Files Modified

**Core Implementation:**
- `pkg/parsley/evaluator/evaluator.go` - Part module loading, <Part/> component, URL conversion, JS injection
- `server/parts.go` - Part request handler, prop parsing, type coercion
- `server/handler.go` - Part routing, JS runtime injection

**Tests:**
- `pkg/parsley/evaluator/parts_test.go` - Part module loading tests
- `pkg/parsley/evaluator/part_component_test.go` - <Part/> rendering tests
- `pkg/parsley/evaluator/part_attributes_test.go` - Data attribute tests
- `pkg/parsley/evaluator/nested_parts_test.go` - Nested Parts tests
- `pkg/parsley/evaluator/part_errors_test.go` - Error handling tests

**Example:**
- `examples/parts/handlers/counter.part` - Interactive counter Part
- `examples/parts/handlers/index.pars` - Main page using Part
- `examples/parts/basil.yaml` - Route configuration

### Issues Resolved During Implementation

1. **JSON Array Response**: Initial example had multiple top-level expressions, causing server to return JSON array instead of HTML string. Fixed by wrapping in template literal.

2. **Props Not Passing**: JavaScript runtime wasn't collecting `part-*` attributes from clicked elements. Fixed by merging clicked element's attributes with container props.

3. **404 on Part Requests**: URL generation was making paths relative to handlers directory instead of handler route. Fixed `convertPathToPartURL` to use handler route path as base.

### Deferred to BACKLOG

- CSS animation classes (`part-leave`/`part-enter`) - Use `part-loading` class for now
- Auto-refresh (`part-refresh={ms}`)
- Lazy loading (`part-load="view"`)
- Responsive Parts with media queries
- Part response caching

