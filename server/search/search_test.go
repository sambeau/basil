package search

import (
	"database/sql"
	"testing"

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
	updates := map[string]interface{}{
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
