package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestLetThenTableWithHeadingsDoesNotCrash(t *testing.T) {
	input := `<div>
	let data = [
		{name: "Alice", country:"USA", paid: £32},
		{name: "Bob", country:"CA", paid: £16},
		{name: "Charlie", country:"UK", paid: £11},
		{name: "Sam", country:"UK", paid: £12},
	]
	headings = <tr>
		for (k in 1..3) {<td>k</td>}
	</tr>
	<table>
		headings
	</table>
</div>`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if errObj, ok := result.(*evaluator.Error); ok {
		t.Fatalf("unexpected error: %s", errObj.Inspect())
	}

	expected := "<div><table><tr><td>1</td><td>2</td><td>3</td></tr></table></div>"
	if result.Inspect() != expected {
		t.Fatalf("expected %q, got %q", expected, result.Inspect())
	}
}
