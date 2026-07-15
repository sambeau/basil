---
id: man-bas-search
title: "Search"
system: basil
type: feature
name: search
created: 2026-07-12
version: 1.0.0-alpha.4
author: "@sam"
keywords:
  - search
  - full-text
  - fts5
  - "@SEARCH"
  - index
  - query
  - snippets
---

# Search

Full-text search over your content, powered by SQLite FTS5 — no external search service, nothing to install. Point it at a folder and start querying.

> **Basil-only.** `@SEARCH` requires the Basil server environment.

## Quick Start

```parsley
let {search, error} = @SEARCH({watch: @./docs})

let q = @params.q ?? ""
let results = search.query(q, {limit: 10})

<form method="get">
    <input type="search" name="q" value={q} placeholder="Search…"/>
</form>

<ul>
    for (result in results.items) {
        <li>
            <a href={`/{result.path}`}>result.title</a>
            <p>result.snippet</p>
        </li>
    }
</ul>
```

On the first query, Basil scans the watched folder, reads titles, tags, and dates from YAML frontmatter, extracts headings for ranking, and builds the index. Every query after that checks the folder for changes, so new, edited, and deleted files show up on their own.

## `@SEARCH(options)`

Returns `{search, error}` — the search instance, or an error message if setup failed.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `watch` | path or array | — | Folder(s) to index and keep indexed |
| `backend` | path or string | `<watch-folder>_search.db` | The SQLite index file. Use `":memory:"` for tests |
| `extensions` | array | `[".md", ".html"]` | File types to index |
| `tokenizer` | string | `"porter"` | `"porter"` stems English ("running" matches "run"); `"unicode61"` doesn't stem — better for other languages |
| `weights` | dictionary | `{title: 10.0, headings: 5.0, tags: 3.0, content: 1.0}` | Ranking boosts per field |

Markdown and HTML are indexed out of the box. Add `".docx"` or `".pdf"` to `extensions` to index Word documents and text-based PDFs too (50 MB per-file limit).

## `search.query(text, options?)`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `limit` | int | 10 | Maximum results |
| `offset` | int | 0 | Pagination offset |
| `raw` | bool | `false` | Pass the query straight to FTS5 (advanced) |
| `filters` | dictionary | — | Filter by tags: `{tags: ["tutorial"]}` |

The query syntax works like a search engine: words are ANDed together, `"quotes make phrases"`, and a `-minus` excludes a word. An empty query returns empty results, so it's safe to wire straight to a form.

Returns `{items, total, limit, offset, query}`. Each item has:

| Field | Description |
|-------|-------------|
| `path` | Path object for the source file — build links with `` `/{result.path}` `` |
| `title` | Document title (frontmatter, first heading, or filename) |
| `snippet` | Excerpt with matched terms wrapped in `<mark>…</mark>` |
| `rank` | 1-based position in the results |
| `score` | Relevance score, mostly useful for debugging |
| `date` | The document's date, when it has one |

## Beyond Watched Folders

The instance has a few more methods, all covered in [the guide](search-guide.md):

- **`add`**, **`update`**, **`remove`** — index things that aren't files: database rows, API responses, generated pages
- **`stats()`** — document count, index size, last indexed time
- **`reindex()`** — drop everything and rebuild from the watched folders

## When to Use It

Good for documentation sites, blogs, wikis, and apps up to roughly 100k documents — anything where "just works, zero config" beats search-cluster features. If you need typo tolerance, multi-language stemming, or millions of documents, reach for Meilisearch or Elasticsearch instead.

## See Also

- [The Search Guide](search-guide.md) — how indexing works file type by file type, ranking and tuning, query syntax, manual indexing, and troubleshooting
- [Routing](routing.md) — `@params` and friends
