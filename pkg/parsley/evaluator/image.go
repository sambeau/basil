package evaluator

import (
	"path/filepath"
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
