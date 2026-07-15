---
id: man-bas-search
title: "Search"
system: basil
type: feature
name: search
created: 2026-07-12
version: 1.0.0-alpha.3
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

Full-text search over your content, powered by SQLite FTS5 — no external search service. Point it at a directory and query it.

> **Basil-only.** `@SEARCH` requires the Basil server environment.

## Quick Start

```parsley
let search = @SEARCH({
    watch: @./docs,
    path: "search.db"
})

let results = search.query(@params.q, {limit: 10})

<ul>
    for (result in results.items) {
        <li>
            <a href={"/" + result.path}>result.title</a>
            <p>result.snippet</p>
        </li>
    }
</ul>
```

On first query, Basil scans the watched folder, parses YAML frontmatter (title, tags, date), extracts headings for ranking, and builds the index. Subsequent queries hit the index.

## `@SEARCH(options)`

| Key | Type | Description |
|-----|------|-------------|
| `path` | string | SQLite index file (`":memory:"` for tests) |
| `watch` | path or array | Folder(s) to index automatically |
| `snippetLength` | int | Approximate snippet length in characters (default 200, max ~320) |
| `highlightTag` | string | HTML tag wrapped around matched terms (default `"mark"`) |

Indexes Markdown and HTML out of the box, plus text-based PDF and DOCX files (50 MB per-file limit). Unrecognised option keys are an error, so typos don't fall back to silent defaults. See the [search guide](https://github.com/sambeau/basil/blob/main/docs/guide/search.md) for `extensions`, `weights`, and `tokenizer`.

## `search.query(text, options?)`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `limit` | int | 10 | Maximum results |
| `offset` | int | 0 | Pagination offset |
| `raw` | bool | false | Pass the query straight to FTS5 (advanced) |
| `filters` | dictionary | — | Narrow results by tag or date (see below) |

Returns `{items, total}` where each item has `url`, `path`, `title`, and a highlighted `snippet`. For documents added manually with `search.add()` there is no source file, so `path` is omitted — use `url` to link to the result. `total` reflects the filtered match count.

### Filters

```parsley
let results = search.query(@params.q, {
    filters: {
        tags: ["tutorial", "guide"],  // Match any of these tags
        dateAfter: @2024-01-01,       // On or after this date (inclusive)
        dateBefore: @2024-12-31       // On or before this date (inclusive)
    }
})
```

| Key | Type | Description |
|-----|------|-------------|
| `tags` | array or string | Keep documents carrying any of the listed tags |
| `dateAfter` | date or string | Keep documents dated on or after this point |
| `dateBefore` | date or string | Keep documents dated on or before this point |

Dates accept Parsley datetime literals (`@2024-01-01`) or ISO strings (`"2024-01-01"`). Bounds are inclusive and compared in UTC. Documents without a `date` are excluded whenever a date filter is set.

## When to Use It

Good for documentation sites, blogs, wikis, and apps up to roughly 100k documents — anything where "just works, zero config" beats search-cluster features. If you need typo tolerance, multi-language stemming, or millions of documents, reach for Meilisearch or Elasticsearch instead.

## See Also

- The [search guide](https://github.com/sambeau/basil/blob/main/docs/guide/search.md) — manual indexing, custom fields, ranking, and tuning
