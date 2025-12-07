package evaluator

import (
	"path/filepath"
	"strings"
)

// NewPublicURLBuiltin creates the publicUrl() builtin function.
// This function makes private files accessible via content-hashed public URLs.
func NewPublicURLBuiltin() *StdlibBuiltin {
	return &StdlibBuiltin{
		Name: "publicUrl",
		Fn:   evalPublicURL,
	}
}

// evalPublicURL handles the publicUrl(@./path) builtin.
// It registers the file with the asset registry and returns the public URL.
func evalPublicURL(args []Object, env *Environment) Object {
	if len(args) != 1 {
		return newArityError("publicUrl", len(args), 1)
	}

	// Check if asset registry is available
	if env.AssetRegistry == nil {
		return &Error{
			Class:   ErrorClass("state"),
			Message: "publicUrl() is only available in Basil server handlers",
			Hints:   []string{"This function requires the Basil server environment"},
		}
	}

	// Get path from argument
	var pathStr string
	switch arg := args[0].(type) {
	case *Dictionary:
		// Path dictionary (from @./file.svg)
		if !isPathDict(arg) {
			return newTypeError("TYPE-0012", "publicUrl", "a path", DICTIONARY_OBJ)
		}
		pathStr = pathDictToString(arg)
	case *String:
		// Plain string path
		pathStr = arg.Value
	default:
		return newTypeError("TYPE-0012", "publicUrl", "a path", arg.Type())
	}

	// Resolve relative path based on current file's directory
	var absPath string
	if filepath.IsAbs(pathStr) {
		absPath = pathStr
	} else if env.Filename != "" {
		// Relative to current file's directory
		currentDir := filepath.Dir(env.Filename)
		absPath = filepath.Join(currentDir, pathStr)
	} else {
		// No context, try current working directory
		absPath = pathStr
	}

	// Clean and normalize the path
	absPath = filepath.Clean(absPath)

	// Security check: ensure path is within handler root (RootPath)
	if env.RootPath != "" {
		// Both paths should be absolute for proper comparison
		rootAbs, _ := filepath.Abs(env.RootPath)
		pathAbs, _ := filepath.Abs(absPath)

		// Check if path starts with root (is within or under root directory)
		if !strings.HasPrefix(pathAbs, rootAbs+string(filepath.Separator)) && pathAbs != rootAbs {
			return &Error{
				Class:   ErrorClass("security"),
				Message: "publicUrl(): path must be within handler directory",
				Hints:   []string{"Use relative paths like @./asset.svg", "Path traversal outside handler root is not allowed"},
			}
		}
	}

	// Register with asset registry
	url, err := env.AssetRegistry.Register(absPath)
	if err != nil {
		return newIOError("IO-0001", absPath, err)
	}

	return &String{Value: url}
}
