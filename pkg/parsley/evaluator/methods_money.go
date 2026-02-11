// Package evaluator provides money method implementations via declarative registry.
// This file implements money methods for FEAT-111: Declarative Method Registry
package evaluator

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// MoneyMethodRegistry defines all methods available on money values.
// This is the single source of truth for money method dispatch and introspection.
var MoneyMethodRegistry MethodRegistry

func init() {
	MoneyMethodRegistry = MethodRegistry{
		"format": {
			Fn:          moneyFormat,
			Arity:       "0-1",
			Description: "Format with locale",
		},
		"abs": {
			Fn:          moneyAbsMethod,
			Arity:       "0",
			Description: "Absolute value",
		},
		"negate": {
			Fn:          moneyNegate,
			Arity:       "0",
			Description: "Negate amount",
		},
		"split": {
			Fn:          moneySplitMethod,
			Arity:       "1",
			Description: "Split into n parts that sum to original",
		},
		"toJSON": {
			Fn:          moneyToJSON,
			Arity:       "0",
			Description: "Convert to JSON string",
		},
		"toBox": {
			Fn:          moneyToBoxMethod,
			Arity:       "0-1",
			Description: "Render as box diagram",
		},
		"repr": {
			Fn:          moneyRepr,
			Arity:       "0",
			Description: "Get representation string",
		},
		"toDict": {
			Fn:          moneyToDict,
			Arity:       "0",
			Description: "Convert to dictionary",
		},
		"inspect": {
			Fn:          moneyInspect,
			Arity:       "0",
			Description: "Get debug dictionary with internal values",
		},
	}
	RegisterMethodRegistry("money", MoneyMethodRegistry)
}

// Money method implementations

func moneyFormat(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	localeStr := "en-US" // default locale
	if len(args) == 1 {
		localeArg, ok := args[0].(*String)
		if !ok {
			return newTypeError("TYPE-0012", "format", "a string", args[0].Type())
		}
		localeStr = localeArg.Value
	}
	return formatMoney(money, localeStr)
}

func moneyAbsMethod(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	amount := money.Amount
	if amount < 0 {
		amount = -amount
	}
	return &Money{
		Amount:   amount,
		Currency: money.Currency,
		Scale:    money.Scale,
	}
}

func moneyNegate(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	return &Money{
		Amount:   -money.Amount,
		Currency: money.Currency,
		Scale:    money.Scale,
	}
}

func moneySplitMethod(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	nArg, ok := args[0].(*Integer)
	if !ok {
		return newTypeError("TYPE-0012", "split", "an integer", args[0].Type())
	}
	n := nArg.Value
	if n <= 0 {
		return newStructuredError("VAL-0021", map[string]any{"Function": "split", "Expected": "a positive integer", "Got": n})
	}
	return splitMoney(money, n)
}

func moneyToJSON(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	// Format the amount as a decimal string to preserve precision
	amountStr := money.formatAmount()
	currencyJSON, _ := json.Marshal(money.Currency)
	return &String{Value: fmt.Sprintf(`{"amount":"%s","currency":%s}`, amountStr, currencyJSON)}
}

func moneyToBoxMethod(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	return moneyToBox(money, args)
}

func moneyRepr(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	return &String{Value: money.Inspect()}
}

func moneyToDict(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	// Calculate user-friendly amount (e.g., 50.00 not 5000)
	divisor := math.Pow10(int(money.Scale))
	amount := float64(money.Amount) / divisor
	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"amount":   createLiteralExpression(&Float{Value: amount}),
			"currency": createLiteralExpression(&String{Value: money.Currency}),
		},
		Env: NewEnvironment(),
	}
}

func moneyInspect(receiver Object, args []Object, env *Environment) Object {
	money := receiver.(*Money)
	return &Dictionary{
		Pairs: map[string]ast.Expression{
			"__type":   createLiteralExpression(&String{Value: "money"}),
			"amount":   createLiteralExpression(&Integer{Value: money.Amount}),
			"currency": createLiteralExpression(&String{Value: money.Currency}),
			"scale":    createLiteralExpression(&Integer{Value: int64(money.Scale)}),
		},
		Env: NewEnvironment(),
	}
}
