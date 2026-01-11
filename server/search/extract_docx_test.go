package search

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// createTestDOCX creates a minimal valid DOCX file for testing
func createTestDOCX(t *testing.T, dir, name, title, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)

	// Add [Content_Types].xml (required for valid DOCX)
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
</Types>`
	addZipFile(t, w, "[Content_Types].xml", contentTypes)

	// Add _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
</Relationships>`
	addZipFile(t, w, "_rels/.rels", rels)

	// Add word/document.xml with content
	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
      <w:r><w:t>` + escapeXML(title) + `</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>` + escapeXML(content) + `</w:t></w:r>
    </w:p>
  </w:body>
</w:document>`
	addZipFile(t, w, "word/document.xml", document)

	// Add docProps/core.xml with metadata
	coreProps := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
                   xmlns:dc="http://purl.org/dc/elements/1.1/"
                   xmlns:dcterms="http://purl.org/dc/terms/">
  <dc:title>` + escapeXML(title) + `</dc:title>
  <dc:keywords>test, docx, search</dc:keywords>
  <dcterms:created>2024-01-15T10:30:00Z</dcterms:created>
  <dcterms:modified>2024-01-15T10:30:00Z</dcterms:modified>
</cp:coreProperties>`
	addZipFile(t, w, "docProps/core.xml", coreProps)

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	return path
}

func addZipFile(t *testing.T, w *zip.Writer, name, content string) {
	t.Helper()
	f, err := w.Create(name)
	if err != nil {
		t.Fatalf("failed to create zip entry %s: %v", name, err)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write zip entry %s: %v", name, err)
	}
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func TestProcessDOCX(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("basic document", func(t *testing.T) {
		path := createTestDOCX(t, tmpDir, "test.docx", "Test Document", "This is the document content for testing.")
		mtime := time.Now()

		doc, err := ProcessDOCX(path, mtime)
		if err != nil {
			t.Fatalf("ProcessDOCX failed: %v", err)
		}

		if doc.Title != "Test Document" {
			t.Errorf("expected title 'Test Document', got '%s'", doc.Title)
		}

		if !strings.Contains(doc.Content, "document content for testing") {
			t.Errorf("expected content to contain 'document content for testing', got '%s'", doc.Content)
		}

		// Check that heading was extracted
		if !strings.Contains(doc.Headings, "Test Document") {
			t.Errorf("expected headings to contain 'Test Document', got '%s'", doc.Headings)
		}

		// Check tags from keywords
		if len(doc.Tags) != 3 {
			t.Errorf("expected 3 tags, got %d: %v", len(doc.Tags), doc.Tags)
		}

		// Check URL generation
		if !strings.HasSuffix(doc.URL, "/test") {
			t.Errorf("expected URL to end with '/test', got '%s'", doc.URL)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := ProcessDOCX("/nonexistent/file.docx", time.Now())
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("invalid zip", func(t *testing.T) {
		// Create a non-zip file
		invalidPath := filepath.Join(tmpDir, "invalid.docx")
		if err := os.WriteFile(invalidPath, []byte("not a zip file"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := ProcessDOCX(invalidPath, time.Now())
		if err == nil {
			t.Error("expected error for invalid zip file")
		}
	})
}

func TestIsDOCX(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"document.docx", true},
		{"DOCUMENT.DOCX", true},
		{"file.DocX", true},
		{"document.doc", false},
		{"document.pdf", false},
		{"document.md", false},
		{"/path/to/file.docx", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := IsDOCX(tt.path)
			if result != tt.expected {
				t.Errorf("IsDOCX(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestScanFolderWithDOCX(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mix of files
	createTestDOCX(t, tmpDir, "report.docx", "Annual Report", "Financial summary for 2024.")

	// Create a markdown file too
	mdContent := `---
title: Guide
---
# Getting Started
This is a guide.`
	if err := os.WriteFile(filepath.Join(tmpDir, "guide.md"), []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Scan with both extensions
	opts := &ScanOptions{
		Extensions: []string{".md", ".docx"},
		Recursive:  true,
	}

	docs, err := ScanFolder(tmpDir, opts)
	if err != nil {
		t.Fatalf("ScanFolder failed: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("expected 2 documents, got %d", len(docs))
	}

	// Check that both documents were processed correctly
	foundDocx := false
	foundMd := false
	for _, doc := range docs {
		if strings.HasSuffix(doc.Path, ".docx") {
			foundDocx = true
			if doc.Title != "Annual Report" {
				t.Errorf("DOCX title mismatch: got %s", doc.Title)
			}
		}
		if strings.HasSuffix(doc.Path, ".md") {
			foundMd = true
			if doc.Title != "Guide" {
				t.Errorf("MD title mismatch: got %s", doc.Title)
			}
		}
	}

	if !foundDocx {
		t.Error("DOCX file was not found in scan results")
	}
	if !foundMd {
		t.Error("MD file was not found in scan results")
	}
}
