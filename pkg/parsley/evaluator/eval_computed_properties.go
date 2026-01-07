package evaluator

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

// evalUrlComputedProperty returns computed properties for URL dictionaries
// Returns nil if the property doesn't exist
func evalUrlComputedProperty(dict *Dictionary, key string, env *Environment) Object {
	switch key {
	case "origin":
		// scheme://host[:port]
		var result strings.Builder

		if schemeExpr, ok := dict.Pairs["scheme"]; ok {
			schemeObj := Eval(schemeExpr, env)
			if str, ok := schemeObj.(*String); ok {
				result.WriteString(str.Value)
				result.WriteString("://")
			}
		}

		if hostExpr, ok := dict.Pairs["host"]; ok {
			hostObj := Eval(hostExpr, env)
			if str, ok := hostObj.(*String); ok {
				result.WriteString(str.Value)
			}
		}

		if portExpr, ok := dict.Pairs["port"]; ok {
			portObj := Eval(portExpr, env)
			if i, ok := portObj.(*Integer); ok && i.Value != 0 {
				result.WriteString(":")
				result.WriteString(strconv.FormatInt(i.Value, 10))
			}
		}

		return &String{Value: result.String()}

	case "pathname":
		// Just the path part as a string (always with leading /)
		if pathExpr, ok := dict.Pairs["path"]; ok {
			pathObj := Eval(pathExpr, env)
			if arr, ok := pathObj.(*Array); ok {
				var parts []string
				for _, elem := range arr.Elements {
					if str, ok := elem.(*String); ok && str.Value != "" {
						parts = append(parts, str.Value)
					}
				}
				// URL paths always start with /
				return &String{Value: "/" + strings.Join(parts, "/")}
			}
		}
		return &String{Value: "/"}

	case "hostname":
		// Alias for host
		if hostExpr, ok := dict.Pairs["host"]; ok {
			return Eval(hostExpr, env)
		}
		return &String{Value: ""}

	case "protocol":
		// Scheme with colon suffix (e.g., "https:")
		if schemeExpr, ok := dict.Pairs["scheme"]; ok {
			schemeObj := Eval(schemeExpr, env)
			if str, ok := schemeObj.(*String); ok {
				return &String{Value: str.Value + ":"}
			}
		}
		return &String{Value: ""}

	case "search":
		// Query string with ? prefix (e.g., "?key=value&foo=bar")
		if queryExpr, ok := dict.Pairs["query"]; ok {
			queryObj := Eval(queryExpr, env)
			if queryDict, ok := queryObj.(*Dictionary); ok {
				if len(queryDict.Pairs) == 0 {
					return &String{Value: ""}
				}
				var result strings.Builder
				result.WriteString("?")
				first := true
				for key, expr := range queryDict.Pairs {
					val := Eval(expr, env)
					if str, ok := val.(*String); ok {
						if !first {
							result.WriteString("&")
						}
						result.WriteString(key)
						result.WriteString("=")
						result.WriteString(str.Value)
						first = false
					}
				}
				return &String{Value: result.String()}
			}
		}
		return &String{Value: ""}

	case "href":
		// Full URL as string (alias for toString)
		return &String{Value: urlDictToString(dict)}

	case "string":
		// Full URL as string (alias for href)
		return &String{Value: urlDictToString(dict)}
	}

	return nil // Property doesn't exist
}
func evalDatetimeComputedProperty(dict *Dictionary, key string, env *Environment) Object {
	switch key {
	case "date":
		// Just the date part as string (YYYY-MM-DD)
		if yearExpr, ok := dict.Pairs["year"]; ok {
			if monthExpr, ok := dict.Pairs["month"]; ok {
				if dayExpr, ok := dict.Pairs["day"]; ok {
					year := Eval(yearExpr, env)
					month := Eval(monthExpr, env)
					day := Eval(dayExpr, env)
					if yInt, ok := year.(*Integer); ok {
						if mInt, ok := month.(*Integer); ok {
							if dInt, ok := day.(*Integer); ok {
								return &String{Value: fmt.Sprintf("%04d-%02d-%02d", yInt.Value, mInt.Value, dInt.Value)}
							}
						}
					}
				}
			}
		}
		return NULL

	case "time":
		// Just the time part as string (HH:MM:SS or HH:MM if seconds are zero)
		if hourExpr, ok := dict.Pairs["hour"]; ok {
			if minExpr, ok := dict.Pairs["minute"]; ok {
				if secExpr, ok := dict.Pairs["second"]; ok {
					hour := Eval(hourExpr, env)
					minute := Eval(minExpr, env)
					second := Eval(secExpr, env)
					if hInt, ok := hour.(*Integer); ok {
						if mInt, ok := minute.(*Integer); ok {
							if sInt, ok := second.(*Integer); ok {
								if sInt.Value == 0 {
									return &String{Value: fmt.Sprintf("%02d:%02d", hInt.Value, mInt.Value)}
								}
								return &String{Value: fmt.Sprintf("%02d:%02d:%02d", hInt.Value, mInt.Value, sInt.Value)}
							}
						}
					}
				}
			}
		}
		return NULL

	case "format":
		// Human-readable format: "Month DD, YYYY" or "Month DD, YYYY at HH:MM"
		//
		// Note: THIS IS A SIMPLE IMPLEMENTATION
		// as it does not handle localization.
		//
		if yearExpr, ok := dict.Pairs["year"]; ok {
			if monthExpr, ok := dict.Pairs["month"]; ok {
				if dayExpr, ok := dict.Pairs["day"]; ok {
					year := Eval(yearExpr, env)
					month := Eval(monthExpr, env)
					day := Eval(dayExpr, env)
					if yInt, ok := year.(*Integer); ok {
						if mInt, ok := month.(*Integer); ok {
							if dInt, ok := day.(*Integer); ok {
								monthNames := []string{
									"January", "February", "March", "April", "May", "June",
									"July", "August", "September", "October", "November", "December",
								}
								monthName := "Invalid"
								if mInt.Value >= 1 && mInt.Value <= 12 {
									monthName = monthNames[mInt.Value-1]
								}

								// Check if time is set (not all zeros)
								hasTime := false
								if hourExpr, ok := dict.Pairs["hour"]; ok {
									if minExpr, ok := dict.Pairs["minute"]; ok {
										hour := Eval(hourExpr, env)
										minute := Eval(minExpr, env)
										if hInt, ok := hour.(*Integer); ok {
											if mInt, ok := minute.(*Integer); ok {
												if hInt.Value != 0 || mInt.Value != 0 {
													hasTime = true
													timeStr := fmt.Sprintf("%02d:%02d", hInt.Value, mInt.Value)
													return &String{Value: fmt.Sprintf("%s %d, %d at %s", monthName, dInt.Value, yInt.Value, timeStr)}
												}
											}
										}
									}
								}

								if !hasTime {
									return &String{Value: fmt.Sprintf("%s %d, %d", monthName, dInt.Value, yInt.Value)}
								}
							}
						}
					}
				}
			}
		}
		return NULL

	case "timestamp":
		// Alias for unix field - more intuitive name
		if unixExpr, ok := dict.Pairs["unix"]; ok {
			return Eval(unixExpr, env)
		}
		return NULL

	case "dayOfYear":
		// Calculate day of year (1-366)
		if unixExpr, ok := dict.Pairs["unix"]; ok {
			unixObj := Eval(unixExpr, env)
			if unixInt, ok := unixObj.(*Integer); ok {
				t := time.Unix(unixInt.Value, 0).UTC()
				return &Integer{Value: int64(t.YearDay())}
			}
		}
		return NULL

	case "week":
		// ISO week number (1-53)
		if unixExpr, ok := dict.Pairs["unix"]; ok {
			unixObj := Eval(unixExpr, env)
			if unixInt, ok := unixObj.(*Integer); ok {
				t := time.Unix(unixInt.Value, 0).UTC()
				_, week := t.ISOWeek()
				return &Integer{Value: int64(week)}
			}
		}
		return NULL
	}

	return nil // Property doesn't exist
}

// evalDurationComputedProperty returns computed properties for duration dictionaries
// Returns nil if the property doesn't exist
func evalDurationComputedProperty(dict *Dictionary, key string, env *Environment) Object {
	// Get the months and seconds components
	monthsExpr, hasMonths := dict.Pairs["months"]
	secondsExpr, hasSeconds := dict.Pairs["seconds"]

	if !hasMonths || !hasSeconds {
		return nil
	}

	monthsObj := Eval(monthsExpr, env)
	secondsObj := Eval(secondsExpr, env)

	monthsInt, monthsOk := monthsObj.(*Integer)
	secondsInt, secondsOk := secondsObj.(*Integer)

	if !monthsOk || !secondsOk {
		return nil
	}

	// For month-based durations, computed properties return null
	// because months have variable lengths (28-31 days)
	if monthsInt.Value != 0 {
		switch key {
		case "days", "hours", "minutes":
			return NULL
		}
	}

	switch key {
	case "days":
		// Total seconds as days (integer division)
		return &Integer{Value: secondsInt.Value / 86400}

	case "hours":
		// Total seconds as hours (integer division)
		return &Integer{Value: secondsInt.Value / 3600}

	case "minutes":
		// Total seconds as minutes (integer division)
		return &Integer{Value: secondsInt.Value / 60}
	}

	return nil // Property doesn't exist
}

// getPublicDirComponents extracts public_dir components from basil config in environment
