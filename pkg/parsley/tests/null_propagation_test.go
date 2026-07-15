package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// FEAT-150: Consistent null propagation for index access.
//
// Access rule: absence yields null; an out-of-range position is an error.
// These tests pin the new behaviour (a null receiver propagates through
// bracket/slice access, mirroring dot access) and guard the behaviour that
// must NOT change (out-of-range and type-mismatch still error).

// TestNullReceiverPropagation: indexing or slicing into null yields null,
// exactly as dot access already does — so bracket chains survive a missing
// intermediate the way dot chains do.
func TestNullReceiverPropagation(t *testing.T) {
	inputs := []string{
		`null["x"]`,
		`null[0]`,
		`null[?"x"]`,
		`null[1:2]`,
		// chained bracket access through a missing intermediate
		`let u = {name: "a"}; u["profile"]["avatar"]`,
		// mixed dot + bracket chains all agree
		`let u = {name: "a"}; u["profile"].avatar`,
		`let u = {name: "a"}; u.profile["avatar"]`,
		`let u = {name: "a"}; u.profile.avatar`,
		// deeper chain
		`let u = {name: "a"}; u["a"]["b"]["c"]`,
	}
	for _, input := range inputs {
		result := testEvalOptionalIndex(input)
		if result.Type() != evaluator.NULL_OBJ {
			t.Errorf("For input '%s': expected NULL, got %T (%+v)", input, result, result)
		}
	}
}

// TestNullReceiverShortCircuit: a null receiver skips index evaluation entirely,
// matching dot access (which never evaluates the key on a null receiver). If the
// index were evaluated, fail()/an undefined identifier would surface an error;
// getting NULL proves the index sub-expression was not evaluated.
func TestNullReceiverShortCircuit(t *testing.T) {
	inputs := []string{
		`null[fail("index should not be evaluated")]`,
		`null[thisIdentifierIsNotDefined]`,
	}
	for _, input := range inputs {
		result := testEvalOptionalIndex(input)
		if result.Type() != evaluator.NULL_OBJ {
			t.Errorf("For input '%s': expected NULL (short-circuit), got %T (%+v)", input, result, result)
		}
	}
}

// TestOutOfRangeStillErrors: out-of-range indexing on a PRESENT container stays
// strict. Only absence (null receiver, missing dict key) is lenient.
func TestOutOfRangeStillErrors(t *testing.T) {
	inputs := []string{
		`[1, 2, 3][99]`,
		`[1, 2, 3][-99]`,
		`[][0]`,
		`"hello"[99]`,
		`""[0]`,
	}
	for _, input := range inputs {
		result := testEvalOptionalIndex(input)
		if result.Type() != evaluator.ERROR_OBJ {
			t.Errorf("For input '%s': expected ERROR (out of range), got %T (%+v)", input, result, result)
		}
	}
}

// TestTypeMismatchStillErrors: leniency applies only to a null receiver, never
// to a type mismatch — and [?] must not swallow a type mismatch either.
func TestTypeMismatchStillErrors(t *testing.T) {
	inputs := []string{
		`[1, 2, 3]["x"]`,  // string index on an array
		`5[0]`,            // index into an integer
		`true[0]`,         // index into a bool
		`[1, 2, 3][?"x"]`, // [?] does not turn a type mismatch into null
	}
	for _, input := range inputs {
		result := testEvalOptionalIndex(input)
		if result.Type() != evaluator.ERROR_OBJ {
			t.Errorf("For input '%s': expected ERROR (type mismatch), got %T (%+v)", input, result, result)
		}
	}
}

// TestAbsenceIsNull: missing dictionary keys remain null (unchanged) — absence
// on a present container, distinct from an out-of-range positional index.
func TestAbsenceIsNull(t *testing.T) {
	inputs := []string{
		`{a: 1}["missing"]`,
		`{a: 1}[?"missing"]`,
		`{}["any"]`,
	}
	for _, input := range inputs {
		result := testEvalOptionalIndex(input)
		if result.Type() != evaluator.NULL_OBJ {
			t.Errorf("For input '%s': expected NULL (absent key), got %T (%+v)", input, result, result)
		}
	}
}

// TestStrictAssertionIdiom: `?? fail("msg")` gives an inline, per-hop "must
// exist" assertion — the strict counterpart to forgiving-by-default access.
func TestStrictAssertionIdiom(t *testing.T) {
	// Present: the assertion passes and the chain continues.
	present := testEvalOptionalIndex(`let a = {b: {c: 42}}; (a["b"] ?? fail("b is required"))["c"]`)
	if intObj, ok := present.(*evaluator.Integer); !ok || intObj.Value != 42 {
		t.Errorf("expected 42 for present assertion, got %T (%+v)", present, present)
	}

	// Missing: the assertion fires, producing an error at that hop.
	missing := testEvalOptionalIndex(`let a = {}; (a["b"] ?? fail("b is required"))["c"]`)
	if missing.Type() != evaluator.ERROR_OBJ {
		t.Errorf("expected ERROR for missing assertion, got %T (%+v)", missing, missing)
	}

	// ?? short-circuits: a present value never evaluates the fail() branch.
	sc := testEvalOptionalIndex(`1 ?? fail("should not run")`)
	if intObj, ok := sc.(*evaluator.Integer); !ok || intObj.Value != 1 {
		t.Errorf("expected 1 (short-circuit), got %T (%+v)", sc, sc)
	}
}
