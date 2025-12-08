# Implementation Plan: FEAT-045 Redirect Helper Function

## Overview
Add a `redirect(url, status?)` helper function to simplify HTTP redirects.

## Implementation Steps

### Step 1: Add Redirect object type (evaluator/evaluator.go)
Add REDIRECT_OBJ constant and Redirect struct similar to APIError.

### Step 2: Add redirect builtin (evaluator/stdlib_api.go)
Add redirect function to the api module:
- `redirect(url)` - 302 Found
- `redirect(url, status)` - Custom status (must be 3xx)

### Step 3: Handle Redirect in handler (server/handler.go)
Check for Redirect result before processing response:
- Set Location header
- Set status code
- Write empty body

### Step 4: Tests
- Test basic redirect (302)
- Test custom status codes (301, 303, 307, 308)
- Test invalid status codes (should error)
- Test empty URL (should error)
- Test with path literals

### Step 5: Documentation
- Update reference.md
- Update CHEATSHEET.md
- Update comparison table

## Progress Log
- [ ] Step 1: Redirect type
- [ ] Step 2: redirect builtin
- [ ] Step 3: handler support
- [ ] Step 4: tests
- [ ] Step 5: documentation
