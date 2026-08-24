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

	last, err := rec.LastDeployed()
	if err != nil {
		t.Fatalf("LastDeployed on empty record: %v", err)
	}
	if last != nil {
		t.Fatalf("LastDeployed on empty record = %+v, want nil", last)
	}

	for _, outcome := range []string{OutcomeRejected, OutcomeFailed, OutcomeNoOp} {
		if err := rec.Add(testEntry("bad", outcome)); err != nil {
			t.Fatalf("Add(%s): %v", outcome, err)
		}
	}
	if last, err = rec.LastDeployed(); err != nil || last != nil {
		t.Fatalf("LastDeployed with only non-activations = %+v, %v; want nil, nil", last, err)
	}

	if err := rec.Add(testEntry("aaa", OutcomeDeployed)); err != nil {
		t.Fatal(err)
	}
	if err := rec.Add(testEntry("zzz", OutcomeFailed)); err != nil {
		t.Fatal(err)
	}
	last, err = rec.LastDeployed()
	if err != nil {
		t.Fatalf("LastDeployed: %v", err)
	}
	if last == nil || last.CommitSHA != "aaa" {
		t.Fatalf("LastDeployed = %+v, want the deployed aaa, not the failed zzz", last)
	}

	// A rollback activates a release too: it must be what the next rollback
	// rolls back from.
	if err := rec.Add(testEntry("bbb", OutcomeRolledBack)); err != nil {
		t.Fatal(err)
	}
	last, err = rec.LastDeployed()
	if err != nil {
		t.Fatalf("LastDeployed: %v", err)
	}
	if last == nil || last.CommitSHA != "bbb" {
		t.Fatalf("LastDeployed = %+v, want the rolled-back-to bbb", last)
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
