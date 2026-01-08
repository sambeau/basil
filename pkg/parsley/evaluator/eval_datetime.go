// eval_datetime.go - Datetime and duration conversion helpers for the Parsley evaluator
//
// This file contains functions for converting between Go time.Time/Duration values
// and Parsley dictionary representations. Includes formatting and string conversion.

package evaluator

import (
	"fmt"
	"strings"
	"time"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// timeToDictWithKind converts a Go time.Time to a Parsley datetime dictionary with a specific kind.
// kind can be "datetime", "date", or "time"
func timeToDictWithKind(t time.Time, kind string, env *Environment) *Dictionary {
	pairs := make(map[string]ast.Expression)

	// Mark this as a datetime dictionary for special operator handling
	pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "datetime"},
		Value: "datetime",
	}

	// Store the kind (datetime, date, or time)
	pairs["kind"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: kind},
		Value: kind,
	}

	// Create integer literals for numeric values with proper tokens
	pairs["year"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", t.Year())},
		Value: int64(t.Year()),
	}
	pairs["month"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", t.Month())},
		Value: int64(t.Month()),
	}
	pairs["day"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", t.Day())},
		Value: int64(t.Day()),
	}
	pairs["hour"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", t.Hour())},
		Value: int64(t.Hour()),
	}
	pairs["minute"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", t.Minute())},
		Value: int64(t.Minute()),
	}
	pairs["second"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", t.Second())},
		Value: int64(t.Second()),
	}
	pairs["unix"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", t.Unix())},
		Value: t.Unix(),
	}

	// Create string literals for string values with proper tokens
	weekday := t.Weekday().String()
	pairs["weekday"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: weekday},
		Value: weekday,
	}
	iso := t.Format(time.RFC3339)
	pairs["iso"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: iso},
		Value: iso,
	}

	return &Dictionary{Pairs: pairs, Env: env}
}

// timeToDict converts a Go time.Time to a Parsley Dictionary (defaults to kind: "datetime")
func timeToDict(t time.Time, env *Environment) *Dictionary {
	return timeToDictWithKind(t, "datetime", env)
}

// dictToTime converts a Parsley Dictionary to a Go time.Time
func dictToTime(dict *Dictionary, env *Environment) (time.Time, error) {
	// Evaluate the year field
	yearExpr, ok := dict.Pairs["year"]
	if !ok {
		return time.Time{}, fmt.Errorf("missing 'year' field")
	}
	yearObj := Eval(yearExpr, env)
	yearInt, ok := yearObj.(*Integer)
	if !ok {
		return time.Time{}, fmt.Errorf("'year' must be an integer")
	}

	// Evaluate the month field
	monthExpr, ok := dict.Pairs["month"]
	if !ok {
		return time.Time{}, fmt.Errorf("missing 'month' field")
	}
	monthObj := Eval(monthExpr, env)
	monthInt, ok := monthObj.(*Integer)
	if !ok {
		return time.Time{}, fmt.Errorf("'month' must be an integer")
	}

	// Evaluate the day field
	dayExpr, ok := dict.Pairs["day"]
	if !ok {
		return time.Time{}, fmt.Errorf("missing 'day' field")
	}
	dayObj := Eval(dayExpr, env)
	dayInt, ok := dayObj.(*Integer)
	if !ok {
		return time.Time{}, fmt.Errorf("'day' must be an integer")
	}

	// Hour, minute, second are optional (default to 0)
	var hour, minute, second int64

	if hExpr, ok := dict.Pairs["hour"]; ok {
		hObj := Eval(hExpr, env)
		if hInt, ok := hObj.(*Integer); ok {
			hour = hInt.Value
		}
	}

	if mExpr, ok := dict.Pairs["minute"]; ok {
		mObj := Eval(mExpr, env)
		if mInt, ok := mObj.(*Integer); ok {
			minute = mInt.Value
		}
	}

	if sExpr, ok := dict.Pairs["second"]; ok {
		sObj := Eval(sExpr, env)
		if sInt, ok := sObj.(*Integer); ok {
			second = sInt.Value
		}
	}

	return time.Date(
		int(yearInt.Value),
		time.Month(monthInt.Value),
		int(dayInt.Value),
		int(hour),
		int(minute),
		int(second),
		0,
		time.UTC,
	), nil
}

// durationToDict converts months and seconds into a Parsley duration dictionary
func durationToDict(months, seconds int64, env *Environment) *Dictionary {
	dict := &Dictionary{Pairs: make(map[string]ast.Expression)}

	// Add __type field
	dict.Pairs["__type"] = &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: "duration"},
		Value: "duration",
	}

	// Add months field
	dict.Pairs["months"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", months)},
		Value: months,
	}

	// Add seconds field
	dict.Pairs["seconds"] = &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", seconds)},
		Value: seconds,
	}

	// Add totalSeconds field (only present if no months)
	if months == 0 {
		dict.Pairs["totalSeconds"] = &ast.IntegerLiteral{
			Token: lexer.Token{Type: lexer.INT, Literal: fmt.Sprintf("%d", seconds)},
			Value: seconds,
		}
	}

	return dict
}

// getDurationComponents extracts months and seconds from a duration dictionary
func getDurationComponents(dict *Dictionary, env *Environment) (int64, int64, error) {
	monthsExpr, ok := dict.Pairs["months"]
	if !ok {
		return 0, 0, fmt.Errorf("duration dictionary missing months field")
	}
	monthsObj := Eval(monthsExpr, env)
	monthsInt, ok := monthsObj.(*Integer)
	if !ok {
		return 0, 0, fmt.Errorf("months must be an integer")
	}

	secondsExpr, ok := dict.Pairs["seconds"]
	if !ok {
		return 0, 0, fmt.Errorf("duration dictionary missing seconds field")
	}
	secondsObj := Eval(secondsExpr, env)
	secondsInt, ok := secondsObj.(*Integer)
	if !ok {
		return 0, 0, fmt.Errorf("seconds must be an integer")
	}

	return monthsInt.Value, secondsInt.Value, nil
}

// getDatetimeKind extracts the kind from a datetime dictionary (defaults to "datetime")
func getDatetimeKind(dict *Dictionary, env *Environment) string {
	if kindExpr, ok := dict.Pairs["kind"]; ok {
		kindObj := Eval(kindExpr, env)
		if kindStr, ok := kindObj.(*String); ok {
			return kindStr.Value
		}
	}
	return "datetime"
}

// getDatetimeUnix extracts the unix timestamp from a datetime dictionary
func getDatetimeUnix(dict *Dictionary, env *Environment) (int64, error) {
	unixExpr, ok := dict.Pairs["unix"]
	if !ok {
		return 0, fmt.Errorf("datetime dictionary missing unix field")
	}
	unixObj := Eval(unixExpr, env)
	unixInt, ok := unixObj.(*Integer)
	if !ok {
		return 0, fmt.Errorf("unix field is not an integer")
	}
	return unixInt.Value, nil
}

// datetimeDictToString converts a datetime dictionary to a human-friendly ISO 8601 string.
// Uses the "kind" field to determine output format: "datetime", "date", or "time"
func datetimeDictToString(dict *Dictionary) string {
	// Check for kind field to determine output format
	kind := "datetime" // default
	if kindExpr, ok := dict.Pairs["kind"]; ok {
		if kindLit, ok := kindExpr.(*ast.StringLiteral); ok {
			kind = kindLit.Value
		}
	}

	// Extract time components
	var hour, minute, second int64
	if hExpr, ok := dict.Pairs["hour"]; ok {
		if hLit, ok := hExpr.(*ast.IntegerLiteral); ok {
			hour = hLit.Value
		}
	}
	if minExpr, ok := dict.Pairs["minute"]; ok {
		if minLit, ok := minExpr.(*ast.IntegerLiteral); ok {
			minute = minLit.Value
		}
	}
	if sExpr, ok := dict.Pairs["second"]; ok {
		if sLit, ok := sExpr.(*ast.IntegerLiteral); ok {
			second = sLit.Value
		}
	}

	// Extract date components
	var year, month, day int64
	if yearExpr, ok := dict.Pairs["year"]; ok {
		if yLit, ok := yearExpr.(*ast.IntegerLiteral); ok {
			year = yLit.Value
		}
	}
	if mExpr, ok := dict.Pairs["month"]; ok {
		if mLit, ok := mExpr.(*ast.IntegerLiteral); ok {
			month = mLit.Value
		}
	}
	if dExpr, ok := dict.Pairs["day"]; ok {
		if dLit, ok := dExpr.(*ast.IntegerLiteral); ok {
			day = dLit.Value
		}
	}

	// Format based on kind
	switch kind {
	case "time":
		// Time only without seconds: HH:MM
		return fmt.Sprintf("%02d:%02d", hour, minute)

	case "time_seconds":
		// Time with seconds: HH:MM:SS
		return fmt.Sprintf("%02d:%02d:%02d", hour, minute, second)

	case "date":
		// Date only: YYYY-MM-DD
		return fmt.Sprintf("%04d-%02d-%02d", year, month, day)

	default:
		// Full datetime: YYYY-MM-DDTHH:MM:SSZ
		// If time is all zeros, still include it for datetime kind
		return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02dZ", year, month, day, hour, minute, second)
	}
}

// durationDictToString converts a duration dictionary to a human-readable string
func durationDictToString(dict *Dictionary) string {
	var months, seconds int64

	// Get months
	if monthsExpr, ok := dict.Pairs["months"]; ok {
		monthsObj := Eval(monthsExpr, dict.Env)
		if i, ok := monthsObj.(*Integer); ok {
			months = i.Value
		}
	}

	// Get seconds
	if secondsExpr, ok := dict.Pairs["seconds"]; ok {
		secondsObj := Eval(secondsExpr, dict.Env)
		if i, ok := secondsObj.(*Integer); ok {
			seconds = i.Value
		}
	}

	// Handle zero duration
	if months == 0 && seconds == 0 {
		return "0 seconds"
	}

	var parts []string
	isNegative := months < 0 || seconds < 0

	// Handle negative values
	if months < 0 {
		months = -months
	}
	if seconds < 0 {
		seconds = -seconds
	}

	// Convert months to years and months
	years := months / 12
	months = months % 12

	if years > 0 {
		if years == 1 {
			parts = append(parts, "1 year")
		} else {
			parts = append(parts, fmt.Sprintf("%d years", years))
		}
	}
	if months > 0 {
		if months == 1 {
			parts = append(parts, "1 month")
		} else {
			parts = append(parts, fmt.Sprintf("%d months", months))
		}
	}

	// Convert seconds to days, hours, minutes, seconds
	days := seconds / 86400
	seconds = seconds % 86400
	hours := seconds / 3600
	seconds = seconds % 3600
	minutes := seconds / 60
	seconds = seconds % 60

	if days > 0 {
		if days == 1 {
			parts = append(parts, "1 day")
		} else {
			parts = append(parts, fmt.Sprintf("%d days", days))
		}
	}
	if hours > 0 {
		if hours == 1 {
			parts = append(parts, "1 hour")
		} else {
			parts = append(parts, fmt.Sprintf("%d hours", hours))
		}
	}
	if minutes > 0 {
		if minutes == 1 {
			parts = append(parts, "1 minute")
		} else {
			parts = append(parts, fmt.Sprintf("%d minutes", minutes))
		}
	}
	if seconds > 0 {
		if seconds == 1 {
			parts = append(parts, "1 second")
		} else {
			parts = append(parts, fmt.Sprintf("%d seconds", seconds))
		}
	}

	result := strings.Join(parts, " ")
	if isNegative {
		return "-" + result
	}
	return result
}
