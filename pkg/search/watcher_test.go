package search

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestCheckForChangesNewFiles(t *testing.T) {
	// Create temp directory with markdown files
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "new.md")
	err := os.WriteFile(file1, []byte("# New File\nContent here."), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create database with empty metadata
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = CreateMetadataTable(db)
	if err != nil {
		t.Fatalf("CreateMetadataTable() failed: %v", err)
	}

	// Check for changes
	changes, err := CheckForChanges(db, []string{tmpDir}, []string{".md"})
	if err != nil {
		t.Fatalf("CheckForChanges() failed: %v", err)
	}

	if len(changes.New) != 1 {
		t.Errorf("Expected 1 new file, got %d", len(changes.New))
	}
	if len(changes.Changed) != 0 {
		t.Errorf("Expected 0 changed files, got %d", len(changes.Changed))
	}
	if len(changes.Deleted) != 0 {
		t.Errorf("Expected 0 deleted files, got %d", len(changes.Deleted))
	}
}

func TestCheckForChangesChangedFiles(t *testing.T) {
	// Create temp directory with markdown file
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "changed.md")
	err := os.WriteFile(file1, []byte("# Original\nOriginal content."), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Wait to ensure mtime is different
	time.Sleep(10 * time.Millisecond)

	// Get initial mtime
	stat1, err := os.Stat(file1)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	mtime1 := stat1.ModTime().Unix()

	// Process the file to get its URL
	doc, err := ProcessMarkdown("# Original\nOriginal content.", file1, stat1.ModTime())
	if err != nil {
		t.Fatalf("ProcessMarkdown() failed: %v", err)
	}

	// Create database and store metadata
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = CreateMetadataTable(db)
	if err != nil {
		t.Fatalf("CreateMetadataTable() failed: %v", err)
	}

	// Store old metadata with the actual URL from ProcessMarkdown
	meta := FileMetadata{
		URL:       doc.URL,
		Path:      file1,
		Mtime:     mtime1,
		IndexedAt: time.Now().Unix(),
		Source:    "auto",
	}
	err = StoreMetadata(db, meta)
	if err != nil {
		t.Fatalf("StoreMetadata() failed: %v", err)
	}

	// Modify the file
	time.Sleep(1100 * time.Millisecond) // Need >1 second for Unix timestamp difference
	err = os.WriteFile(file1, []byte("# Changed\nNew content here."), 0644)
	if err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Check for changes
	changes, err := CheckForChanges(db, []string{tmpDir}, []string{".md"})
	if err != nil {
		t.Fatalf("CheckForChanges() failed: %v", err)
	}

	if len(changes.New) != 0 {
		t.Errorf("Expected 0 new files, got %d", len(changes.New))
	}
	if len(changes.Changed) != 1 {
		t.Errorf("Expected 1 changed file, got %d", len(changes.Changed))
	}
	if len(changes.Deleted) != 0 {
		t.Errorf("Expected 0 deleted files, got %d", len(changes.Deleted))
	}
}

func TestCheckForChangesDeletedFiles(t *testing.T) {
	// Create temp directory with markdown file
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "deleted.md")
	err := os.WriteFile(file1, []byte("# To Delete\nWill be deleted."), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	stat1, err := os.Stat(file1)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// Create database and store metadata
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = CreateMetadataTable(db)
	if err != nil {
		t.Fatalf("CreateMetadataTable() failed: %v", err)
	}

	// Store metadata
	meta := FileMetadata{
		URL:       "/deleted",
		Path:      file1,
		Mtime:     stat1.ModTime().Unix(),
		IndexedAt: time.Now().Unix(),
		Source:    "auto",
	}
	err = StoreMetadata(db, meta)
	if err != nil {
		t.Fatalf("StoreMetadata() failed: %v", err)
	}

	// Delete the file
	err = os.Remove(file1)
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Check for changes
	changes, err := CheckForChanges(db, []string{tmpDir}, []string{".md"})
	if err != nil {
		t.Fatalf("CheckForChanges() failed: %v", err)
	}

	if len(changes.New) != 0 {
		t.Errorf("Expected 0 new files, got %d", len(changes.New))
	}
	if len(changes.Changed) != 0 {
		t.Errorf("Expected 0 changed files, got %d", len(changes.Changed))
	}
	if len(changes.Deleted) != 1 {
		t.Errorf("Expected 1 deleted file, got %d", len(changes.Deleted))
	}

	if len(changes.Deleted) > 0 && changes.Deleted[0] != "/deleted" {
		t.Errorf("Expected deleted URL /deleted, got %s", changes.Deleted[0])
	}
}

func TestCheckForChangesNoChanges(t *testing.T) {
	// Create temp directory with markdown file
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "unchanged.md")
	err := os.WriteFile(file1, []byte("# Unchanged\nStays the same."), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	stat1, err := os.Stat(file1)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// Process file to get actual URL
	doc, err := ProcessMarkdown("# Unchanged\nStays the same.", file1, stat1.ModTime())
	if err != nil {
		t.Fatalf("ProcessMarkdown() failed: %v", err)
	}

	// Create database and store metadata
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = CreateMetadataTable(db)
	if err != nil {
		t.Fatalf("CreateMetadataTable() failed: %v", err)
	}

	// Store metadata with same mtime
	meta := FileMetadata{
		URL:       doc.URL,
		Path:      file1,
		Mtime:     stat1.ModTime().Unix(),
		IndexedAt: time.Now().Unix(),
		Source:    "auto",
	}
	err = StoreMetadata(db, meta)
	if err != nil {
		t.Fatalf("StoreMetadata() failed: %v", err)
	}

	// Check for changes
	changes, err := CheckForChanges(db, []string{tmpDir}, []string{".md"})
	if err != nil {
		t.Fatalf("CheckForChanges() failed: %v", err)
	}

	if len(changes.New) != 0 {
		t.Errorf("Expected 0 new files, got %d", len(changes.New))
	}
	if len(changes.Changed) != 0 {
		t.Errorf("Expected 0 changed files, got %d", len(changes.Changed))
	}
	if len(changes.Deleted) != 0 {
		t.Errorf("Expected 0 deleted files, got %d", len(changes.Deleted))
	}
}

func TestUpdateIndex(t *testing.T) {
	// Create database and FTS5 index
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// CreateMetadataTable is called by NewFTS5Index, but we'll call it explicitly too
	index, err := NewFTS5Index(db, "porter", DefaultWeights())
	if err != nil {
		t.Fatalf("NewFTS5Index() failed: %v", err)
	}

	// Create changeset
	changes := &ChangeSet{
		New: []Document{
			{
				URL:     "/new1",
				Title:   "New Document 1",
				Content: "This is a new document.",
				Mtime:   time.Now().Unix(),
			},
			{
				URL:     "/new2",
				Title:   "New Document 2",
				Content: "Another new document.",
				Mtime:   time.Now().Unix(),
			},
		},
		Changed: []Document{
			{
				URL:     "/changed1",
				Title:   "Changed Document",
				Content: "This document was updated.",
				Mtime:   time.Now().Unix(),
			},
		},
		Deleted: []string{"/deleted1"},
	}

	// Pre-index a document to be changed
	err = index.IndexDocument(&Document{
		URL:     "/changed1",
		Title:   "Old Title",
		Content: "Old content.",
		Mtime:   time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("Failed to pre-index document: %v", err)
	}

	// Pre-index a document to be deleted
	err = index.IndexDocument(&Document{
		URL:     "/deleted1",
		Title:   "To Delete",
		Content: "Will be deleted.",
		Mtime:   time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("Failed to pre-index document: %v", err)
	}

	// Update index
	err = UpdateIndex(index, changes)
	if err != nil {
		t.Fatalf("UpdateIndex() failed: %v", err)
	}

	// Verify new documents indexed
	results, err := index.Search("new document", SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}
	if len(results.Results) < 2 {
		t.Errorf("Expected at least 2 results for new documents, got %d", len(results.Results))
	}

	// Verify changed document updated
	results, err = index.Search("updated", SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}
	if len(results.Results) != 1 {
		t.Errorf("Expected 1 result for changed document, got %d", len(results.Results))
	}

	// Verify deleted document removed
	results, err = index.Search("deleted", SearchOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}
	if len(results.Results) != 0 {
		t.Errorf("Expected 0 results for deleted document, got %d", len(results.Results))
	}

	// Verify metadata stored
	meta, err := GetMetadata(db, "/new1")
	if err != nil {
		t.Fatalf("GetMetadata() failed: %v", err)
	}
	if meta == nil {
		t.Error("Expected metadata for /new1, got nil")
	}
}

func TestCheckAndUpdate(t *testing.T) {
	// Create temp directory with markdown files
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(file1, []byte("# Test\nTest content."), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create database and FTS5 index
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	index, err := NewFTS5Index(db, "porter", DefaultWeights())
	if err != nil {
		t.Fatalf("NewFTS5Index() failed: %v", err)
	}

	// First check - should find new file
	stats, err := CheckAndUpdate(index, []string{tmpDir}, []string{".md"})
	if err != nil {
		t.Fatalf("CheckAndUpdate() failed: %v", err)
	}

	if stats.NewFiles != 1 {
		t.Errorf("Expected 1 new file, got %d", stats.NewFiles)
	}
	if stats.ChangedFiles != 0 {
		t.Errorf("Expected 0 changed files, got %d", stats.ChangedFiles)
	}
	if stats.DeletedFiles != 0 {
		t.Errorf("Expected 0 deleted files, got %d", stats.DeletedFiles)
	}

	// Second check - should find no changes
	stats, err = CheckAndUpdate(index, []string{tmpDir}, []string{".md"})
	if err != nil {
		t.Fatalf("CheckAndUpdate() failed: %v", err)
	}

	if stats.NewFiles != 0 {
		t.Errorf("Expected 0 new files on second check, got %d", stats.NewFiles)
	}

	// Modify file
	time.Sleep(1100 * time.Millisecond) // Need >1 second for Unix timestamp difference
	err = os.WriteFile(file1, []byte("# Modified\nModified content."), 0644)
	if err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Third check - should find changed file
	stats, err = CheckAndUpdate(index, []string{tmpDir}, []string{".md"})
	if err != nil {
		t.Fatalf("CheckAndUpdate() failed: %v", err)
	}

	if stats.ChangedFiles != 1 {
		t.Errorf("Expected 1 changed file, got %d", stats.ChangedFiles)
	}
}

func TestCheckForChangesIgnoresManualFiles(t *testing.T) {
	// Create database with manual and auto metadata
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = CreateMetadataTable(db)
	if err != nil {
		t.Fatalf("CreateMetadataTable() failed: %v", err)
	}

	// Store manual metadata (should not be checked for deletion)
	meta := FileMetadata{
		URL:       "/manual",
		Path:      "/fake/path/manual.md",
		Mtime:     time.Now().Unix(),
		IndexedAt: time.Now().Unix(),
		Source:    "manual",
	}
	err = StoreMetadata(db, meta)
	if err != nil {
		t.Fatalf("StoreMetadata() failed: %v", err)
	}

	// Check for changes (no watch folders)
	changes, err := CheckForChanges(db, []string{}, []string{".md"})
	if err != nil {
		t.Fatalf("CheckForChanges() failed: %v", err)
	}

	// Manual file should not be marked as deleted
	if len(changes.Deleted) != 0 {
		t.Errorf("Expected 0 deleted files (manual files ignored), got %d", len(changes.Deleted))
	}
}
