// eval_locale.go - Locale and formatting helpers for the Parsley evaluator
//
// This file contains helper functions for locale-aware formatting of numbers,
// currencies, percentages, and dates. These functions handle mapping between
// locale strings and locale-specific formatting rules.

package evaluator

import (
	"strings"
	"time"

	"github.com/goodsign/monday"
	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// getMondayLocale maps a locale string to a monday.Locale for date formatting.
// Supports common locale codes with fallbacks.
func getMondayLocale(locale string) monday.Locale {
	// Normalize locale string
	locale = strings.ToLower(strings.ReplaceAll(locale, "-", "_"))

	localeMap := map[string]monday.Locale{
		"en":    monday.LocaleEnUS,
		"en_us": monday.LocaleEnUS,
		"en_gb": monday.LocaleEnGB,
		"en_au": monday.LocaleEnUS, // Fallback to US
		"de":    monday.LocaleDeDE,
		"de_de": monday.LocaleDeDE,
		"de_at": monday.LocaleDeDE,
		"de_ch": monday.LocaleDeDE,
		"fr":    monday.LocaleFrFR,
		"fr_fr": monday.LocaleFrFR,
		"fr_ca": monday.LocaleFrCA,
		"fr_be": monday.LocaleFrFR,
		"es":    monday.LocaleEsES,
		"es_es": monday.LocaleEsES,
		"es_mx": monday.LocaleEsES,
		"it":    monday.LocaleItIT,
		"it_it": monday.LocaleItIT,
		"pt":    monday.LocalePtPT,
		"pt_pt": monday.LocalePtPT,
		"pt_br": monday.LocalePtBR,
		"nl":    monday.LocaleNlNL,
		"nl_nl": monday.LocaleNlNL,
		"nl_be": monday.LocaleNlBE,
		"ru":    monday.LocaleRuRU,
		"ru_ru": monday.LocaleRuRU,
		"pl":    monday.LocalePlPL,
		"pl_pl": monday.LocalePlPL,
		"cs":    monday.LocaleCsCZ,
		"cs_cz": monday.LocaleCsCZ,
		"da":    monday.LocaleDaDK,
		"da_dk": monday.LocaleDaDK,
		"fi":    monday.LocaleFiFI,
		"fi_fi": monday.LocaleFiFI,
		"sv":    monday.LocaleSvSE,
		"sv_se": monday.LocaleSvSE,
		"nb":    monday.LocaleNbNO,
		"nb_no": monday.LocaleNbNO,
		"nn":    monday.LocaleNnNO,
		"nn_no": monday.LocaleNnNO,
		"ja":    monday.LocaleJaJP,
		"ja_jp": monday.LocaleJaJP,
		"zh":    monday.LocaleZhCN,
		"zh_cn": monday.LocaleZhCN,
		"zh_tw": monday.LocaleZhTW,
		"ko":    monday.LocaleKoKR,
		"ko_kr": monday.LocaleKoKR,
		"tr":    monday.LocaleTrTR,
		"tr_tr": monday.LocaleTrTR,
		"uk":    monday.LocaleUkUA,
		"uk_ua": monday.LocaleUkUA,
		"el":    monday.LocaleElGR,
		"el_gr": monday.LocaleElGR,
		"ro":    monday.LocaleRoRO,
		"ro_ro": monday.LocaleRoRO,
		"hu":    monday.LocaleHuHU,
		"hu_hu": monday.LocaleHuHU,
		"bg":    monday.LocaleBgBG,
		"bg_bg": monday.LocaleBgBG,
		"id":    monday.LocaleIdID,
		"id_id": monday.LocaleIdID,
		"th":    monday.LocaleThTH,
		"th_th": monday.LocaleThTH,
	}

	if loc, ok := localeMap[locale]; ok {
		return loc
	}

	// Try just the language part
	parts := strings.Split(locale, "_")
	if len(parts) > 1 {
		if loc, ok := localeMap[parts[0]]; ok {
			return loc
		}
	}

	return monday.LocaleEnUS // Default fallback
}

// getDateFormatForStyle returns the Go time format string for a given style and locale.
// Styles: "short" (numeric), "medium" (abbreviated), "long" (full month), "full" (with weekday)
func getDateFormatForStyle(style string, locale monday.Locale) string {
	switch style {
	case "short":
		// Numeric format - varies by locale
		switch locale {
		case monday.LocaleEnUS:
			return "1/2/06"
		case monday.LocaleEnGB:
			return "02/01/06"
		case monday.LocaleDeDE:
			return "02.01.06"
		case monday.LocaleFrFR, monday.LocaleFrCA:
			return "02/01/06"
		case monday.LocaleJaJP:
			return "06/01/02"
		case monday.LocaleZhCN, monday.LocaleZhTW:
			return "06/1/2"
		case monday.LocaleKoKR:
			return "06. 1. 2."
		default:
			return "02/01/06"
		}
	case "medium":
		// Abbreviated month - locale-aware order
		switch locale {
		case monday.LocaleEnUS:
			return "Jan 2, 2006"
		case monday.LocaleEnGB:
			return "2 Jan 2006"
		case monday.LocaleDeDE:
			return "2. Jan. 2006"
		case monday.LocaleFrFR, monday.LocaleFrCA:
			return "2 Jan 2006"
		case monday.LocaleEsES:
			return "2 Jan 2006"
		case monday.LocaleItIT:
			return "2 Jan 2006"
		case monday.LocaleJaJP:
			return "2006年1月2日"
		case monday.LocaleZhCN, monday.LocaleZhTW:
			return "2006年1月2日"
		case monday.LocaleKoKR:
			return "2006년 1월 2일"
		case monday.LocalePtBR:
			return "2 Jan 2006"
		case monday.LocaleRuRU:
			return "2 Jan 2006"
		case monday.LocaleNlNL, monday.LocaleNlBE:
			return "2 Jan 2006"
		default:
			return "2 Jan 2006"
		}
	case "long":
		// Full month name - locale-aware order
		switch locale {
		case monday.LocaleEnUS:
			return "January 2, 2006"
		case monday.LocaleEnGB:
			return "2 January 2006"
		case monday.LocaleDeDE:
			return "2. January 2006"
		case monday.LocaleFrFR, monday.LocaleFrCA:
			return "2 January 2006"
		case monday.LocaleEsES:
			return "2 de January de 2006"
		case monday.LocaleItIT:
			return "2 January 2006"
		case monday.LocaleJaJP:
			return "2006年1月2日"
		case monday.LocaleZhCN, monday.LocaleZhTW:
			return "2006年1月2日"
		case monday.LocaleKoKR:
			return "2006년 1월 2일"
		case monday.LocaleRuRU:
			return "2 January 2006"
		default:
			return "2 January 2006"
		}
	case "full":
		// With weekday - locale-aware
		switch locale {
		case monday.LocaleEnUS:
			return "Monday, January 2, 2006"
		case monday.LocaleEnGB:
			return "Monday, 2 January 2006"
		case monday.LocaleDeDE:
			return "Monday, 2. January 2006"
		case monday.LocaleFrFR, monday.LocaleFrCA:
			return "Monday 2 January 2006"
		case monday.LocaleEsES:
			return "Monday, 2 de January de 2006"
		case monday.LocaleJaJP:
			return "2006年1月2日 Monday"
		case monday.LocaleZhCN, monday.LocaleZhTW:
			return "2006年1月2日 Monday"
		case monday.LocaleKoKR:
			return "2006년 1월 2일 Monday"
		default:
			return "Monday, 2 January 2006"
		}
	default:
		return "January 2, 2006" // Default to long English
	}
}

// formatNumberWithLocale formats a number with the given locale
func formatNumberWithLocale(value float64, localeStr string) Object {
	tag, err := language.Parse(localeStr)
	if err != nil {
		return newLocaleError(localeStr)
	}
	p := message.NewPrinter(tag)
	return &String{Value: p.Sprintf("%v", number.Decimal(value))}
}

// formatCurrencyWithLocale formats a currency value with the given locale
func formatCurrencyWithLocale(value float64, currencyCode string, localeStr string) Object {
	cur, err := currency.ParseISO(currencyCode)
	if err != nil {
		return newValidationError("VAL-0001", map[string]any{"Code": currencyCode})
	}

	tag, err := language.Parse(localeStr)
	if err != nil {
		return newLocaleError(localeStr)
	}

	p := message.NewPrinter(tag)
	amount := cur.Amount(value)
	return &String{Value: p.Sprintf("%v", currency.Symbol(amount))}
}

// formatPercentWithLocale formats a percentage with the given locale
func formatPercentWithLocale(value float64, localeStr string) Object {
	tag, err := language.Parse(localeStr)
	if err != nil {
		return newLocaleError(localeStr)
	}
	p := message.NewPrinter(tag)
	return &String{Value: p.Sprintf("%v", number.Percent(value))}
}

// formatDateWithStyleAndLocale formats a datetime dictionary with the given style and locale
func formatDateWithStyleAndLocale(dict *Dictionary, style string, localeStr string, env *Environment) Object {
	// Extract time from datetime dictionary
	var t time.Time
	if unixExpr, ok := dict.Pairs["unix"]; ok {
		unixObj := Eval(unixExpr, NewEnvironment())
		if unixInt, ok := unixObj.(*Integer); ok {
			t = time.Unix(unixInt.Value, 0).UTC()
		}
	}

	// Validate style
	validStyles := map[string]bool{"short": true, "medium": true, "long": true, "full": true}
	if !validStyles[style] {
		return newValidationError("VAL-0002", map[string]any{"Style": style, "Context": "formatDate", "ValidOptions": "short, medium, long, full"})
	}

	// Map locale string to monday.Locale
	mondayLocale := getMondayLocale(localeStr)

	// Get format pattern for style
	format := getDateFormatForStyle(style, mondayLocale)

	return &String{Value: monday.Format(t, format, mondayLocale)}
}
