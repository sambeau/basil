package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/search"
)

func TestSearchAddMethod(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create search instance
	instance, cleanup := createTestSearchInstance(t, dbPath)
	defer cleanup()

	env := evaluator.NewEnvironment()

	// Test adding a document
	doc := evaluator.NewDictionaryFromObjects(map[string]evaluator.Object{
		"url":     &evaluator.String{Value: "/test/doc1"},
		"title":   &evaluator.String{Value: "Test Document"},
		"content": &evaluator.String{Value: "This is test content for searching"},
	})
	doc.Env = env

	result := searchAddMethod(instance, []evaluator.Object{doc}, env)
	if _, ok := result.(*evaluator.Boolean); !ok {
		t.Fatalf("Expected boolean result, got %T: %v", result, result)
	}

	// Verify document was indexed by searching for it
	queryDict := evaluator.NewDictionaryFromObjects(map[string]evaluator.Object{})
	queryDict.Env = env
	searchResult := searchQueryMethod(instance, []evaluator.Object{
		&evaluator.String{Value: "test"},
		queryDict,
	}, env)

	resultsDict, ok := searchResult.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("Expected dictionary result, got %T: %v", searchResult, searchResult)
	}

	totalObj := evaluator.Eval(resultsDict.Pairs["total"], env)
	total, ok := totalObj.(*evaluator.Integer)
	if !ok || total.Value != 1 {
		t.Errorf("Expected 1 result, got %v", totalObj)
	}
}

func TestSearchAddMethodWithOptionalFields(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	instance, cleanup := createTestSearchInstance(t, dbPath)
	defer cleanup()

	env := evaluator.NewEnvironment()

	// Test adding a document with optional fields
	tags := &evaluator.Array{Elements: []evaluator.Object{
		&evaluator.String{Value: "golang"},
		&evaluator.String{Value: "testing"},
	}}

	doc := evaluator.NewDictionaryFromObjects(map[string]evaluator.Object{
		"url":      &evaluator.String{Value: "/test/doc2"},
		"title":    &evaluator.String{Value: "Advanced Document"},
		"content":  &evaluator.String{Value: "Content with tags and date"},
		"headings": &evaluator.String{Value: "Section 1\nSection 2"},
		"tags":     tags,
		"date":     &evaluator.String{Value: "2025-01-09"},
	})
	doc.Env = env

	result := searchAddMethod(instance, []evaluator.Object{doc}, env)
	if _, ok := result.(*evaluator.Boolean); !ok {
		t.Fatalf("Expected boolean result, got %T: %v", result, result)
	}

	// Search for document by tag
	queryDict := evaluator.NewDictionaryFromObjects(map[string]evaluator.Object{})
	queryDict.Env = env
	searchResult := searchQueryMethod(instance, []evaluator.Object{
		&evaluator.String{Value: "golang"},
		queryDict,
	}, env)

	resultsDict, ok := searchResult.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("Expected dictionary result, got %T", searchResult)
	}

	totalObj := evaluator.Eval(resultsDict.Pairs["total"], env)
	total, ok := totalObj.(*evaluator.Integer)
	if !ok || total.Value != 1 {
		t.Errorf("Expected 1 result when searching by tag, got %v", totalObj)
	}
}

func TestSearchAddMethodValidation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	instance, cleanup := createTestSearchInstance(t, dbPath)
	defer cleanup()

	env := evaluator.NewEnvironment()

	tests := []struct {
		name     string
		doc      map[string]evaluator.Object
		wantErr  bool
		errClass string
	}{
		{
			name: "missing url",
			doc: map[string]evaluator.Object{
				"title":   &evaluator.String{Value: "Test"},
				"content": &evaluator.String{Value: "Content"},
			},
			wantErr:  true,
			errClass: "value",
		},
		{
			name: "missing title",
			doc: map[string]evaluator.Object{
				"url":     &evaluator.String{Value: "/test"},
				"content": &evaluator.String{Value: "Content"},
			},
			wantErr:  true,
			errClass: "value",
		},
		{
			name: "missing content",
			doc: map[string]evaluator.Object{
				"url":   &evaluator.String{Value: "/test"},
				"title": &evaluator.String{Value: "Test"},
			},
			wantErr:  true,
			errClass: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := evaluator.NewDictionaryFromObjects(tt.doc)
			doc.Env = env

			result := searchAddMethod(instance, []evaluator.Object{doc}, env)
			if tt.wantErr {
				if err, ok := result.(*evaluator.Error); ok {
					if string(err.Class) != tt.errClass {
						t.Errorf("Expected error class %s, got %s", tt.errClass, err.Class)
					}
				} else {
					t.Errorf("Expected error, got %T", result)
				}
			}
		})
	}
}

func TestSearchUpdateMethod(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	instance, cleanup := createTestSearchInstance(t, dbPath)
	defer cleanup()

	env := evaluator.NewEnvironment()

	// First add a document
	doc1 := evaluator.NewDictionaryFromObjects(map[string]evaluator.Object{
		"url":     &evaluator.String{Value: "/test/doc3"},
		"title":   &evaluator.String{Value: "Original Title"},
		"content": &evaluator.String{Value: "Original content"},
	})
	doc1.Env = env

	result := searchAddMethod(instance, []evaluator.Object{doc1}, env)
	if _, ok := result.(*evaluator.Boolean); !ok {
		t.Fatalf("Failed to add initial document: %v", result)
	}

	// Update the document
	doc2 := evaluator.NewDictionaryFromObjects(map[string]evaluator.Object{
		"url":     &evaluator.String{Value: "/test/doc3"},
		"title":   &evaluator.String{Value: "Updated Title"},
		"content": &evaluator.String{Value: "Updated content with new information"},
	})
	doc2.Env = env

	result = searchUpdateMethod(instance, []evaluator.Object{doc2}, env)
	if _, ok := result.(*evaluator.Boolean); !ok {
		t.Fatalf("Expected boolean result, got %T: %v", result, result)
	}

	// Search for updated content
	queryDict := evaluator.NewDictionaryFromObjects(map[string]evaluator.Object{})
	queryDict.Env = env
	searchResult := searchQueryMethod(instance, []evaluator.Object{
		&evaluator.String{Value: "new information"},
		queryDict,
	}, env)

	resultsDict, ok := searchResult.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("Expected dictionary result, got %T", searchResult)
	}

	totalObj := evaluator.Eval(resultsDict.Pairs["total"], env)
	total, ok := totalObj.(*evaluator.Integer)
	if !ok || total.Value != 1 {
		t.Errorf("Expected 1 result for updated content, got %v", totalObj)
	}

	// Verify old content is not found
	searchResult2 := searchQueryMethod(instance, []evaluator.Object{
		&evaluator.String{Value: "Original"},
		queryDict,
	}, env)

	resultsDict2, ok := searchResult2.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("Expected dictionary result, got %T", searchResult2)
	}

	totalObj2 := evaluator.Eval(resultsDict2.Pairs["total"], env)
	total2, ok := totalObj2.(*evaluator.Integer)
	if !ok || total2.Value != 0 {
		t.Errorf("Expected 0 results for old content, got %v", totalObj2)
	}
}

func TestSearchRemoveMethod(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	instance, cleanup := createTestSearchInstance(t, dbPath)
	defer cleanup()

	env := evaluator.NewEnvironment()

	// Add a document
	doc := evaluator.NewDictionaryFromObjects(map[string]evaluator.Object{
		"url":     &evaluator.String{Value: "/test/doc4"},
		"title":   &evaluator.String{Value: "Document to Delete"},
		"content": &evaluator.String{Value: "This document will be removed"},
	})
	doc.Env = env

	result := searchAddMethod(instance, []evaluator.Object{doc}, env)
	if _, ok := result.(*evaluator.Boolean); !ok {
		t.Fatalf("Failed to add document: %v", result)
	}

	// Verify it exists
	queryDict := evaluator.NewDictionaryFromObjects(map[string]evaluator.Object{})
	queryDict.Env = env
	searchResult := searchQueryMethod(instance, []evaluator.Object{
		&evaluator.String{Value: "removed"},
		queryDict,
	}, env)

	resultsDict, ok := searchResult.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("Expected dictionary result, got %T", searchResult)
	}

	totalObj := evaluator.Eval(resultsDict.Pairs["total"], env)
	total, ok := totalObj.(*evaluator.Integer)
	if !ok || total.Value != 1 {
		t.Fatalf("Expected 1 result before removal, got %v", totalObj)
	}

	// Remove the document
	result = searchRemoveMethod(instance, []evaluator.Object{
		&evaluator.String{Value: "/test/doc4"},
	}, env)
	if _, ok := result.(*evaluator.Boolean); !ok {
		t.Fatalf("Expected boolean result, got %T: %v", result, result)
	}

	// Verify it's gone
	searchResult2 := searchQueryMethod(instance, []evaluator.Object{
		&evaluator.String{Value: "removed"},
		queryDict,
	}, env)

	resultsDict2, ok := searchResult2.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("Expected dictionary result, got %T", searchResult2)
	}

	totalObj2 := evaluator.Eval(resultsDict2.Pairs["total"], env)
	total2, ok := totalObj2.(*evaluator.Integer)
	if !ok || total2.Value != 0 {
		t.Errorf("Expected 0 results after removal, got %v", totalObj2)
	}
}

func TestSearchMixedStaticAndManual(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create a test markdown file
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	mdContent := `---
title: Static Document
tags: [static, auto]
---

Unique static content xyzabc789.
`
	mdPath := filepath.Join(docsDir, "static.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create search instance with watch folder
	instance, cleanup := createTestSearchInstanceWithWatch(t, dbPath, docsDir)
	defer cleanup()

	env := evaluator.NewEnvironment()

	// Trigger auto-indexing
	if err := instance.ensureInitialized(); err != nil {
		t.Fatal(err)
	}

	// Add a manual document
	doc := evaluator.NewDictionaryFromObjects(map[string]evaluator.Object{
		"url":     &evaluator.String{Value: "/manual/doc"},
		"title":   &evaluator.String{Value: "Manual Document"},
		"content": &evaluator.String{Value: "Unique manual content defghj456"},
	})
	doc.Env = env

	result := searchAddMethod(instance, []evaluator.Object{doc}, env)
	if _, ok := result.(*evaluator.Boolean); !ok {
		t.Fatalf("Failed to add manual document: %v", result)
	}

	// Search for static document with very specific term
	queryDict := evaluator.NewDictionaryFromObjects(map[string]evaluator.Object{})
	queryDict.Env = env
	searchResult := searchQueryMethod(instance, []evaluator.Object{
		&evaluator.String{Value: "xyzabc789"},
		queryDict,
	}, env)

	resultsDict, ok := searchResult.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("Expected dictionary result, got %T", searchResult)
	}

	totalObj := evaluator.Eval(resultsDict.Pairs["total"], env)
	total, ok := totalObj.(*evaluator.Integer)
	if !ok || total.Value < 1 {
		t.Errorf("Expected at least 1 result for static document, got %v", totalObj)
	}

	// Search for manual document with very specific term
	searchResult2 := searchQueryMethod(instance, []evaluator.Object{
		&evaluator.String{Value: "defghj456"},
		queryDict,
	}, env)

	resultsDict2, ok := searchResult2.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("Expected dictionary result, got %T", searchResult2)
	}

	totalObj2 := evaluator.Eval(resultsDict2.Pairs["total"], env)
	total2, ok := totalObj2.(*evaluator.Integer)
	if !ok || total2.Value < 1 {
		t.Errorf("Expected at least 1 result for manual document, got %v", totalObj2)
	}
}

func TestSearchReindexMethod(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create a test markdown file
	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	mdContent := `---
title: Original Document
tags: [test]
---

Original content here.
`
	mdPath := filepath.Join(docsDir, "doc.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create search instance with watch folder
	instance, cleanup := createTestSearchInstanceWithWatch(t, dbPath, docsDir)
	defer cleanup()

	env := evaluator.NewEnvironment()

	// Trigger initial indexing
	if err := instance.ensureInitialized(); err != nil {
		t.Fatal(err)
	}

	// Verify initial document is indexed
	queryDict := evaluator.NewDictionaryFromObjects(map[string]evaluator.Object{})
	queryDict.Env = env
	searchResult := searchQueryMethod(instance, []evaluator.Object{
		&evaluator.String{Value: "Original"},
		queryDict,
	}, env)

	resultsDict, ok := searchResult.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("Expected dictionary result, got %T", searchResult)
	}

	totalObj := evaluator.Eval(resultsDict.Pairs["total"], env)
	total, ok := totalObj.(*evaluator.Integer)
	if !ok || total.Value < 1 {
		t.Fatalf("Expected at least 1 result initially, got %v", totalObj)
	}

	// Get initial count for comparison later
	initialTotal := total.Value

	// Modify the file
	mdContent2 := `---
title: Updated Document
tags: [test, updated]
---

Updated content with new information.
`
	if err := os.WriteFile(mdPath, []byte(mdContent2), 0644); err != nil {
		t.Fatal(err)
	}

	// Call reindex
	result := searchReindexMethod(instance, []evaluator.Object{}, env)
	if _, ok := result.(*evaluator.Boolean); !ok {
		t.Fatalf("Expected boolean result, got %T: %v", result, result)
	}

	// Verify updated content is now indexed
	searchResult2 := searchQueryMethod(instance, []evaluator.Object{
		&evaluator.String{Value: "Updated"},
		queryDict,
	}, env)

	resultsDict2, ok := searchResult2.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("Expected dictionary result, got %T", searchResult2)
	}

	totalObj2 := evaluator.Eval(resultsDict2.Pairs["total"], env)
	total2, ok := totalObj2.(*evaluator.Integer)
	if !ok || total2.Value < 1 {
		t.Errorf("Expected at least 1 result for updated content, got %v", totalObj2)
	}

	// Verify old content is not found after reindex
	searchResult3 := searchQueryMethod(instance, []evaluator.Object{
		&evaluator.String{Value: "Original"},
		queryDict,
	}, env)

	resultsDict3, ok := searchResult3.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("Expected dictionary result, got %T", searchResult3)
	}

	totalObj3 := evaluator.Eval(resultsDict3.Pairs["total"], env)
	total3, ok := totalObj3.(*evaluator.Integer)
	if !ok {
		t.Fatalf("Expected integer for total, got %T", totalObj3)
	}

	// After reindex, searching for "Original" should find 0 results
	if total3.Value != 0 {
		t.Errorf("Expected 0 results for old content after reindex, got %d (initial was %d)", total3.Value, initialTotal)
	}
}

func TestSearchReindexMethodNoWatch(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create search instance WITHOUT watch folder (manual only)
	instance, cleanup := createTestSearchInstance(t, dbPath)
	defer cleanup()

	env := evaluator.NewEnvironment()

	// Attempt to call reindex without watch paths
	result := searchReindexMethod(instance, []evaluator.Object{}, env)

	// Should return error
	if err, ok := result.(*evaluator.Error); ok {
		if string(err.Class) != "runtime" {
			t.Errorf("Expected runtime error, got %s", err.Class)
		}
		if err.Message != "reindex() requires watch paths to be configured" {
			t.Errorf("Expected specific error message, got: %s", err.Message)
		}
	} else {
		t.Errorf("Expected error for reindex without watch, got %T", result)
	}
}

// Helper functions

func createTestSearchInstance(t *testing.T, dbPath string) (*SearchInstance, func()) {
	opts := SearchOptions{
		Backend:      dbPath,
		Tokenizer:    "porter",
		Weights:      search.DefaultWeights(),
		SnippetLen:   150,
		HighlightTag: "mark",
	}

	env := evaluator.NewEnvironment()
	instance, err := createSearchInstance(opts, env)
	if err != nil {
		t.Fatalf("Failed to create search instance: %v", err)
	}

	cleanup := func() {
		if instance.db != nil {
			instance.db.Close()
		}
	}

	return instance, cleanup
}

func createTestSearchInstanceWithWatch(t *testing.T, dbPath, watchDir string) (*SearchInstance, func()) {
	opts := SearchOptions{
		Backend:      dbPath,
		Watch:        []string{watchDir},
		Extensions:   []string{".md"},
		Tokenizer:    "porter",
		Weights:      search.DefaultWeights(),
		SnippetLen:   150,
		HighlightTag: "mark",
	}

	env := evaluator.NewEnvironment()
	instance, err := createSearchInstance(opts, env)
	if err != nil {
		t.Fatalf("Failed to create search instance: %v", err)
	}

	cleanup := func() {
		if instance.db != nil {
			instance.db.Close()
		}
	}

	return instance, cleanup
}
