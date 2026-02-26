package evaluator

import (
	"regexp"
	"strings"
)

// Precompiled regex patterns for validation
var (
	uuidRegex     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	ulidRegex     = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
	nanoidRegex   = regexp.MustCompile(`^[0-9A-Za-z_-]+$`)
	cuidRegex     = regexp.MustCompile(`^c[0-9a-z]{24}$`)
	usPostalRegex = regexp.MustCompile(`^\d{5}(-\d{4})?$`)
	gbPostalRegex = regexp.MustCompile(`(?i)^[A-Z]{1,2}[0-9][0-9A-Z]?\s?[0-9][A-Z]{2}$`)
)

var validModuleMeta = ModuleMeta{
	Description: "Validation predicates for IDs, financial data, and locale-specific formats",
	Exports: map[string]ExportMeta{
		// ID validators
		"uuid":   {Kind: "function", Arity: "1", Description: "Check UUID v4/v7 format"},
		"ulid":   {Kind: "function", Arity: "1", Description: "Check ULID format"},
		"nanoid": {Kind: "function", Arity: "1-2", Description: "Check NanoID format (length?)"},
		"cuid":   {Kind: "function", Arity: "1", Description: "Check CUID2 format"},
		// Financial validators
		"creditCard": {Kind: "function", Arity: "1", Description: "Check credit card number (Luhn)"},
		"luhn":       {Kind: "function", Arity: "1", Description: "Check Luhn algorithm"},
		// Locale-aware validators
		"postalCode": {Kind: "function", Arity: "2", Description: "Check postal code format (locale)"},
	},
}

// loadValidModule returns the valid module as a StdlibModuleDict
func loadValidModule(env *Environment) Object {
	return &StdlibModuleDict{
		Meta: &validModuleMeta,
		Exports: map[string]Object{
			// ID validators
			"uuid":   &Builtin{Fn: validUUID},
			"ulid":   &Builtin{Fn: validULID},
			"nanoid": &Builtin{Fn: validNanoID},
			"cuid":   &Builtin{Fn: validCUID},

			// Financial validators
			"creditCard": &Builtin{Fn: validCreditCard},
			"luhn":       &Builtin{Fn: validLuhn},

			// Locale-aware validators
			"postalCode": &Builtin{Fn: validPostalCode},
		},
	}
}

// =============================================================================
// ID Validators
// =============================================================================

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

func validULID(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.ulid", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}
	return nativeBoolToParsBoolean(ulidRegex.MatchString(str.Value))
}

func validNanoID(args ...Object) Object {
	if len(args) < 1 || len(args) > 2 {
		return newArityErrorRange("valid.nanoid", len(args), 1, 2)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}

	// Default NanoID length is 21
	expectedLen := 21
	if len(args) == 2 {
		lenArg, ok := args[1].(*Integer)
		if !ok {
			return newTypeError("TYPE-0013", "valid.nanoid", "an integer", args[1].Type())
		}
		expectedLen = int(lenArg.Value)
	}

	// Check character set and length
	if len(str.Value) != expectedLen {
		return FALSE
	}
	return nativeBoolToParsBoolean(nanoidRegex.MatchString(str.Value))
}

func validCUID(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.cuid", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}
	return nativeBoolToParsBoolean(cuidRegex.MatchString(str.Value))
}

// =============================================================================
// Financial Validators
// =============================================================================

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

func validLuhn(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("valid.luhn", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return FALSE
	}
	return nativeBoolToParsBoolean(luhnCheckGeneric(str.Value))
}

// luhnCheckGeneric validates any number string using the Luhn algorithm (no length restriction)
func luhnCheckGeneric(number string) bool {
	// Strip non-digits
	var digits []int
	for _, r := range number {
		if r >= '0' && r <= '9' {
			digits = append(digits, int(r-'0'))
		}
	}
	if len(digits) == 0 {
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

// =============================================================================
// Locale-Aware Validators
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
