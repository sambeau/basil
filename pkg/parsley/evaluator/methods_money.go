// Package evaluator provides money method implementations via declarative registry.
// This file implements money methods for FEAT-111: Declarative Method Registry
// Updated for FEAT-121: Unified Formatter API
package evaluator

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// MoneyMethodRegistry defines all methods available on money values.
// This is the single source of truth for money method dispatch and introspection.
var MoneyMethodRegistry MethodRegistry

func init() {
	MoneyMethodRegistry = MethodRegistry{
		"fmt": {
			Fn:          moneyFmt,
			Arity:       "0-2",
			Description: "Format with style and/or locale",
		},
		"format": {
			Fn:          moneyFmt,
			Arity:       "0-2",
			Description: "Format with style and/or locale (alias for fmt)",
		},
		"short": {
			Fn:          moneyShort,
			Arity:       "0-1",
			Description: "Compact format ($1K)",
		},
		"medium": {
			Fn:          moneyMedium,
			Arity:       "0-1",
			Description: "Standard currency format",
		},
		"long": {
			Fn:          moneyLong,
			Arity:       "0-1",
			Description: "Full precision format",
		},
		"full": {
			Fn:          moneyFull,
			Arity:       "0-1",
			Description: "Spelled out currency name",
		},
		"abs": {
			Fn:          moneyAbsMethod,
			Arity:       "0",
			Description: "Absolute value",
		},
		"negate": {
			Fn:          moneyNegate,
			Arity:       "0",
			Description: "Negate amount",
		},
		"split": {
			Fn:          moneySplitMethod,
			Arity:       "1",
			Description: "Split into n parts that sum to original",
		},
		"toJSON": {
			Fn:          moneyToJSON,
			Arity:       "0",
			Description: "Convert to JSON string",
		},
		"toBox": {
			Fn:          moneyToBoxMethod,
			Arity:       "0-1",
			Description: "Render as box diagram",
		},
		"repr": {
			Fn:          moneyRepr,
			Arity:       "0",
			Description: "Get representation string",
		},
		"toDict": {
			Fn:          moneyToDict,
			Arity:       "0",
			Description: "Convert to dictionary",
		},
		"inspect": {
			Fn:          moneyInspect,
			Arity:       "0",
			Description: "Get debug dictionary with internal values",
		},
	}
	RegisterMethodRegistry("money", MoneyMethodRegistry)
}

// Money formatting methods

// moneyFmt formats money with style and locale options.
// Overloads:
//   - .fmt() - medium style, default locale
//   - .fmt(n) - precision (decimal places)
//   - .fmt("style") - named style
//   - .fmt("style", "locale") - style + locale
//   - .fmt({...}) - options dictionary
func moneyFmt(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	opts, err := parseFormatArgs(args)
	if err != nil {
		return err
	}
	return formatMoneyWithOpts(money, opts)
}

func moneyShort(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	opts, err := parseStyleMethodArgs("short", args, "short")
	if err != nil {
		return err
	}
	return formatMoneyWithOpts(money, opts)
}

func moneyMedium(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	opts, err := parseStyleMethodArgs("medium", args, "medium")
	if err != nil {
		return err
	}
	return formatMoneyWithOpts(money, opts)
}

func moneyLong(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	opts, err := parseStyleMethodArgs("long", args, "long")
	if err != nil {
		return err
	}
	return formatMoneyWithOpts(money, opts)
}

func moneyFull(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	opts, err := parseStyleMethodArgs("full", args, "full")
	if err != nil {
		return err
	}
	return formatMoneyWithOpts(money, opts)
}

// formatMoneyWithOpts formats money using FormatOpts.
func formatMoneyWithOpts(money *Money, opts FormatOpts) Object {
	switch opts.Style {
	case "short":
		return formatMoneyShort(money, opts.Locale)
	case "medium":
		return formatMoney(money, opts.Locale)
	case "long":
		return formatMoneyLong(money, opts)
	case "full":
		return formatMoneyFull(money, opts.Locale)
	default:
		return formatMoney(money, opts.Locale)
	}
}

// formatMoneyShort formats money in compact form ($1K, €500)
func formatMoneyShort(money *Money, locale string) Object {
	// Get the actual amount
	divisor := math.Pow10(int(money.Scale))
	amount := float64(money.Amount) / divisor

	// Get currency symbol
	symbol := getCurrencySymbol(money.Currency, locale)

	// Humanize the number
	humanized := humanizeNumber(math.Abs(amount), locale)

	// Build result with symbol placement based on locale
	var result string
	if amount < 0 {
		result = "-"
	}

	if symbolAfter(locale) {
		result += humanized + symbol
	} else {
		result += symbol + humanized
	}

	return &String{Value: result}
}

// formatMoneyLong formats money with full precision
func formatMoneyLong(money *Money, opts FormatOpts) Object {
	// If precision specified, use it; otherwise use scale
	precision := int(money.Scale)
	if opts.Precision >= 0 {
		precision = opts.Precision
	}

	divisor := math.Pow10(int(money.Scale))
	amount := float64(money.Amount) / divisor

	// Get currency symbol
	symbol := getCurrencySymbol(money.Currency, opts.Locale)

	// Format number with precision
	formatted := formatNumberWithPrecision(math.Abs(amount), precision, opts.Locale)

	// Build result
	var result string
	if amount < 0 {
		result = "-"
	}

	if symbolAfter(opts.Locale) {
		result += formatted + " " + symbol
	} else {
		result += symbol + formatted
	}

	return &String{Value: result}
}

// formatMoneyFull formats money with spelled out currency name
func formatMoneyFull(money *Money, locale string) Object {
	divisor := math.Pow10(int(money.Scale))
	amount := float64(money.Amount) / divisor

	// Format the number
	formatted := formatNumberWithPrecision(math.Abs(amount), int(money.Scale), locale)

	// Get currency name
	currencyName := getCurrencyName(money.Currency, locale, math.Abs(amount) != 1)

	// Build result: "1,234.56 US dollars"
	var result string
	if amount < 0 {
		result = "-"
	}
	result += formatted + " " + currencyName

	return &String{Value: result}
}

// formatNumberWithPrecision formats a number with a specific number of decimal places.
func formatNumberWithPrecision(value float64, precision int, locale string) string {
	// Get locale-specific separators
	thousandSep, decimalSep := getLocaleSeparators(locale)

	// Round to precision
	multiplier := math.Pow10(precision)
	rounded := math.Round(value*multiplier) / multiplier

	// Format integer part with thousand separators
	intPart := int64(rounded)
	intStr := formatIntegerWithSeparator(intPart, thousandSep)

	if precision == 0 {
		return intStr
	}

	// Get decimal part
	decPart := rounded - float64(intPart)
	decStr := fmt.Sprintf("%.*f", precision, decPart)
	// Remove "0." prefix
	if len(decStr) > 2 {
		decStr = decStr[2:]
	} else {
		decStr = strings.Repeat("0", precision)
	}

	return intStr + decimalSep + decStr
}

// formatIntegerWithSeparator formats an integer with thousand separators.
func formatIntegerWithSeparator(n int64, sep string) string {
	if n < 0 {
		n = -n
	}
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}

	// Insert separators from right to left
	var result strings.Builder
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteString(sep)
		}
		result.WriteRune(c)
	}
	return result.String()
}

// getLocaleSeparators returns the thousand and decimal separators for a locale.
func getLocaleSeparators(locale string) (thousand, decimal string) {
	switch locale {
	case "de-DE", "fr-FR", "es-ES", "it-IT", "pt-BR", "nl-NL":
		return ".", ","
	case "fr-CA":
		return " ", ","
	default: // en-US, en-GB, etc.
		return ",", "."
	}
}

// getCurrencySymbol returns the currency symbol for a currency code and locale.
func getCurrencySymbol(currency, locale string) string {
	switch currency {
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	case "JPY":
		return "¥"
	case "CNY":
		return "¥"
	case "CHF":
		return "CHF"
	case "CAD":
		if locale == "en-CA" || locale == "en-US" {
			return "CA$"
		}
		return "$"
	case "AUD":
		return "A$"
	default:
		return currency + " "
	}
}

// getCurrencyName returns the spelled-out currency name.
func getCurrencyName(currency, locale string, plural bool) string {
	names := map[string]map[string][2]string{
		"USD": {
			"en-US": {"US dollar", "US dollars"},
			"en-GB": {"US dollar", "US dollars"},
			"de-DE": {"US-Dollar", "US-Dollar"},
			"fr-FR": {"dollar américain", "dollars américains"},
			"es-ES": {"dólar estadounidense", "dólares estadounidenses"},
		},
		"EUR": {
			"en-US": {"euro", "euros"},
			"en-GB": {"euro", "euros"},
			"de-DE": {"Euro", "Euro"},
			"fr-FR": {"euro", "euros"},
			"es-ES": {"euro", "euros"},
		},
		"GBP": {
			"en-US": {"British pound", "British pounds"},
			"en-GB": {"pound sterling", "pounds sterling"},
			"de-DE": {"Pfund Sterling", "Pfund Sterling"},
			"fr-FR": {"livre sterling", "livres sterling"},
		},
		"JPY": {
			"en-US": {"Japanese yen", "Japanese yen"},
			"en-GB": {"Japanese yen", "Japanese yen"},
			"de-DE": {"Japanischer Yen", "Japanische Yen"},
		},
	}

	if localeNames, ok := names[currency]; ok {
		if pair, ok := localeNames[locale]; ok {
			if plural {
				return pair[1]
			}
			return pair[0]
		}
		// Fall back to en-US
		if pair, ok := localeNames["en-US"]; ok {
			if plural {
				return pair[1]
			}
			return pair[0]
		}
	}

	// Generic fallback
	if plural {
		return currency
	}
	return currency
}

// symbolAfter returns true if the currency symbol should come after the number.
func symbolAfter(locale string) bool {
	switch locale {
	case "de-DE", "fr-FR", "es-ES", "it-IT", "pt-BR", "nl-NL":
		return true
	default:
		return false
	}
}

// Other money methods

func moneyAbsMethod(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	amount := money.Amount
	if amount < 0 {
		amount = -amount
	}
	return &Money{
		Amount:   amount,
		Currency: money.Currency,
		Scale:    money.Scale,
	}
}

func moneyNegate(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	return &Money{
		Amount:   -money.Amount,
		Currency: money.Currency,
		Scale:    money.Scale,
	}
}

func moneySplitMethod(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	nArg, ok := args[0].(*Integer)
	if !ok {
		return newTypeError("TYPE-0012", "split", "an integer", args[0].Type())
	}
	n := nArg.Value
	if n <= 0 {
		return newStructuredError("VAL-0021", map[string]any{"Function": "split", "Expected": "a positive integer", "Got": n})
	}
	return splitMoney(money, n)
}

func moneyToJSON(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	// Format the amount as a decimal string to preserve precision
	amountStr := money.formatAmount()
	currencyJSON, _ := json.Marshal(money.Currency)
	return &String{Value: fmt.Sprintf(`{"amount":"%s","currency":%s}`, amountStr, currencyJSON)}
}

func moneyToBoxMethod(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	return moneyToBox(money, args)
}

func moneyRepr(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	return &String{Value: money.Inspect()}
}

func moneyToDict(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	// Calculate user-friendly amount (e.g., 50.00 not 5000)
	divisor := math.Pow10(int(money.Scale))
	amount := float64(money.Amount) / divisor
	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"amount":   createLiteralExpression(&Float{Value: amount}),
			"currency": createLiteralExpression(&String{Value: money.Currency}),
		},
		Env: NewEnvironment(),
	}
}

func moneyInspect(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"__type":   createLiteralExpression(&String{Value: "money"}),
			"amount":   createLiteralExpression(&Integer{Value: money.Amount}),
			"currency": createLiteralExpression(&String{Value: money.Currency}),
			"scale":    createLiteralExpression(&Integer{Value: int64(money.Scale)}),
		},
		Env: NewEnvironment(),
	}
}
