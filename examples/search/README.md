# Search Example

This example demonstrates Phase 2 (Auto-Indexing) of the full-text search feature using SQLite FTS5.

## Running the Example

```bash
# From the basil repository root
./basil examples/search
```

Then visit http://localhost:8080 in your browser.

## What's Demonstrated

### Phase 2 Auto-Indexing Features

1. **@SEARCH Connection Literal with Watch**
   - Creates a search instance with file watching
   - Automatically scans directories for markdown files

2. **Automatic Document Indexing**
   - Scans `./docs` directory on first query
   - Parses YAML frontmatter (title, tags, date)
   - Extracts headings for better ranking
   - No manual `.add()` calls needed!

3. **Frontmatter Parsing**
   - Extracts metadata from `---` delimited YAML
   - Supports title, tags, date, authors, draft status
   - Falls back to H1 or filename if no title

4. **Markdown Processing**
   - Extracts all headings (H1-H6)
   - Strips formatting for plain-text search
   - Generates URLs from file paths
   - Captures modification times

5. **Search Query Execution**
   - `.query()` method with query string and options
   - Returns results with snippets, scores, and metadata

6. **Index Statistics**
   - `.stats()` method returns document count and size

### Search Results

Results include:
- **URL**: Generated from file path
- **Title**: From frontmatter, H1, or filename
- **Snippet**: Context snippet with `<mark>` highlighting
- **Score**: BM25 relevance score (normalized 0-1)
- **Rank**: Result position
- **Date**: From frontmatter metadata
- **Tags**: From frontmatter

### Query Syntax

Phase 2 supports:
- **Simple terms**: `basil` → searches for "basil"
- **Multiple terms**: `basil server` → "basil AND server" (both required)
- **Quoted phrases**: `"web server"` → exact phrase match
- **Negation**: `basil -python` → has "basil" but not "python"
- **Raw FTS5**: Use `raw: true` option for direct FTS5 syntax

## Example Files

The example includes three markdown documents in `./docs/`:

1. **getting-started.md** - Installation and quick start guide
2. **parsley.md** - Language reference with syntax examples  
3. **search.md** - Full-text search feature documentation

Each file includes YAML frontmatter with metadata.

## Phase 2 Features

✅ **Frontmatter Parsing** - YAML metadata extraction  
✅ **Markdown Processing** - Headings, formatting stripping  
✅ **File Scanner** - Recursive directory traversal  
✅ **Auto-Indexing** - Automatic indexing on first query  
✅ **Multiple Watch Folders** - Support for multiple directories

## Phase 2 Limitations

- **No File Watching**: Changes require manual reindex (Phase 3)
- **No Metadata Filtering**: Can't filter by tags/dates yet (Task 2.5)
- **No Incremental Updates**: Full reindex required (Phase 3)

## Next: Phase 3

Phase 3 will add:
- File watching for automatic updates
- Incremental indexing (update only changed files)
- Metadata tracking for efficient updates
