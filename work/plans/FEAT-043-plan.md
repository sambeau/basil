# Implementation Plan: FEAT-043 Cookie Support

## Overview
Add the ability to read and set HTTP cookies from Parsley handlers.

## Implementation Steps

### Step 1: Update buildRequestContext (server/handler.go)
Parse incoming cookies from `http.Request` and add them to `basil.http.request.cookies` as a simple dict of name→value.

### Step 2: Update buildBasilContext (server/handler.go)  
Add `cookies: {}` to `basil.http.response` for scripts to set cookies.

### Step 3: Update extractResponseMeta (server/handler.go)
Extract cookie settings from `basil.http.response.cookies` and convert to `http.Cookie` objects.

Key conversions:
- Duration literals (e.g., `@30d`) → seconds for MaxAge
- Dev vs Prod defaults for Secure/HttpOnly
- Validate SameSite=None requires Secure=true

### Step 4: Add cookie helpers (new functions in server/handler.go)
- `parseCookies(r *http.Request) map[string]string` - parse request cookies
- `buildResponseCookies(cookies map[string]interface{}, devMode bool) []*http.Cookie` - build response cookies
- `durationToSeconds(durationDict) int` - convert Parsley duration to seconds

### Step 5: Update writeResponse (server/handler.go)
Apply cookies from responseMeta to the http.ResponseWriter before writing body.

### Step 6: Add tests (server/handler_test.go or new server/cookies_test.go)
- Test reading cookies from request
- Test setting cookies with simple values
- Test setting cookies with full options
- Test secure defaults in dev vs prod mode
- Test duration conversion
- Test SameSite=None validation

### Step 7: Update documentation
- docs/parsley/reference.md - Add cookie API section
- docs/parsley/CHEATSHEET.md - Add cookie gotchas if any

## Progress Log
- [x] Step 1: buildRequestContext - Added cookies map to request context
- [x] Step 2: buildBasilContext - Added empty cookies dict to response
- [x] Step 3: extractResponseMeta - Extract cookies, pass devMode for secure defaults
- [x] Step 4: cookie helpers - Added buildCookie() and durationToSeconds()
- [x] Step 5: writeResponse - Apply cookies with http.SetCookie()
- [x] Step 6: tests - Created server/cookies_test.go with comprehensive tests
- [x] Step 7: documentation - Updated reference.md, CHEATSHEET.md, comparison table

## Implementation Notes
- Cookies are read into `basil.http.request.cookies` as a simple name→value dict
- Response cookies support both simple strings and option dicts
- Dev mode defaults: Secure=false, HttpOnly=true, SameSite=Lax
- Prod mode defaults: Secure=true, HttpOnly=true, SameSite=Lax
- Duration dicts convert via totalSeconds field for accuracy
- SameSite=None automatically forces Secure=true
