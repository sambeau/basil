package evaluator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Precompiled regex patterns for validation
var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	uuidRegex     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	timeRegex     = regexp.MustCompile(`^([01]?[0-9]|2[0-3]):[0-5][0-9](:[0-5][0-9])?$`)
	alphaRegex    = regexp.MustCompile(`^[a-zA-Z]+$`)
	alphanumRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	phoneRegex    = regexp.MustCompile(`^[\d\s\+\-\(\)\.]+$`)
	usPostalRegex = regexp.MustCompile(`^\d{5}(-\d{4})?$`)
	gbPostalRegex = regexp.MustCompile(`(?i)^[A-Z]{1,2}[0-9][0-9A-Z]?\s?[0-9][A-Z]{2}$`)
	isoDateRegex  = regexp.MustCompile(`^\d{4}-\d{1,2}-\d{1,2}$`)
	usDateRegex   = regexp.MustCompile(`^\d{1,2}/\d{1,2}/\d{4}$`)
	gbDateRegex   = regexp.MustCompile(`^\d{1,2}/\d{1,2}/\d{4}$`)
)

// loadValidModule returns the valid module as a StdlibModuleDict
func loadValidModule(env *Environment) Object {
	return &StdlibModuleDict{
		Exports: map[string]Object{
			// Type validators
			"string":  &Builtin{Fn: validString},
			"number":  &Builtin{Fn: validNumber},
			"integer": &Builtin{Fn: validInteger},
			"boolean": &Builtin{Fn: validBoolean},
			"array":   &Builtin{Fn: validArray},
			"dict":    &Builtin{Fn: validDict},

			// String validators
			"empty":        &Builtin{Fn: validEmpty},
			"minLen":       &Builtin{Fn: validMinLen},
			"maxLen":       &Builtin{Fn: validMaxLen},
			"length":       &Builtin{Fn: validLength},
			"matches":      &Builtin{Fn: validMatches},
			"alpha":        &Builtin{Fn: validAlpha},
			"alphanumeric": &Builtin{Fn: validAlphanumeric},
			"numeric":      &Builtin{Fn: validNumeric},

			// Number validators
			"min":      &Builtin{Fn: validMin},
			"max":      &Builtin{Fn: validMax},
			"between":  &Builtin{Fn: validBetween},
			"positive": &Builtin{Fn: validPositive},
			"negative": &Builtin{Fn: validNegative},

			// Format validators
			"email":      &Builtin{Fn: validEmail},
			"url":        &Builtin{Fn: validURL},
			"uuid":       &Builtin{Fn: validUUID},
			"phone":      &Builtin{Fn: validPhone},
			"creditCard": &Builtin{Fn: validCreditCard},
			"date":       &Builtin{Fn: validDate},
			"time":       &Builtin{Fn: validTime},

			// Locale-aware validators
			"postalCode": &Builtin{Fn: validPostalCode},
			"parseDate":  &Builtin{Fn: validParseDate},

			// Collection validators
			"contains": &Builtin{Fn: validContains},
			"oneOf":    &Builtin{Fn: validOneOf},
		},
	}
}

// =============================================================================
// Type Validators
// =============================================================================

func validString(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.string", len(args), 1)
	}
	_, ok := args[0].(*String)
	return nativeBoolToParsBoolean(ok)
}

func validNumber(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.number", len(args), 1)
	}
	switch args[0].(type) {
	case *Integer, *Float:
		return TRUE
	default:
		return FALSE
	}
}

func validInteger(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.integer", len(args), 1)
	}
	_, ok := args[0].(*Integer)
	return nativeBoolToParsBoolean(ok)
}

func validBoolean(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.boolean", len(args), 1)
	}
	_, ok := args[0].(*Boolean)
	return nativeBoolToParsBoolean(ok)
}

func validArray(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.array", len(args), 1)
	}
	_, ok := args[0].(*Array)
	return nativeBoolToParsBoolean(ok)
}

func validDict(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.dict", len(args), 1)
	}
	_, ok := args[0].(*Dictionary)
	return nativeBoolToParsBoolean(ok)
}

// =============================================================================
// String Validators
// =============================================================================

func validEmpty(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.empty", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}
	return nativeBoolToParsBoolean(strings.TrimSpace(str.Value) == "")
}

func validMinLen(args ...Object) Object {
	if len(args) != 2 {
		return newArityError("valid.minLen", len(args), 2)
	}
	str, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "valid.minLen", "string", args[0].Type())
	}
	n, ok := args[1].(*Integer)
	if !ok {
		return newTypeError("TYPE-0001", "valid.minLen", "integer", args[1].Type())
	}
	// Count runes, not bytes, for Unicode correctness
	runeCount := int64(len([]rune(str.Value)))
	return nativeBoolToParsBoolean(runeCount >= n.Value)
}

func validMaxLen(args ...Object) Object {
	if len(args) != 2 {
		return newArityError("valid.maxLen", len(args), 2)
	}
	str, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "valid.maxLen", "string", args[0].Type())
	}
	n, ok := args[1].(*Integer)
	if !ok {
		return newTypeError("TYPE-0001", "valid.maxLen", "integer", args[1].Type())
	}
	runeCount := int64(len([]rune(str.Value)))
	return nativeBoolToParsBoolean(runeCount <= n.Value)
}

func validLength(args ...Object) Object {
	if len(args) != 3 {
		return newArityError("valid.length", len(args), 3)
	}
	str, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "valid.length", "string", args[0].Type())
	}
	minN, ok := args[1].(*Integer)
	if !ok {
		return newTypeError("TYPE-0001", "valid.length", "integer", args[1].Type())
	}
	maxN, ok := args[2].(*Integer)
	if !ok {
		return newTypeError("TYPE-0001", "valid.length", "integer", args[2].Type())
	}
	runeCount := int64(len([]rune(str.Value)))
	return nativeBoolToParsBoolean(runeCount >= minN.Value && runeCount <= maxN.Value)
}

func validMatches(args ...Object) Object {
	if len(args) != 2 {
		return newArityError("valid.matches", len(args), 2)
	}
	str, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "valid.matches", "string", args[0].Type())
	}
	pattern, ok := args[1].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "valid.matches", "string", args[1].Type())
	}
	re, err := regexp.Compile(pattern.Value)
	if err != nil {
		return newValueError("VALUE-0003", map[string]any{
			"Function": "valid.matches",
			"Reason":   "invalid regex pattern: " + err.Error(),
		})
	}
	return nativeBoolToParsBoolean(re.MatchString(str.Value))
}

func validAlpha(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.alpha", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}
	if str.Value == "" {
		return FALSE
	}
	return nativeBoolToParsBoolean(alphaRegex.MatchString(str.Value))
}

func validAlphanumeric(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.alphanumeric", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}
	if str.Value == "" {
		return FALSE
	}
	return nativeBoolToParsBoolean(alphanumRegex.MatchString(str.Value))
}

func validNumeric(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.numeric", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}
	if str.Value == "" {
		return FALSE
	}
	// Try to parse as float (covers both int and float)
	_, err := strconv.ParseFloat(str.Value, 64)
	return nativeBoolToParsBoolean(err == nil)
}

// =============================================================================
// Number Validators
// =============================================================================

func validMin(args ...Object) Object {
	if len(args) != 2 {
		return newArityError("valid.min", len(args), 2)
	}
	x, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0001", "valid.min", "number", args[0].Type())
	}
	n, ok := toFloat64(args[1])
	if !ok {
		return newTypeError("TYPE-0001", "valid.min", "number", args[1].Type())
	}
	return nativeBoolToParsBoolean(x >= n)
}

func validMax(args ...Object) Object {
	if len(args) != 2 {
		return newArityError("valid.max", len(args), 2)
	}
	x, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0001", "valid.max", "number", args[0].Type())
	}
	n, ok := toFloat64(args[1])
	if !ok {
		return newTypeError("TYPE-0001", "valid.max", "number", args[1].Type())
	}
	return nativeBoolToParsBoolean(x <= n)
}

func validBetween(args ...Object) Object {
	if len(args) != 3 {
		return newArityError("valid.between", len(args), 3)
	}
	x, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0001", "valid.between", "number", args[0].Type())
	}
	lo, ok := toFloat64(args[1])
	if !ok {
		return newTypeError("TYPE-0001", "valid.between", "number", args[1].Type())
	}
	hi, ok := toFloat64(args[2])
	if !ok {
		return newTypeError("TYPE-0001", "valid.between", "number", args[2].Type())
	}
	return nativeBoolToParsBoolean(x >= lo && x <= hi)
}

func validPositive(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.positive", len(args), 1)
	}
	x, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0001", "valid.positive", "number", args[0].Type())
	}
	return nativeBoolToParsBoolean(x > 0)
}

func validNegative(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.negative", len(args), 1)
	}
	x, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0001", "valid.negative", "number", args[0].Type())
	}
	return nativeBoolToParsBoolean(x < 0)
}

// =============================================================================
// Format Validators
// =============================================================================

func validEmail(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.email", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}
	return nativeBoolToParsBoolean(emailRegex.MatchString(str.Value))
}

func validURL(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.url", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}
	// Check for http:// or https:// prefix and basic structure
	s := str.Value
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return FALSE
	}
	// Must have something after the protocol
	rest := strings.TrimPrefix(strings.TrimPrefix(s, "https://"), "http://")
	if rest == "" || !strings.Contains(rest, ".") {
		return FALSE
	}
	return TRUE
}

func validUUID(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.uuid", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}
	return nativeBoolToParsBoolean(uuidRegex.MatchString(str.Value))
}

func validPhone(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.phone", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}
	if str.Value == "" {
		return FALSE
	}
	// Must match pattern and have at least some digits
	if !phoneRegex.MatchString(str.Value) {
		return FALSE
	}
	// Count digits - must have at least 7
	digitCount := 0
	for _, r := range str.Value {
		if unicode.IsDigit(r) {
			digitCount++
		}
	}
	return nativeBoolToParsBoolean(digitCount >= 7)
}

// luhnCheck validates a number string using the Luhn algorithm
func luhnCheck(number string) bool {
	// Strip non-digits
	var digits []int
	for _, r := range number {
		if r >= '0' && r <= '9' {
			digits = append(digits, int(r-'0'))
		}
	}
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}

	// Luhn algorithm
	sum := 0
	isSecond := false
	for i := len(digits) - 1; i >= 0; i-- {
		d := digits[i]
		if isSecond {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		isSecond = !isSecond
	}
	return sum%10 == 0
}

func validCreditCard(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.creditCard", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}
	return nativeBoolToParsBoolean(luhnCheck(str.Value))
}

func validTime(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.time", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}
	return nativeBoolToParsBoolean(timeRegex.MatchString(str.Value))
}

// =============================================================================
// Date Validators (Locale-aware)
// =============================================================================

// parseAndValidateDate parses a date string according to locale and validates it's real
func parseAndValidateDate(dateStr, locale string) (time.Time, bool) {
	locale = strings.ToUpper(locale)
	var layout string
	var datePattern *regexp.Regexp

	switch locale {
	case "US":
		layout = "1/2/2006"
		datePattern = usDateRegex
	case "GB":
		layout = "2/1/2006"
		datePattern = gbDateRegex
	case "ISO", "":
		layout = "2006-1-2"
		datePattern = isoDateRegex
	default:
		return time.Time{}, false
	}

	if !datePattern.MatchString(dateStr) {
		return time.Time{}, false
	}

	t, err := time.Parse(layout, dateStr)
	if err != nil {
		return time.Time{}, false
	}

	// Verify the parsed date is valid by checking that format->parse roundtrips
	// This catches invalid dates like Feb 30 where Go would roll over to Mar 2
	formatted := t.Format(layout)
	roundtrip, _ := time.Parse(layout, formatted)
	if !roundtrip.Equal(t) {
		return time.Time{}, false
	}

	// Also verify the input parses to the same date (catches rollover)
	// by checking year/month/day components match
	var year, month, day int
	switch locale {
	case "US":
		fmt.Sscanf(dateStr, "%d/%d/%d", &month, &day, &year)
	case "GB":
		fmt.Sscanf(dateStr, "%d/%d/%d", &day, &month, &year)
	default:
		fmt.Sscanf(dateStr, "%d-%d-%d", &year, &month, &day)
	}
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return time.Time{}, false
	}

	return t, true
}

func validDate(args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return newArityErrorRange("valid.date", len(args), 1, 2)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}

	locale := "ISO"
	if len(args) == 2 {
		loc, ok := args[1].(*String)
		if !ok {
			return newTypeError("TYPE-0001", "valid.date", "string", args[1].Type())
		}
		locale = loc.Value
	}

	_, valid := parseAndValidateDate(str.Value, locale)
	return nativeBoolToParsBoolean(valid)
}

func validParseDate(args ...Object) Object {
	if len(args) != 2 {
		return newArityError("valid.parseDate", len(args), 2)
	}
	str, ok := args[0].(*String)
	if !ok {
		return NULL
	}
	loc, ok := args[1].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "valid.parseDate", "string", args[1].Type())
	}

	t, valid := parseAndValidateDate(str.Value, loc.Value)
	if !valid {
		return NULL
	}
	return &String{Value: t.Format("2006-01-02")}
}

// =============================================================================
// Postal Code Validator (Locale-aware)
// =============================================================================

func validPostalCode(args ...Object) Object {
	if len(args) != 2 {
		return newArityError("valid.postalCode", len(args), 2)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}
	loc, ok := args[1].(*String)
	if !ok {
		return newTypeError("TYPE-0001", "valid.postalCode", "string", args[1].Type())
	}

	locale := strings.ToUpper(loc.Value)
	switch locale {
	case "US":
		return nativeBoolToParsBoolean(usPostalRegex.MatchString(str.Value))
	case "GB":
		return nativeBoolToParsBoolean(gbPostalRegex.MatchString(str.Value))
	default:
		return newValueError("VALUE-0003", map[string]any{
			"Function": "valid.postalCode",
			"Reason":   "unsupported locale: " + loc.Value + ". Supported: US, GB",
		})
	}
}

// =============================================================================
// Collection Validators
// =============================================================================

func validContains(args ...Object) Object {
	if len(args) != 2 {
		return newArityError("valid.contains", len(args), 2)
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newTypeError("TYPE-0001", "valid.contains", "array", args[0].Type())
	}
	needle := args[1]

	for _, elem := range arr.Elements {
		if objectsEqual(elem, needle) {
			return TRUE
		}
	}
	return FALSE
}

func validOneOf(args ...Object) Object {
	if len(args) != 2 {
		return newArityError("valid.oneOf", len(args), 2)
	}
	needle := args[0]
	arr, ok := args[1].(*Array)
	if !ok {
		return newTypeError("TYPE-0001", "valid.oneOf", "array", args[1].Type())
	}

	for _, elem := range arr.Elements {
		if objectsEqual(elem, needle) {
			return TRUE
		}
	}
	return FALSE
}
