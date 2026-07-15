package search

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestNewFTS5Index(t *testing.T) {
	// Use in-memory database for tests
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	t.Run("create with porter tokenizer", func(t *testing.T) {
		idx, err := NewFTS5Index(db, "porter", DefaultWeights())
		if err != nil {
			t.Fatalf("failed to create index: %v", err)
		}
		if idx == nil {
			t.Fatal("index is nil")
		}
	})

	t.Run("create with unicode61 tokenizer", func(t *testing.T) {
		// Use a fresh database for each test
		db2, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer db2.Close()

		idx, err := NewFTS5Index(db2, "unicode61", DefaultWeights())
		if err != nil {
			t.Fatalf("failed to create index: %v", err)
		}
		if idx == nil {
			t.Fatal("index is nil")
		}
	})

	t.Run("invalid tokenizer", func(t *testing.T) {
		db3, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer db3.Close()

		_, err = NewFTS5Index(db3, "invalid", DefaultWeights())
		if err == nil {
			t.Fatal("expected error for invalid tokenizer")
		}
	})
}

func TestIndexDocument(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	idx, err := NewFTS5Index(db, "porter", DefaultWeights())
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	t.Run("index valid document", func(t *testing.T) {
		doc := &Document{
			URL:      "/test",
			Title:    "Test Document",
			Content:  "This is a test document with some content.",
			Tags:     []string{"test", "example"},
			Headings: "Introduction\nConclusion",
		}
		err := idx.IndexDocument(doc)
		if err != nil {
			t.Fatalf("failed to index document: %v", err)
		}
	})

	t.Run("index document with HTML", func(t *testing.T) {
		doc := &Document{
			URL:     "/html",
			Title:   "<h1>Title with HTML</h1>",
			Content: "<p>Content with <strong>bold</strong> text.</p>",
		}
		err := idx.IndexDocument(doc)
		if err != nil {
			t.Fatalf("failed to index document: %v", err)
		}
	})

	t.Run("index document with markdown", func(t *testing.T) {
		doc := &Document{
			URL:     "/markdown",
			Title:   "**Bold Title**",
			Content: "Content with *italic* and **bold** text.",
		}
		err := idx.IndexDocument(doc)
		if err != nil {
			t.Fatalf("failed to index document: %v", err)
		}
	})

	t.Run("invalid document - missing URL", func(t *testing.T) {
		doc := &Document{
			Title:   "Test",
			Content: "Content",
		}
		err := idx.IndexDocument(doc)
		if err == nil {
			t.Fatal("expected error for missing URL")
		}
	})

	t.Run("invalid document - missing title", func(t *testing.T) {
		doc := &Document{
			URL:     "/test",
			Content: "Content",
		}
		err := idx.IndexDocument(doc)
		if err == nil {
			t.Fatal("expected error for missing title")
		}
	})

	t.Run("invalid document - missing content", func(t *testing.T) {
		doc := &Document{
			URL:   "/test",
			Title: "Title",
		}
		err := idx.IndexDocument(doc)
		if err == nil {
			t.Fatal("expected error for missing content")
		}
	})
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<p>Hello</p>", "Hello"},
		{"<div><strong>Bold</strong></div>", "Bold"},
		{"Plain text", "Plain text"},
		{"<a href='link'>Text</a>", "Text"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := StripHTML(tt.input)
			if result != tt.expected {
				t.Errorf("StripHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"**bold**", "bold"},
		{"*italic*", "italic"},
		{"`code`", "code"},
		{"~~strikethrough~~", "strikethrough"},
		{"[link](url)", "link"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := StripMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("StripMarkdown(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSearch(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	idx, err := NewFTS5Index(db, "porter", DefaultWeights())
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	// Index some test documents
	docs := []*Document{
		{
			URL:     "/doc1",
			Title:   "First Document",
			Content: "This is the first document with some interesting content.",
			Tags:    []string{"first", "test"},
		},
		{
			URL:     "/doc2",
			Title:   "Second Document",
			Content: "This is the second document with different content.",
			Tags:    []string{"second", "test"},
		},
		{
			URL:     "/doc3",
			Title:   "Third Document",
			Content: "Yet another document about completely different topics.",
			Tags:    []string{"third", "other"},
		},
	}

	for _, doc := range docs {
		if err := idx.IndexDocument(doc); err != nil {
			t.Fatalf("failed to index document: %v", err)
		}
	}

	t.Run("search for existing term", func(t *testing.T) {
		results, err := idx.Search("document", DefaultSearchOptions())
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if results.Total < 3 {
			t.Errorf("expected at least 3 results, got %d", results.Total)
		}
	})

	t.Run("search with limit", func(t *testing.T) {
		opts := DefaultSearchOptions()
		opts.Limit = 2
		results, err := idx.Search("document", opts)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results.Results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results.Results))
		}
	})

	t.Run("search with AND query", func(t *testing.T) {
		results, err := idx.Search("first interesting", DefaultSearchOptions())
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if results.Total == 0 {
			t.Error("expected at least 1 result")
		}
	})

	t.Run("empty query", func(t *testing.T) {
		results, err := idx.Search("", DefaultSearchOptions())
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if results.Total != 0 {
			t.Errorf("expected 0 results for empty query, got %d", results.Total)
		}
	})

	t.Run("no matching results", func(t *testing.T) {
		results, err := idx.Search("nonexistent", DefaultSearchOptions())
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if results.Total != 0 {
			t.Errorf("expected 0 results, got %d", results.Total)
		}
	})
}

func TestSnippetOptions(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	idx, err := NewFTS5Index(db, "porter", DefaultWeights())
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	doc := &Document{
		URL:   "/doc1",
		Title: "Long Document",
		Content: "The quick brown fox jumps over the lazy dog again and again, " +
			"running through fields and forests for many days and nights without rest.",
	}
	if err := idx.IndexDocument(doc); err != nil {
		t.Fatalf("failed to index document: %v", err)
	}

	t.Run("default highlight tag is mark", func(t *testing.T) {
		results, err := idx.Search("fox", DefaultSearchOptions())
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results.Results) == 0 {
			t.Fatal("expected a result")
		}
		if !strings.Contains(results.Results[0].Snippet, "<mark>fox</mark>") {
			t.Errorf("expected <mark> highlighting, got %q", results.Results[0].Snippet)
		}
	})

	t.Run("custom tag and length", func(t *testing.T) {
		if err := idx.SetSnippetOptions(4, "em"); err != nil {
			t.Fatalf("SetSnippetOptions failed: %v", err)
		}
		results, err := idx.Search("fox", DefaultSearchOptions())
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		if len(results.Results) == 0 {
			t.Fatal("expected a result")
		}
		snippet := results.Results[0].Snippet
		if !strings.Contains(snippet, "<em>fox</em>") {
			t.Errorf("expected <em> highlighting, got %q", snippet)
		}
		if strings.Contains(snippet, "<mark>") {
			t.Errorf("expected no <mark> tags, got %q", snippet)
		}
		if words := strings.Fields(strings.TrimSuffix(snippet, "...")); len(words) > 4 {
			t.Errorf("expected at most 4 tokens, got %d: %q", len(words), snippet)
		}
	})

	t.Run("rejects invalid highlight tag", func(t *testing.T) {
		if err := idx.SetSnippetOptions(10, "mark'); DROP TABLE documents_fts;--"); err == nil {
			t.Fatal("expected error for invalid highlight tag")
		}
	})
}

func TestRemoveDocument(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	idx, err := NewFTS5Index(db, "porter", DefaultWeights())
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	// Index a document
	doc := &Document{
		URL:     "/test",
		Title:   "Test Document",
		Content: "Test content",
	}
	if err := idx.IndexDocument(doc); err != nil {
		t.Fatalf("failed to index document: %v", err)
	}

	// Verify it exists
	results, err := idx.Search("test", DefaultSearchOptions())
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if results.Total == 0 {
		t.Fatal("document not found after indexing")
	}

	// Remove it
	if err := idx.RemoveDocument("/test"); err != nil {
		t.Fatalf("failed to remove document: %v", err)
	}

	// Verify it's gone
	results, err = idx.Search("test", DefaultSearchOptions())
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if results.Total != 0 {
		t.Error("document still found after removal")
	}
}

func TestUpdateDocument(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	idx, err := NewFTS5Index(db, "porter", DefaultWeights())
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	// Index a document
	doc := &Document{
		URL:     "/test",
		Title:   "Original Title",
		Content: "Original content",
	}
	if err := idx.IndexDocument(doc); err != nil {
		t.Fatalf("failed to index document: %v", err)
	}

	// Update it
	updates := map[string]any{
		"title":   "Updated Title",
		"content": "Updated content with new information",
	}
	if err := idx.UpdateDocument("/test", updates); err != nil {
		t.Fatalf("failed to update document: %v", err)
	}

	// Search for new content
	results, err := idx.Search("updated", DefaultSearchOptions())
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if results.Total == 0 {
		t.Error("updated document not found")
	}
}

func TestBatchIndex(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	idx, err := NewFTS5Index(db, "porter", DefaultWeights())
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	// Create multiple documents
	docs := []*Document{
		{URL: "/doc1", Title: "Doc 1", Content: "Content 1"},
		{URL: "/doc2", Title: "Doc 2", Content: "Content 2"},
		{URL: "/doc3", Title: "Doc 3", Content: "Content 3"},
	}

	// Batch index
	if err := idx.BatchIndex(docs); err != nil {
		t.Fatalf("batch index failed: %v", err)
	}

	// Verify all documents are indexed
	results, err := idx.Search("content", DefaultSearchOptions())
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if results.Total != 3 {
		t.Errorf("expected 3 results, got %d", results.Total)
	}
}

func TestStats(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	idx, err := NewFTS5Index(db, "porter", DefaultWeights())
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	// Index some documents
	docs := []*Document{
		{URL: "/doc1", Title: "Doc 1", Content: "Content 1"},
		{URL: "/doc2", Title: "Doc 2", Content: "Content 2"},
	}
	if err := idx.BatchIndex(docs); err != nil {
		t.Fatalf("batch index failed: %v", err)
	}

	// Get stats
	stats, err := idx.Stats()
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}

	// Check document count
	if docs, ok := stats["documents"].(int); !ok || docs != 2 {
		t.Errorf("expected 2 documents, got %v", stats["documents"])
	}

	// Check size exists
	if _, ok := stats["size"]; !ok {
		t.Error("stats missing size field")
	}
}

func TestReindex(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	idx, err := NewFTS5Index(db, "porter", DefaultWeights())
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	// Index initial documents
	initialDocs := []*Document{
		{URL: "/doc1", Title: "Original 1", Content: "Original content 1"},
		{URL: "/doc2", Title: "Original 2", Content: "Original content 2"},
	}
	if err := idx.BatchIndex(initialDocs); err != nil {
		t.Fatalf("initial batch index failed: %v", err)
	}

	// Verify initial documents are indexed
	results, err := idx.Search("Original", DefaultSearchOptions())
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if results.Total != 2 {
		t.Fatalf("expected 2 initial results, got %d", results.Total)
	}

	// Call Reindex (drops and recreates tables)
	if err := idx.Reindex(); err != nil {
		t.Fatalf("reindex failed: %v", err)
	}

	// Verify index is now empty
	results2, err := idx.Search("Original", DefaultSearchOptions())
	if err != nil {
		t.Fatalf("search after reindex failed: %v", err)
	}
	if results2.Total != 0 {
		t.Errorf("expected 0 results after reindex, got %d", results2.Total)
	}

	// Re-index with new documents
	newDocs := []*Document{
		{URL: "/doc3", Title: "New 1", Content: "New content 1"},
		{URL: "/doc4", Title: "New 2", Content: "New content 2"},
		{URL: "/doc5", Title: "New 3", Content: "New content 3"},
	}
	if err := idx.BatchIndex(newDocs); err != nil {
		t.Fatalf("reindex batch failed: %v", err)
	}

	// Verify new documents are indexed
	results3, err := idx.Search("New", DefaultSearchOptions())
	if err != nil {
		t.Fatalf("search for new docs failed: %v", err)
	}
	if results3.Total != 3 {
		t.Errorf("expected 3 results after reindexing, got %d", results3.Total)
	}

	// Verify old documents are not found
	results4, err := idx.Search("Original", DefaultSearchOptions())
	if err != nil {
		t.Fatalf("search for old docs failed: %v", err)
	}
	if results4.Total != 0 {
		t.Errorf("expected 0 results for old docs, got %d", results4.Total)
	}
}

func TestSearchDateFilters(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	idx, err := NewFTS5Index(db, "porter", DefaultWeights())
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	docs := []*Document{
		{URL: "/old", Title: "Old Post", Content: "basil search content", Date: time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)},
		{URL: "/new", Title: "New Post", Content: "basil search content", Date: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
		{URL: "/undated", Title: "Undated Post", Content: "basil search content"},
	}
	if err := idx.BatchIndex(docs); err != nil {
		t.Fatalf("failed to index documents: %v", err)
	}

	cutoff := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	search := func(t *testing.T, filters SearchFilters) []SearchResult {
		t.Helper()
		opts := DefaultSearchOptions()
		opts.Filters = filters
		results, err := idx.Search("basil", opts)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
		return results.Results
	}

	t.Run("dateAfter", func(t *testing.T) {
		results := search(t, SearchFilters{DateAfter: cutoff})
		if len(results) != 1 || results[0].URL != "/new" {
			t.Errorf("expected only /new, got %+v", results)
		}
	})

	t.Run("dateBefore", func(t *testing.T) {
		results := search(t, SearchFilters{DateBefore: cutoff})
		if len(results) != 1 || results[0].URL != "/old" {
			t.Errorf("expected only /old (undated docs excluded), got %+v", results)
		}
	})

	t.Run("date range", func(t *testing.T) {
		results := search(t, SearchFilters{
			DateAfter:  time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC),
			DateBefore: cutoff,
		})
		if len(results) != 1 || results[0].URL != "/old" {
			t.Errorf("expected only /old, got %+v", results)
		}
	})

	t.Run("boundary is inclusive", func(t *testing.T) {
		exact := time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)
		results := search(t, SearchFilters{DateAfter: exact, DateBefore: exact})
		if len(results) != 1 || results[0].URL != "/old" {
			t.Errorf("expected /old on exact boundary, got %+v", results)
		}
	})

	t.Run("timezones normalized to UTC", func(t *testing.T) {
		// 2023-01-01T08:00+10:00 is 2022-12-31T22:00Z — before the cutoff
		offsetDoc := &Document{
			URL:     "/offset",
			Title:   "Offset Post",
			Content: "basil search content",
			Date:    time.Date(2023, 1, 1, 8, 0, 0, 0, time.FixedZone("UTC+10", 10*3600)),
		}
		if err := idx.IndexDocument(offsetDoc); err != nil {
			t.Fatalf("failed to index document: %v", err)
		}
		defer idx.RemoveDocument("/offset")

		results := search(t, SearchFilters{DateAfter: cutoff})
		for _, r := range results {
			if r.URL == "/offset" {
				t.Errorf("/offset is before the cutoff in UTC, should be excluded")
			}
		}
		results = search(t, SearchFilters{DateBefore: cutoff})
		found := false
		for _, r := range results {
			if r.URL == "/offset" {
				found = true
			}
		}
		if !found {
			t.Errorf("/offset is before the cutoff in UTC, should match dateBefore")
		}
	})
}

func TestSearchTotalWithFilters(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	idx, err := NewFTS5Index(db, "porter", DefaultWeights())
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	// Four documents match the query, but only three pass the date filter
	docs := []*Document{
		{URL: "/a", Title: "Post A", Content: "basil content", Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{URL: "/b", Title: "Post B", Content: "basil content", Date: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
		{URL: "/c", Title: "Post C", Content: "basil content", Date: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
		{URL: "/d", Title: "Post D", Content: "basil content", Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	if err := idx.BatchIndex(docs); err != nil {
		t.Fatalf("failed to index documents: %v", err)
	}

	// Limit smaller than the filtered match count forces the COUNT query
	opts := DefaultSearchOptions()
	opts.Limit = 2
	opts.Filters.DateAfter = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	results, err := idx.Search("basil", opts)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results.Results) != 2 {
		t.Errorf("expected 2 results in page, got %d", len(results.Results))
	}
	if results.Total != 3 {
		t.Errorf("expected total 3 (filters applied to count), got %d", results.Total)
	}
}
