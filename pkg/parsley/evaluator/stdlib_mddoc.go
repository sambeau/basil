package evaluator

import (
	"fmt"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// MdDoc represents a markdown document AST with methods for manipulation.
// Similar to Table, it wraps an underlying dictionary (the AST) and provides
// convenient methods for querying and transforming markdown documents.
type MdDoc struct {
	AST *Dictionary // The underlying AST dictionary
	Env *Environment
}

func (md *MdDoc) Type() ObjectType { return MDDOC_OBJ }
func (md *MdDoc) Inspect() string {
	// Try to get the title for a nice display
	if md.AST != nil {
		title := findTitle(md.AST, md.Env)
		if title != "" {
			if len(title) > 30 {
				title = title[:27] + "..."
			}
			return fmt.Sprintf("mdDoc(%q)", title)
		}
	}
	return "mdDoc()"
}

// loadMdDocModule returns the mdDoc module
func loadMdDocModule(env *Environment) Object {
	return &StdlibModuleDict{
		Exports: map[string]Object{
			"mdDoc": &MdDocModule{},
		},
	}
}

// MdDocModule represents the mdDoc constructor
// It can be called directly as mdDoc(text) or mdDoc(astDict)
type MdDocModule struct{}

func (mm *MdDocModule) Type() ObjectType { return BUILTIN_OBJ }
func (mm *MdDocModule) Inspect() string  { return "mdDoc" }

// evalMdDocModuleCall handles calling mdDoc() as a constructor
// Usage: mdDoc(text) - parse markdown text into an mdDoc
// Usage: mdDoc(dict) - wrap an existing AST dictionary as an mdDoc
func evalMdDocModuleCall(args []Object, env *Environment) Object {
	if len(args) < 1 {
		return newArityErrorRange("mdDoc", len(args), 1, 1)
	}

	switch arg := args[0].(type) {
	case *String:
		// Parse markdown text
		astDict := parseMarkdownToAST([]byte(arg.Value), env)
		if dict, ok := astDict.(*Dictionary); ok {
			return &MdDoc{AST: dict, Env: env}
		}
		return astDict // Return error if parsing failed
	case *Dictionary:
		// Wrap existing AST dictionary
		// Validate it looks like a markdown AST (has type field)
		if _, hasType := arg.Pairs["type"]; !hasType {
			return &Error{Message: "mdDoc: dictionary must have a 'type' field to be a valid markdown AST"}
		}
		return &MdDoc{AST: arg, Env: env}
	case *MdDoc:
		// Already an mdDoc, return as-is
		return arg
	default:
		return newTypeError("TYPE-0005", "mdDoc", "string or dictionary", arg.Type())
	}
}

// evalMdDocMethod handles method calls on MdDoc objects
func evalMdDocMethod(md *MdDoc, method string, args []Object, env *Environment) Object {
	switch method {
	// Rendering methods
	case "toMarkdown":
		return mdDocToMarkdown(md, args, env)
	case "toHTML":
		return mdDocToHTML(md, args, env)

	// Query methods
	case "findAll":
		return mdDocFindAll(md, args, env)
	case "findFirst":
		return mdDocFindFirst(md, args, env)
	case "headings":
		return mdDocHeadings(md, args, env)
	case "links":
		return mdDocLinks(md, args, env)
	case "images":
		return mdDocImages(md, args, env)
	case "codeBlocks":
		return mdDocCodeBlocks(md, args, env)

	// Convenience methods
	case "title":
		return mdDocTitle(md, args, env)
	case "toc":
		return mdDocTOC(md, args, env)
	case "text":
		return mdDocText(md, args, env)
	case "wordCount":
		return mdDocWordCount(md, args, env)

	// Transform methods
	case "walk":
		return mdDocWalk(md, args, env)
	case "map":
		return mdDocMap(md, args, env)
	case "filter":
		return mdDocFilter(md, args, env)

	// AST access
	case "ast":
		return md.AST

	default:
		return unknownMethodError(method, "mdDoc", []string{
			"toMarkdown", "toHTML",
			"findAll", "findFirst", "headings", "links", "images", "codeBlocks",
			"title", "toc", "text", "wordCount",
			"walk", "map", "filter",
			"ast",
		})
	}
}

// ============================================================================
// Rendering Methods
// ============================================================================

// mdDocToMarkdown renders the document back to markdown
// Usage: doc.toMarkdown()
func mdDocToMarkdown(md *MdDoc, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("mdDoc.toMarkdown", len(args), 0)
	}
	var buf strings.Builder
	renderMarkdownNode(&buf, md.AST, 0, md.Env)
	return &String{Value: strings.TrimSpace(buf.String())}
}

// mdDocToHTML renders the document to HTML
// Usage: doc.toHTML()
func mdDocToHTML(md *MdDoc, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("mdDoc.toHTML", len(args), 0)
	}
	var buf strings.Builder
	renderHTMLNode(&buf, md.AST, md.Env)
	return &String{Value: buf.String()}
}

// ============================================================================
// Query Methods
// ============================================================================

// mdDocFindAll finds all nodes of a given type
// Usage: doc.findAll("heading") or doc.findAll(["heading", "link"])
func mdDocFindAll(md *MdDoc, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("mdDoc.findAll", len(args), 1)
	}

	// Get types to find (can be string or array of strings)
	typesToFind := make(map[string]bool)
	switch t := args[0].(type) {
	case *String:
		typesToFind[t.Value] = true
	case *Array:
		for _, elem := range t.Elements {
			if s, ok := elem.(*String); ok {
				typesToFind[s.Value] = true
			}
		}
	default:
		return newTypeError("TYPE-0005", "mdDoc.findAll", "string or array of strings", args[0].Type())
	}

	results := make([]Object, 0)
	findAllNodes(md.AST, typesToFind, &results, md.Env)
	return &Array{Elements: results}
}

// mdDocFindFirst finds the first node of a given type
// Usage: doc.findFirst("heading")
func mdDocFindFirst(md *MdDoc, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("mdDoc.findFirst", len(args), 1)
	}

	typeStr, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0005", "mdDoc.findFirst", "string", args[0].Type())
	}

	result := findFirstNode(md.AST, typeStr.Value, md.Env)
	if result == nil {
		return NULL
	}
	return result
}

// mdDocHeadings extracts all headings with their metadata
// Usage: doc.headings()
// Returns: [{level: 1, text: "Title", id: "title"}, ...]
func mdDocHeadings(md *MdDoc, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("mdDoc.headings", len(args), 0)
	}
	results := make([]Object, 0)
	collectHeadings(md.AST, &results, md.Env)
	return &Array{Elements: results}
}

// mdDocLinks extracts all links with their metadata
// Usage: doc.links()
// Returns: [{url: "...", title: "...", text: "..."}, ...]
func mdDocLinks(md *MdDoc, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("mdDoc.links", len(args), 0)
	}
	results := make([]Object, 0)
	collectLinks(md.AST, &results, md.Env)
	return &Array{Elements: results}
}

// mdDocImages extracts all images with their metadata
// Usage: doc.images()
// Returns: [{url: "...", alt: "...", title: "..."}, ...]
func mdDocImages(md *MdDoc, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("mdDoc.images", len(args), 0)
	}
	results := make([]Object, 0)
	collectImages(md.AST, &results, md.Env)
	return &Array{Elements: results}
}

// mdDocCodeBlocks extracts all code blocks with their metadata
// Usage: doc.codeBlocks()
// Returns: [{language: "go", code: "..."}, ...]
func mdDocCodeBlocks(md *MdDoc, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("mdDoc.codeBlocks", len(args), 0)
	}
	results := make([]Object, 0)
	collectCodeBlocks(md.AST, &results, md.Env)
	return &Array{Elements: results}
}

// ============================================================================
// Convenience Methods
// ============================================================================

// mdDocTitle extracts the document title (first h1)
// Usage: doc.title()
// Returns: string or null
func mdDocTitle(md *MdDoc, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("mdDoc.title", len(args), 0)
	}
	title := findTitle(md.AST, md.Env)
	if title == "" {
		return NULL
	}
	return &String{Value: title}
}

// mdDocTOC generates a table of contents
// Usage: doc.toc() or doc.toc({minLevel: 1, maxLevel: 3})
// Returns: [{level: 1, text: "...", id: "...", indent: 0}, ...]
func mdDocTOC(md *MdDoc, args []Object, env *Environment) Object {
	if len(args) > 1 {
		return newArityErrorRange("mdDoc.toc", len(args), 0, 1)
	}

	// Default options
	minLevel := int64(1)
	maxLevel := int64(6)

	// Parse options if provided
	if len(args) == 1 {
		opts, ok := args[0].(*Dictionary)
		if !ok {
			return newTypeError("TYPE-0005", "mdDoc.toc", "options dictionary", args[0].Type())
		}
		if min := getDictInt(opts, "minLevel", md.Env); min > 0 {
			minLevel = min
		}
		if max := getDictInt(opts, "maxLevel", md.Env); max > 0 {
			maxLevel = max
		}
	}

	// Collect headings
	headings := make([]Object, 0)
	collectHeadings(md.AST, &headings, md.Env)

	// Filter by level and add indent
	results := make([]Object, 0)
	for _, h := range headings {
		heading := h.(*Dictionary)
		level := getDictInt(heading, "level", md.Env)
		if level >= minLevel && level <= maxLevel {
			tocItem := &Dictionary{
				Pairs: map[string]ast.Expression{
					"level":  &ast.ObjectLiteralExpression{Obj: &Integer{Value: level}},
					"text":   &ast.ObjectLiteralExpression{Obj: &String{Value: getDictString(heading, "text", md.Env)}},
					"id":     &ast.ObjectLiteralExpression{Obj: &String{Value: getDictString(heading, "id", md.Env)}},
					"indent": &ast.ObjectLiteralExpression{Obj: &Integer{Value: level - minLevel}},
				},
				KeyOrder: []string{"level", "text", "id", "indent"},
				Env:      md.Env,
			}
			results = append(results, tocItem)
		}
	}

	return &Array{Elements: results}
}

// mdDocText extracts all plain text from the document
// Usage: doc.text()
// Returns: string
func mdDocText(md *MdDoc, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("mdDoc.text", len(args), 0)
	}
	var buf strings.Builder
	extractPlainText(md.AST, &buf, md.Env)
	return &String{Value: buf.String()}
}

// mdDocWordCount counts words in the document
// Usage: doc.wordCount()
// Returns: integer
func mdDocWordCount(md *MdDoc, args []Object, env *Environment) Object {
	if len(args) != 0 {
		return newArityError("mdDoc.wordCount", len(args), 0)
	}
	var buf strings.Builder
	extractPlainText(md.AST, &buf, md.Env)
	text := buf.String()
	words := strings.Fields(text)
	return &Integer{Value: int64(len(words))}
}

// ============================================================================
// Transform Methods
// ============================================================================

// mdDocWalk walks the tree and calls a function on each node
// Usage: doc.walk(fn(node) { ... })
// The function receives each node but return value is ignored
func mdDocWalk(md *MdDoc, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("mdDoc.walk", len(args), 1)
	}

	fn, ok := args[0].(*Function)
	if !ok {
		return newTypeError("TYPE-0005", "mdDoc.walk", "function", args[0].Type())
	}

	walkNode(md.AST, fn, md.Env)
	return NULL
}

// mdDocMap transforms nodes by applying a function
// Usage: doc.map(fn(node) { return modifiedNode })
// Returns a new mdDoc with transformed nodes
func mdDocMap(md *MdDoc, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("mdDoc.map", len(args), 1)
	}

	fn, ok := args[0].(*Function)
	if !ok {
		return newTypeError("TYPE-0005", "mdDoc.map", "function", args[0].Type())
	}

	result := mapNode(md.AST, fn, md.Env)
	if result == nil {
		return NULL
	}
	return &MdDoc{AST: result, Env: md.Env}
}

// mdDocFilter removes nodes that don't match the predicate
// Usage: doc.filter(fn(node) { return true/false })
// Returns a new mdDoc with only nodes where fn returns true
func mdDocFilter(md *MdDoc, args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("mdDoc.filter", len(args), 1)
	}

	fn, ok := args[0].(*Function)
	if !ok {
		return newTypeError("TYPE-0005", "mdDoc.filter", "function", args[0].Type())
	}

	result := filterNode(md.AST, fn, md.Env)
	if result == nil {
		return NULL
	}
	return &MdDoc{AST: result, Env: md.Env}
}
