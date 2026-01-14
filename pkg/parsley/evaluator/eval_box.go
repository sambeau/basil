package evaluator

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// BoxStyle defines the characters used for box drawing
type BoxStyle struct {
	TopLeft     string
	TopRight    string
	BottomLeft  string
	BottomRight string
	Horizontal  string
	Vertical    string
	LeftT       string
	RightT      string
	TopT        string
	BottomT     string
	Cross       string
}

// Predefined box styles
var (
	BoxStyleSingle = BoxStyle{
		TopLeft:     "┌",
		TopRight:    "┐",
		BottomLeft:  "└",
		BottomRight: "┘",
		Horizontal:  "─",
		Vertical:    "│",
		LeftT:       "├",
		RightT:      "┤",
		TopT:        "┬",
		BottomT:     "┴",
		Cross:       "┼",
	}

	BoxStyleDouble = BoxStyle{
		TopLeft:     "╔",
		TopRight:    "╗",
		BottomLeft:  "╚",
		BottomRight: "╝",
		Horizontal:  "═",
		Vertical:    "║",
		LeftT:       "╠",
		RightT:      "╣",
		TopT:        "╦",
		BottomT:     "╩",
		Cross:       "╬",
	}

	BoxStyleASCII = BoxStyle{
		TopLeft:     "+",
		TopRight:    "+",
		BottomLeft:  "+",
		BottomRight: "+",
		Horizontal:  "-",
		Vertical:    "|",
		LeftT:       "+",
		RightT:      "+",
		TopT:        "+",
		BottomT:     "+",
		Cross:       "+",
	}

	BoxStyleRounded = BoxStyle{
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
		Horizontal:  "─",
		Vertical:    "│",
		LeftT:       "├",
		RightT:      "┤",
		TopT:        "┬",
		BottomT:     "┴",
		Cross:       "┼",
	}
)

// BoxAlign represents text alignment within a cell
type BoxAlign int

const (
	BoxAlignLeft BoxAlign = iota
	BoxAlignRight
	BoxAlignCenter
)

// BoxRenderer handles rendering values as ASCII box tables
type BoxRenderer struct {
	Style    BoxStyle
	Align    BoxAlign
	Title    string
	MaxWidth int
}

// NewBoxRenderer creates a new BoxRenderer with default settings
func NewBoxRenderer() *BoxRenderer {
	return &BoxRenderer{
		Style:    BoxStyleSingle,
		Align:    BoxAlignLeft,
		Title:    "",
		MaxWidth: 0,
	}
}

// displayWidth returns the display width of a string, accounting for unicode
func displayWidth(s string) int {
	return utf8.RuneCountInString(s)
}

// padString pads a string to the given width with the specified alignment
func (br *BoxRenderer) padString(s string, width int) string {
	sWidth := displayWidth(s)
	if sWidth >= width {
		return s
	}

	padding := width - sWidth

	switch br.Align {
	case BoxAlignRight:
		return strings.Repeat(" ", padding) + s
	case BoxAlignCenter:
		leftPad := padding / 2
		rightPad := padding - leftPad
		return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
	default: // BoxAlignLeft
		return s + strings.Repeat(" ", padding)
	}
}

// RenderSingleValue renders a single value in a box
func (br *BoxRenderer) RenderSingleValue(value string) string {
	value = boxEscapeString(value)
	if br.MaxWidth > 0 {
		value = truncateToWidth(value, br.MaxWidth)
	}
	width := displayWidth(value)

	// If title is set, ensure box is wide enough for title
	if br.Title != "" {
		titleWidth := displayWidth(br.Title)
		if titleWidth > width {
			width = titleWidth
		}
	}

	var sb strings.Builder
	s := br.Style

	// Top border
	sb.WriteString(s.TopLeft)
	sb.WriteString(strings.Repeat(s.Horizontal, width+2))
	sb.WriteString(s.TopRight)
	sb.WriteString("\n")

	// Title row (if set)
	if br.Title != "" {
		sb.WriteString(s.Vertical)
		sb.WriteString(" ")
		sb.WriteString(br.centerString(br.Title, width))
		sb.WriteString(" ")
		sb.WriteString(s.Vertical)
		sb.WriteString("\n")

		// Title separator
		sb.WriteString(s.LeftT)
		sb.WriteString(strings.Repeat(s.Horizontal, width+2))
		sb.WriteString(s.RightT)
		sb.WriteString("\n")
	}

	// Value row
	sb.WriteString(s.Vertical)
	sb.WriteString(" ")
	sb.WriteString(br.padString(value, width))
	sb.WriteString(" ")
	sb.WriteString(s.Vertical)
	sb.WriteString("\n")

	// Bottom border
	sb.WriteString(s.BottomLeft)
	sb.WriteString(strings.Repeat(s.Horizontal, width+2))
	sb.WriteString(s.BottomRight)

	return sb.String()
}

// centerString centers a string within a given width
func (br *BoxRenderer) centerString(s string, width int) string {
	sWidth := displayWidth(s)
	if sWidth >= width {
		return s
	}
	padding := width - sWidth
	leftPad := padding / 2
	rightPad := padding - leftPad
	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}

// RenderVerticalList renders an array of values vertically
func (br *BoxRenderer) RenderVerticalList(values []string) string {
	if len(values) == 0 {
		return br.RenderSingleValue("(empty)")
	}

	// Escape all values, apply maxWidth, and calculate max width
	escaped := make([]string, len(values))
	maxWidth := 0
	for i, v := range values {
		escaped[i] = boxEscapeString(v)
		if br.MaxWidth > 0 {
			escaped[i] = truncateToWidth(escaped[i], br.MaxWidth)
		}
		w := displayWidth(escaped[i])
		if w > maxWidth {
			maxWidth = w
		}
	}

	// If title is set, ensure box is wide enough
	if br.Title != "" {
		titleWidth := displayWidth(br.Title)
		if titleWidth > maxWidth {
			maxWidth = titleWidth
		}
	}

	var sb strings.Builder
	s := br.Style

	// Top border
	sb.WriteString(s.TopLeft)
	sb.WriteString(strings.Repeat(s.Horizontal, maxWidth+2))
	sb.WriteString(s.TopRight)
	sb.WriteString("\n")

	// Title row (if set)
	if br.Title != "" {
		sb.WriteString(s.Vertical)
		sb.WriteString(" ")
		sb.WriteString(br.centerString(br.Title, maxWidth))
		sb.WriteString(" ")
		sb.WriteString(s.Vertical)
		sb.WriteString("\n")

		// Title separator
		sb.WriteString(s.LeftT)
		sb.WriteString(strings.Repeat(s.Horizontal, maxWidth+2))
		sb.WriteString(s.RightT)
		sb.WriteString("\n")
	}

	// Value rows
	for i, val := range escaped {
		sb.WriteString(s.Vertical)
		sb.WriteString(" ")
		sb.WriteString(br.padString(val, maxWidth))
		sb.WriteString(" ")
		sb.WriteString(s.Vertical)
		sb.WriteString("\n")

		// Separator between rows (except after last)
		if i < len(escaped)-1 {
			sb.WriteString(s.LeftT)
			sb.WriteString(strings.Repeat(s.Horizontal, maxWidth+2))
			sb.WriteString(s.RightT)
			sb.WriteString("\n")
		}
	}

	// Bottom border
	sb.WriteString(s.BottomLeft)
	sb.WriteString(strings.Repeat(s.Horizontal, maxWidth+2))
	sb.WriteString(s.BottomRight)

	return sb.String()
}

// RenderHorizontalList renders an array of values horizontally
func (br *BoxRenderer) RenderHorizontalList(values []string) string {
	if len(values) == 0 {
		return br.RenderSingleValue("(empty)")
	}

	// Escape all values, apply maxWidth, and calculate widths
	escaped := make([]string, len(values))
	widths := make([]int, len(values))
	totalWidth := 0
	for i, v := range values {
		escaped[i] = boxEscapeString(v)
		if br.MaxWidth > 0 {
			escaped[i] = truncateToWidth(escaped[i], br.MaxWidth)
		}
		widths[i] = displayWidth(escaped[i])
		totalWidth += widths[i] + 2 // +2 for padding
	}
	totalWidth += len(values) - 1 // Add separators

	var sb strings.Builder
	s := br.Style

	// Calculate title row width if needed
	titleRowWidth := 0
	if br.Title != "" {
		titleWidth := displayWidth(br.Title)
		if titleWidth > totalWidth {
			titleRowWidth = titleWidth
		}
	}

	// Top border
	sb.WriteString(s.TopLeft)
	if titleRowWidth > 0 {
		sb.WriteString(strings.Repeat(s.Horizontal, titleRowWidth+2))
	} else {
		for i, w := range widths {
			sb.WriteString(strings.Repeat(s.Horizontal, w+2))
			if i < len(widths)-1 {
				sb.WriteString(s.TopT)
			}
		}
	}
	sb.WriteString(s.TopRight)
	sb.WriteString("\n")

	// Title row (if set)
	if br.Title != "" {
		rowWidth := titleRowWidth
		if rowWidth == 0 {
			rowWidth = totalWidth
		}
		sb.WriteString(s.Vertical)
		sb.WriteString(" ")
		sb.WriteString(br.centerString(br.Title, rowWidth))
		sb.WriteString(" ")
		sb.WriteString(s.Vertical)
		sb.WriteString("\n")

		// Title separator with column dividers
		sb.WriteString(s.LeftT)
		for i, w := range widths {
			sb.WriteString(strings.Repeat(s.Horizontal, w+2))
			if i < len(widths)-1 {
				sb.WriteString(s.TopT)
			}
		}
		sb.WriteString(s.RightT)
		sb.WriteString("\n")
	}

	// Value row
	sb.WriteString(s.Vertical)
	for i, val := range escaped {
		sb.WriteString(" ")
		sb.WriteString(br.padString(val, widths[i]))
		sb.WriteString(" ")
		sb.WriteString(s.Vertical)
	}
	sb.WriteString("\n")

	// Bottom border
	sb.WriteString(s.BottomLeft)
	for i, w := range widths {
		sb.WriteString(strings.Repeat(s.Horizontal, w+2))
		if i < len(widths)-1 {
			sb.WriteString(s.BottomT)
		}
	}
	sb.WriteString(s.BottomRight)

	return sb.String()
}

// RenderGrid renders a 2D array as a grid
func (br *BoxRenderer) RenderGrid(rows [][]string) string {
	if len(rows) == 0 {
		return br.RenderSingleValue("(empty)")
	}

	// Find max columns
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	if maxCols == 0 {
		return br.RenderSingleValue("(empty)")
	}

	// Escape all values, apply maxWidth, and calculate column widths
	escaped := make([][]string, len(rows))
	colWidths := make([]int, maxCols)

	for i, row := range rows {
		escaped[i] = make([]string, maxCols)
		for j := 0; j < maxCols; j++ {
			if j < len(row) {
				escaped[i][j] = boxEscapeString(row[j])
				if br.MaxWidth > 0 {
					escaped[i][j] = truncateToWidth(escaped[i][j], br.MaxWidth)
				}
			} else {
				escaped[i][j] = "" // Pad jagged arrays
			}
			w := displayWidth(escaped[i][j])
			if w > colWidths[j] {
				colWidths[j] = w
			}
		}
	}

	// Calculate total row width
	totalWidth := 0
	for _, w := range colWidths {
		totalWidth += w + 2 // +2 for padding
	}
	totalWidth += len(colWidths) - 1 // Add separators

	var sb strings.Builder
	s := br.Style

	// Calculate title row width if needed
	titleRowWidth := 0
	if br.Title != "" {
		titleWidth := displayWidth(br.Title)
		if titleWidth > totalWidth {
			titleRowWidth = titleWidth
		}
	}

	// Helper to write horizontal line
	writeHLine := func(left, mid, right string) {
		sb.WriteString(left)
		for i, w := range colWidths {
			sb.WriteString(strings.Repeat(s.Horizontal, w+2))
			if i < len(colWidths)-1 {
				sb.WriteString(mid)
			}
		}
		sb.WriteString(right)
		sb.WriteString("\n")
	}

	// Top border
	if titleRowWidth > 0 {
		sb.WriteString(s.TopLeft)
		sb.WriteString(strings.Repeat(s.Horizontal, titleRowWidth+2))
		sb.WriteString(s.TopRight)
		sb.WriteString("\n")
	} else {
		writeHLine(s.TopLeft, s.TopT, s.TopRight)
	}

	// Title row (if set)
	if br.Title != "" {
		rowWidth := titleRowWidth
		if rowWidth == 0 {
			rowWidth = totalWidth
		}
		sb.WriteString(s.Vertical)
		sb.WriteString(" ")
		sb.WriteString(br.centerString(br.Title, rowWidth))
		sb.WriteString(" ")
		sb.WriteString(s.Vertical)
		sb.WriteString("\n")

		// Title separator with column dividers
		writeHLine(s.LeftT, s.TopT, s.RightT)
	}

	// Data rows
	for i, row := range escaped {
		sb.WriteString(s.Vertical)
		for j, val := range row {
			sb.WriteString(" ")
			sb.WriteString(br.padString(val, colWidths[j]))
			sb.WriteString(" ")
			sb.WriteString(s.Vertical)
		}
		sb.WriteString("\n")

		// Separator between rows (except after last)
		if i < len(escaped)-1 {
			writeHLine(s.LeftT, s.Cross, s.RightT)
		}
	}

	// Bottom border
	writeHLine(s.BottomLeft, s.BottomT, s.BottomRight)

	return sb.String()
}

// RenderTable renders a table with headers and rows
func (br *BoxRenderer) RenderTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return br.RenderSingleValue("(empty)")
	}

	s := br.Style
	numCols := len(headers)

	// Escape all values and calculate column widths
	escapedHeaders := make([]string, numCols)
	for i, h := range headers {
		escapedHeaders[i] = boxEscapeString(h)
	}

	escapedRows := make([][]string, len(rows))
	for i, row := range rows {
		escapedRows[i] = make([]string, numCols)
		for j := 0; j < numCols; j++ {
			if j < len(row) {
				escapedRows[i][j] = boxEscapeString(row[j])
				if br.MaxWidth > 0 {
					escapedRows[i][j] = truncateToWidth(escapedRows[i][j], br.MaxWidth)
				}
			} else {
				escapedRows[i][j] = ""
			}
		}
	}

	// Calculate column widths from headers and all rows
	colWidths := make([]int, numCols)
	for i, h := range escapedHeaders {
		colWidths[i] = displayWidth(h)
	}
	for _, row := range escapedRows {
		for i, val := range row {
			w := displayWidth(val)
			if w > colWidths[i] {
				colWidths[i] = w
			}
		}
	}

	// Calculate total row width
	totalWidth := 0
	for _, w := range colWidths {
		totalWidth += w + 2 // +2 for padding
	}
	totalWidth += numCols - 1 // Add separators

	var sb strings.Builder

	// Calculate title row width if needed
	titleRowWidth := 0
	if br.Title != "" {
		titleWidth := displayWidth(br.Title)
		if titleWidth > totalWidth {
			titleRowWidth = titleWidth
		}
	}

	// Helper to write a horizontal line
	writeHLine := func(left, mid, right string) {
		sb.WriteString(left)
		for i, w := range colWidths {
			sb.WriteString(strings.Repeat(s.Horizontal, w+2))
			if i < numCols-1 {
				sb.WriteString(mid)
			}
		}
		sb.WriteString(right)
		sb.WriteString("\n")
	}

	// Helper to write a row
	writeRow := func(values []string) {
		sb.WriteString(s.Vertical)
		for i, val := range values {
			sb.WriteString(" ")
			sb.WriteString(br.padString(val, colWidths[i]))
			sb.WriteString(" ")
			sb.WriteString(s.Vertical)
		}
		sb.WriteString("\n")
	}

	// Top border
	if titleRowWidth > 0 {
		sb.WriteString(s.TopLeft)
		sb.WriteString(strings.Repeat(s.Horizontal, titleRowWidth+2))
		sb.WriteString(s.TopRight)
		sb.WriteString("\n")
	} else {
		writeHLine(s.TopLeft, s.TopT, s.TopRight)
	}

	// Title row (if set)
	if br.Title != "" {
		rowWidth := titleRowWidth
		if rowWidth == 0 {
			rowWidth = totalWidth
		}
		sb.WriteString(s.Vertical)
		sb.WriteString(" ")
		sb.WriteString(br.centerString(br.Title, rowWidth))
		sb.WriteString(" ")
		sb.WriteString(s.Vertical)
		sb.WriteString("\n")

		// Title separator with column dividers
		writeHLine(s.LeftT, s.TopT, s.RightT)
	}

	// Header row
	writeRow(escapedHeaders)

	// Header separator
	writeHLine(s.LeftT, s.Cross, s.RightT)

	// Data rows
	for _, row := range escapedRows {
		writeRow(row)
	}

	// Bottom border
	writeHLine(s.BottomLeft, s.BottomT, s.BottomRight)

	return sb.String()
}

// RenderKeyValue renders a dictionary as key-value pairs
func (br *BoxRenderer) RenderKeyValue(keys []string, values []string) string {
	if len(keys) == 0 {
		return br.RenderSingleValue("(empty)")
	}

	// Calculate column widths
	keyWidth := 0
	valWidth := 0

	escapedKeys := make([]string, len(keys))
	escapedVals := make([]string, len(values))

	for i := range keys {
		escapedKeys[i] = boxEscapeString(keys[i])
		escapedVals[i] = boxEscapeString(values[i])

		// Apply maxWidth to values (not keys)
		if br.MaxWidth > 0 {
			escapedVals[i] = truncateToWidth(escapedVals[i], br.MaxWidth)
		}

		kw := displayWidth(escapedKeys[i])
		vw := displayWidth(escapedVals[i])

		if kw > keyWidth {
			keyWidth = kw
		}
		if vw > valWidth {
			valWidth = vw
		}
	}

	// Calculate total row width
	totalWidth := keyWidth + valWidth + 5 // +5 for padding and separator

	var sb strings.Builder
	s := br.Style

	// Calculate title row width if needed
	titleRowWidth := 0
	if br.Title != "" {
		titleWidth := displayWidth(br.Title)
		if titleWidth > totalWidth {
			titleRowWidth = titleWidth
		}
	}

	// Helper to write horizontal line
	writeHLine := func(left, mid, right string) {
		sb.WriteString(left)
		sb.WriteString(strings.Repeat(s.Horizontal, keyWidth+2))
		sb.WriteString(mid)
		sb.WriteString(strings.Repeat(s.Horizontal, valWidth+2))
		sb.WriteString(right)
		sb.WriteString("\n")
	}

	// Top border
	if titleRowWidth > 0 {
		sb.WriteString(s.TopLeft)
		sb.WriteString(strings.Repeat(s.Horizontal, titleRowWidth+2))
		sb.WriteString(s.TopRight)
		sb.WriteString("\n")
	} else {
		writeHLine(s.TopLeft, s.TopT, s.TopRight)
	}

	// Title row (if set)
	if br.Title != "" {
		rowWidth := titleRowWidth
		if rowWidth == 0 {
			rowWidth = totalWidth
		}
		sb.WriteString(s.Vertical)
		sb.WriteString(" ")
		sb.WriteString(br.centerString(br.Title, rowWidth))
		sb.WriteString(" ")
		sb.WriteString(s.Vertical)
		sb.WriteString("\n")

		// Title separator with column divider
		writeHLine(s.LeftT, s.TopT, s.RightT)
	}

	// Key-value rows
	for i := range escapedKeys {
		sb.WriteString(s.Vertical)
		sb.WriteString(" ")
		sb.WriteString(br.padString(escapedKeys[i], keyWidth))
		sb.WriteString(" ")
		sb.WriteString(s.Vertical)
		sb.WriteString(" ")
		sb.WriteString(br.padString(escapedVals[i], valWidth))
		sb.WriteString(" ")
		sb.WriteString(s.Vertical)
		sb.WriteString("\n")

		// Separator between rows (except after last)
		if i < len(escapedKeys)-1 {
			writeHLine(s.LeftT, s.Cross, s.RightT)
		}
	}

	// Bottom border
	writeHLine(s.BottomLeft, s.BottomT, s.BottomRight)

	return sb.String()
}

// RenderKeysOnly renders just the keys in a horizontal row
func (br *BoxRenderer) RenderKeysOnly(keys []string) string {
	return br.RenderHorizontalList(keys)
}

// boxEscapeString escapes characters that would break box formatting
func boxEscapeString(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	return s
}

// truncateToWidth truncates a string to maxWidth characters, adding ellipsis if needed
// Returns the original string if maxWidth <= 3 (can't fit meaningful content + ellipsis)
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 3 {
		return s // Too small to truncate meaningfully
	}
	if displayWidth(s) <= maxWidth {
		return s
	}
	// Truncate by runes to handle unicode properly
	runes := []rune(s)
	if len(runes) <= maxWidth-3 {
		return s
	}
	return string(runes[:maxWidth-3]) + "..."
}

// BoxOptions holds parsed toBox options
type BoxOptions struct {
	Direction string
	Align     BoxAlign
	KeysOnly  bool
	Style     BoxStyle
	Title     string
	MaxWidth  int
}

// parseBoxOptions parses toBox options from arguments
func parseBoxOptions(args []Object) (opts BoxOptions, err Object) {
	// Set defaults
	opts = BoxOptions{
		Direction: "vertical",
		Align:     BoxAlignLeft,
		KeysOnly:  false,
		Style:     BoxStyleSingle,
		Title:     "",
		MaxWidth:  0,
	}

	if len(args) == 0 {
		return
	}

	if len(args) > 1 {
		err = newArityErrorRange("toBox", len(args), 0, 1)
		return
	}

	optsDict, ok := args[0].(*Dictionary)
	if !ok {
		err = newTypeError("TYPE-0001", "toBox", "dictionary", args[0].Type())
		return
	}

	// Parse direction
	if dirVal := getDictValue(optsDict, "direction"); dirVal != nil && dirVal != NULL {
		if dirStr, ok := dirVal.(*String); ok {
			switch dirStr.Value {
			case "vertical", "horizontal", "grid":
				opts.Direction = dirStr.Value
			default:
				err = newValueError("VALUE-0001", map[string]any{
					"Message": "direction must be 'vertical', 'horizontal', or 'grid'",
				})
				return
			}
		}
	}

	// Parse align
	if alignVal := getDictValue(optsDict, "align"); alignVal != nil && alignVal != NULL {
		if alignStr, ok := alignVal.(*String); ok {
			switch alignStr.Value {
			case "left":
				opts.Align = BoxAlignLeft
			case "right":
				opts.Align = BoxAlignRight
			case "center":
				opts.Align = BoxAlignCenter
			default:
				err = newValueError("VALUE-0001", map[string]any{
					"Message": "align must be 'left', 'right', or 'center'",
				})
				return
			}
		}
	}

	// Parse keys (for dictionaries)
	if keysVal := getDictValue(optsDict, "keys"); keysVal != nil && keysVal != NULL {
		if keysBool, ok := keysVal.(*Boolean); ok {
			opts.KeysOnly = keysBool.Value
		}
	}

	// Parse style
	if styleVal := getDictValue(optsDict, "style"); styleVal != nil && styleVal != NULL {
		if styleStr, ok := styleVal.(*String); ok {
			switch styleStr.Value {
			case "single":
				opts.Style = BoxStyleSingle
			case "double":
				opts.Style = BoxStyleDouble
			case "ascii":
				opts.Style = BoxStyleASCII
			case "rounded":
				opts.Style = BoxStyleRounded
			default:
				err = &Error{Message: "toBox: invalid style '" + styleStr.Value + "', must be 'single', 'double', 'ascii', or 'rounded'"}
				return
			}
		} else {
			err = &Error{Message: "toBox: style option must be a string, got " + string(styleVal.Type())}
			return
		}
	}

	// Parse title
	if titleVal := getDictValue(optsDict, "title"); titleVal != nil && titleVal != NULL {
		if titleStr, ok := titleVal.(*String); ok {
			opts.Title = titleStr.Value
		} else {
			err = &Error{Message: "toBox: title option must be a string, got " + string(titleVal.Type())}
			return
		}
	}

	// Parse maxWidth
	if maxWidthVal := getDictValue(optsDict, "maxWidth"); maxWidthVal != nil && maxWidthVal != NULL {
		if maxWidthInt, ok := maxWidthVal.(*Integer); ok {
			if maxWidthInt.Value < 0 {
				err = &Error{Message: "toBox: maxWidth must be a non-negative integer"}
				return
			}
			opts.MaxWidth = int(maxWidthInt.Value)
		} else {
			err = &Error{Message: "toBox: maxWidth option must be an integer, got " + string(maxWidthVal.Type())}
			return
		}
	}

	return
}

// objectToBoxString converts an object to its string representation for boxing
// For complex types, this produces an inline summary rather than recursive boxes
func objectToBoxString(obj Object) string {
	if obj == nil || obj == NULL {
		return "null"
	}

	switch v := obj.(type) {
	case *String:
		return v.Value
	case *Integer:
		return v.Inspect()
	case *Float:
		return v.Inspect()
	case *Boolean:
		return v.Inspect()
	case *Array:
		// Inline array representation
		if len(v.Elements) == 0 {
			return "[]"
		}
		parts := make([]string, len(v.Elements))
		for i, el := range v.Elements {
			parts[i] = objectToBoxString(el)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *Dictionary:
		// Inline dict representation - use Keys() to get evaluated keys
		keys := v.Keys()
		if len(keys) == 0 {
			return "{}"
		}
		parts := make([]string, len(keys))
		for i, key := range keys {
			val := getDictValue(v, key)
			parts[i] = key + ": " + objectToBoxString(val)
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case *Table:
		return "table(" + strconv.Itoa(len(v.Rows)) + " rows)"
	default:
		return obj.Inspect()
	}
}

// isArrayOfArrays checks if all elements of an array are arrays
func isArrayOfArrays(arr *Array) bool {
	if len(arr.Elements) == 0 {
		return false
	}
	for _, el := range arr.Elements {
		if _, ok := el.(*Array); !ok {
			return false
		}
	}
	return true
}

// arrayToBox renders an array as a box
func arrayToBox(arr *Array, args []Object, env *Environment) Object {
	opts, err := parseBoxOptions(args)
	if err != nil {
		return err
	}

	br := NewBoxRenderer()
	br.Align = opts.Align
	br.Style = opts.Style
	br.Title = opts.Title
	br.MaxWidth = opts.MaxWidth

	// Grid mode: explicit direction or auto-detect array of arrays
	if opts.Direction == "grid" || isArrayOfArrays(arr) {
		rows := make([][]string, len(arr.Elements))
		for i, el := range arr.Elements {
			if innerArr, ok := el.(*Array); ok {
				row := make([]string, len(innerArr.Elements))
				for j, item := range innerArr.Elements {
					row[j] = objectToBoxString(item)
				}
				rows[i] = row
			} else {
				// Single element row if not an array
				rows[i] = []string{objectToBoxString(el)}
			}
		}
		return &String{Value: br.RenderGrid(rows)}
	}

	// Convert elements to strings
	values := make([]string, len(arr.Elements))
	for i, el := range arr.Elements {
		values[i] = objectToBoxString(el)
	}

	// Render based on direction
	if opts.Direction == "horizontal" {
		return &String{Value: br.RenderHorizontalList(values)}
	}
	return &String{Value: br.RenderVerticalList(values)}
}

// dictToBox renders a dictionary as a box
func dictToBox(dict *Dictionary, args []Object, env *Environment) Object {
	opts, err := parseBoxOptions(args)
	if err != nil {
		return err
	}

	br := NewBoxRenderer()
	br.Align = opts.Align
	br.Style = opts.Style
	br.Title = opts.Title
	br.MaxWidth = opts.MaxWidth

	// Get keys in order
	keys := dict.Keys()

	if opts.KeysOnly {
		return &String{Value: br.RenderKeysOnly(keys)}
	}

	// Get values
	values := make([]string, len(keys))
	for i, key := range keys {
		val := getDictValue(dict, key)
		values[i] = objectToBoxString(val)
	}

	return &String{Value: br.RenderKeyValue(keys, values)}
}

// datetimeToBox renders a datetime value as a box
func datetimeToBox(dict *Dictionary, args []Object, env *Environment) Object {
	opts, err := parseBoxOptions(args)
	if err != nil {
		return err
	}

	br := NewBoxRenderer()
	br.Align = opts.Align
	br.Style = opts.Style
	br.MaxWidth = opts.MaxWidth

	// Set title to "datetime" if not specified
	if opts.Title != "" {
		br.Title = opts.Title
	} else {
		br.Title = "datetime"
	}

	// Get datetime components
	keys := []string{"year", "month", "day", "hour", "minute", "second", "weekday", "timezone"}
	values := make([]string, 0, len(keys))
	actualKeys := make([]string, 0, len(keys))

	for _, key := range keys {
		if expr, ok := dict.Pairs[key]; ok {
			val := Eval(expr, env)
			if val != nil && val != NULL {
				actualKeys = append(actualKeys, key)
				values = append(values, objectToBoxString(val))
			}
		}
	}

	return &String{Value: br.RenderKeyValue(actualKeys, values)}
}

// durationToBox renders a duration value as a box
func durationToBox(dict *Dictionary, args []Object, env *Environment) Object {
	opts, err := parseBoxOptions(args)
	if err != nil {
		return err
	}

	br := NewBoxRenderer()
	br.Align = opts.Align
	br.Style = opts.Style
	br.MaxWidth = opts.MaxWidth

	if opts.Title != "" {
		br.Title = opts.Title
	} else {
		br.Title = "duration"
	}

	// Get duration components - show only non-zero values
	months, seconds, _ := getDurationComponents(dict, env)

	keys := []string{}
	values := []string{}

	years := months / 12
	months = months % 12
	days := seconds / (24 * 3600)
	seconds = seconds % (24 * 3600)
	hours := seconds / 3600
	seconds = seconds % 3600
	minutes := seconds / 60
	seconds = seconds % 60

	if years > 0 {
		keys = append(keys, "years")
		values = append(values, strconv.FormatInt(years, 10))
	}
	if months > 0 {
		keys = append(keys, "months")
		values = append(values, strconv.FormatInt(months, 10))
	}
	if days > 0 {
		keys = append(keys, "days")
		values = append(values, strconv.FormatInt(days, 10))
	}
	if hours > 0 {
		keys = append(keys, "hours")
		values = append(values, strconv.FormatInt(hours, 10))
	}
	if minutes > 0 {
		keys = append(keys, "minutes")
		values = append(values, strconv.FormatInt(minutes, 10))
	}
	if seconds > 0 || len(keys) == 0 {
		keys = append(keys, "seconds")
		values = append(values, strconv.FormatInt(seconds, 10))
	}

	return &String{Value: br.RenderKeyValue(keys, values)}
}

// pathToBox renders a path value as a box
func pathToBox(dict *Dictionary, args []Object, env *Environment) Object {
	opts, err := parseBoxOptions(args)
	if err != nil {
		return err
	}

	br := NewBoxRenderer()
	br.Align = opts.Align
	br.Style = opts.Style
	br.MaxWidth = opts.MaxWidth

	if opts.Title != "" {
		br.Title = opts.Title
	} else {
		br.Title = "path"
	}

	// Get path components
	pathStr := pathDictToString(dict)
	keys := []string{"path"}
	values := []string{pathStr}

	// Add computed properties
	if absExpr, ok := dict.Pairs["absolute"]; ok {
		val := Eval(absExpr, env)
		if b, ok := val.(*Boolean); ok {
			keys = append(keys, "absolute")
			values = append(values, b.Inspect())
		}
	}
	if extExpr, ok := dict.Pairs["extension"]; ok {
		val := Eval(extExpr, env)
		if s, ok := val.(*String); ok && s.Value != "" {
			keys = append(keys, "extension")
			values = append(values, s.Value)
		}
	}

	return &String{Value: br.RenderKeyValue(keys, values)}
}

// urlToBox renders a URL value as a box
func urlToBox(dict *Dictionary, args []Object, env *Environment) Object {
	opts, err := parseBoxOptions(args)
	if err != nil {
		return err
	}

	br := NewBoxRenderer()
	br.Align = opts.Align
	br.Style = opts.Style
	br.MaxWidth = opts.MaxWidth

	if opts.Title != "" {
		br.Title = opts.Title
	} else {
		br.Title = "url"
	}

	// Get URL components
	urlStr := urlDictToString(dict)
	keys := []string{"url"}
	values := []string{urlStr}

	// Add individual components
	components := []string{"scheme", "host", "port", "path", "fragment"}
	for _, comp := range components {
		if expr, ok := dict.Pairs[comp]; ok {
			val := Eval(expr, env)
			if val != nil && val != NULL {
				switch v := val.(type) {
				case *String:
					if v.Value != "" {
						keys = append(keys, comp)
						values = append(values, v.Value)
					}
				case *Integer:
					if v.Value > 0 {
						keys = append(keys, comp)
						values = append(values, strconv.FormatInt(v.Value, 10))
					}
				case *Array:
					if len(v.Elements) > 0 {
						keys = append(keys, comp)
						values = append(values, objectToBoxString(v))
					}
				}
			}
		}
	}

	return &String{Value: br.RenderKeyValue(keys, values)}
}

// regexToBox renders a regex value as a box
func regexToBox(dict *Dictionary, args []Object, env *Environment) Object {
	opts, err := parseBoxOptions(args)
	if err != nil {
		return err
	}

	br := NewBoxRenderer()
	br.Align = opts.Align
	br.Style = opts.Style
	br.MaxWidth = opts.MaxWidth

	if opts.Title != "" {
		br.Title = opts.Title
	} else {
		br.Title = "regex"
	}

	keys := []string{}
	values := []string{}

	if patternExpr, ok := dict.Pairs["pattern"]; ok {
		val := Eval(patternExpr, env)
		if s, ok := val.(*String); ok {
			keys = append(keys, "pattern")
			values = append(values, s.Value)
		}
	}
	if flagsExpr, ok := dict.Pairs["flags"]; ok {
		val := Eval(flagsExpr, env)
		if s, ok := val.(*String); ok && s.Value != "" {
			keys = append(keys, "flags")
			values = append(values, s.Value)
		}
	}

	return &String{Value: br.RenderKeyValue(keys, values)}
}

// moneyToBox renders a money value as a box
func moneyToBox(money *Money, args []Object) Object {
	opts, err := parseBoxOptions(args)
	if err != nil {
		return err
	}

	br := NewBoxRenderer()
	br.Align = opts.Align
	br.Style = opts.Style
	br.MaxWidth = opts.MaxWidth

	if opts.Title != "" {
		br.Title = opts.Title
	} else {
		br.Title = "money"
	}

	keys := []string{"amount", "currency"}
	values := []string{money.formatAmount(), money.Currency}

	return &String{Value: br.RenderKeyValue(keys, values)}
}
