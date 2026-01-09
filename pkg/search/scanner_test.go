package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanFolder(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"doc1.md": `---
title: Document 1
tags: [test, example]
---

# Document 1

This is the first test document.`,

		"doc2.md": `# Document 2

This is the second test document without frontmatter.`,

		"nested/doc3.md": `---
title: Nested Document
---

# Nested

A document in a subdirectory.`,

		"nested/deep/doc4.md": `# Deep Document

Very nested file.`,

		"index.html": `<html><body><h1>HTML Document</h1></body></html>`,

		"README.txt": "This should be ignored (wrong extension)",

		".hidden.md": "This should be ignored (hidden file)",
	}

	// Create test files
	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	t.Run("scan with default options", func(t *testing.T) {
		docs, err := ScanFolder(tmpDir, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should find 5 files (.md and .html, excluding .txt and .hidden.md)
		if len(docs) != 5 {
			t.Errorf("expected 5 documents, got %d", len(docs))
		}

		// Check that doc1.md was processed correctly
		foundDoc1 := false
		for _, doc := range docs {
			if doc.Title == "Document 1" {
				foundDoc1 = true
				if len(doc.Tags) != 2 {
					t.Errorf("expected 2 tags, got %d", len(doc.Tags))
				}
			}
		}
		if !foundDoc1 {
			t.Error("doc1.md not found in results")
		}
	})

	t.Run("scan with custom extensions", func(t *testing.T) {
		opts := &ScanOptions{
			Extensions: []string{".txt"},
			Recursive:  true,
		}
		docs, err := ScanFolder(tmpDir, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should only find README.txt
		if len(docs) != 1 {
			t.Errorf("expected 1 document, got %d", len(docs))
		}
	})

	t.Run("scan non-existent folder", func(t *testing.T) {
		_, err := ScanFolder("/nonexistent/path", nil)
		if err == nil {
			t.Error("expected error for non-existent folder")
		}
	})

	t.Run("scan file instead of folder", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "doc1.md")
		_, err := ScanFolder(filePath, nil)
		if err == nil {
			t.Error("expected error when scanning a file")
		}
	})

	t.Run("documents have correct paths", func(t *testing.T) {
		docs, err := ScanFolder(tmpDir, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, doc := range docs {
			if doc.Path == "" {
				t.Error("document missing path")
			}
			if doc.Mtime == 0 {
				t.Error("document missing mtime")
			}
		}
	})

	t.Run("nested directories scanned", func(t *testing.T) {
		docs, err := ScanFolder(tmpDir, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		foundNested := false
		foundDeep := false
		for _, doc := range docs {
			if doc.Title == "Nested Document" {
				foundNested = true
			}
			if doc.Title == "Deep Document" {
				foundDeep = true
			}
		}

		if !foundNested {
			t.Error("nested document not found")
		}
		if !foundDeep {
			t.Error("deep nested document not found")
		}
	})
}

func TestScanMultipleFolders(t *testing.T) {
	// Create two temporary directories
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create test files in first directory
	file1 := filepath.Join(tmpDir1, "doc1.md")
	if err := os.WriteFile(file1, []byte("# Doc 1"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create test files in second directory
	file2 := filepath.Join(tmpDir2, "doc2.md")
	if err := os.WriteFile(file2, []byte("# Doc 2"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	t.Run("scan multiple folders", func(t *testing.T) {
		folders := []string{tmpDir1, tmpDir2}
		docs, err := ScanMultipleFolders(folders, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(docs) != 2 {
			t.Errorf("expected 2 documents, got %d", len(docs))
		}
	})

	t.Run("scan with one invalid folder", func(t *testing.T) {
		folders := []string{tmpDir1, "/nonexistent"}
		docs, err := ScanMultipleFolders(folders, nil)
		// Should succeed with partial results
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(docs) != 1 {
			t.Errorf("expected 1 document from valid folder, got %d", len(docs))
		}
	})

	t.Run("scan with all invalid folders", func(t *testing.T) {
		folders := []string{"/nonexistent1", "/nonexistent2"}
		_, err := ScanMultipleFolders(folders, nil)
		if err == nil {
			t.Error("expected error when all folders are invalid")
		}
	})
}

func TestCountFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := []string{
		"doc1.md",
		"doc2.md",
		"nested/doc3.md",
		"nested/doc4.html",
		"README.txt", // Should not be counted
	}

	for _, file := range files {
		fullPath := filepath.Join(tmpDir, file)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	t.Run("count with default extensions", func(t *testing.T) {
		count, err := CountFiles(tmpDir, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should count 4 files (.md and .html, not .txt)
		if count != 4 {
			t.Errorf("expected 4 files, got %d", count)
		}
	})

	t.Run("count with custom extensions", func(t *testing.T) {
		opts := &ScanOptions{
			Extensions: []string{".txt"},
			Recursive:  true,
		}
		count, err := CountFiles(tmpDir, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if count != 1 {
			t.Errorf("expected 1 file, got %d", count)
		}
	})

	t.Run("count in empty directory", func(t *testing.T) {
		emptyDir := t.TempDir()
		count, err := CountFiles(emptyDir, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if count != 0 {
			t.Errorf("expected 0 files, got %d", count)
		}
	})
}
