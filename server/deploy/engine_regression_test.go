package deploy

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A dangling `current` - the link still names the sha but the release
// directory was removed by hand - must be repaired by deploying the same
// sha, not declared a no-op that leaves the site down.
func TestDeployRepairsDanglingCurrent(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)

	if _, err := e.Deploy(f.good1); err != nil {
		t.Fatalf("deploying good1: %v", err)
	}
	releaseDir := filepath.Join(f.releasesDir(), f.good1)
	if err := os.RemoveAll(releaseDir); err != nil {
		t.Fatal(err)
	}

	res, err := e.Deploy(f.good1)
	if err != nil {
		t.Fatalf("re-deploying good1 over a dangling current: %v", err)
	}
	if res.Outcome != OutcomeDeployed {
		t.Errorf("Outcome = %q, want %q (a dangling current is not a no-op)", res.Outcome, OutcomeDeployed)
	}
	if info, err := os.Stat(releaseDir); err != nil || !info.IsDir() {
		t.Errorf("release directory was not re-materialised: %v", err)
	}
	if got := currentSHA(t, f.siteRoot); got != f.good1 {
		t.Errorf("current = %q, want %q", got, f.good1)
	}
}

// installRecordWriteFailure makes every INSERT into the deploy record fail,
// modelling a record that cannot be written (full disk, permissions) while
// the rest of the pipeline works.
func installRecordWriteFailure(t *testing.T, recordPath string) {
	t.Helper()
	db, err := sql.Open("sqlite", recordPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TRIGGER refuse_insert BEFORE INSERT ON deploys
		BEGIN SELECT RAISE(ABORT, 'record write refused by test'); END;`); err != nil {
		t.Fatalf("installing failing trigger: %v", err)
	}
}

// A deploy whose activation succeeded but whose record write failed must
// report the release as live - Result returned, error typed - never as a
// failed deploy.
func TestDeploySucceededButRecordWriteFailed(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)

	if _, err := e.Deploy(f.good1); err != nil {
		t.Fatalf("deploying good1: %v", err)
	}
	installRecordWriteFailure(t, e.RecordPath)

	res, err := e.Deploy(f.good2)
	if err == nil {
		t.Fatal("expected an error for the failed record write")
	}
	var rfErr *RecordFailedError
	if !errors.As(err, &rfErr) {
		t.Fatalf("error = %T (%v), want *RecordFailedError", err, err)
	}
	if res == nil {
		t.Fatal("Result must be returned alongside RecordFailedError: the release IS live")
	}
	if res.Outcome != OutcomeDeployed {
		t.Errorf("Outcome = %q, want %q", res.Outcome, OutcomeDeployed)
	}
	if got := currentSHA(t, f.siteRoot); got != f.good2 {
		t.Errorf("current = %q, want %q: the release must be live despite the record failure", got, f.good2)
	}
	if msg := err.Error(); !strings.Contains(msg, "is live") || strings.Contains(msg, "deploy failed") {
		t.Errorf("error message must state the release is live without claiming the deploy failed: %q", msg)
	}
}

// Rollback with a failing record write: same contract as Deploy.
func TestRollbackSucceededButRecordWriteFailed(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)

	if _, err := e.Deploy(f.good1); err != nil {
		t.Fatalf("deploying good1: %v", err)
	}
	if _, err := e.Deploy(f.good2); err != nil {
		t.Fatalf("deploying good2: %v", err)
	}
	installRecordWriteFailure(t, e.RecordPath)

	res, err := e.Rollback("")
	var rfErr *RecordFailedError
	if !errors.As(err, &rfErr) {
		t.Fatalf("error = %T (%v), want *RecordFailedError", err, err)
	}
	if res == nil || res.Outcome != OutcomeRolledBack {
		t.Fatalf("Result = %+v, want a rolled-back result alongside the error", res)
	}
	if got := currentSHA(t, f.siteRoot); got != f.good1 {
		t.Errorf("current = %q, want %q: the rollback must be in effect", got, f.good1)
	}
}

// Prune must never remove the previously activated release, even when its
// mtime makes it the oldest candidate and keep would otherwise discard it:
// a lagging server process may still be serving it, and it is what rollback
// rolls back to.
func TestDeployPruneProtectsPreviousActivatedRelease(t *testing.T) {
	f := newEngineFixture(t)
	e := f.engine(nil)
	e.Keep = 1 // clamped to 2 at prune time; deliberately hostile

	if _, err := e.Deploy(f.good1); err != nil {
		t.Fatalf("deploying good1: %v", err)
	}
	if _, err := e.Deploy(f.good2); err != nil {
		t.Fatalf("deploying good2: %v", err)
	}
	// Roll back so good1 - the OLDEST directory by mtime - is the most
	// recently activated release besides what the next deploy activates.
	if _, err := e.Rollback(""); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	res, err := e.Deploy(f.good3)
	if err != nil {
		t.Fatalf("deploying good3: %v", err)
	}
	// good1 is beyond keep by age but is the previous activated release:
	// it must survive for rollback.
	if _, statErr := os.Stat(filepath.Join(f.releasesDir(), f.good1)); statErr != nil {
		t.Errorf("previous activated release %s was pruned: %v (pruned: %v)", shortSHA(f.good1), statErr, res.Pruned)
	}
	if got := currentSHA(t, f.siteRoot); got != f.good3 {
		t.Errorf("current = %q, want %q", got, f.good3)
	}
	// And rollback still works, proving the protection is meaningful.
	if _, err := e.Rollback(""); err != nil {
		t.Fatalf("rollback after prune: %v", err)
	}
	if got := currentSHA(t, f.siteRoot); got != f.good1 {
		t.Errorf("current after rollback = %q, want %q", got, f.good1)
	}
}

// keep: 1 is clamped to 2 so a site is never one deploy away from having
// only a single release on disk.
func TestPruneClampsKeepToTwo(t *testing.T) {
	root := siteRootWithReleases(t, "r1", "r2", "r3")
	releasesDir := filepath.Join(root, "releases")

	base := time.Now().Add(-time.Hour)
	for i, name := range []string{"r1", "r2", "r3"} {
		ts := base.Add(time.Duration(i) * time.Minute)
		if err := os.Chtimes(filepath.Join(releasesDir, name), ts, ts); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := Prune(releasesDir, 1, filepath.Join(releasesDir, "r3"))
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if len(removed) != 1 || filepath.Base(removed[0]) != "r1" {
		t.Errorf("Prune(keep=1) removed %v, want exactly r1 (keep clamped to 2)", removed)
	}
	for _, name := range []string{"r2", "r3"} {
		if _, err := os.Stat(filepath.Join(releasesDir, name)); err != nil {
			t.Errorf("%s should have survived the clamped prune: %v", name, err)
		}
	}
}
