package search

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// MaxDOCXSize is the maximum file size we'll attempt to process (50MB)
	MaxDOCXSize = 50 * 1024 * 1024

	// XML namespaces used in DOCX
	nsWordML  = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	nsDCTerms = "http://purl.org/dc/terms/"
	nsDC      = "http://purl.org/dc/elements/1.1/"
)

// DOCX XML structures for parsing word/document.xml
// We only extract text content, ignoring formatting, images, etc.
// Note: DOCX uses namespaced XML (w: prefix = WordprocessingML namespace)

type docxDocument struct {
	XMLName xml.Name `xml:"document"`
	Body    docxBody `xml:"body"`
}

type docxBody struct {
	Paragraphs []docxParagraph `xml:"p"`
	Tables     []docxTable     `xml:"tbl"`
}

type docxParagraph struct {
	Properties docxParagraphProps `xml:"pPr"`
	Runs       []docxRun          `xml:"r"`
}

type docxParagraphProps struct {
	Style docxParagraphStyle `xml:"pStyle"`
}

type docxParagraphStyle struct {
	Val string `xml:"val,attr"`
}

type docxRun struct {
	Text []docxText `xml:"t"`
}

type docxText struct {
	Content string `xml:",chardata"`
}

type docxTable struct {
	Rows []docxTableRow `xml:"tr"`
}

type docxTableRow struct {
	Cells []docxTableCell `xml:"tc"`
}

type docxTableCell struct {
	Paragraphs []docxParagraph `xml:"p"`
}

// DOCX core properties (docProps/core.xml)
type docxCoreProps struct {
	Title    string `xml:"title"`
	Subject  string `xml:"subject"`
	Creator  string `xml:"creator"`
	Keywords string `xml:"keywords"`
	Created  string `xml:"created"`
	Modified string `xml:"modified"`
}

// ProcessDOCX extracts text content from a DOCX file and returns a Document for indexing.
// It extracts the title from document properties or first heading, and all text content.
func ProcessDOCX(filePath string, mtime time.Time) (*Document, error) {
	// Check file size first
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot stat file: %w", err)
	}
	if info.Size() > MaxDOCXSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", info.Size(), MaxDOCXSize)
	}

	// Open the ZIP archive
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open DOCX file: %w", err)
	}
	defer r.Close()

	var docContent string
	var headings []string
	var props *docxCoreProps

	// Process files in the archive
	for _, f := range r.File {
		switch f.Name {
		case "word/document.xml":
			content, heads, err := extractDocumentContent(f)
			if err != nil {
				return nil, fmt.Errorf("error extracting document content: %w", err)
			}
			docContent = content
			headings = heads

		case "docProps/core.xml":
			p, err := extractCoreProperties(f)
			if err != nil {
				// Non-fatal: properties are optional
				continue
			}
			props = p
		}
	}

	if docContent == "" {
		return nil, fmt.Errorf("no document content found in DOCX")
	}

	// Generate URL from file path
	url := GenerateURL(filePath)

	// Determine title (priority: doc properties > first heading > filename)
	title := ""
	if props != nil && props.Title != "" {
		title = props.Title
	}
	if title == "" && len(headings) > 0 {
		title = headings[0]
	}
	if title == "" {
		// Fall back to filename
		base := filepath.Base(filePath)
		title = strings.TrimSuffix(base, filepath.Ext(base))
		title = strings.ReplaceAll(title, "-", " ")
		title = strings.ReplaceAll(title, "_", " ")
	}

	// Extract tags from keywords if available
	var tags []string
	if props != nil && props.Keywords != "" {
		// Keywords are typically comma or semicolon separated
		for _, kw := range strings.FieldsFunc(props.Keywords, func(r rune) bool {
			return r == ',' || r == ';'
		}) {
			kw = strings.TrimSpace(kw)
			if kw != "" {
				tags = append(tags, kw)
			}
		}
	}

	// Parse date from properties if available
	var docDate time.Time
	if props != nil && props.Modified != "" {
		if t, err := time.Parse(time.RFC3339, props.Modified); err == nil {
			docDate = t
		}
	}
	if docDate.IsZero() && props != nil && props.Created != "" {
		if t, err := time.Parse(time.RFC3339, props.Created); err == nil {
			docDate = t
		}
	}

	doc := &Document{
		URL:      url,
		Title:    title,
		Content:  docContent,
		Tags:     tags,
		Headings: strings.Join(headings, "\n"),
		Date:     docDate,
		Path:     filePath,
		Mtime:    mtime.Unix(),
	}

	return doc, nil
}

// extractDocumentContent parses word/document.xml and extracts text content.
// Returns the full text content and a list of headings.
func extractDocumentContent(f *zip.File) (string, []string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", nil, err
	}
	defer rc.Close()

	// Limit reader to prevent zip bombs
	limitedReader := io.LimitReader(rc, MaxDOCXSize)

	var doc docxDocument
	decoder := xml.NewDecoder(limitedReader)
	if err := decoder.Decode(&doc); err != nil {
		return "", nil, fmt.Errorf("error parsing document XML: %w", err)
	}

	var paragraphs []string
	var headings []string

	// Extract text from paragraphs
	for _, p := range doc.Body.Paragraphs {
		text := extractParagraphText(p)
		if text != "" {
			paragraphs = append(paragraphs, text)

			// Check if this is a heading style
			style := strings.ToLower(p.Properties.Style.Val)
			if strings.HasPrefix(style, "heading") || strings.HasPrefix(style, "title") {
				headings = append(headings, text)
			}
		}
	}

	// Extract text from tables
	for _, table := range doc.Body.Tables {
		for _, row := range table.Rows {
			var cellTexts []string
			for _, cell := range row.Cells {
				for _, p := range cell.Paragraphs {
					text := extractParagraphText(p)
					if text != "" {
						cellTexts = append(cellTexts, text)
					}
				}
			}
			if len(cellTexts) > 0 {
				paragraphs = append(paragraphs, strings.Join(cellTexts, " | "))
			}
		}
	}

	return strings.Join(paragraphs, "\n\n"), headings, nil
}

// extractParagraphText extracts all text from a paragraph's runs
func extractParagraphText(p docxParagraph) string {
	var texts []string
	for _, run := range p.Runs {
		for _, t := range run.Text {
			if t.Content != "" {
				texts = append(texts, t.Content)
			}
		}
	}
	return strings.TrimSpace(strings.Join(texts, ""))
}

// extractCoreProperties parses docProps/core.xml for document metadata
func extractCoreProperties(f *zip.File) (*docxCoreProps, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var props docxCoreProps
	decoder := xml.NewDecoder(rc)
	if err := decoder.Decode(&props); err != nil {
		return nil, err
	}

	return &props, nil
}

// IsDOCX checks if a file path has a DOCX extension
func IsDOCX(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".docx"
}
