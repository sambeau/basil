---
id: FEAT-085
title: "Full-Text Search with SQLite FTS5"
status: draft
priority: high
created: 2026-01-09
author: "@sambeau"
---

# FEAT-085: Full-Text Search with SQLite FTS5

## Summary

Add batteries-included full-text search functionality to Basil using SQLite FTS5, providing a simple, composable API for indexing and searching markdown files and dynamic content.

## Motivation

**Current state:** No built-in search capability. Developers must integrate external search engines (Elasticsearch, Meilisearch) or build custom solutions.

**Problems:**
1. External search engines add deployment complexity
2. Small-to-medium sites (100-10,000 docs) don't need distributed search
3. Documentation sites lack simple "add search in 5 minutes" solution
4. No Parsley-native API for search functionality

**Goals:**
1. **Simple**: Index documents with minimal configuration
2. **Batteries included**: Works out-of-box for 90% of use cases
3. **Parsley-native**: Feels like native language feature (path literals, expressions)
4. **Composable**: Works with existing file I/O, HTTP, database patterns
5. **No external dependencies**: Use SQLite FTS5 (already available)
6. **Performant**: <10ms search latency for thousands of documents

## Non-Goals

- Elasticsearch/Solr-level features (fuzzy search, faceting, instant search UI)
- Multi-language stemming beyond Porter (English) in v1
- Distributed search across multiple servers
- Real-time indexing with sub-second latency
- Binary file content extraction (PDF, DOCX, etc.)

## User Stories

### As a documentation site developer
- I want to add search to my markdown docs with 3 lines of code
- I want search to work automatically when files change
- I want highlighted snippets in search results
- I want to filter results by tags or date
- I don't want to configure database schemas or indexing logic

### As a blog developer
- I want to search both static markdown posts and database content
- I want to index custom fields (author, category, excerpt)
- I want to manually trigger reindexing after bulk updates
- I want separate search indexes for different content types

### As a power user
- I want to use advanced FTS5 query syntax for complex searches
- I want to build query builder UIs with field-specific searches
- I want to customize ranking weights for different fields
- I want to choose tokenizers for different languages

## Technical Design

### Basil Built-in: @SEARCH

`@SEARCH` is a Basil built-in (like `@DB`), not a Parsley library. Available only in Basil server context, not standalone Parsley scripts.

**Why built-in:**
- Automatic persistence management across requests
- Intelligent caching (per-config handle caching)
- Background file watching integration
- Dev tools integration

### Factory Function API

```parsley
let search = @SEARCH(options)
```

**Options structure:**

```parsley
{
    backend: @./search.db,           // SQLite database (required if no watch)
                                      // Use @:memory: for testing (no persistence)
    watch: @./content,                // Path or array of paths (recursive)
    extensions: [".md", ".html"],     // File types to index
    
    // Field weights for BM25 ranking
    weights: {
        title: 10.0,      // Default: 10x weight
        headings: 5.0,    // Default: 5x weight
        tags: 3.0,        // Default: 3x weight
        content: 1.0      // Default: baseline weight
    },
    
    // Snippet generation
    snippetLength: 200,               // Characters (default: 200)
    highlightTag: "mark",             // HTML tag (default: "mark")
    
    // Metadata extraction (for markdown files)
    extractTitle: true,               // From frontmatter or first H1
    extractTags: true,                // From frontmatter YAML
    extractDate: true,                // From frontmatter YAML
    
    // Tokenization
    tokenizer: "porter"               // "porter" (English) or "unicode61" (all languages)
}
```

**Default values:**

| Option | Default | Notes |
|--------|---------|-------|
| `backend` | Auto-generated if `watch` provided | `{watch_path}_search.db` or `@:memory:` for tests |
| `watch` | `null` | No auto-indexing |
| `extensions` | `[".md", ".html"]` | Common content formats |
| `weights.title` | `10.0` | Title 10x more important |
| `weights.headings` | `5.0` | Headings 5x more important |
| `weights.tags` | `3.0` | Tags 3x more important |
| `weights.content` | `1.0` | Content baseline weight |
| `snippetLength` | `200` | ~30-40 words |
| `highlightTag` | `"mark"` | HTML5 standard |
| `extractTitle` | `true` | Parse frontmatter/H1 |
| `extractTags` | `true` | Parse frontmatter |
| `extractDate` | `true` | Parse frontmatter |
| `tokenizer` | `"porter"` | English stemming |

**Minimal usage:**

```parsley
// Backend auto-created at ./docs_search.db
let search = @SEARCH({watch: @./docs})
```

### Query Method

```parsley
let results = search.query(queryString, options)
```

**Query options:**

```parsley
{
    limit: 10,                        // Results per page (default: 10)
    offset: 0,                        // Pagination offset (default: 0)
    raw: false,                       // Use raw FTS5 syntax (default: false)
    filters: {                        // Filter by metadata
        tags: ["tutorial"],           // Array: match any
        dateAfter: @2024-01-01,      // Date comparison
        dateBefore: @2024-12-31      // Date comparison
    }
}
```

**Results structure:**

```parsley
{
    query: "parsley syntax",          // Original query string
    total: 42,                        // Total matching documents
    items: [
        {
            url: "/docs/syntax",      // Document URL
            title: "Parsley Syntax Guide",
            snippet: "Learn about <mark>parsley</mark> <mark>syntax</mark>...",
            score: 0.85,              // BM25 score (0-1)
            tags: ["tutorial", "reference"],
            date: @2024-01-15         // Optional
        }
    ]
}
```

**Query behavior:**

| User Input | FTS5 Query | Behavior |
|------------|------------|----------|
| `hello world` | `hello AND world` | Both terms required (Google-like) |
| `"hello world"` | `"hello world"` | Exact phrase |
| `hello -world` | `hello NOT world` | Exclude term |
| `hello world` (raw: true) | `hello world` | Pass through to FTS5 (OR by default) |
| `title:hello` (raw: true) | `title:hello` | Field-specific search |

### Manual Indexing Methods

```parsley
// Add document
search.add({
    url: "/path",               // Required: unique identifier
    title: "Title",             // Required
    content: "...",             // Required: markdown or plain text
    tags: ["tag1", "tag2"],     // Optional
    date: @2024-01-15           // Optional
})

// Update specific fields
search.update("/path", {
    content: "Updated content",
    tags: ["updated"]
})

// Remove document
search.remove("/path")
```

### Maintenance Methods

```parsley
// Force full reindex (watched folders only)
search.reindex()

// Get index statistics
let stats = search.stats()
// Returns: {documents: 142, size: "5.2MB", last_indexed: @2024-01-09T14:30:00}
```

### Caching Behavior

**Not a singleton** - multiple search instances supported:

```parsley
// Different configs = different instances
let docsSearch = @SEARCH({watch: @./docs})
let blogSearch = @SEARCH({watch: @./blog, tokenizer: "unicode61"})

// Same config = cached handle (reused across requests)
let search1 = @SEARCH({watch: @./docs})
let search2 = @SEARCH({watch: @./docs})  // Returns same instance as search1
```

**Cache key includes:** All options (watch paths, tokenizer, weights, extensions, etc.)

**Common pattern (90% of apps):**

```parsley
// Single search for entire site
let search = @SEARCH({watch: [@./docs, @./blog, @./guides]})
```

## Database Schema

### FTS5 Virtual Table

```sql
CREATE VIRTUAL TABLE documents_fts USING fts5(
    title,           -- Weighted 10x in BM25
    headings,        -- Weighted 5x in BM25
    tags,            -- Weighted 3x in BM25
    content,         -- Weighted 1x (baseline)
    url UNINDEXED,   -- Store but don't search
    date UNINDEXED,  -- Store but don't search
    tokenize='porter'  -- or 'unicode61'
);
```

### Metadata Table (for incremental updates)

```sql
CREATE TABLE search_metadata (
    url TEXT PRIMARY KEY,
    path TEXT,                    -- Filesystem path (for watched files)
    mtime INTEGER,                -- Modification time (Unix timestamp)
    indexed_at INTEGER,           -- When indexed (Unix timestamp)
    source TEXT                   -- 'file' or 'manual'
);
```

### Query Example

```sql
SELECT 
    url,
    title,
    snippet(documents_fts, 3, '<mark>', '</mark>', '...', 32) as snippet,
    bm25(documents_fts, 10.0, 5.0, 3.0, 1.0) as score,
    date
FROM documents_fts
WHERE documents_fts MATCH ?
ORDER BY bm25(documents_fts, 10.0, 5.0, 3.0, 1.0)
LIMIT ? OFFSET ?;
```

## File Processing

### Markdown Files

**Frontmatter parsing:**

```markdown
---
title: Getting Started
tags: [tutorial, beginner]
date: 2024-01-15
---

# Getting Started

Welcome to **Basil**!

## Installation

First, install Basil...
```

**Extraction logic:**

1. Parse YAML frontmatter (between `---` delimiters)
2. Extract `title` (fallback: first H1 heading)
3. Extract `tags` (array of strings)
4. Extract `date` (ISO date or datetime)
5. Extract all headings (H1-H6) as separate field
6. Strip markdown formatting from content → plain text
7. Generate URL from file path: `./docs/guide.md` → `/docs/guide`

**HTML safety:**

- Index plain text only (stripped of HTML/markdown tags)
- Snippets contain plain text + `<mark>` tags only
- No risk of XSS or broken HTML in search results

### HTML Files

1. Strip all HTML tags → plain text
2. Try to extract `<title>` if present
3. Try to extract `<meta name="keywords">` for tags
4. Index plain text content

### File Watching

**When `watch` option is provided:**

1. **First request:**
   - Recursively scan all watched folders
   - Filter by `extensions` option
   - Parse and index all matching files
   - Store mtimes in metadata table
   - Create SQLite database if not exists

2. **Subsequent requests:**
   - Check mtimes of watched files
   - Reindex only changed/new files
   - Remove deleted files from index
   - Overhead: ~1-2ms for 1000 files

**Recursive by default:**
- `watch: @./docs` indexes `./docs/**/*.md` (all subdirectories)
- `watch: [@./docs, @./blog]` indexes both folder trees
- No depth limit in Phase 1

**Manual reindex trigger:**
- `search.reindex()` - Force full rebuild
- HUP signal: `kill -HUP $(cat basil.pid)` - Trigger reindex

## Performance Characteristics

### Query Performance

| Scenario | Latency | Documents | Notes |
|----------|---------|-----------|-------|
| Simple query (first time) | 1-5ms | 1,000 | Cold cache |
| Simple query (cached) | <0.1ms | 1,000 | SQLite page cache |
| Complex query + filters | 1-5ms | 1,000 | Multiple conditions |
| 99th percentile | <10ms | 10,000 | Target |

### Storage

- FTS5 index: ~2-3x original content size
- Example: 1000 docs × 5KB = 5MB content → 10-15MB index
- SQLite page cache: 2-10MB RAM (configurable)
- Search handle cache: <1MB per instance

### Scaling Limits

**Comfortable (recommended):**
- 100-10,000 documents
- 1-50 MB total content
- Single server
- <10ms query latency

**Reasonable (with optimization):**
- 10,000-100,000 documents
- 50-500 MB content
- May need index tuning, larger cache

**Not recommended:**
- >100,000 documents
- >500 MB content
- Consider external search engine (Meilisearch, Typesense, Elasticsearch)

## Use Cases & Examples

### Use Case 1: Documentation Site (Primary)

**Scenario:** Static markdown documentation with frontmatter

```parsley
// Simplest possible usage - zero config
let search = @SEARCH({watch: @./docs})
let results = search.query("installation")

// Render results
for (item in results.items) {
    <article>
        <h3><a href={item.url}>{item.title}</a></h3>
        <p class="snippet">{item.snippet}</p>
    </article>
}
```

**Complete search handler:**

```parsley
// search.pars - Handler for /search
let {query} = import @basil/http

let search = @SEARCH({watch: @./docs})

let q = query.q ?? ""
let page = (query.page ?? "1").toInt() ?? 1
let limit = 20
let offset = (page - 1) * limit

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
            <p>{results.total} " results for \"" {q} "\""</p>
            
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
                <p>"No results found"</p>
            }}
            
            {if (results.total > limit) {
                <nav class="pagination">
                    {if (page > 1) {
                        <a href={`?q={q}&page={page - 1}`}>"Previous"</a>
                    }}
                    <span>"Page " {page} " of " {(results.total / limit).ceil()}</span>
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

### Use Case 2: Dynamic Content (Manual Indexing)

**Scenario:** Database-driven blog posts

```parsley
// blog.pars - Blog post handler
let {db} = import @basil/db
let search = @SEARCH({backend: @./blog.db})

// When creating a post
let post = {
    title: "My New Post",
    content: markdownText,
    tags: ["tutorial", "beginner"],
    date: @now
}

// Insert into database
db <=!=> "INSERT INTO posts (title, content, tags, date) VALUES (?, ?, ?, ?)",
    [post.title, post.content, post.tags.join(","), post.date]

// Index for search
search.add({
    url: `/blog/post-{post.id}`,
    title: post.title,
    content: post.content,
    tags: post.tags,
    date: post.date
})
```

**Update handler:**

```parsley
// When updating a post
search.update(`/blog/post-{postId}`, {
    content: updatedContent,
    tags: updatedTags
})
```

**Delete handler:**

```parsley
// When deleting a post
search.remove(`/blog/post-{postId}`)
```

### Use Case 3: Mixed Static + Dynamic

**Scenario:** Static docs + dynamic user profiles

```parsley
let search = @SEARCH({
    watch: @./docs,
    backend: @./site.db
})

// Static docs auto-indexed from ./docs
// Plus manual additions for dynamic content

// Index user profile
search.add({
    url: `/user/{userId}`,
    title: `{user.name}'s Profile`,
    content: user.bio,
    tags: user.skills,
    date: user.created_at
})
```

### Use Case 4: Multi-Language Support

**Scenario:** Separate indexes for English and Japanese content

```parsley
// English content with stemming
let docsEN = @SEARCH({
    watch: @./docs/en,
    tokenizer: "porter"
})

// Japanese content without stemming
let docsJA = @SEARCH({
    watch: @./docs/ja,
    tokenizer: "unicode61"
})

// Search based on user language
let lang = query.lang ?? "en"
let search = if (lang == "ja") docsJA else docsEN
let results = search.query(query.q)
```

### Use Case 5: Advanced Query Builder

**Scenario:** Power user interface with field-specific search

```parsley
let {query} = import @basil/http
let search = @SEARCH({watch: @./docs})

// Build FTS5 query from form inputs
let field = query.field ?? "content"  // title, content, tags
let term = query.term ?? ""
let operator = query.operator ?? "AND"

// Construct raw FTS5 query
let ftsQuery = `{field}:{term}`

// Use raw mode
let results = search.query(ftsQuery, {raw: true})

// Render query builder form + results
<form method="GET">
    <select name="field">
        <option value="title" selected={field == "title"}>"Title"</option>
        <option value="content" selected={field == "content"}>"Content"</option>
        <option value="tags" selected={field == "tags"}>"Tags"</option>
    </select>
    <input type="text" name="term" value={term}/>
    <button type="submit">"Search"</button>
</form>
```

### Use Case 6: Blog Search with Filters

**Scenario:** Filter search results by tag and date

```parsley
let {query} = import @basil/http
let search = @SEARCH({watch: @./blog})

let q = query.q ?? ""
let tag = query.tag ?? null
let year = query.year ?? null

// Build filters
let filters = {}
if (tag) {
    filters.tags = [tag]
}
if (year) {
    let yearInt = year.toInt()
    filters.dateAfter = @({yearInt}-01-01)
    filters.dateBefore = @({yearInt}-12-31)
}

// Search with filters
let results = if (q != "") {
    search.query(q, {
        limit: 20,
        filters: filters
    })
} else {
    null
}

// Render search form with filter options
<form method="GET">
    <input type="search" name="q" value={q}/>
    <select name="tag">
        <option value="">"All tags"</option>
        <option value="tutorial" selected={tag == "tutorial"}>"Tutorials"</option>
        <option value="news" selected={tag == "news"}>"News"</option>
    </select>
    <select name="year">
        <option value="">"All years"</option>
        <option value="2024" selected={year == "2024"}>"2024"</option>
        <option value="2023" selected={year == "2023"}>"2023"</option>
    </select>
    <button type="submit">"Search"</button>
</form>
```

### Use Case 7: Testing with In-Memory Database

**Scenario:** Unit/integration tests that need search functionality

```parsley
// test_search.pars - Test file
let search = @SEARCH({backend: @:memory:})

// Add test documents
search.add({
    url: "/doc1",
    title: "First Document",
    content: "This is a test document about testing",
    tags: ["test"]
})

search.add({
    url: "/doc2",
    title: "Second Document",
    content: "This is another test document",
    tags: ["test", "example"]
})

// Run test queries
let results = search.query("test")
assert(results.total == 2, "Should find 2 documents")

let filtered = search.query("test", {filters: {tags: ["example"]}})
assert(filtered.total == 1, "Should find 1 document with 'example' tag")

// Memory freed when test completes - no cleanup needed
```

**Benefits for testing:**
- **Fast:** No disk I/O overhead
- **Isolated:** Each test gets clean state
- **Parallel-safe:** No file conflicts between concurrent tests
- **No cleanup:** Memory automatically freed
- **CI-friendly:** No disk space concerns

## Implementation Phases

### Phase 1: Core FTS5 Backend (MVP)

**Duration:** 5-7 days

**Components:**
- SQLite FTS5 wrapper in Go
- `@SEARCH()` factory function (Basil built-in)
- `.query()` method with basic options (limit, offset)
- Simple snippet generation using FTS5 `snippet()` function
- Query sanitization (AND by default, preserve quotes)
- Full reindex only (manual trigger via `search.reindex()`)

**Success Criteria:**
- [ ] Simplest API works: `@SEARCH({watch: @./docs}).query("hello")`
- [ ] Can index 100 markdown files
- [ ] Search returns highlighted snippets
- [ ] Query latency <10ms for 1000 documents
- [ ] Sanitized queries work like Google (AND by default)

**Test Requirements:**
- Unit tests for query sanitization
- Unit tests for FTS5 query generation
- Integration test: index files → search → verify results
- Performance test: 1000 docs in <5 seconds, query in <10ms

### Phase 2: Auto-Indexing & Metadata

**Duration:** 3-5 days

**Components:**
- Initial folder scan on first `@SEARCH()` call
- Frontmatter parsing (YAML)
- Auto-extract title (frontmatter or first H1)
- Auto-extract headings (H1-H6)
- Auto-extract metadata (tags, date)
- Search handle caching (per-config)

**Success Criteria:**
- [ ] Zero-config works for documentation sites
- [ ] Frontmatter parsing works (title, tags, date)
- [ ] Metadata filters work (tags, dateAfter, dateBefore)
- [ ] Second request uses cached handle
- [ ] Multiple watch folders work

**Test Requirements:**
- Unit tests for frontmatter parsing
- Unit tests for metadata extraction
- Integration test: frontmatter → index → filter by metadata
- Test handle caching behavior

### Phase 3: Incremental Updates

**Duration:** 5-7 days

**Components:**
- Metadata table for tracking file mtimes
- Check mtimes on each request
- Update only changed files
- Background watching (if file watcher available in Basil)
- Performance optimization (batch updates)

**Success Criteria:**
- [ ] File changes detected automatically
- [ ] Update overhead <2ms per request (1000 files)
- [ ] No full reindex needed for file changes
- [ ] Deleted files removed from index

**Test Requirements:**
- Unit tests for mtime tracking
- Integration test: modify file → automatic reindex
- Integration test: delete file → removed from index
- Performance test: mtime check overhead <2ms

### Phase 4: Manual Indexing & Advanced

**Duration:** 3-5 days

**Components:**
- `.add()`, `.remove()`, `.update()` methods
- Dynamic content support (no file path)
- Statistics (`.stats()` method)
- Dev tools integration (if available)
- Tokenizer options (porter vs unicode61)
- Raw query mode (`raw: true` option)

**Success Criteria:**
- [ ] Can index database-driven content
- [ ] Mixed static + dynamic works
- [ ] `.stats()` returns accurate info
- [ ] Raw query mode exposes FTS5 syntax
- [ ] Tokenizer option works (porter/unicode61)

**Test Requirements:**
- Unit tests for manual indexing methods
- Integration test: add/update/remove documents
- Integration test: raw query mode with FTS5 syntax
- Test tokenizer switching

## Configuration

No configuration file changes needed. `@SEARCH` is configured programmatically.

**Optional basil.yaml additions (future):**

```yaml
# Optional: Global search defaults
search:
  default_tokenizer: porter
  page_cache_size: 10MB
  max_snippet_length: 500
```

## Error Handling

### Query Errors

```parsley
// Invalid query (empty string)
let results = search.query("")
// Returns: {query: "", total: 0, items: []}

// FTS5 syntax error (raw mode only)
let {result, error} = try search.query("invalid (syntax", {raw: true})
if (error) {
    log("Query error:", error)
    // Return user-friendly error message
}
```

### File Errors

```parsley
// Watch path doesn't exist
let {result, error} = try @SEARCH({watch: @./nonexistent})
// error = "Watch path does not exist: ./nonexistent"

// Invalid YAML frontmatter
// Behavior: Log warning, skip frontmatter, use first H1 as title
```

### Database Errors

```parsley
// Database locked (rare with SQLite)
let {result, error} = try search.query("hello")
// Retry logic built-in (SQLite default: 5 seconds)

// Disk full
let {result, error} = try search.add({url: "/doc", title: "Doc", content: "..."})
// error = "Database write failed: disk full"
```

## Security Considerations

### XSS Prevention

- **Snippets are HTML-safe:** Plain text + `<mark>` tags only
- **No user HTML in snippets:** Strip all tags during indexing
- **No code injection:** FTS5 queries are parameterized

### Query Sanitization

- **Default mode (sanitized):** Convert to safe AND queries
- **Raw mode (advanced):** Developer responsibility to validate
- **No SQL injection:** All queries parameterized

### Rate Limiting

Not built into `@SEARCH`. Recommend using Basil's built-in rate limiting for search endpoints:

```parsley
// basil.yaml
rate_limit:
  enabled: true
  rules:
    - path: /search
      limit: 100
      window: 1m
```

### File System Access

- **Watch paths:** Only readable paths (no privilege escalation)
- **Database:** Configurable location (default: project directory)
- **No arbitrary file access:** Paths validated

## Migration & Compatibility

### Breaking Changes

None. This is a new feature.

### Backward Compatibility

Fully backward compatible. Opt-in feature.

### Database Migrations

Automatic schema creation on first use. No migrations needed.

### Upgrade Path

1. Add `@SEARCH()` call to handler
2. First request creates database and indexes files
3. No downtime required

## Testing Strategy

### Unit Tests

1. **Query sanitization:** `"hello world"` → `"hello AND world"`
2. **Frontmatter parsing:** YAML → dict
3. **Metadata extraction:** Markdown → title, tags, date
4. **FTS5 query generation:** Options → SQL
5. **Snippet generation:** Content + query → highlighted snippet
6. **Mtime tracking:** File changes → update detection

### Integration Tests

1. **End-to-end:** Index files → search → verify results
2. **Pagination:** Query with limit/offset → correct pages
3. **Filters:** Search with tags/date filters → correct subset
4. **Multiple instances:** Two `@SEARCH()` calls → independent indexes
5. **Caching:** Same config → reused handle
6. **Manual indexing:** Add/update/remove → index updated
7. **Raw mode:** FTS5 syntax → correct results

### Performance Tests

1. **Indexing:** 1000 files indexed in <5 seconds
2. **Query latency:** <10ms p99 for 1000 documents
3. **Mtime check overhead:** <2ms for 1000 files
4. **Memory usage:** <20MB for 10,000 documents

### Manual Testing

1. **Documentation site:** Add search to Basil's own docs
2. **Blog with filters:** Test tag/date filtering
3. **Multi-language:** Test porter vs unicode61 tokenizers
4. **Query builder:** Test raw mode with field-specific queries
5. **Large corpus:** Test with 10,000+ documents

## Documentation Requirements

### User Guide

1. **Quick start:** 5-minute tutorial (add search to docs site)
2. **Use cases:** Documentation, blog, mixed content
3. **API reference:** Factory, methods, options
4. **Query syntax:** Sanitized vs raw mode
5. **Performance tuning:** When to use external search
6. **Troubleshooting:** Common issues and solutions

### Developer Guide

1. **Architecture:** FTS5 integration, caching strategy
2. **Database schema:** Tables, indexes, queries
3. **File processing:** Frontmatter, metadata extraction
4. **Testing:** How to write tests for search features

### Examples

1. **Minimal example:** 3 lines of code
2. **Complete search page:** Full handler with pagination
3. **Blog with filters:** Tag and date filtering
4. **Multi-language:** Separate indexes for different languages
5. **Query builder:** Advanced UI with raw FTS5 queries

### FAQ

1. **Q:** Can I use this in standalone Parsley?
   **A:** No, `@SEARCH` is Basil-only (server infrastructure).

2. **Q:** What happens on first request?
   **A:** Database created, folders scanned, files indexed (2-5s for 1000 files).

3. **Q:** How do I force reindex?
   **A:** `search.reindex()` or HUP signal.

4. **Q:** Can I share database with my app?
   **A:** Yes! FTS5 tables are namespaced.

5. **Q:** Does it work for non-English?
   **A:** Yes, use `tokenizer: "unicode61"` option.

6. **Q:** Can I have multiple search instances?
   **A:** Yes! Different configs = separate instances.

7. **Q:** How much memory does it use?
   **A:** Minimal. <20MB for 10,000 documents.

8. **Q:** When should I use external search?
   **A:** >100,000 documents or need fuzzy search/faceting.

9. **Q:** Can I use `:memory:` databases for testing?
   **A:** Yes! `@SEARCH({backend: @:memory:})` is perfect for unit/integration tests. Fast, isolated, no cleanup needed. Not for production (loses data on restart).

## Success Metrics

### Must Have (P0)

- [ ] Index 1000 markdown docs in <5 seconds
- [ ] Search latency <10ms (p99)
- [ ] Simple 3-line example works
- [ ] Highlighted snippets in results
- [ ] Query sanitization works (Google-like)

### Should Have (P1)

- [ ] Frontmatter parsing (title, tags, date)
- [ ] Metadata filtering (tags, dates)
- [ ] Auto-indexing on file changes (<2ms overhead)
- [ ] Multiple watch folders
- [ ] Handle caching (per-config)

### Nice to Have (P2)

- [ ] Raw query mode for power users
- [ ] Multiple tokenizers (porter, unicode61)
- [ ] Dev tools integration
- [ ] Statistics method
- [ ] <1ms search latency (cached)

## Related Work

- **Design document:** [search-design.md](../design/search-design.md)
- **SQLite FTS5:** https://www.sqlite.org/fts5.html
- **Porter Stemmer:** https://tartarus.org/martin/PorterStemmer/
- **BM25 Ranking:** https://en.wikipedia.org/wiki/Okapi_BM25

## Open Questions

1. **Background watching:** Should we use Basil's file watcher or poll mtimes?
   - **Recommendation:** Start with mtime polling (simpler), add watcher in Phase 3

2. **Database location:** Should we allow `:memory:` databases?
   - **Recommendation:** Yes (extremely useful for testing, explicitly document as test-only)

3. **Binary content:** Should we support PDF/DOCX extraction in future?
   - **Recommendation:** Defer to future feature (adds dependency complexity)

4. **Search-as-you-type:** Should we support instant search UI?
   - **Recommendation:** Defer to future feature (needs WebSocket/SSE)

5. **Pagination helpers:** Should we provide pagination components?
   - **Recommendation:** Defer to `@std/html` library (not search-specific)

## Approval

**Status:** Approved - Ready for implementation

**Reviewers:**
- [x] @sambeau (author)

**Sign-off:**
- [x] Design approved
- [ ] Implementation plan approved
- [ ] Test strategy approved
- [ ] Documentation plan approved
