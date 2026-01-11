# Server-Enhanced Components

Design document for Basil's next generation of components that leverage the unique position of being both a backend server and frontend renderer.

## Overview

Basil has a unique architectural advantage: it's a server that renders HTML and serves JavaScript. This means we can create components that seamlessly bridge server and client in ways that traditional frontend frameworks or pure backend templates cannot.

### The Server+Client Pattern

Traditional approaches:
- **Backend templates** (Jinja, ERB, Go templates): Server renders HTML, client interactions require full page reloads or manual AJAX
- **Frontend frameworks** (React, Vue, Svelte): Client renders everything, server is just an API, lots of boilerplate for data fetching
- **Islands architecture** (Astro, Fresh): Static HTML with hydrated islands, but still requires explicit API layer

**Basil's opportunity:**
- Server has the data and logic
- Server controls what HTML is sent
- Server controls what JavaScript is included
- Parts system already enables server-rendered partial updates
- We can create components where the JS "just knows" how to talk to the server

```
┌─────────────────────────────────────────────────────────────┐
│                        Basil Server                          │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ Component Definition (Parsley)                          ││
│  │                                                         ││
│  │ <SearchField                                            ││
│  │     source={fn(q) { db.query(...) }}  ← Server logic    ││
│  │     displayKey="name"                                   ││
│  │ />                                                      ││
│  └─────────────────────────────────────────────────────────┘│
│                           │                                  │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ Rendered Output                                         ││
│  │                                                         ││
│  │ <search-field data-endpoint="/_/search/abc123">        ││
│  │   <input type="text" ...>                               ││
│  │ </search-field>                                         ││
│  │                                                         ││
│  │ + Auto-generated endpoint that invokes source function  ││
│  │ + JavaScript that knows how to call that endpoint       ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

The key insight: **the server can generate both the HTML AND a temporary endpoint for that specific component instance**, eliminating the need for users to manually create API routes.

---

## Component Proposals

### 1. SearchField / ComboField

**Category:** Server+Client Pattern (High Value Prototype)

**Problem:** Searching large datasets requires server-side filtering, but traditional approaches require manually creating API endpoints.

**Proposed API:**

```parsley
// Option A: Inline function (generates temporary endpoint)
<SearchField 
    name="product_id"
    label="Search Products"
    source={fn(query) { 
        db.query(
            "SELECT id, name, price FROM products WHERE name ILIKE ? LIMIT 10",
            "%" ++ query ++ "%"
        )
    }}
    displayKey="name"
    valueKey="id"
    minChars={2}
    debounce={300}
/>

// Option B: Named Part (more explicit, reusable)
<SearchField 
    name="customer_id"
    label="Customer"
    src="search/customers.part"
    displayKey="name"
    valueKey="id"
/>

// Option C: Explicit endpoint (traditional)
<SearchField 
    name="city"
    label="City"
    endpoint="/api/cities/search"
    displayKey="name"
    valueKey="code"
/>
```

**Rendered HTML:**
```html
<div class="search-field" id="field-product_id">
    <label for="field-product_id-input">Search Products</label>
    <input 
        type="text" 
        id="field-product_id-input"
        autocomplete="off"
        aria-autocomplete="list"
        aria-controls="field-product_id-listbox"
        aria-expanded="false"
        role="combobox"
    />
    <input type="hidden" name="product_id" value=""/>
    <ul id="field-product_id-listbox" role="listbox" hidden></ul>
</div>
```

**JavaScript behavior:**
1. User types → debounced fetch to endpoint
2. Results rendered in listbox
3. Keyboard navigation (↑↓ to select, Enter to confirm, Escape to close)
4. Selection updates hidden input value
5. Works without JS (falls back to regular text input)

**Implementation approach for Option A (inline function):**
```go
// When rendering SearchField with source function:
// 1. Serialize the function to a unique ID
// 2. Register a temporary endpoint: /_/search/{id}
// 3. Endpoint invokes the function with query param
// 4. Emit data-endpoint="/_/search/{id}" on the element

type SearchEndpoint struct {
    ID       string
    Function *evaluator.Function
    Env      *evaluator.Environment
    Expires  time.Time  // Auto-cleanup after 1 hour of no use
}
```

**Complexity:** Medium-High
- Server: Temporary endpoint registry, function serialization
- Client: ~150 lines JS for combobox behavior
- Accessibility: Full ARIA combobox pattern

**Open questions:**
- How to handle function closure state? (captured variables)
- Security: How to prevent endpoint enumeration/abuse?
- Caching: Should results be cached? Per-user? Globally?

---

### 2. SortableList

**Category:** Server+Client with External Library

**Problem:** Drag-and-drop reordering needs immediate visual feedback (client) but persistent storage (server).

**Proposed API:**

```parsley
<SortableList 
    items={tasks}
    endpoint="/api/tasks/reorder"
    itemKey="id"
    group="kanban"                    // Cross-list dragging
    handle=".drag-handle"             // Optional drag handle selector
    animation={150}                   // ms
>
    {task => 
        <li class="task">
            <span class="drag-handle">"⋮⋮"</span>
            task.title
        </li>
    }
</SortableList>
```

**Server endpoint contract:**
```
POST /api/tasks/reorder
Content-Type: application/json

{
    "ids": ["task-3", "task-1", "task-2"],  // New order
    "movedId": "task-3",                     // Which item moved
    "fromIndex": 2,                          // Where it was
    "toIndex": 0,                            // Where it went
    "fromGroup": "todo",                     // If cross-list
    "toGroup": "done"                        // If cross-list
}
```

**Implementation:**
- Use SortableJS (10KB, MIT, no dependencies)
- Optimistic UI: Reorder immediately, rollback on error
- Server returns success/error, optionally new item state

**Complexity:** High
- External dependency management
- Cross-list state synchronization
- Error handling and rollback

**Recommendation:** Implement as optional extension, not core. Document the pattern for users who want it.

---

### 3. Pagination

**Category:** Pure Server Component (Quick Win)

**Problem:** Every app with lists needs pagination. Currently users must build it manually.

**Proposed API:**

```parsley
let page = (basil.http.request.query.page ?? "1").toInt()
let perPage = 20
let {rows, total} = db.paginate("SELECT * FROM products ORDER BY name", page, perPage)

<DataTable rows={rows} columns={["name", "price"]}/>

<Pagination 
    current={page}
    total={total}
    perPage={perPage}
    href="/products?page={page}"      // URL template
/>

// Or for Part-based navigation (no page reload):
<Pagination 
    current={page}
    total={total}
    perPage={perPage}
    target="#product-list"            // Part to update
    param="page"                      // Query param name
/>
```

**Rendered HTML:**
```html
<nav aria-label="Pagination" class="pagination">
    <a href="/products?page=1" aria-label="First page">«</a>
    <a href="/products?page=2" aria-label="Previous page">‹</a>
    
    <a href="/products?page=1">1</a>
    <a href="/products?page=2">2</a>
    <span aria-current="page" class="pagination-current">3</span>
    <a href="/products?page=4">4</a>
    <a href="/products?page=5">5</a>
    <span class="pagination-ellipsis">…</span>
    <a href="/products?page=20">20</a>
    
    <a href="/products?page=4" aria-label="Next page">›</a>
    <a href="/products?page=20" aria-label="Last page">»</a>
</nav>
```

**Props:**

| Prop | Type | Description |
|------|------|-------------|
| `current` | number | Current page (required) |
| `total` | number | Total item count (required) |
| `perPage` | number | Items per page. Default: 20 |
| `href` | string | URL template with `{page}` placeholder |
| `target` | string | Part target selector (for AJAX pagination) |
| `param` | string | Query param name. Default: "page" |
| `window` | number | Pages to show around current. Default: 2 |
| `showFirst` | boolean | Show first/last buttons. Default: true |
| `showPrev` | boolean | Show prev/next buttons. Default: true |
| `labels` | object | Custom labels for buttons |

**Complexity:** Low - Pure server-side rendering, no JS required (optional Part integration)

**Database helper needed:**
```parsley
// New db.paginate() method
let {rows, total, page, perPage, totalPages} = db.paginate(query, page, perPage)
```

---

### 4. Toasts / Flash Messages

**Category:** Pure Server Component with Client Animation (Quick Win)

**Problem:** Flash messages exist in session, but there's no component to render them nicely.

**Proposed API:**

```parsley
// In layout/page component:
<Toasts position="top-right" duration={5000}/>

// In handler (already exists):
basil.session.flash("success", "User saved successfully!")
basil.session.flash("error", "Failed to delete item")
basil.session.flash("info", "Your session will expire in 5 minutes")
```

**Rendered HTML:**
```html
<div class="toasts toasts-top-right" aria-live="polite" aria-atomic="true">
    <div class="toast toast-success" role="alert" data-duration="5000">
        <span class="toast-icon">✓</span>
        <span class="toast-message">User saved successfully!</span>
        <button class="toast-dismiss" aria-label="Dismiss">×</button>
    </div>
</div>
```

**JavaScript behavior:**
- Auto-dismiss after duration
- Manual dismiss on click
- Pause timer on hover
- Stacking animation for multiple toasts
- Accessible announcements via aria-live

**Complexity:** Low
- Server: Read flash messages, render HTML
- Client: ~50 lines JS for animation/dismiss

---

### 5. Skeleton Loaders

**Category:** Pure CSS Component (Quick Win)

**Problem:** Parts with `lazy={true}` show nothing while loading. Skeleton loaders provide better UX.

**Proposed API:**

```parsley
// As Part placeholder:
<Part src="dashboard.part" lazy={true}>
    <Skeleton type="card" count={3}/>
</Part>

// Standalone:
<Skeleton type="text" lines={4}/>
<Skeleton type="avatar" size="lg"/>
<Skeleton type="table" rows={5} cols={4}/>
```

**Types:**

| Type | Description |
|------|-------------|
| `text` | Paragraph lines with varying widths |
| `heading` | Single wider line |
| `avatar` | Circle |
| `image` | Rectangle with aspect ratio |
| `card` | Image + heading + text lines |
| `table` | Grid of cells |
| `list` | Repeated list items |

**Rendered HTML:**
```html
<div class="skeleton skeleton-card" aria-hidden="true">
    <div class="skeleton-image"></div>
    <div class="skeleton-heading"></div>
    <div class="skeleton-text"></div>
    <div class="skeleton-text" style="width: 80%"></div>
</div>
```

**CSS (shimmer animation):**
```css
.skeleton {
    background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%);
    background-size: 200% 100%;
    animation: skeleton-shimmer 1.5s infinite;
}
@keyframes skeleton-shimmer {
    0% { background-position: 200% 0; }
    100% { background-position: -200% 0; }
}
```

**Complexity:** Very Low - CSS only, no JS

---

### 6. Modal with Part Content

**Category:** Enhanced Existing Component

**Problem:** `<Dialog>` exists but loading content from Parts requires manual wiring.

**Proposed Enhancement:**

```parsley
// Trigger button that loads Part into modal:
<Button modal="edit-user.part?id={user.id}">"Edit User"</Button>

// Or declarative modal with Part source:
<Dialog id="edit-modal" src="edit-form.part" title="Edit User">
    <Skeleton type="form"/>  // Placeholder while loading
</Dialog>
<Button toggle="#edit-modal" data-params="id={user.id}">"Edit"</Button>

// Part file (edit-form.part):
let id = basil.http.request.query.id
let user = db.queryOne("SELECT * FROM users WHERE id = ?", id)

<form method="POST" action="/users/{id}">
    <TextField name="name" label="Name" value={user.name}/>
    <Button type="submit">"Save"</Button>
</form>
```

**Behavior:**
1. Click button → Open modal with skeleton
2. Fetch Part content → Replace skeleton
3. Form submit → Close modal, optionally refresh parent Part

**Complexity:** Medium
- Extend existing Dialog component
- Part loading integration
- Focus management

---

### 7. InfiniteScroll

**Category:** Server+Client Pattern

**Problem:** Loading more items as user scrolls requires coordination between scroll detection and server fetching.

**Proposed API:**

```parsley
<InfiniteList
    src="items.part"
    cursor={lastItem?.id}           // For cursor-based pagination
    // OR
    page={page}                      // For offset-based pagination
    threshold={200}                  // px from bottom to trigger
    loading={<Skeleton type="list" count={3}/>}
    end={"No more items"}
>
    {items.map(item => <ItemCard item={item}/>)}
</InfiniteList>
```

**How it works:**
1. Render initial items
2. IntersectionObserver watches sentinel element near bottom
3. When visible, fetch next page from Part
4. Append results, update cursor
5. Repeat until server returns empty or `end: true`

**Part contract:**
```parsley
// items.part
let cursor = basil.http.request.query.cursor
let items = db.query(
    "SELECT * FROM items WHERE id > ? ORDER BY id LIMIT 20", 
    cursor ?? 0
)
let hasMore = items.length == 20

// Return items plus metadata
<div data-cursor={items[-1]?.id} data-has-more={hasMore}>
    {items.map(item => <ItemCard item={item}/>)}
</div>
```

**Complexity:** Medium
- Client: IntersectionObserver + Part fetching
- Server: Cursor/pagination convention

---

### 8. FileField with Upload Progress

**Category:** Server+Client Pattern

**Problem:** File uploads need progress indication, drag-drop, and preview.

**Proposed API:**

```parsley
<FileField
    name="attachments"
    label="Upload Files"
    accept="image/*,.pdf"
    multiple={true}
    maxSize={10 * 1024 * 1024}       // 10MB per file
    maxFiles={5}
    endpoint="/api/upload"            // Where to POST files
    preview={true}                    // Show image thumbnails
    dragDrop={true}                   // Enable drag-drop zone
/>
```

**Upload endpoint contract:**
```
POST /api/upload
Content-Type: multipart/form-data

Response:
{
    "id": "file-abc123",
    "name": "photo.jpg",
    "size": 1024000,
    "url": "/uploads/photo.jpg",
    "thumbnail": "/uploads/photo-thumb.jpg"
}
```

**Rendered HTML:**
```html
<div class="file-field" id="field-attachments">
    <label>Upload Files</label>
    <div class="file-field-dropzone" data-drag-drop="true">
        <input type="file" multiple accept="image/*,.pdf" hidden/>
        <p>Drag files here or <button type="button">browse</button></p>
    </div>
    <ul class="file-field-list">
        <!-- Populated dynamically -->
    </ul>
    <input type="hidden" name="attachments" value="[]"/>
</div>
```

**JavaScript behavior:**
- Drag-drop zone with visual feedback
- File validation (size, type, count)
- XHR upload with progress events
- Thumbnail preview for images
- Remove button for each file
- Hidden input stores uploaded file IDs as JSON array

**Complexity:** High
- Client: ~200 lines JS
- Server: Upload endpoint (may already exist in user's app)

---

### 9. LiveForm / Auto-Save Fields

**Category:** Server+Client Pattern

**Problem:** Some forms (settings, profiles) should save automatically without explicit submit.

**Proposed API:**

```parsley
// Whole form auto-saves:
<LiveForm endpoint="/api/settings" debounce={1000}>
    <TextField name="siteName" label="Site Name"/>
    <Checkbox name="maintenance" label="Maintenance Mode"/>
</LiveForm>

// Individual field auto-saves:
<TextField 
    name="bio" 
    label="Bio" 
    autoSave="/api/profile"
    debounce={500}
/>
```

**Behavior:**
1. User changes field
2. Debounce delay
3. PATCH request with changed fields only
4. Show "Saving..." indicator
5. Show "Saved" or error state
6. Handle conflicts (optimistic locking via ETag?)

**Endpoint contract:**
```
PATCH /api/settings
Content-Type: application/json

{"siteName": "New Name"}

Response 200: {"siteName": "New Name", "updatedAt": "..."}
Response 409: {"conflict": true, "serverValue": "Other Name"}
```

**Complexity:** Medium
- Client: Debounce, PATCH, status indication
- Server: Partial update handling

---

### 10. Push-Parts (Server-Sent Events)

**Category:** Advanced Server+Client Pattern (Future)

**Problem:** Some UIs need real-time updates (notifications, dashboards, chat) without polling.

**Proposed API:**

```parsley
// Subscribe to updates:
<Part src="notifications.part" push={true} channel="user:{user.id}"/>

// Server triggers update (in another handler):
basil.push("user:123", "notifications")  // Refresh Part for all subscribers

// Or push data directly:
basil.push("user:123", "notifications", {count: 5})
```

**Architecture:**

```
┌─────────────┐         ┌─────────────────┐
│   Browser   │◄───SSE──│  Basil Server   │
│             │         │                 │
│ EventSource │         │ ┌─────────────┐ │
│ connection  │         │ │ Pub/Sub Hub │ │
│             │         │ └─────────────┘ │
└─────────────┘         │       ▲         │
                        │       │         │
                        │ basil.push()    │
                        │       │         │
                        │ ┌─────────────┐ │
                        │ │  Handler    │ │
                        │ └─────────────┘ │
                        └─────────────────┘
```

**Implementation considerations:**
- SSE endpoint: `/_/events?channels=user:123,global`
- In-memory pub/sub for single server
- Redis pub/sub for multi-server scaling
- Reconnection handling
- Connection limits (max connections per user?)
- Fallback to polling for older browsers

**Complexity:** Very High
- Significant server infrastructure
- Connection management
- Scaling considerations

**Recommendation:** Start with polling (`refresh={5000}` on Parts), implement SSE as opt-in v0.4+ feature.

---

### 11. CommandPalette / Spotlight

**Category:** Luxury Feature (Defer)

**Problem:** Power users want keyboard-driven navigation.

**Proposed API:**

```parsley
<CommandPalette 
    hotkey="cmd+k"
    placeholder="Search or type a command..."
    sources={[
        {
            name: "Pages",
            src: "search/pages.part",
            icon: "📄"
        },
        {
            name: "Users", 
            endpoint: "/api/users/search",
            icon: "👤"
        },
        {
            name: "Commands",
            items: [
                {label: "New User", action: "/users/new", icon: "➕"},
                {label: "Settings", action: "/settings", icon: "⚙️"},
                {label: "Logout", action: "/logout", icon: "🚪"}
            ]
        }
    ]}
/>
```

**Complexity:** High
- Modal with search input
- Multi-source async search
- Keyboard navigation
- Recent/frequent items
- Fuzzy matching

**Recommendation:** Defer. Cool but not essential for v1.

---

## Priority Matrix

| Component | User Value | Complexity | Dependencies | Priority |
|-----------|------------|------------|--------------|----------|
| **Pagination** | ⭐⭐⭐⭐⭐ | Low | None | 🟢 **P0 - Do Now** |
| **Toasts** | ⭐⭐⭐⭐ | Low | None | 🟢 **P0 - Do Now** |
| **Skeleton** | ⭐⭐⭐ | Very Low | None | 🟢 **P0 - Do Now** |
| **SearchField** | ⭐⭐⭐⭐⭐ | Medium-High | Prototype needed | 🟡 **P1 - Prototype** |
| **ComboField** | ⭐⭐⭐⭐⭐ | Medium | Shares SearchField code | 🟡 **P1 - Prototype** |
| **Modal+Part** | ⭐⭐⭐⭐ | Medium | Existing Dialog | 🟡 **P1 - Prototype** |
| **InfiniteScroll** | ⭐⭐⭐⭐ | Medium | Parts system | 🟡 **P1 - Explore** |
| **LiveForm** | ⭐⭐⭐ | Medium | None | 🟠 **P2 - Later** |
| **FileField** | ⭐⭐⭐⭐ | High | Upload endpoint | 🟠 **P2 - Later** |
| **SortableList** | ⭐⭐⭐⭐ | High | SortableJS | 🟠 **P2 - Extension** |
| **Push-Parts** | ⭐⭐⭐⭐ | Very High | SSE infrastructure | 🔴 **P3 - v0.4+** |
| **WYSIWYG** | ⭐⭐⭐ | Very High | Editor library | 🔴 **P3 - Document pattern** |
| **CommandPalette** | ⭐⭐⭐ | High | None | 🔴 **P3 - Defer** |

---

## Prototype Plan: SearchField

The SearchField is the best candidate for prototyping the server+client pattern because:
1. High user value
2. Medium complexity
3. Pattern is reusable for ComboField, InfiniteScroll
4. Tests the temporary endpoint concept

### Prototype Scope

**Phase 1: Static endpoint (prove the JS works)**
```parsley
<SearchField 
    name="product"
    label="Product"
    endpoint="/api/products/search"
    displayKey="name"
    valueKey="id"
/>
```
- Implement client-side combobox behavior
- Use existing API endpoint
- ~150 lines JS

**Phase 2: Part-based results (prove Part integration)**
```parsley
<SearchField 
    name="product"
    label="Product"
    src="search/products.part"
    valueKey="id"
/>
```
- Results rendered by Part
- Full HTML flexibility for result items
- Leverages existing Part infrastructure

**Phase 3: Inline function (prove the magic)**
```parsley
<SearchField 
    name="product"
    label="Product"
    source={fn(q) { db.query("SELECT * FROM products WHERE name ILIKE ?", "%" ++ q ++ "%") }}
    displayKey="name"
    valueKey="id"
/>
```
- Server generates temporary endpoint
- Function is registered and invokable
- This is the "wow" feature

### Technical Spikes Needed

1. **Temporary endpoint registry:**
   - How to register a function as a callable endpoint?
   - How to pass environment/closure state?
   - How to garbage collect unused endpoints?

2. **Security model:**
   - How to prevent endpoint enumeration?
   - Rate limiting per endpoint?
   - CSRF considerations?

3. **Function serialization:**
   - Can we serialize a Parsley function reference?
   - Or do we need to keep it in memory?
   - What about server restarts?

---

## Prototype Plan: Quick Wins

### Pagination

1. Create `server/prelude/components/pagination.pars`
2. Add `db.paginate()` helper to evaluator
3. Props: current, total, perPage, href, target
4. Output: nav with proper ARIA
5. Test with DataTable example

### Toasts

1. Create `server/prelude/components/toasts.pars`
2. Read from `basil.session.flash` in render
3. Add ~50 lines JS to basil.js for animation/dismiss
4. Props: position, duration
5. Test in example app

### Skeleton

1. Create `server/prelude/components/skeleton.pars`
2. Pure CSS shimmer animation
3. Types: text, heading, avatar, image, card, table, list
4. Props: type, count, lines, rows, cols
5. Add CSS to prelude

---

## Open Questions

1. **Temporary endpoints lifetime:** How long should a generated endpoint live? Options:
   - Until server restart (simple, may accumulate)
   - TTL with activity refresh (1 hour, reset on each call)
   - Tied to session (cleaned up on logout)
   - Reference counted (cleaned when no pages reference it)

2. **Function closure capture:** What happens if `source` function references variables from the handler scope?
   ```parsley
   let category = "electronics"
   <SearchField source={fn(q) { 
       // Does this capture `category`?
       db.query("... WHERE category = ?", category) 
   }}/>
   ```
   Options:
   - Capture at render time (snapshot)
   - Error if closure detected
   - Pass as explicit params

3. **Multi-server deployment:** Temporary endpoints only exist in one server's memory. Options:
   - Sticky sessions (load balancer)
   - Shared endpoint registry (Redis)
   - Serialize function to endpoint URL (complex)

4. **Caching strategy for search results:**
   - No caching (always fresh)
   - Short TTL (5 seconds)
   - Per-query caching with invalidation

5. **Extension mechanism:** Should complex components like SortableList be:
   - Built into core (increases bundle size)
   - Separate import (`@std/sortable`)
   - External package with documentation
   - Plugin system?

---

## Next Steps

1. **Immediate (this sprint):**
   - [ ] Implement Pagination component
   - [ ] Implement Toasts component
   - [ ] Implement Skeleton component

2. **Next sprint:**
   - [ ] Prototype SearchField Phase 1 (static endpoint)
   - [ ] Prototype SearchField Phase 2 (Part-based)
   - [ ] Design temporary endpoint registry

3. **Future:**
   - [ ] SearchField Phase 3 (inline function)
   - [ ] ComboField (share SearchField code)
   - [ ] Modal+Part integration
   - [ ] InfiniteScroll

---

## Appendix: JavaScript Bundle Size Considerations

Current `basil.js`: ~290 lines (unminified)

Estimated additions:

| Feature | Lines | Minified |
|---------|-------|----------|
| SearchField combobox | ~150 | ~3KB |
| Toast animations | ~50 | ~1KB |
| InfiniteScroll | ~80 | ~2KB |
| FileField upload | ~200 | ~4KB |
| SortableJS (external) | - | ~10KB |

**Target:** Keep core basil.js under 10KB minified. Larger features should be:
- Lazy loaded when component is used
- Or documented as "bring your own library"

---

## Appendix: Related Work

- **HTMX:** Similar philosophy (HTML-driven interactions), but requires learning htmx attributes. Basil can be more declarative.
- **Hotwire/Turbo:** Rails approach to partial updates. Similar to Parts.
- **Alpine.js:** Minimal JS framework. Could complement Basil for complex interactions.
- **LiveView (Phoenix):** Full server-rendered interactivity via WebSocket. Inspirational but higher complexity.

Basil's advantage is being the whole stack—we control server, templates, and client, so we can optimize the integration.
