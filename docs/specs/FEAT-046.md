---
id: FEAT-046
title: "Path Pattern Matching"
status: implemented
priority: medium
created: 2025-12-07
implemented: 2025-12-08
author: "@human"
---

# FEAT-046: Path Pattern Matching

## Summary
Add a `match(path, pattern)` function that extracts named parameters from URL paths. This allows developers to define route patterns in Parsley code rather than requiring config file changes, making routing more flexible and keeping logic close to where it's used.

## User Story
As a Parsley developer, I want to extract parameters from URL paths using patterns so that I can handle dynamic routes without manual string parsing.

## Acceptance Criteria
- [x] `match(path, pattern)` returns dict of captured values on match, `null` on no match
- [x] Supports `:name` for single segment capture
- [x] Supports `*name` for rest/glob capture (multiple segments)
- [x] Supports literal segments that must match exactly
- [x] Returns `null` (not error) when pattern doesn't match
- [x] Works with `basil.http.request.path` and string paths
- [x] Pattern syntax documented with examples

## Design Decisions

- **Function not config**: Keep routing in Parsley code for flexibility and testability
- **Return null on no match**: Allows easy chaining with `??` or conditional checks
- **Express-style syntax**: `:param` is familiar from Express, Sinatra, Flask
- **Glob with `*`**: `*rest` captures remaining path segments as array
- **No regex**: Keep it simple — regex available via `~` operator if needed
- **Trailing slash flexible**: `/users/:id` matches both `/users/123` and `/users/123/`

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### API Design

**Basic parameter extraction:**
```parsley
let params = match(basil.http.request.path, "/users/:id")
// /users/123 → {id: "123"}
// /posts/456 → null

if (params) {
    let user = getUser(params.id)
}
```

**Multiple parameters:**
```parsley
let params = match(path, "/users/:userId/posts/:postId")
// /users/42/posts/99 → {userId: "42", postId: "99"}
```

**With destructuring:**
```parsley
let {userId, postId} = match(path, "/users/:userId/posts/:postId") ?? {}
```

**Glob/rest capture:**
```parsley
let params = match(path, "/files/*path")
// /files/docs/2025/report.pdf → {path: ["docs", "2025", "report.pdf"]}

// Access as array
let filePath = params.path.join("/")  // "docs/2025/report.pdf"
```

**Route dispatch pattern:**
```parsley
let path = basil.http.request.path
let method = basil.http.request.method

// Try each route pattern
if (let params = match(path, "/users/:id")) {
    if (method == "GET") { showUser(params.id) }
    else if (method == "DELETE") { deleteUser(params.id) }
}
else if (let params = match(path, "/users")) {
    if (method == "GET") { listUsers() }
    else if (method == "POST") { createUser() }
}
else if (let params = match(path, "/files/*path")) {
    serveFile(params.path)
}
else {
    error(404, "Not found")
}
```

**With site mode subpath:**
```parsley
// Handler at /api/index.pars handles /api/*
let subpath = basil.http.request.subpath.segments.join("/")
let params = match("/" + subpath, "/users/:id/posts/:postId")
```

### Pattern Syntax

| Pattern | Matches | Captures |
|---------|---------|----------|
| `/users` | `/users`, `/users/` | `{}` |
| `/users/:id` | `/users/123` | `{id: "123"}` |
| `/users/:id/posts` | `/users/42/posts` | `{id: "42"}` |
| `/:a/:b/:c` | `/x/y/z` | `{a: "x", b: "y", c: "z"}` |
| `/files/*path` | `/files/a/b/c` | `{path: ["a", "b", "c"]}` |
| `/api/*rest` | `/api/v1/users` | `{rest: ["v1", "users"]}` |
| `/*all` | `/any/thing` | `{all: ["any", "thing"]}` |

### Edge Cases

| Input | Pattern | Result |
|-------|---------|--------|
| `/users/123` | `/users/:id` | `{id: "123"}` |
| `/users/` | `/users/:id` | `null` (no id segment) |
| `/users` | `/users/:id` | `null` (no id segment) |
| `/users/123/extra` | `/users/:id` | `null` (extra segment) |
| `/users/123/extra` | `/users/:id/*rest` | `{id: "123", rest: ["extra"]}` |
| `/users/` | `/users` | `{}` (trailing slash ok) |
| `/Users/123` | `/users/:id` | `null` (case sensitive) |
| `/users/123` | `/users/:id/` | `{id: "123"}` (trailing slash in pattern ok) |

### Implementation

```go
// In stdlib or builtins
func matchPath(path string, pattern string) map[string]interface{} {
    // Normalize trailing slashes
    path = strings.TrimSuffix(path, "/")
    pattern = strings.TrimSuffix(pattern, "/")
    
    pathSegs := strings.Split(path, "/")
    patternSegs := strings.Split(pattern, "/")
    
    result := make(map[string]interface{})
    
    pi := 0 // pattern index
    for i := 0; i < len(pathSegs); i++ {
        if pi >= len(patternSegs) {
            return nil // path has extra segments
        }
        
        seg := patternSegs[pi]
        
        if strings.HasPrefix(seg, "*") {
            // Glob: capture rest of path as array
            name := seg[1:]
            result[name] = pathSegs[i:]
            return result
        }
        
        if strings.HasPrefix(seg, ":") {
            // Parameter: capture single segment
            name := seg[1:]
            result[name] = pathSegs[i]
        } else if seg != pathSegs[i] {
            // Literal: must match exactly
            return nil
        }
        
        pi++
    }
    
    // Check all pattern segments consumed
    if pi < len(patternSegs) {
        return nil
    }
    
    return result
}
```

### Affected Components
- `pkg/parsley/evaluator/builtins.go` — Add `match` builtin function
- `pkg/parsley/evaluator/evaluator.go` — Register builtin

### Dependencies
- Depends on: None
- Blocks: None

### Comparison with Alternatives

**This approach (function):**
```parsley
let params = match(path, "/users/:id")
if (params) { showUser(params.id) }
```

**Config-based (Express style):**
```yaml
routes:
  - path: /users/:id
    handler: ./user.pars
```
```parsley
// In user.pars
let id = basil.http.request.params.id
```

**Manual parsing:**
```parsley
let segments = path.split("/")
if (segments[1] == "users" && segments[2]) {
    let id = segments[2]
}
```

Function approach wins because:
- No config file access needed
- Can match multiple patterns in one handler
- Testable in isolation
- Works with site mode subpaths

## Implementation Notes
*Added during/after implementation*

### Implementation Details (2025-12-08)

**Files modified:**
- `pkg/parsley/evaluator/evaluator.go` — Added `match` builtin function and `matchPathPattern` helper
- `server/match_test.go` — Comprehensive tests (15 test functions)
- `docs/parsley/reference.md` — Added Path Pattern Matching section in Utility Functions
- `docs/parsley/CHEATSHEET.md` — Added Path Pattern Matching section after Redirects

**Key decisions:**
1. `match()` is a global builtin (not in a module) since it's a general-purpose utility
2. Returns `NULL` on no match (not error) for easy conditional checking
3. Trailing slashes are normalized for both path and pattern
4. Glob capture (`*name`) returns an array of remaining segments
5. Empty glob (no remaining segments) returns empty array, not null
6. Path dict support via `pathDictToString` for path literal arguments

**Test coverage:**
- Basic single parameter
- Multiple parameters
- Glob capture
- Literal-only patterns
- No match cases (wrong prefix, missing segment, extra segment, case sensitivity)
- Trailing slash handling
- Mixed param + glob
- Empty glob
- Root path
- Catch-all glob
- Invalid arguments
- Special characters in segments

## Related
- Alternative to: Express-style route params in config
- Complements: Site mode `basil.http.request.subpath`
