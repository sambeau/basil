# SortableList Component Design

## Overview

A drag-and-drop sortable list component powered by SortableJS, using idiomatic Parsley syntax.

**Goals:**
- Sensible defaults that "just work" (animation, touch, keyboard)
- Simple, composable API
- Backend integration via endpoint
- Cross-list dragging via groups
- Stylable drag states

## API

### Basic Usage

```parsley
<SortableList endpoint="/api/tasks/reorder">
    for (task in tasks) {
        <SortableItem id={task.id}>
            <div class="card">{task.title}</div>
        </SortableItem>
    }
</SortableList>
```

### Cross-List (Kanban)

```parsley
<div class="kanban">
    for (col in ["todo", "doing", "done"]) {
        let columnTasks = tasks.filter(fn(t) { t.status == col })
        <div class="column">
            <h3>{col}</h3>
            <SortableList endpoint="/api/tasks/move" group="kanban" listId={col}>
                for (task in columnTasks) {
                    <SortableItem id={task.id}>
                        <div class="card">{task.title}</div>
                    </SortableItem>
                }
            </SortableList>
        </div>
    }
</div>
```

### With Drag Handle

```parsley
<SortableList endpoint="/api/items/reorder" handle=".drag-handle">
    for (item in items) {
        <SortableItem id={item.id}>
            <div class="item">
                <span class="drag-handle">⋮⋮</span>
                <span>{item.name}</span>
            </div>
        </SortableItem>
    }
</SortableList>
```

## Component Props

### `<SortableList>`

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `endpoint` | string | required | URL to POST reorder data |
| `group` | string | - | Group name for cross-list dragging |
| `listId` | string | - | Identifier for this list (sent to backend) |
| `handle` | string | - | CSS selector for drag handle |
| `disabled` | boolean | false | Disable dragging |
| `animation` | number | 150 | Animation duration in ms |

### `<SortableItem>`

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `id` | string/number | required | Unique item identifier |

## Rendered HTML

```parsley
<SortableList endpoint="/api/reorder" group="tasks" listId="todo">
    <SortableItem id={1}><div>Task 1</div></SortableItem>
    <SortableItem id={2}><div>Task 2</div></SortableItem>
</SortableList>
```

Renders to:

```html
<ul class="sortable-list"
    data-sortable
    data-sortable-endpoint="/api/reorder"
    data-sortable-group="tasks"
    data-sortable-list-id="todo"
    data-sortable-animation="150">
    <li class="sortable-item" data-sortable-id="1"><div>Task 1</div></li>
    <li class="sortable-item" data-sortable-id="2"><div>Task 2</div></li>
</ul>
```

## CSS Classes (Automatic)

SortableJS adds these classes automatically during drag operations:

| Class | Applied To | When |
|-------|-----------|------|
| `.sortable-ghost` | Placeholder | Where item will drop |
| `.sortable-chosen` | Dragged item | Item being dragged |
| `.sortable-drag` | Clone | The moving clone (if cloning) |
| `.sortable-fallback` | Item | During fallback (no native drag) |

### Recommended Base Styles

```css
/* Smooth transitions */
.sortable-list {
    list-style: none;
    padding: 0;
    margin: 0;
}

/* Drop placeholder */
.sortable-ghost {
    opacity: 0.4;
    background: #c8ebfb;
}

/* Item being dragged */
.sortable-chosen {
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
}

/* Drag handle cursor */
.drag-handle {
    cursor: grab;
}
.drag-handle:active {
    cursor: grabbing;
}
```

## Backend Protocol

### Same-List Reorder

When items are reordered within the same list:

```http
POST /api/tasks/reorder
Content-Type: application/json

{
    "itemId": "task-123",
    "fromIndex": 2,
    "toIndex": 0,
    "listId": "todo"
}
```

### Cross-List Move

When an item moves between lists:

```http
POST /api/tasks/move
Content-Type: application/json

{
    "itemId": "task-123",
    "fromListId": "todo",
    "toListId": "doing",
    "fromIndex": 2,
    "toIndex": 0
}
```

### Response

```json
{ "success": true }
```

Or on error:

```json
{ "success": false, "error": "Permission denied" }
```

On error, the item reverts to its original position.

## JavaScript Implementation

Add to `basil.js` (~60 lines):

```javascript
// SortableList - Drag and drop lists powered by SortableJS
(function() {
    // Check if SortableJS is available
    if (typeof Sortable === 'undefined') {
        console.warn('SortableJS not loaded. Include it before basil.js for drag-drop support.');
        return;
    }

    function initSortableLists() {
        document.querySelectorAll('[data-sortable]').forEach(list => {
            // Skip if already initialized
            if (list._sortable) return;

            const endpoint = list.dataset.sortableEndpoint;
            const group = list.dataset.sortableGroup;
            const listId = list.dataset.sortableListId;
            const handle = list.dataset.sortableHandle;
            const animation = parseInt(list.dataset.sortableAnimation) || 150;

            list._sortable = Sortable.create(list, {
                group: group || undefined,
                handle: handle || undefined,
                animation: animation,
                ghostClass: 'sortable-ghost',
                chosenClass: 'sortable-chosen',
                dragClass: 'sortable-drag',

                onEnd: function(evt) {
                    if (!endpoint) return;

                    const itemId = evt.item.dataset.sortableId;
                    const fromListId = evt.from.dataset.sortableListId || listId;
                    const toListId = evt.to.dataset.sortableListId || listId;

                    const payload = {
                        itemId: itemId,
                        fromIndex: evt.oldIndex,
                        toIndex: evt.newIndex
                    };

                    // Add list info for cross-list moves
                    if (fromListId !== toListId) {
                        payload.fromListId = fromListId;
                        payload.toListId = toListId;
                    } else {
                        payload.listId = listId || undefined;
                    }

                    fetch(endpoint, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify(payload)
                    })
                    .then(r => r.json())
                    .then(data => {
                        if (!data.success) {
                            // Revert on error
                            if (evt.from !== evt.to) {
                                evt.from.insertBefore(evt.item, evt.from.children[evt.oldIndex]);
                            } else {
                                const ref = list.children[evt.oldIndex];
                                list.insertBefore(evt.item, evt.newIndex > evt.oldIndex ? ref.nextSibling : ref);
                            }
                            console.error('Sortable error:', data.error);
                        }
                    })
                    .catch(err => {
                        console.error('Sortable fetch error:', err);
                    });
                }
            });
        });
    }

    // Initialize on load and after HTMX/Part swaps
    document.addEventListener('DOMContentLoaded', initSortableLists);
    document.addEventListener('htmx:afterSwap', initSortableLists);
    document.addEventListener('part:updated', initSortableLists);
})();
```

## Parsley Component Implementation

### `server/prelude/components/sortable_list.pars`

```parsley
// SortableList - Drag-and-drop sortable container
// Requires SortableJS to be loaded for drag-drop functionality
//
// Usage:
//   <SortableList endpoint="/api/reorder">
//       for (item in items) {
//           <SortableItem id={item.id}>content</SortableItem>
//       }
//   </SortableList>

export SortableList = fn({endpoint, group, listId, handle, disabled, animation, class, contents}) {
    <ul 
        class={"sortable-list" + if (class) { " " + class } else { "" }}
        data-sortable={if (!disabled) { "" } else { null }}
        data-sortable-endpoint={endpoint}
        data-sortable-group={group}
        data-sortable-list-id={listId}
        data-sortable-handle={handle}
        data-sortable-animation={animation ?? 150}
    >
        (contents)
    </ul>
}
```

### `server/prelude/components/sortable_item.pars`

```parsley
// SortableItem - Individual draggable item within a SortableList
//
// Usage:
//   <SortableItem id={item.id}>
//       <div>Item content</div>
//   </SortableItem>

export SortableItem = fn({id, class, contents}) {
    <li 
        class={"sortable-item" + if (class) { " " + class } else { "" }}
        data-sortable-id={id}
    >
        (contents)
    </li>
}
```

## Including SortableJS

SortableJS is ~10KB gzipped. Options:

### Option 1: CDN (Recommended for Getting Started)

```parsley
<Page title="My App">
    <script src="https://cdn.jsdelivr.net/npm/sortablejs@1.15.0/Sortable.min.js"/>
    ...
</Page>
```

### Option 2: Local Copy

Download to `handlers/lib/sortable.min.js`, included automatically via `<Javascript/>`.

### Option 3: Future - Bundled in BasilJS

Could bundle SortableJS into `basil.js` for zero-config experience.

## Complete Example

```parsley
// handlers/kanban.pars

let tasks = [
    {id: 1, title: "Design API", status: "done"},
    {id: 2, title: "Write tests", status: "doing"},
    {id: 3, title: "Deploy", status: "todo"},
    {id: 4, title: "Document", status: "todo"}
]

export default = fn(req) {
    <Page title="Kanban Board">
        <script src="https://cdn.jsdelivr.net/npm/sortablejs@1.15.0/Sortable.min.js"/>
        <style>
            .kanban { display: flex; gap: 1rem; }
            .column { background: #f5f5f5; padding: 1rem; min-width: 200px; border-radius: 8px; }
            .column h3 { margin: 0 0 1rem; text-transform: uppercase; font-size: 0.875rem; color: #666; }
            .sortable-list { min-height: 100px; }
            .card { background: white; padding: 0.75rem; margin-bottom: 0.5rem; border-radius: 4px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
            .sortable-ghost { opacity: 0.4; background: #c8ebfb; }
            .sortable-chosen { box-shadow: 0 4px 12px rgba(0,0,0,0.15); }
        </style>

        <h1>Kanban Board</h1>
        <div class="kanban">
            for (col in ["todo", "doing", "done"]) {
                let columnTasks = tasks.filter(fn(t) { t.status == col })
                <div class="column">
                    <h3>{col}</h3>
                    <SortableList endpoint="/api/tasks/move" group="kanban" listId={col}>
                        for (task in columnTasks) {
                            <SortableItem id={task.id}>
                                <div class="card">{task.title}</div>
                            </SortableItem>
                        }
                    </SortableList>
                </div>
            }
        </div>
    </Page>
}
```

## Backend Handler Example

```parsley
// handlers/api/tasks/move.pars

export default = fn(req) {
    let body = req.body.parseJSON()
    
    // Update database
    let result = db.exec(<SQL params={[body.toListId, body.itemId]}>
        UPDATE tasks SET status = ? WHERE id = ?
    </SQL>)
    
    if (result.error) {
        {success: false, error: result.error}
    } else {
        {success: true}
    }
}
```

## Summary

| Feature | How |
|---------|-----|
| Animation | Built-in (150ms default) |
| Touch support | Built-in via SortableJS |
| Keyboard | Built-in via SortableJS |
| Backend sync | `endpoint` prop → POST on drop |
| Cross-list | `group` prop (same group = shared) |
| Drag handles | `handle` prop (CSS selector) |
| Styling | `.sortable-ghost`, `.sortable-chosen` classes |
| Accessibility | SortableJS handles focus management |

**Dependencies:** SortableJS (~10KB gzipped)

**Parsley code:** ~30 lines (two components)

**JavaScript:** ~60 lines (in basil.js)

---

## Decisions Made

| Decision | Resolution |
|----------|------------|
| **SortableJS bundling** | External dependency for v1. Optional bundling later if adoption warrants. |
| **HTML element type** | `<ul>/<li>` only for v1. No grids/trees. |
| **Error handling** | Silent revert + `sortable:error` event |
| **Optimistic updates** | Keep optimistic (better UX) |
| **Ranking** | Fractional ranking in `@std/sortable`, invisible for common cases |
| **Phasing** | Phase 1 (components) first, Phase 2 (backend helpers) soon after |

---

## Module Design: `@std/sortable`

Everything sortable-related in one cohesive module. Ranking is **invisible** for the common case:

```parsley
import @std/sortable

// Primary API - ranking handled internally
export default = sortable.reorder({table: "tasks", rankColumn: "rank"})
export default = sortable.move({table: "tasks", rankColumn: "rank", listColumn: "status"})

// For programmatic insertion
sortable.insert(db, {table: "tasks", rankColumn: "rank", data: newTask})
sortable.insertAfter(db, {table: "tasks", rankColumn: "rank", afterId: 123, data: newTask})

// Low-level primitives (advanced users only)
sortable.rankBetween(0.25, 0.5)   // Calculate rank between two values
sortable.redistribute(db, "tasks") // Regenerate all ranks
```

This feels natural: "I'm building sortable lists, I import `@std/sortable`."

### Full Example with `@std/sortable`

```parsley
// handlers/api/tasks/reorder.pars
import @std/sortable

export default = sortable.reorder({
    table: "tasks",
    idColumn: "id",
    rankColumn: "rank",
    scope: fn(req, body) { {status: body.listId} }
})
```

```parsley
// handlers/kanban.pars

let tasks = db.query(<SQL>
    SELECT * FROM tasks WHERE status = 'todo' ORDER BY rank
</SQL>)

<SortableList endpoint="/api/tasks/reorder" listId="todo">
    for (task in tasks) {
        <SortableItem id={task.id}>
            <div class="card">{task.title}</div>
        </SortableItem>
    }
</SortableList>
```

Note: `<SortableList>` and `<SortableItem>` are prelude components (auto-available), while `@std/sortable` provides the backend helpers.

---

## Blocking Prerequisites

| Prerequisite | Status | Notes |
|--------------|--------|-------|
| Component system in prelude | ✅ Done | Already have `<Page>`, `<Head>`, etc. |
| basil.js infrastructure | ✅ Done | Already exists for time components |
| SortableJS loading mechanism | ✅ Decided | CDN/manual include for v1 |
| Parts re-initialization | ⚠️ Needs check | Verify `part:updated` event is emitted |

**No blockers for Phase 1.** Phase 2 requires implementing `@std/sortable`.

---

## Server-Side Helpers: `@std/sortable`

### Design Philosophy: Invisible Ranking

For the common case (drag-drop reordering), fractional ranking should be **invisible** to the user. They just use the handler helpers:

```parsley
// This is all most users need - ranking handled internally
export default = sortable.reorder({table: "tasks", rankColumn: "rank"})
```

**When users need direct access to ranking:**
- Inserting a new item programmatically (not via drag-drop)
- Redistributing after precision exhaustion (rare)
- Custom reordering logic

For these cases, we expose primitives - but they're not the primary API.

### `sortable.reorder()` Handler Helper (Primary API)

The main interface - handles everything automatically:

```parsley
// handlers/api/tasks/reorder.pars
import @std/sortable

export default = sortable.reorder({
    table: "tasks",
    idColumn: "id", 
    rankColumn: "rank",
    // Optional: scope reordering to a subset
    scope: fn(req, body) {
        {status: body.listId}  // Only reorder within same status
    }
})
```

Internally:
1. Parses the request body (`itemId`, `fromIndex`, `toIndex`)
2. Queries adjacent items to calculate new rank (fractional)
3. Updates the database
4. Returns `{success: true}` or `{success: false, error: ...}`

### `sortable.move()` Handler Helper

For cross-list moves (like Kanban):

```parsley
// handlers/api/tasks/move.pars
import @std/sortable

export default = sortable.move({
    table: "tasks",
    idColumn: "id",
    rankColumn: "rank",
    listColumn: "status",  // Column that determines which list
})
```

Handles:
- Calculating rank within the new list
- Updating the list column (e.g., `status = 'doing'`)
- Atomic transaction

### `sortable.insert()` Helper

For programmatic insertion (not drag-drop):

```parsley
import @std/sortable

// Insert at end of list
sortable.insert(db, {
    table: "tasks",
    rankColumn: "rank",
    data: {title: "New task", status: "todo"}
})

// Insert after specific item
sortable.insertAfter(db, {
    table: "tasks",
    rankColumn: "rank",
    afterId: 123,
    data: {title: "New task", status: "todo"}
})
```

### Low-Level Ranking Primitives (Advanced)

Exposed for custom logic, but most users won't need these:

```parsley
import @std/sortable

// Calculate rank between two items
let newRank = sortable.rankBetween(0.25, 0.5)  // Returns ~0.375

// Calculate rank before first item
let firstRank = sortable.rankBefore(0.25)  // Returns ~0.125

// Calculate rank after last item  
let lastRank = sortable.rankAfter(0.75)  // Returns ~0.875

// Redistribute all ranks (rare - when precision exhausted)
let items = sortable.redistribute(db, "tasks", {status: "todo"})
// Updates ranks to 0.1, 0.2, 0.3, ... and returns updated items
```

### Future: Migration Helper (Phase 3?)

Add rank column to existing table:

```parsley
import @std/sortable

sortable.migrate(db, "tasks", {
    rankColumn: "rank",
    orderBy: "created_at"  // Initial order based on existing column
})
```

---

## Implementation Phases

### Phase 1: Core Components (MVP)
- [ ] `<SortableList>` component in prelude
- [ ] `<SortableItem>` component in prelude
- [ ] basil.js SortableJS integration (~60 lines)
- [ ] Documentation for CDN include
- [ ] Example: basic sortable list
- [ ] Example: kanban board

**Deliverable:** Working drag-drop with manual backend. Users write their own ranking logic.

### Phase 2: Backend Helpers
- [ ] `@std/sortable` module
  - [ ] `sortable.reorder({...})` - handler helper (primary API)
  - [ ] `sortable.move({...})` - cross-list handler helper
  - [ ] `sortable.insert()` - insert with auto-rank
  - [ ] `sortable.insertAfter()` - insert after specific item
  - [ ] `sortable.rankBetween(a, b)` - low-level primitive
  - [ ] `sortable.redistribute(db, table)` - regenerate ranks

**Deliverable:** One-liner backend setup. Ranking is invisible for common cases.

### Phase 3: Polish
- [ ] Event system (`sortable:start`, `sortable:end`, `sortable:error`)
- [ ] Toast integration for errors (if Toast component exists)
- [ ] Consider bundling SortableJS as optional
- [ ] Accessibility audit
- [ ] Migration helper (`sortable.migrate()`)

**Deliverable:** Production-ready, polished experience.

---

## Value Proposition

> "Reorderable/sortable lists are something that many small websites need but always do terribly just to avoid having to make something like this. Providing an elegant, batteries-included, 'plug & play' solution would be a great selling point."

**For users:** Drag-drop lists in ~10 lines of Parsley + 3 lines for backend.

**Competitive advantage:** Most frameworks require significant JS knowledge or third-party packages with complex integration. Basil's approach: import one module, use two components, done.
