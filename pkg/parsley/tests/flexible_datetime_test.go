package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// Helper function to evaluate Parsley code and return the result
func evalFlexibleDatetime(input string) (evaluator.Object, *evaluator.Error) {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		return nil, &evaluator.Error{Message: strings.Join(p.Errors(), "; ")}
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if err, ok := result.(*evaluator.Error); ok {
		return nil, err
	}

	return result, nil
}

// TestDateFunctionBasic tests the date() function with various formats
func TestDateFunctionBasic(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		checkKey string
		expected int64
	}{
		{
			name:     "date with natural language format",
			code:     `date("22 April 2005").day`,
			checkKey: "day",
			expected: 22,
		},
		{
			name:     "date with ISO format",
			code:     `date("2005-04-22").year`,
			checkKey: "year",
			expected: 2005,
		},
		{
			name:     "date with US format",
			code:     `date("04/22/2005").month`,
			checkKey: "month",
			expected: 4,
		},
		{
			name:     "date with full month name",
			code:     `date("December 25, 2024").day`,
			checkKey: "day",
			expected: 25,
		},
		{
			name:     "date with abbreviated month",
			code:     `date("Dec 25, 2024").month`,
			checkKey: "month",
			expected: 12,
		},
		{
			name:     "date with day-first natural format",
			code:     `date("25 December 2024").day`,
			checkKey: "day",
			expected: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalFlexibleDatetime(tt.code)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			intResult, ok := result.(*evaluator.Integer)
			if !ok {
				t.Fatalf("expected Integer, got %T (%v)", result, result)
			}

			if intResult.Value != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, intResult.Value)
			}
		})
	}
}

// TestDateFunctionLocale tests the date() function with locale options
func TestDateFunctionLocale(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int64
	}{
		{
			name:     "US locale (MM/DD/YYYY) - month first",
			code:     `date("01/02/2005").month`,
			expected: 1, // January
		},
		{
			name:     "US locale explicit - month first",
			code:     `date("01/02/2005", {locale: "en-US"}).month`,
			expected: 1, // January
		},
		{
			name:     "UK locale (DD/MM/YYYY) - day first",
			code:     `date("01/02/2005", {locale: "en-GB"}).month`,
			expected: 2, // February
		},
		{
			name:     "French month name",
			code:     `date("22 avril 2005", {locale: "fr-FR"}).month`,
			expected: 4, // April
		},
		{
			name:     "German month name",
			code:     `date("22 März 2005", {locale: "de-DE"}).month`,
			expected: 3, // March
		},
		{
			name:     "Spanish month name",
			code:     `date("22 diciembre 2024", {locale: "es-ES"}).month`,
			expected: 12, // December
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalFlexibleDatetime(tt.code)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			intResult, ok := result.(*evaluator.Integer)
			if !ok {
				t.Fatalf("expected Integer, got %T (%v)", result, result)
			}

			if intResult.Value != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, intResult.Value)
			}
		})
	}
}

// TestTimeFunctionBasic tests the time() function with various formats
func TestTimeFunctionBasic(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int64
	}{
		{
			name:     "time with 24-hour format",
			code:     `time("15:45").hour`,
			expected: 15,
		},
		{
			name:     "time with PM",
			code:     `time("3:45 PM").hour`,
			expected: 15,
		},
		{
			name:     "time with am",
			code:     `time("10:30 am").hour`,
			expected: 10,
		},
		{
			name:     "time with seconds",
			code:     `time("15:45:30").second`,
			expected: 30,
		},
		{
			name:     "time minute extraction",
			code:     `time("3:45 PM").minute`,
			expected: 45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalFlexibleDatetime(tt.code)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			intResult, ok := result.(*evaluator.Integer)
			if !ok {
				t.Fatalf("expected Integer, got %T (%v)", result, result)
			}

			if intResult.Value != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, intResult.Value)
			}
		})
	}
}

// TestDatetimeFunctionBasic tests the datetime() function with various formats
func TestDatetimeFunctionBasic(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		checkKey string
		expected int64
	}{
		{
			name:     "datetime with natural language",
			code:     `datetime("April 22, 2005 3:45 PM").hour`,
			checkKey: "hour",
			expected: 15,
		},
		{
			name:     "datetime with ISO 8601",
			code:     `datetime("2005-04-22T15:45:00Z").minute`,
			checkKey: "minute",
			expected: 45,
		},
		{
			name:     "datetime from unix timestamp",
			code:     `datetime(1704110400).year`,
			checkKey: "year",
			expected: 2024,
		},
		{
			name:     "datetime from dictionary",
			code:     `datetime({year: 2024, month: 7, day: 4}).month`,
			checkKey: "month",
			expected: 7,
		},
		{
			name:     "datetime with human-readable format",
			code:     `datetime("May 8, 2009 5:57:51 PM").day`,
			checkKey: "day",
			expected: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalFlexibleDatetime(tt.code)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			intResult, ok := result.(*evaluator.Integer)
			if !ok {
				t.Fatalf("expected Integer, got %T (%v)", result, result)
			}

			if intResult.Value != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, intResult.Value)
			}
		})
	}
}

// TestDatetimeFunctionLocale tests datetime() with locale options
func TestDatetimeFunctionLocale(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int64
	}{
		{
			name:     "datetime with UK locale",
			code:     `datetime("01/02/2005 3pm", {locale: "en-GB"}).month`,
			expected: 2, // February
		},
		{
			name:     "datetime with French locale",
			code:     `datetime("22 avril 2005", {locale: "fr-FR"}).month`,
			expected: 4, // April
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalFlexibleDatetime(tt.code)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			intResult, ok := result.(*evaluator.Integer)
			if !ok {
				t.Fatalf("expected Integer, got %T (%v)", result, result)
			}

			if intResult.Value != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, intResult.Value)
			}
		})
	}
}

// TestDatetimeKindField tests that the kind field is correctly set
func TestDatetimeKindFieldFlexible(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "date function sets kind to date",
			code:     `date("2024-12-25").kind`,
			expected: "date",
		},
		{
			name:     "time function sets kind to time",
			code:     `time("15:45").kind`,
			expected: "time",
		},
		{
			name:     "datetime function sets kind to datetime",
			code:     `datetime("2024-12-25T15:45:00Z").kind`,
			expected: "datetime",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalFlexibleDatetime(tt.code)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			strResult, ok := result.(*evaluator.String)
			if !ok {
				t.Fatalf("expected String, got %T (%v)", result, result)
			}

			if strResult.Value != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, strResult.Value)
			}
		})
	}
}

// TestDatetimeErrorHandling tests error cases
func TestDatetimeErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		errPart string
	}{
		{
			name:    "date with invalid input",
			code:    `date("not a date")`,
			errPart: "Cannot parse",
		},
		{
			name:    "time with invalid input",
			code:    `time("not a time")`,
			errPart: "Cannot parse",
		},
		{
			name:    "datetime with invalid input",
			code:    `datetime("garbage")`,
			errPart: "Cannot parse",
		},
		{
			name:    "date with wrong type",
			code:    `date(123)`,
			errPart: "must be a string",
		},
		{
			name:    "time with wrong type",
			code:    `time(true)`,
			errPart: "must be a string",
		},
		{
			name:    "date with too many arguments",
			code:    `date("2024-01-01", {}, "extra")`,
			errPart: "expects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := evalFlexibleDatetime(tt.code)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			errStr := err.Message
			if !strings.Contains(strings.ToLower(errStr), strings.ToLower(tt.errPart)) {
				t.Errorf("error %q does not contain %q", errStr, tt.errPart)
			}
		})
	}
}

// TestDatetimeTimezone tests timezone option
func TestDatetimeTimezone(t *testing.T) {
	// Basic timezone test - just ensure it doesn't error
	code := `datetime("2024-01-01 12:00", {timezone: "America/New_York"}).hour`
	result, err := evalFlexibleDatetime(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T (%v)", result, result)
	}
}

// TestOriginalDateFormats tests that originally problematic date format now works
func TestOriginalDateFormats(t *testing.T) {
	// This was the original user request - "22 April 2005" should parse
	tests := []struct {
		name string
		code string
	}{
		{
			name: "22 April 2005 - the original failing format",
			code: `date("22 April 2005").year`,
		},
		{
			name: "1 July 2013 - day month year",
			code: `date("1 July 2013").month`,
		},
		{
			name: "October 7th, 1970",
			code: `date("October 7th, 1970").day`,
		},
		{
			name: "oct 7, 1970 - abbreviated",
			code: `date("oct 7, 1970").month`,
		},
		{
			name: "12 Feb 2006, 19:17 - with time",
			code: `datetime("12 Feb 2006, 19:17").hour`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := evalFlexibleDatetime(tt.code)
			if err != nil {
				t.Fatalf("unexpected error parsing %s: %v", tt.name, err)
			}
		})
	}
}
