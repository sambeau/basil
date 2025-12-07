---
id: FEAT-045
title: "Redirect Helper Function"
status: draft
priority: medium
created: 2025-12-07
author: "@human"
---

# FEAT-045: Redirect Helper Function

## Summary
Add a `redirect(url, status?)` helper function to simplify HTTP redirects. Currently, redirects require manually setting the status code, Location header, and returning an empty body. This is verbose and error-prone.

## User Story
As a Parsley developer, I want a simple redirect function so that I can redirect users without boilerplate code.

## Acceptance Criteria
- [ ] `redirect("/path")` returns a 302 Found redirect
- [ ] `redirect("/path", 301)` returns a 301 Moved Permanently redirect
- [ ] `redirect(url)` accepts absolute URLs like `@https://example.com`
- [ ] Function terminates handler execution (like `error()`)
- [ ] Works with path literals: `redirect(@/dashboard)`
- [ ] Accepts URL objects for dynamic URLs
- [ ] Documentation updated with examples

## Design Decisions

- **Default to 302**: Temporary redirect is safer default. Use 301 explicitly for permanent moves (SEO implications)
- **Terminate execution**: Like `error()`, redirect should stop further handler execution
- **Support path literals**: `@/dashboard` should work for type safety
- **No body needed**: Redirect response has no body, function handles everything
- **Relative paths**: Relative URLs are resolved against the current request path

---
<!-- BELOW THIS LINE: AI-FOCUSED IMPLEMENTATION DETAILS -->

## Technical Context

### API Design

**Basic redirect:**
```parsley
// After form submission
if (success) {
    redirect("/dashboard")
}

// This code is never reached after redirect()
<p>This won't render</p>
```

**With status code:**
```parsley
// Permanent redirect (301) - use for moved pages
redirect("/new-location", 301)

// See Other (303) - use after POST to prevent resubmit
redirect("/result", 303)

// Temporary redirect (307) - preserves method
redirect("/maintenance", 307)
```

**With path/URL literals:**
```parsley
redirect(@/users/{userId}/profile)
redirect(@https://login.example.com/oauth)
```

**Dynamic URL:**
```parsley
let returnUrl = basil.http.request.query.return ?? "/home"
redirect(returnUrl)
```

### Supported Status Codes

| Code | Name | Use Case |
|------|------|----------|
| 301 | Moved Permanently | Page has moved forever (SEO transfers) |
| 302 | Found | Default, temporary redirect |
| 303 | See Other | After POST, redirect to GET (prevents resubmit) |
| 307 | Temporary Redirect | Like 302 but preserves HTTP method |
| 308 | Permanent Redirect | Like 301 but preserves HTTP method |

### Implementation

The function should:
1. Validate URL (string, path literal, or URL object)
2. Validate status code (must be 3xx redirect)
3. Set `basil.http.response.status = status`
4. Set `basil.http.response.headers.Location = url`
5. Return a special "redirect" signal that terminates handler evaluation

**Option A: Return special object**
```go
// In evaluator
type Redirect struct {
    URL    string
    Status int
}

func (r *Redirect) Type() ObjectType { return REDIRECT_OBJ }
```
Handler checks if result is `*Redirect` and handles accordingly.

**Option B: Modify response and return early**
Set response metadata and return a sentinel value (like `error()` does).

### Affected Components
- `pkg/parsley/evaluator/builtins.go` — Add `redirect` builtin function
- `pkg/parsley/evaluator/evaluator.go` — Handle redirect return value (like error)
- `server/handler.go` — Detect redirect result and write response

### Dependencies
- Depends on: None
- Blocks: None

### Edge Cases & Constraints
1. **Invalid status code**: Error if not 3xx (e.g., `redirect("/", 200)` should fail)
2. **Empty URL**: Error if URL is empty string
3. **Relative URLs**: Should work, browser resolves them
4. **After output**: Redirect after content started should error (can't change headers)
5. **In loops/conditions**: Should terminate entire handler, not just current block

### Current Workaround

```parsley
// Today's verbose approach
basil.http.response.status = 302
basil.http.response.headers.Location = "/dashboard"
""  // Must return empty body
```

### After Implementation

```parsley
// Clean and clear
redirect("/dashboard")
```

## Implementation Notes
*Added during/after implementation*

## Related
- Similar to: `error(code, message)` function pattern
