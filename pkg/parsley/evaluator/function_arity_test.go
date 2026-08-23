package evaluator

import (
	"strings"
	"testing"
)

// BUG-032: user-defined functions must enforce arity at the call site.
//
// Before this fix, extra arguments were silently dropped and missing ones
// surfaced as `Identifier not found` pointing inside the callee's body.

func arityError(t *testing.T, input string) *Error {
	t.Helper()
	result := testEval(input)
	errObj, ok := result.(*Error)
	if !ok {
		t.Fatalf("expected Error for %q, got %T (%v)", input, result, result)
	}
	return errObj
}

func TestUserFunctionTooManyArguments(t *testing.T) {
	errObj := arityError(t, `let add = fn(a, b) { a + b }; add(1, 2, 3, 4)`)

	if errObj.Code != "ARITY-0007" {
		t.Errorf("expected ARITY-0007, got %s", errObj.Code)
	}
	if errObj.Class != ClassArity {
		t.Errorf("expected arity class, got %s", errObj.Class)
	}
	if want := "`add` expects 2 arguments, got 4"; errObj.Message != want {
		t.Errorf("expected %q, got %q", want, errObj.Message)
	}
}

func TestUserFunctionTooFewArguments(t *testing.T) {
	errObj := arityError(t, `let one = fn(a) { a }; one()`)

	if errObj.Code != "ARITY-0007" {
		t.Errorf("expected ARITY-0007, got %s", errObj.Code)
	}
	// Singular "argument" for a one-parameter function.
	if want := "`one` expects 1 argument, got 0"; errObj.Message != want {
		t.Errorf("expected %q, got %q", want, errObj.Message)
	}
}

// The old failure mode: a missing argument was reported as an unknown
// identifier inside the body, sometimes with a "did you mean" hint naming
// another parameter.
func TestUserFunctionTooFewIsNotAnUndefinedIdentifier(t *testing.T) {
	errObj := arityError(t, `let greet = fn(name, greeting) { greeting + ", " + name }; greet("Sam")`)

	if errObj.Code == "UNDEF-0001" {
		t.Fatalf("missing argument still reported as an unknown identifier: %s", errObj.Message)
	}
	if want := "`greet` expects 2 arguments, got 1"; errObj.Message != want {
		t.Errorf("expected %q, got %q", want, errObj.Message)
	}
	for _, hint := range errObj.Hints {
		if strings.Contains(hint, "Did you mean") {
			t.Errorf("unexpected typo hint on an arity error: %q", hint)
		}
	}
}

// The error must be positioned at the call, not at the parameter's use inside
// the callee's body.
func TestUserFunctionArityErrorIsAtTheCallSite(t *testing.T) {
	// Column 24 is the `o` of the `one()` call; column 19 would be the `a`
	// inside the body, which is where the old error pointed.
	errObj := arityError(t, `let one = fn(a) { a }; one()`)

	if errObj.Line != 1 {
		t.Errorf("expected line 1, got %d", errObj.Line)
	}
	if errObj.Column != 24 {
		t.Errorf("expected column 24 (the call), got %d", errObj.Column)
	}
}

func TestUserFunctionExactArityStillWorks(t *testing.T) {
	result := testEval(`let add = fn(a, b) { a + b }; add(1, 2)`)
	intObj, ok := result.(*Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T (%v)", result, result)
	}
	if intObj.Value != 3 {
		t.Errorf("expected 3, got %d", intObj.Value)
	}
}

// A destructuring parameter counts as one positional parameter; its inner
// leniency is unchanged.
func TestDestructuringParameterCountsAsOne(t *testing.T) {
	result := testEval(`let f = fn({a, b}) { a + b }; f({a: 1, b: 2})`)
	intObj, ok := result.(*Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T (%v)", result, result)
	}
	if intObj.Value != 3 {
		t.Errorf("expected 3, got %d", intObj.Value)
	}

	errObj := arityError(t, `let f = fn({a, b}) { a + b }; f({a: 1}, {b: 2})`)
	if errObj.Code != "ARITY-0007" {
		t.Errorf("expected ARITY-0007, got %s", errObj.Code)
	}
}

// An unbound function literal has no name to report.
func TestAnonymousFunctionArityError(t *testing.T) {
	errObj := arityError(t, `fn(a) { a }(1, 2)`)
	if want := "`anonymous fn` expects 1 argument, got 2"; errObj.Message != want {
		t.Errorf("expected %q, got %q", want, errObj.Message)
	}
}

// Functions reached through a dictionary key report that key.
func TestDictMethodArityErrorIsNamed(t *testing.T) {
	errObj := arityError(t, `let u = {name: "Sam", greet: fn(g) { g }}; u.greet("hi", "extra")`)
	if want := "`greet` expects 1 argument, got 2"; errObj.Message != want {
		t.Errorf("expected %q, got %q", want, errObj.Message)
	}
}

// Tag dispatch adapts: a prop-less component is declared `fn()` and must keep
// working, because tag dispatch passes the props dict only when the component
// declares a parameter for it.
func TestZeroParamComponentViaSelfClosingTag(t *testing.T) {
	result := testEval(`let C = fn() { "<p>hi</p>" }; <div><C/></div>`)
	if errObj, ok := result.(*Error); ok {
		t.Fatalf("prop-less component failed: %s", errObj.Message)
	}
	if got := result.Inspect(); !strings.Contains(got, "<p>hi</p>") {
		t.Errorf("expected rendered component, got %s", got)
	}
}

func TestOneParamComponentStillReceivesProps(t *testing.T) {
	result := testEval(`let C = fn(props) { props.name }; <div><C name="Sam"/></div>`)
	if errObj, ok := result.(*Error); ok {
		t.Fatalf("component with props failed: %s", errObj.Message)
	}
	if got := result.Inspect(); !strings.Contains(got, "Sam") {
		t.Errorf("expected props to reach the component, got %s", got)
	}
}

// .reduce adapts to 1- or 2-parameter reducers.
func TestReduceWithTwoParamReducer(t *testing.T) {
	result := testEval(`[1, 2, 3].reduce(fn(acc, x) { acc + x }, 0)`)
	intObj, ok := result.(*Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T (%v)", result, result)
	}
	if intObj.Value != 6 {
		t.Errorf("expected 6, got %d", intObj.Value)
	}
}

func TestReduceWithOneParamReducer(t *testing.T) {
	result := testEval(`[1, 2, 3].reduce(fn(acc) { acc + 1 }, 0)`)
	intObj, ok := result.(*Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T (%v)", result, result)
	}
	if intObj.Value != 3 {
		t.Errorf("expected 3, got %d", intObj.Value)
	}
}

func TestReduceRejectsUnsupportedReducerArity(t *testing.T) {
	errObj := arityError(t, `[1, 2, 3].reduce(fn(a, b, c) { a }, 0)`)
	if errObj.Code != "ARITY-0008" {
		t.Errorf("expected ARITY-0008, got %s", errObj.Code)
	}
	if want := "Function passed to `reduce` must take 1 or 2 parameters, got 3"; errObj.Message != want {
		t.Errorf("expected %q, got %q", want, errObj.Message)
	}
}

// The one-argument callback sites say so rather than silently dropping.
func TestMapRejectsWrongCallbackArity(t *testing.T) {
	errObj := arityError(t, `[1, 2, 3].map(fn(a, b) { a })`)
	if errObj.Code != "ARITY-0008" {
		t.Errorf("expected ARITY-0008, got %s", errObj.Code)
	}
	if want := "Function passed to `map` must take 1 parameter, got 2"; errObj.Message != want {
		t.Errorf("expected %q, got %q", want, errObj.Message)
	}
}

func TestFilterRejectsWrongCallbackArity(t *testing.T) {
	errObj := arityError(t, `[1, 2, 3].filter(fn() { true })`)
	if errObj.Code != "ARITY-0008" {
		t.Errorf("expected ARITY-0008, got %s", errObj.Code)
	}
}

// `for (…) fn` already adapted to the callee's arity; it must keep doing so.
func TestForLoopCallbacksStillAdapt(t *testing.T) {
	one := testEval(`for ([1, 2, 3]) fn(x) { x * 2 }`)
	if _, ok := one.(*Array); !ok {
		t.Fatalf("one-param loop callback failed: %T (%v)", one, one)
	}
	two := testEval(`for ([1, 2, 3]) fn(i, x) { i }`)
	if _, ok := two.(*Array); !ok {
		t.Fatalf("two-param loop callback failed: %T (%v)", two, two)
	}
}

// `.replace` with a function adapts to the callback's arity: it is handed
// exactly as many arguments as it declares, drawn from match then groups.
func TestReplaceWithFunctionAdaptsToArity(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"a-b-c".replace("-", fn(m) { "+" })`, "a+b+c"},
		{`"2024-01-02".replace(/(\d+)-(\d+)-(\d+)/, fn(m, y) { y })`, "2024"},
	}
	for _, tt := range tests {
		result := testEval(tt.input)
		strObj, ok := result.(*String)
		if !ok {
			t.Fatalf("expected String for %q, got %T (%v)", tt.input, result, result)
		}
		if strObj.Value != tt.expected {
			t.Errorf("for %q expected %q, got %q", tt.input, tt.expected, strObj.Value)
		}
	}
}
