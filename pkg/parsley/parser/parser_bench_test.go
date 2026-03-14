package parser

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// Realistic Parsley code samples of varying complexity
var (
	// Simple expression: literals and arithmetic
	simpleExpr = `let x = 1 + 2 * 3 - 4 / 2`

	// Medium complexity: function definition with conditionals
	mediumCode = `
let greet = fn(name, formal) {
    if formal {
        "Dear " + name
    } else {
        "Hi " + name + "!"
    }
}

let result = greet("Alice", true)
`

	// Complex program: multiple statements, nested structures
	complexCode = `
let users = [
    {name: "Alice", age: 30, active: true},
    {name: "Bob", age: 25, active: false},
    {name: "Charlie", age: 35, active: true}
]

let activeUsers = for user in users {
    if user.active {
        user
    }
}

let formatUser = fn(user) {
    let status = if user.active { "Active" } else { "Inactive" }
    {
        display: user.name + " (" + status + ")",
        sortKey: user.age
    }
}

let formatted = for user in activeUsers {
    formatUser(user)
}
`

	// HTML/tag-heavy code (common Parsley workload)
	htmlCode = `
let page = fn(title, items) {
    <html lang="en">
        <head>
            <title>{title}</title>
            <meta charset="utf-8"/>
            <link rel="stylesheet" href="/styles.css"/>
        </head>
        <body>
            <header class="main-header">
                <nav>
                    <a href="/">Home</a>
                    <a href="/about">About</a>
                </nav>
            </header>
            <main>
                <h1>{title}</h1>
                <ul class="item-list">
                    {for item in items}
                        <li class="item" data-id={item.id}>
                            <span class="name">{item.name}</span>
                            <span class="price">{item.price}</span>
                        </li>
                    {/for}
                </ul>
            </main>
            <footer>
                <p>"Copyright 2026"</p>
            </footer>
        </body>
    </html>
}
`

	// String operations and template literals
	stringCode = `
let name = "World"
let greeting = "Hello, " + name + "!"
let template = "The answer is {40 + 2}"
let multiConcat = "a" + "b" + "c" + "d" + "e"
let methods = "  hello  ".trim().toUpper()
`

	// Array and dict operations
	collectionCode = `
let numbers = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
let doubled = for n in numbers { n * 2 }
let filtered = for n in numbers { if n > 5 { n } }

let config = {
    host: "localhost",
    port: 8080,
    settings: {
        debug: true,
        timeout: 30
    }
}

let host = config.host
let debug = config.settings.debug
`
)

func BenchmarkParser_SimpleExpression(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		l := lexer.New(simpleExpr)
		p := New(l)
		_ = p.ParseProgram()
	}
}

func BenchmarkParser_MediumComplexity(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		l := lexer.New(mediumCode)
		p := New(l)
		_ = p.ParseProgram()
	}
}

func BenchmarkParser_ComplexProgram(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		l := lexer.New(complexCode)
		p := New(l)
		_ = p.ParseProgram()
	}
}

func BenchmarkParser_HTMLTags(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		l := lexer.New(htmlCode)
		p := New(l)
		_ = p.ParseProgram()
	}
}

func BenchmarkParser_StringOperations(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		l := lexer.New(stringCode)
		p := New(l)
		_ = p.ParseProgram()
	}
}

func BenchmarkParser_Collections(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		l := lexer.New(collectionCode)
		p := New(l)
		_ = p.ParseProgram()
	}
}
