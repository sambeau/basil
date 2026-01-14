package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// Helper to test money evaluations
func testEvalMoney(input string) evaluator.Object {
	return testEvalHelper(input)
}

func testExpectedMoney(t *testing.T, input string, obj evaluator.Object, expected string) {
	if obj == nil {
		t.Errorf("For input '%s': got nil object", input)
		return
	}

	if err, ok := obj.(*evaluator.Error); ok {
		t.Errorf("For input '%s': got error: %s", input, err.Message)
		return
	}

	actual := obj.Inspect()
	if actual != expected {
		t.Errorf("For input '%s': expected %s, got %s", input, expected, actual)
	}
}

func testExpectedError(t *testing.T, input string, obj evaluator.Object, expectedSubstring string) {
	if obj == nil {
		t.Errorf("For input '%s': expected error but got nil", input)
		return
	}

	err, ok := obj.(*evaluator.Error)
	if !ok {
		t.Errorf("For input '%s': expected error but got %s", input, obj.Inspect())
		return
	}

	if !strings.Contains(strings.ToLower(err.Message), strings.ToLower(expectedSubstring)) {
		t.Errorf("For input '%s': expected error containing '%s', got '%s'", input, expectedSubstring, err.Message)
	}
}

// ============================================================================
// Money Literals
// ============================================================================

func TestMoneyLiteralDollar(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`$0`, `$0.00`},
		{`$1`, `$1.00`},
		{`$99`, `$99.00`},
		{`$12.34`, `$12.34`},
		{`$0.99`, `$0.99`},
		{`$1000.00`, `$1000.00`},
		{`$0.01`, `$0.01`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneyLiteralPound(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`£0`, `£0.00`},
		{`£1`, `£1.00`},
		{`£99`, `£99.00`},
		{`£12.34`, `£12.34`},
		{`£0.99`, `£0.99`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneyLiteralEuro(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`€0`, `€0.00`},
		{`€1`, `€1.00`},
		{`€99`, `€99.00`},
		{`€12.34`, `€12.34`},
		{`€0.99`, `€0.99`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneyLiteralYen(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`¥0`, `¥0`},
		{`¥1`, `¥1`},
		{`¥100`, `¥100`},
		{`¥1000`, `¥1000`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneyLiteralCurrencyCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`USD#12.34`, `$12.34`},
		{`GBP#99.99`, `£99.99`},
		{`EUR#50.00`, `€50.00`},
		{`JPY#1000`, `¥1000`},
		{`CAD#25.00`, `CA$25.00`},
		{`CHF#100.50`, `CHF#100.50`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneyLiteralCompoundSymbols(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`CA$25.00`, `CA$25.00`},
		{`AU$50.00`, `AU$50.00`},
		{`HK$100.00`, `HK$100.00`},
		{`S$75.50`, `S$75.50`},
		{`CN¥500`, `CN¥500.00`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Money Arithmetic
// ============================================================================

func TestMoneyAddition(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`$10.00 + $5.00`, `$15.00`},
		{`$0.99 + $0.01`, `$1.00`},
		{`£100 + £50`, `£150.00`},
		{`€1.50 + €2.50`, `€4.00`},
		{`¥100 + ¥50`, `¥150`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneySubtraction(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`$10.00 - $5.00`, `$5.00`},
		{`$1.00 - $0.01`, `$0.99`},
		{`£100 - £50`, `£50.00`},
		{`€10.00 - €15.00`, `€-5.00`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneyMultiplication(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`$10.00 * 2`, `$20.00`},
		{`3 * $5.00`, `$15.00`},
		{`$1.00 * 0.5`, `$0.50`},
		{`£10 * 3`, `£30.00`},
		{`2.5 * €4.00`, `€10.00`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneyDivision(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`$10.00 / 2`, `$5.00`},
		{`$10.00 / 4`, `$2.50`},
		{`£100.00 / 3`, `£33.33`},
		{`€1.00 / 3`, `€0.33`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneyUnaryMinus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`-$10.00`, `$-10.00`},
		{`-£50`, `£-50.00`},
		{`-€0.99`, `€-0.99`},
		{`-(-$10.00)`, `$10.00`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Money Comparison
// ============================================================================

func TestMoneyEquality(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`$10.00 == $10.00`, `true`},
		{`$10.00 == $10`, `true`},
		{`$10.00 != $5.00`, `true`},
		{`£100 == £100.00`, `true`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneyComparison(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`$10.00 > $5.00`, `true`},
		{`$5.00 < $10.00`, `true`},
		{`$10.00 >= $10.00`, `true`},
		{`$10.00 <= $10.00`, `true`},
		{`$5.00 > $10.00`, `false`},
		{`£100 > £50`, `true`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Money Properties
// ============================================================================

func TestMoneyProperties(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`$12.34.currency`, `USD`},
		{`$12.34.amount`, `1234`},
		{`$12.34.scale`, `2`},
		{`£99.99.currency`, `GBP`},
		{`€50.00.amount`, `5000`},
		{`¥1000.currency`, `JPY`},
		{`¥1000.scale`, `0`},
		{`¥1000.amount`, `1000`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Money Methods
// ============================================================================

func TestMoneyAbs(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`$10.00.abs()`, `$10.00`},
		{`(-$10.00).abs()`, `$10.00`},
		{`$0.00.abs()`, `$0.00`},
		{`(-£50.00).abs()`, `£50.00`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneySplit(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`$10.00.split(2)`, `[$5.00, $5.00]`},
		{`$10.00.split(3)`, `[$3.34, $3.33, $3.33]`},
		{`$1.00.split(3)`, `[$0.34, $0.33, $0.33]`},
		{`$0.01.split(3)`, `[$0.01, $0.00, $0.00]`},
		{`£100.00.split(4)`, `[£25.00, £25.00, £25.00, £25.00]`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneyFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`$12.34.format()`, `$ 12.34`},
		{`$1234.56.format()`, `$ 1,234.56`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Money Constructor
// ============================================================================

func TestMoneyConstructor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`money(12.34, "USD")`, `$12.34`},
		{`money(99.99, "GBP")`, `£99.99`},
		{`money(50, "EUR")`, `€50.00`},
		{`money(1000, "JPY")`, `¥1000`},
		{`money(25.50, "CAD")`, `CA$25.50`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneyConstructorWithScale(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// With explicit scale, amount is in minor units
		{`money(1234, "USD", 2)`, `$12.34`},
		{`money(9999, "GBP", 2)`, `£99.99`},
		{`money(5000, "EUR", 2)`, `€50.00`},
		{`money(1000, "JPY", 0)`, `¥1000`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Money Errors
// ============================================================================

func TestMoneyCurrencyMismatch(t *testing.T) {
	tests := []struct {
		input          string
		expectedSubstr string
	}{
		{`$10.00 + £5.00`, "cannot mix currencies"},
		{`$10.00 - €5.00`, "cannot mix currencies"},
		{`$10.00 == £10.00`, "cannot mix currencies"},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedError(t, tt.input, evaluated, tt.expectedSubstr)
	}
}

func TestMoneyDivisionByZero(t *testing.T) {
	tests := []struct {
		input          string
		expectedSubstr string
	}{
		{`$10.00 / 0`, "division by zero"},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedError(t, tt.input, evaluated, tt.expectedSubstr)
	}
}

func TestMoneyInvalidOperations(t *testing.T) {
	tests := []struct {
		input          string
		expectedSubstr string
	}{
		{`$10.00 + 5`, "unsupported operation"},
		{`$10.00 * $5.00`, "unsupported operation"},
		{`$10.00 / $5.00`, "unsupported operation"},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedError(t, tt.input, evaluated, tt.expectedSubstr)
	}
}

// ============================================================================
// Money in Variables and Expressions
// ============================================================================

func TestMoneyVariables(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`price = $19.99; price`, `$19.99`},
		{`a = $10.00; b = $5.00; a + b`, `$15.00`},
		{`tax = $10.00 * 0.2; tax`, `$2.00`},
		{`total = $100.00; discount = $20.00; total - discount`, `$80.00`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneyInConditionals(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`if $10.00 > $5.00 { "expensive" } else { "cheap" }`, `expensive`},
		{`price = $25.00; if price >= $20.00 { "premium" } else { "budget" }`, `premium`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneyInArrays(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`prices = [$10.00, $20.00, $30.00]; prices[0]`, `$10.00`},
		{`prices = [$10.00, $20.00, $30.00]; prices[1] + prices[2]`, `$50.00`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneyInDictionaries(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`product = { price: $29.99 }; product.price`, `$29.99`},
		{`order = { subtotal: $100.00, tax: $8.00 }; order.subtotal + order.tax`, `$108.00`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

// ============================================================================
// Money Methods: repr, toDict, inspect
// ============================================================================

func TestMoneyReprMethod(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`$50.00.repr()`, `$50.00`},
		{`£99.99.repr()`, `£99.99`},
		{`¥1000.repr()`, `¥1000`},
		// EUR gets normalized to € symbol
		{`EUR#25.50.repr()`, `€25.50`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		str, ok := evaluated.(*evaluator.String)
		if !ok {
			t.Errorf("For input '%s': expected string, got %T", tt.input, evaluated)
			continue
		}
		if str.Value != tt.expected {
			t.Errorf("For input '%s': expected '%s', got '%s'", tt.input, tt.expected, str.Value)
		}
	}
}

func TestMoneyToDictMethod(t *testing.T) {
	// toDict returns clean dict for reconstruction (user-friendly amount)
	result := testEvalMoney(`$50.00.toDict()`)
	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected dictionary, got %T: %s", result, result.Inspect())
	}

	// Check amount is user-friendly (50.0, not 5000)
	amountExpr, hasAmount := dict.Pairs["amount"]
	if !hasAmount {
		t.Fatal("toDict should have 'amount' key")
	}
	// Dictionary pairs store ast.Expression (FloatLiteral)
	floatLit, ok := amountExpr.(*ast.FloatLiteral)
	if !ok {
		t.Fatalf("amount should be FloatLiteral, got %T", amountExpr)
	}
	if floatLit.Value != 50.0 {
		t.Errorf("amount should be 50.0, got %f", floatLit.Value)
	}

	// Check currency
	currencyExpr, hasCurrency := dict.Pairs["currency"]
	if !hasCurrency {
		t.Fatal("toDict should have 'currency' key")
	}
	strLit, ok := currencyExpr.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("currency should be StringLiteral, got %T", currencyExpr)
	}
	if strLit.Value != "USD" {
		t.Errorf("currency should be 'USD', got '%s'", strLit.Value)
	}

	// Should NOT have __type
	if _, hasType := dict.Pairs["__type"]; hasType {
		t.Error("toDict should NOT have __type key")
	}
}

func TestMoneyInspectMethod(t *testing.T) {
	// inspect returns debug dict with __type and raw internal values
	result := testEvalMoney(`$50.00.inspect()`)
	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected dictionary, got %T: %s", result, result.Inspect())
	}

	// Should have __type
	typeExpr, hasType := dict.Pairs["__type"]
	if !hasType {
		t.Fatal("inspect should have '__type' key")
	}
	typeLit, ok := typeExpr.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("__type should be StringLiteral, got %T", typeExpr)
	}
	if typeLit.Value != "money" {
		t.Errorf("__type should be 'money', got '%s'", typeLit.Value)
	}

	// Check amount is raw cents (5000, not 50.0)
	amountExpr, hasAmount := dict.Pairs["amount"]
	if !hasAmount {
		t.Fatal("inspect should have 'amount' key")
	}
	intLit, ok := amountExpr.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("amount should be IntegerLiteral (raw cents), got %T", amountExpr)
	}
	if intLit.Value != 5000 {
		t.Errorf("amount should be 5000 (cents), got %d", intLit.Value)
	}

	// Check scale
	scaleExpr, hasScale := dict.Pairs["scale"]
	if !hasScale {
		t.Fatal("inspect should have 'scale' key")
	}
	scaleLit, ok := scaleExpr.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("scale should be IntegerLiteral, got %T", scaleExpr)
	}
	if scaleLit.Value != 2 {
		t.Errorf("scale should be 2, got %d", scaleLit.Value)
	}
}

// ============================================================================
// Money Constructor from Dictionary
// ============================================================================

func TestMoneyConstructorFromDict(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic dict constructor
		{`money({amount: 50.00, currency: "USD"})`, `$50.00`},
		{`money({amount: 99.99, currency: "GBP"})`, `£99.99`},
		{`money({amount: 1000, currency: "JPY"})`, `¥1000`},

		// Round-trip: value -> toDict() -> money(dict)
		{`money($50.00.toDict())`, `$50.00`},
		{`money(£99.99.toDict())`, `£99.99`},
		{`money(¥1000.toDict())`, `¥1000`},
		{`money(EUR#25.50.toDict())`, `€25.50`},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedMoney(t, tt.input, evaluated, tt.expected)
	}
}

func TestMoneyConstructorFromDictErrors(t *testing.T) {
	tests := []struct {
		input           string
		expectedContain string
	}{
		// Missing amount
		{`money({currency: "USD"})`, "missing"},
		// Missing currency
		{`money({amount: 50.00})`, "missing"},
		// Wrong type for amount
		{`money({amount: "fifty", currency: "USD"})`, "number"},
		// Wrong type for currency
		{`money({amount: 50.00, currency: 123})`, "string"},
	}

	for _, tt := range tests {
		evaluated := testEvalMoney(tt.input)
		testExpectedError(t, tt.input, evaluated, tt.expectedContain)
	}
}
