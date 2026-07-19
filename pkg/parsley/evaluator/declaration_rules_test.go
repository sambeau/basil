package evaluator

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// evalInEnv parses and evaluates input in the given environment (REPL-style)
func evalInEnv(t *testing.T, input string, env *Environment) Object {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse error for %q: %s", input, p.Errors()[0])
	}
	return Eval(program, env)
}

// TestSameScopeRedeclarationErrors verifies that declaring a name twice in the
// same scope is a DECL-0001 error, for every declaration form.
func TestSameScopeRedeclarationErrors(t *testing.T) {
	tests := []struct {
		desc  string
		input string
	}{
		{"let then let", "let x = 1\nlet x = 2"},
		{"var then var", "var x = 1\nvar x = 2"},
		{"let then var", "let x = 1\nvar x = 2"},
		{"var then let", "var x = 1\nlet x = 2"},
		{"let then array destructure", "let x = 1\nlet [x, y] = [2, 3]"},
		{"array destructure then let", "let [a, b] = [1, 2]\nlet a = 3"},
		{"duplicate name in array pattern", "let [a, a] = [1, 2]"},
		{"let then dict destructure", "let name = \"x\"\nlet {name} = {name: \"y\"}"},
		{"dict destructure alias collision", "let who = 1\nlet {name as who} = {name: \"y\"}"},
		{"rest collision", "let rest = 1\nlet [a, ...rest] = [1, 2, 3]"},
		{"dict rest collision", "let rest = 1\nlet {a, ...rest} = {a: 1, b: 2}"},
		{"redeclaring a function parameter", "let f = fn(x) {\n    let x = 2\n    x\n}\nf(1)"},
		{"redeclare inside function body", "let f = fn() {\n    let y = 1\n    let y = 2\n}\nf()"},
		{"redeclare inside if block", "if true {\n    let z = 1\n    let z = 2\n}"},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		err, ok := result.(*Error)
		if !ok {
			t.Errorf("%s: expected DECL-0001 error, got %s (%s)", tt.desc, result.Type(), result.Inspect())
			continue
		}
		if err.Code != "DECL-0001" {
			t.Errorf("%s: expected code DECL-0001, got %q (message: %s)", tt.desc, err.Code, err.Message)
		}
	}
}

// TestShadowingInInnerScopesAllowed verifies that declaring the same name in a
// genuinely inner scope is still allowed and leaves the outer binding intact.
func TestShadowingInInnerScopesAllowed(t *testing.T) {
	tests := []struct {
		desc     string
		input    string
		expected int64
	}{
		{"function body shadows outer", "let x = 1\nlet f = fn() {\n    let x = 2\n    x\n}\nlet inner = f()\ninner * 10 + x", 21},
		{"if block shadows outer, outer intact", "let x = 1\nif true {\n    let x = 2\n}\nx", 1},
		{"if block sees shadowed value", "let x = 1\nlet y = if true {\n    let x = 2\n    x\n} else {\n    0\n}\ny", 2},
		{"sequential if blocks reuse a name", "var out = 0\nif true {\n    let msg = 10\n    out = msg\n}\nif true {\n    let msg = 32\n    out = out + msg\n}\nout", 42},
		{"loop body redeclares per iteration", "var total = 0\nlet _ = for n in [1, 2, 3] {\n    let double = n * 2\n    total = total + double\n}\ntotal", 12},
		{"else block scope", "let x = 1\nif false {\n    0\n} else {\n    let x = 3\n}\nx", 1},
	}

	for _, tt := range tests {
		result := testEval(tt.input)
		if err, ok := result.(*Error); ok {
			t.Errorf("%s: unexpected error: %s", tt.desc, err.Inspect())
			continue
		}
		intObj, ok := result.(*Integer)
		if !ok {
			t.Errorf("%s: expected INTEGER, got %s (%s)", tt.desc, result.Type(), result.Inspect())
			continue
		}
		if intObj.Value != tt.expected {
			t.Errorf("%s: expected %d, got %d", tt.desc, tt.expected, intObj.Value)
		}
	}
}

// TestIfBlockScoping verifies that if/else blocks have their own scope:
// declarations don't leak out, but outer variables remain visible and
// assignable from inside the block.
func TestIfBlockScoping(t *testing.T) {
	// Declarations inside an if block must not leak
	result := testEval("if true {\n    let inner = 1\n}\ninner")
	if err, ok := result.(*Error); !ok {
		t.Errorf("expected identifier-not-found error, got %s (%s)", result.Type(), result.Inspect())
	} else if !strings.Contains(err.Message, "inner") {
		t.Errorf("expected error about 'inner', got: %s", err.Message)
	}

	// Outer variables are visible and assignable from inside the block
	result = testEval("var count = 0\nif true {\n    count = count + 1\n}\ncount")
	if intObj, ok := result.(*Integer); !ok || intObj.Value != 1 {
		t.Errorf("expected 1, got %s", result.Inspect())
	}
}

// TestUnderscoreNeverConflicts verifies '_' can be bound repeatedly
func TestUnderscoreNeverConflicts(t *testing.T) {
	tests := []string{
		"let _ = 1\nlet _ = 2\n3",
		"let [_, _, a] = [1, 2, 3]\na",
	}
	for _, input := range tests {
		result := testEval(input)
		if err, ok := result.(*Error); ok {
			t.Errorf("unexpected error for %q: %s", input, err.Inspect())
		}
	}
}

// TestAllowRedeclareInREPL verifies the REPL carve-out: with AllowRedeclare set
// on the top-level environment, redeclaration works and 'var x' after 'let x'
// is genuinely mutable. Enclosed scopes (function bodies) do not inherit it.
func TestAllowRedeclareInREPL(t *testing.T) {
	env := NewEnvironment()
	env.AllowRedeclare = true

	evalInEnv(t, "let x = 1", env)
	result := evalInEnv(t, "let x = 2", env)
	if err, ok := result.(*Error); ok {
		t.Fatalf("REPL redeclaration should be allowed, got: %s", err.Inspect())
	}
	if intObj, ok := evalInEnv(t, "x", env).(*Integer); !ok || intObj.Value != 2 {
		t.Errorf("expected x == 2 after REPL redeclaration")
	}

	// var after let must clear immutability
	evalInEnv(t, "var x = 3", env)
	result = evalInEnv(t, "x = 4", env)
	if err, ok := result.(*Error); ok {
		t.Fatalf("x should be mutable after 'var x' redeclaration, got: %s", err.Inspect())
	}
	if intObj, ok := evalInEnv(t, "x", env).(*Integer); !ok || intObj.Value != 4 {
		t.Errorf("expected x == 4 after reassignment")
	}

	// let after var restores immutability
	evalInEnv(t, "let x = 5", env)
	result = evalInEnv(t, "x = 6", env)
	if err, ok := result.(*Error); !ok || err.Code != "ASSIGN-0001" {
		t.Errorf("expected ASSIGN-0001 after 'let x' redeclaration, got: %s", result.Inspect())
	}

	// Function bodies evaluated under a REPL env still enforce the rule
	result = evalInEnv(t, "let f = fn() {\n    let y = 1\n    let y = 2\n}\nf()", env)
	if err, ok := result.(*Error); !ok || err.Code != "DECL-0001" {
		t.Errorf("expected DECL-0001 inside function body, got: %s", result.Inspect())
	}
}
