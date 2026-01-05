---
id: FEAT-040
title: "Filesystem-Based Routing"
status: implemented
priority: medium
created: 2025-12-07
author: "@human"
---

# FEAT-040: Filesystem-Based Routing

## Summary
Add a "site mode" routing option where URL paths map to handler files in a directory tree. This provides batteries-included routing for beginners while preserving the explicit routes model for advanced users. When a path is requested, Basil walks back from the leaf toward the root until it finds a handler file, passing the remaining path segments to the handler.

**Handler Resolution (Priority Order):**
1. `{foldername}/{foldername}.pars` - Folder-named index (e.g., `/admin/admin.pars`)
2. `{foldername}/index.pars` - Traditional index file (fallback)

The folder-named convention makes it easier to identify files in editors when you have many folders open.

## User Story
As a developer building a content-driven site, I want URLs to automatically map to handler files in my folder structure so that I don't need to manually configure every route.

## Acceptance Criteria
- [x] New `site:` config option specifies filesystem routing root
- [x] `site:` and `routes:` are mutually exclusive (validation error if both present)
- [x] `static:` continues to work alongside `site:` (orthogonal concerns)
- [x] Given path `/reports/2025/Q4/`, Basil walks back looking for a handler
- [x] First handler found (`{folder}.pars` or `index.pars`) is executed
- [x] Folder-named files (e.g., `admin/admin.pars`) take precedence over `index.pars`
- [x] Remaining path available via `basil.http.request.subpath` (Path object with `.segments` array)
- [x] Returns 404 if no handler found walking back to site root
- [x] Handlers can explicitly return 404 for paths they don't handle
- [x] Only handler files are executed; other `.pars` files are not directly routable
- [x] Directory requests without trailing slash redirect to trailing slash (SEO canonical)
- [x] Query parameters (`?foo=bar`) continue to work as normal

## Design Decisions

- **Mutually exclusive modes**: `site:` and `routes:` cannot be combined. This keeps the mental model simple—choose explicit routing or filesystem routing, not both. Users needing hybrid approaches can use filesystem routing with explicit 404 handling in their `index.pars`.

- **Walk-back algorithm**: Rather than requiring a handler at every level, Basil walks up the tree from the requested path. This allows a single handler to handle an entire subtree (e.g., `/reports/index.pars` handles `/reports/2025/Q4/data.pdf`).

- **Path as data**: The remaining path after the handler location becomes input data to the handler. This enables clean URL designs where the path encodes parameters (e.g., `/reports/2025/Q4/` → handler receives `["2025", "Q4"]`).

- **Folder-named indexes**: Files matching their folder name (e.g., `admin/admin.pars`) are checked before `index.pars`. This improves editor usability—tabs show `admin.pars`, `edit.pars` instead of multiple `index.pars` tabs. This convention is optional; `index.pars` still works as before.

- **One site per server**: Multi-site/virtual host support is deferred. Use a reverse proxy (Caddy, nginx) for multi-site deployments.

- **Security boundary unchanged**: The site folder is within the handler root. Basil never serves raw `.pars` files. Only `index.pars` is executed; other files in the tree remain private.

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### Config Changes

```yaml
# New site mode (filesystem routing)
site: ./site                    # path relative to config file

static:
  - path: /static/
    root: ./public

# sqlite, auth, etc. continue to work
```

```yaml
# Existing explicit mode (unchanged)
routes:
  - path: /
    handler: ./handlers/index.pars
  - path: /api/users
    handler: ./handlers/api/users.pars

static:
  - path: /static/
    root: ./public
```

### Request Flow

```
Request: GET /reports/2025/Q4/

1. Check static routes first (no match)
2. Site mode routing:
   a. Check /site/reports/2025/Q4/index.pars → not found
   b. Check /site/reports/2025/index.pars → not found
   c. Check /site/reports/index.pars → FOUND
3. Execute /site/reports/index.pars with:
   - basil.http.request.path = "/reports/2025/Q4/"
   - basil.http.request.subpath = Path object (@/2025/Q4/)
     - .string = "/2025/Q4/"
     - .segments = ["2025", "Q4"]
```

### Environment Additions

```parsley
basil.http.request.subpath            // Path object for remaining path after index.pars
basil.http.request.subpath.string     // "/2025/Q4/"
basil.http.request.subpath.segments // ["2025", "Q4"]
```

### Affected Components
- `config/config.go` — Add `Site` field, validation for mutual exclusion with `Routes`
- `config/load.go` — Parse `site:` from YAML
- `server/server.go` — Route setup logic to choose site vs explicit mode
- `server/site_handler.go` — New file: walk-back routing logic
- `server/handler.go` — Add `subpath` and `pathSegments` to basil context

### Edge Cases & Constraints

1. **Trailing slash normalization** — `/reports/2025` redirects to `/reports/2025/` (302) for canonical URLs. This matches traditional web server behavior.

2. **Root index.pars** — `/site/index.pars` handles `/` and any path with no other `index.pars` higher in the tree.

3. **Empty subpath** — When request exactly matches `index.pars` location, `subpath.segments` is `[]` (empty array). The `subpath.string` property returns `"."` for relative paths.

4. **File extensions in path** — `/reports/2025/Q4/data.pdf` still walks back to find `index.pars`. The handler decides whether to serve a file, generate content, or 404.

5. **Dot files/folders** — Paths containing `.` segments (e.g., `/.git/`) should be rejected for security. Return 404.

6. **Path traversal** — `/../` in paths must be rejected or normalized. Standard security practice.

7. **Case sensitivity** — Follow OS filesystem behavior (case-sensitive on Linux, case-insensitive on macOS/Windows in dev mode).

### Example Directory Structure

```
myapp/
├── basil.yaml
├── public/
│   ├── style.css
│   └── favicon.ico
├── site/
│   ├── index.pars              # Handles /
│   ├── about/
│   │   └── index.pars          # Handles /about/
│   ├── reports/
│   │   └── index.pars          # Handles /reports/* (walk-back)
│   └── api/
│       ├── index.pars          # Handles /api/
│       └── users/
│           └── index.pars      # Handles /api/users/*
└── modules/
    └── components.pars         # Private, importable
```

### Example Handler

**site/reports/index.pars:**

```parsley
let segments = basil.http.request.subpath.segments
let subpath = basil.http.request.subpath

// Handle /reports/ (no subpath)
if (segments.length() == 0) {
  <html>
    <body><h1>All Reports</h1></body>
  </html>
}

// Handle /reports/2025/Q4/ etc.
let year = segments[0] ?? null
let quarter = segments[1] ?? null

if (year and quarter) {
  let reportPath = @~/data/reports/{year}/{quarter}/report.pdf
  if (reportPath.exists()) {
    // Serve the PDF or render a viewer
    <html>
      <body>
        <h1>Report: {year} {quarter}</h1>
        <a href={"/data/reports/{year}/{quarter}/report.pdf"}>Download PDF</a>
      </body>
    </html>
  } else {
    error(404, "Report not found")
  }
} else {
  error(404, "Invalid report path")
}
```

## Implementation Notes
*Added during/after implementation*

- **2025-12-08**: Implemented in `server/site.go` with walk-back routing algorithm
- **2026-01-05**: Added folder-named index convention (`{folder}/{folder}.pars` checked before `index.pars`)
- Config validation added to `config/load.go` (mutual exclusion check)
- `basil.http.request.subpath` is a Path object with `__type: "path"`, `absolute: false`, and `segments: [...]`
- Empty subpath (exact handler match) has empty segments array `[]`
- Trailing slash redirect returns 302 Found
- Security: path traversal (`..`) returns 400 Bad Request, dotfiles return 404
- Static files from `public_dir` are served before walk-back routing
- Site handler uses existing `parsleyHandler` infrastructure for executing `index.pars` files

## Related
- Design doc: `docs/parsley/design/Filepath routing.md`
- Related: FEAT-041 (Public files) — component-local assets
