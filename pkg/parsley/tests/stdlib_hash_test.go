package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func evalHashTest(t *testing.T, input string) evaluator.Object {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	env := evaluator.NewEnvironment()
	return evaluator.Eval(program, env)
}

func TestHashImport(t *testing.T) {
	input := `
		import @std/hash
		hash.md5("hello")
	`
	result := evalHashTest(t, input)
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("unexpected error: %s", result.Inspect())
	}
	if result.Inspect() != `5d41402abc4b2a76b9719d911017c592` {
		t.Errorf("expected md5 hash, got %s", result.Inspect())
	}
}

func TestHashDestructuringImport(t *testing.T) {
	input := `
		let { md5, sha256 } = import @std/hash
		md5("test")
	`
	result := evalHashTest(t, input)
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("unexpected error: %s", result.Inspect())
	}
	if result.Inspect() != `098f6bcd4621d373cade4e832627b4f6` {
		t.Errorf("expected md5 hash of 'test', got %s", result.Inspect())
	}
}

func TestHashMD5(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"hello", `import @std/hash; hash.md5("hello")`, `5d41402abc4b2a76b9719d911017c592`},
		{"empty", `import @std/hash; hash.md5("")`, `d41d8cd98f00b204e9800998ecf8427e`},
		{"test", `import @std/hash; hash.md5("test")`, `098f6bcd4621d373cade4e832627b4f6`},
		{"unicode", `import @std/hash; hash.md5("日本語")`, `00110af8b4393ef3f72c50be5b332bec`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalHashTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestHashSHA1(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"hello", `import @std/hash; hash.sha1("hello")`, `aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d`},
		{"empty", `import @std/hash; hash.sha1("")`, `da39a3ee5e6b4b0d3255bfef95601890afd80709`},
		{"test", `import @std/hash; hash.sha1("test")`, `a94a8fe5ccb19ba61c4c0873d391e987982fbbd3`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalHashTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestHashSHA256(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"hello", `import @std/hash; hash.sha256("hello")`, `2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824`},
		{"empty", `import @std/hash; hash.sha256("")`, `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`},
		{"test", `import @std/hash; hash.sha256("test")`, `9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalHashTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestHashSHA512(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"hello", `import @std/hash; hash.sha512("hello")`, `9b71d224bd62f3785d96d46ad3ea3d73319bfbc2890caadae2dff72519673ca72323c3d99ba5c11d7c7acc6e14b8c5da0c4663475c2e5c3adef46f73bcdec043`},
		{"empty", `import @std/hash; hash.sha512("")`, `cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalHashTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

func TestHashArityErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"md5_no_args", `import @std/hash; hash.md5()`},
		{"md5_too_many", `import @std/hash; hash.md5("a", "b")`},
		{"sha1_no_args", `import @std/hash; hash.sha1()`},
		{"sha256_no_args", `import @std/hash; hash.sha256()`},
		{"sha512_no_args", `import @std/hash; hash.sha512()`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalHashTest(t, tt.input)
			if result.Type() != evaluator.ERROR_OBJ {
				t.Errorf("expected arity error, got %s", result.Inspect())
			}
		})
	}
}

func TestHashTypeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"md5_number", `import @std/hash; hash.md5(123)`},
		{"md5_array", `import @std/hash; hash.md5([1, 2, 3])`},
		{"md5_null", `import @std/hash; hash.md5(null)`},
		{"sha1_number", `import @std/hash; hash.sha1(456)`},
		{"sha256_dict", `import @std/hash; hash.sha256({a: 1})`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalHashTest(t, tt.input)
			if result.Type() != evaluator.ERROR_OBJ {
				t.Errorf("expected type error, got %s", result.Inspect())
			}
		})
	}
}
