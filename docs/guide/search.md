# Full-Text Search

> **⚠️ Superseded.** This page has been replaced by the source-verified docs at [docs/basil/manual/search.md](../basil/manual/search.md) and [docs/basil/manual/search-guide.md](../basil/manual/search-guide.md). Some details below may be out of date.

Basil includes built-in full-text search powered by SQLite FTS5. No external search engines needed—just add a few lines of code.

## Quick Start

### 1. Create a Search Instance

In your Parsley handler:

```parsley
let {search, error} = @SEARCH({
  watch: @./docs,
  path: "search.db"
})
```

`@SEARCH` returns a `{search, error}` pair: the search instance, or an error message
if setup failed. Everything below calls methods on that `search` value.

That's all the setup. The search engine automatically:
- Scan your `docs` folder for markdown files
- Parse YAML frontmatter (title, tags, date)
- Extract headings for better ranking
- Build a searchable index on first query

### 2. Query the Index

```parsley
let results = search.query("hello world", {
  limit: 10,
  offset: 0
})

<ul>
  for (result in results.items) {
    <li>
      <a href={`/{result.path}`}>result.title</a>
      <p>result.snippet</p>
    </li>
  }
</ul>
```

### 3. Run and Test

```bash
./basil
```

Visit your handler and run a search. The index builds automatically on the first query.

## When to Use @SEARCH

**Good for:**
- Documentation sites (100-10,000 pages)
- Blogs and content sites
- Internal tools and wikis
- Apps with <100K documents
- Sites that want zero-config search

**Not for:**
- Real-time instant search (>1M documents)
- Multi-language stemming (beyond English)
- Fuzzy/typo-tolerant search
- Distributed search across servers
- Scanned/image-based PDFs (no OCR — only text-based PDFs are supported)

For those needs, use Meilisearch or Elasticsearch instead.

## Configuration Options

### Factory Function: @SEARCH()

```parsley
let {search, error} = @SEARCH({
  // Required if no watch paths
  path: "search.db",           // SQLite database file
                                // Use ":memory:" for tests
  
  // Optional: Auto-indexing
  watch: @./docs,               // Single path, or an array: [@./docs, @./blog]
  extensions: [".md", ".html"], // File types (default: [".md", ".html"])
  
  // Optional: Ranking weights
  weights: {
    title: 10.0,      // Title field weight (default: 10.0)
    headings: 5.0,    // Headings weight (default: 5.0)
    tags: 3.0,        // Tags weight (default: 3.0)
    content: 1.0      // Content baseline (default: 1.0)
  },
  
  // Optional: Snippet generation
  snippetLength: 200,           // Characters (default: 200)
  highlightTag: "mark",         // HTML tag (default: "mark")
  
  // Optional: Tokenizer
  tokenizer: "porter",          // "porter" (English) or "unicode61"
                                // Default: "porter"
})
```

### Configuration Details

**path** — SQLite database file:
- Relative to project root: `"search.db"`
- Absolute path: `"/var/data/search.db"`
- In-memory testing: `":memory:"`
- If `watch` is set, defaults to `"<first-watch-dir>_search.db"` next to the watched folder

**watch** — Auto-indexed directories:
- Scanned recursively for matching extensions
- Checked for updates on each query (low overhead: <2ms)
- Can be a single path or array of paths
- Supports `@./relative` or absolute paths

**extensions** — File types to index:
- Default: `[".md", ".html"]`
- Common: `[".md", ".html", ".txt", ".docx", ".pdf"]`
- DOCX support extracts text, headings, and metadata (title, keywords, dates)
- PDF support extracts plain text (text-based PDFs only, no OCR for scanned documents)
- Case-insensitive matching

**weights** — Field importance for ranking:
- Higher weight = more important in results
- Title gets 10x boost by default
- Adjust based on your content structure

**snippetLength** — Characters in result snippets:
- Default: 200 characters
- Approximate — truncated to word boundaries
- Maximum ~320 characters (FTS5's 64-word snippet limit)
- Includes matched terms with highlighting

**highlightTag** — HTML element for highlights:
- Default: `"mark"` (renders as `<mark>term</mark>`)
- Use `"em"` or `"strong"` for alternatives
- Styled with CSS in your template

**tokenizer** — Text processing:
- `"porter"`: English stemming ("running" → "run")
- `"unicode61"`: No stemming, better for non-English
- Affects search behavior and index size

## API Reference

### .query(searchTerm, options)

Execute a search query.

```parsley
let results = search.query("hello world", {
  limit: 10,       // Results per page (default: 10)
  offset: 0,       // Skip N results (default: 0)
  raw: false,      // Pass query directly to FTS5? (default: false)
  filters: {
    tags: ["tutorial", "guide"],     // Match any of these tags
    dateAfter: @2024-01-01,          // On or after this date (inclusive)
    dateBefore: @2024-12-31          // On or before this date (inclusive)
  }
})
```

**Filter behavior:**
- `tags` → Keep documents carrying any of the listed tags (array or single string).
  Matching is substring-based, so a filter for `"art"` also matches documents tagged
  `"articles"` — keep tag names distinct
- `dateAfter` / `dateBefore` → Inclusive bounds, compared in UTC. Accept datetime literals (`@2024-01-01`) or ISO strings (`"2024-01-01"`)
- Documents without a `date` are excluded whenever a date filter is set
- `total` reflects the filtered match count, so pagination math stays correct

**Query behavior:**
- `hello world` → Both words must appear (AND)
- `"hello world"` → Exact phrase
- `hello -world` → "hello" but not "world" (NOT)
- Empty string → Returns empty results
- `raw: true` → Use FTS5 syntax directly (advanced)

**Returns:**
```parsley
{
  items: [
    {
      url: "/docs/getting-started",      // Unique identifier (always present)
      path: @./docs/getting-started.md,  // Path object to source file
      title: "Getting Started",
      snippet: "...learn how to <mark>hello</mark> <mark>world</mark>...",
      highlight: "<mark>hello</mark> <mark>world</mark>",
      score: 3.14,
      rank: 1,
      date: @2024-01-15
    }
  ],
  total: 42,
  query: "hello world",
  limit: 10,
  offset: 0
}
```

**Note:** `path` is a path object, so you can use path methods like `.basename()`, `.extension()`, `.parent()` etc. To create a URL from the path, use string interpolation: `` `/{result.path}` ``

**Note:** Documents added manually with `.add()` have no source file, so `path` is omitted for them — use `url` (the identifier you passed to `.add()`) to link to the result.

**Pagination:**
```parsley
// Page 1 (results 1-10)
let page1 = search.query("hello", {limit: 10, offset: 0})

// Page 2 (results 11-20)
let page2 = search.query("hello", {limit: 10, offset: 10})

// Calculate pages
let totalPages = (page1.total + 9) / 10  // Round up
```

### .add(document)

Manually index a document.

```parsley
search.add({
  url: "/blog/my-post",             // Required: unique identifier
  title: "My Post Title",           // Required: document title
  content: "Full text content...",  // Required: searchable content
  date: "2024-01-15",               // Optional: date for filtering
  tags: ["tutorial", "parsley"],    // Optional: tags for filtering
  headings: "Intro,Setup,Usage"     // Optional: comma-separated headings
})
```

**When to use:**
- Database-driven content (posts, comments, products)
- API-fetched content
- Dynamically generated pages
- Non-file content

**Requirements:**
- `url` must be unique (overwrites existing)
- `url`, `title`, `content`, `date` and `headings` must be strings — convert dates to
  ISO format (`"2024-01-15"`), and pass strings rather than path or datetime literals
- `tags` must be an array of strings

### .update(document)

Replace an existing document.

```parsley
search.update({
  url: "/blog/my-post",             // Required: identifies the document
  title: "Updated Title",           // Required
  content: "Updated text...",       // Required
  tags: ["updated", "revised"]
})
```

**Behavior:**
- Takes one dictionary — the same shape `.add()` takes, not a `(url, fields)` pair
- Replaces the document wholesale: it removes the old entry and re-adds it, so any
  field you leave out is dropped, not preserved
- If the document doesn't exist, it's created
- Because `.add()` already overwrites on a matching `url`, `.add()` is usually all you need

### .remove(url)

Remove a document from the index.

```parsley
search.remove("/blog/old-post")
```

`.remove()` takes a **string**, not a path literal.

**Behavior:**
- Deletes document by URL
- Idempotent (safe to call multiple times)
- No error if document doesn't exist

### .stats()

Get index statistics.

```parsley
let stats = search.stats()
// Returns: {
//   documents: 142,
//   size: "5.2MB",
//   last_indexed: "2024-01-09T14:30:00Z"
// }
```

`last_indexed` is a string, not a datetime value.

**Use cases:**
- Display "Searching 142 documents" message
- Monitor index growth
- Debug indexing issues

### .reindex()

Force a full reindex of watched folders.

```parsley
search.reindex()
```

**When to use:**
- After bulk file changes
- Index corruption recovery
- Testing index behavior

**Requirements:**
- Only works with `watch` configured
- Returns error if no watch paths set
- Drops and rebuilds entire index
- Re-scans all watched directories

**Note:** Normally not needed—index updates automatically on each query. Use only for manual control or troubleshooting.

## Common Patterns

### Documentation Site Search

**Scenario:** Search across markdown documentation.

```parsley
let {search, error} = @SEARCH({
  watch: @./docs,
  path: "search.db"
})

// In your search page — @params merges query string and form input
let query = @params.q ?? ""
let results = search.query(query, {limit: 20})

<form method="get">
  <input type="search" name="q" value={query} placeholder="Search docs..."/>
</form>

if (results.total > 0) {
  <p>`Found {results.total} results`</p>
  for (result in results.items) {
    <article>
      <h3><a href={`/{result.path}`}>result.title</a></h3>
      <p>result.snippet</p>
    </article>
  }
} else {
  <p>"No results found. Try different keywords."</p>
}
```

### Blog with Tag Filtering

**Scenario:** Search blog posts, filter by tags and date.

```parsley
let {search, error} = @SEARCH({
  watch: @./posts,
  path: "blog.db",
  weights: {
    title: 15.0,    // Boost titles more
    content: 1.0
  }
})

// Filter by tag from URL
let tag = @params.tag ?? ""
let query = @params.q ?? ""

var filters = {}
if (tag) {
  filters.tags = [tag]
}

let results = search.query(query, {
  limit: 10,
  offset: 0,
  filters: filters
})

<form method="get">
  <input type="search" name="q" value={query}/>
  <select name="tag">
    <option value="">"All tags"</option>
    <option value="tutorial">"Tutorials"</option>
    <option value="news">"News"</option>
  </select>
  <button>"Search"</button>
</form>
```

### Mixed Static + Dynamic Content

**Scenario:** Search both markdown files and database records.

```parsley
// Auto-index markdown files
let {search, error} = @SEARCH({
  watch: @./docs,
  path: "search.db"
})

// Manually index database posts
let db = @sqlite("./app.db")
let posts = db <=??=> "SELECT * FROM posts WHERE published = 1"

for (post in posts) {
  search.add({
    url: "/blog/" + post.slug,
    title: post.title,
    content: post.body,
    date: post.published_at,   // must be an ISO date string
    tags: post.tags.split(",")
  })
}

// Search everything together
let results = search.query((@params.q ?? ""), {limit: 10})
```

### Multiple Search Indexes

**Scenario:** Separate indexes for different content types.

Parsley's destructuring can't rename keys, so bind each `@SEARCH` result to its own
name and reach through `.search`:

```parsley
let {request} = import @basil/http

// Documentation search
let docsIndex = @SEARCH({
  watch: @./docs,
  path: "docs.db",
  tokenizer: "porter"
})

// Blog search (different tokenizer)
let blogIndex = @SEARCH({
  watch: @./blog,
  path: "blog.db",
  tokenizer: "unicode61"  // Better for international content
})

// Product search (manual indexing)
let productIndex = @SEARCH({
  path: "products.db"
})

let db = @sqlite("./app.db")
let products = db <=??=> "SELECT * FROM products WHERE active = 1"
for (product in products) {
  productIndex.search.add({
    url: "/products/" + product.id,
    title: product.name,
    content: product.description,
    tags: [product.category]
  })
}

// Use appropriate search based on context
let q = @params.q ?? ""
let section = request.path.split("/")[1]

var results = null
if (section == "docs") {
  results = docsIndex.search.query(q)
} else if (section == "blog") {
  results = blogIndex.search.query(q)
} else {
  results = productIndex.search.query(q)
}
```

### Search Results Page Template

**Scenario:** A complete search UI.

```parsley
let {search, error} = @SEARCH({watch: @./content, path: "search.db"})

let query = @params.q ?? ""
let page = toInt((@params.page ?? "1"))
let perPage = 20

let results = search.query(query, {
  limit: perPage,
  offset: (page - 1) * perPage
})

let stats = search.stats()
let totalPages = (results.total + perPage - 1) / perPage

<html>
<head>
  <title>if (query) {"Search: " + query} else {"Search"}</title>
  <style>
    body { max-width: 800px; margin: 40px auto; font-family: sans-serif; }
    .search-box input { width: 100%; padding: 12px; font-size: 16px; }
    .stats { color: #666; margin: 20px 0; font-size: 14px; }
    .result { border-bottom: 1px solid #eee; padding: 20px 0; }
    .result h3 { margin: 0 0 5px 0; }
    .result h3 a { color: #1a0dab; text-decoration: none; }
    .result .snippet { color: #545454; line-height: 1.5; }
    .pagination { margin: 30px 0; text-align: center; }
    .pagination a { padding: 8px 12px; margin: 0 4px; border: 1px solid #ddd; }
    .pagination a.active { background: #1a0dab; color: white; border-color: #1a0dab; }
    mark { background: #ffeb3b; padding: 0 2px; font-weight: bold; }
  </style>
</head>
<body>
  <h1>"Search"</h1>
  
  <div class="search-box">
    <form method="get">
      <input type="search" name="q" value={query} 
             placeholder={`Search {stats.documents} documents...`} autofocus/>
    </form>
  </div>

  if (query) {
    <div class="stats">
      `Found {results.total} results`
    </div>

    if (results.total > 0) {
      for (result in results.items) {
        <div class="result">
          <h3><a href={`/{result.path}`}>result.title</a></h3>
          <div class="url" style="color: #006621; font-size: 14px;">`/{result.path}`</div>
          <p class="snippet">result.snippet</p>
          if (result.date) {
            <div style="color: #999; font-size: 12px;">result.date</div>
          }
        </div>
      }

      if (totalPages > 1) {
        <div class="pagination">
          if (page > 1) {
            <a href={`?q={query}&page={page - 1}`}>"← Previous"</a>
          }
          
          for (i in 1..totalPages) {
            if (i == page) {
              <a class="active" href={`?q={query}&page={i}`}>i</a>
            } else {
              <a href={`?q={query}&page={i}`}>i</a>
            }
          }
          
          if (page < totalPages) {
            <a href={`?q={query}&page={page + 1}`}>"Next →"</a>
          }
        </div>
      }
    } else {
      <p>`No results found for "{query}". Try different keywords.`</p>
    }
  } else {
    <p>"Enter a search query above to find content."</p>
  }
</body>
</html>
```

## Advanced Topics

### Custom Ranking Weights

Adjust field weights based on your content structure:

```parsley
// Documentation site: boost headings
let docsIndex = @SEARCH({
  watch: @./docs,
  weights: {
    title: 10.0,
    headings: 8.0,    // Higher than default
    tags: 2.0,
    content: 1.0
  }
})

// Blog: boost tags for discovery
let blogIndex = @SEARCH({
  watch: @./blog,
  weights: {
    title: 15.0,
    headings: 3.0,
    tags: 7.0,        // Higher than default
    content: 1.0
  }
})
```

**Tuning tips:**
- Start with defaults
- Run test queries and note what ranks poorly
- Adjust weights incrementally (1-2x changes)
- Higher ratio = more impact (10:1 vs 3:1)

### Raw FTS5 Queries

For power users who need advanced FTS5 features:

```parsley
// Boolean operators
let boolResults = search.query("parsley OR basil", {raw: true})

// Phrase with proximity
let nearResults = search.query('"web server" NEAR/5 parsley', {raw: true})

// Field-specific search
let fieldResults = search.query("title:tutorial content:advanced", {raw: true})

// Column filters (requires FTS5 knowledge)
let columnResults = search.query("{title}: tutorial", {raw: true})
```

**Warning:** Raw queries bypass safety features:
- No automatic AND logic
- No query sanitization
- Can cause SQL errors if malformed
- Use only for trusted/validated input

**FTS5 documentation:** https://www.sqlite.org/fts5.html

### Performance Tuning

**Index size:**
- Expect 2-3x content size on disk
- 1000 markdown files (~1MB each) → ~3GB index
- Use `stats.size` to monitor growth

**Query speed:**
- Simple queries: <5ms typical
- Complex queries with filters: <10ms
- First query (cold cache): <10ms
- Cached queries: <1ms

**Optimization tips:**

```parsley
// 1. Use pagination (don't fetch everything)
let good = search.query(query, {limit: 20})   // Good
let bad = search.query(query, {limit: 1000})  // Bad

// 2. Cache instances (automatic with same config)
let cached = @SEARCH({watch: @./docs})  // Reuses connection

// 3. Limit snippet length for faster generation
let shortSnippets = @SEARCH({
  watch: @./docs,
  snippetLength: 150  // Shorter = faster
})

// 4. Use filters to narrow results
let filtered = search.query(query, {
  limit: 10,
  filters: {tags: ["tutorial"]}  // Faster than scanning all results
})
```

**Watch folder overhead:**
- Mtime check per file: ~1-2ms total
- Happens on each query
- Negligible for <10,000 files
- Consider manual indexing for 100,000+ files

### Multi-Language Support

**English (default):**
```parsley
let {search, error} = @SEARCH({
  watch: @./docs,
  tokenizer: "porter"  // Stems words: "running" → "run"
})
```

**Other languages:**
```parsley
let {search, error} = @SEARCH({
  watch: @./docs,
  tokenizer: "unicode61"  // No stemming, better for non-English
})
```

**Multiple languages:**
```parsley
// Separate indexes per language
let enIndex = @SEARCH({watch: @./docs/en, tokenizer: "porter"})
let esIndex = @SEARCH({watch: @./docs/es, tokenizer: "unicode61"})
let frIndex = @SEARCH({watch: @./docs/fr, tokenizer: "unicode61"})

// Route based on language
let lang = @params.lang ?? "en"
let q = @params.q ?? ""

var results = null
if (lang == "es") {
  results = esIndex.search.query(q)
} else if (lang == "fr") {
  results = frIndex.search.query(q)
} else {
  results = enIndex.search.query(q)
}
```

### Testing with In-Memory Database

For tests, use `:memory:` to avoid file I/O:

```parsley
let {search, error} = @SEARCH({
  path: ":memory:",
  tokenizer: "porter"
})

// Manually add test documents
search.add({
  url: "/test/doc1",
  title: "Test Document",
  content: "Test content here"
})

let results = search.query("test")
// Test assertions...
```

**Benefits:**
- No file cleanup needed
- Faster test execution
- Isolated per-test instance

## Troubleshooting

### Documents not appearing in search

**Check 1:** Is the file extension included?
```parsley
let {search, error} = @SEARCH({
  watch: @./docs,
  extensions: [".md", ".html", ".txt"]  // Defaults are only .md and .html
})
```

**Check 2:** Is frontmatter valid YAML?
```markdown
---
title: My Document
tags: [tutorial]  # Must be valid YAML array
---
```

**Check 3:** Run stats to verify indexing:
```parsley
let stats = search.stats()
// Should show expected document count
```

**Check 4:** Force reindex:
```parsley
search.reindex()  // Drops and rebuilds index
```

### Search returns no results

**Check 1:** Verify query syntax:
```parsley
let bothWords = search.query("hello world")  // Both words must appear
let oneWord = search.query("hello")          // Try single term
```

**Check 2:** Check if documents exist:
```parsley
let stats = search.stats()
// If documents = 0, indexing isn't working
```

**Check 3:** Try raw query for debugging:
```parsley
let results = search.query("hello", {raw: true})
```

### Slow query performance

**Check 1:** Are you fetching too many results?
```parsley
let results = search.query(query, {limit: 20})  // Not 1000
```

**Check 2:** Is the database file huge?
```parsley
let stats = search.stats()
// If size > 10GB, consider splitting indexes
```

**Check 3:** Reduce snippet length:
```parsley
let {search, error} = @SEARCH({
  watch: @./docs,
  snippetLength: 100  // Default is 200
})
```

### Index not updating after file changes

**Automatic updates:** Index checks mtimes on each query.

**Force update:**
```parsley
search.reindex()
```

**Check last indexed time:**
```parsley
let stats = search.stats()
// stats.last_indexed shows when index was built
```

### "watch paths required for reindex" error

**Problem:** Calling `.reindex()` on manual-only index.

**Solution:** Add watch paths:
```parsley
let {search, error} = @SEARCH({
  path: "search.db",
  watch: @./docs  // Required for .reindex()
})
```

**Or:** Don't call `.reindex()` for manual indexes. Use `.add()`, `.update()`, `.remove()` to manage documents manually.

### Syntax errors in queries

**Problem:** Special characters breaking queries.

**Solution:** Use default query processing (not raw):
```parsley
// Good: Auto-sanitized
let safe = search.query((@params.q ?? ""))

// Bad: Can break on special chars
let unsafe = search.query((@params.q ?? ""), {raw: true})
```

**Or:** Validate user input:
```parsley
var query = @params.q ?? ""
if (query.length() > 100) {
  query = query.truncate(100)  // Limit length
}
let results = search.query(query)
```

## FAQ

**Q: Can I search multiple fields separately?**  
A: Not directly. Use raw queries with FTS5 column syntax: `{title}: hello {content}: world`

**Q: Does search work with non-markdown files?**  
A: Yes! Add extensions: `extensions: [".md", ".html", ".txt", ".docx"]`. DOCX files extract text content, headings, and metadata (title, keywords). Frontmatter parsing only applies to markdown.

**Q: Can I exclude certain files from indexing?**  
A: Not yet. Workaround: use separate watch folders or manual `.add()` with filtering.

**Q: How do I implement autocomplete/instant search?**  
A: Query on keypress with debouncing. Keep queries short and use `limit: 5` for suggestions.

**Q: Can I export/import the index?**  
A: The index is a SQLite file. Copy `search.db` to backup/restore.

**Q: Does it support fuzzy search (typos)?**  
A: No. FTS5 requires exact token matches (with stemming). Use external search engines for fuzzy matching.

**Q: Can I boost recent documents?**  
A: Not automatically. Workaround: Add recency scoring in your handler after query.

**Q: How do I search across multiple sites?**  
A: Create separate indexes per site, then merge results in your handler.

**Q: Is there a query size limit?**  
A: SQLite limits strings to 1MB. Practically, queries should be <1000 characters.

**Q: Can I use this for production?**  
A: Yes — suitable for sites with <100K documents. Tested with thousands of files, <10ms query times.

## Examples

See working examples in `examples/search/`:
- `index.pars` — Full search page with UI
- `docs/*.md` — Sample documents with frontmatter
- `README.md` — Setup instructions

Run the example:
```bash
./basil examples/search
```

## Related Features

- [Database Access](/docs/guide/api-table-binding.md) — `@DB` for dynamic content
- [Query DSL](/docs/guide/query-dsl.md) — URL query parameters
- [Parts](/docs/basil/manual/parts-guide.md) — Reusable components for search UI

## Further Reading

- [FEAT-085 Specification](/docs/specs/FEAT-085.md) — Technical design details
- [SQLite FTS5 Documentation](https://www.sqlite.org/fts5.html) — Underlying search engine
- [Porter Stemming Algorithm](https://tartarus.org/martin/PorterStemmer/) — How tokenizer works
