package evaluator

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// benchEval parses and evaluates code, returning the result
func benchEval(input string) Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := NewEnvironment()
	return Eval(program, env)
}

// Benchmark code samples
var (
	// Arithmetic operations
	arithmeticCode = `
let a = 1 + 2 + 3 + 4 + 5
let b = 10 * 20 * 30
let c = 100 / 4 / 2
let d = a + b - c
let e = (a * b) / (c + 1)
d + e
`

	// String operations
	stringCode = `
let first = "Hello"
let second = "World"
let greeting = first + ", " + second + "!"
let upper = greeting.toUpper()
let lower = greeting.toLower()
let trimmed = "  spaces  ".trim()
let replaced = greeting.replace("World", "Parsley")
replaced
`

	// Array operations
	arrayCode = `
let numbers = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
let doubled = for n in numbers { n * 2 }
let filtered = for n in numbers { if n > 5 { n } }
let first = numbers[0]
let last = numbers[9]
let slice = numbers[2:5]
let length = len(numbers)
doubled
`

	// Dictionary operations
	dictCode = `
let person = {
    name: "Alice",
    age: 30,
    city: "London",
    active: true
}
let name = person.name
let age = person.age
let keys = person.keys()
let values = person.values()
let merged = person.merge({country: "UK", zip: "SW1A"})
merged
`

	// Function calls (user-defined)
	functionCode = `
let add = fn(a, b) { a + b }
let multiply = fn(a, b) { a * b }
let compose = fn(f, g, x) { f(g(x, x), x) }

let r1 = add(1, 2)
let r2 = multiply(3, 4)
let r3 = compose(add, multiply, 5)
r3
`

	// Builtin function calls
	builtinCode = `
let arr = [5, 2, 8, 1, 9, 3, 7, 4, 6]
let length = len(arr)
let str = string(12345)
let num = int("42")
let t = type(arr)
let rangeArr = range(1, 10)
length
`

	// Control flow (conditionals)
	controlFlowCode = `
let score = 85

let grade = if score >= 90 {
    "A"
} else if score >= 80 {
    "B"
} else if score >= 70 {
    "C"
} else if score >= 60 {
    "D"
} else {
    "F"
}

let status = if score >= 60 { "Pass" } else { "Fail" }
grade
`

	// Loops (for expressions)
	loopCode = `
let items = [
    {name: "apple", price: 1.50},
    {name: "banana", price: 0.75},
    {name: "cherry", price: 2.00}
]

let names = for item in items { item.name }
let prices = for item in items { item.price }
let expensive = for item in items { if item.price > 1.00 { item } }
let formatted = for item in items {
    item.name + ": $" + string(item.price)
}
formatted
`

	// HTML generation (tag evaluation)
	htmlCode = `
let item = {name: "Widget", price: 9.99, inStock: true}

<div class="product">
    <h2 class="title">{item.name}</h2>
    <p class="price">"$" + string(item.price)</p>
    {if item.inStock}
        <span class="status available">"In Stock"</span>
    {else}
        <span class="status unavailable">"Out of Stock"</span>
    {/if}
</div>
`

	// Nested structures
	nestedCode = `
let data = {
    users: [
        {id: 1, name: "Alice", roles: ["admin", "user"]},
        {id: 2, name: "Bob", roles: ["user"]},
        {id: 3, name: "Charlie", roles: ["user", "editor"]}
    ],
    settings: {
        theme: "dark",
        notifications: {
            email: true,
            push: false
        }
    }
}

let firstUser = data.users[0]
let theme = data.settings.theme
let emailEnabled = data.settings.notifications.email
let adminUsers = for user in data.users {
    if "admin" in user.roles { user }
}
adminUsers
`

	// Mixed operations (realistic workload)
	mixedCode = `
let users = [
    {name: "Alice", score: 95},
    {name: "Bob", score: 82},
    {name: "Charlie", score: 78},
    {name: "Diana", score: 91}
]

let getGrade = fn(score) {
    if score >= 90 { "A" }
    else if score >= 80 { "B" }
    else if score >= 70 { "C" }
    else { "D" }
}

let results = for user in users {
    {
        name: user.name,
        score: user.score,
        grade: getGrade(user.score),
        passed: user.score >= 70
    }
}

let passing = for r in results { if r.passed { r } }
let average = 86.5
results
`
)

func BenchmarkEvaluator_Arithmetic(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchEval(arithmeticCode)
	}
}

func BenchmarkEvaluator_Strings(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchEval(stringCode)
	}
}

func BenchmarkEvaluator_Arrays(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchEval(arrayCode)
	}
}

func BenchmarkEvaluator_Dicts(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchEval(dictCode)
	}
}

func BenchmarkEvaluator_Functions(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchEval(functionCode)
	}
}

func BenchmarkEvaluator_Builtins(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchEval(builtinCode)
	}
}

func BenchmarkEvaluator_ControlFlow(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchEval(controlFlowCode)
	}
}

func BenchmarkEvaluator_Loops(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchEval(loopCode)
	}
}

func BenchmarkEvaluator_HTMLGeneration(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchEval(htmlCode)
	}
}

func BenchmarkEvaluator_NestedStructures(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchEval(nestedCode)
	}
}

func BenchmarkEvaluator_MixedWorkload(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = benchEval(mixedCode)
	}
}
