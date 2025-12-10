package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestPartTagParsing(t *testing.T) {
	input := `<Part src={@./test_fixtures/parts/counter.part} />`

	l := lexer.NewWithFilename(input, "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars")
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	// Check the AST structure
	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	exprStmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
	}

	tagLit, ok := exprStmt.Expression.(*ast.TagLiteral)
	if !ok {
		t.Fatalf("expected TagLiteral, got %T", exprStmt.Expression)
	}

	t.Logf("Tag raw: %q", tagLit.Raw)

	env := evaluator.NewEnvironment()
	env.Filename = "/Users/samphillips/Dev/basil/pkg/parsley/tests/test.pars"
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	result := evaluator.Eval(program, env)

	if result.Type() == evaluator.ERROR_OBJ {
		t.Logf("evaluation error: %s", result.Inspect())
	} else {
		t.Logf("result type: %s", result.Type())
		t.Logf("result value: %s", result.Inspect())
	}
}
