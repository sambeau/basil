package lexer

import (
	"testing"
)

// Realistic Parsley code samples of varying complexity
var (
	simpleCode = `let x = 1 + 2 * 3`

	mediumCode = `
fn greet(name) {
    let message = "Hello, " + name + "!"
    <div class="greeting">
        <h1>{message}</h1>
    </div>
}
`

	complexCode = `// Handler with database query
import @basil/http
import @std/table

let db = @sqlite("./app.db")

fn handler(request) {
    let users = db |< @query Users
        | status == "active"
        | age >= 18
        ??->

    <html>
        <body>
            <table>
                {for user in users}
                    <tr>
                        <td>{user.name}</td>
                        <td>{user.email}</td>
                    </tr>
                {/for}
            </table>
        </body>
    </html>
}

export handler
`

	// Benchmark with Unicode identifiers (will work after FEAT-103)
	unicodeCode = `
let x = 3.14159
let y = 2 * x
let z = x - y
let name = "Tanaka"
fn area(r) = x * r * r
`
)

func BenchmarkLexer_Simple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		l := New(simpleCode)
		for tok := l.NextToken(); tok.Type != EOF; tok = l.NextToken() {
		}
	}
}

func BenchmarkLexer_Medium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		l := New(mediumCode)
		for tok := l.NextToken(); tok.Type != EOF; tok = l.NextToken() {
		}
	}
}

func BenchmarkLexer_Complex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		l := New(complexCode)
		for tok := l.NextToken(); tok.Type != EOF; tok = l.NextToken() {
		}
	}
}

func BenchmarkLexer_UnicodeEquivalent(b *testing.B) {
	// ASCII-only equivalent of what will become Unicode code
	// Used to compare apples-to-apples before/after
	for i := 0; i < b.N; i++ {
		l := New(unicodeCode)
		for tok := l.NextToken(); tok.Type != EOF; tok = l.NextToken() {
		}
	}
}
