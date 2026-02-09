package pln

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unsafe"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// Serializer converts Parsley objects to PLN strings
type Serializer struct {
	visited map[uintptr]bool       // cycle detection
	indent  string                 // indent string for pretty printing
	depth   int                    // current depth for indentation
	pretty  bool                   // whether to pretty print
	env     *evaluator.Environment // for evaluating lazy expressions
}

// NewSerializer creates a new PLN serializer
func NewSerializer() *Serializer {
	return &Serializer{
		visited: make(map[uintptr]bool),
	}
}

// NewSerializerWithEnv creates a serializer with an environment for lazy evaluation
func NewSerializerWithEnv(env *evaluator.Environment) *Serializer {
	return &Serializer{
		visited: make(map[uintptr]bool),
		env:     env,
	}
}

// NewPrettySerializer creates a serializer that outputs formatted PLN
func NewPrettySerializer(indent string) *Serializer {
	return &Serializer{
		visited: make(map[uintptr]bool),
		indent:  indent,
		pretty:  true,
	}
}

// Serialize converts a Parsley object to a PLN string
func (s *Serializer) Serialize(obj evaluator.Object) (string, error) {
	return s.serialize(obj)
}

func (s *Serializer) serialize(obj evaluator.Object) (string, error) {
	if obj == nil {
		return "null", nil
	}

	switch v := obj.(type) {
	case *evaluator.Integer:
		return strconv.FormatInt(v.Value, 10), nil

	case *evaluator.Float:
		return formatFloat(v.Value), nil

	case *evaluator.String:
		return formatString(v.Value), nil

	case *evaluator.Boolean:
		if v.Value {
			return "true", nil
		}
		return "false", nil

	case *evaluator.Null:
		return "null", nil

	case *evaluator.Array:
		return s.serializeArray(v)

	case *evaluator.Dictionary:
		// Check if it's a special type dict (datetime, path, URL)
		if isDatetimeDict(v) {
			return s.serializeDatetime(v)
		}
		if isPathDict(v) {
			return s.serializePath(v)
		}
		if isURLDict(v) {
			return s.serializeURL(v)
		}
		return s.serializeDict(v)

	case *evaluator.Record:
		return s.serializeRecord(v)

	case *evaluator.Table:
		return s.serializeTable(v)

	case *evaluator.Function:
		return "", fmt.Errorf("cannot serialize function")

	case *evaluator.Builtin:
		return "", fmt.Errorf("cannot serialize builtin function")

	case *evaluator.DBConnection:
		return "", fmt.Errorf("cannot serialize database connection")

	default:
		return "", fmt.Errorf("cannot serialize type %T", obj)
	}
}

func (s *Serializer) serializeArray(arr *evaluator.Array) (string, error) {
	// Check for circular reference
	ptr := getPointer(arr)
	if s.visited[ptr] {
		return "", fmt.Errorf("circular reference detected in array")
	}
	s.visited[ptr] = true
	defer delete(s.visited, ptr)

	if len(arr.Elements) == 0 {
		return "[]", nil
	}

	var parts []string
	s.depth++
	for _, elem := range arr.Elements {
		str, err := s.serialize(elem)
		if err != nil {
			return "", err
		}
		parts = append(parts, str)
	}
	s.depth--

	if s.pretty {
		return s.formatArrayPretty(parts), nil
	}
	return "[" + strings.Join(parts, ", ") + "]", nil
}

func (s *Serializer) formatArrayPretty(parts []string) string {
	if len(parts) == 0 {
		return "[]"
	}

	indent := strings.Repeat(s.indent, s.depth)
	innerIndent := strings.Repeat(s.indent, s.depth+1)

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, part := range parts {
		sb.WriteString(innerIndent)
		sb.WriteString(part)
		if i < len(parts)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(indent)
	sb.WriteString("]")
	return sb.String()
}

func (s *Serializer) serializeDict(d *evaluator.Dictionary) (string, error) {
	// Check for circular reference
	ptr := getPointer(d)
	if s.visited[ptr] {
		return "", fmt.Errorf("circular reference detected in dict")
	}
	s.visited[ptr] = true
	defer delete(s.visited, ptr)

	if len(d.Pairs) == 0 {
		return "{}", nil
	}

	// Get keys in insertion order (preserving dictionary order)
	keys := d.Keys()

	var parts []string
	s.depth++
	for _, k := range keys {
		expr := d.Pairs[k]
		keyStr := formatKey(k)

		// Get the object value from the expression
		obj, err := s.exprToObject(expr, d.Env)
		if err != nil {
			return "", fmt.Errorf("error serializing key %q: %w", k, err)
		}

		valStr, err := s.serialize(obj)
		if err != nil {
			return "", err
		}
		parts = append(parts, keyStr+": "+valStr)
	}
	s.depth--

	if s.pretty {
		return s.formatDictPretty(parts), nil
	}
	return "{" + strings.Join(parts, ", ") + "}", nil
}

// exprToObject converts an ast.Expression to an evaluator.Object
func (s *Serializer) exprToObject(expr ast.Expression, dictEnv *evaluator.Environment) (evaluator.Object, error) {
	if expr == nil {
		return &evaluator.Null{}, nil
	}

	// Check for ObjectLiteralExpression which wraps an already-evaluated Object
	if ole, ok := expr.(*ast.ObjectLiteralExpression); ok {
		if obj, ok := ole.Obj.(evaluator.Object); ok {
			return obj, nil
		}
		return nil, fmt.Errorf("ObjectLiteralExpression contains non-Object type: %T", ole.Obj)
	}

	// For other expressions, we need to evaluate them
	env := s.env
	if env == nil {
		env = dictEnv
	}
	if env == nil {
		// No environment available - create a minimal one for basic expressions
		env = evaluator.NewEnvironment()
	}

	// Evaluate the expression
	result := evaluator.Eval(expr, env)
	if result == nil {
		return &evaluator.Null{}, nil
	}
	if errObj, ok := result.(*evaluator.Error); ok {
		return nil, fmt.Errorf("error evaluating expression: %s", errObj.Message)
	}
	return result, nil
}

func (s *Serializer) formatDictPretty(parts []string) string {
	if len(parts) == 0 {
		return "{}"
	}

	indent := strings.Repeat(s.indent, s.depth)
	innerIndent := strings.Repeat(s.indent, s.depth+1)

	var sb strings.Builder
	sb.WriteString("{\n")
	for i, part := range parts {
		sb.WriteString(innerIndent)
		sb.WriteString(part)
		if i < len(parts)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(indent)
	sb.WriteString("}")
	return sb.String()
}

func (s *Serializer) serializeRecord(r *evaluator.Record) (string, error) {
	// Check for circular reference
	ptr := getPointer(r)
	if s.visited[ptr] {
		return "", fmt.Errorf("circular reference detected in record")
	}
	s.visited[ptr] = true
	defer delete(s.visited, ptr)

	schemaName := "Record"
	if r.Schema != nil {
		schemaName = r.Schema.Name
	}

	// Serialize fields (Data map)
	var parts []string
	keys := make([]string, 0, len(r.Data))
	for k := range r.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	env := s.env
	if env == nil {
		env = r.Env
	}

	s.depth++
	for _, k := range keys {
		expr := r.Data[k]
		keyStr := formatKey(k)

		obj, err := s.exprToObject(expr, env)
		if err != nil {
			return "", fmt.Errorf("error serializing field %q: %w", k, err)
		}

		valStr, err := s.serialize(obj)
		if err != nil {
			return "", err
		}
		parts = append(parts, keyStr+": "+valStr)
	}
	s.depth--

	var sb strings.Builder
	sb.WriteString("@")
	sb.WriteString(schemaName)
	sb.WriteString("(")
	if s.pretty {
		sb.WriteString(s.formatDictPretty(parts))
	} else {
		sb.WriteString("{")
		sb.WriteString(strings.Join(parts, ", "))
		sb.WriteString("}")
	}
	sb.WriteString(")")

	// Serialize errors if present
	if len(r.Errors) > 0 {
		errParts := []string{}
		errKeys := make([]string, 0, len(r.Errors))
		for k := range r.Errors {
			errKeys = append(errKeys, k)
		}
		sort.Strings(errKeys)

		s.depth++
		for _, k := range errKeys {
			err := r.Errors[k]
			keyStr := formatKey(k)
			// RecordError has Message field
			errParts = append(errParts, keyStr+": "+formatString(err.Message))
		}
		s.depth--

		sb.WriteString(" @errors ")
		if s.pretty {
			sb.WriteString(s.formatDictPretty(errParts))
		} else {
			sb.WriteString("{")
			sb.WriteString(strings.Join(errParts, ", "))
			sb.WriteString("}")
		}
	}

	return sb.String(), nil
}

func (s *Serializer) serializeTable(t *evaluator.Table) (string, error) {
	// Check for circular reference
	ptr := getPointer(t)
	if s.visited[ptr] {
		return "", fmt.Errorf("circular reference detected in table")
	}
	s.visited[ptr] = true
	defer delete(s.visited, ptr)

	// Serialize as an array of dictionaries (the Rows)
	if len(t.Rows) == 0 {
		return "[]", nil
	}

	var parts []string
	s.depth++
	for _, row := range t.Rows {
		// Ensure the row has an environment for evaluation
		if row.Env == nil && s.env != nil {
			row.Env = s.env
		}
		rowStr, err := s.serializeDict(row)
		if err != nil {
			return "", err
		}
		parts = append(parts, rowStr)
	}
	s.depth--

	if s.pretty {
		return s.formatArrayPretty(parts), nil
	}
	return "[" + strings.Join(parts, ", ") + "]", nil
}

func (s *Serializer) serializeDatetime(d *evaluator.Dictionary) (string, error) {
	kindObj, err := s.exprToObject(d.Pairs["kind"], d.Env)
	if err != nil || kindObj == nil {
		return "", fmt.Errorf("datetime missing kind field")
	}
	kindStr, ok := kindObj.(*evaluator.String)
	if !ok {
		return "", fmt.Errorf("datetime kind must be string")
	}

	switch kindStr.Value {
	case "date":
		yearObj, _ := s.exprToObject(d.Pairs["year"], d.Env)
		monthObj, _ := s.exprToObject(d.Pairs["month"], d.Env)
		dayObj, _ := s.exprToObject(d.Pairs["day"], d.Env)
		year, _ := yearObj.(*evaluator.Integer)
		month, _ := monthObj.(*evaluator.Integer)
		day, _ := dayObj.(*evaluator.Integer)
		return fmt.Sprintf("@%04d-%02d-%02d", year.Value, month.Value, day.Value), nil

	case "datetime":
		yearObj, _ := s.exprToObject(d.Pairs["year"], d.Env)
		monthObj, _ := s.exprToObject(d.Pairs["month"], d.Env)
		dayObj, _ := s.exprToObject(d.Pairs["day"], d.Env)
		hourObj, _ := s.exprToObject(d.Pairs["hour"], d.Env)
		minuteObj, _ := s.exprToObject(d.Pairs["minute"], d.Env)
		secondObj, _ := s.exprToObject(d.Pairs["second"], d.Env)
		year, _ := yearObj.(*evaluator.Integer)
		month, _ := monthObj.(*evaluator.Integer)
		day, _ := dayObj.(*evaluator.Integer)
		hour, _ := hourObj.(*evaluator.Integer)
		minute, _ := minuteObj.(*evaluator.Integer)
		second, _ := secondObj.(*evaluator.Integer)

		// Check for timezone
		if tzExpr, ok := d.Pairs["timezone"]; ok {
			tzObj, _ := s.exprToObject(tzExpr, d.Env)
			if tzStr, ok := tzObj.(*evaluator.String); ok && tzStr.Value != "" {
				if tzStr.Value == "UTC" {
					return fmt.Sprintf("@%04d-%02d-%02dT%02d:%02d:%02dZ",
						year.Value, month.Value, day.Value, hour.Value, minute.Value, second.Value), nil
				}
				// For other timezones, output with offset (simplified - just use the stored offset)
				return fmt.Sprintf("@%04d-%02d-%02dT%02d:%02d:%02d",
					year.Value, month.Value, day.Value, hour.Value, minute.Value, second.Value), nil
			}
		}
		return fmt.Sprintf("@%04d-%02d-%02dT%02d:%02d:%02d",
			year.Value, month.Value, day.Value, hour.Value, minute.Value, second.Value), nil

	case "time":
		hourObj, _ := s.exprToObject(d.Pairs["hour"], d.Env)
		minuteObj, _ := s.exprToObject(d.Pairs["minute"], d.Env)
		secondObj, _ := s.exprToObject(d.Pairs["second"], d.Env)
		hour, _ := hourObj.(*evaluator.Integer)
		minute, _ := minuteObj.(*evaluator.Integer)
		second, _ := secondObj.(*evaluator.Integer)
		return fmt.Sprintf("@T%02d:%02d:%02d", hour.Value, minute.Value, second.Value), nil

	default:
		return "", fmt.Errorf("unknown datetime kind: %s", kindStr.Value)
	}
}

func (s *Serializer) serializePath(d *evaluator.Dictionary) (string, error) {
	pathObj, err := s.exprToObject(d.Pairs["value"], d.Env)
	if err != nil || pathObj == nil {
		return "", fmt.Errorf("path missing value field")
	}
	pathStr, ok := pathObj.(*evaluator.String)
	if !ok {
		return "", fmt.Errorf("path value must be string")
	}

	return "@" + pathStr.Value, nil
}

func (s *Serializer) serializeURL(d *evaluator.Dictionary) (string, error) {
	urlObj, err := s.exprToObject(d.Pairs["value"], d.Env)
	if err != nil || urlObj == nil {
		return "", fmt.Errorf("URL missing value field")
	}
	urlStr, ok := urlObj.(*evaluator.String)
	if !ok {
		return "", fmt.Errorf("URL value must be string")
	}

	return "@" + urlStr.Value, nil
}

// Helper functions

func formatFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	// Ensure there's a decimal point
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

func formatString(str string) string {
	var sb strings.Builder
	sb.WriteString("\"")
	for _, r := range str {
		switch r {
		case '"':
			sb.WriteString("\\\"")
		case '\\':
			sb.WriteString("\\\\")
		case '\n':
			sb.WriteString("\\n")
		case '\r':
			sb.WriteString("\\r")
		case '\t':
			sb.WriteString("\\t")
		default:
			if r < 32 {
				sb.WriteString(fmt.Sprintf("\\u%04x", r))
			} else {
				sb.WriteRune(r)
			}
		}
	}
	sb.WriteString("\"")
	return sb.String()
}

func formatKey(k string) string {
	// Check if key is a valid identifier
	if isValidIdent(k) {
		return k
	}
	return formatString(k)
}

func isValidIdent(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !isLetter(byte(r)) {
				return false
			}
		} else {
			if !isLetter(byte(r)) && !isDigit(byte(r)) {
				return false
			}
		}
	}
	// Check for reserved words
	switch s {
	case "true", "false", "null":
		return false
	}
	return true
}

// isTypedDict checks if an object is a dictionary with a specific __type value.
func isTypedDict(obj evaluator.Object, typeName string) bool {
	d, ok := obj.(*evaluator.Dictionary)
	if !ok {
		return false
	}
	typeExpr, ok := d.Pairs["__type"]
	if !ok {
		return false
	}
	// Check if it's an ObjectLiteralExpression wrapping a String
	if ole, ok := typeExpr.(*ast.ObjectLiteralExpression); ok {
		if strObj, ok := ole.Obj.(*evaluator.String); ok {
			return strObj.Value == typeName
		}
	}
	return false
}

func isDatetimeDict(obj evaluator.Object) bool {
	return isTypedDict(obj, "datetime")
}

func isPathDict(obj evaluator.Object) bool {
	return isTypedDict(obj, "path")
}

func isURLDict(obj evaluator.Object) bool {
	return isTypedDict(obj, "url")
}

func getPointer(obj evaluator.Object) uintptr {
	switch v := obj.(type) {
	case *evaluator.Array:
		return uintptr(unsafe.Pointer(v))
	case *evaluator.Dictionary:
		return uintptr(unsafe.Pointer(v))
	case *evaluator.Record:
		return uintptr(unsafe.Pointer(v))
	default:
		return 0
	}
}
