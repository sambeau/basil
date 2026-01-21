// eval_datetime.go - Datetime and duration conversion helpers for the Parsley evaluator
//
// This file contains functions for converting between Go time.Time/Duration values
// and Parsley dictionary representations. Includes formatting and string conversion.

package evaluator

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/araddon/dateparse"
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
	// Helper to extract int64 from expression (handles both AST literals and evaluated objects)
	getInt := func(key string) int64 {
		expr, ok := dict.Pairs[key]
		if !ok {
			return 0
		}
		// Try AST literal first
		if lit, ok := expr.(*ast.IntegerLiteral); ok {
			return lit.Value
		}
		// Fall back to evaluation
		obj := Eval(expr, dict.Env)
		if i, ok := obj.(*Integer); ok {
			return i.Value
		}
		return 0
	}

	// Helper to extract string from expression
	getString := func(key string) string {
		expr, ok := dict.Pairs[key]
		if !ok {
			return ""
		}
		// Try AST literal first
		if lit, ok := expr.(*ast.StringLiteral); ok {
			return lit.Value
		}
		// Fall back to evaluation
		obj := Eval(expr, dict.Env)
		if s, ok := obj.(*String); ok {
			return s.Value
		}
		return ""
	}

	// Check for kind field to determine output format
	kind := getString("kind")
	if kind == "" {
		kind = "datetime" // default
	}

	// Extract time components
	hour := getInt("hour")
	minute := getInt("minute")
	second := getInt("second")

	// Extract date components
	year := getInt("year")
	month := getInt("month")
	day := getInt("day")

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

// ============================================================================
// Locale-aware Date/Time Parsing
// ============================================================================

// LocaleConfig defines locale-specific parsing behavior
type LocaleConfig struct {
	DayFirst   bool           // DD/MM vs MM/DD for ambiguous dates
	MonthNames map[string]int // Localized month name → number (1-12)
}

// English month names (used by en-US and en-GB)
var englishMonths = map[string]int{
	"january": 1, "jan": 1,
	"february": 2, "feb": 2,
	"march": 3, "mar": 3,
	"april": 4, "apr": 4,
	"may": 5,
	"june": 6, "jun": 6,
	"july": 7, "jul": 7,
	"august": 8, "aug": 8,
	"september": 9, "sep": 9, "sept": 9,
	"october": 10, "oct": 10,
	"november": 11, "nov": 11,
	"december": 12, "dec": 12,
}

// French month names
var frenchMonths = map[string]int{
	"janvier": 1, "janv": 1,
	"février": 2, "fevrier": 2, "fév": 2, "fev": 2,
	"mars": 3,
	"avril": 4, "avr": 4,
	"mai": 5,
	"juin": 6,
	"juillet": 7, "juil": 7,
	"août": 8, "aout": 8,
	"septembre": 9, "sept": 9,
	"octobre": 10, "oct": 10,
	"novembre": 11, "nov": 11,
	"décembre": 12, "decembre": 12, "déc": 12, "dec": 12,
}

// German month names
var germanMonths = map[string]int{
	"januar": 1, "jan": 1,
	"februar": 2, "feb": 2,
	"märz": 3, "marz": 3, "mär": 3, "mar": 3,
	"april": 4, "apr": 4,
	"mai": 5,
	"juni": 6, "jun": 6,
	"juli": 7, "jul": 7,
	"august": 8, "aug": 8,
	"september": 9, "sep": 9, "sept": 9,
	"oktober": 10, "okt": 10,
	"november": 11, "nov": 11,
	"dezember": 12, "dez": 12,
}

// Spanish month names
var spanishMonths = map[string]int{
	"enero": 1, "ene": 1,
	"febrero": 2, "feb": 2,
	"marzo": 3, "mar": 3,
	"abril": 4, "abr": 4,
	"mayo": 5, "may": 5,
	"junio": 6, "jun": 6,
	"julio": 7, "jul": 7,
	"agosto": 8, "ago": 8,
	"septiembre": 9, "sep": 9, "sept": 9,
	"octubre": 10, "oct": 10,
	"noviembre": 11, "nov": 11,
	"diciembre": 12, "dic": 12,
}

// localeConfigs maps locale codes to their configurations
var localeConfigs = map[string]*LocaleConfig{
	"en-US": {DayFirst: false, MonthNames: englishMonths},
	"en-GB": {DayFirst: true, MonthNames: englishMonths},
	"fr-FR": {DayFirst: true, MonthNames: frenchMonths},
	"de-DE": {DayFirst: true, MonthNames: germanMonths},
	"es-ES": {DayFirst: true, MonthNames: spanishMonths},
}

// getLocaleConfig returns the configuration for a locale, defaulting to en-US
func getLocaleConfig(locale string) *LocaleConfig {
	if config, ok := localeConfigs[locale]; ok {
		return config
	}
	return localeConfigs["en-US"]
}

// mergeMonthNames creates a combined map of all month names from multiple locales
func mergeMonthNames(configs ...*LocaleConfig) map[string]int {
	result := make(map[string]int)
	for _, config := range configs {
		for name, num := range config.MonthNames {
			result[name] = num
		}
	}
	return result
}

// allMonthNames is a combined map for recognizing month names from any supported locale
var allMonthNames = mergeMonthNames(
	localeConfigs["en-US"],
	localeConfigs["fr-FR"],
	localeConfigs["de-DE"],
	localeConfigs["es-ES"],
)

// normalizeMonthNames replaces localized month names with English equivalents
func normalizeMonthNames(input string, locale *LocaleConfig) string {
	// Build a regex pattern to match any month name from the locale
	if locale == nil || len(locale.MonthNames) == 0 {
		return input
	}

	// Collect month names that need translation (non-English)
	englishNames := map[string]bool{
		"january": true, "jan": true,
		"february": true, "feb": true,
		"march": true, "mar": true,
		"april": true, "apr": true,
		"may": true,
		"june": true, "jun": true,
		"july": true, "jul": true,
		"august": true, "aug": true,
		"september": true, "sep": true, "sept": true,
		"october": true, "oct": true,
		"november": true, "nov": true,
		"december": true, "dec": true,
	}

	// English month names for replacement (full names)
	monthToEnglish := []string{
		"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}

	result := input
	for monthName, monthNum := range locale.MonthNames {
		// Skip if it's already an English name
		if englishNames[strings.ToLower(monthName)] {
			continue
		}

		// Create case-insensitive replacement
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(monthName) + `\b`)
		result = pattern.ReplaceAllString(result, monthToEnglish[monthNum])
	}

	return result
}

// ParsedDateKind indicates what components were found in the parsed string
type ParsedDateKind string

const (
	KindDate     ParsedDateKind = "date"
	KindTime     ParsedDateKind = "time"
	KindDatetime ParsedDateKind = "datetime"
)

// parseFlexibleDateTime parses a date/time string with locale awareness
func parseFlexibleDateTime(input string, locale *LocaleConfig, tzName string, strict bool) (time.Time, ParsedDateKind, error) {
	if input == "" {
		return time.Time{}, "", fmt.Errorf("empty input")
	}

	// 1. Normalize localized month names to English
	normalized := normalizeMonthNames(input, locale)

	// 2. Load timezone (default UTC)
	loc := time.UTC
	if tzName != "" && tzName != "UTC" {
		var err error
		loc, err = time.LoadLocation(tzName)
		if err != nil {
			return time.Time{}, "", fmt.Errorf("unknown timezone: %s", tzName)
		}
	}

	// 3. Configure dateparse options based on locale
	var opts []dateparse.ParserOption
	if locale != nil {
		opts = append(opts, dateparse.PreferMonthFirst(!locale.DayFirst))
	} else {
		opts = append(opts, dateparse.PreferMonthFirst(true))
	}

	// 4. Parse with dateparse
	t, err := dateparse.ParseIn(normalized, loc, opts...)
	if err != nil {
		return time.Time{}, "", err
	}

	// 5. Detect kind based on what was parsed
	kind := detectParsedKind(normalized, t)

	return t, kind, nil
}

// detectParsedKind tries to determine if the input was date-only, time-only, or full datetime
func detectParsedKind(input string, t time.Time) ParsedDateKind {
	lower := strings.ToLower(strings.TrimSpace(input))

	// Time-only patterns: starts with digit and contains : but no date-like separators
	timeOnlyPattern := regexp.MustCompile(`^(\d{1,2}:\d{2}|\d{1,2}\s*(am|pm|a\.m\.|p\.m\.))`)
	if timeOnlyPattern.MatchString(lower) && !containsDateIndicator(lower) {
		return KindTime
	}

	// Check if time components are all zero (likely date-only input)
	if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && !containsTimeIndicator(lower) {
		return KindDate
	}

	return KindDatetime
}

// containsDateIndicator checks if input contains date-like patterns
func containsDateIndicator(input string) bool {
	// Check for date separators with numbers
	datePattern := regexp.MustCompile(`\d+[/\-\.]\d+[/\-\.]\d+|\d+\s+(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)|` +
		`(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)\w*\s+\d+`)
	return datePattern.MatchString(input)
}

// containsTimeIndicator checks if input explicitly contains time indicators
func containsTimeIndicator(input string) bool {
	// Check for time patterns
	timePattern := regexp.MustCompile(`\d{1,2}:\d{2}|am|pm|a\.m\.|p\.m\.`)
	return timePattern.MatchString(input)
}

// parseTimeOnly parses a time-only string (no date component)
func parseTimeOnly(input string) (time.Time, error) {
	if input == "" {
		return time.Time{}, fmt.Errorf("empty input")
	}

	// Try various time formats
	formats := []string{
		"15:04:05.999999999",
		"15:04:05",
		"15:04",
		"3:04:05 PM",
		"3:04:05 pm",
		"3:04 PM",
		"3:04 pm",
		"3:04PM",
		"3:04pm",
		"3PM",
		"3pm",
		"3 PM",
		"3 pm",
	}

	input = strings.TrimSpace(input)

	for _, format := range formats {
		if t, err := time.Parse(format, input); err == nil {
			// Return with a zero date (just the time component)
			return time.Date(0, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
		}
	}

	// Fall back to dateparse but verify it looks like time-only
	t, err := dateparse.ParseAny(input)
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot parse time: %s", input)
	}

	// Check if the result has a meaningful date - if so, it wasn't time-only input
	if t.Year() != 0 && t.Year() != time.Now().Year() {
		return time.Time{}, fmt.Errorf("expected time-only, got date: %s", input)
	}

	return time.Date(0, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
}

// extractParseOptions extracts locale, strict, and timezone from an options dictionary
func extractParseOptions(opts *Dictionary, env *Environment) (locale string, strict bool, timezone string) {
	locale = "en-US"
	strict = false
	timezone = "UTC"

	if opts == nil {
		return
	}

	// Extract locale
	if localeExpr, ok := opts.Pairs["locale"]; ok {
		localeObj := Eval(localeExpr, env)
		if localeStr, ok := localeObj.(*String); ok {
			locale = localeStr.Value
		}
	}

	// Extract strict
	if strictExpr, ok := opts.Pairs["strict"]; ok {
		strictObj := Eval(strictExpr, env)
		if strictBool, ok := strictObj.(*Boolean); ok {
			strict = strictBool.Value
		}
	}

	// Extract timezone
	if tzExpr, ok := opts.Pairs["timezone"]; ok {
		tzObj := Eval(tzExpr, env)
		if tzStr, ok := tzObj.(*String); ok {
			timezone = tzStr.Value
		}
	}

	return
}
