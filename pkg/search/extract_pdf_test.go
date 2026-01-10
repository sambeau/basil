package search

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProcessPDF(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string // returns file path
		wantErr     bool
		checkResult func(t *testing.T, doc *Document)
	}{
		{
			name: "file not found",
			setup: func(t *testing.T) string {
				return "/nonexistent/file.pdf"
			},
			wantErr: true,
		},
		{
			name: "invalid pdf",
			setup: func(t *testing.T) string {
				// Create a file that's not a valid PDF
				tmpDir := t.TempDir()
				path := filepath.Join(tmpDir, "invalid.pdf")
				err := os.WriteFile(path, []byte("not a pdf file"), 0644)
				if err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			doc, err := ProcessPDF(path, time.Now())

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.checkResult != nil {
				tt.checkResult(t, doc)
			}
		})
	}
}

func TestIsPDF(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"document.pdf", true},
		{"DOCUMENT.PDF", true},
		{"file.Pdf", true},
		{"document.docx", false},
		{"document.md", false},
		{"document.pdf.bak", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsPDF(tt.path)
			if got != tt.want {
				t.Errorf("IsPDF(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestCleanPDFText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single newlines preserved",
			input: "line1\nline2\nline3",
			want:  "line1\nline2\nline3",
		},
		{
			name:  "multiple newlines collapsed",
			input: "line1\n\n\n\nline2",
			want:  "line1\n\nline2",
		},
		{
			name:  "whitespace trimmed",
			input: "  line1  \n  line2  ",
			want:  "line1\nline2",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanPDFText(tt.input)
			if got != tt.want {
				t.Errorf("cleanPDFText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractPDFHeadings(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int // number of headings expected
	}{
		{
			name:    "empty content",
			content: "",
			want:    0,
		},
		{
			name:    "single line",
			content: "Title",
			want:    1,
		},
		{
			name: "all caps heading",
			content: `INTRODUCTION

This is some body text that continues for a while.`,
			want: 1, // "INTRODUCTION" (all caps)
		},
		{
			name: "heading followed by empty line",
			content: `Document Title

Some introduction text here.

Chapter One

More text content here that is longer.`,
			want: 2, // "Document Title" and "Chapter One"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPDFHeadings(tt.content)
			if len(got) < tt.want {
				t.Errorf("extractPDFHeadings() got %d headings, want at least %d", len(got), tt.want)
			}
		})
	}
}
