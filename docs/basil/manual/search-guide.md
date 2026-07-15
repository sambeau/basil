---
id: man-bas-search-guide
title: "The Search Guide"
system: basil
type: guide
name: search-guide
created: 2026-07-14
version: 1.0.0-alpha.4
author: "@sam"
keywords:
  - search
  - guide
  - fts5
  - indexing
  - frontmatter
  - ranking
  - weights
  - tokenizer
  - snippets
  - pdf
  - docx
  - add
  - reindex
  - stats
---

# The Search Guide

The [Search page](search.md) covers the essentials: `@SEARCH`, watched folders, and `query()`. This guide is the deep dive — what actually gets indexed, how ranking works, the full query syntax, and how to index content that isn't a file.

## How Indexing Works

`@SEARCH` is lazy in the best way. Creating an instance costs almost nothing; the work happens on the first query:

1. The watched folders are scanned recursively for files matching `extensions`.
2. Each file is parsed — frontmatter, headings, text — and added to the index.
3. The index is written to the SQLite file named by `path`.

Every query after that re-checks the watched folders and compares file modification times against the index. New files are added, changed files are re-indexed, and deleted files are removed — no restarts, no build step. For a folder of hundreds or a few thousand documents this check is cheap; if you're watching truly huge trees, an occasional [`reindex()`](#stats-and-reindex) and less frequent querying will serve you better.

Some scanning details worth knowing:

- Hidden files and dot-directories (`.git`, `.obsidian`, …) are skipped.
- Extension matching is case-insensitive (`.MD` works).
- Files over 50 MB are skipped.
- The index persists — restart the server and it picks up where it left off.

## What Gets Indexed

Each document gets four searchable fields — **title**, **headings**, **tags**, and **content** — which is what the ranking weights refer to. How those fields are filled depends on the file type.

### Markdown (`.md`)

The richest source. YAML frontmatter is parsed for metadata:

```markdown
---
title: Growing Basil Indoors
tags: [herbs, indoor, beginner]
date: 2026-03-01
---

# Growing Basil Indoors

Basil likes light. A lot of light...
```

- **title** — frontmatter `title`, or the first `# H1`, or the filename ("growing-basil.md" becomes "Growing Basil").
- **tags** — frontmatter `tags`, as a YAML array or a comma-separated string.
- **date** — frontmatter `date` (`2026-03-01`, with a time, or full RFC 3339).
- **headings** — every `#` through `######` heading, indexed separately so heading matches can rank higher.
- **content** — the text with markdown formatting stripped. Code blocks are indexed too, so searching for a function name works.

### HTML (`.html`)

Indexed as plain text with the tags stripped. HTML files have no frontmatter, so the title falls back to the filename — name them well.

### Word (`.docx`)

Text is extracted from paragraphs *and* tables. The title comes from the document's properties, then the first heading-styled paragraph, then the filename. Document keywords become tags, and the modified/created date becomes the document date.

### PDF (`.pdf`)

Plain text extraction, title from the filename. **Text-based PDFs only** — there's no OCR, so a scanned document indexes as roughly nothing.

## The Index File

The index is an ordinary SQLite file. With `watch: @./docs` and no `path`, it lands next to the watched folder as `docs_search.db`. Things to know:

- **Ignore it in git.** It's a build artifact; add `*_search.db` to your `.gitignore`.
- **Delete it any time.** The next query rebuilds it from the watched folders.
- **Copy it to back it up** — it's just a file. (You may see `-wal` and `-shm` companions while the server runs; that's SQLite's write-ahead log.)
- **Use `":memory:"` in tests** — nothing touches disk, and every test run starts clean.
- **Instances are cached.** Two handlers calling `@SEARCH` with the same options share one instance and one database connection.
- **Changing `tokenizer` needs a rebuild.** The index keeps the tokenizer it was created with, so call `reindex()` (or delete the file) after switching.

## Ranking

Results are ranked with BM25 — the same family of algorithm most search engines use — with a twist: matches in different fields count differently. The defaults:

```parsley
weights: {
    title: 10.0,      // a title match is worth 10 content matches
    headings: 5.0,
    tags: 3.0,
    content: 1.0
}
```

Tune them to your content. A documentation site might boost headings; a blog might boost tags:

```parsley
let {search, error} = @SEARCH({
    watch: @./posts,
    weights: {title: 15.0, headings: 3.0, tags: 7.0, content: 1.0}
})
```

Start with the defaults, run the searches your visitors actually run, and nudge weights when something ranks oddly. Big ratios have big effects; change one thing at a time.

Each result carries a `rank` (its 1-based position, offsets included) and a `score` (normalized 0–1 within the returned page, `0` being the strongest match). Results arrive best-first, so you rarely need either — they're there for debugging and "why is this ranking above that?" sessions.

## Query Syntax

By default, `query()` speaks search engine, not SQL:

| You type | It means |
|----------|----------|
| `basil watering` | both words must appear |
| `"growing basil"` | this exact phrase |
| `basil -hydroponics` | "basil", but not "hydroponics" |

Special characters like quotes and parentheses are escaped automatically, so user input can't break the query. An empty query returns an empty result set rather than an error.

With the `"porter"` tokenizer, words are stemmed: searching "watering" finds "water", "watered", and "waters". If exact matches matter more than stemming — part numbers, non-English text — use `"unicode61"`.

### Raw FTS5 Queries

For power users, `raw: true` passes your query straight to FTS5:

```parsley
search.query("basil OR parsley", {raw: true})
search.query("NEAR(compost worm, 5)", {raw: true})
search.query("title: pesto", {raw: true})
```

Raw mode skips all sanitization, so only feed it trusted input. A malformed raw query doesn't raise an error — it just returns zero results, which is worth remembering when your clever query mysteriously finds nothing. The full syntax is in the [FTS5 documentation](https://www.sqlite.org/fts5.html).

## Filtering by Tag

Narrow a query to documents carrying certain tags:

```parsley
let results = search.query(q, {
    limit: 10,
    filters: {tags: ["tutorial", "guide"]}    // matches either tag
})
```

A single string works too: `filters: {tags: "tutorial"}`. Two caveats:

- Matching is substring-based, so a filter for `"art"` also matches documents tagged `"articles"`. Keep tag names distinct.
- When filters are active, `total` can overcount — it reports matches before filtering when there are more results than `limit`.

## Manual Indexing

Watched folders cover files. For everything else — database rows, API responses, generated pages — add documents yourself:

```parsley
let {search, error} = @SEARCH({path: "products.db"})

search.add({
    url: "/products/42",                    // required, unique
    title: "Self-Watering Herb Planter",    // required
    content: "Terracotta planter with...",  // required
    tags: ["planters", "herbs"],            // optional: array of strings
    headings: "Features\nCare",             // optional: string
    date: "2026-05-01"                      // optional: date string
})
```

`add()` is also the update: adding a document with an existing `url` replaces it wholesale. (There's an `update()` method that takes the same full document and does the same replace, so `add()` is usually all you need.) `remove()` deletes by URL and doesn't mind if the document is already gone:

```parsley
search.remove("/products/42")
```

The classic pattern — keep the index in step with a table:

```parsley
let {search, error} = @SEARCH({path: "products.db"})

let products = @DB <=??=> "SELECT * FROM products WHERE active = 1"
for (product in products) {
    search.add({
        url: `/products/{product.id}`,
        title: product.name,
        content: product.description,
        tags: [product.category]
    })
}
```

> **Heads-up:** query results don't currently echo back the `url` you gave `add()` — file-indexed documents have `path`, but manual documents have neither. Until that gap is filled, make titles distinctive enough to identify the page, or keep a title-to-URL lookup on the side.

Mixing works fine, by the way: one instance can watch a folder *and* take manual additions, and a query searches everything together.

## Pagination

`limit` and `offset` do what you'd expect:

```parsley
let perPage = 10
let page = toInt(@params.page ?? "1")

let results = search.query(q, {
    limit: perPage,
    offset: (page - 1) * perPage
})

let totalPages = (results.total + perPage - 1) / perPage
```

`results.total` is the full match count, not just this page, so page math works from any offset.

## `stats()` and `reindex()`

`stats()` returns a small dictionary for dashboards and debugging:

```parsley
let stats = search.stats()
// {documents: 142, size: "1.2 MB", last_indexed: "2026-07-14T09:30:00Z"}

<p>`Searching {stats.documents} documents`</p>
```

`reindex()` drops the entire index and rebuilds it from the watched folders — the "turn it off and on again" of search:

```parsley
search.reindex()
```

You shouldn't need it in normal use (the per-query freshness check handles day-to-day changes), but it's the right tool after switching tokenizers or if the index ever looks wrong. It requires `watch` paths — on a manual-only index it returns an error, since there'd be nothing to rebuild from.

## Testing with `:memory:`

An in-memory index makes search tests fast and self-cleaning:

```parsley
let {search, error} = @SEARCH({path: ":memory:"})

search.add({url: "/t/1", title: "Test Doc", content: "findable content"})

let results = search.query("findable")
// results.total == 1
```

## Troubleshooting

**A file isn't showing up.** Check the `extensions` list (defaults are only `.md` and `.html`), whether it's inside a dot-directory, and whether it's over 50 MB. For PDFs: is it a scan? No text layer, no index. `search.stats()` tells you how many documents made it in.

**A search finds nothing it should.** Remember the default AND logic — every word must appear. Try a single word. If special syntax is involved, remember malformed `raw` queries return empty rather than erroring.

**Stemming surprises.** "Porter" happily equates "cooking" and "cook". If that's wrong for your content, switch to `unicode61` — and `reindex()` afterwards, since the tokenizer is baked into the index.

**The index looks stale or strange.** `search.reindex()`, or stop the server and delete the `_search.db` file. Both rebuild from scratch.

**`reindex() requires watch paths to be configured`.** You called it on a manual-only index. Re-`add()` your documents instead — `add()` already replaces by URL.

## See Also

- [Search](search.md) — the essentials
- [Database](database.md) — `@DB` and the query operators used above
- [Routing](routing.md) — `@params` for reading the search box
