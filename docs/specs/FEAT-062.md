---
id: FEAT-062
title: "Parts V1.1: Auto-Refresh and Lazy Loading"
status: draft
priority: medium
created: 2025-12-10
author: "@human + AI"
---

# FEAT-062: Parts V1.1: Auto-Refresh and Lazy Loading

## Summary

Add auto-refresh and lazy loading capabilities to Parts. `part-refresh={ms}` enables periodic updates for live data (dashboards, notifications, real-time stats), while `part-load="view"` defers Part rendering until visible, improving initial page load performance.

## User Story

As a web developer, I want Parts that can automatically refresh themselves on an interval and load content only when needed, so that I can build live dashboards and optimize page load performance without manual polling or complex lazy-loading code.

## Acceptance Criteria

### Auto-Refresh
- [ ] `part-refresh={milliseconds}` attribute on `<Part/>` tag
- [ ] Part automatically fetches current view at specified interval
- [ ] Interval resets after manual interactions (click/submit)
- [ ] Refresh pauses when tab is hidden (Page Visibility API)
- [ ] Refresh stops when Part is removed from DOM
- [ ] Props accumulate across auto-refreshes (same as manual updates)

### Lazy Loading
- [ ] `part-load="view"` attribute on `<Part/>` tag
- [ ] Initial render shows placeholder (empty or custom)
- [ ] Part loads when scrolled into viewport (Intersection Observer)
- [ ] Load threshold configurable via `part-load-threshold={px}`
- [ ] Already-loaded Parts don't reload on re-entry
- [ ] Works with nested Parts (child Parts lazy-load independently)

### Combined Use
- [ ] `part-refresh` and `part-load` work together (refresh starts after load)
- [ ] Refresh respects visibility (pause when scrolled out of view)

## Design Decisions

- **Milliseconds not seconds**: More flexible, aligns with `setTimeout` API
- **Pause when hidden**: Saves bandwidth, battery; resumes on visibility
- **Reset on interaction**: User action implies fresh data; avoid double-fetch
- **Intersection Observer**: Modern, performant, built-in lazy-load support
- **Props accumulate**: Consistent with V1 behavior
- **No custom placeholders (V1.1)**: Keep simple; defer to V1.2 if needed
- **Threshold in pixels**: Easier to reason about than percentage for most use cases

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Specification

### Auto-Refresh Syntax

```parsley
<Part 
    src={@./live-stats.part} 
    view="default" 
    part-refresh={5000}  // Refresh every 5 seconds
/>
```

**Generated HTML:**

```html
<div data-part-src="/live-stats.part" 
     data-part-view="default"
     data-part-props='{}' 
     data-part-refresh="5000">
    <!-- Initial content -->
</div>
```

### Lazy Loading Syntax

```parsley
<Part 
    src={@./heavy-chart.part} 
    view="chart"
    part-load="chart"           // Load when visible
    part-load-threshold={200}    // Start loading 200px before entering viewport
/>
```

**Generated HTML (before load):**

```html
<div data-part-src="/heavy-chart.part" 
     data-part-view="chart"
     data-part-props='{}' 
     data-part-load="chart"
     data-part-load-threshold="200">
    <!-- Empty or minimal placeholder -->
</div>
```

### JavaScript Runtime Changes

**Auto-Refresh Implementation:**

```javascript
// Track active intervals by Part element
const refreshIntervals = new WeakMap();

function startAutoRefresh(part) {
    const interval = parseInt(part.dataset.partRefresh);
    if (!interval || interval < 100) return; // Minimum 100ms
    
    // Clear existing interval
    stopAutoRefresh(part);
    
    // Start new interval
    const timerId = setInterval(() => {
        if (document.hidden) return; // Pause when tab hidden
        const view = part.dataset.partView;
        const propsJson = part.dataset.partProps;
        const props = propsJson ? JSON.parse(propsJson) : {};
        refresh(part, view, props, 'GET');
    }, interval);
    
    refreshIntervals.set(part, timerId);
}

function stopAutoRefresh(part) {
    const timerId = refreshIntervals.get(part);
    if (timerId) {
        clearInterval(timerId);
        refreshIntervals.delete(part);
    }
}

// Pause/resume on visibility change
document.addEventListener('visibilitychange', () => {
    document.querySelectorAll('[data-part-refresh]').forEach(part => {
        if (document.hidden) {
            stopAutoRefresh(part);
        } else {
            startAutoRefresh(part);
        }
    });
});

// Reset interval after manual interaction
function refresh(part, view, props, method) {
    // ... existing refresh logic ...
    
    // Reset auto-refresh timer
    if (part.dataset.partRefresh) {
        startAutoRefresh(part);
    }
}
```

**Lazy Loading Implementation:**

```javascript
const lazyParts = new WeakMap(); // Track loaded state

function initLazyLoading() {
    const parts = document.querySelectorAll('[data-part-load]');
    
    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                const part = entry.target;
                
                // Load only once
                if (lazyParts.get(part)) return;
                lazyParts.set(part, true);
                
                const view = part.dataset.partLoad;
                const propsJson = part.dataset.partProps;
                const props = propsJson ? JSON.parse(propsJson) : {};
                
                refresh(part, view, props, 'GET');
                
                // Start auto-refresh after load (if configured)
                if (part.dataset.partRefresh) {
                    startAutoRefresh(part);
                }
            }
        });
    }, {
        rootMargin: getPart.dataset.partLoadThreshold || '0'} + 'px'
    });
    
    parts.forEach(part => observer.observe(part));
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    initLazyLoading();
    
    // Start auto-refresh for non-lazy Parts
    document.querySelectorAll('[data-part-refresh]:not([data-part-load])').forEach(startAutoRefresh);
});
```

### Server Implementation

No server changes required. Auto-refresh and lazy loading are client-side features that use existing Part request infrastructure.

### Edge Cases

1. **Minimum interval**: Enforce 100ms minimum to prevent abuse
2. **Rapid interactions**: Reset timer on each interaction to avoid double-fetch
3. **Part removed from DOM**: Use `WeakMap` so intervals are garbage-collected
4. **Nested lazy Parts**: Each Part independently observes and loads
5. **Lazy + Refresh**: Refresh starts after initial lazy load completes
6. **Hidden tab**: Pause refresh when `document.hidden === true`
7. **Network errors**: Keep existing interval (don't stop on error)
8. **Threshold edge cases**: Negative values treated as 0, missing defaults to 0

### Performance Considerations

- **Memory**: `WeakMap` prevents memory leaks for removed Parts
- **Battery**: Pausing on hidden tabs saves resources
- **Bandwidth**: Auto-refresh only fetches changed content (no full page reload)
- **Lazy loading**: Reduces initial payload for below-the-fold content

### Browser Compatibility

- **Intersection Observer**: Supported in all modern browsers (Chrome 51+, Firefox 55+, Safari 12.1+)
- **Page Visibility API**: Widely supported (IE 10+)
- **WeakMap**: Universal support (IE 11+)

## Versioned Scope

### V1.1 (This Spec)

- `part-refresh={ms}` for auto-refresh
- `part-load="view"` for lazy loading
- `part-load-threshold={px}` for load offset
- Visibility-aware refresh (pause when hidden)
- Interaction resets refresh timer

### V1.2 (Future)

- Custom lazy-load placeholders
- Multiple refresh intervals (fast/slow based on activity)
- Refresh on reconnect (after network loss)
- Debounced refresh (coalesce rapid updates)

## Related

- Depends on: `docs/specs/FEAT-061.md` (Parts V1) ✅ Complete
- Plan: `docs/plans/FEAT-062-plan.md` (to be created)

## Implementation Notes

*To be added during implementation*
