package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
)

// divergentZone returns a fixed zone in which the local date is guaranteed to
// differ from the UTC date right now, whatever the time of day.
//
// TestTimeOnlyUsesCurrentDate found BUG-044 only between local midnight and UTC
// midnight — one hour a night in BST, and nothing at all for a developer in
// UTC. A bug that is reproducible for one hour a day is one that gets rebased
// past, so this pins the zone instead of waiting for the clock.
func divergentZone(t *testing.T) *time.Location {
	t.Helper()
	h := time.Now().UTC().Hour()
	offset := -(h + 1) // push local back a day; valid for h <= 13
	if h >= 14 {
		offset = 24 - h // push local forward a day; valid for h >= 14, so <= 10
	}
	return time.FixedZone("BUG044", offset*3600)
}

// withLocal runs fn with time.Local set to loc.
func withLocal(t *testing.T, loc *time.Location, fn func()) {
	t.Helper()
	saved := time.Local
	time.Local = loc
	defer func() { time.Local = saved }()
	fn()
}

func intOf(t *testing.T, obj evaluator.Object, what string) int64 {
	t.Helper()
	i, ok := obj.(*evaluator.Integer)
	if !ok {
		t.Fatalf("%s: expected Integer, got %T (%v)", what, obj, obj)
	}
	return i.Value
}

// TestDatetime_LocalDateIsConsistent guards BUG-044.
//
// Every way of asking a datetime what day it is must give the same answer. Two
// did not: a time-only literal was dated to the UTC day while @now was dated to
// the local one, and formatting re-derived the calendar fields in UTC, so a
// value reporting .day 27, .weekday "Thursday" and .iso 2026-08-27T00:32+01:00
// printed as "Wednesday, August 26, 2026".
func TestDatetime_LocalDateIsConsistent(t *testing.T) {
	withLocal(t, divergentZone(t), func() {
		nowDay := intOf(t, testEvalKind(`@now.day`), "@now.day")

		// A time-only literal is dated to today — the same today.
		if got := intOf(t, testEvalKind(`@12:30.day`), "@12:30.day"); got != nowDay {
			t.Errorf("@12:30.day = %d, @now.day = %d: a time literal must be dated to the same day @now reports", got, nowDay)
		}
		if got := intOf(t, testEvalKind(`@timeNow.day`), "@timeNow.day"); got != nowDay {
			t.Errorf("@timeNow.day = %d, @now.day = %d", got, nowDay)
		}

		// Formatting must render the day the value reports, not a re-derivation
		// of it in another zone.
		weekday, ok := testEvalKind(`@now.weekday`).(*evaluator.String)
		if !ok {
			t.Fatal("@now.weekday did not evaluate to a string")
		}
		full, ok := testEvalKind(`@now.fmt("full")`).(*evaluator.String)
		if !ok {
			t.Fatal(`@now.fmt("full") did not evaluate to a string`)
		}
		if !strings.Contains(full.Value, weekday.Value) {
			t.Errorf("@now.fmt(\"full\") = %q but @now.weekday = %q: formatting disagrees with the value it formats",
				full.Value, weekday.Value)
		}
	})
}
