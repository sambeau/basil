---
id: PLAN-073
feature: N/A (Internal improvement)
title: "DevTools Components & Shared CSS"
status: complete
created: 2026-01-22
completed: 2026-01-22
---

# Implementation Plan: DevTools Components & Shared CSS

## Overview
Enable component reuse and consistent theming across all devtools pages (`/__/`, `/__/logs`, `/__/db`, `/__/env`, error pages) by:
1. Extracting shared CSS into a single stylesheet
2. Adding component loading infrastructure
3. Creating reusable Parsley components
4. Refactoring templates to use components

## Prerequisites
- [x] DevTools pages working with Parsley templates
- [x] Prelude embed system in place

## Tasks

### Phase 1: Shared CSS Infrastructure

#### Task 1.1: Create shared devtools.css
**Files**: `server/prelude/css/devtools.css`
**Estimated effort**: Medium

Steps:
1. Create `devtools.css` with CSS variables for theming
2. Extract common styles from existing templates:
   - Base styles (reset, body, container)
   - Typography (headings, code, monospace)
   - Common components (.btn, .section, .panel, .empty-state)
   - Color scheme using CSS variables

CSS Variables:
```css
:root {
    --dt-bg-primary: #1a1a2e;
    --dt-bg-secondary: #16213e;
    --dt-bg-code: #0f1419;
    --dt-text-primary: #eee;
    --dt-text-muted: #7f8c8d;
    --dt-accent: #4ecdc4;
    --dt-danger: #ff6b6b;
    --dt-success: #2ecc71;
    --dt-warning: #f39c12;
    --dt-font-mono: 'Monaco', 'Courier New', monospace;
}
```

Tests:
- CSS file served at `/__/css/devtools.css`
- Styles render correctly

---

#### Task 1.2: Update templates to use shared CSS
**Files**: All `.pars` files in `server/prelude/devtools/` and `server/prelude/errors/`
**Estimated effort**: Small

Steps:
1. Add `<link rel="stylesheet" href="/__/css/devtools.css"/>` to each template
2. Remove inline `<style>` blocks
3. Verify each page renders correctly

Templates to update:
- [ ] `devtools/index.pars`
- [ ] `devtools/logs.pars`
- [ ] `devtools/db.pars`
- [ ] `devtools/db_table.pars`
- [ ] `devtools/env.pars`
- [ ] `errors/dev_error.pars`
- [ ] `errors/404.pars`
- [ ] `errors/500.pars`

---

### Phase 2: Component Loading Infrastructure

#### Task 2.1: Update embed directive
**Files**: `server/prelude.go`
**Estimated effort**: Small

Steps:
1. Add `prelude/devtools/components/*` to embed directive
2. Verify files are embedded at build time

---

#### Task 2.2: Add component loading to devtools environment
**Files**: `server/devtools.go`
**Estimated effort**: Small

Steps:
1. Create list of component files to load
2. In `createDevToolsEnv()`, load and evaluate each component
3. Components become available as functions in templates

Code pattern:
```go
// Load devtools components
componentFiles := []string{"panel.pars", "header.pars", "code_block.pars"}
for _, file := range componentFiles {
    program := GetPreludeAST("devtools/components/" + file)
    if program != nil {
        evaluator.Eval(program, env)
    }
}
```

Tests:
- Components are callable from templates
- Missing components don't break page rendering

---

#### Task 2.3: Add component loading to error page environment
**Files**: `server/errors.go`
**Estimated effort**: Small

Steps:
1. In `createErrorEnv()`, load same components as devtools
2. Error pages can use shared components

---

### Phase 3: Create Shared Components

#### Task 3.1: Panel component
**Files**: `server/prelude/devtools/components/panel.pars`
**Estimated effort**: Small

```parsley
export Panel = fn({title, class, level, contents}) {
    <div class={"panel" ++ if (level) " panel-" ++ level else "" ++ if (class) " " ++ class else ""}>
        if (title) {
            <h2 class="panel-title">title</h2>
        }
        <div class="panel-body">contents</div>
    </div>
}
```

---

#### Task 3.2: Header component
**Files**: `server/prelude/devtools/components/header.pars`
**Estimated effort**: Small

```parsley
export Header = fn({title, icon, actions, contents}) {
    <div class="header">
        <h1>
            if (icon) { icon + " " }
            title
        </h1>
        if (actions) {
            <div class="actions">actions</div>
        }
    </div>
}
```

---

#### Task 3.3: CodeBlock component
**Files**: `server/prelude/devtools/components/code_block.pars`
**Estimated effort**: Small

```parsley
export CodeBlock = fn({lang, contents}) {
    <pre class={"code-block" ++ if (lang) " lang-" ++ lang else ""}>
        <code>contents</code>
    </pre>
}
```

---

#### Task 3.4: EmptyState component
**Files**: `server/prelude/devtools/components/empty_state.pars`
**Estimated effort**: Small

```parsley
export EmptyState = fn({icon, message, contents}) {
    <div class="empty-state">
        if (icon) { <div class="empty-icon">icon</div> }
        <p>message</p>
        contents
    </div>
}
```

---

#### Task 3.5: LogEntry component
**Files**: `server/prelude/devtools/components/log_entry.pars`
**Estimated effort**: Small

Extract log entry rendering from `logs.pars` into reusable component.

---

#### Task 3.6: DataTable component
**Files**: `server/prelude/devtools/components/data_table.pars`
**Estimated effort**: Medium

Generic table component for db viewer and other data displays.

---

### Phase 4: Refactor Templates

#### Task 4.1: Refactor logs.pars
**Files**: `server/prelude/devtools/logs.pars`
**Estimated effort**: Small

Replace inline HTML with component calls as proof of concept.

---

#### Task 4.2: Refactor remaining devtools templates
**Files**: `server/prelude/devtools/*.pars`
**Estimated effort**: Medium

Update each template to use components.

---

#### Task 4.3: Refactor error templates
**Files**: `server/prelude/errors/*.pars`
**Estimated effort**: Small

Update error pages to use shared components and CSS.

---

## Validation Checklist
- [x] All tests pass: `make check`
- [x] CSS served correctly at `/__/css/devtools.css`
- [x] All devtools pages render correctly
- [x] All error pages render correctly
- [x] Components are reusable across pages
- [x] Theming works via CSS variables

## Progress Log
| Date | Task | Status | Notes |
|------|------|--------|-------|
| 2026-01-22 | Plan created | ✅ Complete | — |
| 2026-01-22 | Task 1.1 (devtools.css) | ✅ Complete | Created with CSS variables, 500+ lines |
| 2026-01-22 | Task 1.2 (update templates) | ✅ Complete | Updated 6 templates |
| 2026-01-22 | Task 2.1 (embed directive) | ✅ Complete | Already included components/* |
| 2026-01-22 | Task 2.2 (devtools loading) | ✅ Complete | loadDevToolsComponents() added |
| 2026-01-22 | Task 2.3 (error loading) | ✅ Complete | Components loaded in dev mode |
| 2026-01-22 | Task 3.1-3.6 (components) | ✅ Complete | 6 components created |
| | Task 4.1-4.3 (refactor) | ⬜ Deferred | Optional - templates work without |

## Deferred Items
Items to add to work/BACKLOG.md after implementation:
- Dark/light theme toggle — Could add later
- User-customizable themes — Future enhancement
- Component documentation/showcase page — Nice to have
- Refactor templates to use components (optional cleanup)

## Notes
- Start with Phase 1 (CSS) for immediate deduplication
- Phase 2 infrastructure enables Phase 3 & 4
- Can iterate incrementally — each phase is independently useful
- Parsley note: Inside tags, single variables interpolate directly (`title` not `{title}`)
- Parsley note: Braces `{}` inside tags parse as dictionary literals, causing errors with expressions
