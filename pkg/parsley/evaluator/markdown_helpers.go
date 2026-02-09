package evaluator

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/yuin/goldmark"
	gmast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	goldmarkParser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// ============================================================================
// Markdown Helper Functions
// These functions support @std/mdDoc and were extracted from the deprecated
// @std/markdown module. They provide core markdown parsing and rendering.
// ============================================================================
func parseMarkdownToAST(source []byte, env *Environment) Object {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(goldmarkParser.WithAutoHeadingID()),
	)
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	return convertGoldmarkNode(doc, source, env)
}

// convertGoldmarkNode recursively converts a goldmark AST node to a Parsley dictionary
func convertGoldmarkNode(node gmast.Node, source []byte, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)
	keyOrder := []string{}

	// Get node type name
	nodeType := goldmarkNodeTypeName(node)
	pairs["type"] = &ast.ObjectLiteralExpression{Obj: &String{Value: nodeType}}
	keyOrder = append(keyOrder, "type")

	// Add type-specific properties
	switch n := node.(type) {
	case *gmast.Document:
		// Document is the root, no special properties

	case *gmast.Heading:
		pairs["level"] = &ast.ObjectLiteralExpression{Obj: &Integer{Value: int64(n.Level)}}
		keyOrder = append(keyOrder, "level")
		textContent := extractGoldmarkText(n, source)
		pairs["text"] = &ast.ObjectLiteralExpression{Obj: &String{Value: textContent}}
		keyOrder = append(keyOrder, "text")

		// Extract ID from Goldmark's attributes if present (set by WithAutoHeadingID)
		// Otherwise generate our own slug
		var headingID string
		if id, ok := n.AttributeString("id"); ok {
			headingID = string(id.([]byte))
		} else {
			headingID = generateSlug(textContent)
		}
		pairs["id"] = &ast.ObjectLiteralExpression{Obj: &String{Value: headingID}}
		keyOrder = append(keyOrder, "id")

	case *gmast.Paragraph:
		// Paragraph has no special properties beyond children

	case *gmast.Text:
		pairs["value"] = &ast.ObjectLiteralExpression{Obj: &String{Value: string(n.Text(source))}}
		keyOrder = append(keyOrder, "value")
		if n.SoftLineBreak() {
			pairs["softBreak"] = &ast.ObjectLiteralExpression{Obj: TRUE}
			keyOrder = append(keyOrder, "softBreak")
		}
		if n.HardLineBreak() {
			pairs["hardBreak"] = &ast.ObjectLiteralExpression{Obj: TRUE}
			keyOrder = append(keyOrder, "hardBreak")
		}

	case *gmast.String:
		pairs["value"] = &ast.ObjectLiteralExpression{Obj: &String{Value: string(n.Value)}}
		keyOrder = append(keyOrder, "value")

	case *gmast.CodeSpan:
		pairs["code"] = &ast.ObjectLiteralExpression{Obj: &String{Value: extractGoldmarkText(n, source)}}
		keyOrder = append(keyOrder, "code")

	case *gmast.Emphasis:
		pairs["level"] = &ast.ObjectLiteralExpression{Obj: &Integer{Value: int64(n.Level)}}
		keyOrder = append(keyOrder, "level")

	case *gmast.Link:
		pairs["url"] = &ast.ObjectLiteralExpression{Obj: &String{Value: string(n.Destination)}}
		keyOrder = append(keyOrder, "url")
		if len(n.Title) > 0 {
			pairs["title"] = &ast.ObjectLiteralExpression{Obj: &String{Value: string(n.Title)}}
			keyOrder = append(keyOrder, "title")
		}

	case *gmast.Image:
		pairs["url"] = &ast.ObjectLiteralExpression{Obj: &String{Value: string(n.Destination)}}
		keyOrder = append(keyOrder, "url")
		pairs["alt"] = &ast.ObjectLiteralExpression{Obj: &String{Value: extractGoldmarkText(n, source)}}
		keyOrder = append(keyOrder, "alt")
		if len(n.Title) > 0 {
			pairs["title"] = &ast.ObjectLiteralExpression{Obj: &String{Value: string(n.Title)}}
			keyOrder = append(keyOrder, "title")
		}

	case *gmast.AutoLink:
		pairs["url"] = &ast.ObjectLiteralExpression{Obj: &String{Value: string(n.URL(source))}}
		keyOrder = append(keyOrder, "url")
		pairs["protocol"] = &ast.ObjectLiteralExpression{Obj: &String{Value: string(n.Protocol)}}
		keyOrder = append(keyOrder, "protocol")

	case *gmast.RawHTML:
		var html strings.Builder
		for i := 0; i < n.Segments.Len(); i++ {
			segment := n.Segments.At(i)
			html.Write(segment.Value(source))
		}
		pairs["html"] = &ast.ObjectLiteralExpression{Obj: &String{Value: html.String()}}
		keyOrder = append(keyOrder, "html")

	case *gmast.CodeBlock:
		pairs["code"] = &ast.ObjectLiteralExpression{Obj: &String{Value: extractCodeBlockContent(n, source)}}
		keyOrder = append(keyOrder, "code")

	case *gmast.FencedCodeBlock:
		if n.Language(source) != nil {
			pairs["language"] = &ast.ObjectLiteralExpression{Obj: &String{Value: string(n.Language(source))}}
			keyOrder = append(keyOrder, "language")
		}
		pairs["code"] = &ast.ObjectLiteralExpression{Obj: &String{Value: extractCodeBlockContent(n, source)}}
		keyOrder = append(keyOrder, "code")

	case *gmast.HTMLBlock:
		var html strings.Builder
		for i := 0; i < n.Lines().Len(); i++ {
			line := n.Lines().At(i)
			html.Write(line.Value(source))
		}
		pairs["html"] = &ast.ObjectLiteralExpression{Obj: &String{Value: html.String()}}
		keyOrder = append(keyOrder, "html")

	case *gmast.List:
		pairs["ordered"] = &ast.ObjectLiteralExpression{Obj: nativeBoolToParsBoolean(n.IsOrdered())}
		keyOrder = append(keyOrder, "ordered")
		if n.IsOrdered() && n.Start != 1 {
			pairs["start"] = &ast.ObjectLiteralExpression{Obj: &Integer{Value: int64(n.Start)}}
			keyOrder = append(keyOrder, "start")
		}
		pairs["tight"] = &ast.ObjectLiteralExpression{Obj: nativeBoolToParsBoolean(n.IsTight)}
		keyOrder = append(keyOrder, "tight")

	case *gmast.ListItem:
		pairs["offset"] = &ast.ObjectLiteralExpression{Obj: &Integer{Value: int64(n.Offset)}}
		keyOrder = append(keyOrder, "offset")

	case *gmast.TextBlock:
		// TextBlock is a container, no special properties

	case *gmast.ThematicBreak:
		// Thematic break (---) has no properties

	case *gmast.Blockquote:
		// Blockquote is a container, no special properties

	// GFM extensions
	case *extast.Strikethrough:
		// Strikethrough wraps children, no special properties

	case *extast.TaskCheckBox:
		pairs["checked"] = &ast.ObjectLiteralExpression{Obj: nativeBoolToParsBoolean(n.IsChecked)}
		keyOrder = append(keyOrder, "checked")

	case *extast.Table:
		// Table is a container for rows

	case *extast.TableHeader:
		// TableHeader contains header cells

	case *extast.TableRow:
		// TableRow contains cells

	case *extast.TableCell:
		pairs["alignment"] = &ast.ObjectLiteralExpression{Obj: &String{Value: cellAlignmentToString(n.Alignment)}}
		keyOrder = append(keyOrder, "alignment")
	}

	// Add children
	children := []Object{}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		children = append(children, convertGoldmarkNode(child, source, env))
	}
	if len(children) > 0 {
		childArray := &Array{Elements: children}
		pairs["children"] = &ast.ObjectLiteralExpression{Obj: childArray}
		keyOrder = append(keyOrder, "children")
	}

	return &Dictionary{Pairs: pairs, KeyOrder: keyOrder, Env: env}
}

// goldmarkNodeTypeName returns the Parsley type name for a goldmark AST node
func goldmarkNodeTypeName(node gmast.Node) string {
	switch node.(type) {
	case *gmast.Document:
		return "document"
	case *gmast.Heading:
		return "heading"
	case *gmast.Paragraph:
		return "paragraph"
	case *gmast.Text:
		return "text"
	case *gmast.String:
		return "string"
	case *gmast.CodeSpan:
		return "code_span"
	case *gmast.Emphasis:
		return "emphasis"
	case *gmast.Link:
		return "link"
	case *gmast.Image:
		return "image"
	case *gmast.AutoLink:
		return "autolink"
	case *gmast.RawHTML:
		return "raw_html"
	case *gmast.CodeBlock:
		return "code_block"
	case *gmast.FencedCodeBlock:
		return "fenced_code_block"
	case *gmast.HTMLBlock:
		return "html_block"
	case *gmast.List:
		return "list"
	case *gmast.ListItem:
		return "list_item"
	case *gmast.TextBlock:
		return "text_block"
	case *gmast.ThematicBreak:
		return "thematic_break"
	case *gmast.Blockquote:
		return "blockquote"
	// GFM extensions
	case *extast.Strikethrough:
		return "strikethrough"
	case *extast.TaskCheckBox:
		return "task_checkbox"
	case *extast.Table:
		return "table"
	case *extast.TableHeader:
		return "table_header"
	case *extast.TableRow:
		return "table_row"
	case *extast.TableCell:
		return "table_cell"
	default:
		return "unknown"
	}
}

// extractGoldmarkText extracts all text content from a node and its children
func extractGoldmarkText(node gmast.Node, source []byte) string {
	var buf strings.Builder
	extractGoldmarkTextRecursive(node, source, &buf)
	return buf.String()
}

func extractGoldmarkTextRecursive(node gmast.Node, source []byte, buf *strings.Builder) {
	switch n := node.(type) {
	case *gmast.Text:
		buf.Write(n.Text(source))
	case *gmast.String:
		buf.Write(n.Value)
	case *gmast.CodeSpan:
		// For code spans, get the raw content
		for i := 0; i < n.ChildCount(); i++ {
			child := n.FirstChild()
			for j := 0; j < i; j++ {
				child = child.NextSibling()
			}
			extractGoldmarkTextRecursive(child, source, buf)
		}
	default:
		// Recurse into children
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			extractGoldmarkTextRecursive(child, source, buf)
		}
	}
}

// extractCodeBlockContent extracts the code content from a code block
func extractCodeBlockContent(node gmast.Node, source []byte) string {
	var buf strings.Builder
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		buf.Write(line.Value(source))
	}
	return buf.String()
}

// cellAlignmentToString converts a table cell alignment to a string
func cellAlignmentToString(alignment extast.Alignment) string {
	switch alignment {
	case extast.AlignLeft:
		return "left"
	case extast.AlignCenter:
		return "center"
	case extast.AlignRight:
		return "right"
	default:
		return "none"
	}
}

// generateSlug generates a URL-friendly slug from text
func generateSlug(text string) string {
	// Convert to lowercase
	slug := strings.ToLower(text)

	// Replace spaces and underscores with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range slug {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()

	// Collapse multiple hyphens
	re := regexp.MustCompile(`-+`)
	slug = re.ReplaceAllString(slug, "-")

	// Trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}

// getDictString safely gets a string value from a dictionary
func getDictString(dict *Dictionary, key string, env *Environment) string {
	if expr, ok := dict.Pairs[key]; ok {
		obj := Eval(expr, env)
		if str, ok := obj.(*String); ok {
			return str.Value
		}
	}
	return ""
}

// getDictInt safely gets an integer value from a dictionary
func getDictInt(dict *Dictionary, key string, env *Environment) int64 {
	if expr, ok := dict.Pairs[key]; ok {
		obj := Eval(expr, env)
		if i, ok := obj.(*Integer); ok {
			return i.Value
		}
	}
	return 0
}

// getDictBool safely gets a boolean value from a dictionary
func getDictBool(dict *Dictionary, key string, env *Environment) bool {
	if expr, ok := dict.Pairs[key]; ok {
		obj := Eval(expr, env)
		if b, ok := obj.(*Boolean); ok {
			return b.Value
		}
	}
	return false
}

// getDictChildren safely gets the children array from a dictionary
func getDictChildren(dict *Dictionary, env *Environment) []*Dictionary {
	if expr, ok := dict.Pairs["children"]; ok {
		obj := Eval(expr, env)
		if arr, ok := obj.(*Array); ok {
			children := make([]*Dictionary, 0, len(arr.Elements))
			for _, elem := range arr.Elements {
				if d, ok := elem.(*Dictionary); ok {
					children = append(children, d)
				}
			}
			return children
		}
	}
	return nil
}

// renderMarkdownNode renders a Parsley AST node back to markdown
func renderMarkdownNode(buf *strings.Builder, node *Dictionary, depth int, env *Environment) {
	if node == nil {
		return
	}

	nodeType := getDictString(node, "type", env)
	children := getDictChildren(node, env)

	switch nodeType {
	case "document":
		for i, child := range children {
			renderMarkdownNode(buf, child, depth, env)
			if i < len(children)-1 {
				buf.WriteString("\n")
			}
		}

	case "heading":
		level := getDictInt(node, "level", env)
		buf.WriteString(strings.Repeat("#", int(level)))
		buf.WriteString(" ")
		for _, child := range children {
			renderMarkdownNode(buf, child, depth, env)
		}
		buf.WriteString("\n\n")

	case "paragraph":
		for _, child := range children {
			renderMarkdownNode(buf, child, depth, env)
		}
		buf.WriteString("\n\n")

	case "text":
		buf.WriteString(getDictString(node, "value", env))

	case "string":
		buf.WriteString(getDictString(node, "value", env))

	case "code_span":
		buf.WriteString("`")
		buf.WriteString(getDictString(node, "code", env))
		buf.WriteString("`")

	case "emphasis":
		level := getDictInt(node, "level", env)
		marker := "*"
		if level == 2 {
			marker = "**"
		}
		buf.WriteString(marker)
		for _, child := range children {
			renderMarkdownNode(buf, child, depth, env)
		}
		buf.WriteString(marker)

	case "strikethrough":
		buf.WriteString("~~")
		for _, child := range children {
			renderMarkdownNode(buf, child, depth, env)
		}
		buf.WriteString("~~")

	case "link":
		buf.WriteString("[")
		for _, child := range children {
			renderMarkdownNode(buf, child, depth, env)
		}
		buf.WriteString("](")
		buf.WriteString(getDictString(node, "url", env))
		if title := getDictString(node, "title", env); title != "" {
			buf.WriteString(` "`)
			buf.WriteString(title)
			buf.WriteString(`"`)
		}
		buf.WriteString(")")

	case "image":
		buf.WriteString("![")
		buf.WriteString(getDictString(node, "alt", env))
		buf.WriteString("](")
		buf.WriteString(getDictString(node, "url", env))
		if title := getDictString(node, "title", env); title != "" {
			buf.WriteString(` "`)
			buf.WriteString(title)
			buf.WriteString(`"`)
		}
		buf.WriteString(")")

	case "autolink":
		buf.WriteString("<")
		buf.WriteString(getDictString(node, "url", env))
		buf.WriteString(">")

	case "code_block":
		code := getDictString(node, "code", env)
		// Indent each line with 4 spaces
		lines := strings.Split(code, "\n")
		for _, line := range lines {
			buf.WriteString("    ")
			buf.WriteString(line)
			buf.WriteString("\n")
		}
		buf.WriteString("\n")

	case "fenced_code_block":
		lang := getDictString(node, "language", env)
		buf.WriteString("```")
		buf.WriteString(lang)
		buf.WriteString("\n")
		buf.WriteString(getDictString(node, "code", env))
		buf.WriteString("```\n\n")

	case "html_block", "raw_html":
		buf.WriteString(getDictString(node, "html", env))
		if nodeType == "html_block" {
			buf.WriteString("\n")
		}

	case "list":
		ordered := getDictBool(node, "ordered", env)
		start := getDictInt(node, "start", env)
		if start == 0 {
			start = 1
		}
		for i, child := range children {
			if ordered {
				buf.WriteString(fmt.Sprintf("%d. ", int(start)+i))
			} else {
				buf.WriteString("- ")
			}
			renderListItem(buf, child, depth+1, ordered, env)
		}
		buf.WriteString("\n")

	case "list_item":
		// Handled by list
		for _, child := range children {
			renderMarkdownNode(buf, child, depth, env)
		}

	case "text_block":
		for _, child := range children {
			renderMarkdownNode(buf, child, depth, env)
		}

	case "thematic_break":
		buf.WriteString("---\n\n")

	case "blockquote":
		// Render children with > prefix
		var quoteBuf strings.Builder
		for _, child := range children {
			renderMarkdownNode(&quoteBuf, child, depth, env)
		}
		lines := strings.Split(strings.TrimSuffix(quoteBuf.String(), "\n"), "\n")
		for _, line := range lines {
			buf.WriteString("> ")
			buf.WriteString(line)
			buf.WriteString("\n")
		}
		buf.WriteString("\n")

	case "task_checkbox":
		if getDictBool(node, "checked", env) {
			buf.WriteString("[x] ")
		} else {
			buf.WriteString("[ ] ")
		}

	case "table":
		renderMarkdownTable(buf, children, env)

	default:
		// For unknown types, just render children
		for _, child := range children {
			renderMarkdownNode(buf, child, depth, env)
		}
	}
}

// renderListItem renders a list item with proper indentation
func renderListItem(buf *strings.Builder, node *Dictionary, depth int, ordered bool, env *Environment) {
	children := getDictChildren(node, env)

	// First, check if the item contains a task checkbox
	for i, child := range children {
		childType := getDictString(child, "type", env)
		if childType == "task_checkbox" {
			if getDictBool(child, "checked", env) {
				buf.WriteString("[x] ")
			} else {
				buf.WriteString("[ ] ")
			}
			// Render remaining children
			for j := i + 1; j < len(children); j++ {
				renderListItemContent(buf, children[j], depth, env)
			}
			buf.WriteString("\n")
			return
		}
	}

	// No task checkbox, render normally
	for _, child := range children {
		renderListItemContent(buf, child, depth, env)
	}
	buf.WriteString("\n")
}

// renderListItemContent renders the content of a list item
func renderListItemContent(buf *strings.Builder, node *Dictionary, depth int, env *Environment) {
	nodeType := getDictString(node, "type", env)
	children := getDictChildren(node, env)

	switch nodeType {
	case "paragraph", "text_block":
		for _, child := range children {
			renderMarkdownNode(buf, child, 0, env)
		}
	case "list":
		buf.WriteString("\n")
		indent := strings.Repeat("  ", depth)
		ordered := getDictBool(node, "ordered", env)
		for i, child := range children {
			buf.WriteString(indent)
			if ordered {
				buf.WriteString(fmt.Sprintf("%d. ", i+1))
			} else {
				buf.WriteString("- ")
			}
			renderListItem(buf, child, depth+1, ordered, env)
		}
	default:
		renderMarkdownNode(buf, node, depth, env)
	}
}

// renderMarkdownTable renders a GFM table
func renderMarkdownTable(buf *strings.Builder, children []*Dictionary, env *Environment) {
	if len(children) == 0 {
		return
	}

	// First child should be table_header
	var header *Dictionary
	var rows []*Dictionary

	for _, child := range children {
		childType := getDictString(child, "type", env)
		if childType == "table_header" {
			header = child
		} else if childType == "table_row" {
			rows = append(rows, child)
		}
	}

	if header == nil {
		return
	}

	// Render header
	headerCells := getDictChildren(header, env)
	alignments := make([]string, len(headerCells))

	buf.WriteString("|")
	for i, cell := range headerCells {
		buf.WriteString(" ")
		renderTableCellContent(buf, cell, env)
		buf.WriteString(" |")
		alignments[i] = getDictString(cell, "alignment", env)
	}
	buf.WriteString("\n")

	// Render separator
	buf.WriteString("|")
	for _, align := range alignments {
		switch align {
		case "left":
			buf.WriteString(":---|")
		case "center":
			buf.WriteString(":---:|")
		case "right":
			buf.WriteString("---:|")
		default:
			buf.WriteString("---|")
		}
	}
	buf.WriteString("\n")

	// Render rows
	for _, row := range rows {
		cells := getDictChildren(row, env)
		buf.WriteString("|")
		for _, cell := range cells {
			buf.WriteString(" ")
			renderTableCellContent(buf, cell, env)
			buf.WriteString(" |")
		}
		buf.WriteString("\n")
	}
	buf.WriteString("\n")
}

// renderTableCellContent renders the content of a table cell
func renderTableCellContent(buf *strings.Builder, cell *Dictionary, env *Environment) {
	children := getDictChildren(cell, env)
	for _, child := range children {
		renderMarkdownNode(buf, child, 0, env)
	}
}

// renderHTMLNode renders a Parsley AST node to HTML
func renderHTMLNode(buf *strings.Builder, node *Dictionary, env *Environment) {
	if node == nil {
		return
	}

	nodeType := getDictString(node, "type", env)
	children := getDictChildren(node, env)

	switch nodeType {
	case "document":
		for _, child := range children {
			renderHTMLNode(buf, child, env)
		}

	case "heading":
		level := getDictInt(node, "level", env)
		id := getDictString(node, "id", env)
		buf.WriteString(fmt.Sprintf("<h%d", level))
		if id != "" {
			buf.WriteString(fmt.Sprintf(` id="%s"`, id))
		}
		buf.WriteString(">")
		for _, child := range children {
			renderHTMLNode(buf, child, env)
		}
		buf.WriteString(fmt.Sprintf("</h%d>\n", level))

	case "paragraph":
		buf.WriteString("<p>")
		for _, child := range children {
			renderHTMLNode(buf, child, env)
		}
		buf.WriteString("</p>\n")

	case "text":
		buf.WriteString(htmlEscape(getDictString(node, "value", env)))
		if getDictBool(node, "hardBreak", env) {
			buf.WriteString("<br/>\n")
		}

	case "string":
		buf.WriteString(htmlEscape(getDictString(node, "value", env)))

	case "code_span":
		buf.WriteString("<code>")
		buf.WriteString(htmlEscape(getDictString(node, "code", env)))
		buf.WriteString("</code>")

	case "emphasis":
		level := getDictInt(node, "level", env)
		tag := "em"
		if level == 2 {
			tag = "strong"
		}
		buf.WriteString("<")
		buf.WriteString(tag)
		buf.WriteString(">")
		for _, child := range children {
			renderHTMLNode(buf, child, env)
		}
		buf.WriteString("</")
		buf.WriteString(tag)
		buf.WriteString(">")

	case "strikethrough":
		buf.WriteString("<del>")
		for _, child := range children {
			renderHTMLNode(buf, child, env)
		}
		buf.WriteString("</del>")

	case "link":
		buf.WriteString(`<a href="`)
		buf.WriteString(htmlEscape(getDictString(node, "url", env)))
		buf.WriteString(`"`)
		if title := getDictString(node, "title", env); title != "" {
			buf.WriteString(` title="`)
			buf.WriteString(htmlEscape(title))
			buf.WriteString(`"`)
		}
		buf.WriteString(`>`)
		for _, child := range children {
			renderHTMLNode(buf, child, env)
		}
		buf.WriteString("</a>")

	case "image":
		buf.WriteString(`<img src="`)
		buf.WriteString(htmlEscape(getDictString(node, "url", env)))
		buf.WriteString(`" alt="`)
		buf.WriteString(htmlEscape(getDictString(node, "alt", env)))
		buf.WriteString(`"`)
		if title := getDictString(node, "title", env); title != "" {
			buf.WriteString(` title="`)
			buf.WriteString(htmlEscape(title))
			buf.WriteString(`"`)
		}
		buf.WriteString(`/>`)

	case "autolink":
		url := getDictString(node, "url", env)
		buf.WriteString(`<a href="`)
		buf.WriteString(htmlEscape(url))
		buf.WriteString(`">`)
		buf.WriteString(htmlEscape(url))
		buf.WriteString("</a>")

	case "code_block", "fenced_code_block":
		lang := getDictString(node, "language", env)
		buf.WriteString("<pre><code")
		if lang != "" {
			buf.WriteString(` class="language-`)
			buf.WriteString(htmlEscape(lang))
			buf.WriteString(`"`)
		}
		buf.WriteString(">")
		buf.WriteString(htmlEscape(getDictString(node, "code", env)))
		buf.WriteString("</code></pre>\n")

	case "html_block", "raw_html":
		// Pass through raw HTML
		buf.WriteString(getDictString(node, "html", env))

	case "list":
		ordered := getDictBool(node, "ordered", env)
		if ordered {
			start := getDictInt(node, "start", env)
			if start != 1 && start != 0 {
				buf.WriteString(fmt.Sprintf(`<ol start="%d">`, start))
			} else {
				buf.WriteString("<ol>")
			}
		} else {
			buf.WriteString("<ul>")
		}
		buf.WriteString("\n")
		for _, child := range children {
			renderHTMLNode(buf, child, env)
		}
		if ordered {
			buf.WriteString("</ol>\n")
		} else {
			buf.WriteString("</ul>\n")
		}

	case "list_item":
		buf.WriteString("<li>")
		// Check for task checkbox
		hasCheckbox := false
		for _, child := range children {
			if getDictString(child, "type", env) == "task_checkbox" {
				hasCheckbox = true
				if getDictBool(child, "checked", env) {
					buf.WriteString(`<input type="checkbox" checked disabled/> `)
				} else {
					buf.WriteString(`<input type="checkbox" disabled/> `)
				}
			}
		}
		for _, child := range children {
			if getDictString(child, "type", env) != "task_checkbox" || !hasCheckbox {
				renderHTMLNode(buf, child, env)
			}
		}
		buf.WriteString("</li>\n")

	case "text_block":
		for _, child := range children {
			renderHTMLNode(buf, child, env)
		}

	case "thematic_break":
		buf.WriteString("<hr/>\n")

	case "blockquote":
		buf.WriteString("<blockquote>\n")
		for _, child := range children {
			renderHTMLNode(buf, child, env)
		}
		buf.WriteString("</blockquote>\n")

	case "task_checkbox":
		// Handled in list_item

	case "table":
		buf.WriteString("<table>\n")
		for _, child := range children {
			renderHTMLNode(buf, child, env)
		}
		buf.WriteString("</table>\n")

	case "table_header":
		buf.WriteString("<thead>\n<tr>\n")
		for _, cell := range children {
			align := getDictString(cell, "alignment", env)
			buf.WriteString("<th")
			if align != "" && align != "none" {
				buf.WriteString(fmt.Sprintf(` style="text-align: %s"`, align))
			}
			buf.WriteString(">")
			renderHTMLNode(buf, cell, env)
			buf.WriteString("</th>\n")
		}
		buf.WriteString("</tr>\n</thead>\n")

	case "table_row":
		buf.WriteString("<tr>\n")
		for _, cell := range children {
			align := getDictString(cell, "alignment", env)
			buf.WriteString("<td")
			if align != "" && align != "none" {
				buf.WriteString(fmt.Sprintf(` style="text-align: %s"`, align))
			}
			buf.WriteString(">")
			renderHTMLNode(buf, cell, env)
			buf.WriteString("</td>\n")
		}
		buf.WriteString("</tr>\n")

	case "table_cell":
		for _, child := range children {
			renderHTMLNode(buf, child, env)
		}

	default:
		// For unknown types, just render children
		for _, child := range children {
			renderHTMLNode(buf, child, env)
		}
	}
}

// ============================================================================
// Query Helpers (used by @std/mdDoc methods in stdlib_mddoc.go)
// ============================================================================

// findAllNodes recursively finds all nodes matching the given types
func findAllNodes(node *Dictionary, types map[string]bool, results *[]Object, env *Environment) {
	nodeType := getDictString(node, "type", env)
	if types[nodeType] {
		*results = append(*results, node)
	}

	children := getDictChildren(node, env)
	for _, child := range children {
		findAllNodes(child, types, results, env)
	}
}

// findFirstNode recursively finds the first node of a given type
func findFirstNode(node *Dictionary, nodeType string, env *Environment) *Dictionary {
	if getDictString(node, "type", env) == nodeType {
		return node
	}

	children := getDictChildren(node, env)
	for _, child := range children {
		if result := findFirstNode(child, nodeType, env); result != nil {
			return result
		}
	}
	return nil
}

// collectHeadings recursively collects all headings
func collectHeadings(node *Dictionary, results *[]Object, env *Environment) {
	nodeType := getDictString(node, "type", env)
	if nodeType == "heading" {
		heading := &Dictionary{
			Pairs: map[string]ast.Expression{
				"level": &ast.ObjectLiteralExpression{Obj: &Integer{Value: getDictInt(node, "level", env)}},
				"text":  &ast.ObjectLiteralExpression{Obj: &String{Value: getDictString(node, "text", env)}},
				"id":    &ast.ObjectLiteralExpression{Obj: &String{Value: getDictString(node, "id", env)}},
			},
			KeyOrder: []string{"level", "text", "id"},
			Env:      env,
		}
		*results = append(*results, heading)
	}

	children := getDictChildren(node, env)
	for _, child := range children {
		collectHeadings(child, results, env)
	}
}

// collectLinks recursively collects all links
func collectLinks(node *Dictionary, results *[]Object, env *Environment) {
	nodeType := getDictString(node, "type", env)
	if nodeType == "link" || nodeType == "autolink" {
		// Extract link text from children
		var textBuf strings.Builder
		extractPlainText(node, &textBuf, env)

		link := &Dictionary{
			Pairs: map[string]ast.Expression{
				"url":   &ast.ObjectLiteralExpression{Obj: &String{Value: getDictString(node, "url", env)}},
				"title": &ast.ObjectLiteralExpression{Obj: &String{Value: getDictString(node, "title", env)}},
				"text":  &ast.ObjectLiteralExpression{Obj: &String{Value: textBuf.String()}},
			},
			KeyOrder: []string{"url", "title", "text"},
			Env:      env,
		}
		*results = append(*results, link)
	}

	children := getDictChildren(node, env)
	for _, child := range children {
		collectLinks(child, results, env)
	}
}

// collectImages recursively collects all images
func collectImages(node *Dictionary, results *[]Object, env *Environment) {
	nodeType := getDictString(node, "type", env)
	if nodeType == "image" {
		image := &Dictionary{
			Pairs: map[string]ast.Expression{
				"url":   &ast.ObjectLiteralExpression{Obj: &String{Value: getDictString(node, "url", env)}},
				"alt":   &ast.ObjectLiteralExpression{Obj: &String{Value: getDictString(node, "alt", env)}},
				"title": &ast.ObjectLiteralExpression{Obj: &String{Value: getDictString(node, "title", env)}},
			},
			KeyOrder: []string{"url", "alt", "title"},
			Env:      env,
		}
		*results = append(*results, image)
	}

	children := getDictChildren(node, env)
	for _, child := range children {
		collectImages(child, results, env)
	}
}

// collectCodeBlocks recursively collects all code blocks
func collectCodeBlocks(node *Dictionary, results *[]Object, env *Environment) {
	nodeType := getDictString(node, "type", env)
	if nodeType == "code_block" || nodeType == "fenced_code_block" {
		codeBlock := &Dictionary{
			Pairs: map[string]ast.Expression{
				"language": &ast.ObjectLiteralExpression{Obj: &String{Value: getDictString(node, "language", env)}},
				"code":     &ast.ObjectLiteralExpression{Obj: &String{Value: getDictString(node, "code", env)}},
			},
			KeyOrder: []string{"language", "code"},
			Env:      env,
		}
		*results = append(*results, codeBlock)
	}

	children := getDictChildren(node, env)
	for _, child := range children {
		collectCodeBlocks(child, results, env)
	}
}

// ============================================================================
// Convenience Helpers (used by @std/mdDoc methods in stdlib_mddoc.go)
// ============================================================================

// findTitle finds the first h1 heading text
func findTitle(node *Dictionary, env *Environment) string {
	nodeType := getDictString(node, "type", env)
	if nodeType == "heading" && getDictInt(node, "level", env) == 1 {
		return getDictString(node, "text", env)
	}

	children := getDictChildren(node, env)
	for _, child := range children {
		if title := findTitle(child, env); title != "" {
			return title
		}
	}
	return ""
}

// extractPlainText recursively extracts all plain text from a node
func extractPlainText(node *Dictionary, buf *strings.Builder, env *Environment) {
	nodeType := getDictString(node, "type", env)

	switch nodeType {
	case "text", "string":
		buf.WriteString(getDictString(node, "value", env))
	case "code_span":
		buf.WriteString(getDictString(node, "code", env))
	case "code_block", "fenced_code_block":
		buf.WriteString(getDictString(node, "code", env))
	case "heading":
		// Use pre-extracted text for headings
		buf.WriteString(getDictString(node, "text", env))
		buf.WriteString(" ")
		return // Don't recurse into children for headings
	default:
		// Add spacing for block elements
		switch nodeType {
		case "paragraph", "list_item", "blockquote":
			if buf.Len() > 0 {
				buf.WriteString(" ")
			}
		}
	}

	children := getDictChildren(node, env)
	for _, child := range children {
		extractPlainText(child, buf, env)
	}
}

// ============================================================================
// Transform Helpers (used by @std/mdDoc methods in stdlib_mddoc.go)
// ============================================================================

// walkNode recursively walks the tree calling fn on each node
func walkNode(node *Dictionary, fn *Function, env *Environment) {
	// Call the function with this node
	extendedEnv := extendFunctionEnv(fn, []Object{node})
	for _, stmt := range fn.Body.Statements {
		Eval(stmt, extendedEnv)
	}

	// Recurse into children
	children := getDictChildren(node, env)
	for _, child := range children {
		walkNode(child, fn, env)
	}
}

// mapNode recursively transforms nodes
func mapNode(node *Dictionary, fn *Function, env *Environment) *Dictionary {
	// First, transform children
	children := getDictChildren(node, env)
	newChildren := make([]Object, 0, len(children))
	for _, child := range children {
		if mapped := mapNode(child, fn, env); mapped != nil {
			newChildren = append(newChildren, mapped)
		}
	}

	// Create new node with transformed children
	newPairs := make(map[string]ast.Expression)
	newKeyOrder := make([]string, 0)
	for _, key := range node.KeyOrder {
		if key == "children" {
			continue
		}
		if expr, ok := node.Pairs[key]; ok {
			newPairs[key] = expr
			newKeyOrder = append(newKeyOrder, key)
		}
	}
	if len(newChildren) > 0 {
		newPairs["children"] = &ast.ObjectLiteralExpression{Obj: &Array{Elements: newChildren}}
		newKeyOrder = append(newKeyOrder, "children")
	}

	newNode := &Dictionary{
		Pairs:    newPairs,
		KeyOrder: newKeyOrder,
		Env:      env,
	}

	// Call the transform function
	extendedEnv := extendFunctionEnv(fn, []Object{newNode})
	var result Object
	for _, stmt := range fn.Body.Statements {
		result = Eval(stmt, extendedEnv)
	}

	// If function returns a dictionary, use it; otherwise keep the node
	if dict, ok := result.(*Dictionary); ok {
		return dict
	}
	return newNode
}

// filterNode recursively filters nodes
func filterNode(node *Dictionary, fn *Function, env *Environment) *Dictionary {
	// Check if this node passes the filter
	extendedEnv := extendFunctionEnv(fn, []Object{node})
	var result Object
	for _, stmt := range fn.Body.Statements {
		result = Eval(stmt, extendedEnv)
	}

	if !isTruthy(result) {
		return nil
	}

	// Filter children
	children := getDictChildren(node, env)
	newChildren := make([]Object, 0, len(children))
	for _, child := range children {
		if filtered := filterNode(child, fn, env); filtered != nil {
			newChildren = append(newChildren, filtered)
		}
	}

	// Create new node with filtered children
	newPairs := make(map[string]ast.Expression)
	newKeyOrder := make([]string, 0)
	for _, key := range node.KeyOrder {
		if key == "children" {
			continue
		}
		if expr, ok := node.Pairs[key]; ok {
			newPairs[key] = expr
			newKeyOrder = append(newKeyOrder, key)
		}
	}
	if len(newChildren) > 0 {
		newPairs["children"] = &ast.ObjectLiteralExpression{Obj: &Array{Elements: newChildren}}
		newKeyOrder = append(newKeyOrder, "children")
	}

	return &Dictionary{
		Pairs:    newPairs,
		KeyOrder: newKeyOrder,
		Env:      env,
	}
}
