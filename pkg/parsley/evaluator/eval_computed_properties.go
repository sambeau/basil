package evaluator

import (
	"os"
	"path/filepath"
	"strings"
)

// evalPathComputedProperty returns computed properties for path dictionaries
// Returns nil if the property doesn't exist
func evalPathComputedProperty(dict *Dictionary, key string, env *Environment) Object {
	switch key {
	case "basename":
		// Get last component
		componentsExpr, ok := dict.Pairs["segments"]
		if !ok {
			return NULL
		}
		componentsObj := Eval(componentsExpr, env)
		arr, ok := componentsObj.(*Array)
		if !ok || len(arr.Elements) == 0 {
			return NULL
		}
		return arr.Elements[len(arr.Elements)-1]

	case "dirname", "parent":
		// Get all but last component, return as path dict
		componentsExpr, ok := dict.Pairs["segments"]
		if !ok {
			return NULL
		}
		componentsObj := Eval(componentsExpr, env)
		arr, ok := componentsObj.(*Array)
		if !ok || len(arr.Elements) == 0 {
			return NULL
		}

		// Get absolute flag
		absoluteExpr, ok := dict.Pairs["absolute"]
		isAbsolute := false
		if ok {
			absoluteObj := Eval(absoluteExpr, env)
			if b, ok := absoluteObj.(*Boolean); ok {
				isAbsolute = b.Value
			}
		}

		// Create new components array (all but last)
		parentComponents := []string{}
		for i := 0; i < len(arr.Elements)-1; i++ {
			if str, ok := arr.Elements[i].(*String); ok {
				parentComponents = append(parentComponents, str.Value)
			}
		}

		return pathToDict(parentComponents, isAbsolute, env)

	case "extension", "ext":
		// Get extension from basename
		componentsExpr, ok := dict.Pairs["segments"]
		if !ok {
			return NULL
		}
		componentsObj := Eval(componentsExpr, env)
		arr, ok := componentsObj.(*Array)
		if !ok || len(arr.Elements) == 0 {
			return NULL
		}
		basename, ok := arr.Elements[len(arr.Elements)-1].(*String)
		if !ok {
			return NULL
		}

		// Find last dot
		lastDot := strings.LastIndex(basename.Value, ".")
		if lastDot == -1 || lastDot == 0 {
			return &String{Value: ""}
		}
		return &String{Value: basename.Value[lastDot+1:]}

	case "stem":
		// Get filename without extension
		componentsExpr, ok := dict.Pairs["segments"]
		if !ok {
			return NULL
		}
		componentsObj := Eval(componentsExpr, env)
		arr, ok := componentsObj.(*Array)
		if !ok || len(arr.Elements) == 0 {
			return NULL
		}
		basename, ok := arr.Elements[len(arr.Elements)-1].(*String)
		if !ok {
			return NULL
		}

		// Find last dot
		lastDot := strings.LastIndex(basename.Value, ".")
		if lastDot == -1 || lastDot == 0 {
			return basename
		}
		return &String{Value: basename.Value[:lastDot]}

	case "name":
		// Alias for basename
		return evalPathComputedProperty(dict, "basename", env)

	case "filename":
		// Alias for basename (more intuitive name)
		return evalPathComputedProperty(dict, "basename", env)

	case "suffix":
		// Alias for extension
		return evalPathComputedProperty(dict, "extension", env)

	case "suffixes":
		// Get all extensions as array (e.g., ["tar", "gz"] from file.tar.gz)
		componentsExpr, ok := dict.Pairs["segments"]
		if !ok {
			return NULL
		}
		componentsObj := Eval(componentsExpr, env)
		arr, ok := componentsObj.(*Array)
		if !ok || len(arr.Elements) == 0 {
			return &Array{Elements: []Object{}}
		}
		basename, ok := arr.Elements[len(arr.Elements)-1].(*String)
		if !ok {
			return &Array{Elements: []Object{}}
		}

		// Find all dots and extract suffixes
		var suffixes []Object
		parts := strings.Split(basename.Value, ".")
		if len(parts) > 1 {
			// Skip the first part (filename), collect rest as suffixes
			for i := 1; i < len(parts); i++ {
				if parts[i] != "" {
					suffixes = append(suffixes, &String{Value: parts[i]})
				}
			}
		}
		return &Array{Elements: suffixes}

	case "parts":
		// Alias for components
		componentsExpr, ok := dict.Pairs["segments"]
		if !ok {
			return NULL
		}
		return Eval(componentsExpr, env)

	case "isAbsolute":
		// Boolean indicating if path is absolute
		absoluteExpr, ok := dict.Pairs["absolute"]
		if !ok {
			return FALSE
		}
		return Eval(absoluteExpr, env)

	case "isRelative":
		// Boolean indicating if path is relative (opposite of absolute)
		absoluteExpr, ok := dict.Pairs["absolute"]
		if !ok {
			return TRUE
		}
		absoluteObj := Eval(absoluteExpr, env)
		if b, ok := absoluteObj.(*Boolean); ok {
			return nativeBoolToParsBoolean(!b.Value)
		}
		return TRUE

	case "string":
		// Full path as string
		return &String{Value: pathDictToString(dict)}

	case "dir":
		// Directory path as string (all but the last component)
		componentsExpr, ok := dict.Pairs["segments"]
		if !ok {
			return &String{Value: ""}
		}
		componentsObj := Eval(componentsExpr, env)
		arr, ok := componentsObj.(*Array)
		if !ok || len(arr.Elements) <= 1 {
			// If only one component (or empty), dir is empty or root
			absoluteExpr, ok := dict.Pairs["absolute"]
			isAbsolute := false
			if ok {
				absoluteObj := Eval(absoluteExpr, env)
				if b, ok := absoluteObj.(*Boolean); ok {
					isAbsolute = b.Value
				}
			}
			if isAbsolute {
				return &String{Value: "/"}
			}
			return &String{Value: "."}
		}

		// Get absolute flag
		absoluteExpr, ok := dict.Pairs["absolute"]
		isAbsolute := false
		if ok {
			absoluteObj := Eval(absoluteExpr, env)
			if b, ok := absoluteObj.(*Boolean); ok {
				isAbsolute = b.Value
			}
		}

		// Build directory path (all but last component)
		var result strings.Builder
		for i := 0; i < len(arr.Elements)-1; i++ {
			if str, ok := arr.Elements[i].(*String); ok {
				if str.Value == "" && i == 0 && isAbsolute {
					result.WriteString("/")
				} else {
					if i > 0 && (i > 1 || !isAbsolute) {
						result.WriteString("/")
					}
					result.WriteString(str.Value)
				}
			}
		}
		return &String{Value: result.String()}
	}

	return nil // Property doesn't exist
}

// evalDirComputedProperty returns computed properties for directory dictionaries
// Returns nil if the property doesn't exist
func evalDirComputedProperty(dict *Dictionary, key string, env *Environment) Object {
	pathStr := getFilePathString(dict, env)

	switch key {
	case "path":
		// Return the underlying path dictionary
		compExpr, ok := dict.Pairs["_pathComponents"]
		if !ok {
			return NULL
		}
		compObj := Eval(compExpr, env)
		arr, ok := compObj.(*Array)
		if !ok {
			return NULL
		}

		absExpr, ok := dict.Pairs["_pathAbsolute"]
		isAbsolute := false
		if ok {
			absObj := Eval(absExpr, env)
			if b, ok := absObj.(*Boolean); ok {
				isAbsolute = b.Value
			}
		}

		components := []string{}
		for _, elem := range arr.Elements {
			if str, ok := elem.(*String); ok {
				components = append(components, str.Value)
			}
		}

		return pathToDict(components, isAbsolute, env)

	case "exists":
		info, err := os.Stat(pathStr)
		return nativeBoolToParsBoolean(err == nil && info.IsDir())

	case "isDir":
		info, err := os.Stat(pathStr)
		if err != nil {
			return FALSE
		}
		return nativeBoolToParsBoolean(info.IsDir())

	case "isFile":
		return FALSE // Directories are not files

	case "name", "basename":
		return &String{Value: filepath.Base(pathStr)}

	case "parent", "dirname":
		dir := filepath.Dir(pathStr)
		components, isAbsolute := parsePathString(dir)
		return pathToDict(components, isAbsolute, env)

	case "mode":
		info, err := os.Stat(pathStr)
		if err != nil {
			return &String{Value: ""}
		}
		return &String{Value: info.Mode().String()}

	case "modified":
		info, err := os.Stat(pathStr)
		if err != nil {
			return NULL
		}
		return timeToDatetimeDict(info.ModTime(), env)

	case "files":
		// Return array of file handles in directory
		return readDirContents(pathStr, env)

	case "count":
		// Return count of items in directory
		entries, err := os.ReadDir(pathStr)
		if err != nil {
			return &Integer{Value: 0}
		}
		return &Integer{Value: int64(len(entries))}
	}

	return nil // Property doesn't exist
}

// evalFileComputedProperty returns computed properties for file dictionaries
// Returns nil if the property doesn't exist
func evalFileComputedProperty(dict *Dictionary, key string, env *Environment) Object {
	pathStr := getFilePathString(dict, env)

	switch key {
	case "path":
		// Return the underlying path dictionary
		compExpr, ok := dict.Pairs["_pathComponents"]
		if !ok {
			return NULL
		}
		compObj := Eval(compExpr, env)
		arr, ok := compObj.(*Array)
		if !ok {
			return NULL
		}

		absExpr, ok := dict.Pairs["_pathAbsolute"]
		isAbsolute := false
		if ok {
			absObj := Eval(absExpr, env)
			if b, ok := absObj.(*Boolean); ok {
				isAbsolute = b.Value
			}
		}

		components := []string{}
		for _, elem := range arr.Elements {
			if str, ok := elem.(*String); ok {
				components = append(components, str.Value)
			}
		}

		return pathToDict(components, isAbsolute, env)

	case "exists":
		_, err := os.Stat(pathStr)
		return nativeBoolToParsBoolean(err == nil)

	case "size":
		info, err := os.Stat(pathStr)
		if err != nil {
			return &Integer{Value: 0}
		}
		return &Integer{Value: info.Size()}

	case "modified":
		info, err := os.Stat(pathStr)
		if err != nil {
			return NULL
		}
		return timeToDatetimeDict(info.ModTime(), env)

	case "isDir":
		info, err := os.Stat(pathStr)
		if err != nil {
			return FALSE
		}
		return nativeBoolToParsBoolean(info.IsDir())

	case "isFile":
		info, err := os.Stat(pathStr)
		if err != nil {
			return FALSE
		}
		return nativeBoolToParsBoolean(!info.IsDir())

	case "mode":
		info, err := os.Stat(pathStr)
		if err != nil {
			return &String{Value: ""}
		}
		return &String{Value: info.Mode().String()}

	case "ext", "extension":
		ext := filepath.Ext(pathStr)
		if len(ext) > 0 && ext[0] == '.' {
			ext = ext[1:]
		}
		return &String{Value: ext}

	case "basename", "name":
		return &String{Value: filepath.Base(pathStr)}

	case "dirname", "parent":
		dir := filepath.Dir(pathStr)
		components, isAbsolute := parsePathString(dir)
		return pathToDict(components, isAbsolute, env)

	case "stem":
		base := filepath.Base(pathStr)
		ext := filepath.Ext(base)
		return &String{Value: strings.TrimSuffix(base, ext)}
	}

	return nil // Property doesn't exist
}
