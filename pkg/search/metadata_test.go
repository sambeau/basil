package search

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestCreateMetadataTable(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = CreateMetadataTable(db)
	if err != nil {
		t.Fatalf("CreateMetadataTable() failed: %v", err)
	}

	// Verify table exists by querying it
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM search_metadata").Scan(&count)
	if err != nil {
		t.Fatalf("Table not created: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected empty table, got %d rows", count)
	}
}

func TestStoreMetadata(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = CreateMetadataTable(db)
	if err != nil {
		t.Fatalf("CreateMetadataTable() failed: %v", err)
	}

	meta := FileMetadata{
		URL:       "/docs/guide",
		Path:      "./docs/guide.md",
		Mtime:     time.Now().Unix(),
		IndexedAt: time.Now().Unix(),
		Source:    "auto",
	}

	err = StoreMetadata(db, meta)
	if err != nil {
		t.Fatalf("StoreMetadata() failed: %v", err)
	}

	// Verify stored
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM search_metadata WHERE url = ?", meta.URL).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify metadata: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 row, got %d", count)
	}
}

func TestStoreMetadataUpdate(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = CreateMetadataTable(db)
	if err != nil {
		t.Fatalf("CreateMetadataTable() failed: %v", err)
	}

	// Store initial metadata
	meta1 := FileMetadata{
		URL:       "/docs/guide",
		Path:      "./docs/guide.md",
		Mtime:     1000,
		IndexedAt: 1000,
		Source:    "auto",
	}
	err = StoreMetadata(db, meta1)
	if err != nil {
		t.Fatalf("StoreMetadata() failed: %v", err)
	}

	// Update with new mtime
	meta2 := FileMetadata{
		URL:       "/docs/guide",
		Path:      "./docs/guide.md",
		Mtime:     2000,
		IndexedAt: 2000,
		Source:    "auto",
	}
	err = StoreMetadata(db, meta2)
	if err != nil {
		t.Fatalf("StoreMetadata() update failed: %v", err)
	}

	// Verify only one row exists with new mtime
	var count int
	var mtime int64
	err = db.QueryRow("SELECT COUNT(*), MAX(mtime) FROM search_metadata WHERE url = ?", meta1.URL).Scan(&count, &mtime)
	if err != nil {
		t.Fatalf("Failed to verify update: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 row, got %d", count)
	}
	if mtime != 2000 {
		t.Errorf("Expected mtime 2000, got %d", mtime)
	}
}

func TestGetMetadata(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = CreateMetadataTable(db)
	if err != nil {
		t.Fatalf("CreateMetadataTable() failed: %v", err)
	}

	meta := FileMetadata{
		URL:       "/docs/guide",
		Path:      "./docs/guide.md",
		Mtime:     1000,
		IndexedAt: 2000,
		Source:    "auto",
	}
	err = StoreMetadata(db, meta)
	if err != nil {
		t.Fatalf("StoreMetadata() failed: %v", err)
	}

	// Retrieve metadata
	retrieved, err := GetMetadata(db, "/docs/guide")
	if err != nil {
		t.Fatalf("GetMetadata() failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected metadata, got nil")
	}

	if retrieved.URL != meta.URL {
		t.Errorf("Expected URL %s, got %s", meta.URL, retrieved.URL)
	}
	if retrieved.Path != meta.Path {
		t.Errorf("Expected Path %s, got %s", meta.Path, retrieved.Path)
	}
	if retrieved.Mtime != meta.Mtime {
		t.Errorf("Expected Mtime %d, got %d", meta.Mtime, retrieved.Mtime)
	}
	if retrieved.IndexedAt != meta.IndexedAt {
		t.Errorf("Expected IndexedAt %d, got %d", meta.IndexedAt, retrieved.IndexedAt)
	}
	if retrieved.Source != meta.Source {
		t.Errorf("Expected Source %s, got %s", meta.Source, retrieved.Source)
	}
}

func TestGetMetadataNotFound(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = CreateMetadataTable(db)
	if err != nil {
		t.Fatalf("CreateMetadataTable() failed: %v", err)
	}

	retrieved, err := GetMetadata(db, "/nonexistent")
	if err != nil {
		t.Fatalf("GetMetadata() failed: %v", err)
	}

	if retrieved != nil {
		t.Errorf("Expected nil for nonexistent URL, got %+v", retrieved)
	}
}

func TestGetAllMetadata(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = CreateMetadataTable(db)
	if err != nil {
		t.Fatalf("CreateMetadataTable() failed: %v", err)
	}

	// Store multiple metadata entries
	metas := []FileMetadata{
		{URL: "/docs/a", Path: "./docs/a.md", Mtime: 1000, IndexedAt: 1000, Source: "auto"},
		{URL: "/docs/b", Path: "./docs/b.md", Mtime: 2000, IndexedAt: 2000, Source: "auto"},
		{URL: "/docs/c", Path: "./docs/c.md", Mtime: 3000, IndexedAt: 3000, Source: "manual"},
	}

	for _, meta := range metas {
		err = StoreMetadata(db, meta)
		if err != nil {
			t.Fatalf("StoreMetadata() failed: %v", err)
		}
	}

	// Retrieve all
	all, err := GetAllMetadata(db)
	if err != nil {
		t.Fatalf("GetAllMetadata() failed: %v", err)
	}

	if len(all) != 3 {
		t.Fatalf("Expected 3 metadata entries, got %d", len(all))
	}

	// Verify order (should be alphabetical by URL)
	if all[0].URL != "/docs/a" {
		t.Errorf("Expected first URL /docs/a, got %s", all[0].URL)
	}
	if all[2].URL != "/docs/c" {
		t.Errorf("Expected third URL /docs/c, got %s", all[2].URL)
	}
}

func TestRemoveMetadata(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = CreateMetadataTable(db)
	if err != nil {
		t.Fatalf("CreateMetadataTable() failed: %v", err)
	}

	meta := FileMetadata{
		URL:       "/docs/guide",
		Path:      "./docs/guide.md",
		Mtime:     1000,
		IndexedAt: 1000,
		Source:    "auto",
	}
	err = StoreMetadata(db, meta)
	if err != nil {
		t.Fatalf("StoreMetadata() failed: %v", err)
	}

	// Remove metadata
	err = RemoveMetadata(db, "/docs/guide")
	if err != nil {
		t.Fatalf("RemoveMetadata() failed: %v", err)
	}

	// Verify removed
	retrieved, err := GetMetadata(db, "/docs/guide")
	if err != nil {
		t.Fatalf("GetMetadata() failed: %v", err)
	}

	if retrieved != nil {
		t.Errorf("Expected nil after removal, got %+v", retrieved)
	}
}

func TestGetMetadataByPath(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = CreateMetadataTable(db)
	if err != nil {
		t.Fatalf("CreateMetadataTable() failed: %v", err)
	}

	// Store metadata in different paths
	metas := []FileMetadata{
		{URL: "/docs/a", Path: "./docs/guide/a.md", Mtime: 1000, IndexedAt: 1000, Source: "auto"},
		{URL: "/docs/b", Path: "./docs/guide/b.md", Mtime: 2000, IndexedAt: 2000, Source: "auto"},
		{URL: "/other/c", Path: "./other/c.md", Mtime: 3000, IndexedAt: 3000, Source: "auto"},
	}

	for _, meta := range metas {
		err = StoreMetadata(db, meta)
		if err != nil {
			t.Fatalf("StoreMetadata() failed: %v", err)
		}
	}

	// Get metadata for docs/guide prefix
	results, err := GetMetadataByPath(db, "./docs/guide")
	if err != nil {
		t.Fatalf("GetMetadataByPath() failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// Verify results
	if results[0].Path != "./docs/guide/a.md" {
		t.Errorf("Expected path ./docs/guide/a.md, got %s", results[0].Path)
	}
	if results[1].Path != "./docs/guide/b.md" {
		t.Errorf("Expected path ./docs/guide/b.md, got %s", results[1].Path)
	}
}
