---
id: FEAT-062
title: "Parts V1.1: Auto-Refresh, Deferred Load, and Lazy Loading"
status: done
priority: medium
created: 2025-12-10
completed: 2025-12-10
author: "@human + AI"
---

# FEAT-062: Parts V1.1: Auto-Refresh, Deferred Load, and Lazy Loading

## Summary

Add auto-refresh, deferred loading, and lazy loading capabilities to Parts:
- `part-refresh={ms}` enables periodic updates for live data (dashboards, notifications, real-time stats)
- `part-load="view"` fetches a view immediately after page load (for slow data with placeholder)
- `part-lazy="view"` defers Part rendering until scrolled into viewport (performance optimization)

## User Story

As a web developer, I want Parts that can automatically refresh themselves, load slow content asynchronously with a placeholder, and defer loading until visible, so that I can build live dashboards and optimize page load performance without manual polling or complex lazy-loading code.

## Acceptance Criteria

### Auto-Refresh
- [x] `part-refresh={milliseconds}` attribute on `<Part/>` tag
- [x] Part automatically fetches current view at specified interval
- [x] Interval resets after manual interactions (click/submit)
- [x] Refresh pauses when tab is hidden (Page Visibility API)
- [x] Refresh stops when Part is removed from DOM
- [x] Props accumulate across auto-refreshes (same as manual updates)

### Deferred Load (Immediate Async)
- [x] `part-load="view"` attribute on `<Part/>` tag
- [x] Initial render shows placeholder view
- [x] Part immediately fetches specified view after page load
- [x] Works for slow data (API calls, database queries)

### Lazy Loading (Scroll-Triggered)
- [x] `part-lazy="view"` attribute on `<Part/>` tag
- [x] Initial render shows placeholder view
- [x] Part loads when scrolled into viewport (Intersection Observer)
- [x] Load threshold configurable via `part-lazy-threshold={px}`
- [x] Already-loaded Parts don't reload on re-entry
- [x] Works with nested Parts (child Parts lazy-load independently)

### Combined Use
- [x] `part-refresh` works with `part-load` (refresh starts after load)
- [x] `part-refresh` works with `part-lazy` (refresh starts after lazy load)

## Design Decisions

- **Two loading modes**: `part-load` (immediate) vs `part-lazy` (viewport-triggered) serve different use cases
- **Milliseconds not seconds**: More flexible, aligns with `setTimeout` API
- **Pause when hidden**: Saves bandwidth, battery; resumes on visibility
- **Reset on interaction**: User action implies fresh data; avoid double-fetch
- **Intersection Observer**: Modern, performant, built-in lazy-load support
- **Props accumulate**: Consistent with V1 behavior
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

### Deferred Load Syntax (Immediate Async)

```parsley
<Part 
    src={@./user-profile.part} 
    view="placeholder"
    part-load="loaded"  // Fetch "loaded" view immediately
/>
```

**Generated HTML:**

```html
<div data-part-src="/user-profile.part" 
     data-part-view="placeholder"
     data-part-props='{}' 
     data-part-load="loaded">
    <!-- Placeholder content rendered initially -->
</div>
```

### Lazy Loading Syntax (Scroll-Triggered)

```parsley
<Part 
    src={@./heavy-chart.part} 
    view="placeholder"
    part-lazy="chart"            // Load when visible
    part-lazy-threshold={200}    // Start loading 200px before entering viewport
/>
```

**Generated HTML:**

```html
<div data-part-src="/heavy-chart.part" 
     data-part-view="placeholder"
     data-part-props='{}' 
     data-part-lazy="chart"
     data-part-lazy-threshold="200">
    <!-- Placeholder content until scrolled into view -->
</div>
```

### JavaScript Runtime

The runtime handles three loading modes:

1. **Immediate Load (`data-part-load`)**: Fetches the specified view immediately after page load
2. **Lazy Load (`data-part-lazy`)**: Uses IntersectionObserver to fetch when scrolled into viewport
3. **Auto-Refresh (`data-part-refresh`)**: Polls at interval, pauses when tab hidden

For complete implementation, see `server/handler.go` (`partsRuntimeScript` function).

### Server Implementation

No server changes required. All loading modes are client-side features that use existing Part request infrastructure.

### Edge Cases

1. **Minimum interval**: Enforce 100ms minimum to prevent abuse
2. **Rapid interactions**: Reset timer on each interaction to avoid double-fetch
3. **Part removed from DOM**: Use `WeakMap` so intervals are garbage-collected
4. **Nested lazy Parts**: Each Part independently observes and loads
5. **Lazy + Refresh**: Refresh starts after initial lazy load completes
6. **Load + Refresh**: Refresh starts after initial load completes
7. **Hidden tab**: Pause refresh when `document.hidden === true`
8. **Network errors**: Keep existing interval (don't stop on error)
9. **Threshold edge cases**: Negative values treated as 0, missing defaults to 0

### Performance Considerations

- **Memory**: `WeakMap` prevents memory leaks for removed Parts
- **Battery**: Pausing on hidden tabs saves resources
- **Bandwidth**: Auto-refresh only fetches changed content (no full page reload)
- **Lazy loading**: Reduces initial payload for below-the-fold content
- **Immediate load**: Shows placeholder instantly, fetches data asynchronously

### Browser Compatibility

- **Intersection Observer**: Supported in all modern browsers (Chrome 51+, Firefox 55+, Safari 12.1+)
- **Page Visibility API**: Widely supported (IE 10+)
- **WeakMap**: Universal support (IE 11+)

## Versioned Scope

### V1.1 (This Spec)

- `part-refresh={ms}` for auto-refresh
- `part-load="view"` for immediate async loading (slow data with placeholder)
- `part-lazy="view"` for scroll-triggered lazy loading
- `part-lazy-threshold={px}` for lazy load offset
- Visibility-aware refresh (pause when hidden)
- Interaction resets refresh timer

### V1.2 (Future)

- Custom lazy-load placeholders
- Multiple refresh intervals (fast/slow based on activity)
- Refresh on reconnect (after network loss)
- Debounced refresh (coalesce rapid updates)

## Related

- Depends on: `work/specs/FEAT-061.md` (Parts V1) ✅ Complete
- Design: `work/design/DESIGN-parts.md`

## Implementation Notes

**Completed:** 2025-12-10

### Key Distinctions

| Feature | Attribute | When it loads | Use case |
|---------|-----------|---------------|----------|
| Immediate | `part-load="view"` | Right after page load | Slow API/database data with placeholder |
| Lazy | `part-lazy="view"` | When scrolled into viewport | Heavy content below the fold |
| Refresh | `part-refresh={ms}` | Every N milliseconds | Live data (clocks, notifications) |

### Attribute Summary

| Attribute | Type | Description |
|-----------|------|-------------|
| `part-refresh` | number | Auto-refresh interval in milliseconds (min 100ms) |
| `part-load` | string | View to fetch immediately after page load |
| `part-lazy` | string | View to fetch when scrolled into viewport |
| `part-lazy-threshold` | number | Pixels before viewport to trigger lazy load (default 0) |
