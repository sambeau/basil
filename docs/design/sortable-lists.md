# Sortable Lists

Design document for drag-and-drop reorderable lists with backend synchronization.

## Overview

Sortable lists allow users to reorder items via drag-and-drop with automatic persistence to the backend. This is a common pattern for:
- Kanban boards (task management)
- Priority lists
- Playlist ordering
- Image galleries
- Navigation menu editors
- Shopping cart item ordering

## Design Goals

1. **Minimal API surface** - Few attributes, obvious behavior
2. **Progressive enhancement** - Works as static list without JS
3. **Composable** - Multiple lists can share items via groups
4. **Stylable** - All states exposed via CSS classes
5. **Backend-aware** - Automatic persistence, error handling
6. **Accessible** - Keyboard and screen reader support
7. **Lightweight** - No external dependencies, small footprint

## Proposed API

### Basic Usage

```parsley
let {SortableList} = import @std/html

<SortableList 
    items={tasks}
    endpoint="/api/tasks/reorder"
    itemKey="id"
>
    {task => <div class="task">{task.title}</div>}
</SortableList>
```

### Cross-List Dragging (Kanban)

```parsley
<div class="kanban">
    <SortableList 
        items={todoTasks}
        endpoint="/api/tasks/reorder"
        itemKey="id"
        group="kanban"
        listId="todo"
    >
        {task => <TaskCard task={task}/>}
    </SortableList>
    
    <SortableList 
        items={doneTasks}
        endpoint="/api/tasks/reorder"
        itemKey="id"
        group="kanban"
        listId="done"
    >
        {task => <TaskCard task={task}/>}
    </SortableList>
</div>
```

### Props

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `items` | array | Yes | Array of items to render |
| `itemKey` | string | Yes | Property name for unique item ID |
| `endpoint` | string | No | URL to POST reorder events |
| `group` | string | No | Group name for cross-list dragging |
| `listId` | string | No | Identifier for this list (for cross-list payloads) |
| `handle` | string | No | CSS selector for drag handle element |
| `disabled` | boolean | No | Disable drag-and-drop |
| `animation` | number | No | Animation duration in ms (default: 150) |

### Render Function

The child must be a function that receives each item:

```parsley
<SortableList items={items} itemKey="id">
    {item => 
        <li class="my-item">
            <span class="handle">"⋮⋮"</span>
            item.name
        </li>
    }
</SortableList>
```

---

## Rendered HTML

### Input (Parsley)

```parsley
<SortableList 
    items={[{id: 1, name: "Task A"}, {id: 2, name: "Task B"}]}
    endpoint="/api/reorder"
    itemKey="id"
    group="tasks"
    listId="todo"
>
    {item => <div>{item.name}</div>}
</SortableList>
```

### Output (HTML)

```html
<ul class="sortable-list" 
    data-sortable
    data-endpoint="/api/reorder"
    data-group="tasks"
    data-list-id="todo">
    <li class="sortable-item" data-id="1" draggable="true">
        <div>Task A</div>
    </li>
    <li class="sortable-item" data-id="2" draggable="true">
        <div>Task B</div>
    </li>
</ul>
```

---

## CSS Classes for Styling

All drag states are exposed via CSS classes:

| Class | Applied To | When |
|-------|------------|------|
| `.sortable-list` | Container | Always |
| `.sortable-item` | Items | Always |
| `.is-dragging` | Item | Item is being dragged |
| `.is-ghost` | Placeholder | Shows insertion point |
| `.is-drop-target` | List | List can accept dragged item (same group) |
| `.is-drop-target-active` | List | Mouse is hovering over list |
| `.is-disabled` | List/Item | Dragging is disabled |

### Example Styling

```css
/* Base styles */
.sortable-item {
    padding: 1rem;
    background: white;
    border: 1px solid #ddd;
    cursor: grab;
    transition: transform 150ms, box-shadow 150ms;
}

/* Being dragged */
.sortable-item.is-dragging {
    opacity: 0.5;
    cursor: grabbing;
    transform: rotate(2deg);
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
}

/* Insertion point placeholder */
.sortable-item.is-ghost {
    background: #f0f7ff;
    border: 2px dashed #4a90d9;
    opacity: 1;
}
.sortable-item.is-ghost > * {
    visibility: hidden;
}

/* Valid drop zone */
.sortable-list.is-drop-target {
    outline: 2px solid #4a90d9;
    outline-offset: 4px;
}

/* Active drop zone (mouse over) */
.sortable-list.is-drop-target-active {
    background: #f0f7ff;
}
```

---

## Backend Integration

### Payload Structure

When items are reordered, the following JSON is POSTed to `endpoint`:

```json
{
    "ids": ["3", "1", "2"],
    "movedId": "3",
    "fromIndex": 2,
    "toIndex": 0,
    "listId": "todo",
    "fromList": "backlog",
    "toList": "todo",
    "group": "kanban"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `ids` | string[] | Item IDs in new order |
| `movedId` | string | ID of the item that was moved |
| `fromIndex` | number | Original position |
| `toIndex` | number | New position |
| `listId` | string | Current list identifier |
| `fromList` | string\|null | Source list (if cross-list move) |
| `toList` | string\|null | Target list (if cross-list move) |
| `group` | string\|null | Group name |

### Backend Handler Example (Parsley)

```parsley
// api/tasks/reorder.pars
let body = basil.http.request.json()

// Validate
if (!body.ids || !body.movedId) {
    return {error: "Missing required fields"}.withStatus(400)
}

// Update positions
for (i, id) in body.ids.entries() {
    db.exec(
        "UPDATE tasks SET position = ?, list = ? WHERE id = ?",
        i,
        body.toList ?? body.listId,
        id
    )
}

{success: true, ids: body.ids}
```

### Error Handling

On backend error, the UI should rollback:

```javascript
async notifyBackend(list, payload) {
    try {
        const response = await fetch(payload.endpoint, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        
        // Emit success event
        list.dispatchEvent(new CustomEvent('sortable:success', {
            bubbles: true,
            detail: payload
        }));
        
    } catch (error) {
        // Rollback to original order
        this.rollback(list, payload.originalOrder);
        
        // Emit error event
        list.dispatchEvent(new CustomEvent('sortable:error', {
            bubbles: true,
            detail: { payload, error }
        }));
    }
}
```

---

## JavaScript Implementation

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    SortableList Module                   │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  groups: Map<groupName, Set<listElement>>               │
│  dragging: Element | null                               │
│  sourceList: Element | null                             │
│  originalOrder: string[] | null                         │
│                                                         │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  init()           - Find all [data-sortable], init each │
│  initList(list)   - Set up event listeners on list      │
│  initItem(item)   - Make item draggable                 │
│                                                         │
│  onDragStart()    - Store refs, add .is-dragging        │
│  onDragOver()     - Show ghost at insertion point       │
│  onDragLeave()    - Remove ghost when leaving           │
│  onDrop()         - Move item, notify backend           │
│  onDragEnd()      - Cleanup classes                     │
│                                                         │
│  notifyBackend()  - POST to endpoint                    │
│  rollback()       - Restore original order on error     │
│                                                         │
│  getDragAfterElement() - Find insertion point from Y    │
│  createGhost()         - Create placeholder element     │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Core Implementation (~150 lines)

```javascript
const SortableList = {
    groups: new Map(),
    dragging: null,
    sourceList: null,
    originalOrder: null,
    
    init() {
        document.querySelectorAll('[data-sortable]').forEach(list => {
            this.initList(list);
        });
    },
    
    initList(list) {
        const group = list.dataset.group;
        
        // Register in group for cross-list dragging
        if (group) {
            if (!this.groups.has(group)) {
                this.groups.set(group, new Set());
            }
            this.groups.get(group).add(list);
        }
        
        // Initialize items
        list.querySelectorAll('.sortable-item').forEach(item => {
            item.draggable = true;
            item.addEventListener('dragstart', e => this.onDragStart(e, item));
            item.addEventListener('dragend', e => this.onDragEnd(e, item));
        });
        
        // List drop zone events
        list.addEventListener('dragover', e => this.onDragOver(e, list));
        list.addEventListener('dragleave', e => this.onDragLeave(e, list));
        list.addEventListener('drop', e => this.onDrop(e, list));
    },
    
    onDragStart(e, item) {
        this.dragging = item;
        this.sourceList = item.closest('[data-sortable]');
        this.originalOrder = this.getOrder(this.sourceList);
        
        item.classList.add('is-dragging');
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', item.dataset.id);
        
        // Mark valid drop targets
        const group = this.sourceList.dataset.group;
        if (group && this.groups.has(group)) {
            this.groups.get(group).forEach(list => {
                list.classList.add('is-drop-target');
            });
        } else {
            this.sourceList.classList.add('is-drop-target');
        }
    },
    
    onDragEnd(e, item) {
        item.classList.remove('is-dragging');
        document.querySelectorAll('.is-drop-target, .is-drop-target-active')
            .forEach(el => el.classList.remove('is-drop-target', 'is-drop-target-active'));
        document.querySelectorAll('.is-ghost').forEach(el => el.remove());
        
        this.dragging = null;
        this.sourceList = null;
        this.originalOrder = null;
    },
    
    onDragOver(e, list) {
        if (!this.dragging) return;
        if (!this.canDrop(list)) return;
        
        e.preventDefault();
        list.classList.add('is-drop-target-active');
        
        const afterEl = this.getDragAfterElement(list, e.clientY);
        const ghost = list.querySelector('.is-ghost') || this.createGhost();
        
        if (afterEl) {
            list.insertBefore(ghost, afterEl);
        } else {
            list.appendChild(ghost);
        }
    },
    
    onDragLeave(e, list) {
        if (!list.contains(e.relatedTarget)) {
            list.classList.remove('is-drop-target-active');
            list.querySelector('.is-ghost')?.remove();
        }
    },
    
    onDrop(e, list) {
        e.preventDefault();
        if (!this.dragging) return;
        
        const ghost = list.querySelector('.is-ghost');
        if (!ghost) return;
        
        // Move item to ghost position
        list.insertBefore(this.dragging, ghost);
        ghost.remove();
        
        // Notify backend
        this.notifyBackend(list, {
            ids: this.getOrder(list),
            movedId: this.dragging.dataset.id,
            listId: list.dataset.listId,
            fromList: this.sourceList.dataset.listId,
            toList: list.dataset.listId,
            group: list.dataset.group,
            endpoint: list.dataset.endpoint,
            originalOrder: this.originalOrder
        });
    },
    
    canDrop(list) {
        const sourceGroup = this.sourceList.dataset.group;
        const targetGroup = list.dataset.group;
        
        // Same list always allowed
        if (list === this.sourceList) return true;
        
        // Cross-list only if same group
        return sourceGroup && sourceGroup === targetGroup;
    },
    
    getOrder(list) {
        return Array.from(list.querySelectorAll('.sortable-item:not(.is-ghost)'))
            .map(item => item.dataset.id);
    },
    
    getDragAfterElement(list, y) {
        const items = [...list.querySelectorAll('.sortable-item:not(.is-dragging):not(.is-ghost)')];
        
        return items.reduce((closest, child) => {
            const box = child.getBoundingClientRect();
            const offset = y - box.top - box.height / 2;
            if (offset < 0 && offset > closest.offset) {
                return { offset, element: child };
            }
            return closest;
        }, { offset: Number.NEGATIVE_INFINITY }).element;
    },
    
    createGhost() {
        const ghost = document.createElement('li');
        ghost.className = 'sortable-item is-ghost';
        ghost.style.height = this.dragging.offsetHeight + 'px';
        return ghost;
    },
    
    async notifyBackend(list, payload) {
        if (!payload.endpoint) return;
        
        try {
            const response = await fetch(payload.endpoint, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            
            if (!response.ok) throw new Error(`HTTP ${response.status}`);
            
            list.dispatchEvent(new CustomEvent('sortable:success', {
                bubbles: true,
                detail: payload
            }));
        } catch (error) {
            this.rollback(list, payload.originalOrder);
            list.dispatchEvent(new CustomEvent('sortable:error', {
                bubbles: true,
                detail: { payload, error }
            }));
        }
    },
    
    rollback(list, order) {
        order.forEach(id => {
            const item = list.querySelector(`[data-id="${id}"]`);
            if (item) list.appendChild(item);
        });
    }
};

document.addEventListener('DOMContentLoaded', () => SortableList.init());
```

---

## Accessibility

### Keyboard Support

| Key | Action |
|-----|--------|
| `Space` / `Enter` | Pick up / drop item |
| `↑` / `↓` | Move item up / down |
| `Escape` | Cancel drag |
| `Tab` | Navigate between items |

### ARIA Attributes

```html
<ul class="sortable-list" 
    role="listbox"
    aria-label="Reorderable task list">
    <li class="sortable-item" 
        role="option"
        aria-grabbed="false"
        aria-describedby="sortable-instructions"
        tabindex="0">
        Task A
    </li>
</ul>

<div id="sortable-instructions" class="sr-only">
    Press Space to pick up. Use arrow keys to move. Press Space to drop.
</div>

<!-- Live region for announcements -->
<div aria-live="polite" aria-atomic="true" class="sr-only" id="sortable-announcements"></div>
```

### Screen Reader Announcements

```javascript
announce(message) {
    const el = document.getElementById('sortable-announcements');
    if (el) el.textContent = message;
}

// Usage:
this.announce('Task A picked up. Position 1 of 5.');
this.announce('Task A moved to position 3 of 5.');
this.announce('Task A dropped at position 3.');
```

---

## Touch Support

For mobile devices, touch events need additional handling:

```javascript
initItem(item) {
    // ... existing drag events ...
    
    // Touch support
    item.addEventListener('touchstart', e => this.onTouchStart(e, item));
    item.addEventListener('touchmove', e => this.onTouchMove(e, item));
    item.addEventListener('touchend', e => this.onTouchEnd(e, item));
}

onTouchStart(e, item) {
    // Long press to initiate drag (300ms)
    this.touchTimer = setTimeout(() => {
        this.dragging = item;
        this.sourceList = item.closest('[data-sortable]');
        item.classList.add('is-dragging');
        navigator.vibrate?.(50); // Haptic feedback
    }, 300);
}

onTouchMove(e, item) {
    if (!this.dragging) {
        clearTimeout(this.touchTimer);
        return;
    }
    
    e.preventDefault();
    const touch = e.touches[0];
    
    // Move item with finger
    item.style.transform = `translate(${touch.clientX}px, ${touch.clientY}px)`;
    
    // Find drop target
    const elementBelow = document.elementFromPoint(touch.clientX, touch.clientY);
    const list = elementBelow?.closest('[data-sortable]');
    if (list && this.canDrop(list)) {
        this.onDragOver({ clientY: touch.clientY, preventDefault: () => {} }, list);
    }
}

onTouchEnd(e, item) {
    clearTimeout(this.touchTimer);
    if (!this.dragging) return;
    
    // Find final drop target and complete drop
    // ... similar to onDrop ...
}
```

---

## Animation

CSS transitions provide smooth animation:

```css
.sortable-item {
    transition: transform 150ms ease;
}

.sortable-list {
    /* Prevent layout shift during drag */
    min-height: 50px;
}
```

For more complex animations (items sliding to make room), consider using FLIP technique or a library.

---

## Comparison: Native vs SortableJS

| Feature | Native Implementation | SortableJS |
|---------|----------------------|------------|
| **Size** | ~3KB (150 lines) | ~10KB |
| **Dependencies** | None | None |
| **Basic reorder** | ✅ | ✅ |
| **Cross-list groups** | ✅ | ✅ |
| **Animation** | CSS only | Built-in |
| **Touch support** | Basic | Excellent |
| **Nested lists** | ❌ | ✅ |
| **Clone on drag** | ❌ | ✅ |
| **Multi-drag** | ❌ | ✅ |
| **Pull/Put modes** | ❌ | ✅ |
| **Keyboard a11y** | Basic | Basic |

### Recommendation

- **Use native implementation** for simple reordering (80% of use cases)
- **Use SortableJS** for complex needs: nested lists, cloning, animations, better touch

---

## Integration Options

### Option A: Built into basil.js

Add ~150 lines to `basil.js`. Always available.

**Pros:** Zero setup, consistent experience
**Cons:** Bundle size for unused feature

### Option B: Separate module

```parsley
let {SortableList} = import @std/sortable
```

Loaded only when used.

**Pros:** Smaller base bundle
**Cons:** Extra import

### Option C: Lazy load

Include in basil.js but only activate when `[data-sortable]` found.

```javascript
// In basil.js
if (document.querySelector('[data-sortable]')) {
    import('/_/js/sortable.js').then(m => m.init());
}
```

**Pros:** Best of both worlds
**Cons:** Slight delay on first use

### Recommendation

**Option A** (built into basil.js) for simplicity. 150 lines / 3KB is acceptable given the feature's utility.

---

## Events

Custom events for integration:

| Event | When | Detail |
|-------|------|--------|
| `sortable:start` | Drag begins | `{ item, list }` |
| `sortable:move` | Item position changes | `{ item, list, fromIndex, toIndex }` |
| `sortable:end` | Drag ends (before backend) | `{ item, list, changed }` |
| `sortable:success` | Backend confirmed | `{ payload, response }` |
| `sortable:error` | Backend failed | `{ payload, error }` |

```javascript
document.addEventListener('sortable:success', (e) => {
    showToast('Order saved!');
});

document.addEventListener('sortable:error', (e) => {
    showToast('Failed to save order', 'error');
});
```

---

## Future Enhancements

1. **Horizontal lists** - `orientation="horizontal"` prop
2. **Grid sorting** - 2D reordering for image galleries
3. **Drag handles** - `handle=".grip"` to restrict drag initiation
4. **Disabled items** - Items that can't be moved
5. **Drop restrictions** - Limit which items can go where
6. **Optimistic locking** - ETag-based conflict detection
7. **Undo support** - Ctrl+Z to revert last change

---

## Open Questions

1. **Should reorder be optimistic?**
   - Yes (current): Update UI immediately, rollback on error
   - No: Wait for backend confirmation before updating UI
   - Recommendation: Optimistic with rollback

2. **How to handle rapid reorders?**
   - Debounce backend calls?
   - Queue and batch?
   - Recommendation: Debounce 300ms, send only final state

3. **Cross-list: should items be moved or copied?**
   - Move (current): Item leaves source list
   - Copy: Item duplicated to target
   - Recommendation: Move by default, `copy={true}` prop for copy behavior

4. **Empty list handling?**
   - Show placeholder when list is empty
   - Maintain minimum drop zone size
   - Recommendation: CSS-based placeholder

---

## Summary

A lightweight (~150 lines, 3KB) sortable list implementation that:

- Uses native HTML5 drag-and-drop
- Exposes all states via CSS classes
- Supports cross-list dragging via groups
- Auto-syncs to backend with error rollback
- Includes basic keyboard and touch support
- Emits events for external integration

This covers 80% of reordering use cases. For complex needs (nested lists, animations, cloning), users can integrate SortableJS following the documented pattern.
