package evaluator

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/sambeau/basil/pkg/parsley/ast"
	perrors "github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// Tag evaluation functions: evalTagLiteral, evalTagPair, evalCacheTag, evalPartTag,
// evalStandardTagPair, evalCustomTagPair, evalTagContents, evalTagContentsAsArray,
// evalSQLTag, evalTagProps, evalStandardTag, evalCustomTag

func evalTagLiteral(node *ast.TagLiteral, env *Environment) Object {
	raw := node.Raw

	// Parse tag name (first word)
	i := 0
	for i < len(raw) && !unicode.IsSpace(rune(raw[i])) {
		i++
	}
	tagName := raw[:i]
	rest := raw[i:]

	// Special handling for Part component
	if tagName == "Part" {
		return evalPartTag(node.Token, rest, env)
	}

	// Check if it's a custom tag (starts with uppercase)
	isCustom := len(tagName) > 0 && unicode.IsUpper(rune(tagName[0]))

	if isCustom {
		// Custom tag - call function with props dictionary
		return evalCustomTag(node.Token, tagName, rest, env)
	} else {
		// Standard tag - return as interpolated string
		return evalStandardTag(node, tagName, rest, env)
	}
}

// evalTagPair evaluates a paired tag like <div>content</div> or <Component>content</Component>
func evalTagPair(node *ast.TagPairExpression, env *Environment) Object {
	// Empty grouping tag <> just returns its contents
	if node.Name == "" {
		return evalTagContents(node.Contents, env)
	}

	// Special handling for basil.cache.Cache component
	if node.Name == "basil.cache.Cache" {
		return evalCacheTag(node, env)
	}

	// Check if it's a custom component (starts with uppercase)
	isCustom := len(node.Name) > 0 && unicode.IsUpper(rune(node.Name[0]))

	if isCustom {
		// Custom component - call function with props dictionary including contents
		return evalCustomTagPair(node, env)
	} else {
		// Standard tag - return as HTML string
		return evalStandardTagPair(node, env)
	}
}

// evalCacheTag handles the <basil.cache.Cache> component for fragment caching.
// It short-circuits child evaluation on cache hit, or caches the result on miss.
func evalCacheTag(node *ast.TagPairExpression, env *Environment) Object {
	// Parse props to get key and maxAge
	propsCol := node.Token.Column + 1 + len(node.Name) + 1
	propsDict := parseTagProps(node.Props, env, node.Token.Line, propsCol)
	if isError(propsDict) {
		return propsDict
	}
	props := propsDict.(*Dictionary)

	// Extract 'key' attribute (required)
	keyExpr, hasKey := props.Pairs["key"]
	if !hasKey {
		return &Error{
			Class:   ClassValue,
			Code:    "CACHE-0001",
			Message: "Cache component requires 'key' attribute",
			Hints:   []string{"Add a key attribute: <basil.cache.Cache key=\"sidebar\">"},
			Line:    node.Token.Line,
			Column:  node.Token.Column,
		}
	}
	keyObj := Eval(keyExpr, env)
	if isError(keyObj) {
		return keyObj
	}
	keyStr, ok := keyObj.(*String)
	if !ok {
		return &Error{
			Class:   ClassType,
			Code:    "CACHE-0002",
			Message: "Cache key must be a string",
			Hints:   []string{"Use a string value for key: key=\"sidebar\" or key={\"user-\" + id}"},
			Line:    node.Token.Line,
			Column:  node.Token.Column,
		}
	}

	// Extract 'maxAge' attribute (required)
	maxAgeExpr, hasMaxAge := props.Pairs["maxAge"]
	if !hasMaxAge {
		return &Error{
			Class:   ClassValue,
			Code:    "CACHE-0003",
			Message: "Cache component requires 'maxAge' attribute",
			Hints:   []string{"Add a duration: <basil.cache.Cache key=\"sidebar\" maxAge={@1h}>"},
			Line:    node.Token.Line,
			Column:  node.Token.Column,
		}
	}
	maxAgeObj := Eval(maxAgeExpr, env)
	if isError(maxAgeObj) {
		return maxAgeObj
	}

	// Duration is represented as a Dictionary with __type="duration", months, seconds
	maxAgeDict, ok := maxAgeObj.(*Dictionary)
	if !ok || !isDurationDict(maxAgeDict) {
		return &Error{
			Class:   ClassType,
			Code:    "CACHE-0004",
			Message: "Cache maxAge must be a duration",
			Hints:   []string{"Use a duration literal: maxAge={@1h} or maxAge={@15m}"},
			Line:    node.Token.Line,
			Column:  node.Token.Column,
		}
	}

	// Extract duration components (months not used for cache TTL, only seconds)
	_, seconds, err := getDurationComponents(maxAgeDict, env)
	if err != nil {
		return &Error{
			Class:   ClassValue,
			Code:    "CACHE-0005",
			Message: fmt.Sprintf("Invalid duration: %v", err),
			Line:    node.Token.Line,
			Column:  node.Token.Column,
		}
	}
	maxAgeDuration := time.Duration(seconds) * time.Second

	// Extract optional 'enabled' attribute (defaults to true)
	enabled := true
	if enabledExpr, hasEnabled := props.Pairs["enabled"]; hasEnabled {
		enabledObj := Eval(enabledExpr, env)
		if isError(enabledObj) {
			return enabledObj
		}
		if enabledBool, ok := enabledObj.(*Boolean); ok {
			enabled = enabledBool.Value
		}
	}

	// Build full cache key: {handler_path}:{user_key}
	fullKey := env.HandlerPath + ":" + keyStr.Value

	// Check if caching is disabled (dev mode or enabled=false)
	if env.DevMode || !enabled {
		// Dev mode: skip caching and evaluate children normally
		// TODO: Add dev logging when DevLogWriter interface supports cache logging
		return evalTagContents(node.Contents, env)
	}

	// Check if fragment cache is available
	if env.FragmentCache == nil {
		// No cache available, evaluate children normally
		return evalTagContents(node.Contents, env)
	}

	// Check cache for hit
	if cached, hit := env.FragmentCache.Get(fullKey); hit {
		// Cache hit - return cached HTML without evaluating children
		return &String{Value: cached}
	}

	// Cache miss - evaluate children
	result := evalTagContents(node.Contents, env)
	if isError(result) {
		return result
	}

	// Get HTML string from result
	html := objectToTemplateString(result)

	// Store in cache
	env.FragmentCache.Set(fullKey, html, maxAgeDuration)

	return &String{Value: html}
}

// evalPartTag handles the <Part /> component for reloadable HTML fragments.
// It loads a Part module, calls the specified view function, and wraps the result
// with data attributes for JavaScript runtime interactivity.
func evalPartTag(token lexer.Token, propsStr string, env *Environment) Object {
	// Parse props to extract src, view, and additional props
	// Note: propsStr is everything after "Part " in the tag, so offset is len("Part ")
	propsCol := token.Column + 1 + 4 + 1 // "<" + "Part" + " "
	propsDict := parseTagProps(propsStr, env, token.Line, propsCol)
	if isError(propsDict) {
		return propsDict
	}
	props := propsDict.(*Dictionary)

	// Extract 'src' attribute (required - path to Part module)
	srcExpr, hasSrc := props.Pairs["src"]
	if !hasSrc {
		return &Error{
			Class:   ClassValue,
			Code:    "PART-0002",
			Message: "Part component requires 'src' attribute",
			Hints:   []string{"Add a src attribute: <Part src={@./counter.part}/>", "The src should be a path to a .part file"},
			Line:    token.Line,
			Column:  token.Column,
		}
	}

	// Evaluate src to get the path
	srcObj := Eval(srcExpr, env)
	if isError(srcObj) {
		return srcObj
	}

	// Extract path string from path dictionary or string
	var pathStr string
	switch src := srcObj.(type) {
	case *Dictionary:
		// Handle path literal (@./file.part)
		if typeExpr, ok := src.Pairs["__type"]; ok {
			typeVal := Eval(typeExpr, src.Env)
			if typeStr, ok := typeVal.(*String); ok && typeStr.Value == "path" {
				pathStr = pathDictToString(src)
			} else {
				return &Error{
					Class:   ClassType,
					Code:    "PART-0003",
					Message: "Part src must be a path or string",
					Hints:   []string{"Use a path literal: src=@./counter.part", "Or a string: src=\"./counter.part\""},
					Line:    token.Line,
					Column:  token.Column,
				}
			}
		} else {
			return &Error{
				Class:   ClassType,
				Code:    "PART-0003",
				Message: "Part src must be a path or string",
				Hints:   []string{"Use a path literal: src={@./counter.part}", "Or a string: src=\"./counter.part\""},
				Line:    token.Line,
				Column:  token.Column,
			}
		}
	case *String:
		pathStr = src.Value
	default:
		return &Error{
			Class:   ClassType,
			Code:    "PART-0003",
			Message: fmt.Sprintf("Part src must be a path or string, got %s", src.Type()),
			Line:    token.Line,
			Column:  token.Column,
		}
	}

	// Verify path ends with .part
	if !strings.HasSuffix(pathStr, ".part") {
		return &Error{
			Class:   ClassValue,
			Code:    "PART-0004",
			Message: fmt.Sprintf("Part src must reference a .part file, got: %s", pathStr),
			Hints:   []string{"Ensure your Part file has a .part extension"},
			Line:    token.Line,
			Column:  token.Column,
		}
	}

	// Import the Part module
	partModule := importModule(pathStr, env)
	if isError(partModule) {
		// Enrich error with position information from the Part tag
		if err, ok := partModule.(*Error); ok {
			if err.Line == 0 && err.Column == 0 {
				err.Line = token.Line
				err.Column = token.Column
				if err.File == "" && env != nil && env.Filename != "" {
					err.File = env.Filename
				}
			}
		}
		return partModule
	}

	partDict, ok := partModule.(*Dictionary)
	if !ok {
		return &Error{
			Class:   ClassType,
			Code:    "PART-0005",
			Message: "Part module did not return a dictionary",
			Line:    token.Line,
			Column:  token.Column,
		}
	}

	// Extract 'view' attribute (optional, defaults to "default")
	viewName := "default"
	if viewExpr, hasView := props.Pairs["view"]; hasView {
		viewObj := Eval(viewExpr, env)
		if isError(viewObj) {
			return viewObj
		}
		if viewStr, ok := viewObj.(*String); ok {
			viewName = viewStr.Value
		} else {
			return &Error{
				Class:   ClassType,
				Code:    "PART-0006",
				Message: "Part view must be a string",
				Hints:   []string{"Use view=\"edit\" or view={viewName}"},
				Line:    token.Line,
				Column:  token.Column,
			}
		}
	}

	// Optional: auto-refresh interval (milliseconds)
	hasRefresh := false
	refreshValue := ""
	if refreshExpr, ok := props.Pairs["part-refresh"]; ok {
		refreshObj := Eval(refreshExpr, env)
		if isError(refreshObj) {
			return refreshObj
		}
		refreshValue = fmt.Sprint(objectToGoValue(refreshObj))
		hasRefresh = true
	}

	// Optional: immediate load view name (fetch right away for slow data)
	hasLoad := false
	loadValue := ""
	if loadExpr, ok := props.Pairs["part-load"]; ok {
		loadObj := Eval(loadExpr, env)
		if isError(loadObj) {
			return loadObj
		}
		if loadStr, ok := loadObj.(*String); ok {
			loadValue = loadStr.Value
			hasLoad = true
		} else {
			// Coerce non-strings to string for robustness
			loadValue = fmt.Sprint(objectToGoValue(loadObj))
			hasLoad = true
		}
	}

	// Optional: lazy-load view name (fetch when scrolled into viewport)
	hasLazy := false
	lazyValue := ""
	if lazyExpr, ok := props.Pairs["part-lazy"]; ok {
		lazyObj := Eval(lazyExpr, env)
		if isError(lazyObj) {
			return lazyObj
		}
		if lazyStr, ok := lazyObj.(*String); ok {
			lazyValue = lazyStr.Value
			hasLazy = true
		} else {
			// Coerce non-strings to string for robustness
			lazyValue = fmt.Sprint(objectToGoValue(lazyObj))
			hasLazy = true
		}
	}

	// Optional: lazy-load threshold in px
	hasLazyThreshold := false
	lazyThresholdValue := ""
	if thresholdExpr, ok := props.Pairs["part-lazy-threshold"]; ok {
		thresholdObj := Eval(thresholdExpr, env)
		if isError(thresholdObj) {
			return thresholdObj
		}
		lazyThresholdValue = fmt.Sprint(objectToGoValue(thresholdObj))
		hasLazyThreshold = true
	}

	// Optional: id attribute for cross-part targeting
	hasId := false
	idValue := ""
	if idExpr, ok := props.Pairs["id"]; ok {
		idObj := Eval(idExpr, env)
		if isError(idObj) {
			return idObj
		}
		idValue = fmt.Sprint(objectToGoValue(idObj))
		hasId = true
	}

	// Look up the view function in the Part module
	viewExpr, hasView := partDict.Pairs[viewName]
	if !hasView {
		return &Error{
			Class:   ClassValue,
			Code:    "PART-0007",
			Message: fmt.Sprintf("Part does not export view '%s'", viewName),
			Hints:   []string{fmt.Sprintf("Add export %s = fn(props) { ... } to your Part file", viewName)},
			Line:    token.Line,
			Column:  token.Column,
		}
	}

	// Evaluate the view expression to get the function
	viewFn := Eval(viewExpr, partDict.Env)
	if isError(viewFn) {
		return viewFn
	}

	fnObj, ok := viewFn.(*Function)
	if !ok {
		return &Error{
			Class:   ClassType,
			Code:    "PART-0008",
			Message: fmt.Sprintf("Part view '%s' must be a function, got %s", viewName, viewFn.Type()),
			Line:    token.Line,
			Column:  token.Column,
		}
	}

	// Build props dictionary for view function (excluding src and view)
	viewProps := make(map[string]ast.Expression)
	for key, expr := range props.Pairs {
		if key == "src" || key == "view" || key == "id" || key == "part-refresh" || key == "part-load" || key == "part-lazy" || key == "part-lazy-threshold" {
			continue
		}
		viewProps[key] = expr
	}
	viewPropsDict := &Dictionary{Pairs: viewProps, Env: env}

	// Call the view function with props, passing environment for runtime context
	result := ApplyFunctionWithEnv(fnObj, []Object{viewPropsDict}, env)
	if isError(result) {
		return result
	}

	// Get the HTML content from the view function result
	htmlContent := objectToTemplateString(result)

	// Build data attributes for JavaScript runtime
	// Encode props as JSON for the data-part-props attribute
	propsJSON := encodePropsToJSON(viewPropsDict)

	// Resolve the absolute path for the Part file
	absPath, err := resolveModulePath(pathStr, env.Filename, env.RootPath)
	if err != nil {
		return &Error{
			Class:   ClassValue,
			Code:    "PART-0009",
			Message: fmt.Sprintf("Failed to resolve Part path: %v", err),
			Line:    token.Line,
			Column:  token.Column,
		}
	}

	// Convert absolute path to Part URL
	// Use the handler file's directory to distinguish @./ from @~/ parts
	handlerDir := filepath.Dir(env.Filename)
	partURL := convertPathToPartURL(absPath, env.RootPath, env.HandlerPath, handlerDir)

	// Mark that this page contains Parts (for JS injection)
	env.ContainsParts = true

	// Wrap the result in a div with data attributes
	var output strings.Builder
	output.WriteString("<div")
	if hasId {
		output.WriteString(" id=\"")
		output.WriteString(htmlEscape(idValue))
		output.WriteString("\"")
	}
	output.WriteString(" data-part-src=\"")
	output.WriteString(htmlEscape(partURL))
	output.WriteString("\" data-part-view=\"")
	output.WriteString(htmlEscape(viewName))
	output.WriteString("\" data-part-props='")
	output.WriteString(htmlEscape(propsJSON))
	output.WriteString("'")
	if hasRefresh {
		output.WriteString(" data-part-refresh=\"")
		output.WriteString(htmlEscape(refreshValue))
		output.WriteString("\"")
	}
	if hasLoad {
		output.WriteString(" data-part-load=\"")
		output.WriteString(htmlEscape(loadValue))
		output.WriteString("\"")
	}
	if hasLazy {
		output.WriteString(" data-part-lazy=\"")
		output.WriteString(htmlEscape(lazyValue))
		output.WriteString("\"")
	}
	if hasLazyThreshold {
		output.WriteString(" data-part-lazy-threshold=\"")
		output.WriteString(htmlEscape(lazyThresholdValue))
		output.WriteString("\"")
	}
	output.WriteString(">")
	output.WriteString(htmlContent)
	output.WriteString("</div>")

	return &String{Value: output.String()}
}

// convertPathToPartURL converts an absolute file path to a Part URL
// Uses the handler's route path as the base for relative URL calculation
// Example: If handler route is "/dashboard" and Part is "../shared/counter.part",
//
//	the URL becomes "/shared/counter.part"
func convertPathToPartURL(absPath string, rootPath string, handlerPath string, handlerDir string) string {
	// handlerPath is the route path (e.g., "/", "/dashboard/settings")
	// rootPath is the project root (handler's file system root for @~/)
	// handlerDir is the handler file's directory (for @./ resolution)
	// absPath is the Part file's absolute file system path

	if rootPath == "" {
		return absPath
	}

	// Check if the Part is within the handler's directory tree (relative Part, e.g., @./)
	// vs at the project root level (e.g., @~/)
	if handlerDir != "" {
		absHandlerDir, _ := filepath.Abs(handlerDir)
		absPartDir := filepath.Dir(absPath)

		// If Part is within the handler's directory tree, use handler's route as base
		if strings.HasPrefix(absPartDir+string(filepath.Separator), absHandlerDir+string(filepath.Separator)) ||
			absPartDir == absHandlerDir {
			// Part is relative to handler - calculate path relative to handler directory
			relToHandler, err := filepath.Rel(absHandlerDir, absPath)
			if err == nil {
				relURL := filepath.ToSlash(relToHandler)
				if handlerPath != "" {
					routeDir := filepath.Dir(handlerPath)
					if routeDir == "/" || routeDir == "." {
						return "/" + relURL
					}
					return routeDir + "/" + relURL
				}
				return "/" + relURL
			}
		}
	}

	// Part is at project root level (e.g., @~/parts/) - URL is just the relative path from root
	relPath, err := filepath.Rel(rootPath, absPath)
	if err != nil {
		return absPath
	}
	return "/" + filepath.ToSlash(relPath)
}

// htmlEscape escapes special HTML characters for safe attribute values
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// encodePropsToJSON encodes a props dictionary to JSON for data-part-props attribute
func encodePropsToJSON(props *Dictionary) string {
	// Build a map of evaluated prop values
	propsMap := make(map[string]interface{})
	for key, expr := range props.Pairs {
		// Evaluate the expression
		val := Eval(expr, props.Env)
		// Convert to Go type for JSON marshaling
		propsMap[key] = objectToGoValue(val)
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(propsMap)
	if err != nil {
		// If marshaling fails, return empty object
		return "{}"
	}

	return string(jsonBytes)
}

// objectToGoValue converts a Parsley object to a Go value for JSON marshaling
// For database storage, Money values are converted to their integer Amount (cents/minor units).
func objectToGoValue(obj Object) interface{} {
	switch v := obj.(type) {
	case *Integer:
		return v.Value
	case *Float:
		return v.Value
	case *Boolean:
		return v.Value
	case *String:
		return v.Value
	case *Money:
		// Store as integer (cents/minor units) - consistent with Money type storage
		return v.Amount
	case *Array:
		arr := make([]interface{}, len(v.Elements))
		for i, elem := range v.Elements {
			arr[i] = objectToGoValue(elem)
		}
		return arr
	case *Dictionary:
		m := make(map[string]interface{})
		for key, expr := range v.Pairs {
			val := Eval(expr, v.Env)
			m[key] = objectToGoValue(val)
		}
		return m
	case *Null:
		return nil
	default:
		// For other types, use Inspect() as string
		return obj.Inspect()
	}
}

// evalStandardTagPair evaluates a standard (lowercase) tag pair as HTML string
func evalStandardTagPair(node *ast.TagPairExpression, env *Environment) Object {
	var result strings.Builder

	result.WriteByte('<')
	result.WriteString(node.Name)

	// Check for @record attribute on <form> tags (FEAT-091 form binding)
	var formRecord *Record
	workingProps := node.Props
	if node.Name == "form" && strings.Contains(node.Props, "@record") {
		// Parse and evaluate @record expression
		// Calculate props position: tag column + "<" + tag name + space
		propsCol := node.Token.Column + 1 + len(node.Name) + 1
		recordExpr, parseErr := parseFormAtRecord(node.Props, env, node.Token.Line, propsCol)
		if parseErr != nil {
			return parseErr
		}
		if recordExpr != nil {
			recordObj := Eval(recordExpr, env)
			if isError(recordObj) {
				// Adjust error position for runtime errors in @record expression
				if errObj, ok := recordObj.(*Error); ok && errObj.Line <= 1 {
					// Find @record position in props
					atRecordIdx := strings.Index(node.Props, "@record=")
					if atRecordIdx == -1 {
						atRecordIdx = strings.Index(node.Props, "@record =")
					}
					exprOffset := atRecordIdx + len("@record={")
					errObj.Line = node.Token.Line
					errObj.Column = propsCol + exprOffset + (errObj.Column - 1)
					if errObj.File == "" {
						errObj.File = env.Filename
					}
				}
				return recordObj
			}
			var ok bool
			formRecord, ok = recordObj.(*Record)
			if !ok {
				// Find @record position for error
				atRecordIdx := strings.Index(node.Props, "@record=")
				if atRecordIdx == -1 {
					atRecordIdx = strings.Index(node.Props, "@record =")
				}
				return &Error{
					Class:   ClassType,
					Code:    "FORM-0006",
					Message: fmt.Sprintf("@record must be a Record, got %s", recordObj.Type()),
					Hints:   []string{"Use a Record created from a schema: @record={Schema({...})}"},
					Line:    node.Token.Line,
					Column:  propsCol + atRecordIdx,
					File:    env.Filename,
				}
			}
			// Remove @record from props so it doesn't render
			workingProps = removeAtRecord(node.Props)
		}
	}

	// Process props with interpolation (similar to singleton tags)
	if workingProps != "" {
		// Calculate props position: tag column + "<" + tag name + space
		propsCol := node.Token.Column + 1 + len(node.Name) + 1
		propsResult := evalTagProps(workingProps, env, node.Token.Line, propsCol)
		if isError(propsResult) {
			return propsResult
		}
		// Only add space and props if non-empty (spread-only props produce empty result)
		propsStr := propsResult.(*String).Value
		if propsStr != "" {
			result.WriteByte(' ')
			result.WriteString(propsStr)
		}
	}

	// Process spread expressions - merge all into a single map to handle overrides
	spreadAttrs := make(map[string]any)
	spreadOrder := []string{}

	for _, spread := range node.Spreads {
		// Evaluate the spread expression
		spreadObj := Eval(spread.Expression, env)
		if isError(spreadObj) {
			return spreadObj
		}

		// Get dictionary to spread (Record spreads its data)
		var dict *Dictionary
		switch v := spreadObj.(type) {
		case *Dictionary:
			dict = v
		case *Record:
			// Record spreads its data fields only
			dict = v.ToDictionary()
		default:
			perr := perrors.New("SPREAD-0001", map[string]any{
				"Got": spreadObj.Type(),
			})
			return &Error{
				Class:   ErrorClass(perr.Class),
				Code:    perr.Code,
				Message: perr.Message,
				Hints:   perr.Hints,
				Data:    perr.Data,
			}
		}

		// Merge dictionary entries (later spreads override earlier ones)
		// Use Keys() to preserve insertion order
		for _, key := range dict.Keys() {
			expr := dict.Pairs[key]
			// Track order of first appearance
			if _, exists := spreadAttrs[key]; !exists {
				spreadOrder = append(spreadOrder, key)
			}

			// Evaluate the expression in the dictionary's environment
			value := Eval(expr, dict.Env)
			if isError(value) {
				return value
			}

			// Store value (will override if key already exists)
			spreadAttrs[key] = value
		}
	}

	// Write spread attributes in order
	for _, key := range spreadOrder {
		value := spreadAttrs[key]

		// Skip null and false values
		switch v := value.(type) {
		case *Null:
			continue
		case *Boolean:
			if !v.Value {
				continue
			}
			// Boolean true: render as boolean attribute
			result.WriteByte(' ')
			result.WriteString(key)
			continue
		}

		// Render as regular attribute with value
		result.WriteByte(' ')
		result.WriteString(key)
		result.WriteString("=\"")

		// Get string value and escape
		strVal := objectToTemplateString(value.(Object))
		for _, c := range strVal {
			if c == '"' {
				result.WriteString("&quot;")
			} else if c == '&' {
				result.WriteString("&amp;")
			} else if c == '<' {
				result.WriteString("&lt;")
			} else if c == '>' {
				result.WriteString("&gt;")
			} else {
				result.WriteRune(c)
			}
		}

		result.WriteByte('"')
	}

	result.WriteByte('>')

	// Create form context if @record was specified (FEAT-091)
	contentsEnv := env
	if formRecord != nil {
		contentsEnv = NewEnclosedEnvironment(env)
		contentsEnv.FormContext = &FormContext{Record: formRecord}
	}

	// Evaluate and append contents
	contentsObj := evalTagContents(node.Contents, contentsEnv)
	if isError(contentsObj) {
		return contentsObj
	}
	result.WriteString(contentsObj.(*String).Value)

	result.WriteString("</")
	result.WriteString(node.Name)
	result.WriteByte('>')

	return &String{Value: result.String()}
}

// evalCustomTagPair evaluates a custom (uppercase) tag pair as a function call
func evalCustomTagPair(node *ast.TagPairExpression, env *Environment) Object {
	// Special handling for <SQL> tags
	if node.Name == "SQL" {
		return evalSQLTag(node, env)
	}

	// Special handling for <Label>...</Label> form component (FEAT-091)
	if node.Name == "Label" {
		// Evaluate contents
		contentsObj := evalTagContentsAsArray(node.Contents, env)
		if isError(contentsObj) {
			return contentsObj
		}
		var contents []Object
		if contentsArray, ok := contentsObj.(*Array); ok {
			contents = contentsArray.Elements
		} else {
			contents = []Object{contentsObj}
		}
		return evalLabelComponent(node.Props, contents, false, env)
	}

	// Look up the component variable/function
	val, ok := env.Get(node.Name)
	if !ok {
		return newUndefinedComponentError(node.Name)
	}

	// If the value is a String (e.g., loaded SVG), return it directly
	// Note: For tag pairs like <Arrow>...</Arrow>, the contents are ignored for string values
	if str, isString := val.(*String); isString {
		return str
	}

	// Parse props into a dictionary and add contents
	propsCol := node.Token.Column + 1 + len(node.Name) + 1
	propsDict := parseTagProps(node.Props, env, node.Token.Line, propsCol)
	if isError(propsDict) {
		return propsDict
	}

	dict := propsDict.(*Dictionary)

	// Evaluate contents and add to props as "contents"
	contentsObj := evalTagContentsAsArray(node.Contents, env)
	if isError(contentsObj) {
		return contentsObj
	}

	// Create a literal expression for the contents array
	// We need to wrap the evaluated contents in an expression
	dict.Pairs["contents"] = &ast.ArrayLiteral{Elements: []ast.Expression{}}

	// Store the evaluated contents directly in the environment temporarily
	contentsEnv := NewEnclosedEnvironment(env)
	contentsEnv.Set("__tag_contents__", contentsObj)

	// Actually, let's simplify - evaluate contents as a single value
	if contentsArray, ok := contentsObj.(*Array); ok && len(contentsArray.Elements) == 1 {
		// Single item - pass directly
		dict.Pairs["contents"] = createLiteralExpression(contentsArray.Elements[0])
	} else {
		// Multiple items or empty - pass as array
		dict.Pairs["contents"] = createLiteralExpression(contentsObj)
	}

	// Check if component is null (common when import destructuring gets wrong name)
	if val == NULL || val == nil {
		perr := perrors.New("COMP-0001", map[string]any{"Name": node.Name})
		return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data, Line: node.Token.Line, Column: node.Token.Column}
	}

	// Call the function with the props dictionary, passing environment for runtime context
	result := ApplyFunctionWithEnv(val, []Object{dict}, env)

	// Improve error message if function call failed
	if err, isErr := result.(*Error); isErr && strings.Contains(err.Message, "cannot call") {
		perr := perrors.New("COMP-0002", map[string]any{"Name": node.Name, "Got": string(val.Type())})
		return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data, Line: node.Token.Line, Column: node.Token.Column}
	}

	return result
}

// evalTagContents evaluates tag contents and returns as a concatenated string
func evalTagContents(contents []ast.Node, env *Environment) Object {
	var result strings.Builder

	for _, node := range contents {
		obj := Eval(node, env)
		if isError(obj) {
			return obj
		}
		result.WriteString(objectToTemplateString(obj))
	}

	return &String{Value: result.String()}
}

// evalTagContentsAsArray evaluates tag contents and returns as an array
func evalTagContentsAsArray(contents []ast.Node, env *Environment) Object {
	if len(contents) == 0 {
		return NULL
	}

	elements := make([]Object, 0, len(contents))
	for _, node := range contents {
		obj := Eval(node, env)
		if isError(obj) {
			return obj
		}
		// Convert to string for consistency
		elements = append(elements, &String{Value: objectToTemplateString(obj)})
	}

	return &Array{Elements: elements}
}

// evalSQLTag handles <SQL params={...}>...</SQL> tags
func evalSQLTag(node *ast.TagPairExpression, env *Environment) Object {
	// Parse props to get params
	propsCol := node.Token.Column + 1 + len(node.Name) + 1
	propsDict := parseTagProps(node.Props, env, node.Token.Line, propsCol)
	if isError(propsDict) {
		return propsDict
	}

	// Get the SQL content
	sqlContent := evalTagContents(node.Contents, env)
	if isError(sqlContent) {
		return sqlContent
	}

	sqlStr, ok := sqlContent.(*String)
	if !ok {
		perr := perrors.New("SQL-0001", nil)
		return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data, Line: node.Token.Line, Column: node.Token.Column}
	}

	// Build result dictionary with sql and params
	resultPairs := map[string]ast.Expression{
		"sql": &ast.StringLiteral{Value: sqlStr.Value},
	}

	// Add params if provided
	if dict, ok := propsDict.(*Dictionary); ok {
		if paramsExpr, hasParams := dict.Pairs["params"]; hasParams {
			resultPairs["params"] = paramsExpr
		}
	}

	return &Dictionary{
		Pairs: resultPairs,
		Env:   env,
	}
}

// evalTagProps evaluates tag props string with interpolations.
// baseLine and baseCol specify the position of the props string start for error reporting.
func evalTagProps(propsStr string, env *Environment, baseLine, baseCol int) Object {
	var result strings.Builder

	i := 0
	for i < len(propsStr) {
		// Skip leading whitespace, buffering it
		wsStart := i
		for i < len(propsStr) && (propsStr[i] == ' ' || propsStr[i] == '\t' || propsStr[i] == '\n' || propsStr[i] == '\r') {
			i++
		}

		// If we've reached the end, break
		if i >= len(propsStr) {
			break
		}

		// Check for spread syntax ...identifier
		if i+3 <= len(propsStr) && propsStr[i:i+3] == "..." {
			// Skip the ...
			i += 3
			// Skip whitespace
			for i < len(propsStr) && (propsStr[i] == ' ' || propsStr[i] == '\t' || propsStr[i] == '\n' || propsStr[i] == '\r') {
				i++
			}
			// Skip identifier
			for i < len(propsStr) && ((propsStr[i] >= 'a' && propsStr[i] <= 'z') || (propsStr[i] >= 'A' && propsStr[i] <= 'Z') || (propsStr[i] >= '0' && propsStr[i] <= '9') || propsStr[i] == '_') {
				i++
			}
			// Don't write the buffered whitespace for spread operators
			continue
		}

		// Not a spread operator - write the buffered whitespace
		result.WriteString(propsStr[wsStart:i])

		// Look for ={expr} - prop expression syntax
		if propsStr[i] == '=' && i+1 < len(propsStr) && propsStr[i+1] == '{' {
			// Don't write = yet - we need to see if value is null/false first
			i++ // skip =
			i++ // skip {
			braceCount := 1
			exprStart := i

			for i < len(propsStr) && braceCount > 0 {
				if propsStr[i] == '"' {
					// Skip quoted strings
					i++
					for i < len(propsStr) && propsStr[i] != '"' {
						if propsStr[i] == '\\' {
							i += 2
						} else {
							i++
						}
					}
					if i < len(propsStr) {
						i++
					}
					continue
				}
				if propsStr[i] == '{' {
					braceCount++
				} else if propsStr[i] == '}' {
					braceCount--
				}
				if braceCount > 0 {
					i++
				}
			}

			if braceCount != 0 {
				return newParseError("PARSE-0009", "tag props", nil)
			}

			// Extract and evaluate the expression
			exprStr := propsStr[exprStart:i]
			exprOffset := exprStart // offset of expression within props string
			i++                     // skip closing }

			// Parse and evaluate the expression
			l := lexer.NewWithFilename(exprStr, env.Filename)
			p := parser.New(l)
			program := p.ParseProgram()

			if errs := p.StructuredErrors(); len(errs) > 0 {
				perr := errs[0]
				// Adjust position: add props offset to error position
				adjustedCol := baseCol + exprOffset + (perr.Column - 1)
				return &Error{
					Class:   ClassParse,
					Code:    perr.Code,
					Message: perr.Message,
					Hints:   perr.Hints,
					Line:    baseLine,
					Column:  adjustedCol,
					File:    env.Filename,
					Data:    perr.Data,
				}
			}

			// Evaluate the expression
			var evaluated Object
			for _, stmt := range program.Statements {
				evaluated = Eval(stmt, env)
				if isError(evaluated) {
					// Adjust error position for runtime errors
					if errObj, ok := evaluated.(*Error); ok && errObj.Line <= 1 {
						errObj.Line = baseLine
						errObj.Column = baseCol + exprOffset + (errObj.Column - 1)
						if errObj.Column < baseCol+exprOffset {
							errObj.Column = baseCol + exprOffset
						}
						if errObj.File == "" {
							errObj.File = env.Filename
						}
					}
					return evaluated
				}
			}

			// Only write attribute if value is not null or false
			// For null or false, we need to remove the attribute name that was already written
			if evaluated != nil {
				// Check if it's a Null object or Boolean false
				shouldOmit := false
				switch v := evaluated.(type) {
				case *Null:
					shouldOmit = true
				case *Boolean:
					if !v.Value {
						shouldOmit = true
					}
				}

				if shouldOmit {
					// Remove trailing attribute name from result
					// Walk backwards to find the start of the attribute name
					s := result.String()
					j := len(s) - 1
					// Skip trailing whitespace
					for j >= 0 && (s[j] == ' ' || s[j] == '\n' || s[j] == '\t' || s[j] == '\r') {
						j--
					}
					// Walk back to find start of attribute name (stop at space or start)
					for j >= 0 && s[j] != ' ' && s[j] != '\n' && s[j] != '\t' && s[j] != '\r' {
						j--
					}
					// Rebuild result without the attribute name
					result.Reset()
					result.WriteString(s[:j+1])
				} else {
					// Write the attribute value
					strVal := objectToTemplateString(evaluated)
					result.WriteByte('=')
					result.WriteByte('"')
					// Escape quotes in the value
					for _, c := range strVal {
						if c == '"' {
							result.WriteString("\\\"")
						} else {
							result.WriteRune(c)
						}
					}
					result.WriteByte('"')
				}
			}
			continue
		}

		// Handle single-quoted strings (raw with @{} interpolation)
		// This must be checked before {expr} handling
		if propsStr[i] == '\'' {
			result.WriteByte(propsStr[i])
			i++
			// Read until closing single quote, handling @{} interpolation
			for i < len(propsStr) && propsStr[i] != '\'' {
				if propsStr[i] == '\\' && i+1 < len(propsStr) {
					next := propsStr[i+1]
					if next == '\'' {
						// Escaped single quote - write just the quote
						result.WriteByte('\'')
						i += 2
						continue
					} else if next == '@' {
						// Escaped @ - write just the @
						result.WriteByte('@')
						i += 2
						continue
					}
				}
				// Check for @{ interpolation
				if propsStr[i] == '@' && i+1 < len(propsStr) && propsStr[i+1] == '{' {
					i += 2 // skip @{
					braceCount := 1
					exprStart := i

					// Find closing } with brace counting
					for i < len(propsStr) && braceCount > 0 {
						if propsStr[i] == '{' {
							braceCount++
						} else if propsStr[i] == '}' {
							braceCount--
						}
						if braceCount > 0 {
							i++
						}
					}

					if braceCount != 0 {
						return newParseError("PARSE-0009", "raw template in tag props", nil)
					}

					exprStr := propsStr[exprStart:i]
					exprOffset := exprStart // offset within props string
					i++                     // skip closing }

					// Evaluate the expression
					l := lexer.NewWithFilename(exprStr, env.Filename)
					p := parser.New(l)
					program := p.ParseProgram()

					if errs := p.StructuredErrors(); len(errs) > 0 {
						perr := errs[0]
						// Adjust position: add props offset to error position
						adjustedCol := baseCol + exprOffset + (perr.Column - 1)
						return &Error{
							Class:   ClassParse,
							Code:    perr.Code,
							Message: perr.Message,
							Hints:   perr.Hints,
							Line:    baseLine,
							Column:  adjustedCol,
							File:    env.Filename,
							Data:    perr.Data,
						}
					}

					var evaluated Object
					for _, stmt := range program.Statements {
						evaluated = Eval(stmt, env)
						if isError(evaluated) {
							// Adjust error position for runtime errors
							if errObj, ok := evaluated.(*Error); ok && errObj.Line <= 1 {
								errObj.Line = baseLine
								errObj.Column = baseCol + exprOffset + (errObj.Column - 1)
								if errObj.Column < baseCol+exprOffset {
									errObj.Column = baseCol + exprOffset
								}
								if errObj.File == "" {
									errObj.File = env.Filename
								}
							}
							return evaluated
						}
					}

					if evaluated != nil {
						result.WriteString(objectToTemplateString(evaluated))
					}
					continue
				}
				result.WriteByte(propsStr[i])
				i++
			}
			if i < len(propsStr) {
				result.WriteByte(propsStr[i]) // write closing quote
				i++
			}
			continue
		}

		// Look for {expr} - inline interpolation (legacy syntax)
		if propsStr[i] == '{' {
			// Find the closing }
			i++ // skip {
			braceCount := 1
			exprStart := i

			for i < len(propsStr) && braceCount > 0 {
				if propsStr[i] == '"' {
					// Skip quoted strings
					i++
					for i < len(propsStr) && propsStr[i] != '"' {
						if propsStr[i] == '\\' {
							i += 2
						} else {
							i++
						}
					}
					if i < len(propsStr) {
						i++
					}
					continue
				}
				if propsStr[i] == '{' {
					braceCount++
				} else if propsStr[i] == '}' {
					braceCount--
				}
				if braceCount > 0 {
					i++
				}
			}

			if braceCount != 0 {
				return newParseError("PARSE-0009", "tag props", nil)
			}

			// Extract and evaluate the expression
			exprStr := propsStr[exprStart:i]
			exprOffset := exprStart // offset within props string
			i++                     // skip closing }

			// Parse and evaluate the expression (with filename for error reporting)
			l := lexer.NewWithFilename(exprStr, env.Filename)
			p := parser.New(l)
			program := p.ParseProgram()

			if errs := p.StructuredErrors(); len(errs) > 0 {
				// Return first parse error with adjusted position
				perr := errs[0]
				adjustedCol := baseCol + exprOffset + (perr.Column - 1)
				return &Error{
					Class:   ClassParse,
					Code:    perr.Code,
					Message: perr.Message,
					Hints:   perr.Hints,
					Line:    baseLine,
					Column:  adjustedCol,
					File:    env.Filename,
					Data:    perr.Data,
				}
			}

			// Evaluate the expression
			var evaluated Object
			for _, stmt := range program.Statements {
				evaluated = Eval(stmt, env)
				if isError(evaluated) {
					// Adjust error position for runtime errors
					if errObj, ok := evaluated.(*Error); ok && errObj.Line <= 1 {
						errObj.Line = baseLine
						errObj.Column = baseCol + exprOffset + (errObj.Column - 1)
						if errObj.Column < baseCol+exprOffset {
							errObj.Column = baseCol + exprOffset
						}
						if errObj.File == "" {
							errObj.File = env.Filename
						}
					}
					return evaluated
				}
			}

			// Convert result to string
			if evaluated != nil {
				result.WriteString(objectToTemplateString(evaluated))
			}
		} else {
			// Regular character
			result.WriteByte(propsStr[i])
			i++
		}
	}

	return &String{Value: result.String()}
}

// createLiteralExpression creates an AST expression from an evaluated object
// This is a helper for passing evaluated values back through the AST
func createLiteralExpression(obj Object) ast.Expression {
	switch obj := obj.(type) {
	case *Integer:
		return &ast.IntegerLiteral{
			Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", obj.Value)},
			Value: obj.Value,
		}
	case *Float:
		return &ast.FloatLiteral{
			Token: lexer.Token{Type: lexer.FLOAT, Literal: fmt.Sprintf("%g", obj.Value)},
			Value: obj.Value,
		}
	case *String:
		return &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: obj.Value},
			Value: obj.Value,
		}
	case *Boolean:
		lit := "false"
		if obj.Value {
			lit = "true"
		}
		return &ast.Boolean{
			Token: lexer.Token{Type: lexer.IDENT, Literal: lit},
			Value: obj.Value,
		}
	case *Null:
		// Use an identifier that will evaluate to the NULL object
		return &ast.Identifier{
			Token: lexer.Token{Type: lexer.IDENT, Literal: "__null__"},
			Value: "__null__",
		}
	case *Array:
		// For arrays, create array literal with elements
		elements := make([]ast.Expression, len(obj.Elements))
		for i, elem := range obj.Elements {
			elements[i] = createLiteralExpression(elem)
		}
		return &ast.ArrayLiteral{
			Token:    lexer.Token{Type: lexer.LBRACKET, Literal: "["},
			Elements: elements,
		}
	case *Dictionary:
		// For dictionaries, create dictionary literal with pairs
		pairs := make(map[string]ast.Expression)
		for key, expr := range obj.Pairs {
			// Evaluate the expression to get the value
			val := Eval(expr, obj.Env)
			pairs[key] = createLiteralExpression(val)
		}
		return &ast.DictionaryLiteral{
			Token: lexer.Token{Type: lexer.LBRACE, Literal: "{"},
			Pairs: pairs,
		}
	default:
		// For other types, return a string literal
		return &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: obj.Inspect()},
			Value: obj.Inspect(),
		}
	}
}

// evalStandardTag evaluates a standard (lowercase) tag as an interpolated string
func evalStandardTag(node *ast.TagLiteral, tagName string, propsStr string, env *Environment) Object {
	// Handle @field attribute for form binding (FEAT-091)
	if tagName == "input" && strings.Contains(propsStr, "@field") {
		fieldName := parseFieldAttribute(propsStr)
		if fieldName != "" {
			formCtx := getFormContext(env)
			if formCtx == nil {
				return &Error{
					Class:   ClassValue,
					Code:    "FORM-0002",
					Message: "Input with @field must be inside a <form @record={...}> context",
					Hints:   []string{`Wrap in a form with @record: <form @record={myRecord}><input @field="name"/></form>`},
					Line:    node.Token.Line,
					Column:  node.Token.Column,
				}
			}

			record := formCtx.Record
			if record == nil || record.Schema == nil {
				return &Error{
					Class:   ClassValue,
					Code:    "FORM-0003",
					Message: "Form context has no valid record or schema",
					Line:    node.Token.Line,
					Column:  node.Token.Column,
				}
			}

			// Get existing type attribute (if any)
			existingType := parseExistingType(propsStr)

			// Build the input tag with generated attributes
			var result strings.Builder
			result.WriteString("<input")

			// Add auto-generated attributes based on schema
			if existingType == "checkbox" {
				result.WriteString(buildCheckboxAttributes(record, fieldName))
			} else if existingType == "radio" {
				radioValue := parseExistingValue(propsStr)
				result.WriteString(buildRadioAttributes(record, fieldName, radioValue))
			} else {
				result.WriteString(buildInputAttributes(record, fieldName, existingType))
			}

			// Add remaining props (without @field)
			cleanedProps := removeFieldAttribute(propsStr)
			// Also remove name and value as we generate them
			cleanedProps = removeAttr(cleanedProps, "name")
			if existingType != "radio" { // Keep value for radio inputs
				cleanedProps = removeAttr(cleanedProps, "value")
			}

			if cleanedProps = strings.TrimSpace(cleanedProps); cleanedProps != "" {
				// Process remaining props with interpolation
				propsResult := evalTagProps(cleanedProps, env, node.Token.Line, node.Token.Column)
				if isError(propsResult) {
					return propsResult
				}
				propsStrClean := propsResult.(*String).Value
				if propsStrClean != "" {
					result.WriteByte(' ')
					result.WriteString(propsStrClean)
				}
			}

			result.WriteString("/>")
			return &String{Value: result.String()}
		}
	}

	// Handle @field attribute for textarea (FEAT-091)
	if tagName == "textarea" && strings.Contains(propsStr, "@field") {
		fieldName := parseFieldAttribute(propsStr)
		if fieldName != "" {
			formCtx := getFormContext(env)
			if formCtx == nil {
				return &Error{
					Class:   ClassValue,
					Code:    "FORM-0002",
					Message: "Textarea with @field must be inside a <form @record={...}> context",
					Hints:   []string{`Wrap in a form with @record: <form @record={myRecord}><textarea @field="bio"/></form>`},
					Line:    node.Token.Line,
					Column:  node.Token.Column,
				}
			}

			record := formCtx.Record
			if record == nil || record.Schema == nil {
				return &Error{
					Class:   ClassValue,
					Code:    "FORM-0003",
					Message: "Form context has no valid record or schema",
					Line:    node.Token.Line,
					Column:  node.Token.Column,
				}
			}

			// Build textarea tag
			var result strings.Builder
			result.WriteString("<textarea")
			result.WriteString(fmt.Sprintf(` name="%s"`, fieldName))

			// Add validation attributes from schema
			field := record.Schema.Fields[fieldName]
			if field != nil {
				if field.Required {
					result.WriteString(` required`)
					result.WriteString(` aria-required="true"`)
				}
				if field.MinLength != nil {
					result.WriteString(fmt.Sprintf(` minlength="%d"`, *field.MinLength))
				}
				if field.MaxLength != nil {
					result.WriteString(fmt.Sprintf(` maxlength="%d"`, *field.MaxLength))
				}
				if field.Metadata != nil {
					if placeholder, ok := field.Metadata["placeholder"]; ok {
						if strVal, ok := placeholder.(*String); ok {
							result.WriteString(fmt.Sprintf(` placeholder="%s"`, escapeAttrValue(strVal.Value)))
						}
					}
				}
			}

			// Add ARIA validation state
			hasError := record.Errors != nil && record.Errors[fieldName] != nil
			if hasError {
				result.WriteString(` aria-invalid="true"`)
				result.WriteString(fmt.Sprintf(` aria-describedby="%s-error"`, fieldName))
			} else if record.Validated {
				result.WriteString(` aria-invalid="false"`)
			}

			// Add remaining props
			cleanedProps := removeFieldAttribute(propsStr)
			cleanedProps = removeAttr(cleanedProps, "name")

			if cleanedProps = strings.TrimSpace(cleanedProps); cleanedProps != "" {
				propsResult := evalTagProps(cleanedProps, env, node.Token.Line, node.Token.Column)
				if isError(propsResult) {
					return propsResult
				}
				propsStrClean := propsResult.(*String).Value
				if propsStrClean != "" {
					result.WriteByte(' ')
					result.WriteString(propsStrClean)
				}
			}

			result.WriteString(">")

			// Add value as content
			value := record.Get(fieldName, record.Env)
			if value != nil && value != NULL {
				valueStr := objectToTemplateString(value)
				result.WriteString(escapeHTMLText(valueStr))
			}

			result.WriteString("</textarea>")
			return &String{Value: result.String()}
		}
	}

	var result strings.Builder
	result.WriteByte('<')
	result.WriteString(tagName)

	// Process props with interpolation
	// Buffer whitespace so we can skip it if followed by spread operator
	i := 0
	for i < len(propsStr) {
		// Skip leading whitespace, buffering it
		wsStart := i
		for i < len(propsStr) && (propsStr[i] == ' ' || propsStr[i] == '\t' || propsStr[i] == '\n' || propsStr[i] == '\r') {
			i++
		}

		// If we've reached the end, break
		if i >= len(propsStr) {
			break
		}

		// Check for spread syntax ...identifier
		if i+3 <= len(propsStr) && propsStr[i:i+3] == "..." {
			// Skip the ...
			i += 3
			// Skip whitespace
			for i < len(propsStr) && (propsStr[i] == ' ' || propsStr[i] == '\t' || propsStr[i] == '\n' || propsStr[i] == '\r') {
				i++
			}
			// Skip identifier
			for i < len(propsStr) && ((propsStr[i] >= 'a' && propsStr[i] <= 'z') || (propsStr[i] >= 'A' && propsStr[i] <= 'Z') || (propsStr[i] >= '0' && propsStr[i] <= '9') || propsStr[i] == '_') {
				i++
			}
			// Don't write the buffered whitespace for spread operators
			continue
		}

		// Not a spread operator - write the buffered whitespace
		result.WriteString(propsStr[wsStart:i])

		// Handle single-quoted strings (raw with @{} interpolation)
		if propsStr[i] == '\'' {
			result.WriteByte(propsStr[i])
			i++
			// Read until closing single quote, handling @{} interpolation
			for i < len(propsStr) && propsStr[i] != '\'' {
				if propsStr[i] == '\\' && i+1 < len(propsStr) {
					next := propsStr[i+1]
					if next == '\'' {
						// Escaped single quote - write just the quote
						result.WriteByte('\'')
						i += 2
						continue
					} else if next == '@' {
						// Escaped @ - write just the @
						result.WriteByte('@')
						i += 2
						continue
					}
				}
				// Check for @{ interpolation
				if propsStr[i] == '@' && i+1 < len(propsStr) && propsStr[i+1] == '{' {
					i += 2 // skip @{
					braceCount := 1
					exprStart := i

					// Find closing } with brace counting
					for i < len(propsStr) && braceCount > 0 {
						if propsStr[i] == '{' {
							braceCount++
						} else if propsStr[i] == '}' {
							braceCount--
						}
						if braceCount > 0 {
							i++
						}
					}

					if braceCount != 0 {
						return newParseError("PARSE-0009", "raw template in standard tag props", nil)
					}

					exprStr := propsStr[exprStart:i]
					i++ // skip closing }

					// Evaluate the expression
					l := lexer.NewWithFilename(exprStr, env.Filename)
					p := parser.New(l)
					program := p.ParseProgram()

					if errs := p.StructuredErrors(); len(errs) > 0 {
						perr := errs[0]
						return &Error{
							Class:   ClassParse,
							Code:    perr.Code,
							Message: perr.Message,
							Hints:   perr.Hints,
							Line:    node.Token.Line,
							Column:  node.Token.Column + exprStart + (perr.Column - 1),
							File:    env.Filename,
							Data:    perr.Data,
						}
					}

					var evaluated Object
					for _, stmt := range program.Statements {
						evaluated = Eval(stmt, env)
						if isError(evaluated) {
							// Adjust error position for runtime errors
							if errObj, ok := evaluated.(*Error); ok && errObj.Line <= 1 {
								errObj.Line = node.Token.Line
								errObj.Column = node.Token.Column + exprStart + (errObj.Column - 1)
								if errObj.File == "" {
									errObj.File = env.Filename
								}
							}
							return evaluated
						}
					}

					if evaluated != nil {
						result.WriteString(objectToTemplateString(evaluated))
					}
					continue
				}
				result.WriteByte(propsStr[i])
				i++
			}
			if i < len(propsStr) {
				result.WriteByte(propsStr[i]) // write closing quote
				i++
			}
			continue
		}

		// Look for {expr}
		if propsStr[i] == '{' {
			// Check if this is new syntax attr={expr} or old syntax attr="{expr}"
			// Walk back to see if we just wrote =" or just =
			s := result.String()
			hasQuoteBefore := len(s) > 0 && s[len(s)-1] == '"'
			hasEqualsBefore := len(s) > 1 && s[len(s)-2] == '=' || (len(s) > 0 && s[len(s)-1] == '=' && !hasQuoteBefore)
			isNewSyntax := hasEqualsBefore && !hasQuoteBefore

			// Find the closing }
			i++ // skip {
			braceCount := 1
			exprStart := i

			for i < len(propsStr) && braceCount > 0 {
				if propsStr[i] == '"' {
					// Skip quoted strings
					i++
					for i < len(propsStr) && propsStr[i] != '"' {
						if propsStr[i] == '\\' {
							i += 2
						} else {
							i++
						}
					}
					if i < len(propsStr) {
						i++
					}
					continue
				}
				if propsStr[i] == '{' {
					braceCount++
				} else if propsStr[i] == '}' {
					braceCount--
				}
				if braceCount > 0 {
					i++
				}
			}

			if braceCount != 0 {
				return newParseError("PARSE-0009", "tag", nil)
			}

			// Extract and evaluate the expression
			exprStr := propsStr[exprStart:i]
			exprOffset := exprStart // offset within propsStr
			i++                     // skip closing }

			// Parse and evaluate the expression (with filename for error reporting)
			l := lexer.NewWithFilename(exprStr, env.Filename)
			p := parser.New(l)
			program := p.ParseProgram()

			if errs := p.StructuredErrors(); len(errs) > 0 {
				// Return first parse error with adjusted position
				perr := errs[0]
				return &Error{
					Class:   ClassParse,
					Code:    perr.Code,
					Message: perr.Message,
					Hints:   perr.Hints,
					Line:    node.Token.Line,
					Column:  node.Token.Column + exprOffset + (perr.Column - 1),
					File:    env.Filename,
					Data:    perr.Data,
				}
			}

			// Evaluate the expression
			var evaluated Object
			for _, stmt := range program.Statements {
				evaluated = Eval(stmt, env)
				if isError(evaluated) {
					// Adjust error position for runtime errors
					if errObj, ok := evaluated.(*Error); ok && errObj.Line <= 1 {
						errObj.Line = node.Token.Line
						errObj.Column = node.Token.Column + exprOffset + (errObj.Column - 1)
						if errObj.File == "" {
							errObj.File = env.Filename
						}
					}
					return evaluated
				}
			}

			// For new syntax (attr={expr}), omit null/false values
			// For old syntax (attr="{expr}"), render even null/empty to maintain compatibility
			if evaluated != nil {
				if isNewSyntax {
					// Check if we should omit this attribute
					shouldOmit := false
					switch v := evaluated.(type) {
					case *Null:
						shouldOmit = true
					case *Boolean:
						if !v.Value {
							shouldOmit = true
						}
					}

					if shouldOmit {
						// Remove trailing "attrname=" and preceding whitespace from result
						s := result.String()
						j := len(s) - 1

						// Walk back past the = sign
						if j >= 0 && s[j] == '=' {
							j--
						}

						// Walk back past the attribute name
						for j >= 0 && s[j] != ' ' && s[j] != '\n' && s[j] != '\t' && s[j] != '\r' {
							j--
						}

						// Also remove ALL preceding whitespace - next attribute will have its own
						for j >= 0 && (s[j] == ' ' || s[j] == '\n' || s[j] == '\t' || s[j] == '\r') {
							j--
						}

						// Rebuild result without the attribute
						result.Reset()
						result.WriteString(s[:j+1])
					} else {
						// Write the value - wrap in quotes for HTML attributes
						switch evaluated.(type) {
						case *Boolean:
							// Boolean true renders as just the attribute name (HTML5 boolean attribute)
							// e.g., <input disabled/> not <input disabled="true"/>
							// We already handled false (omitted) above, so this must be true
							// Remove the trailing "=" from attr=
							s := result.String()
							if len(s) > 0 && s[len(s)-1] == '=' {
								result.Reset()
								result.WriteString(s[:len(s)-1])
							}
						default:
							// For strings and other values, quote them
							result.WriteByte('"')
							strVal := objectToTemplateString(evaluated)
							// Escape quotes in the value
							for _, c := range strVal {
								if c == '"' {
									result.WriteString("&quot;")
								} else if c == '&' {
									result.WriteString("&amp;")
								} else if c == '<' {
									result.WriteString("&lt;")
								} else if c == '>' {
									result.WriteString("&gt;")
								} else {
									result.WriteRune(c)
								}
							}
							result.WriteByte('"')
						}
					}
				} else {
					// Old syntax - always write
					result.WriteString(objectToTemplateString(evaluated))
				}
			}
		} else {
			// Regular character
			result.WriteByte(propsStr[i])
			i++
		}
	}

	// Process spread expressions - merge all into a single map to handle overrides
	spreadAttrs := make(map[string]any)
	spreadOrder := []string{}

	for _, spread := range node.Spreads {
		// Evaluate the spread expression
		spreadObj := Eval(spread.Expression, env)
		if isError(spreadObj) {
			return spreadObj
		}

		// Get dictionary to spread (Record spreads its data)
		var dict *Dictionary
		switch v := spreadObj.(type) {
		case *Dictionary:
			dict = v
		case *Record:
			// Record spreads its data fields only
			dict = v.ToDictionary()
		default:
			perr := perrors.New("SPREAD-0001", map[string]any{
				"Got": spreadObj.Type(),
			})
			return &Error{
				Class:   ErrorClass(perr.Class),
				Code:    perr.Code,
				Message: perr.Message,
				Hints:   perr.Hints,
				Data:    perr.Data,
			}
		}

		// Merge dictionary entries (later spreads override earlier ones)
		// Use Keys() to preserve insertion order
		for _, key := range dict.Keys() {
			expr := dict.Pairs[key]
			// Track order of first appearance
			if _, exists := spreadAttrs[key]; !exists {
				spreadOrder = append(spreadOrder, key)
			}

			// Evaluate the expression in the dictionary's environment
			value := Eval(expr, dict.Env)
			if isError(value) {
				return value
			}

			// Store value (will override if key already exists)
			spreadAttrs[key] = value
		}
	}

	// Write spread attributes in order
	for _, key := range spreadOrder {
		value := spreadAttrs[key]

		// Skip null and false values
		switch v := value.(type) {
		case *Null:
			continue
		case *Boolean:
			if !v.Value {
				continue
			}
			// Boolean true: render as boolean attribute
			result.WriteByte(' ')
			result.WriteString(key)
			continue
		}

		// Render as regular attribute with value
		result.WriteByte(' ')
		result.WriteString(key)
		result.WriteString("=\"")

		// Get string value and escape
		strVal := objectToTemplateString(value.(Object))
		for _, c := range strVal {
			if c == '"' {
				result.WriteString("&quot;")
			} else if c == '&' {
				result.WriteString("&amp;")
			} else if c == '<' {
				result.WriteString("&lt;")
			} else if c == '>' {
				result.WriteString("&gt;")
			} else {
				result.WriteRune(c)
			}
		}

		result.WriteByte('"')
	}

	result.WriteString(" />")
	return &String{Value: result.String()}
}

// evalCustomTag evaluates a custom (uppercase) tag as a function call
func evalCustomTag(tok lexer.Token, tagName string, propsStr string, env *Environment) Object {
	// Special handling for CSS and Javascript bundle tags
	if tagName == "CSS" {
		if env.AssetBundle == nil {
			return &String{Value: ""} // No bundle available
		}
		url := env.AssetBundle.CSSUrl()
		if url == "" {
			return &String{Value: ""} // No CSS files in bundle
		}
		return &String{Value: fmt.Sprintf(`<link rel="stylesheet" href="%s">`, url)}
	}
	if tagName == "Javascript" {
		if env.AssetBundle == nil {
			return &String{Value: ""} // No bundle available
		}
		url := env.AssetBundle.JSUrl()
		if url == "" {
			return &String{Value: ""} // No JS files in bundle
		}
		return &String{Value: fmt.Sprintf(`<script src="%s"></script>`, url)}
	}
	// Special handling for BasilJS prelude script tag
	if tagName == "BasilJS" {
		if env.BasilJSURL == "" {
			return &String{Value: ""} // No basil.js URL available
		}
		return &String{Value: fmt.Sprintf(`<script src="%s"></script>`, env.BasilJSURL)}
	}

	// Special handling for form binding components (FEAT-091)
	if tagName == "Label" {
		return evalLabelComponent(propsStr, nil, true, env)
	}
	if tagName == "Error" {
		return evalErrorComponent(propsStr, env)
	}
	if tagName == "Meta" {
		return evalMetaComponent(propsStr, env)
	}
	if tagName == "Select" {
		return evalSelectComponent(propsStr, env)
	}

	// Look up the variable/function
	val, ok := env.Get(tagName)
	if !ok {
		if builtin, ok := getBuiltins()[tagName]; ok {
			val = builtin
		} else {
			return newUndefinedComponentError(tagName)
		}
	}

	// Check if component is null (common when import destructuring gets wrong name)
	if val == NULL || val == nil {
		perr := perrors.New("COMP-0001", map[string]any{"Name": tagName})
		return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data, Line: tok.Line, Column: tok.Column}
	}

	// If the value is a String (e.g., loaded SVG), return it directly
	if str, isString := val.(*String); isString {
		return str
	}

	// Parse props into a dictionary
	// propsStr is everything after the tag name, calculate props column
	propsCol := tok.Column + 1 + len(tagName) + 1 // "<" + tagName + " "
	props := parseTagProps(propsStr, env, tok.Line, propsCol)
	if isError(props) {
		return props
	}

	// Call the function with the props dictionary, passing environment for runtime context
	result := ApplyFunctionWithEnv(val, []Object{props}, env)

	// Improve error message if function call failed
	if err, isErr := result.(*Error); isErr && strings.Contains(err.Message, "cannot call") {
		perr := perrors.New("COMP-0002", map[string]any{"Name": tagName, "Got": string(val.Type())})
		return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data, Line: tok.Line, Column: tok.Column}
	}

	return result
}

// parseTagProps parses tag properties into a dictionary
// baseLine and baseCol are optional position offsets for error reporting (default 0)
func parseTagProps(propsStr string, env *Environment, basePos ...int) Object {
	baseLine := 0
	baseCol := 0
	if len(basePos) >= 1 {
		baseLine = basePos[0]
	}
	if len(basePos) >= 2 {
		baseCol = basePos[1]
	}

	pairs := make(map[string]ast.Expression)

	i := 0
	for i < len(propsStr) {
		// Skip whitespace
		for i < len(propsStr) && unicode.IsSpace(rune(propsStr[i])) {
			i++
		}
		if i >= len(propsStr) {
			break
		}

		// Check for spread operator at prop level: ...expr
		if i+3 <= len(propsStr) && propsStr[i] == '.' && propsStr[i+1] == '.' && propsStr[i+2] == '.' {
			i += 3 // skip ...

			// Read the expression (identifier or complex expression)
			exprStart := i
			for i < len(propsStr) && !unicode.IsSpace(rune(propsStr[i])) {
				i++
			}

			exprStr := propsStr[exprStart:i]

			// Parse and evaluate the spread expression
			l := lexer.NewWithFilename(exprStr, env.Filename)
			p := parser.New(l)
			program := p.ParseProgram()

			if errs := p.StructuredErrors(); len(errs) > 0 {
				perr := errs[0]
				errLine := perr.Line
				errCol := perr.Column
				if baseLine > 0 {
					errLine = baseLine
					errCol = baseCol + exprStart + (perr.Column - 1)
				}
				return &Error{
					Class:   ClassParse,
					Code:    perr.Code,
					Message: perr.Message,
					Hints:   perr.Hints,
					Line:    errLine,
					Column:  errCol,
					File:    env.Filename,
					Data:    perr.Data,
				}
			}

			if len(program.Statements) > 0 {
				if exprStmt, ok := program.Statements[0].(*ast.ExpressionStatement); ok {
					// Evaluate the spread expression
					spreadObj := Eval(exprStmt.Expression, env)
					if isError(spreadObj) {
						// Adjust runtime error position
						if errObj, ok := spreadObj.(*Error); ok && baseLine > 0 && errObj.Line <= 1 {
							errObj.Line = baseLine
							errObj.Column = baseCol + exprStart + (errObj.Column - 1)
							if errObj.File == "" {
								errObj.File = env.Filename
							}
						}
						return spreadObj
					}

					// Get dictionary to spread (Record spreads its data)
					var spreadDict *Dictionary
					switch v := spreadObj.(type) {
					case *Dictionary:
						spreadDict = v
					case *Record:
						spreadDict = v.ToDictionary()
					default:
						perr := perrors.New("SPREAD-0001", map[string]any{"Got": string(spreadObj.Type())})
						errLine := 0
						errCol := 0
						if baseLine > 0 {
							errLine = baseLine
							errCol = baseCol + exprStart
						}
						return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Line: errLine, Column: errCol, File: env.Filename, Data: perr.Data}
					}

					// Evaluate each property in the spread dict's environment
					// and wrap it as a literal expression for the new dictionary
					for key, expr := range spreadDict.Pairs {
						// Evaluate in the spread dict's environment to get the actual value
						value := Eval(expr, spreadDict.Env)
						if isError(value) {
							return value
						}
						// Wrap the evaluated value as a literal expression
						pairs[key] = objectToExpression(value)
					}
				}
			}
			continue
		}

		// Read prop name
		nameStart := i
		for i < len(propsStr) && !unicode.IsSpace(rune(propsStr[i])) && propsStr[i] != '=' {
			i++
		}
		if nameStart == i {
			break
		}
		propName := propsStr[nameStart:i]

		// Skip whitespace
		for i < len(propsStr) && unicode.IsSpace(rune(propsStr[i])) {
			i++
		}

		// Check for = or standalone prop
		if i >= len(propsStr) || propsStr[i] != '=' {
			// Standalone prop (boolean)
			pairs[propName] = &ast.Boolean{Value: true}
			continue
		}

		i++ // skip =

		// Skip whitespace
		for i < len(propsStr) && unicode.IsSpace(rune(propsStr[i])) {
			i++
		}

		if i >= len(propsStr) {
			break
		}

		// Read prop value
		var valueStr string
		if propsStr[i] == '"' {
			// Quoted string - always treated as a literal (no interpolation)
			// Use prop={expr} syntax for expressions
			i++ // skip opening quote
			valueStart := i
			for i < len(propsStr) && propsStr[i] != '"' {
				if propsStr[i] == '\\' {
					i += 2
				} else {
					i++
				}
			}
			valueStr = propsStr[valueStart:i]
			if i < len(propsStr) {
				i++ // skip closing quote
			}
			pairs[propName] = &ast.StringLiteral{Value: valueStr}
		} else if propsStr[i] == '{' {
			// Expression in braces
			i++ // skip {

			// Check for spread operator ...expr
			if i+3 <= len(propsStr) && propsStr[i] == '.' && propsStr[i+1] == '.' && propsStr[i+2] == '.' {
				i += 3 // skip ...
				exprStart := i
				braceCount := 1

				for i < len(propsStr) && braceCount > 0 {
					if propsStr[i] == '{' {
						braceCount++
					} else if propsStr[i] == '}' {
						braceCount--
					}
					if braceCount > 0 {
						i++
					}
				}

				if braceCount != 0 {
					return newParseError("PARSE-0009", "tag spread operator", nil)
				}

				exprStr := propsStr[exprStart:i]
				i++ // skip }

				// Parse and evaluate the spread expression (with filename for error reporting)
				l := lexer.NewWithFilename(exprStr, env.Filename)
				p := parser.New(l)
				program := p.ParseProgram()

				if errs := p.StructuredErrors(); len(errs) > 0 {
					// Return first parse error with adjusted position
					perr := errs[0]
					errLine := perr.Line
					errCol := perr.Column
					if baseLine > 0 {
						errLine = baseLine
						errCol = baseCol + exprStart + (perr.Column - 1)
					}
					return &Error{
						Class:   ClassParse,
						Code:    perr.Code,
						Message: perr.Message,
						Hints:   perr.Hints,
						Line:    errLine,
						Column:  errCol,
						File:    env.Filename,
						Data:    perr.Data,
					}
				}

				if len(program.Statements) > 0 {
					if exprStmt, ok := program.Statements[0].(*ast.ExpressionStatement); ok {
						// Evaluate the spread expression immediately
						spreadObj := Eval(exprStmt.Expression, env)
						if isError(spreadObj) {
							// Adjust runtime error position
							if errObj, ok := spreadObj.(*Error); ok && baseLine > 0 && errObj.Line <= 1 {
								errObj.Line = baseLine
								errObj.Column = baseCol + exprStart + (errObj.Column - 1)
								if errObj.File == "" {
									errObj.File = env.Filename
								}
							}
							return spreadObj
						}

						// Get dictionary to spread (Record spreads its data)
						var spreadDict *Dictionary
						switch v := spreadObj.(type) {
						case *Dictionary:
							spreadDict = v
						case *Record:
							spreadDict = v.ToDictionary()
						default:
							perr := perrors.New("SPREAD-0001", map[string]any{"Got": string(spreadObj.Type())})
							return &Error{Class: ErrorClass(perr.Class), Code: perr.Code, Message: perr.Message, Hints: perr.Hints, Data: perr.Data}
						}

						// Evaluate each property in the spread dict's environment
						// and wrap it as a literal expression for the new dictionary
						for key, expr := range spreadDict.Pairs {
							// Evaluate in the spread dict's environment to get the actual value
							value := Eval(expr, spreadDict.Env)
							if isError(value) {
								return value
							}
							// Wrap the evaluated value as a literal expression
							pairs[key] = objectToExpression(value)
						}
					}
				}
				continue
			}

			braceCount := 1
			exprStart := i

			for i < len(propsStr) && braceCount > 0 {
				if propsStr[i] == '"' {
					// Skip quoted strings
					i++
					for i < len(propsStr) && propsStr[i] != '"' {
						if propsStr[i] == '\\' {
							i += 2
						} else {
							i++
						}
					}
					if i < len(propsStr) {
						i++
					}
					continue
				}
				if propsStr[i] == '{' {
					braceCount++
				} else if propsStr[i] == '}' {
					braceCount--
				}
				if braceCount > 0 {
					i++
				}
			}

			if braceCount != 0 {
				return newParseError("PARSE-0009", "tag prop", nil)
			}

			exprStr := propsStr[exprStart:i]
			i++ // skip }

			// Parse the expression (with filename for error reporting)
			l := lexer.NewWithFilename(exprStr, env.Filename)
			p := parser.New(l)
			program := p.ParseProgram()

			if errs := p.StructuredErrors(); len(errs) > 0 {
				// Return first parse error with adjusted position
				perr := errs[0]
				errLine := perr.Line
				errCol := perr.Column
				if baseLine > 0 {
					errLine = baseLine
					errCol = baseCol + exprStart + (perr.Column - 1)
				}
				return &Error{
					Class:   ClassParse,
					Code:    perr.Code,
					Message: perr.Message,
					Hints:   perr.Hints,
					Line:    errLine,
					Column:  errCol,
					File:    env.Filename,
					Data:    perr.Data,
				}
			}

			// Store as expression statement
			if len(program.Statements) > 0 {
				if exprStmt, ok := program.Statements[0].(*ast.ExpressionStatement); ok {
					pairs[propName] = exprStmt.Expression
				}
			}
		}
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// String conversion functions moved to eval_string_conversions.go:
// - objectToTemplateString
// - evalDictionarySpread
// - objectToUserString
// - objectToPrintString
// - ObjectToPrintString (exported)
// - objectToDebugString
