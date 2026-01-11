package evaluator

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestEnvGlobal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType ObjectType
	}{
		{"@env returns dictionary", "@env", DICTIONARY_OBJ},
		{"@env.HOME returns string", "@env.HOME", STRING_OBJ},
		{"@env.PATH returns string", "@env.PATH", STRING_OBJ},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			prog := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := NewEnvironmentWithArgs(nil)
			result := Eval(prog, env)

			if result == nil {
				t.Fatal("result is nil")
			}

			if result.Type() == ERROR_OBJ {
				t.Fatalf("got error: %s", result.Inspect())
			}

			if result.Type() != tt.wantType {
				t.Errorf("wrong type: got %s, want %s", result.Type(), tt.wantType)
			}
		})
	}
}

func TestArgsGlobal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		args     []string
		wantType ObjectType
		wantVal  string
	}{
		{
			name:     "@args returns array",
			input:    "@args",
			args:     []string{"foo", "bar"},
			wantType: ARRAY_OBJ,
		},
		{
			name:     "@args[0] returns first arg",
			input:    "@args[0]",
			args:     []string{"foo", "bar"},
			wantType: STRING_OBJ,
			wantVal:  "foo",
		},
		{
			name:     "@args[1] returns second arg",
			input:    "@args[1]",
			args:     []string{"foo", "bar"},
			wantType: STRING_OBJ,
			wantVal:  "bar",
		},
		{
			name:     "@args.length() returns count",
			input:    "@args.length()",
			args:     []string{"a", "b", "c"},
			wantType: INTEGER_OBJ,
			wantVal:  "3",
		},
		{
			name:     "empty @args",
			input:    "@args.length()",
			args:     nil,
			wantType: INTEGER_OBJ,
			wantVal:  "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			prog := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			env := NewEnvironmentWithArgs(tt.args)
			result := Eval(prog, env)

			if result == nil {
				t.Fatal("result is nil")
			}

			if result.Type() == ERROR_OBJ {
				t.Fatalf("got error: %s", result.Inspect())
			}

			if result.Type() != tt.wantType {
				t.Errorf("wrong type: got %s, want %s", result.Type(), tt.wantType)
			}

			if tt.wantVal != "" && result.Inspect() != tt.wantVal {
				t.Errorf("wrong value: got %q, want %q", result.Inspect(), tt.wantVal)
			}
		})
	}
}

func TestArgsIntegration(t *testing.T) {
	// Test that args can be iterated
	input := `for (arg in @args) { arg }`
	args := []string{"hello", "world"}

	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	env := NewEnvironmentWithArgs(args)
	result := Eval(prog, env)

	if result.Type() == ERROR_OBJ {
		t.Fatalf("got error: %s", result.Inspect())
	}

	arr, ok := result.(*Array)
	if !ok {
		t.Fatalf("expected Array, got %T", result)
	}

	if len(arr.Elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(arr.Elements))
	}
}
