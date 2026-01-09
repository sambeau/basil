---
title: Full-Text Search with FTS5
tags: [search, fts5, database, features]
date: 2026-01-09
---

# Full-Text Search with FTS5

Basil includes powerful full-text search capabilities using SQLite FTS5.

## Quick Start

Create a search instance that automatically indexes markdown files:

```parsley
search = @SEARCH({watch: @./docs})
results = search.query("basil tutorial")
```

## Features

### Automatic Indexing

The search engine automatically:
- Scans watched directories for markdown files
- Parses YAML frontmatter (title, tags, date)
- Extracts headings for better ranking
- Updates the index when files change

### BM25 Ranking

Results are ranked using BM25 algorithm with configurable weights:

```parsley
search = @SEARCH({
  watch: @./docs,
  weights: {
    title: 10.0,
    headings: 5.0,
    tags: 3.0,
    content: 1.0
  }
})
```

### Query Syntax

- **Simple terms**: `basil` - searches for "basil"
- **Multiple terms**: `basil server` - both terms required (AND)
- **Quoted phrases**: `"web server"` - exact phrase match
- **Negation**: `basil -python` - has "basil" but not "python"

### Highlighted Snippets

Search results include context snippets with highlighted matches:

```html
This is a <mark>search</mark> result snippet.
```

## Performance

- Query latency: <10ms for 1000 documents
- Indexing speed: ~100 documents/second
- Incremental updates with file watching (Phase 3)
