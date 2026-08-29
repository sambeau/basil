// eval_locale.go - Locale and formatting helpers for the Parsley evaluator
//
// This file contains helper functions for locale-aware formatting of numbers,
// currencies, percentages, and dates. These functions handle mapping between
// locale strings and locale-specific formatting rules.

package evaluator

import (
	"math"
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

// formatNumberWithPrecisionAndLocale formats a number with exactly the given
// number of decimal places, in the given locale.
//
// The rounding is ours and the digits are the formatter's. This used to round
// correctly and then hand the rounded value to formatNumberWithLocale, whose
// number.Decimal caps fraction digits at three, and pad the truncated result
// out to the requested width with zeros — so 3.14159.fmt(4) rounded to 3.1416,
// printed "3.142", and padded to "3.1420". Every precision above 3 was wrong,
// silently, and the padding made it look deliberate rather than truncated.
// Asking the formatter for the width we want also gets the locale's decimal
// separator and grouping for free: the old padding carried its own
// three-locale table of separators and appended ASCII digits to a string it
// had not really parsed, so 1234.56789.fmt({precision: 4, locale: "de-DE"})
// came out "1.234,5680".
func formatNumberWithPrecisionAndLocale(value float64, precision int, locale string) Object {
	tag, err := language.Parse(locale)
	if err != nil {
		return newLocaleError(locale)
	}

	// Round half away from zero, which is what this function has always done
	// and what a reader expects of 2.5.fmt(0); x/text rounds half to even.
	multiplier := math.Pow(10, float64(precision))
	rounded := math.Round(value*multiplier) / multiplier

	p := message.NewPrinter(tag)
	return &String{Value: p.Sprintf("%v", number.Decimal(rounded,
		number.MinFractionDigits(precision),
		number.MaxFractionDigits(precision),
	))}
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

// datetimeDictWallClock rebuilds the wall clock a datetime dictionary holds.
//
// It reads the dictionary's own fields, the way every other datetime accessor
// does. This used to go via the unix timestamp — time.Unix(u, 0).UTC() — which
// re-derived the calendar fields in UTC and so printed a different day from
// the one the value reports. At 00:32 BST a value whose .day was 27, whose
// .weekday was "Thursday" and whose .iso was 2026-08-27T00:32:17+01:00
// formatted as "Wednesday, August 26, 2026". One hour a night in Britain, and
// up to thirteen hours a day in New Zealand (BUG-044).
//
// dictToTime reads the stored year/month/day/hour/minute/second, which ARE the
// value's wall clock. It labels them UTC, which does not matter to a caller
// that formats: formatting reads only those fields, never the zone.
func datetimeDictWallClock(dict *Dictionary, env *Environment) time.Time {
	if built, err := dictToTime(dict, env); err == nil {
		return built
	}
	if unixExpr, ok := dict.Pairs["unix"]; ok {
		// A dictionary without the calendar fields is not one this evaluator
		// built. Local, not UTC: it is the better guess of the two.
		unixObj := Eval(unixExpr, NewEnvironment())
		if unixInt, ok := unixObj.(*Integer); ok {
			return time.Unix(unixInt.Value, 0).Local()
		}
	}
	return time.Time{}
}

// formatDatePortionWithStyleAndLocale renders the date part of t alone, for the
// given style and locale. Callers that want a date and only a date — a record
// column declared `{format: "date"}`, say — use this directly.
func formatDatePortionWithStyleAndLocale(t time.Time, style string, localeStr string) string {
	mondayLocale := getMondayLocale(localeStr)
	return monday.Format(t, getDateFormatForStyle(style, mondayLocale), mondayLocale)
}

// formatDateWithStyleAndLocale formats a datetime dictionary with the given style and locale
func formatDateWithStyleAndLocale(dict *Dictionary, style string, localeStr string, env *Environment) Object {
	t := datetimeDictWallClock(dict, env)

	// Validate style
	validStyles := map[string]bool{"short": true, "medium": true, "long": true, "full": true}
	if !validStyles[style] {
		return newValidationError("VAL-0002", map[string]any{"Style": style, "Context": "formatDate", "ValidOptions": "short, medium, long, full"})
	}

	// Print what the value actually holds, and only that.
	//
	// This function had a single code path — date patterns — for all four
	// kinds, which meant @timeNow, a time-only value, formatted as a bare
	// date: "Aug 26, 2026" for something whose whole content is 00:32:17.
	// Since FEAT-146 routed template interpolation through here, `{t}` printed
	// a date too. datetimeDictToString has always switched on kind correctly;
	// this is the same switch, for the styled renderer (BUG-045).
	//
	// Times are 24-hour and locale-independent, matching the .time property and
	// datetimeDictToString. Whether en-US should get "12:32 AM" is a separate
	// question from whether a time should print as a date.
	switch getDictString(dict, "kind", env) {
	case "time":
		return &String{Value: t.Format("15:04")}
	case "time_seconds":
		return &String{Value: t.Format("15:04:05")}
	case "date":
		return &String{Value: formatDatePortionWithStyleAndLocale(t, style, localeStr)}
	}

	// A datetime, which BUG-045 left printing as a bare date: every style gave
	// @2024-12-25T14:30:00 the same "Dec 25, 2024" as the date-only value, so
	// there was no way to display a datetime's time through .fmt() at all, and
	// a page showing an event time silently showed only its day. That was the
	// open design question BUG-045 recorded; this settles it the way the rest
	// of the file already leans — a renderer prints what the value holds.
	//
	// A datetime has one kind for three precisions, so the rule is the value's
	// own content: print the time to the precision it carries, and not at all
	// when it carries none. That mirrors the time/time_seconds split above and
	// keeps the common case quiet — datetime("2025-06-15"), a DB date column
	// and a JSON date all widen a date into a datetime at exact midnight, and
	// hanging ", 00:00" off every one of them would put a meaningless clock on
	// most pages. Nothing is lost either way: a value at midnight prints what a
	// human would write for it, and a value carrying seconds keeps them.
	//
	// The connector is a comma in every locale and every style. English "at"
	// reads better in long and full, but then every locale needs its own word,
	// and a table of thirty guessed translations is worse than the comma that
	// nearly all of them use for the shorter styles anyway.
	formatted := formatDatePortionWithStyleAndLocale(t, style, localeStr)
	switch {
	case t.Second() != 0:
		formatted += ", " + t.Format("15:04:05")
	case t.Hour() != 0 || t.Minute() != 0:
		formatted += ", " + t.Format("15:04")
	}
	return &String{Value: formatted}
}
