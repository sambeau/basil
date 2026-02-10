package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func testDurationCode(code string) (evaluator.Object, bool) {
	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		return &evaluator.String{Value: p.Errors()[0]}, true
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)
	if result != nil && result.Type() == evaluator.ERROR_OBJ {
		return result, true
	}

	return result, false
}

func TestDurationLiterals(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "simple seconds duration",
			code:     `let d = @30s; d.seconds`,
			expected: "30",
		},
		{
			name:     "simple minutes duration",
			code:     `let d = @5m; d.seconds`,
			expected: "300", // 5 * 60
		},
		{
			name:     "simple hours duration",
			code:     `let d = @2h; d.seconds`,
			expected: "7200", // 2 * 3600
		},
		{
			name:     "simple days duration",
			code:     `let d = @7d; d.seconds`,
			expected: "604800", // 7 * 86400
		},
		{
			name:     "simple weeks duration",
			code:     `let d = @2w; d.seconds`,
			expected: "1209600", // 2 * 7 * 86400
		},
		{
			name:     "simple months duration",
			code:     `let d = @6mo; d.months`,
			expected: "6",
		},
		{
			name:     "simple years duration",
			code:     `let d = @1y; d.months`,
			expected: "12",
		},
		{
			name:     "compound duration hours and minutes",
			code:     `let d = @2h30m; d.seconds`,
			expected: "9000", // 2*3600 + 30*60
		},
		{
			name:     "compound duration days and hours",
			code:     `let d = @3d12h; d.seconds`,
			expected: "302400", // 3*86400 + 12*3600
		},
		{
			name:     "compound duration with all units",
			code:     `let d = @1y2mo3w4d5h6m7s; d.months`,
			expected: "14", // 1*12 + 2
		},
		{
			name:     "compound duration seconds calculation",
			code:     `let d = @1y2mo3w4d5h6m7s; d.seconds`,
			expected: "2178367", // 3*604800 + 4*86400 + 5*3600 + 6*60 + 7
		},
		{
			name:     "totalSeconds exists for pure seconds durations",
			code:     `let d = @2h30m; d.totalSeconds`,
			expected: "9000",
		},
		{
			name:     "totalSeconds is null for month-based durations",
			code:     `let d = @1y; d.totalSeconds`,
			expected: "null",
		},
		{
			name:     "duration type field",
			code:     `let d = @1h; d.__type`,
			expected: "duration",
		},
		// Computed properties
		{
			name:     "computed days property",
			code:     `let d = @2d12h; d.days`,
			expected: "2", // 216000 / 86400 = 2
		},
		{
			name:     "computed hours property",
			code:     `let d = @2d12h; d.hours`,
			expected: "60", // 216000 / 3600 = 60
		},
		{
			name:     "computed minutes property",
			code:     `let d = @2h30m; d.minutes`,
			expected: "150", // 9000 / 60 = 150
		},
		{
			name:     "computed days for week",
			code:     `let d = @1w; d.days`,
			expected: "7",
		},
		{
			name:     "computed hours for week",
			code:     `let d = @1w; d.hours`,
			expected: "168", // 7 * 24
		},
		{
			name:     "computed properties null for month-based durations",
			code:     `let d = @1y; d.days`,
			expected: "null",
		},
		{
			name:     "computed hours null for month-based durations",
			code:     `let d = @1mo; d.hours`,
			expected: "null",
		},
		{
			name:     "computed minutes null for mixed durations",
			code:     `let d = @1y2d; d.minutes`,
			expected: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, hasErr := testDurationCode(tt.code)
			if hasErr {
				t.Fatalf("testDurationCode() unexpected error: %v", result)
			}
			if result.Inspect() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestDurationArithmetic(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "add two durations (seconds only)",
			code:     `let d = @2h + @30m; d.seconds`,
			expected: "9000",
		},
		{
			name:     "subtract two durations (seconds only)",
			code:     `let d = @1d - @6h; d.seconds`,
			expected: "64800", // 86400 - 21600
		},
		{
			name:     "add durations with months",
			code:     `let d = @1y + @6mo; d.months`,
			expected: "18",
		},
		{
			name:     "subtract durations with months",
			code:     `let d = @2y - @3mo; d.months`,
			expected: "21", // 24 - 3
		},
		{
			name:     "add mixed durations (months and seconds)",
			code:     `let d1 = @1y2mo; let d2 = @3d4h; let d = d1 + d2; d.months`,
			expected: "14",
		},
		{
			name:     "add mixed durations seconds part",
			code:     `let d1 = @1y2mo; let d2 = @3d4h; let d = d1 + d2; d.seconds`,
			expected: "273600", // 3*86400 + 4*3600
		},
		{
			name:     "multiply duration by integer",
			code:     `let d = @2h * 3; d.seconds`,
			expected: "21600", // 7200 * 3
		},
		{
			name:     "multiply integer by duration (commutative)",
			code:     `let d = 3 * @2h; d.seconds`,
			expected: "21600", // 3 * 7200
		},
		{
			name:     "multiply integer by simple duration",
			code:     `let d = 3 * @1d; d.seconds`,
			expected: "259200", // 3 * 86400
		},
		{
			name:     "multiply integer by compound duration",
			code:     `let d = 2 * @2h30m; d.seconds`,
			expected: "18000", // 2 * 9000
		},
		{
			name:     "multiply negative integer by duration",
			code:     `let d = -2 * @1d; d.seconds`,
			expected: "-172800", // -2 * 86400
		},
		{
			name:     "commutativity: duration * int equals int * duration",
			code:     `(@2h * 3) == (3 * @2h)`,
			expected: "true",
		},
		{
			name:     "divide duration by integer",
			code:     `let d = @1d / 2; d.seconds`,
			expected: "43200", // 86400 / 2
		},
		{
			name:     "multiply month-based duration",
			code:     `let d = @1y * 2; d.months`,
			expected: "24",
		},
		{
			name:     "divide month-based duration",
			code:     `let d = @2y / 4; d.months`,
			expected: "6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, hasErr := testDurationCode(tt.code)
			if hasErr {
				t.Fatalf("testDurationCode() unexpected error: %v", result)
			}
			if result.Inspect() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestDatetimeDurationOperations(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "add duration to datetime",
			code:     `let dt = datetime("2024-01-15T00:00:00Z") + @2d; dt.day`,
			expected: "17",
		},
		{
			name:     "add hours to datetime",
			code:     `let dt = datetime("2024-01-15T10:00:00Z") + @3h; dt.hour`,
			expected: "13",
		},
		{
			name:     "subtract duration from datetime",
			code:     `let dt = datetime("2024-01-15T00:00:00Z") - @2d; dt.day`,
			expected: "13",
		},
		{
			name:     "add months to datetime",
			code:     `let dt = datetime("2024-01-31T00:00:00Z") + @1mo; dt.month`,
			expected: "3", // Jan 31 + 1 month = Mar 2 (Feb 31 doesn't exist)
		},
		{
			name:     "add months to datetime (day normalization)",
			code:     `let dt = datetime("2024-01-31T00:00:00Z") + @1mo; dt.day`,
			expected: "2", // Normalized from Feb 31 to Mar 2
		},
		{
			name:     "add years to datetime",
			code:     `let dt = datetime("2024-06-15T00:00:00Z") + @1y; dt.year`,
			expected: "2025",
		},
		{
			name:     "duration plus datetime is commutative",
			code:     `let dt = @2d + datetime("2024-01-15T00:00:00Z"); dt.day`,
			expected: "17",
		},
		{
			name:     "duration plus datetime equals datetime plus duration",
			code:     `let d = @1y2mo; let dt = datetime("2024-01-01T00:00:00Z"); (d + dt) == (dt + d)`,
			expected: "true",
		},
		{
			name:     "add compound duration to datetime",
			code:     `let dt = datetime("2024-01-01T00:00:00Z") + @1y2mo3d; dt.month`,
			expected: "3", // Jan + 14 months = March of next year
		},
		{
			name:     "datetime minus datetime returns duration",
			code:     `let diff = datetime("2024-01-20T00:00:00Z") - datetime("2024-01-15T00:00:00Z"); diff.seconds`,
			expected: "432000", // 5 days
		},
		{
			name:     "datetime minus datetime has no months",
			code:     `let diff = datetime("2024-01-20T00:00:00Z") - datetime("2024-01-15T00:00:00Z"); diff.months`,
			expected: "0",
		},
		{
			name:     "datetime minus datetime totalSeconds",
			code:     `let diff = datetime("2024-01-20T00:00:00Z") - datetime("2024-01-15T00:00:00Z"); diff.totalSeconds`,
			expected: "432000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, hasErr := testDurationCode(tt.code)
			if hasErr {
				t.Fatalf("testDurationCode() unexpected error: %v", result)
			}
			if result.Inspect() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestDurationComparison(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "compare equal durations",
			code:     `@2h == @7200s`,
			expected: "true",
		},
		{
			name:     "compare unequal durations",
			code:     `@2h != @3h`,
			expected: "true",
		},
		{
			name:     "less than comparison",
			code:     `@1h < @2h`,
			expected: "true",
		},
		{
			name:     "greater than comparison",
			code:     `@3h > @2h`,
			expected: "true",
		},
		{
			name:     "less than or equal",
			code:     `@2h <= @2h`,
			expected: "true",
		},
		{
			name:     "greater than or equal",
			code:     `@3h >= @2h`,
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, hasErr := testDurationCode(tt.code)
			if hasErr {
				t.Fatalf("testDurationCode() unexpected error: %v", result)
			}
			if result.Inspect() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestDurationDivision(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "divide seconds-only durations",
			code:     `@7d / @1d`,
			expected: "7",
		},
		{
			name:     "divide hours by hours",
			code:     `@6h / @2h`,
			expected: "3",
		},
		{
			name:     "divide mixed units",
			code:     `@2d / @12h`,
			expected: "4",
		},
		{
			name:     "divide with decimal result",
			code:     `@1d / @3d`,
			expected: "0.3333333333333333",
		},
		{
			name:     "divide year by year",
			code:     `@1y / @1y`,
			expected: "1",
		},
		{
			name:     "divide years",
			code:     `@2y / @1y`,
			expected: "2",
		},
		{
			name:     "divide months by year",
			code:     `@6mo / @1y`,
			expected: "0.5",
		},
		{
			name:     "divide months by months",
			code:     `@3mo / @1mo`,
			expected: "3",
		},
		{
			name:     "age calculation example",
			code:     `let birthdate = @1990-05-15; let today = datetime("2024-12-15T00:00:00Z"); (today - birthdate) / @1y`,
			expected: "34.58797921928582",
		},
		{
			name:     "mixed seconds and months",
			code:     `(@1y + @30d) / @1y`,
			expected: "1.0821372102096551",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, hasErr := testDurationCode(tt.code)
			if hasErr {
				t.Fatalf("testDurationCode() unexpected error: %v", result)
			}
			if result.Inspect() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestDurationErrors(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		expectError bool
	}{
		{
			name:        "cannot compare durations with months",
			code:        `@1y < @12mo`,
			expectError: true,
		},
		{
			name:        "cannot subtract datetime from duration",
			code:        `@1d - datetime("2024-01-01T00:00:00Z")`,
			expectError: true,
		},
		{
			name:        "division by zero",
			code:        `@1d / 0`,
			expectError: true,
		},
		{
			name:        "division by zero duration",
			code:        `@1d / @0s`,
			expectError: true,
		},
		{
			name:        "integer divided by duration is not supported",
			code:        `3 / @1d`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, hasErr := testDurationCode(tt.code)
			if tt.expectError && !hasErr {
				t.Errorf("Expected error but got result: %s", result.Inspect())
			}
			if !tt.expectError && hasErr {
				t.Errorf("Unexpected error: %v", result)
			}
		})
	}
}

func TestDurationConstructor(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		// String parsing
		{
			name:     "constructor from string - seconds",
			code:     `duration("30s").seconds`,
			expected: "30",
		},
		{
			name:     "constructor from string - minutes",
			code:     `duration("5m").seconds`,
			expected: "300",
		},
		{
			name:     "constructor from string - hours",
			code:     `duration("2h").seconds`,
			expected: "7200",
		},
		{
			name:     "constructor from string - days",
			code:     `duration("1d").seconds`,
			expected: "86400",
		},
		{
			name:     "constructor from string - weeks",
			code:     `duration("1w").seconds`,
			expected: "604800",
		},
		{
			name:     "constructor from string - months",
			code:     `duration("6mo").months`,
			expected: "6",
		},
		{
			name:     "constructor from string - years",
			code:     `duration("2y").months`,
			expected: "24",
		},
		{
			name:     "constructor from string - mixed",
			code:     `duration("1d2h30m").seconds`,
			expected: "95400", // 86400 + 7200 + 1800
		},
		{
			name:     "constructor from string - negative",
			code:     `duration("-1d").seconds`,
			expected: "-86400",
		},
		// Dictionary input
		{
			name:     "constructor from dict - seconds",
			code:     `duration({seconds: 30}).seconds`,
			expected: "30",
		},
		{
			name:     "constructor from dict - minutes",
			code:     `duration({minutes: 5}).seconds`,
			expected: "300",
		},
		{
			name:     "constructor from dict - hours",
			code:     `duration({hours: 2}).seconds`,
			expected: "7200",
		},
		{
			name:     "constructor from dict - days",
			code:     `duration({days: 1}).seconds`,
			expected: "86400",
		},
		{
			name:     "constructor from dict - weeks",
			code:     `duration({weeks: 1}).seconds`,
			expected: "604800",
		},
		{
			name:     "constructor from dict - months",
			code:     `duration({months: 6}).months`,
			expected: "6",
		},
		{
			name:     "constructor from dict - years",
			code:     `duration({years: 2}).months`,
			expected: "24",
		},
		{
			name:     "constructor from dict - mixed time",
			code:     `duration({days: 1, hours: 2, minutes: 30}).seconds`,
			expected: "95400",
		},
		{
			name:     "constructor from dict - mixed calendar",
			code:     `duration({years: 1, months: 6}).months`,
			expected: "18",
		},
		// Format method works
		{
			name:     "format method on constructor result",
			code:     `duration("2h30m").format()`,
			expected: "in 2 hours",
		},
		{
			name:     "format method on dict constructor",
			code:     `duration({hours: 2, minutes: 30}).format()`,
			expected: "in 2 hours",
		},
		// Arithmetic with constructed durations
		{
			name:     "add constructed durations",
			code:     `(duration("1d") + duration("2h")).seconds`,
			expected: "93600", // 86400 + 7200
		},
		{
			name:     "constructed equals literal",
			code:     `duration("1d2h30m") == @1d2h30m`,
			expected: "true",
		},
		{
			name:     "dict constructed equals literal",
			code:     `duration({days: 1, hours: 2, minutes: 30}) == @1d2h30m`,
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, hasErr := testDurationCode(tt.code)
			if hasErr {
				t.Fatalf("unexpected error: %v", result)
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestDurationConstructorErrors(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{
			name: "invalid string format",
			code: `duration("invalid")`,
		},
		{
			name: "wrong argument type - number",
			code: `duration(123)`,
		},
		{
			name: "wrong argument type - array",
			code: `duration([1, 2, 3])`,
		},
		{
			name: "no arguments",
			code: `duration()`,
		},
		{
			name: "too many arguments",
			code: `duration("1d", "2h")`,
		},
		{
			name: "dict with non-integer value",
			code: `duration({days: "one"})`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, hasErr := testDurationCode(tt.code)
			if !hasErr {
				t.Errorf("expected error for %s", tt.code)
			}
		})
	}
}
