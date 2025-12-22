package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// testEvalSort evaluates a Parsley expression and returns the result as a string
func testEvalSort(input string) evaluator.Object {
	return testEvalHelper(input)
}

// TestNaturalSortBasic tests basic natural sort ordering
func TestNaturalSortBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "file names with numbers",
			input:    `["file10", "file2", "file1"].sort().join(",")`,
			expected: "file1,file2,file10",
		},
		{
			name:     "version numbers",
			input:    `["v2.0", "v10.0", "v1.5"].sort().join(",")`,
			expected: "v1.5,v2.0,v10.0",
		},
		{
			name:     "image files",
			input:    `["img12.png", "img2.png", "img1.png"].sort().join(",")`,
			expected: "img1.png,img2.png,img12.png",
		},
		{
			name:     "chapter numbers",
			input:    `["chapter9", "chapter10", "chapter11", "chapter1"].sort().join(",")`,
			expected: "chapter1,chapter9,chapter10,chapter11",
		},
		{
			name:     "mixed alphanumeric",
			input:    `["a2b", "a10b", "a1b"].sort().join(",")`,
			expected: "a1b,a2b,a10b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalSort(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestNaturalSortLeadingZeros tests that leading zeros are handled correctly
func TestNaturalSortLeadingZeros(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "leading zeros come first",
			input:    `["a01", "a1", "a001"].sort().join(",")`,
			expected: "a001,a01,a1",
		},
		{
			name:     "mixed leading zeros",
			input:    `["test01", "test1", "test001", "test10", "test010"].sort().join(",")`,
			expected: "test001,test01,test010,test1,test10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalSort(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestNaturalSortStringsOnly tests sorting of pure strings
func TestNaturalSortStringsOnly(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "alphabetical strings",
			input:    `["banana", "apple", "cherry"].sort().join(",")`,
			expected: "apple,banana,cherry",
		},
		{
			name:     "case sensitivity",
			input:    `["Banana", "apple", "Cherry"].sort().join(",")`,
			expected: "Banana,Cherry,apple",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalSort(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestNaturalSortNumbers tests sorting of number arrays
func TestNaturalSortNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "integers",
			input:    `[10, 2, 1, 20, 3].sort().join(",")`,
			expected: "1,2,3,10,20",
		},
		{
			name:     "floats",
			input:    `[1.5, 1.2, 2.0, 1.10].sort().join(",")`,
			expected: "1.1,1.2,1.5,2",
		},
		{
			name:     "negative numbers",
			input:    `[5, -3, 0, -10, 2].sort().join(",")`,
			expected: "-10,-3,0,2,5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalSort(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestLexicographicSort tests the {natural: false} option
func TestLexicographicSort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lexicographic file sort",
			input:    `["file10", "file2", "file1"].sort({natural: false}).join(",")`,
			expected: "file1,file10,file2",
		},
		{
			name:     "lexicographic version sort",
			input:    `["v2.0", "v10.0", "v1.5"].sort({natural: false}).join(",")`,
			expected: "v1.5,v10.0,v2.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalSort(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestMixedTypeSort tests sorting arrays with mixed types
func TestMixedTypeSort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "null numbers strings",
			input:    `[null, 3, 1, "b", "a", 2].sort().map(fn(x) { if x == null { "null" } else { toString(x) } }).join(",")`,
			expected: "null,1,2,3,a,b",
		},
		{
			name:     "strings and numbers",
			input:    `["z", 1, "a", 2].sort().map(fn(x) { toString(x) }).join(",")`,
			expected: "1,2,a,z",
		},
		{
			name:     "booleans after strings",
			input:    `["a", true, false, "b"].sort().map(fn(x) { toString(x) }).join(",")`,
			expected: "a,b,false,true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalSort(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestStringComparisonOperators tests string comparison with natural ordering
func TestStringComparisonOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "file2 < file10",
			input:    `"file2" < "file10"`,
			expected: "true",
		},
		{
			name:     "file10 > file2",
			input:    `"file10" > "file2"`,
			expected: "true",
		},
		{
			name:     "file10 >= file10",
			input:    `"file10" >= "file10"`,
			expected: "true",
		},
		{
			name:     "file10 <= file2",
			input:    `"file10" <= "file2"`,
			expected: "false",
		},
		{
			name:     "v1.5 < v2.0",
			input:    `"v1.5" < "v2.0"`,
			expected: "true",
		},
		{
			name:     "v10.0 > v2.0",
			input:    `"v10.0" > "v2.0"`,
			expected: "true",
		},
		{
			name:     "alphabetical a < b",
			input:    `"apple" < "banana"`,
			expected: "true",
		},
		{
			name:     "equal strings",
			input:    `"test" <= "test"`,
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalSort(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestSortByNatural tests that sortBy inherits natural sort
func TestSortByNatural(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "sortBy with natural order",
			input:    `[{name: "file10"}, {name: "file2"}, {name: "file1"}].sortBy(fn(x) { x.name }).map(fn(x) { x.name }).join(",")`,
			expected: "file1,file2,file10",
		},
		{
			name:     "sortBy with version numbers",
			input:    `[{ver: "v10.0"}, {ver: "v2.0"}, {ver: "v1.5"}].sortBy(fn(x) { x.ver }).map(fn(x) { x.ver }).join(",")`,
			expected: "v1.5,v2.0,v10.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalSort(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestSortDescending tests descending sort using reverse
func TestSortDescending(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "descending natural sort",
			input:    `["file1", "file10", "file2"].sort().reverse().join(",")`,
			expected: "file10,file2,file1",
		},
		{
			name:     "descending lexicographic sort",
			input:    `["file1", "file10", "file2"].sort({natural: false}).reverse().join(",")`,
			expected: "file2,file10,file1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalSort(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestUnicodeNaturalSort tests natural sort with unicode digits
func TestUnicodeNaturalSort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "arabic-indic digits",
			input:    `["file١٠", "file٢", "file١"].sort().join(",")`,
			expected: "file١,file٢,file١٠",
		},
		{
			name:     "mixed ascii and unicode text",
			input:    `["файл10", "файл2", "файл1"].sort().join(",")`,
			expected: "файл1,файл2,файл10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalSort(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// TestSortEmptyAndSingle tests edge cases
func TestSortEmptyAndSingle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty array sort returns empty",
			input:    `[].sort().join(",")`,
			expected: "",
		},
		{
			name:     "single element",
			input:    `["only"].sort().join(",")`,
			expected: "only",
		},
		{
			name:     "already sorted",
			input:    `["a", "b", "c"].sort().join(",")`,
			expected: "a,b,c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEvalSort(tt.input)
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}
