---
id: FEAT-016
title: "Auto-rewrite public_dir paths to web URLs"
status: implemented
priority: high
created: 2025-12-02
implemented: 2025-12-02
---

# FEAT-016: Auto-rewrite public_dir paths to web URLs

## Summary
When Basil renders HTML output containing filesystem paths under `public_dir`, automatically rewrite them to web-root-relative URLs.

**Before:** `<img src="./public/images/foo.png"/>`
**After:** `<img src="/images/foo.png"/>`

## Motivation
Filesystem paths and web URLs are different address spaces. Currently, users must manually transform paths when using `files()` results in HTML:

```parsley
// Current workaround - ugly
let tubs = files(@./public/images/tubs/*)
for(tub in tubs) {
    <img src={"/" + tub.path.segments[1:].join("/")}/>
}
```

Should be:
```parsley
// Desired - just works
let tubs = files(@./public/images/tubs/*)
for(tub in tubs) {
    <img src={tub}/>
}
```

## Design Principles
1. **Basil-only** - Parsley remains unaware of web server concepts
2. **Config-driven** - Single `public_dir` setting, no multiple roots
3. **Predictable** - Only rewrites paths that clearly match public_dir
4. **Non-invasive** - Paths outside public_dir unchanged

## Configuration

```yaml
# basil.yaml
public_dir: ./public  # Default value
```

If not specified, defaults to `./public` (current implicit behavior).

## Rewrite Rules

| Filesystem Path | Web URL | Notes |
|-----------------|---------|-------|
| `./public/images/foo.png` | `/images/foo.png` | Standard case |
| `./public/style.css` | `/style.css` | Root of public |
| `./public/` | `/` | Directory itself |
| `./other/file.txt` | `./other/file.txt` | Not under public_dir, unchanged |
| `/absolute/path` | `/absolute/path` | Absolute paths unchanged |
| `https://example.com` | `https://example.com` | URLs unchanged |

## Implementation Options

### Option A: Response Middleware (Not Recommended)
Regex replace on HTML response body.
- ❌ Brittle - could match strings inside `<script>` or text content
- ❌ Performance - parsing entire response
- ❌ Could break legitimate uses of the string

### Option B: Parsley Path Serialization Hook (Recommended)
Basil provides a path transformer function that Parsley calls when serializing paths to strings in HTML context.

```go
// Basil provides this to Parsley
type PathTransformer func(path string, isAbsolute bool) string

// Parsley uses it in pathDictToString when in HTML context
func pathDictToString(dict *Dictionary, transformer PathTransformer) string {
    path := buildPath(dict)
    if transformer != nil {
        return transformer(path, dict.IsAbsolute())
    }
    return path
}
```

**Pros:**
- Clean separation - Parsley just calls a hook
- Only affects path→string conversion in HTML attributes
- No regex, no false positives
- Basil controls the transformation logic

**Cons:**
- Need to thread transformer through eval context
- Need to distinguish "HTML attribute context" from other string contexts

### Option C: Environment Variable (Simpler Alternative)
Basil sets `basil.public_dir` in the environment. Parsley checks it when serializing paths.

```go
// In pathDictToString or when path becomes HTML attribute
if publicDir := env.Get("basil.public_dir"); publicDir != nil {
    // Check if path starts with public_dir and transform
}
```

**Pros:**
- Simpler implementation
- No new interfaces
- Uses existing environment mechanism

**Cons:**
- Parsley gains implicit knowledge of Basil convention
- Always transforms, not just in HTML context

## Recommended Approach: Option C

Given that:
1. Parsley already knows about `basil.*` variables
2. Path-to-string conversion is well-defined
3. Transforming in all contexts is probably fine (console output, etc.)

Option C is simplest and covers the use case.

### Implementation Steps

1. **Config**: Ensure `public_dir` is in basil.yaml schema (may already exist)
2. **Environment**: Basil sets `basil.public_dir` when initializing Parsley env
3. **Path serialization**: In `pathDictToString()`, check for public_dir match and transform
4. **File handles**: Same logic for file handle paths

## Edge Cases

1. **Nested public_dir**: `public_dir: ./dist/public` → path `./dist/public/x` → `/x`
2. **Trailing slashes**: Normalize both config and paths before comparison
3. **Case sensitivity**: Match filesystem behavior (case-sensitive on Linux/Mac)
4. **URL encoding**: Paths with spaces/special chars should be URL-encoded in output

## Testing

```parsley
// With public_dir: ./public

// File paths
let f = file(@./public/images/test.png)
toString(f.path)  // Expected: "/images/test.png"

// Files glob
let files = files(@./public/css/*.css)
files[0].path     // Expected: path with components ["css", "style.css"], absolute=true? or web-path?

// In HTML
<img src={f.path}/>  // Expected: src="/images/test.png"

// Non-public paths unchanged
let other = @./data/config.json
toString(other)   // Expected: "./data/config.json"
```

## Open Questions

1. **Should file handles also get transformed?** Yes, they wrap paths.
2. **What about paths in CSS `url()`?** If CSS is served from public_dir, relative paths work. Low priority.
3. **Should this affect `toString()` globally or only HTML attributes?** Globally is simpler and probably fine.

## Non-Goals

- Multiple public directories (use nginx if needed)
- Different routes for different dirs (use nginx)
- Transformation of URLs (only filesystem paths)

## Estimate

~2-3 hours implementation + testing
