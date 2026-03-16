package evaluator

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// NewImageBuiltin creates the image() builtin function.
// This function transforms images and returns content-hashed public URLs.
func NewImageBuiltin() *StdlibBuiltin {
	return &StdlibBuiltin{
		Name: "image",
		Fn:   evalImage,
	}
}

// NewImageInfoBuiltin creates the imageInfo() builtin function.
// This function returns metadata about an image without transforming it.
func NewImageInfoBuiltin() *StdlibBuiltin {
	return &StdlibBuiltin{
		Name: "imageInfo",
		Fn:   evalImageInfo,
	}
}

// NewImageBlurBuiltin creates the imageBlur() builtin function.
// This function generates a Low Quality Image Placeholder (LQIP) data URI.
func NewImageBlurBuiltin() *StdlibBuiltin {
	return &StdlibBuiltin{
		Name: "imageBlur",
		Fn:   evalImageBlur,
	}
}

// NewImageSrcsetBuiltin creates the imageSrcset() builtin function.
// This function generates multiple image variants and returns a dict with src, srcset, width, height.
func NewImageSrcsetBuiltin() *StdlibBuiltin {
	return &StdlibBuiltin{
		Name: "imageSrcset",
		Fn:   evalImageSrcset,
	}
}

// evalImage handles the image(@./path) and image(@./path, options) builtins.
// It transforms the image (if options provided), caches the result, and returns the public URL.
func evalImage(args []Object, env *Environment) Object {
	if len(args) < 1 || len(args) > 2 {
		return newArityErrorExact("image", len(args), 1, 2)
	}

	// Check if image registry is available
	if env.ImageRegistry == nil {
		return &Error{
			Class:   ErrorClass("state"),
			Message: "image() is only available in Basil server handlers",
			Hints:   []string{"This function requires the Basil server environment"},
		}
	}

	// Get path from first argument
	pathStr, err := extractImagePath(args[0], "image")
	if err != nil {
		return err
	}

	// Resolve the path relative to the current file
	absPath, pathErr := resolveImagePath(pathStr, env)
	if pathErr != nil {
		return pathErr
	}

	// Security check: ensure path is within handler root
	if secErr := checkImagePathSecurity(absPath, env, "image"); secErr != nil {
		return secErr
	}

	// Parse options if provided
	opts := make(map[string]any)
	if len(args) == 2 {
		optsDict, ok := args[1].(*Dictionary)
		if !ok {
			return newTypeError("TYPE-0012", "image", "a dictionary", args[1].Type())
		}

		// Convert Parsley dict to map[string]any
		for key, valExpr := range optsDict.Pairs {
			val := Eval(valExpr, optsDict.Env)
			if isError(val) {
				return val
			}
			opts[key] = imageObjectToGo(val)
		}
	}

	// Transform via registry
	url, transformErr := env.ImageRegistry.Transform(absPath, opts)
	if transformErr != nil {
		return &Error{
			Class:   ErrorClass("io"),
			Message: "image(): " + transformErr.Error(),
			Hints:   []string{"Check that the file exists and is a supported image format (JPEG, PNG, GIF, WebP)"},
		}
	}

	return &String{Value: url}
}

// evalImageBlur handles the imageBlur(@./path) builtin.
// It generates a Low Quality Image Placeholder (LQIP) and returns a data URI.
func evalImageBlur(args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("imageBlur", len(args), 1)
	}

	// Check if image registry is available
	if env.ImageRegistry == nil {
		return &Error{
			Class:   ErrorClass("state"),
			Message: "imageBlur() is only available in Basil server handlers",
			Hints:   []string{"This function requires the Basil server environment"},
		}
	}

	// Get path from argument
	pathStr, err := extractImagePath(args[0], "imageBlur")
	if err != nil {
		return err
	}

	// Resolve the path relative to the current file
	absPath, pathErr := resolveImagePath(pathStr, env)
	if pathErr != nil {
		return pathErr
	}

	// Security check: ensure path is within handler root
	if secErr := checkImagePathSecurity(absPath, env, "imageBlur"); secErr != nil {
		return secErr
	}

	// Generate blur placeholder via registry
	dataURI, blurErr := env.ImageRegistry.BlurPlaceholder(absPath)
	if blurErr != nil {
		return &Error{
			Class:   ErrorClass("io"),
			Message: "imageBlur(): " + blurErr.Error(),
			Hints:   []string{"Check that the file exists and is a supported image format (JPEG, PNG, GIF, WebP)"},
		}
	}

	return &String{Value: dataURI}
}

// evalImageSrcset handles the imageSrcset(@./path, style, widths) or imageSrcset(@./path, style, scales, "x") builtin.
// It generates multiple image variants and returns a dict with {src, srcset, width, height}.
func evalImageSrcset(args []Object, env *Environment) Object {
	if len(args) < 3 || len(args) > 4 {
		return newArityErrorExact("imageSrcset", len(args), 3, 4)
	}

	// Check if image registry is available
	if env.ImageRegistry == nil {
		return &Error{
			Class:   ErrorClass("state"),
			Message: "imageSrcset() is only available in Basil server handlers",
			Hints:   []string{"This function requires the Basil server environment"},
		}
	}

	// Get path from first argument
	pathStr, err := extractImagePath(args[0], "imageSrcset")
	if err != nil {
		return err
	}

	// Resolve the path relative to the current file
	absPath, pathErr := resolveImagePath(pathStr, env)
	if pathErr != nil {
		return pathErr
	}

	// Security check: ensure path is within handler root
	if secErr := checkImagePathSecurity(absPath, env, "imageSrcset"); secErr != nil {
		return secErr
	}

	// Parse style options (second argument)
	styleOpts := make(map[string]any)
	if args[1] != NULL {
		styleDict, ok := args[1].(*Dictionary)
		if !ok {
			return newTypeError("TYPE-0012", "imageSrcset", "a dictionary", args[1].Type())
		}
		for key, valExpr := range styleDict.Pairs {
			val := Eval(valExpr, styleDict.Env)
			if isError(val) {
				return val
			}
			styleOpts[key] = imageObjectToGo(val)
		}
	}

	// Parse widths/scales array (third argument)
	sizesArr, ok := args[2].(*Array)
	if !ok {
		return newTypeError("TYPE-0012", "imageSrcset", "an array", args[2].Type())
	}
	if len(sizesArr.Elements) == 0 {
		return &Error{
			Class:   ErrorClass("argument"),
			Message: "imageSrcset(): widths/scales array cannot be empty",
		}
	}

	// Check for density descriptor mode (4th arg = "x")
	densityMode := false
	if len(args) == 4 {
		modeStr, ok := args[3].(*String)
		if !ok || modeStr.Value != "x" {
			return &Error{
				Class:   ErrorClass("argument"),
				Message: "imageSrcset(): fourth argument must be \"x\" for density descriptor mode",
			}
		}
		densityMode = true
	}

	// Parse sizes as integers
	sizes := make([]int, 0, len(sizesArr.Elements))
	for i, elem := range sizesArr.Elements {
		var numVal int
		switch v := elem.(type) {
		case *Integer:
			numVal = int(v.Value)
		case *Float:
			numVal = int(v.Value)
		default:
			return &Error{
				Class:   ErrorClass("type"),
				Message: fmt.Sprintf("imageSrcset(): element %d must be a number, got %s", i, elem.Type()),
			}
		}
		if numVal <= 0 {
			return &Error{
				Class:   ErrorClass("argument"),
				Message: fmt.Sprintf("imageSrcset(): element %d must be positive, got %d", i, numVal),
			}
		}
		sizes = append(sizes, numVal)
	}

	// Get source image info for dimension clamping and aspect ratio
	info, infoErr := env.ImageRegistry.Info(absPath)
	if infoErr != nil {
		return &Error{
			Class:   ErrorClass("io"),
			Message: "imageSrcset(): " + infoErr.Error(),
			Hints:   []string{"Check that the file exists and is a supported image format"},
		}
	}
	srcWidth := int(info["width"].(int64))
	srcHeight := int(info["height"].(int64))

	// Density mode requires style.width
	var baseWidth int
	if densityMode {
		if w, ok := styleOpts["width"]; ok {
			switch v := w.(type) {
			case int:
				baseWidth = v
			case int64:
				baseWidth = int(v)
			case float64:
				baseWidth = int(v)
			}
		}
		if baseWidth <= 0 {
			return &Error{
				Class:   ErrorClass("argument"),
				Message: "imageSrcset(): density mode (\"x\") requires style.width to be set",
			}
		}
	}

	// Generate variants and build srcset
	type variant struct {
		url   string
		width int
	}

	var srcsetParts []string
	var srcURL string
	var resultWidth int

	if densityMode {
		// Density mode: map each scale to a clamped pixel width, generate
		// one transform per unique width, then build descriptors per scale.
		// This preserves correct scale labels even when clamping causes
		// multiple scales to map to the same pixel width.
		scaleWidths := make([]int, len(sizes))
		for i, scale := range sizes {
			w := baseWidth * scale
			if w > srcWidth {
				w = srcWidth
			}
			scaleWidths[i] = w
		}

		// Generate transform for each unique width
		urlByWidth := make(map[int]string)
		for _, w := range scaleWidths {
			if _, ok := urlByWidth[w]; ok {
				continue // already generated
			}
			opts := make(map[string]any)
			for k, v := range styleOpts {
				opts[k] = v
			}
			opts["width"] = w
			url, transformErr := env.ImageRegistry.Transform(absPath, opts)
			if transformErr != nil {
				return &Error{
					Class:   ErrorClass("io"),
					Message: "imageSrcset(): " + transformErr.Error(),
				}
			}
			urlByWidth[w] = url
		}

		// Build srcset with correct density descriptors
		for i, scale := range sizes {
			w := scaleWidths[i]
			srcsetParts = append(srcsetParts, fmt.Sprintf("%s %dx", urlByWidth[w], scale))
		}

		// src = 1x variant (first scale's width)
		srcURL = urlByWidth[scaleWidths[0]]
		resultWidth = scaleWidths[0]
	} else {
		// Width descriptor mode: clamp, deduplicate, sort, generate variants
		var targetWidths []int
		for _, w := range sizes {
			if w > srcWidth {
				w = srcWidth
			}
			targetWidths = append(targetWidths, w)
		}
		targetWidths = deduplicateInts(targetWidths)

		if len(targetWidths) == 0 {
			return &Error{
				Class:   ErrorClass("argument"),
				Message: "imageSrcset(): no valid widths after clamping to source dimensions",
			}
		}

		variants := make([]variant, 0, len(targetWidths))
		for _, w := range targetWidths {
			opts := make(map[string]any)
			for k, v := range styleOpts {
				opts[k] = v
			}
			opts["width"] = w
			url, transformErr := env.ImageRegistry.Transform(absPath, opts)
			if transformErr != nil {
				return &Error{
					Class:   ErrorClass("io"),
					Message: "imageSrcset(): " + transformErr.Error(),
				}
			}
			variants = append(variants, variant{url: url, width: w})
		}

		for _, v := range variants {
			srcsetParts = append(srcsetParts, fmt.Sprintf("%s %dw", v.url, v.width))
		}

		// src = variant matching style.width, or largest variant
		srcURL = variants[len(variants)-1].url
		resultWidth = variants[len(variants)-1].width

		if w, ok := styleOpts["width"]; ok {
			var styleWidth int
			switch v := w.(type) {
			case int:
				styleWidth = v
			case int64:
				styleWidth = int(v)
			case float64:
				styleWidth = int(v)
			}
			for _, v := range variants {
				if v.width == styleWidth || (styleWidth > srcWidth && v.width == srcWidth) {
					srcURL = v.url
					resultWidth = v.width
					break
				}
			}
		}
	}

	srcset := strings.Join(srcsetParts, ", ")
	var resultHeight int

	// Calculate height based on aspect ratio
	if srcWidth > 0 {
		resultHeight = (resultWidth * srcHeight) / srcWidth
	}

	// Build result dictionary
	pairs := map[string]ast.Expression{
		"src":    &ast.ObjectLiteralExpression{Obj: &String{Value: srcURL}},
		"srcset": &ast.ObjectLiteralExpression{Obj: &String{Value: srcset}},
		"width":  &ast.ObjectLiteralExpression{Obj: &Integer{Value: int64(resultWidth)}},
		"height": &ast.ObjectLiteralExpression{Obj: &Integer{Value: int64(resultHeight)}},
	}

	return &Dictionary{
		Pairs:    pairs,
		KeyOrder: []string{"src", "srcset", "width", "height"},
		Env:      env,
	}
}

// deduplicateInts removes duplicates from a slice and returns sorted results.
func deduplicateInts(s []int) []int {
	seen := make(map[int]bool)
	result := make([]int, 0, len(s))
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	// Sort for consistent srcset ordering
	sort.Ints(result)
	return result
}

// evalImageInfo handles the imageInfo(@./path) builtin.
// It returns metadata about an image without transforming it.
func evalImageInfo(args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("imageInfo", len(args), 1)
	}

	// Check if image registry is available
	if env.ImageRegistry == nil {
		return &Error{
			Class:   ErrorClass("state"),
			Message: "imageInfo() is only available in Basil server handlers",
			Hints:   []string{"This function requires the Basil server environment"},
		}
	}

	// Get path from argument
	pathStr, err := extractImagePath(args[0], "imageInfo")
	if err != nil {
		return err
	}

	// Resolve the path relative to the current file
	absPath, pathErr := resolveImagePath(pathStr, env)
	if pathErr != nil {
		return pathErr
	}

	// Security check: ensure path is within handler root
	if secErr := checkImagePathSecurity(absPath, env, "imageInfo"); secErr != nil {
		return secErr
	}

	// Get info via registry
	info, infoErr := env.ImageRegistry.Info(absPath)
	if infoErr != nil {
		return &Error{
			Class:   ErrorClass("io"),
			Message: "imageInfo(): " + infoErr.Error(),
			Hints:   []string{"Check that the file exists and is a supported image format (JPEG, PNG, GIF, WebP)"},
		}
	}

	// Convert map to Parsley dictionary
	return goMapToImageDict(info, env)
}

// extractImagePath extracts a path string from a Parsley argument (path dict or string).
func extractImagePath(arg Object, fnName string) (string, *Error) {
	switch v := arg.(type) {
	case *Dictionary:
		// Path dictionary (from @./file.jpg)
		if !isPathDict(v) {
			return "", newTypeError("TYPE-0012", fnName, "a path", DICTIONARY_OBJ)
		}
		return pathDictToString(v), nil
	case *String:
		// Plain string path
		return v.Value, nil
	default:
		return "", newTypeError("TYPE-0012", fnName, "a path", arg.Type())
	}
}

// resolveImagePath resolves a path relative to the current file and environment.
func resolveImagePath(pathStr string, env *Environment) (string, *Error) {
	var absPath string
	switch {
	case filepath.IsAbs(pathStr):
		absPath = pathStr
	case env.Filename != "":
		// Relative to current file's directory
		currentDir := filepath.Dir(env.Filename)
		absPath = filepath.Join(currentDir, pathStr)
	default:
		// No context, use as-is
		absPath = pathStr
	}

	// Clean and normalize the path
	absPath = filepath.Clean(absPath)
	return absPath, nil
}

// checkImagePathSecurity ensures a path is within the handler root.
func checkImagePathSecurity(absPath string, env *Environment, fnName string) *Error {
	if env.RootPath == "" {
		return nil // No root path set, allow all
	}

	// Both paths should be absolute for proper comparison
	rootAbs, _ := filepath.Abs(env.RootPath)
	pathAbs, _ := filepath.Abs(absPath)

	// Check if path starts with root (is within or under root directory)
	if !strings.HasPrefix(pathAbs, rootAbs+string(filepath.Separator)) && pathAbs != rootAbs {
		return &Error{
			Class:   ErrorClass("security"),
			Message: fnName + "(): path must be within handler directory",
			Hints:   []string{"Use relative paths like @./image.jpg", "Path traversal outside handler root is not allowed"},
		}
	}

	return nil
}

// imageObjectToGo converts a Parsley Object to a Go value.
func imageObjectToGo(obj Object) any {
	switch v := obj.(type) {
	case *Integer:
		return v.Value
	case *Float:
		return v.Value
	case *String:
		return v.Value
	case *Boolean:
		return v.Value
	case *Null:
		return nil
	default:
		return nil
	}
}

// goMapToImageDict converts a Go map to a Parsley Dictionary.
func goMapToImageDict(m map[string]any, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)
	keys := make([]string, 0, len(m))

	for key, val := range m {
		keys = append(keys, key)
		pairs[key] = &ast.ObjectLiteralExpression{Obj: goToImageObject(val)}
	}

	return &Dictionary{
		Pairs:    pairs,
		KeyOrder: keys,
		Env:      env,
	}
}

// goToImageObject converts a Go value to a Parsley Object.
func goToImageObject(val any) Object {
	switch v := val.(type) {
	case int:
		return &Integer{Value: int64(v)}
	case int64:
		return &Integer{Value: v}
	case float64:
		return &Float{Value: v}
	case string:
		return &String{Value: v}
	case bool:
		return &Boolean{Value: v}
	case nil:
		return NULL
	default:
		return NULL
	}
}
