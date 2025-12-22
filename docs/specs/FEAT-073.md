---
id: FEAT-073
title: "SortableList Component with Fractional Ranking"
status: draft
priority: high
created: 2025-12-21
author: "@sambeau"
---

# FEAT-073: SortableList Component with Fractional Ranking

## Summary
Implement drag-and-drop sortable lists with batteries-included backend support. Provides `<SortableList>` and `<SortableItem>` components powered by SortableJS, plus `@std/sortable` module with fractional ranking helpers for O(1) reordering.

## User Story
As a developer, I want to add drag-and-drop reorderable lists to my Basil app with minimal code so that I can provide intuitive interfaces for todo lists, kanban boards, and prioritized items without dealing with complex ranking algorithms or JavaScript integration.

## Acceptance Criteria

### Phase 1: Core Components (MVP)
- [ ] `<SortableList>` component renders `<ul>` with data attributes
- [ ] `<SortableItem>` component renders `<li>` with unique identifier
- [ ] basil.js initializes SortableJS on `[data-sortable]` elements
- [ ] Drag events POST JSON to backend endpoint
- [ ] Failed backend calls revert the UI change
- [ ] Cross-list dragging works with `group` prop
- [ ] Drag handles work with `handle` prop (CSS selector)
- [ ] Components work with Parts (re-initialize after swaps)
- [ ] Documentation includes CDN loading instructions
- [ ] Example: basic sortable list works end-to-end
- [ ] Example: kanban board works end-to-end

### Phase 2: Backend Helpers
- [ ] `@std/sortable` module is importable
- [ ] `sortable.reorder()` returns handler function
- [ ] `sortable.move()` returns handler function for cross-list moves
- [ ] `sortable.insert()` inserts item at end with calculated rank
- [ ] `sortable.insertAfter()` inserts item after specific ID
- [ ] `sortable.rankBetween()` calculates midpoint rank
- [ ] `sortable.redistribute()` regenerates all ranks
- [ ] Fractional ranking handles precision exhaustion gracefully
- [ ] All backend helpers work with SQLite database
- [ ] Documentation includes backend setup examples

### Phase 3: Polish (Future)
- [ ] Event system emits `sortable:start`, `sortable:end`, `sortable:error`
- [ ] Error Toast integration (if Toast component exists)
- [ ] Optional SortableJS bundling mechanism
- [ ] Accessibility audit passed
- [ ] Migration helper `sortable.migrate()` implemented

## Design Decisions

- **SortableJS Bundling**: External CDN dependency for Phase 1. Optional bundling in Phase 3.
  - Rationale: Keeps basil.js small; users control version; can add bundling later if adoption is high.

- **HTML Element Type**: `<ul>/<li>` only. No `<div>`, `<table>`, or custom elements in Phase 1.
  - Rationale: Lists cover 90% of use cases; semantic HTML; simpler implementation.

- **Ranking Strategy**: Fractional ranking (floats) over integer indices.
  - Rationale: O(1) reordering vs O(n); no need to renumber all rows on reorder.

- **Ranking Visibility**: Invisible by default. Low-level primitives available for advanced cases.
  - Rationale: Most users just want drag-drop to work; power users can optimize insertion.

- **Error Handling**: Optimistic updates with silent revert + console.error on failure.
  - Rationale: Better UX than pessimistic; visual feedback feels instant; errors are rare.

- **Module Name**: `@std/sortable` (not `@std/position` or `@std/ranking`).
  - Rationale: Cohesive API; importing sortable features from a sortable module feels natural.

- **Column Name**: `rankColumn: "rank"` (not `position`, `order`, `sort_order`).
  - Rationale: "Rank" better conveys a sortable float value; avoids confusion with array indices.

---

## Technical Specification

### Component API

#### `<SortableList>`

**Location**: `server/prelude/components/sortable_list.pars`

**Props**:
| Prop | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `endpoint` | string | Yes | - | URL to POST reorder data |
| `group` | string | No | - | Group name for cross-list dragging |
| `listId` | string | No | - | Identifier for this list (sent to backend) |
| `handle` | string | No | - | CSS selector for drag handle (e.g., `".drag-handle"`) |
| `disabled` | boolean | No | false | Disable dragging entirely |
| `animation` | number | No | 150 | Animation duration in milliseconds |
| `class` | string | No | - | Additional CSS classes for `<ul>` |
| `contents` | string | Auto | - | Child elements (auto-passed by tag pair) |

**Rendered HTML**:
```html
<ul class="sortable-list [custom-class]"
    data-sortable
    data-sortable-endpoint="[endpoint]"
    data-sortable-group="[group]"
    data-sortable-list-id="[listId]"
    data-sortable-handle="[handle]"
    data-sortable-animation="[animation]">
    [contents]
</ul>
```

**Implementation** (idiomatic Parsley):
```parsley
// server/prelude/components/sortable_list.pars

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

**Validation Rules**:
- `endpoint` must be non-empty string (MUST)
- `animation` must be non-negative integer (SHOULD)
- If `disabled` is true, omit `data-sortable` attribute entirely

---

#### `<SortableItem>`

**Location**: `server/prelude/components/sortable_item.pars`

**Props**:
| Prop | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `id` | string/number | Yes | - | Unique item identifier (sent to backend) |
| `class` | string | No | - | Additional CSS classes for `<li>` |
| `contents` | string | Auto | - | Child elements (auto-passed by tag pair) |

**Rendered HTML**:
```html
<li class="sortable-item [custom-class]"
    data-sortable-id="[id]">
    [contents]
</li>
```

**Implementation** (idiomatic Parsley):
```parsley
// server/prelude/components/sortable_item.pars

export SortableItem = fn({id, class, contents}) {
    <li 
        class={"sortable-item" + if (class) { " " + class } else { "" }}
        data-sortable-id={id}
    >
        (contents)
    </li>
}
```

**Validation Rules**:
- `id` must be non-empty (MUST)
- `id` must be unique within the parent `<SortableList>` (MUST)
- `id` values should be stable across re-renders (SHOULD)

---

### JavaScript Integration

**Location**: `server/basil.js` (append to existing file)

**Behavior**:
1. On `DOMContentLoaded`, `htmx:afterSwap`, and `part:updated` events, scan for `[data-sortable]` elements
2. Skip elements already initialized (check `list._sortable` property)
3. Initialize SortableJS with:
   - `group` from `data-sortable-group` (if present)
   - `handle` from `data-sortable-handle` (if present)
   - `animation` from `data-sortable-animation` (default 150)
   - `ghostClass: 'sortable-ghost'`
   - `chosenClass: 'sortable-chosen'`
   - `dragClass: 'sortable-drag'`
4. On `onEnd` event:
   - Extract `itemId` from dragged element's `data-sortable-id`
   - Extract `fromListId` and `toListId` from list containers
   - Build JSON payload (see Backend Protocol below)
   - POST to endpoint with `Content-Type: application/json`
   - On `{success: false}` or fetch error, revert the DOM change
   - Log errors to console

**Implementation** (JavaScript):
```javascript
// server/basil.js (append to existing file)

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

**Validation Rules**:
- MUST NOT initialize an element twice (check `_sortable` property)
- MUST revert DOM on backend failure
- MUST log errors to console
- SHOULD emit `part:updated` event after Parts swaps (verify this exists)

---

### Backend Protocol

#### Same-List Reorder

**Request**:
```http
POST [endpoint]
Content-Type: application/json

{
    "itemId": "task-123",
    "fromIndex": 2,
    "toIndex": 0,
    "listId": "todo"
}
```

**Field Definitions**:
- `itemId` (string): Unique identifier from `data-sortable-id`
- `fromIndex` (number): Zero-based index before drag
- `toIndex` (number): Zero-based index after drag
- `listId` (string, optional): Value from `data-sortable-list-id`

**Response** (success):
```json
{"success": true}
```

**Response** (failure):
```json
{"success": false, "error": "Permission denied"}
```

---

#### Cross-List Move

**Request**:
```http
POST [endpoint]
Content-Type: application/json

{
    "itemId": "task-123",
    "fromListId": "todo",
    "toListId": "doing",
    "fromIndex": 2,
    "toIndex": 0
}
```

**Field Definitions**:
- `itemId` (string): Unique identifier from `data-sortable-id`
- `fromListId` (string): Source list's `data-sortable-list-id`
- `toListId` (string): Destination list's `data-sortable-list-id`
- `fromIndex` (number): Zero-based index in source list
- `toIndex` (number): Zero-based index in destination list

**Response**: Same as same-list reorder

---

### `@std/sortable` Module API

**Location**: `server/stdlib/sortable.pars` (new file)

**Module Structure**:
```parsley
// @std/sortable - Sortable list backend helpers with fractional ranking

// Primary API - Handler Generators
export reorder = fn(config) { ... }
export move = fn(config) { ... }

// Insertion Helpers
export insert = fn(db, config) { ... }
export insertAfter = fn(db, config) { ... }

// Low-Level Primitives
export rankBetween = fn(rankA, rankB) { ... }
export rankBefore = fn(rank) { ... }
export rankAfter = fn(rank) { ... }
export redistribute = fn(db, table, where) { ... }
```

---

#### `sortable.reorder(config)`

**Purpose**: Generate a handler function for same-list reordering

**Parameters** (config dictionary):
| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `table` | string | Yes | Database table name |
| `idColumn` | string | No | Column name for item ID (default: `"id"`) |
| `rankColumn` | string | No | Column name for rank (default: `"rank"`) |
| `scope` | function | No | Function returning WHERE clause dict |

**Returns**: Handler function `fn(req) { ... }` that returns success/error dict

**Behavior**:
1. Parse request body JSON
2. Extract `itemId`, `toIndex`, `listId` (if present)
3. If `scope` provided, call `scope(req, body)` to get WHERE clause
4. Query items in scope, ordered by rank
5. Find items before and after `toIndex`
6. Calculate new rank using `rankBetween()`, `rankBefore()`, or `rankAfter()`
7. Update item's rank in database
8. Return `{success: true}` or `{success: false, error: "..."}`

**Example** (idiomatic Parsley):
```parsley
// handlers/api/tasks/reorder.pars
import @std/sortable

export default = sortable.reorder({
    table: "tasks",
    idColumn: "id",
    rankColumn: "rank",
    scope: fn(req, body) {
        {status: body.listId}  // Only reorder within same status
    }
})
```

**Test Criteria**:
- ✓ Reorders first item to last position correctly
- ✓ Reorders last item to first position correctly
- ✓ Reorders middle item up/down correctly
- ✓ No-op when dragged to same position (returns success)
- ✓ Returns error if item not found
- ✓ Returns error if database update fails
- ✓ Scope function correctly filters items
- ✓ Works with integer and string IDs

---

#### `sortable.move(config)`

**Purpose**: Generate a handler function for cross-list moves (e.g., Kanban)

**Parameters** (config dictionary):
| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `table` | string | Yes | Database table name |
| `idColumn` | string | No | Column name for item ID (default: `"id"`) |
| `rankColumn` | string | No | Column name for rank (default: `"rank"`) |
| `listColumn` | string | Yes | Column name that determines list membership |

**Returns**: Handler function `fn(req) { ... }` that returns success/error dict

**Behavior**:
1. Parse request body JSON
2. Extract `itemId`, `fromListId`, `toListId`, `toIndex`
3. Begin database transaction
4. Query items in destination list (WHERE `listColumn = toListId`), ordered by rank
5. Calculate new rank for position `toIndex`
6. Update item: set `listColumn = toListId` and `rankColumn = newRank`
7. Commit transaction
8. Return `{success: true}` or `{success: false, error: "..."}`

**Example** (idiomatic Parsley):
```parsley
// handlers/api/tasks/move.pars
import @std/sortable

export default = sortable.move({
    table: "tasks",
    idColumn: "id",
    rankColumn: "rank",
    listColumn: "status"
})
```

**Test Criteria**:
- ✓ Moves item from list A to list B correctly
- ✓ Calculates rank in destination list context
- ✓ Updates both list column and rank column
- ✓ Transaction rolls back on error
- ✓ Works with empty destination list
- ✓ Returns error if item not found

---

#### `sortable.insert(db, config)`

**Purpose**: Insert a new item at the end of a list with calculated rank

**Parameters**:
| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `db` | database | Yes | Database connection |
| `table` | string | Yes | Database table name |
| `rankColumn` | string | No | Column name for rank (default: `"rank"`) |
| `data` | dict | Yes | Column/value pairs to insert |
| `where` | dict | No | WHERE clause to scope the list |

**Returns**: Inserted item dictionary (with generated ID and rank)

**Behavior**:
1. Query max rank in list (optionally filtered by WHERE)
2. Calculate new rank using `rankAfter(maxRank)`
3. Insert row with `data` plus calculated rank
4. Return inserted item with all columns

**Example** (idiomatic Parsley):
```parsley
import @std/sortable

let newTask = sortable.insert(db, {
    table: "tasks",
    rankColumn: "rank",
    data: {title: "New task", status: "todo"},
    where: {status: "todo"}
})

log("Inserted task with rank: " + newTask.rank)
```

**Test Criteria**:
- ✓ Inserts at end of empty list (rank = 0.5)
- ✓ Inserts after last item correctly
- ✓ Returns inserted item with ID and rank
- ✓ WHERE clause correctly scopes the list

---

#### `sortable.insertAfter(db, config)`

**Purpose**: Insert a new item after a specific item

**Parameters**:
| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `db` | database | Yes | Database connection |
| `table` | string | Yes | Database table name |
| `idColumn` | string | No | Column name for item ID (default: `"id"`) |
| `rankColumn` | string | No | Column name for rank (default: `"rank"`) |
| `afterId` | string/number | Yes | ID of item to insert after |
| `data` | dict | Yes | Column/value pairs to insert |

**Returns**: Inserted item dictionary (with generated ID and rank)

**Behavior**:
1. Query item with `afterId` to get its rank
2. Query next item after that rank
3. Calculate rank between them using `rankBetween()`
4. Insert row with `data` plus calculated rank
5. Return inserted item

**Example** (idiomatic Parsley):
```parsley
import @std/sortable

let newTask = sortable.insertAfter(db, {
    table: "tasks",
    rankColumn: "rank",
    afterId: 123,
    data: {title: "New task", status: "todo"}
})
```

**Test Criteria**:
- ✓ Inserts between two items correctly
- ✓ Inserts after last item (acts like `insert()`)
- ✓ Returns error if `afterId` not found
- ✓ Handles rank precision exhaustion (triggers redistribute)

---

#### `sortable.rankBetween(rankA, rankB)`

**Purpose**: Calculate midpoint rank between two values

**Parameters**:
- `rankA` (float): Lower rank value
- `rankB` (float): Higher rank value

**Returns**: Float representing midpoint

**Behavior**:
- Return `(rankA + rankB) / 2`
- If precision exhausted (difference < 1e-10), trigger redistribute

**Example** (idiomatic Parsley):
```parsley
import @std/sortable

let rank = sortable.rankBetween(0.25, 0.75)  // Returns 0.5
```

**Test Criteria**:
- ✓ Returns exact midpoint for normal ranges
- ✓ Handles very small differences
- ✓ Detects precision exhaustion (difference < threshold)

---

#### `sortable.rankBefore(rank)`

**Purpose**: Calculate rank before the first item

**Parameters**:
- `rank` (float): Rank of current first item

**Returns**: Float representing rank before first

**Behavior**:
- Return `rank / 2`
- If result too small (< 1e-10), trigger redistribute

**Example** (idiomatic Parsley):
```parsley
import @std/sortable

let rank = sortable.rankBefore(0.5)  // Returns 0.25
```

---

#### `sortable.rankAfter(rank)`

**Purpose**: Calculate rank after the last item

**Parameters**:
- `rank` (float): Rank of current last item

**Returns**: Float representing rank after last

**Behavior**:
- Return `rank + ((1.0 - rank) / 2)`
- Always results in value between `rank` and 1.0

**Example** (idiomatic Parsley):
```parsley
import @std/sortable

let rank = sortable.rankAfter(0.5)  // Returns 0.75
```

---

#### `sortable.redistribute(db, table, where)`

**Purpose**: Regenerate all ranks in a list with even spacing

**Parameters**:
- `db` (database): Database connection
- `table` (string): Table name
- `where` (dict, optional): WHERE clause to scope redistribution

**Returns**: Array of updated items with new ranks

**Behavior**:
1. Query all items in scope, ordered by current rank
2. Assign new ranks: 0.1, 0.2, 0.3, ..., 0.9, 0.91, 0.92, ...
3. Update all rows in a transaction
4. Return updated items

**Example** (idiomatic Parsley):
```parsley
import @std/sortable

let items = sortable.redistribute(db, "tasks", {status: "todo"})
log("Redistributed " + items.length() + " items")
```

**Test Criteria**:
- ✓ Maintains relative order
- ✓ Assigns evenly spaced ranks
- ✓ Handles 1000+ items without precision loss
- ✓ WHERE clause correctly scopes redistribution

---

## CSS Integration

**SortableJS Automatic Classes**:
| Class | Applied To | Description |
|-------|-----------|-------------|
| `.sortable-ghost` | Placeholder `<li>` | Where item will drop |
| `.sortable-chosen` | Dragged `<li>` | Item being dragged |
| `.sortable-drag` | Clone `<li>` | The moving visual clone |

**Recommended Base Styles** (optional, provided in documentation):
```css
.sortable-list {
    list-style: none;
    padding: 0;
    margin: 0;
}

.sortable-ghost {
    opacity: 0.4;
    background: #c8ebfb;
}

.sortable-chosen {
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
}

.drag-handle {
    cursor: grab;
}

.drag-handle:active {
    cursor: grabbing;
}
```

---

## Examples (Acceptance Tests)

### Example 1: Basic Sortable List

**File**: `examples/sortable/basic/handlers/index.pars`

```parsley
let items = [
    {id: 1, title: "Buy groceries", rank: 0.1},
    {id: 2, title: "Walk dog", rank: 0.2},
    {id: 3, title: "Read book", rank: 0.3}
]

export default = fn(req) {
    <Page title="Basic Sortable List">
        <script src="https://cdn.jsdelivr.net/npm/sortablejs@1.15.0/Sortable.min.js"/>
        <style>
            .sortable-ghost { opacity: 0.4; background: #c8ebfb; }
            .sortable-chosen { box-shadow: 0 4px 12px rgba(0,0,0,0.15); }
        </style>

        <h1>My Todo List</h1>
        <SortableList endpoint="/api/reorder">
            for (item in items) {
                <SortableItem id={item.id}>
                    <div>{item.title}</div>
                </SortableItem>
            }
        </SortableList>
    </Page>
}
```

**Backend**: `examples/sortable/basic/handlers/api/reorder.pars`

```parsley
import @std/sortable

export default = sortable.reorder({
    table: "items",
    idColumn: "id",
    rankColumn: "rank"
})
```

**Test Criteria**:
- ✓ Page loads with 3 items
- ✓ Items can be dragged and dropped
- ✓ Reordering sends POST to `/api/reorder`
- ✓ Database rank is updated correctly
- ✓ Page refresh shows new order

---

### Example 2: Kanban Board

**File**: `examples/sortable/kanban/handlers/index.pars`

```parsley
let tasks = db.query(<SQL>
    SELECT * FROM tasks ORDER BY rank
</SQL>)

export default = fn(req) {
    <Page title="Kanban Board">
        <script src="https://cdn.jsdelivr.net/npm/sortablejs@1.15.0/Sortable.min.js"/>
        <style>
            .kanban { display: flex; gap: 1rem; }
            .column { background: #f5f5f5; padding: 1rem; min-width: 200px; }
            .sortable-list { min-height: 100px; }
            .card { background: white; padding: 0.75rem; margin-bottom: 0.5rem; }
            .sortable-ghost { opacity: 0.4; }
        </style>

        <h1>Kanban Board</h1>
        <div class="kanban">
            for (col in ["todo", "doing", "done"]) {
                let columnTasks = tasks.filter(fn(t) { t.status == col })
                <div class="column">
                    <h3>{col}</h3>
                    <SortableList 
                        endpoint="/api/tasks/move" 
                        group="kanban" 
                        listId={col}
                    >
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

**Backend**: `examples/sortable/kanban/handlers/api/tasks/move.pars`

```parsley
import @std/sortable

export default = sortable.move({
    table: "tasks",
    idColumn: "id",
    rankColumn: "rank",
    listColumn: "status"
})
```

**Test Criteria**:
- ✓ Three columns render (todo, doing, done)
- ✓ Tasks can be dragged within same column
- ✓ Tasks can be dragged between columns
- ✓ Database status column updates on cross-column move
- ✓ Database rank updates correctly in destination column

---

### Example 3: Drag Handles

**File**: `examples/sortable/handles/handlers/index.pars`

```parsley
let items = db.query(<SQL>SELECT * FROM items ORDER BY rank</SQL>)

export default = fn(req) {
    <Page title="Drag Handles">
        <script src="https://cdn.jsdelivr.net/npm/sortablejs@1.15.0/Sortable.min.js"/>
        <style>
            .item { display: flex; gap: 0.5rem; align-items: center; padding: 0.5rem; }
            .drag-handle { cursor: grab; font-size: 1.5em; color: #999; }
            .drag-handle:active { cursor: grabbing; }
        </style>

        <h1>Items with Drag Handles</h1>
        <SortableList endpoint="/api/reorder" handle=".drag-handle">
            for (item in items) {
                <SortableItem id={item.id}>
                    <div class="item">
                        <span class="drag-handle">⋮⋮</span>
                        <span>{item.name}</span>
                    </div>
                </SortableItem>
            }
        </SortableList>
    </Page>
}
```

**Test Criteria**:
- ✓ Items cannot be dragged by clicking content
- ✓ Items CAN be dragged by clicking the handle (⋮⋮)
- ✓ Handle cursor changes to `grab`/`grabbing`

---

## Validation & Testing

### Unit Tests (Go)

**File**: `pkg/parsley/tests/stdlib_sortable_test.go`

Test functions for `@std/sortable` module:
- `TestSortableRankBetween()` - midpoint calculation
- `TestSortableRankBefore()` - before first
- `TestSortableRankAfter()` - after last
- `TestSortableRedistribute()` - regenerate ranks
- `TestSortableInsert()` - insert at end
- `TestSortableInsertAfter()` - insert between items
- `TestSortableReorderHandler()` - handler function
- `TestSortableMoveHandler()` - cross-list handler

### Component Tests (Go)

**File**: `pkg/parsley/tests/components_sortable_test.go`

Test rendering:
- `TestSortableListRendering()` - HTML output
- `TestSortableItemRendering()` - HTML output
- `TestSortableListDataAttributes()` - data-* correctness
- `TestSortableListDisabled()` - disabled state

### Integration Tests (Parsley)

**File**: `examples/sortable/test.pars`

End-to-end tests:
- Create database with test data
- Start server
- Simulate reorder POST request
- Verify database updated
- Verify response format

### Manual Testing Checklist

- [ ] Drag item to new position → database updates
- [ ] Drag item to same position → no database call
- [ ] Backend returns error → item reverts
- [ ] Drag between lists (kanban) → both columns update
- [ ] Drag handle works → only handle triggers drag
- [ ] Animation plays → smooth transitions
- [ ] Touch devices work → mobile drag works
- [ ] Keyboard accessibility → can reorder with keyboard (SortableJS feature)
- [ ] Works with Parts → re-initializes after Part swap
- [ ] Large lists (100+ items) → performance acceptable

---

## Documentation Requirements

### User Guide

**File**: `docs/guide/sortable-lists.md`

Sections:
1. Quick Start - basic example
2. Kanban Board - cross-list example
3. Drag Handles - handle selector
4. Backend Setup - using `@std/sortable`
5. Styling - CSS classes
6. Troubleshooting - common issues

### API Reference

**File**: `docs/manual/components/sortable.md`

Document `<SortableList>` and `<SortableItem>` props, examples

**File**: `docs/manual/stdlib/sortable.md`

Document `@std/sortable` module functions with signatures and examples

### Migration Guide

**File**: `docs/guide/adding-sortable-to-existing-project.md`

Steps:
1. Add rank column to database
2. Populate initial ranks
3. Update queries to `ORDER BY rank`
4. Add `<SortableList>` components
5. Add backend handler

---

## Dependencies

### External

- **SortableJS** v1.15.0 or later
  - Source: https://cdn.jsdelivr.net/npm/sortablejs@1.15.0/Sortable.min.js
  - Size: ~10KB gzipped
  - License: MIT

### Internal

- Depends on: Component system in prelude (already exists)
- Depends on: basil.js infrastructure (already exists)
- Depends on: Database module `db` (already exists)
- Blocks: None

---

## Edge Cases & Constraints

### Edge Case: Precision Exhaustion

**Scenario**: After thousands of reorders between the same two items, float precision exhausted

**Detection**: When `rankBetween(a, b)` results in `b - a < 1e-10`

**Handling**: Automatically call `redistribute()` to regenerate all ranks

**Test**: Insert 10,000 items between rank 0.5 and 0.5000001

---

### Edge Case: Empty List

**Scenario**: First item added to empty sortable list

**Handling**: 
- `insert()` assigns rank `0.5`
- `rankBefore(null)` returns `0.5`
- `rankAfter(null)` returns `0.5`

**Test**: Insert into empty database table

---

### Edge Case: Concurrent Reorders

**Scenario**: Two users reorder the same list simultaneously

**Handling**: Last write wins. No optimistic locking in Phase 1.

**Future**: Phase 3 could add version column for optimistic locking

---

### Constraint: Single Column Rank

**Limitation**: Cannot have multiple independent sort orders for the same table

**Workaround**: Use separate tables or scoped queries

---

### Constraint: No Nested Lists

**Limitation**: Phase 1 does not support drag-drop between nested lists or tree structures

**Scope**: Only flat lists (`<ul>/<li>`)

---

## Implementation Notes

*Filled during/after implementation*

---

## Related

- Design: `docs/design/sortable-list-component.md` - detailed design exploration
- Design: `docs/design/sortable-list-syntax.md` - syntax alternatives explored
- Design: `docs/design/sortable-lists.md` - early prototypes (native and SortableJS)
- Plan: `docs/plans/FEAT-073-plan.md` (to be created from this spec)
