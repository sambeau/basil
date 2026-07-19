package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
)

// TestRecursionLimitCaught verifies that unbounded recursion is converted into a
// catchable Parsley error rather than overflowing the Go stack and crashing the
// process. See CALL-0007 and evaluator.MaxCallDepth.
func TestRecursionLimitCaught(t *testing.T) {
	src := `
let loop = fn(n) { loop(n + 1) }
loop(0)
`
	_, err := parsley.Eval(src)
	if err == nil {
		t.Fatal("expected a runtime error from unbounded recursion, got nil")
	}
	if !strings.Contains(err.Error(), "call depth") {
		t.Fatalf("expected a 'maximum call depth exceeded' error, got: %v", err)
	}
}

// TestBoundedRecursionSucceeds verifies that legitimate recursion well within the
// limit still evaluates correctly.
func TestBoundedRecursionSucceeds(t *testing.T) {
	src := `
let sum = fn(n) { if n == 0 { 0 } else { n + sum(n - 1) } }
sum(1000)
`
	result, err := parsley.Eval(src)
	if err != nil {
		t.Fatalf("bounded recursion should succeed, got error: %v", err)
	}
	intObj, ok := result.Value.(*evaluator.Integer)
	if !ok {
		t.Fatalf("expected an Integer result, got %T", result.Value)
	}
	if intObj.Value != 500500 {
		t.Fatalf("expected sum(1000) == 500500, got %d", intObj.Value)
	}
}

// TestRecursionLimitReleasesDepth verifies that the shared depth counter is
// decremented on return, so sequential (non-nested) calls do not accumulate and
// exhaust the limit.
func TestRecursionLimitReleasesDepth(t *testing.T) {
	src := `
let f = fn(n) { if n == 0 { 0 } else { f(n - 1) } }
let total = fn(i, acc) { if i == 0 { acc } else { total(i - 1, acc + f(100)) } }
total(200, 0)
`
	result, err := parsley.Eval(src)
	if err != nil {
		t.Fatalf("sequential calls should not accumulate depth, got error: %v", err)
	}
	if intObj, ok := result.Value.(*evaluator.Integer); !ok || intObj.Value != 0 {
		t.Fatalf("expected 0, got %v", result.Value)
	}
}
