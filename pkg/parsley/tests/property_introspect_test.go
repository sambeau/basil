package tests

import (
	"strings"
	"testing"
)

func TestPropertyIntrospection(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		contains []string
	}{
		{
			name: "money properties in inspect",
			code: `inspect($10.00)`,
			contains: []string{
				"properties",
				"amount",
				"currency",
				"scale",
			},
		},
		{
			name: "money properties in describe",
			code: `describe($10.00)`,
			contains: []string{
				"Properties:",
				".amount",
				".currency",
				".scale",
				"Amount in smallest currency unit",
			},
		},
		{
			name: "datetime properties in inspect",
			code: `inspect(@2024-12-25)`,
			contains: []string{
				"properties",
				"year",
				"month",
				"day",
				"hour",
				"minute",
				"second",
				"weekday",
				"unix",
				"iso",
				"kind",
				"date",
				"time",
				"dayOfYear",
				"week",
				"timestamp",
			},
		},
		{
			name: "datetime properties in describe",
			code: `describe(@2024-12-25T14:30:00)`,
			contains: []string{
				"Properties:",
				".year",
				".month",
				".day",
				".hour",
				".minute",
				".second",
				".date",
				".time",
				"Day of month",
				"ISO 8601 datetime string",
			},
		},
		{
			name: "duration properties in inspect",
			code: `inspect(@1d)`,
			contains: []string{
				"properties",
				"months",
				"seconds",
				"totalSeconds",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalHelper(tt.code)

			resultStr := result.Inspect()
			for _, want := range tt.contains {
				if !strings.Contains(resultStr, want) {
					t.Errorf("Expected output to contain %q, got:\n%s", want, resultStr)
				}
			}
		})
	}
}

func TestPropertyTypesAreCorrect(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "money has amount property",
			code: `let info = inspect($1.00); info["properties"].length() > 0`,
			want: "true",
		},
		{
			name: "datetime has year property",
			code: `let info = inspect(@2024-12-25); info["properties"].length() > 10`,
			want: "true",
		},
		{
			name: "first money property has name",
			code: `let info = inspect($1.00); info["properties"][0]["name"] == "amount"`,
			want: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalHelper(tt.code)

			if result.Inspect() != tt.want {
				t.Errorf("Expected %s, got %s", tt.want, result.Inspect())
			}
		})
	}
}

func TestDescribeShowsPropertiesBeforeMethods(t *testing.T) {
	code := `describe($1.00)`
	result := testEvalHelper(code)

	output := result.Inspect()

	// Properties should come before Methods in the output
	propsIndex := strings.Index(output, "Properties:")
	methodsIndex := strings.Index(output, "Methods:")

	if propsIndex == -1 {
		t.Error("Expected to find 'Properties:' in output")
	}
	if methodsIndex == -1 {
		t.Error("Expected to find 'Methods:' in output")
	}
	if propsIndex >= methodsIndex {
		t.Error("Expected Properties to appear before Methods in describe() output")
	}
}
