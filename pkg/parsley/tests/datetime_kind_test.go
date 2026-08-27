package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// testEvalKind evaluates a Parsley expression and returns the result
func testEvalKind(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

// TestDatetimeKindField tests that the kind field is correctly set
func TestDatetimeKindField(t *testing.T) {
	tests := []struct {
		input        string
		expectedKind string
	}{
		// Date-only literals
		{`@2024-12-25.kind`, "date"},
		{`@2024-01-01.kind`, "date"},
		{`@2024-06-15.kind`, "date"},
		// Datetime literals (with time component)
		{`@2024-12-25T14:30:00.kind`, "datetime"},
		{`@2024-01-01T00:00:00.kind`, "datetime"},
		{`@2024-06-15T23:59:59.kind`, "datetime"},
		// Datetime with timezone
		{`@2024-12-25T14:30:00Z.kind`, "datetime"},
		{`@2024-12-25T14:30:00-05:00.kind`, "datetime"},
		// datetime() builtin returns datetime
		{`datetime("2024-12-25").kind`, "datetime"},
		{`@now.kind`, "datetime"},
	}

	for _, tt := range tests {
		evaluated := testEvalKind(tt.input)
		if evaluated == nil {
			t.Errorf("For input '%s': got nil object", tt.input)
			continue
		}
		if err, ok := evaluated.(*evaluator.Error); ok {
			t.Errorf("For input '%s': got error: %s", tt.input, err.Message)
			continue
		}
		str, ok := evaluated.(*evaluator.String)
		if !ok {
			t.Errorf("For input '%s': expected String, got %T", tt.input, evaluated)
			continue
		}
		if str.Value != tt.expectedKind {
			t.Errorf("For input '%s': expected kind=%s, got kind=%s", tt.input, tt.expectedKind, str.Value)
		}
	}
}

// TestTimeOnlyLiterals tests time-only literal syntax @HH:MM and @HH:MM:SS
func TestTimeOnlyLiterals(t *testing.T) {
	tests := []struct {
		input        string
		expectedKind string
		hour         int64
		minute       int64
		second       int64
	}{
		{`@12:30`, "time", 12, 30, 0},
		{`@09:15`, "time", 9, 15, 0},
		{`@00:00`, "time", 0, 0, 0},
		{`@23:59`, "time", 23, 59, 0},
		{`@12:30:45`, "time_seconds", 12, 30, 45},
		{`@09:15:30`, "time_seconds", 9, 15, 30},
		{`@00:00:00`, "time_seconds", 0, 0, 0},
		{`@23:59:59`, "time_seconds", 23, 59, 59},
	}

	for _, tt := range tests {
		// Test kind
		kindInput := tt.input + ".kind"
		evaluated := testEvalKind(kindInput)
		if err, ok := evaluated.(*evaluator.Error); ok {
			t.Errorf("For input '%s': got error: %s", kindInput, err.Message)
			continue
		}
		if str, ok := evaluated.(*evaluator.String); ok {
			if str.Value != tt.expectedKind {
				t.Errorf("For input '%s': expected kind=%s, got kind=%s", tt.input, tt.expectedKind, str.Value)
			}
		}

		// Test hour
		hourInput := tt.input + ".hour"
		hourEval := testEvalKind(hourInput)
		if intObj, ok := hourEval.(*evaluator.Integer); ok {
			if intObj.Value != tt.hour {
				t.Errorf("For input '%s': expected hour=%d, got hour=%d", tt.input, tt.hour, intObj.Value)
			}
		}

		// Test minute
		minInput := tt.input + ".minute"
		minEval := testEvalKind(minInput)
		if intObj, ok := minEval.(*evaluator.Integer); ok {
			if intObj.Value != tt.minute {
				t.Errorf("For input '%s': expected minute=%d, got minute=%d", tt.input, tt.minute, intObj.Value)
			}
		}

		// Test second
		secInput := tt.input + ".second"
		secEval := testEvalKind(secInput)
		if intObj, ok := secEval.(*evaluator.Integer); ok {
			if intObj.Value != tt.second {
				t.Errorf("For input '%s': expected second=%d, got second=%d", tt.input, tt.second, intObj.Value)
			}
		}
	}
}

// TestTimeOnlyStringConversion tests that time-only literals convert back to time format
func TestTimeOnlyStringConversion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`toString(@12:30)`, "12:30"},
		{`toString(@09:15)`, "09:15"},
		{`toString(@00:00)`, "00:00"},
		{`toString(@23:59)`, "23:59"},
		{`toString(@12:30:45)`, "12:30:45"},
		{`toString(@09:15:30)`, "09:15:30"},
		{`toString(@00:00:00)`, "00:00:00"},
		{`toString(@23:59:59)`, "23:59:59"},
	}

	for _, tt := range tests {
		evaluated := testEvalKind(tt.input)
		if err, ok := evaluated.(*evaluator.Error); ok {
			t.Errorf("For input '%s': got error: %s", tt.input, err.Message)
			continue
		}
		str, ok := evaluated.(*evaluator.String)
		if !ok {
			t.Errorf("For input '%s': expected String, got %T", tt.input, evaluated)
			continue
		}
		if str.Value != tt.expected {
			t.Errorf("For input '%s': expected %s, got %s", tt.input, tt.expected, str.Value)
		}
	}
}

// TestDateStringConversion tests that date-only literals convert to date format
func TestDateStringConversion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`toString(@2024-12-25)`, "2024-12-25"},
		{`toString(@2024-01-01)`, "2024-01-01"},
		{`toString(@2024-06-15)`, "2024-06-15"},
	}

	for _, tt := range tests {
		evaluated := testEvalKind(tt.input)
		if err, ok := evaluated.(*evaluator.Error); ok {
			t.Errorf("For input '%s': got error: %s", tt.input, err.Message)
			continue
		}
		str, ok := evaluated.(*evaluator.String)
		if !ok {
			t.Errorf("For input '%s': expected String, got %T", tt.input, evaluated)
			continue
		}
		if str.Value != tt.expected {
			t.Errorf("For input '%s': expected %s, got %s", tt.input, tt.expected, str.Value)
		}
	}
}

// TestDatetimeStringConversion tests that datetime literals convert to ISO format
func TestDatetimeStringConversion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`toString(@2024-12-25T14:30:00)`, "2024-12-25T14:30:00Z"},
		{`toString(@2024-01-01T00:00:00)`, "2024-01-01T00:00:00Z"},
		{`toString(@2024-06-15T23:59:59)`, "2024-06-15T23:59:59Z"},
	}

	for _, tt := range tests {
		evaluated := testEvalKind(tt.input)
		if err, ok := evaluated.(*evaluator.Error); ok {
			t.Errorf("For input '%s': got error: %s", tt.input, err.Message)
			continue
		}
		str, ok := evaluated.(*evaluator.String)
		if !ok {
			t.Errorf("For input '%s': expected String, got %T", tt.input, evaluated)
			continue
		}
		if str.Value != tt.expected {
			t.Errorf("For input '%s': expected %s, got %s", tt.input, tt.expected, str.Value)
		}
	}
}

// TestDatetimeTemplateInterpolationMedium tests that datetime uses .medium() in template interpolation (FEAT-146)
func TestDatetimeTemplateInterpolationMedium(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"date_only", "let d = datetime(\"2025-06-15\"); `Date: {d}`", "Date: Jun 15, 2025"},
		{"full_datetime", "let d = datetime(\"2025-12-25T14:30:00Z\"); `Date: {d}`", "Date: Dec 25, 2025"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEvalKind(tt.input)
			if err, ok := evaluated.(*evaluator.Error); ok {
				t.Fatalf("For input '%s': got error: %s", tt.input, err.Message)
			}
			str, ok := evaluated.(*evaluator.String)
			if !ok {
				t.Fatalf("For input '%s': expected String, got %T", tt.input, evaluated)
			}
			if str.Value != tt.expected {
				t.Errorf("For input '%s': expected %q, got %q", tt.input, tt.expected, str.Value)
			}
		})
	}
}

// TestDatetimeToStringPreservesISO tests that toString() still returns ISO format (FEAT-146)
func TestDatetimeToStringPreservesISO(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`toString(@2024-12-25T14:30:00)`, "2024-12-25T14:30:00Z"},
		{`toString(datetime("2025-06-15"))`, "2025-06-15T00:00:00Z"},
	}

	for _, tt := range tests {
		evaluated := testEvalKind(tt.input)
		if err, ok := evaluated.(*evaluator.Error); ok {
			t.Errorf("For input '%s': got error: %s", tt.input, err.Message)
			continue
		}
		str, ok := evaluated.(*evaluator.String)
		if !ok {
			t.Errorf("For input '%s': expected String, got %T", tt.input, evaluated)
			continue
		}
		if str.Value != tt.expected {
			t.Errorf("For input '%s': expected %s, got %s", tt.input, tt.expected, str.Value)
		}
	}
}

// TestKindPreservationInArithmetic tests that kind is preserved after arithmetic operations
func TestKindPreservationInArithmetic(t *testing.T) {
	tests := []struct {
		input        string
		expectedKind string
	}{
		// Date arithmetic preserves date kind
		{`(@2024-12-25 + 86400).kind`, "date"}, // date + seconds
		{`(@2024-12-25 - 86400).kind`, "date"}, // date - seconds
		{`(86400 + @2024-12-25).kind`, "date"}, // seconds + date
		{`(@2024-12-25 + @1mo).kind`, "date"},  // date + duration (month)
		// Datetime arithmetic preserves datetime kind
		{`(@2024-12-25T14:30:00 + 3600).kind`, "datetime"}, // datetime + seconds
		{`(@2024-12-25T14:30:00 - 3600).kind`, "datetime"}, // datetime - seconds
		{`(3600 + @2024-12-25T14:30:00).kind`, "datetime"}, // seconds + datetime
		{`(@2024-12-25T14:30:00 + @1h).kind`, "datetime"},  // datetime + duration (hour)
		// Time arithmetic preserves time kind
		{`(@12:30 + 3600).kind`, "time"}, // time + seconds
		{`(@12:30 - 1800).kind`, "time"}, // time - seconds
		{`(@12:30 + @1h).kind`, "time"},  // time + duration (hour)
	}

	for _, tt := range tests {
		evaluated := testEvalKind(tt.input)
		if evaluated == nil {
			t.Errorf("For input '%s': got nil object", tt.input)
			continue
		}
		if err, ok := evaluated.(*evaluator.Error); ok {
			t.Errorf("For input '%s': got error: %s", tt.input, err.Message)
			continue
		}
		str, ok := evaluated.(*evaluator.String)
		if !ok {
			t.Errorf("For input '%s': expected String, got %T (%v)", tt.input, evaluated, evaluated)
			continue
		}
		if str.Value != tt.expectedKind {
			t.Errorf("For input '%s': expected kind=%s, got kind=%s", tt.input, tt.expectedKind, str.Value)
		}
	}
}

// TestTimeOnlyUsesCurrentDate tests that time-only literals use today's date internally
func TestTimeOnlyUsesCurrentDate(t *testing.T) {
	// Get today's date components from @now
	nowYear := testEvalKind(`@now.year`)
	nowMonth := testEvalKind(`@now.month`)
	nowDay := testEvalKind(`@now.day`)

	// Time-only literal should have the same date components
	timeYear := testEvalKind(`@12:30.year`)
	timeMonth := testEvalKind(`@12:30.month`)
	timeDay := testEvalKind(`@12:30.day`)

	// Compare - note we're comparing the .Value property directly
	nowYearVal := nowYear.(*evaluator.Integer).Value
	timeYearVal := timeYear.(*evaluator.Integer).Value
	if nowYearVal != timeYearVal {
		t.Errorf("Expected time literal year=%d (from @now), got %d", nowYearVal, timeYearVal)
	}

	nowMonthVal := nowMonth.(*evaluator.Integer).Value
	timeMonthVal := timeMonth.(*evaluator.Integer).Value
	if nowMonthVal != timeMonthVal {
		t.Errorf("Expected time literal month=%d (from @now), got %d", nowMonthVal, timeMonthVal)
	}

	nowDayVal := nowDay.(*evaluator.Integer).Value
	timeDayVal := timeDay.(*evaluator.Integer).Value
	if nowDayVal != timeDayVal {
		t.Errorf("Expected time literal day=%d (from @now), got %d", nowDayVal, timeDayVal)
	}
}

// TestTimeOnlyComparisons tests comparisons between time-only values
func TestTimeOnlyComparisons(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`@12:30 < @14:00`, true},
		{`@14:00 > @12:30`, true},
		{`@12:30 == @12:30`, true},
		{`@12:30 != @14:00`, true},
		{`@12:30 <= @12:30`, true},
		{`@12:30 >= @12:30`, true},
		{`@23:59 > @00:00`, true},
		{`@00:00 < @23:59`, true},
		{`@12:30:00 == @12:30:00`, true},
		{`@12:30:45 > @12:30:30`, true},
	}

	for _, tt := range tests {
		evaluated := testEvalKind(tt.input)
		if err, ok := evaluated.(*evaluator.Error); ok {
			t.Errorf("For input '%s': got error: %s", tt.input, err.Message)
			continue
		}
		boolVal, ok := evaluated.(*evaluator.Boolean)
		if !ok {
			t.Errorf("For input '%s': expected Boolean, got %T", tt.input, evaluated)
			continue
		}
		if boolVal.Value != tt.expected {
			t.Errorf("For input '%s': expected %v, got %v", tt.input, tt.expected, boolVal.Value)
		}
	}
}

// TestTimePrecisionPreservation tests that @12:30 vs @12:30:00 preserves precision
func TestTimePrecisionPreservation(t *testing.T) {
	// Without seconds - should output HH:MM
	noSec := testEvalKind(`toString(@12:30)`)
	if str, ok := noSec.(*evaluator.String); ok {
		if strings.Contains(str.Value, ":00:00") || len(str.Value) > 5 {
			t.Errorf("Expected @12:30 to format as '12:30', got '%s'", str.Value)
		}
	}

	// With seconds - should output HH:MM:SS
	withSec := testEvalKind(`toString(@12:30:00)`)
	if str, ok := withSec.(*evaluator.String); ok {
		if !strings.Contains(str.Value, ":00") || len(str.Value) < 8 {
			t.Errorf("Expected @12:30:00 to format as '12:30:00', got '%s'", str.Value)
		}
	}
}

// TestFormatRespectsKind guards BUG-045.
//
// formatDateWithStyleAndLocale had one code path — date patterns — for all four
// kinds, so a time-only value formatted as a bare date: @timeNow printed
// "Aug 26, 2026" for something whose entire content is 00:32. Since FEAT-146
// routed template interpolation through this renderer, `{t}` printed a date too.
//
// It got past the suite because the only test that formatted a datetime was
// FEAT-146's TestDatetimeTemplateInterpolationMedium, whose two cases are a
// date and a datetime — kinds that produce identical output here, so agreement
// between them looked like kind-awareness. Neither time kind was ever formatted.
// This covers all four.
func TestFormatRespectsKind(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"time renders a time", `@14:30 + ""`, "14:30"},
		{"time_seconds keeps its seconds", `@14:30:45 + ""`, "14:30:45"},
		{"time via fmt", `@14:30.fmt("medium")`, "14:30"},
		{"time via medium sugar", `@14:30.medium()`, "14:30"},
		{"time in interpolation", "let t = @14:30; `at {t}`", "at 14:30"},

		// Date and datetime are unchanged: FEAT-146 chose the date portion for
		// human-facing output deliberately, and that decision stands here.
		{"date is unaffected", `@2025-06-15.fmt("medium")`, "Jun 15, 2025"},
		{"datetime keeps FEAT-146 behaviour", `@2025-06-15T14:30:00Z.fmt("medium")`, "Jun 15, 2025"},
		{"date full style still works", `@2025-06-15.fmt("full")`, "Sunday, June 15, 2025"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			evaluated := testEvalKind(tc.input)
			if err, ok := evaluated.(*evaluator.Error); ok {
				t.Fatalf("%s: %s", tc.input, err.Message)
			}
			str, ok := evaluated.(*evaluator.String)
			if !ok {
				t.Fatalf("%s: expected String, got %T", tc.input, evaluated)
			}
			if str.Value != tc.want {
				t.Errorf("%s = %q, want %q", tc.input, str.Value, tc.want)
			}
		})
	}
}

// TestTimeSecondsIsAFirstClassKind guards BUG-046.
//
// time_seconds could be written as a literal and nothing else. time() hardcoded
// kind "time", so time("14:30:45") silently discarded the precision it was
// handed; the && intersection matched "time" alone, so `@2026-01-01 && @14:30:45`
// failed with "Unknown operator: date && time_seconds" while the identical
// expression with @14:30 worked; and there was no conversion in either
// direction.
func TestTimeSecondsIsAFirstClassKind(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		// @timeNow is the current moment, and a moment has a second in it.
		{"timeNow carries seconds", `@timeNow.kind`, "time_seconds"},

		// time() keeps the precision it is given rather than flattening it.
		{"parsed seconds are kept", `time("14:30:45").kind`, "time_seconds"},
		{"parsed minutes stay minutes", `time("14:30").kind`, "time"},
		{"12-hour with seconds", `time("3:45:20 PM").kind`, "time_seconds"},

		// Conversion, both ways.
		{"widen", `time(@14:30, {seconds: true}).kind`, "time_seconds"},
		{"widen renders", `time(@14:30, {seconds: true}) + ""`, "14:30:00"},
		{"narrow", `time(@14:30:45, {seconds: false}).kind`, "time"},
		{"narrow renders", `time(@14:30:45, {seconds: false}) + ""`, "14:30"},

		// A datetime carries a second, so its time portion keeps it.
		{"time of a datetime", `time(@2025-06-15T14:30:45Z).kind`, "time_seconds"},
		{"time of a datetime renders", `time(@2025-06-15T14:30:45Z) + ""`, "14:30:45"},

		// The intersection treats time_seconds as the time it is.
		{"date && time_seconds", `(@2026-01-01 && @14:30:45).kind`, "datetime"},
		{"date && time_seconds keeps the second", `(@2026-01-01 && @14:30:45).second`, "45"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			evaluated := testEvalKind(tc.input)
			if err, ok := evaluated.(*evaluator.Error); ok {
				t.Fatalf("%s: %s", tc.input, err.Message)
			}
			if got := evaluated.Inspect(); got != tc.want {
				t.Errorf("%s = %s, want %s", tc.input, got, tc.want)
			}
		})
	}
}
