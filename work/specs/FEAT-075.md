---
id: FEAT-075
title: "Parts V1.3: Cross-Part Targeting and JavaScript API"
status: implemented
priority: medium
created: 2025-12-23
implemented: 2025-12-23
author: "@human + AI"
---

# FEAT-075: Parts V1.3: Cross-Part Targeting and JavaScript API

## Summary

Enable Parts to be controlled from outside their boundaries:
- `part-target="id"` attribute allows elements outside a Part to trigger that Part's reload
- `Parts` JavaScript API for programmatic Part control (refresh, read state)
- Enables persistent UI patterns like search boxes that don't reload while results do

## Motivation

Current Parts are self-contained — they can only trigger reloads of themselves. This prevents common UX patterns:

- **Search with persistent input**: Search box stays focused while results reload
- **Master-detail views**: Clicking a list item updates a detail panel
- **Dashboard controls**: Filter dropdowns that update multiple chart Parts
- **Wizard flows**: Navigation buttons that control a steps Part

## User Story

As a web developer, I want to control Parts from anywhere on the page, so that I can build interfaces where some elements persist (like a search box) while others reload (like search results), without losing user input or cursor position.

## Acceptance Criteria

### Cross-Part Targeting (`part-target`)
- [ ] `part-target="id"` attribute specifies which Part to reload
- [ ] Works with `part-click` (GET request)
- [ ] Works with `part-submit` (POST request, collects form data)
- [ ] `part-{prop}` attributes pass props to target Part
- [ ] Target element can be inside or outside any Part
- [ ] Error logged to console if target Part not found

### JavaScript API (`Parts` object)
- [ ] `Parts.refresh(id, props?, options?)` — Trigger Part reload
- [ ] `Parts.get(id)` — Get Part element and current props
- [ ] `Parts.on(id, event, callback)` — Listen for Part events
- [ ] Events: `beforeRefresh`, `afterRefresh`, `error`
- [ ] Debounce support for live search patterns

### Live Search Pattern
- [ ] Example implementation with `oninput` handler
- [ ] Debounce built into `Parts.refresh()` via options
- [ ] Cursor position preserved in search input

## Design Decisions

### Why `part-target` not `hx-target` style?
Keeps the `part-*` namespace consistent. HTMX uses `hx-target` but we're not HTMX — our Parts have views, props, and different semantics.

### Why a global `Parts` object?
- Simple, discoverable API
- No module imports needed in inline handlers
- Matches patterns like `document`, `localStorage`
- Can be namespaced to `window.BasilParts` if collision concerns arise

### Why debounce in the API?
Live search is the primary use case. Debouncing at the API level:
- Avoids boilerplate in every handler
- Cancels in-flight requests automatically
- Provides consistent behavior

### Form Data Collection
When `part-target` is on a submit button inside a `<form>`:
1. Prevent default form submission
2. Collect all form data
3. Merge with `part-{prop}` attributes
4. Send to target Part

When `part-target` is outside a form:
1. Only `part-{prop}` attributes are sent
2. Or use `part-form="formId"` to specify which form's data to collect

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Specification

### Cross-Part Targeting Syntax

**Button targeting a Part:**
```parsley
// Search box NOT in a Part (persists across searches)
<form id="search-form">
    <input name="q" type="search" placeholder="Search..."/>
    <button type="submit" 
            part-target="results" 
            part-submit="search">"Search"</button>
</form>

// Results Part with ID
<Part id="results" src={@./search-results.part}/>
```

**Generated HTML:**
```html
<form id="search-form">
    <input name="q" type="search" placeholder="Search..."/>
    <button type="submit" 
            data-part-target="results" 
            data-part-submit="search">Search</button>
</form>

<div id="results"
     data-part-src="/_parts/search-results" 
     data-part-view="default"
     data-part-props='{}'>
    <!-- Initial content -->
</div>
```

**Click targeting (master-detail):**
```parsley
// List of items (not in a Part, or in a different Part)
<ul>
    for (item in items) {
        <li>
            <button part-target="detail" 
                    part-click="show" 
                    part-itemId={item.id}>{item.name}</button>
        </li>
    }
</ul>

// Detail Part
<Part id="detail" src={@./item-detail.part}/>
```

### JavaScript API

```typescript
interface PartsAPI {
    /**
     * Refresh a Part with new props
     * @param id - Part element ID
     * @param props - Props to merge with existing (optional)
     * @param options - Refresh options (optional)
     */
    refresh(id: string, props?: Record<string, any>, options?: RefreshOptions): Promise<void>;
    
    /**
     * Get Part element and current state
     * @param id - Part element ID
     * @returns Part info or null if not found
     */
    get(id: string): PartInfo | null;
    
    /**
     * Listen for Part events
     * @param id - Part element ID (or '*' for all Parts)
     * @param event - Event name
     * @param callback - Event handler
     */
    on(id: string, event: PartEvent, callback: PartEventCallback): () => void;
}

interface RefreshOptions {
    view?: string;           // View to load (default: current view)
    debounce?: number;       // Debounce ms (default: 0, immediate)
    method?: 'GET' | 'POST'; // HTTP method (default: GET)
}

interface PartInfo {
    element: HTMLElement;    // The Part wrapper element
    src: string;             // Part source URL
    view: string;            // Current view name
    props: Record<string, any>; // Current props
}

type PartEvent = 'beforeRefresh' | 'afterRefresh' | 'error';

interface PartEventDetail {
    id: string;
    view: string;
    props: Record<string, any>;
    error?: Error;           // Only for 'error' event
}
```

### Usage Examples

**Live search with debounce:**
```html
<input type="search" 
       id="q" 
       placeholder="Search..."
       oninput="Parts.refresh('results', {q: this.value}, {debounce: 300})"/>

<div id="results" data-part-src="/_parts/search" ...>
    <!-- Results appear here -->
</div>
```

**Programmatic refresh:**
```javascript
// Refresh with new props
await Parts.refresh('user-profile', {userId: 123});

// Refresh to different view
await Parts.refresh('wizard', {}, {view: 'step2'});

// Get current state
const part = Parts.get('results');
console.log(part.props.q); // Current search query
```

**Event handling:**
```javascript
// Loading indicator
Parts.on('results', 'beforeRefresh', () => {
    document.getElementById('spinner').hidden = false;
});

Parts.on('results', 'afterRefresh', () => {
    document.getElementById('spinner').hidden = true;
});

// Error handling
Parts.on('*', 'error', ({id, error}) => {
    console.error(`Part ${id} failed:`, error);
});
```

### JavaScript Runtime Updates

Add to existing Parts runtime:

```javascript
// Global Parts API
window.Parts = {
    _debounceTimers: {},
    _listeners: {},
    
    refresh(id, props = {}, options = {}) {
        const part = document.getElementById(id);
        if (!part || !part.dataset.partSrc) {
            console.warn(`Parts.refresh: Part "${id}" not found`);
            return Promise.resolve();
        }
        
        // Handle debounce
        if (options.debounce > 0) {
            clearTimeout(this._debounceTimers[id]);
            return new Promise(resolve => {
                this._debounceTimers[id] = setTimeout(() => {
                    this._doRefresh(part, props, options).then(resolve);
                }, options.debounce);
            });
        }
        
        return this._doRefresh(part, props, options);
    },
    
    async _doRefresh(part, props, options) {
        const id = part.id;
        const src = part.dataset.partSrc;
        const currentProps = JSON.parse(part.dataset.partProps || '{}');
        const view = options.view || part.dataset.partView || 'default';
        const merged = {...currentProps, ...props};
        
        // Emit beforeRefresh
        this._emit(id, 'beforeRefresh', {id, view, props: merged});
        
        part.classList.add('part-loading');
        
        try {
            const params = new URLSearchParams({_view: view, ...merged});
            const url = `${src}?${params}`;
            
            const response = options.method === 'POST'
                ? await fetch(url, {method: 'POST', credentials: 'same-origin'})
                : await fetch(url, {credentials: 'same-origin'});
            
            if (!response.ok) throw new Error(`HTTP ${response.status}`);
            
            const html = await response.text();
            part.innerHTML = html;
            part.dataset.partProps = JSON.stringify(merged);
            part.dataset.partView = view;
            
            // Re-init nested Parts
            part.querySelectorAll('[data-part-src]').forEach(init);
            
            // Emit afterRefresh
            this._emit(id, 'afterRefresh', {id, view, props: merged});
        } catch (error) {
            console.error(`Parts.refresh error for "${id}":`, error);
            this._emit(id, 'error', {id, view, props: merged, error});
        } finally {
            part.classList.remove('part-loading');
        }
    },
    
    get(id) {
        const part = document.getElementById(id);
        if (!part || !part.dataset.partSrc) return null;
        
        return {
            element: part,
            src: part.dataset.partSrc,
            view: part.dataset.partView || 'default',
            props: JSON.parse(part.dataset.partProps || '{}')
        };
    },
    
    on(id, event, callback) {
        const key = `${id}:${event}`;
        if (!this._listeners[key]) this._listeners[key] = [];
        this._listeners[key].push(callback);
        
        // Return unsubscribe function
        return () => {
            this._listeners[key] = this._listeners[key].filter(cb => cb !== callback);
        };
    },
    
    _emit(id, event, detail) {
        // Specific listeners
        const key = `${id}:${event}`;
        (this._listeners[key] || []).forEach(cb => cb(detail));
        
        // Wildcard listeners
        const wildKey = `*:${event}`;
        (this._listeners[wildKey] || []).forEach(cb => cb(detail));
    }
};
```

### part-target Handler Updates

Add to existing click/submit handlers:

```javascript
function handlePartTarget(e) {
    const target = e.target.closest('[data-part-target]');
    if (!target) return;
    
    e.preventDefault();
    
    const partId = target.dataset.partTarget;
    const view = target.dataset.partClick || target.dataset.partSubmit || 'default';
    const method = target.dataset.partSubmit ? 'POST' : 'GET';
    
    // Collect props from part-* attributes
    const props = {};
    for (const [key, value] of Object.entries(target.dataset)) {
        if (key.startsWith('part') && 
            !['partTarget', 'partClick', 'partSubmit', 'partForm'].includes(key)) {
            const propName = key.slice(4).toLowerCase(); // partCount -> count
            props[propName] = coerceType(value);
        }
    }
    
    // Collect form data if in a form (for submit) or if part-form specified
    const formId = target.dataset.partForm;
    const form = formId ? document.getElementById(formId) : target.closest('form');
    
    if (form && method === 'POST') {
        const formData = new FormData(form);
        for (const [key, value] of formData.entries()) {
            props[key] = coerceType(value);
        }
    }
    
    Parts.refresh(partId, props, {view, method});
}

// Add to document listeners
document.addEventListener('click', handlePartTarget);
document.addEventListener('submit', e => {
    if (e.target.querySelector('[data-part-target][data-part-submit]')) {
        handlePartTarget(e);
    }
});
```

### Reserved Attributes Update

Add to reserved `part-*` attributes:
- `part-target` — ID of Part to control
- `part-form` — ID of form to collect data from (when not in a form)

### Edge Cases

1. **Target Part not found**: Log warning, no-op
2. **Circular targeting**: Part A targets Part B which targets Part A — allowed, no special handling needed
3. **Target during refresh**: New refresh cancels pending debounce, queues after current
4. **Part removed during refresh**: No-op, element gone
5. **Multiple targets**: Not supported in V1.3; use JavaScript API for complex cases
6. **Nested Part targeting**: Works — IDs are document-global

### Browser Compatibility

All features use standard APIs:
- `fetch` — Universal
- `FormData` — Universal
- `URLSearchParams` — Universal
- `Promise` — Universal (or polyfill)
- `dataset` — IE 11+

## Examples

### Live Search (Primary Use Case)

**Page:**
```parsley
// handlers/search.pars

<main>
    <h1>"Search"</h1>
    
    // Search input stays on page, never reloads
    <input type="search" 
           id="q" 
           placeholder="Type to search..."
           oninput="Parts.refresh('results', {q: this.value}, {debounce: 300})"
           autofocus/>
    
    // Results Part reloads as you type
    <Part id="results" src={@./search-results.part}/>
</main>
```

**Part:**
```parsley
// handlers/search-results.part

export default = fn(q) {
    if (!q || q == "") {
        <p class="hint">"Start typing to search..."</p>
    } else {
        let results = searchDatabase(q)
        if (results.length() == 0) {
            <p class="no-results">"No results for \"{q}\""</p>
        } else {
            <ul class="results">
                for (r in results) {
                    <li><a href={r.url}>{r.title}</a></li>
                }
            </ul>
        }
    }
}
```

### Master-Detail View

**Page:**
```parsley
// handlers/users.pars

<div class="master-detail">
    <aside class="master">
        <h2>"Users"</h2>
        <ul>
            for (user in users) {
                <li>
                    <button part-target="detail" 
                            part-click="show" 
                            part-userId={user.id}>{user.name}</button>
                </li>
            }
        </ul>
    </aside>
    
    <section class="detail">
        <Part id="detail" src={@./user-detail.part}/>
    </section>
</div>
```

### Form with Preview

**Page:**
```parsley
// handlers/editor.pars

<div class="editor-preview">
    <div class="editor">
        <textarea id="content" 
                  oninput="Parts.refresh('preview', {content: this.value}, {debounce: 500})"></textarea>
    </div>
    
    <div class="preview">
        <Part id="preview" src={@./markdown-preview.part}/>
    </div>
</div>
```

## Versioned Scope

### V1.3 (This Spec)

- `part-target="id"` attribute for cross-Part targeting
- `Parts.refresh(id, props, options)` with debounce support
- `Parts.get(id)` for reading Part state
- `Parts.on(id, event, callback)` for event handling
- Form data collection with `part-target` + `part-submit`
- `part-form="id"` for explicit form association

### V1.4 (Future)

- `part-target` accepting multiple IDs (refresh several Parts at once)
- `Parts.refreshAll(selector, props)` for bulk updates
- WebSocket integration for server-push Part updates
- `part-sync` for Parts that share state

## Related

- Depends on: FEAT-061 (Parts V1), FEAT-062 (Parts V1.1)
- Design: `work/design/DESIGN-parts.md`
- Updates needed: Parts runtime in `server/handler.go`
