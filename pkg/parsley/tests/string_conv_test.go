package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func evalStringConv(input string) string {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)
	return evaluator.ObjectToPrintString(result)
}

func evalStringConvInspect(input string) string {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)
	if result != nil {
		return result.Inspect()
	}
	return ""
}

func TestDurationString(t *testing.T) {
	result := evalStringConv("@1d")
	expected := "1 day"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestDurationMultiUnit(t *testing.T) {
	result := evalStringConv("@2d12h")
	expected := "2 days 12 hours"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestDurationYearsMonths(t *testing.T) {
	result := evalStringConv("@1y6mo")
	expected := "1 year 6 months"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRegexString(t *testing.T) {
	result := evalStringConv("/hello.*/i")
	expected := "/hello.*/i"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRegexFormatPattern(t *testing.T) {
	result := evalStringConv(`/hello.*/i.format("pattern")`)
	expected := "hello.*"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRegexFormatVerbose(t *testing.T) {
	result := evalStringConv(`/hello.*/i.format("verbose")`)
	expected := "pattern: hello.*, flags: i"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRegexTest(t *testing.T) {
	result := evalStringConv(`/hello/i.test("Hello World")`)
	if result != "true" {
		t.Errorf("got %q, want true", result)
	}
}

func TestRegexTestNoMatch(t *testing.T) {
	result := evalStringConv(`/goodbye/i.test("Hello World")`)
	if result != "false" {
		t.Errorf("got %q, want false", result)
	}
}

func TestPathString(t *testing.T) {
	result := evalStringConv("@./src/main.go")
	expected := "./src/main.go"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestURLString(t *testing.T) {
	result := evalStringConv("@https://example.com:8080/api/users")
	expected := "https://example.com:8080/api/users"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestPathToDict(t *testing.T) {
	// toDict returns a clean dictionary without __type (for reconstruction)
	result := evalStringConvInspect(`let p = @./src/main.go
p.toDict()`)
	// Should NOT contain __type
	if strings.Contains(result, "__type") {
		t.Errorf("toDict should return dict WITHOUT __type, got %q", result)
	}
	// Should contain path data
	if !strings.Contains(result, "segments") {
		t.Errorf("toDict should contain path segments, got %q", result)
	}
}

func TestPathInspect(t *testing.T) {
	// inspect returns full dictionary with __type (for debugging)
	result := evalStringConvInspect(`let p = @./src/main.go
p.inspect()`)
	if !strings.Contains(result, "__type") || !strings.Contains(result, "path") {
		t.Errorf("inspect should return dict with __type: path, got %q", result)
	}
}

func TestURLToDict(t *testing.T) {
	// toDict returns a clean dictionary without __type (for reconstruction)
	result := evalStringConvInspect(`let u = @https://example.com
u.toDict()`)
	// Should NOT contain __type
	if strings.Contains(result, "__type") {
		t.Errorf("toDict should return dict WITHOUT __type, got %q", result)
	}
	// Should contain URL components
	if !strings.Contains(result, "scheme") || !strings.Contains(result, "host") {
		t.Errorf("toDict should contain URL components, got %q", result)
	}
}

func TestURLInspect(t *testing.T) {
	result := evalStringConvInspect(`let u = @https://example.com
u.inspect()`)
	if !strings.Contains(result, "__type") || !strings.Contains(result, "url") {
		t.Errorf("inspect should return dict with __type: url, got %q", result)
	}
}

func TestDatetimeToDict(t *testing.T) {
	// toDict returns a clean dictionary without __type (for reconstruction)
	result := evalStringConvInspect(`let d = @2024-12-25
d.toDict()`)
	// Should NOT contain __type
	if strings.Contains(result, "__type") {
		t.Errorf("toDict should return dict WITHOUT __type, got %q", result)
	}
	// Should contain datetime components
	if !strings.Contains(result, "year") || !strings.Contains(result, "month") || !strings.Contains(result, "day") {
		t.Errorf("toDict should contain datetime components, got %q", result)
	}
}

func TestDatetimeInspect(t *testing.T) {
	result := evalStringConvInspect(`let d = @2024-12-25
d.inspect()`)
	if !strings.Contains(result, "__type") || !strings.Contains(result, "datetime") {
		t.Errorf("inspect should return dict with __type: datetime, got %q", result)
	}
}

func TestDurationToDict(t *testing.T) {
	// toDict returns a clean dictionary without __type (for reconstruction)
	result := evalStringConvInspect(`let d = @1d
d.toDict()`)
	// Should NOT contain __type
	if strings.Contains(result, "__type") {
		t.Errorf("toDict should return dict WITHOUT __type, got %q", result)
	}
	// Should contain duration components
	if !strings.Contains(result, "seconds") {
		t.Errorf("toDict should contain duration components, got %q", result)
	}
}

func TestDurationInspect(t *testing.T) {
	result := evalStringConvInspect(`let d = @1d
d.inspect()`)
	if !strings.Contains(result, "__type") || !strings.Contains(result, "duration") {
		t.Errorf("inspect should return dict with __type: duration, got %q", result)
	}
}

func TestRegexToDict(t *testing.T) {
	// toDict returns a clean dictionary without __type (for reconstruction)
	result := evalStringConvInspect(`let r = /hello.*/i
r.toDict()`)
	// Should NOT contain __type
	if strings.Contains(result, "__type") {
		t.Errorf("toDict should return dict WITHOUT __type, got %q", result)
	}
	// Should contain regex components
	if !strings.Contains(result, "pattern") || !strings.Contains(result, "flags") {
		t.Errorf("toDict should contain pattern and flags, got %q", result)
	}
}

func TestRegexInspect(t *testing.T) {
	result := evalStringConvInspect(`let r = /hello.*/i
r.inspect()`)
	if !strings.Contains(result, "__type") || !strings.Contains(result, "regex") {
		t.Errorf("inspect should return dict with __type: regex, got %q", result)
	}
}

func TestReprDuration(t *testing.T) {
	result := evalStringConv("repr(@1d)")
	// repr() returns parseable Parsley literal, not dict representation
	if result != "@1d" && !strings.HasPrefix(result, "@") {
		t.Errorf("repr should return parseable duration literal, got %q", result)
	}
}

func TestReprPath(t *testing.T) {
	result := evalStringConv("repr(@./src/main.go)")
	// repr() returns parseable Parsley literal, not dict representation
	if result != "@./src/main.go" {
		t.Errorf("repr should return parseable path literal, got %q", result)
	}
}

func TestReprURL(t *testing.T) {
	result := evalStringConv("repr(@https://example.com)")
	// repr() returns parseable Parsley literal, not dict representation
	if result != "@https://example.com" {
		t.Errorf("repr should return parseable URL literal, got %q", result)
	}
}

func TestReprDatetime(t *testing.T) {
	result := evalStringConv("repr(@2024-12-25)")
	// repr() returns parseable Parsley literal, not dict representation
	if !strings.HasPrefix(result, "@2024-12-25") {
		t.Errorf("repr should return parseable datetime literal, got %q", result)
	}
}

func TestReprRegex(t *testing.T) {
	result := evalStringConv("repr(/test/i)")
	// repr() returns parseable Parsley literal, not dict representation
	if result != "/test/i" {
		t.Errorf("repr should return parseable regex literal, got %q", result)
	}
}

// Array join() tests
func TestArrayJoinNoSeparator(t *testing.T) {
	result := evalStringConv(`["a", "b", "c"].join()`)
	expected := "abc"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestArrayJoinWithSeparator(t *testing.T) {
	result := evalStringConv(`["a", "b", "c"].join("-")`)
	expected := "a-b-c"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestArrayJoinWithComma(t *testing.T) {
	result := evalStringConv(`["apple", "banana", "cherry"].join(", ")`)
	expected := "apple, banana, cherry"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestArrayJoinPathComponents(t *testing.T) {
	// For absolute paths, components no longer include empty string prefix
	// Use path string conversion instead for full path
	result := evalStringConv(`let p = @/usr/local/bin; "/" + p.segments.join("/")`)
	expected := "/usr/local/bin"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}
