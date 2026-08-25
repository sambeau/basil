package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testEntry(sha, outcome string) Entry {
	return Entry{
		CommitSHA:   sha,
		Trigger:     TriggerCLI,
		Publisher:   "cli:sam",
		AuthorName:  "Sam Author",
		AuthorEmail: "sam@example.com",
		StartedAt:   time.Now().UTC(),
		Duration:    1234 * time.Millisecond,
		Outcome:     outcome,
	}
}

func TestRecordAddAndListNewestFirst(t *testing.T) {
	// The path's directory does not exist yet: OpenRecord must create it.
	path := filepath.Join(t.TempDir(), "data", "deploy.db")
	rec, err := OpenRecord(path)
	if err != nil {
		t.Fatalf("OpenRecord: %v", err)
	}
	defer rec.Close()

	for _, sha := range []string{"aaa", "bbb", "ccc"} {
		if err := rec.Add(testEntry(sha, OutcomeDeployed)); err != nil {
			t.Fatalf("Add(%s): %v", sha, err)
		}
	}

	entries, err := rec.List(0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("List returned %d entries, want 3", len(entries))
	}
	for i, want := range []string{"ccc", "bbb", "aaa"} {
		if entries[i].CommitSHA != want {
			t.Errorf("entries[%d].CommitSHA = %q, want %q (newest first)", i, entries[i].CommitSHA, want)
		}
	}
	if entries[0].Seq <= entries[1].Seq {
		t.Errorf("seq is not monotonic: %d then %d", entries[0].Seq, entries[1].Seq)
	}

	limited, err := rec.List(2)
	if err != nil {
		t.Fatalf("List(2): %v", err)
	}
	if len(limited) != 2 || limited[0].CommitSHA != "ccc" {
		t.Errorf("List(2) = %d entries starting %q, want 2 starting ccc", len(limited), limited[0].CommitSHA)
	}

	// Fields must survive the round trip - the record is only useful if it
	// still answers who/when/how-long later.
	got := entries[0]
	if got.Publisher != "cli:sam" || got.AuthorName != "Sam Author" || got.AuthorEmail != "sam@example.com" {
		t.Errorf("identities did not round-trip: %+v", got)
	}
	if got.Duration != 1234*time.Millisecond {
		t.Errorf("Duration = %v, want 1.234s", got.Duration)
	}
	if got.StartedAt.IsZero() {
		t.Error("StartedAt did not round-trip")
	}
}

func TestRecordFailuresAreRecordedWithTheirReason(t *testing.T) {
	rec, err := OpenRecord(filepath.Join(t.TempDir(), "deploy.db"))
	if err != nil {
		t.Fatalf("OpenRecord: %v", err)
	}
	defer rec.Close()

	e := testEntry("aaa", OutcomeRejected)
	e.Reason = "site/index.pars:3: unexpected token"
	if err := rec.Add(e); err != nil {
		t.Fatalf("Add: %v", err)
	}

	entries, err := rec.List(1)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if entries[0].Outcome != OutcomeRejected || entries[0].Reason != e.Reason {
		t.Errorf("failure did not round-trip: %+v", entries[0])
	}
}

func TestRecordAddRejectsIncompleteEntries(t *testing.T) {
	rec, err := OpenRecord(filepath.Join(t.TempDir(), "deploy.db"))
	if err != nil {
		t.Fatalf("OpenRecord: %v", err)
	}
	defer rec.Close()

	if err := rec.Add(testEntry("", OutcomeDeployed)); err == nil {
		t.Error("Add without a commit SHA was accepted")
	}
	if err := rec.Add(testEntry("aaa", "")); err == nil {
		t.Error("Add without an outcome was accepted")
	}
}

func TestLastDeployedSkipsEverythingThatDidNotActivate(t *testing.T) {
	rec, err := OpenRecord(filepath.Join(t.TempDir(), "deploy.db"))
	if err != nil {
		t.Fatalf("OpenRecord: %v", err)
	}
	defer rec.Close()

	last, err := rec.lastDeployed()
	if err != nil {
		t.Fatalf("lastDeployed on empty record: %v", err)
	}
	if last != nil {
		t.Fatalf("lastDeployed on empty record = %+v, want nil", last)
	}

	for _, outcome := range []string{OutcomeRejected, OutcomeFailed, OutcomeNoOp} {
		if err := rec.Add(testEntry("bad", outcome)); err != nil {
			t.Fatalf("Add(%s): %v", outcome, err)
		}
	}
	if last, err = rec.lastDeployed(); err != nil || last != nil {
		t.Fatalf("lastDeployed with only non-activations = %+v, %v; want nil, nil", last, err)
	}

	if err := rec.Add(testEntry("aaa", OutcomeDeployed)); err != nil {
		t.Fatal(err)
	}
	if err := rec.Add(testEntry("zzz", OutcomeFailed)); err != nil {
		t.Fatal(err)
	}
	last, err = rec.lastDeployed()
	if err != nil {
		t.Fatalf("lastDeployed: %v", err)
	}
	if last == nil || last.CommitSHA != "aaa" {
		t.Fatalf("lastDeployed = %+v, want the deployed aaa, not the failed zzz", last)
	}

	// A rollback activates a release too: it must be what the next rollback
	// rolls back from.
	if err := rec.Add(testEntry("bbb", OutcomeRolledBack)); err != nil {
		t.Fatal(err)
	}
	last, err = rec.lastDeployed()
	if err != nil {
		t.Fatalf("lastDeployed: %v", err)
	}
	if last == nil || last.CommitSHA != "bbb" {
		t.Fatalf("lastDeployed = %+v, want the rolled-back-to bbb", last)
	}
}

// The CLI and a running server open the same record concurrently; WAL and the
// busy timeout exist for exactly this.
func TestRecordTwoHandlesShareOneDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deploy.db")

	a, err := OpenRecord(path)
	if err != nil {
		t.Fatalf("OpenRecord (first handle): %v", err)
	}
	defer a.Close()
	b, err := OpenRecord(path)
	if err != nil {
		t.Fatalf("OpenRecord (second handle): %v", err)
	}
	defer b.Close()

	if err := a.Add(testEntry("from-a", OutcomeDeployed)); err != nil {
		t.Fatalf("Add via first handle: %v", err)
	}
	if err := b.Add(testEntry("from-b", OutcomeDeployed)); err != nil {
		t.Fatalf("Add via second handle: %v", err)
	}

	entries, err := a.List(0)
	if err != nil {
		t.Fatalf("List via first handle: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("first handle sees %d entries, want 2", len(entries))
	}
}

func TestOpenRecordFailsWhenThePathIsUnusable(t *testing.T) {
	// A regular file where a directory component should be.
	dir := t.TempDir()
	blocker := filepath.Join(dir, "data")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := OpenRecord(filepath.Join(blocker, "deploy.db"))
	if err == nil {
		t.Fatal("expected an error when the parent directory cannot be created")
	}
	if !strings.Contains(err.Error(), "deploy record") {
		t.Errorf("the error does not say what failed: %v", err)
	}
}

// A corrupt started_at must not vanish silently: the entry keeps the zero
// time (visibly wrong beats plausibly wrong) and the raw string surfaces in
// Reason, where basil releases prints it.
func TestScanEntrySurfacesCorruptStartedAt(t *testing.T) {
	rec, err := OpenRecord(filepath.Join(t.TempDir(), "deploy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer rec.Close()
	if err := rec.Add(testEntry("aaa", OutcomeDeployed)); err != nil {
		t.Fatal(err)
	}
	if _, err := rec.db.Exec(`UPDATE deploys SET started_at = 'last tuesday'`); err != nil {
		t.Fatal(err)
	}

	entries, err := rec.List(0)
	if err != nil {
		t.Fatalf("List over a corrupt timestamp: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	e := entries[0]
	if !e.StartedAt.IsZero() {
		t.Errorf("StartedAt = %v, want the zero time for a corrupt value", e.StartedAt)
	}
	if !strings.Contains(e.Reason, `"last tuesday"`) || !strings.Contains(e.Reason, "started_at") {
		t.Errorf("Reason = %q, want it to carry the raw started_at string", e.Reason)
	}
}

// --- OnlyInitDeployed (the graduation exception's one question) ------------
//
// A true here is permission to rewrite the release branch, so every uncertain
// answer must be false.

// initEntry is a release-1 row as `basil --init --server` seeds it.
func initEntry(sha string) Entry {
	e := testEntry(sha, OutcomeDeployed)
	e.Trigger = TriggerInit
	e.Publisher = "init"
	return e
}

// writeRecord builds a record at a fresh path and fills it with entries.
func writeRecord(t *testing.T, entries ...Entry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data", "deploy.db")
	rec, err := OpenRecord(path)
	if err != nil {
		t.Fatalf("OpenRecord: %v", err)
	}
	defer rec.Close()
	for _, e := range entries {
		if err := rec.Add(e); err != nil {
			t.Fatalf("Add(%s): %v", e.CommitSHA, err)
		}
	}
	return path
}

func TestOnlyInitDeployed(t *testing.T) {
	cases := []struct {
		name    string
		entries []Entry
		want    bool
	}{
		{
			// The state graduation happens in: the hub has published nothing
			// a human wrote.
			name:    "the init release alone",
			entries: []Entry{initEntry("aaa")},
			want:    true,
		},
		{
			// Several init rows can only come from repeated seeding, which is
			// still a hub that has published nothing. Intended, not tolerated.
			name:    "several init releases",
			entries: []Entry{initEntry("aaa"), initEntry("bbb")},
			want:    true,
		},
		{
			// An empty table is not evidence of freshness: release 1 is always
			// recorded, so an empty record is not the one this site deployed
			// through.
			name:    "an empty record",
			entries: nil,
			want:    false,
		},
		{
			name:    "one real deploy spends the exception",
			entries: []Entry{initEntry("aaa"), testEntry("bbb", OutcomeDeployed)},
			want:    false,
		},
		{
			// Even a deploy that FAILED is history: someone has pushed here.
			name:    "a failed deploy still spends it",
			entries: []Entry{initEntry("aaa"), testEntry("bbb", OutcomeFailed)},
			want:    false,
		},
		{
			// Order does not matter: the question is about the whole table.
			name:    "a real deploy before the init row",
			entries: []Entry{testEntry("bbb", OutcomeDeployed), initEntry("aaa")},
			want:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := OnlyInitDeployed(writeRecord(t, tc.entries...))
			if err != nil {
				t.Fatalf("OnlyInitDeployed: %v", err)
			}
			if got != tc.want {
				t.Errorf("OnlyInitDeployed = %v, want %v", got, tc.want)
			}
		})
	}
}

// A record that is not there is not an exception — and asking must not create
// one, or the next caller would find an empty record where there was no file.
func TestOnlyInitDeployedMissingRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "deploy.db")

	got, err := OnlyInitDeployed(path)
	if err == nil {
		t.Fatal("a missing record was not reported as unreadable")
	}
	if got {
		t.Error("a missing record granted the exception")
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Errorf("asking about a missing record created one (stat err = %v)", statErr)
	}
}

// A file that is not a deploy record answers false with the reason, never
// "probably fresh".
func TestOnlyInitDeployedCorruptRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deploy.db")
	if err := os.WriteFile(path, []byte("this is not a database\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := OnlyInitDeployed(path)
	if err == nil {
		t.Fatal("a corrupt record was accepted as readable")
	}
	if got {
		t.Error("a corrupt record granted the exception")
	}
}
