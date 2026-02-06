package evaluator

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"gopkg.in/yaml.v3"
)

// File encoding functions for various output formats

func encodeText(value Object) ([]byte, error) {
	switch v := value.(type) {
	case *String:
		return []byte(v.Value), nil
	default:
		return []byte(value.Inspect()), nil
	}
}

// encodeBytes encodes a value as bytes
func encodeBytes(value Object) ([]byte, error) {
	arr, ok := value.(*Array)
	if !ok {
		return nil, fmt.Errorf("bytes format requires an array, got %s", strings.ToLower(string(value.Type())))
	}

	data := make([]byte, len(arr.Elements))
	for i, elem := range arr.Elements {
		intVal, ok := elem.(*Integer)
		if !ok {
			return nil, fmt.Errorf("bytes array must contain integers, got %s at index %d", strings.ToLower(string(elem.Type())), i)
		}
		if intVal.Value < 0 || intVal.Value > 255 {
			return nil, fmt.Errorf("byte value out of range (0-255): %d at index %d", intVal.Value, i)
		}
		data[i] = byte(intVal.Value)
	}
	return data, nil
}

// encodeLines encodes a value as lines
func encodeLines(value Object, appendMode bool) ([]byte, error) {
	arr, ok := value.(*Array)
	if !ok {
		// Single value - treat as single line
		if appendMode {
			return []byte(value.Inspect() + "\n"), nil
		}
		return []byte(value.Inspect()), nil
	}

	var builder strings.Builder
	for i, elem := range arr.Elements {
		if i > 0 {
			builder.WriteString("\n")
		}
		switch v := elem.(type) {
		case *String:
			builder.WriteString(v.Value)
		default:
			builder.WriteString(elem.Inspect())
		}
	}
	return []byte(builder.String()), nil
}

// encodeJSON encodes a value as JSON
func encodeJSON(value Object) ([]byte, error) {
	goValue := objectToGo(value)
	return json.MarshalIndent(goValue, "", "  ")
}

// objectToGo converts a Parsley Object to a Go interface{} for JSON encoding
func objectToGo(obj Object) interface{} {
	switch v := obj.(type) {
	case *Null:
		return nil
	case *Boolean:
		return v.Value
	case *Integer:
		return v.Value
	case *Float:
		return v.Value
	case *String:
		return v.Value
	case *Array:
		result := make([]interface{}, len(v.Elements))
		for i, elem := range v.Elements {
			result[i] = objectToGo(elem)
		}
		return result
	case *Dictionary:
		result := make(map[string]interface{})
		for key, expr := range v.Pairs {
			// Skip internal fields
			if strings.HasPrefix(key, "_") {
				continue
			}
			// Evaluate expression if it's an ObjectLiteralExpression
			if ole, ok := expr.(*ast.ObjectLiteralExpression); ok {
				result[key] = objectToGo(ole.Obj.(Object))
			} else {
				// For other expressions, we need to evaluate them
				env := NewEnvironment()
				val := Eval(expr, env)
				result[key] = objectToGo(val)
			}
		}
		return result
	default:
		return obj.Inspect()
	}
}

// encodeSVG encodes a value as SVG (text format, for writing)
func encodeSVG(value Object) ([]byte, error) {
	switch v := value.(type) {
	case *String:
		return []byte(v.Value), nil
	default:
		// Convert to string representation
		return []byte(value.Inspect()), nil
	}
}

// encodeYAML encodes a value as YAML
func encodeYAML(value Object) ([]byte, error) {
	goValue := objectToGo(value)
	return yaml.Marshal(goValue)
}

// encodeCSV encodes a value as CSV
func encodeCSV(value Object, hasHeader bool) ([]byte, error) {
	arr, ok := value.(*Array)
	if !ok {
		return nil, fmt.Errorf("CSV format requires an array, got %s", strings.ToLower(string(value.Type())))
	}

	if len(arr.Elements) == 0 {
		return []byte{}, nil
	}

	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// Check if first element is a dictionary (has header) or array (no header)
	firstDict, isDict := arr.Elements[0].(*Dictionary)

	if isDict && hasHeader {
		// Write header from dictionary keys
		var headers []string
		for key := range firstDict.Pairs {
			if !strings.HasPrefix(key, "_") {
				headers = append(headers, key)
			}
		}
		sort.Strings(headers) // Consistent ordering
		if err := writer.Write(headers); err != nil {
			return nil, err
		}

		// Write rows
		for _, elem := range arr.Elements {
			dict, ok := elem.(*Dictionary)
			if !ok {
				return nil, fmt.Errorf("CSV with header requires all rows to be dictionaries")
			}
			row := make([]string, len(headers))
			for i, key := range headers {
				if expr, exists := dict.Pairs[key]; exists {
					if ole, ok := expr.(*ast.ObjectLiteralExpression); ok {
						row[i] = ole.Obj.(Object).Inspect()
					} else {
						env := NewEnvironment()
						val := Eval(expr, env)
						row[i] = val.Inspect()
					}
				}
			}
			if err := writer.Write(row); err != nil {
				return nil, err
			}
		}
	} else {
		// Write as array of arrays
		for _, elem := range arr.Elements {
			rowArr, ok := elem.(*Array)
			if !ok {
				// Single-element row
				if err := writer.Write([]string{elem.Inspect()}); err != nil {
					return nil, err
				}
				continue
			}
			row := make([]string, len(rowArr.Elements))
			for i, cell := range rowArr.Elements {
				switch v := cell.(type) {
				case *String:
					row[i] = v.Value
				default:
					row[i] = cell.Inspect()
				}
			}
			if err := writer.Write(row); err != nil {
				return nil, err
			}
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}
