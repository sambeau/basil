# Search Feature Design

**Status:** Draft  
**Date:** 2026-01-09  
**Author:** AI + Sam

## Overview

Add full-text search functionality to Basil using SQLite FTS5, providing a "batteries included" search solution for small-to-medium websites and internal tools.

## Goals

1. **Simple to use**: Index documents with minimal configuration
2. **Elegant API**: Feel native to Parsley (path literals, operators, etc.)
3. **Composable**: Works with existing file I/O, HTTP, database patterns
4. **90% coverage**: Solve common search needs for hundreds to low thousands of documents
5. **No external dependencies**: Use SQLite FTS5 (already available)

## Non-Goals

- Elasticsearch/Solr-level features (fuzzy search, faceting, instant search)
- Multi-language support beyond English (Porter stemmer limitation)
- Distributed search across multiple servers
- Real-time indexing with sub-second latency

## Use Cases

### Primary Use Case: Static Markdown Documentation

```parsley
// Simplest possible usage - defaults to everything (recursive scan)
let search = @SEARCH({watch: @./docs})
let results = search.query("hello world")

// With options (all optional, shown with defaults)
let search = @SEARCH({
    watch: @./docs,
    extensions: [".md"],       // Default: [".md", ".html"]
    weights: {
        title: 10.0,           // Default
        headings: 5.0,         // Default
        content: 1.0           // Default
    }
})

// Query in a handler
let {query} = import @basil/http
let results = search.query(query.q, {limit: 20})

// Render results
for (item in results.items) {
    <article>
        <h3><a href={item.url}>{item.title}</a></h3>
        <p class="snippet">{item.snippet}</p>
        <div class="meta">
            {if (item.date) {
                <time>{item.date.format("short")}</time>
            }}
        </div>
    </article>
}
```

### Secondary@ Use Case: Dynamic Content

```parsley
// Manual indexing for database-driven content
let search = SEARCH({backend: @./search.db})

// Add a document
search.add({
    url: "/blog/my-post",
    title: "My Post",
    content: markdownText,
    tags: ["tutorial", "beginner"],
    date: @2024-01-15
})

// Update when content changes
search.update("/blog/my-post", {content: updatedText})

// Remove when deleted
search.remove("/blog/my-post")
```

### Tertiary Use Case: Mixed Static + Dynamic

```parsley
// Watch folder + manual additions
let search = @SEARCH({
    watch: @./content,
    backend: @./search.db
})

// Static files auto-indexed from ./content
// Plus manual additions for dynamic content
search.add({
    url: "/user/profile/123",
    title: "Alice's Profile",
    content: bioText
})

// Multiple watch folders (indexes all recursively)
let search = @SEARCH({
    watch: [@./docs, @./blog, @./guides]
})
```

### Factory Function: @SEARCH()

```parsley
let search = SEARCH(options)

// Options:
{
    backend: @./search.db,           // SQLite database (required)
    watch: @./content,                // Optional: auto-index folder (recursive)
                                      // Also accepts array: [@./docs, @./blog]
    extensions: [".md", ".html"],     // File types to index
    
    // Field weights for ranking (BM25)
    weights: {
        title: 10.0,      // Default: 10x weight
        headings: 5.0,    // Default: 5x weight
        tags: 3.0,        // Default: 3x weight
        content: 1.0      // Default: 1x weight (baseline)
    },
    
    // Snippet options
    snippetLength: 200,               // Characters (default: 200)
    highlightTag: "mark",             // HTML tag (default: "mark")
    
    // Metadata extraction (for .md files)
    extractTitle: true,               // From frontmatter or first H1
    extractTags: true,                // From frontmatter
    extractDate: true                 // From frontmatter
}
```

### Methods

```parsley
// Query
let results = search.query(query, options)

// Options for query:
{
    limit: 10,                        // Results per page (default: 10)
    offset: 0,                        // Pagination offset (default: 0)
    raw: false,                       // Use raw FTS5 syntax (default: false)
    filters: {                        // Filter by metadata
        tags: ["tutorial"],
        dateAfter: @2024-01-01,
        dateBefore: @2024-12-31
    }
}

// Results structure:
{
    query: "parsley syntax",
    total: 42,
    items: [
        {
            url: "/docs/syntax",
            title: "Parsley Syntax Guide",
            snippet: "Learn about <mark>parsley</mark> <mark>syntax</mark>...",
            score: 0.85,
            tags: ["tutorial", "reference"],
            date: @2024-01-15
        }
    ]
}

// Manual indexing
search.add({
    url: "/path",               // Required: unique identifier
    title: "Title",             // Required
    content: "...",             // Required: markdown or plain text
    tags: ["tag1", "tag2"],     // Optional
    date: @2024-01-15           // Optional
})

search.update(url, fields)      // Update specific fields
search.remove(url)              // Remove document

// Maintenance
search.reindex()                // Full reindex (for watched folders)
search.stats()                  // Get index statistics
```

### Query Results

```parsley
// Simple query
let results = search.query("hello world")

// Access results
results.total           // Total matching documents
results.items           // Array of result objects
results.items[0].url    // First result URL
results.items[0].title  // First result title
results.items[0].snippet  // Highlighted snippet

// Pagination
let page1 = search.query("query", {limit: 10, offset: 0})
let page2 = search.query("query", {limit: 10, offset: 10})

// Filtering
let filtered = search.query("tutorial", {
    filters: {tags: ["beginner"]}
})

// Raw FTS5 syntax (for power users)
let advanced = search.query('title:parsley OR content:"syntax guide"', {
    raw: true  // No sanitization, passed directly to FTS5
})
```

## Technical Design

### SQLite FTS5 Schema

```sql
-- Main FTS5 table
CREATE VIRTUAL TABLE documents_fts USING fts5(
    title,           -- Weighted 10x
    headings,        -- Weighted 5x  
    tags,            -- Weighted 3x
    content,         -- Weighted 1x (baseline)
    url UNINDEXED,   -- Store but don't index
    date UNINDEXED,  -- Store but don't index
    tokenize='porter'  -- English stemming
);

-- BM25 ranking with custom weights
SELECT 
    url,
    title,
    snippet(documents_fts, 3, '<mark>', '</mark>', '...', 32) as snippet,
    bm25(documents_fts, 10.0, 5.0, 3.0, 1.0) as score
FROM documents_fts
WHERE documents_fts MATCH ?
ORDER BY score
LIMIT ? OFFSET ?;
```

### Markdown File Processing

For `.md` files with frontmatter:

```markdown
---
title: Getting Started
tags: [tutorial, beginner]
date: 2024-01-15
---

# Getting Started

Welcome to **Basil**! This guide will help you...

## Installation

First, install Basil...
```

**Extraction logic:**
1. Parse frontmatter (YAML between `---` delimiters)
2. Extract `title`, `tags`, `date` from frontmatter
3. Fallback: use first H1 if no title in frontmatter
4. Extract all headings (H1-H6) as separate field
5. Strip markdown formatting from content
6. Store URL based on file path: `./docs/guide.md` → `/docs/guide`

### File Watching

When `watch` option is provided:

1. Initial index: **recursively** scan folder tree, index all matching files
   - `watch: @./docs` indexes `./docs/**/*.md` (all subdirectories)
   - `watch: [@./docs, @./blog]` indexes both trees
2. Watch for changes: use existing file watcher (if Basil has one)
3. On file change: re-index that file
4. On file delete: remove from index
5. On file add: index new file

**Recursive by default:** Walks entire directory tree. No option to limit depth in Phase 1.

**Alternative (simpler):** No automatic watching. Provide `search.reindex()` method for manual full reindex.

### Snippet Generation

Use SQLite FTS5's `snippet()` function:

```sql
snippet(table, column, before, after, ellipsis, tokens)
```

Example:
```sql
snippet(documents_fts, 3, '<mark>', '</mark>', '...', 32)
-- Column 3 = content field
-- 32 tokens ≈ 150-200 characters
```

**Advantages:**
- Automatic context extraction around matches
- Handles multiple matches in same snippet
- Database Sharing:**
```parsley
// Search can share database with app
let db = @DB(@./app.db)
let search = @SEARCH({backend: @./app.db})

// Both share same database file
// Search creates its own FTS5 tables (namespaced)
// No conflicts with app
let search = SEARCH({backend: @./app.db})

// Both share same database file
// But search creates its own FTS5 tables
```

**HTTP Integration:**
```parsley
// Handler example
let {query} = import @basil/http

if (query.q) {
    let results = search.query(query.q)
    // render results
}
```

## Basil Built-in: @SEARCH

**Decision:** Make `@SEARCH` a Basil built-in (like `@DB`), not a Parsley library. This allows:
- Automatic persistence management
- Background file watching
- Intelligent caching
- Dev tools integration

**Not available in standalone Parsley** - this is Basil server infrastructure.

```parsley
// Basil handler - @SEARCH is globally available
let search = @SEARCH({watch: @./docs})
let results = search.query("hello world")
```

**Multiple Search Instances:**

`@SEARCH()` is **not a singleton** - you can create multiple search instances with different configurations:

```parsley
// Multiple search instances in same handler
let docsSearch = @SEARCH({watch: @./docs})
let blogSearch = @SEARCH({watch: @./blog, tokenizer: "unicode61"})

// Each has independent index
let docResults = docsSearch.query("installation")
let blogResults = blogSearch.query("announcement")
```

**Caching behavior:**
- Same config → same handle (cached): `@SEARCH({watch: @./docs})` called twice returns same instance
- Different config → different handle: `@SEARCH({watch: @./docs})` and `@SEARCH({watch: @./blog})` are separate
- Cache key includes all options (watch paths, tokenizer, weights, etc.)

**Common pattern (90% of apps):**
```parsley
// Single search for entire site
let search = @SEARCH({watch: [@./docs, @./blog, @./guides]})
```

Optional helper in `@std/search` for manual highlighting (if needed):

```parsley
let {highlight} = import @std/search

let highlighted = highlight("hello world", "world", "mark")
// "hello <mark>world</mark>"
```

## Example: Complete Search Page

```parsley
// search.pars - Handler for /search
let {query} = import @basil/http
let {SEARCH} = import @std/search

// Initialize search (cache this in production)
let search = SEARCH({
    backend: @./search.db,
    watch: @./docs

// Initialize search (automatically cached by Basil)
let search = @SEARCH({watch: @./docs
let results = if (q != "") {
    search.query(q, {limit: limit, offset: offset})
} else {
    null
}

<html>
<head>
    <title>{if (q) `Search: {q}` else "Search"}</title>
    <style>
        "mark { background: yellow; }"
    </style>
</head>
<body>
    <form method="GET">
        <input 
            type="search" 
            name="q" 
            value={q} 
            placeholder="Search documentation..." 
            autofocus={true}
        />
        <button type="submit">"Search"</button>
    </form>
    
    {if (results != null) {
        <div class="results">
            <p>{results.total} results for "{q}"</p>
            
            {if (results.items.length() > 0) {
                for (item in results.items) {
                    <article>
                        <h3>
                            <a href={item.url}>{item.title}</a>
                        </h3>
                        <p class="snippet">{item.snippet}</p>
                        <div class="meta">
                            {if (item.tags && item.tags.length() > 0) {
                                for (tag in item.tags) {
                                    <span class="tag">{tag}</span>
                                }
                            }}
                            {if (item.date) {
                                <time>{item.date.format("short")}</time>
                            }}
                        </div>
                    </article>
                }
            } else {
                <p>"No results found for \"{q}\""</p>
            }}
            
            {if (results.total > limit) {
                <nav class="pagination">
                    {if (page > 1) {
                        <a href={`?q={q}&page={page - 1}`}>"Previous"</a>
                    }}
                    <span>"Page {page} of {(results.total / limit) + 1}"</span>
                    {if (page * limit < results.total) {
                        <a href={`?q={q}&page={page + 1}`}>"Next"</a>
                    }}
                </nav>
            }}
        </div>
    }}
</body>
</html>
```

## Example: Blog Search with Filters

```parsley
// blog-search.pars
let {query} = import @basil/http
let {SEARCH} = import @std/search

let search = SEARCH({backend: @./blog.db})

let q = query.q ?? ""
let tag = query.tag ?? null

// Build filters
let filters = if (tag) {
    {tags: [tag]}

let search = @

let results = if (q != "") {
    search.query(q, {
        limit: 20,
        filters: filters
    })
} else {
    null
}

<html>
<head>
    <title>"Blog Search"</title>
</head>
<body>
    <form method="GET">
        <input type="search" name="q" value={q}/>
        <select name="tag">
            <option value="">"All tags"</option>
            <option value="tutorial" selected={tag == "tutorial"}>
                "Tutorials"
            </option>
            <option value="news" selected={tag == "news"}>
                "News"
            </option>
        </select>
        <button type="submit">"Search"</button>
    </form>
    
    {if (results) {
        for (item in results.items) {
            <article>
                <h2><a href={item.url}>{item.title}</a></h2>
                <p>{item.snippet}</p>
            </article>
        }
    }}
</body>
</html>
```

## Example: CLI Reindexing

```parsley
#!/usr/bin/env pars
// reindex.pars - Rebuild search index

let {SEARCH} = import @std/search

log("Rebuilding search index...")

let search = SEARCH({
    backend: @./search.db,
    watch: @./docs
})

search.reindex()

letbash
# CLI reindexing via HUP signal
kill -HUP $(cat basil.pid)  # Triggers full reindex

# Or via basil CLI (if we add it)
basil search reindex
```

```parsley
// Or programmatically in a handler
let search = @SEARCH({watch: @./docs})**Searching:**
- Simple queries: <1ms for thousands of docs
- Complex queries with filters: 1-5ms
- 99th percentile: <10ms

**Storage:**
- FTS5 index: ~2-3x original content size
- 1000 docs × 5KB = ~5MB content → ~10-15MB index

### Scaling Limits

**Comfortable:**
- 100-10,000 documents
- 1-50 MB total content
- Single server

**Reasonable:**
- 10,000-100,000 documents
- 50-500 MB content
- May need optimization (index tuning, caching)

**Not Recommended:**
- >100,000 documents
- >500 MB content
- Consider external search engine (Meilisearch, Typesense)

## Implementation Phases

### Phase 1: Core FTS5 Backend (MVP)
- SQLite FTS5 wrapper in Go
- `SEARCH()` factory function
- `.query()` with basic options
- `.add()`, `.remove()`, `.update()`
- Simple snippet generation

**Success Criteria:**
- Can index 100 documents
- Can search and get results
- Snippets show highlighted matches

### Phase 2: Markdown Integration
- Frontmatter parsing (YAML)
- Auto-extract title from first H1
- Auto-extract headings
- Metadata (tags, date)

**Success Criteria:**
- C@SEARCH()` factory function (Basil built-in)
- `.query()` with basic options
- Simple snippet generation (FTS5 `snippet()`)
- Query sanitization (AND by default)
- Full reindex only (manual trigger)

**Success Criteria:**
- Simplest API works: `@SEARCH({watch: @./docs}).query("hello")`
- Can index 100 markdown files
- Search returns highlighted snippets
- <10ms query latency

### Phase 2: Auto-Indexing & Metadata
- Initial folder scan on first `@SEARCH()` call
- Frontmatter parsing (YAML)
- Auto-extract title from frontmatter or first H1
- Auto-extract headings
- Metadata (tags, date)
- Search handle caching (per-config)

**Success Criteria:**
- Zero-config works for documentation sites
- Metadata filters work
- Second request uses cached handle

### Phase 3: Incremental Updates
- Track file mtimes in metadata table
- Check mtimes on each request
- Update only changed files
- Background watching (if file watcher available)

**Success Criteria:**
- File changes detected automatically
- Update overhead <2ms per request
- No full reindex needed

### Phase 4: Manual Indexing & Advanced
- `.add()`, `.remove()`, `.update()` methods
- Dynamic content support
- Statistics (`.stats()`)
- Dev tools integration
- Tokenizer options (porter vs unicode61)

**Success Criteria:**
- Can index database-driven content
- Mixed static + dynamic works
- Dev tools show index status
1. Basil maintains in-memory cache of search index handles
2. Check file mtimes against database metadata
3. Update only changed/new/deleted files (incremental)
4. Zero cost if nothing changed

**Background Watching (Optional):**
- If file watcher available, updates happen in background
- No per-request cost
- Database always fresh

**Manual Reindex:**
```parsley
search.reindex()  // Force full rebuild
```

### Caching Strategy

**Recommendation: Hybrid approach**

1. **SQLite Page Cache** (automatic, free):
   - SQLite caches recently-used database pages in memory
   - Typically 2MB default, configurable
   - Subsequent queries hit cache (microsecond latency)
   - No code needed - it just works

2. **Search Handle Cache** (Basil managed):
   - Cache `@SEARCH()` instances **by configuration**
   - Same config → same handle (reused across requests)
   - Different config → different handle
   - Example: `@SEARCH({watch: @./docs})` cached separately from `@SEARCH({watch: @./blog})`
   - Single SQLite connection pool per instance

3. **No Full-Index Memory Cache**:
   - Don't load entire index into RAM
   - Let SQLite handle it (proven, optimized)
   - Keeps memory footprint low

**Result:** <1ms queries without memory bloat, supports multiple search instances

### Update Strategy

**Recommendation: Incremental + Manual Full Reindex**

**Incremental Updates (automatic):**
- On each `@SEARCH()` call, check watched folders
- Compare file mtimes against database metadata
- Update only changed files
- Fast: ~1-2ms overhead for 1000 files (stat calls)

**Full Reindex:**
- Triggered by `search.reindex()` or server HUP signal
- Rebuilds entire index from scratch
- Useful after config changes or corruption

**Complexity:**
- Incremental: Medium (need mtime tracking, differential update)
- Full reindex: Simple (drop tables, rescan)
- **Start with full reindex only, add incremental in Phase 2**

## Answered Questions

### 1. Caching

**SQLite Page Cache** is automatic memory caching of database pages (4KB blocks). When you query:
- First query: Reads from disk (~1-5ms)
- Subsequent queries: Reads from memory cache (<0.1ms)
- LRU eviction when cache fills

**Performance comparison:**
- SQLite page cache: ~1ms first query, ~0.1ms cached
- Full in-memory: ~0.05ms (slightly faster)
- Trade-off: 5-10MB memory saved vs 0.05ms latency

**Recommendation:** Use SQLite page cache (default). The 0.9ms difference is negligible for web searches.

### 2. Rebuild Strategy

**Full Reindex (simpler):**
- Drop FTS5 tables, recreate, rescan all files
- Easy to implement, easy to reason about
- Takes 2-5 seconds for 1000 files (acceptable for manual trigger)

**Incremental Update (complex):**
- Track file mtimes in separate metadata table
- On each request, stat all watched files
- Update only changed files
- Takes 1-2ms per request (stat overhead)

**Recommendation:**
- **Phase 1:** Full reindex only (manual: `search.reindex()`)
- **Phase 2:** Add incremental updates (automatic mtime checks)
- **Trigger:** Reindex on HUP signal (`kill -HUP <pid>`)

**Why:** Start simple, add complexity when proven needed.

### 3. Query Syntax

**Raw FTS5 Syntax:**
```
title:hello OR content:world
"exact phrase"
prefix*
-exclude
```

**Challenges:**
- `"quotes"` for phrases (users forget)
- `-` prefix for exclusion (conflicts with hyphenated words)
- `*` suffix for prefix match (users put it at start)
- `OR` must be uppercase (confusing)

**Recommendation: Sanitize + Translate**

**User types:** `hello world`  
**Translate to:** `hello AND world` (default AND, like Google)

**User types:** `"hello world"`  
**Translate to:** `"hello world"` (preserve phrase)

**User types:** `hello -world`  
**Translate to:** `hello NOT world` (intuitive exclusion)

**What we lose:**
- Field-specific search (`title:hello`)
- Complex boolean (`(a OR b) AND c`)

**What we gain:**
- Works like Google (familiar)
- No syntax errors
- Simple for 95% of users

**Power users: Raw FTS5 syntax**

For developers who want full control:

```parsley
// Raw FTS5 query (no sanitization)
let results = search.query('title:hello OR content:world', {raw: true})
let results = search.query('"exact phrase" -exclude', {raw: true})
let results = search.query('prefix* AND (a OR b)', {raw: true})
```

**Use cases:**
- Query builder UI (user constructs FTS5 query)
- Power user interface (developers know FTS5 syntax)
- Advanced filtering needs

**Trade-off:**
- Developer responsible for syntax errors
- Can break with invalid queries
- But: Escape hatch when needed

### 4. Stemming & Internationalization

**Porter Stemmer:**
- English-only algorithm
- Reduces words to stems: "running" → "run"
- Improves recall: search "run" finds "running", "runs", "ran"
- Industry standard for English

**Unicode61 Tokenizer:**
- Unicode-aware word boundaries
- No stemming (exact word match only)
- Works for all languages (Chinese, Arabic, etc.)
- Lower recall: "running" ≠ "run"

**FTS5 Locale Support:**
- Limited - mostly tokenization (word boundaries)
- No per-language stemming (Porter is English-only)
- For other languages, need external libraries (Snowball stemmers)

**Recommendation: Start with Porter, add unicode61 option**

```parsley
let search = @SEARCH({
    watch: @./docs,
    tokenizer: "porter"  // Default: English stemming
})

let searchJP = @SEARCH({
    watch: @./docs-ja,
    tokenizer: "unicode61"  // No stemming, better for non-English
})
```

**Phase 2:** Could add Snowball multi-language stemmers if needed.

### 5. HTML Safety

**Problem:** HTML in snippets can break search results page:
```html
<p>Search found: <div class="<mark>hello</mark>"></p>  <!-- Broken! -->
```

**Solutions:**

**Option A: HTML Escape (safe but ugly)**
```
&lt;div class="hello"&gt;  <!-- Visible tags -->
```

**Option B: Strip Tags (clean)**
```
Learn about <mark>hello</mark> in this guide  <!-- No tags visible -->
```

**Option C: Store Both (best)**
- Index plain text version (stripped markdown/HTML)
- Store original for context (if needed)
- Snippets are always plain text with `<mark>` only

**Recommendation: Strip HTML tags, keep markdown text**

**For Markdown:**
1. Parse markdown → plain text
2. Index plain text
3. Generate snippets from plain text
4. Snippets contain only `<mark>` tags (safe)

**For HTML:**
1. Strip all tags → plain text
2. Index plain text
3. Snippets from plain text + `<mark>` only

**Result:** Snippets are always HTML-safe.

## Alternatives Considered

### Alternative 1: In-Memory Search

**Pros:**
- Fast reads (microseconds)
- Simple implementation

**Cons:**
- No persistence
- Cold start on restart
- Memory usage scales with content

**Decision:** Rejected. FTS5 is just as fast with better persistence.

### Alternative 2: Bleve (Pure Go Library)

**Pros:**
- More features (fuzzy, faceting)
- Pure Go (no CGo)

**Cons:**
- +10MB binary size
- More complex API
- Overkill for small sites

**Decision:** Defer to future. Start with FTS5, add Bleve if needed.

### Alternative 3: External Engine (Meilisearch, Typesense)

**Pros:**
- Best-in-class features
- Scales to millions of docs

**Cons:**
- External dependency
- Operational complexity
- Breaks single-binary philosophy

**Decision:** Recommend for large deployments, but not in core.

## Success Metrics

**Must Have:**
- Index 1000 markdown docs in <5 seconds
- Search latency <10ms (p99)
- Simple 10-line example works

**Nice to Have:**
- Auto-index on file change
- Rich metadata filtering
- <1ms search latency

**Success if:**
- Developer can add search in <30 minutes
- Works out-of-box for docs sites
- Handles 90% of small-medium site needs

## Summary

This design provides a simple, Parsley-native search solution using SQLite FTS5. It covers the 90% use case (documentation sites, blogs, internal tools) while staying true to Basil's "batteries included" philosophy. The API feels native to Parsley (path literals, familiar operators) and requires minimal configuration.

Key trade-offs:
- **Simple > Feature-rich:** Skip fuzzy search, faceting, typo tolerance
- *Design Decisions Summary

### Key Choices

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **API Style** | `@SEARCH` (Basil built-in) | Enables caching, persistence magic, dev tools |
| **Caching** | SQLite page cache only | Simple, effective, low memory |
| **Rebuild** | Full reindex (Phase 1), incremental (Phase 2) | Start simple, add complexity when needed |
| **Query Syntax** | Sanitize + translate (Google-like) | Familiar to 95% of users |
| **Stemming** | Porter (English) + unicode61 option | Cover common case, add i18n later |
| **HTML Safety** | Strip tags, plain text snippets | Safe, clean, always works |
| **Persistence** | Check mtimes on each request | Simple, reliable, 1-2ms overhead |

### Simplicity Wins

**Minimum API (works out of box):**
```parsley
let search = @SEARCH({watch: @./docs})
let results = search.query("hello world")
```

**Sophisticated defaults:**
- BM25 ranking with sensible weights
- Porter stemming for English
- HTML-safe snippets
- AND query logic (like Google)
- Auto-extract metadata from frontmatter
- Smart caching

**Developer doesn't think about:**
- Database schema
- Tokenization
- Snippet extraction
- HTML escaping
- Cache invalidation
- File watching

## Summary

This design provides a simple, Basil-native search solution using SQLite FTS5. It covers the 90% use case (documentation sites, blogs, internal tools) while staying true to Basil's "batteries included" philosophy.

**Core principles:**
- **Simple > Feature-rich:** Skip fuzzy search, faceting, typo tolerance for v1
- **Sensible defaults:** Everything works with minimal config
- **Progressive enhancement:** Start simple, add features incrementally
- **Google-like UX:** Familiar query behavior (AND by default)
- **International-ready:** Unicode61 option for non-English content
- **Safe by default:** HTML-escaped snippets, no XSS risk

**Next step:** Create FEAT spec with detailed implementation plan.

---

## FAQ

**Q: Can I use this in standalone Parsley scripts?**  
A: No, `@SEARCH` is Basil-only. It requires server infrastructure for caching and persistence.

**Q: What happens on first request?**  
A: Basil creates database, recursively scans watched folders (including all subdirectories), indexes all matching files. Takes 2-5 seconds for 1000 files. Subsequent requests use cached database.

**Q: How do I force a reindex?**  
A: Send HUP signal (`kill -HUP <pid>`) or call `search.reindex()` in code.

**Q: Can I share the database with my app?**  
A: Yes! `@SEARCH({backend: @./app.db})` creates FTS5 tables in same database. No conflicts.

**Q: Does it work for non-English content?**  
A: Yes, use `tokenizer: "unicode61"` option. No stemming, but works for all languages.

**Q: What about fuzzy search / typo tolerance?**  
A: Not in v1. If needed later, consider external engine (Meilisearch) or Bleve library.

**Q: Can I use raw FTS5 query syntax?**  
A: Yes! Use `{raw: true}` option: `search.query('title:hello OR content:world', {raw: true})`. Useful for query builders or power user interfaces.

**Q: Can I have multiple search instances?**  
A: Yes! Each `@SEARCH()` with different config creates a separate instance. Same config returns cached handle. Example: `let docs = @SEARCH({watch: @./docs})` and `let blog = @SEARCH({watch: @./blog})` are independent.

**Q: How much memory does it use?**  
A: Minimal. SQLite page cache is typically 2-10MB. Search handle cache is <1MB. Total: <20MB for 10k documents