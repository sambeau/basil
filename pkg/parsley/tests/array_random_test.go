package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func evalRandomTest(input string) evaluator.Object {
	l := lexer.New(input)
	p := parser.New(l)
	env := evaluator.NewEnvironment()
	program := p.ParseProgram()
	return evaluator.Eval(program, env)
}

// TestShuffleMethod tests array.shuffle()
func TestShuffleMethod(t *testing.T) {
	t.Run("shuffle returns array of same length", func(t *testing.T) {
		result := evalRandomTest(`[1,2,3,4,5].shuffle().length()`)
		if result.Inspect() != "5" {
			t.Errorf("Expected length 5, got %s", result.Inspect())
		}
	})

	t.Run("shuffle empty array", func(t *testing.T) {
		result := evalRandomTest(`[].shuffle()`)
		if result.Inspect() != "[]" {
			t.Errorf("Expected [], got %s", result.Inspect())
		}
	})

	t.Run("shuffle single element", func(t *testing.T) {
		result := evalRandomTest(`[42].shuffle()`)
		if result.Inspect() != "[42]" {
			t.Errorf("Expected [42], got %s", result.Inspect())
		}
	})

	t.Run("shuffle does not modify original", func(t *testing.T) {
		// Test in multiple statements to verify shuffle returns new array
		l := lexer.New("let arr = [1,2,3]\narr.shuffle()\narr")
		p := parser.New(l)
		env := evaluator.NewEnvironment()
		program := p.ParseProgram()

		// Execute all statements
		evaluator.Eval(program, env)

		// Check that original array is unchanged
		arr, ok := env.Get("arr")
		if !ok {
			t.Error("arr not found in environment")
			return
		}
		if arr.Inspect() != "[1, 2, 3]" {
			t.Errorf("Original array was modified: %s", arr.Inspect())
		}
	})

	t.Run("shuffle contains same elements", func(t *testing.T) {
		// Run multiple times to verify elements are preserved
		for i := 0; i < 10; i++ {
			result := evalRandomTest(`[1,2,3].shuffle().sort()`)
			if result.Inspect() != "[1, 2, 3]" {
				t.Errorf("Shuffled array has different elements: %s", result.Inspect())
			}
		}
	})

	t.Run("shuffle with wrong args", func(t *testing.T) {
		result := evalRandomTest(`[1,2,3].shuffle(1)`)
		if !isError(result) {
			t.Errorf("Expected error for shuffle with args, got %s", result.Inspect())
		}
	})
}

// TestPickMethod tests array.pick() and array.pick(n)
func TestPickMethod(t *testing.T) {
	t.Run("pick from array returns element", func(t *testing.T) {
		// Run multiple times - should always return one of the elements
		for i := 0; i < 20; i++ {
			result := evalRandomTest(`[1,2,3].pick()`)
			val := result.Inspect()
			if val != "1" && val != "2" && val != "3" {
				t.Errorf("pick() returned unexpected value: %s", val)
			}
		}
	})

	t.Run("pick from empty array returns null", func(t *testing.T) {
		result := evalRandomTest(`[].pick()`)
		if result.Inspect() != "null" {
			t.Errorf("Expected null, got %s", result.Inspect())
		}
	})

	t.Run("pick(n) returns array of n elements", func(t *testing.T) {
		result := evalRandomTest(`[1,2,3].pick(5).length()`)
		if result.Inspect() != "5" {
			t.Errorf("Expected length 5, got %s", result.Inspect())
		}
	})

	t.Run("pick(0) returns empty array", func(t *testing.T) {
		result := evalRandomTest(`[1,2,3].pick(0)`)
		if result.Inspect() != "[]" {
			t.Errorf("Expected [], got %s", result.Inspect())
		}
	})

	t.Run("pick(n) can exceed array length", func(t *testing.T) {
		result := evalRandomTest(`[1,2].pick(10).length()`)
		if result.Inspect() != "10" {
			t.Errorf("Expected length 10, got %s", result.Inspect())
		}
	})

	t.Run("pick(n) elements are from original array", func(t *testing.T) {
		// All picked elements should be a, b, or c
		for i := 0; i < 10; i++ {
			result := evalRandomTest(`
				let picked = ["a","b","c"].pick(5)
				picked.filter(fn(x) { x == "a" || x == "b" || x == "c" }).length() == 5
			`)
			if result.Inspect() != "true" {
				t.Errorf("pick(n) returned elements not in original array")
			}
		}
	})

	t.Run("pick does not modify original", func(t *testing.T) {
		l := lexer.New("let arr = [1,2,3]\narr.pick(2)\narr")
		p := parser.New(l)
		env := evaluator.NewEnvironment()
		program := p.ParseProgram()

		// Execute all statements
		evaluator.Eval(program, env)

		// Check that original array is unchanged
		arr, ok := env.Get("arr")
		if !ok {
			t.Error("arr not found in environment")
			return
		}
		if arr.Inspect() != "[1, 2, 3]" {
			t.Errorf("Original array was modified: %s", arr.Inspect())
		}
	})

	t.Run("pick with negative n errors", func(t *testing.T) {
		result := evalRandomTest(`[1,2,3].pick(-1)`)
		if !isError(result) {
			t.Errorf("Expected error for negative n, got %s", result.Inspect())
		}
	})

	t.Run("pick with non-integer errors", func(t *testing.T) {
		result := evalRandomTest(`[1,2,3].pick("two")`)
		if !isError(result) {
			t.Errorf("Expected error for non-integer, got %s", result.Inspect())
		}
	})

	t.Run("pick(n) from empty array errors", func(t *testing.T) {
		result := evalRandomTest(`[].pick(1)`)
		if !isError(result) {
			t.Errorf("Expected error for pick from empty array, got %s", result.Inspect())
		}
	})
}

// TestTakeMethod tests array.take(n)
func TestTakeMethod(t *testing.T) {
	t.Run("take(n) returns n unique elements", func(t *testing.T) {
		result := evalRandomTest(`[1,2,3,4,5].take(3).length()`)
		if result.Inspect() != "3" {
			t.Errorf("Expected length 3, got %s", result.Inspect())
		}
	})

	t.Run("take(0) returns empty array", func(t *testing.T) {
		result := evalRandomTest(`[1,2,3].take(0)`)
		if result.Inspect() != "[]" {
			t.Errorf("Expected [], got %s", result.Inspect())
		}
	})

	t.Run("take(n) where n == length returns all elements shuffled", func(t *testing.T) {
		// Should return all elements, just shuffled
		for i := 0; i < 10; i++ {
			result := evalRandomTest(`[1,2,3].take(3).sort()`)
			if result.Inspect() != "[1, 2, 3]" {
				t.Errorf("take(3) from [1,2,3] should contain all elements: %s", result.Inspect())
			}
		}
	})

	t.Run("take(n) elements are unique", func(t *testing.T) {
		// Take 3 from array of 5 - when sorted, should have 3 distinct values
		for i := 0; i < 20; i++ {
			result := evalRandomTest(`
				let taken = [1,2,3,4,5].take(3)
				// If all unique, sorting and checking adjacent pairs should show no duplicates
				let sorted = taken.sort()
				sorted[0] != sorted[1] && sorted[1] != sorted[2]
			`)
			if result.Inspect() != "true" {
				t.Errorf("take(n) returned duplicate elements")
			}
		}
	})

	t.Run("take does not modify original", func(t *testing.T) {
		l := lexer.New("let arr = [1,2,3]\narr.take(2)\narr")
		p := parser.New(l)
		env := evaluator.NewEnvironment()
		program := p.ParseProgram()

		// Execute all statements
		evaluator.Eval(program, env)

		// Check that original array is unchanged
		arr, ok := env.Get("arr")
		if !ok {
			t.Error("arr not found in environment")
			return
		}
		if arr.Inspect() != "[1, 2, 3]" {
			t.Errorf("Original array was modified: %s", arr.Inspect())
		}
	})

	t.Run("take(n) where n > length errors", func(t *testing.T) {
		result := evalRandomTest(`[1,2,3].take(5)`)
		if !isError(result) {
			t.Errorf("Expected error for n > length, got %s", result.Inspect())
		}
	})

	t.Run("take with negative n errors", func(t *testing.T) {
		result := evalRandomTest(`[1,2,3].take(-1)`)
		if !isError(result) {
			t.Errorf("Expected error for negative n, got %s", result.Inspect())
		}
	})

	t.Run("take with non-integer errors", func(t *testing.T) {
		result := evalRandomTest(`[1,2,3].take("two")`)
		if !isError(result) {
			t.Errorf("Expected error for non-integer, got %s", result.Inspect())
		}
	})

	t.Run("take without argument errors", func(t *testing.T) {
		result := evalRandomTest(`[1,2,3].take()`)
		if !isError(result) {
			t.Errorf("Expected error for missing argument, got %s", result.Inspect())
		}
	})

	t.Run("take(0) from empty array succeeds", func(t *testing.T) {
		result := evalRandomTest(`[].take(0)`)
		if result.Inspect() != "[]" {
			t.Errorf("Expected [], got %s", result.Inspect())
		}
	})

	t.Run("take(1) from empty array errors", func(t *testing.T) {
		result := evalRandomTest(`[].take(1)`)
		if !isError(result) {
			t.Errorf("Expected error for take from empty array, got %s", result.Inspect())
		}
	})
}

// isError checks if the object is an error
func isError(obj evaluator.Object) bool {
	return obj != nil && obj.Type() == "ERROR"
}
