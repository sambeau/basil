package deploy

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	// SQLite driver
	_ "modernc.org/sqlite"
)

// Trigger labels: what started a deploy.
const (
	TriggerCLI      = "cli"
	TriggerHook     = "hook"
	TriggerRollback = "rollback"
)

// Outcome labels. Stored as TEXT, not an enum: the record must be able to
// hold outcomes this version of Basil has never heard of, or a downgrade
// could not read its own history.
const (
	OutcomeDeployed   = "deployed"
	OutcomeRejected   = "rejected"
	OutcomeFailed     = "failed"
	OutcomeNoOp       = "no-op"
	OutcomeRolledBack = "rolled-back"
)

// Entry is one row of the deploy record. Every attempt is recorded -
// failures and rejections too - so the record answers "what happened",
// not just "what worked".
//
// Publisher and the author fields are deliberately both here (design
// §5.2.4): the Basil account that pushed the button and the person who
// wrote the commit routinely differ.
type Entry struct {
	Seq         int64         // Monotonic sequence, assigned by the record
	CommitSHA   string        // Full commit SHA (the release id)
	Trigger     string        // TriggerCLI, TriggerHook, TriggerRollback
	Publisher   string        // Basil account (or cli:<os-username>) that triggered the deploy
	AuthorName  string        // Commit author, from the commit itself
	AuthorEmail string        // Commit author email, from the commit itself
	StartedAt   time.Time     // When the deploy began
	Duration    time.Duration // How long it took, stored as milliseconds
	Outcome     string        // One of the Outcome constants (open set)
	Reason      string        // Failure or validation summary; empty on success
}

// recordSchema defines the deploy record table. `trigger` is an SQL keyword,
// so it is quoted everywhere it appears.
const recordSchema = `
CREATE TABLE IF NOT EXISTS deploys (
	seq INTEGER PRIMARY KEY AUTOINCREMENT,
	commit_sha TEXT NOT NULL,
	"trigger" TEXT NOT NULL,
	publisher TEXT NOT NULL DEFAULT '',
	author_name TEXT NOT NULL DEFAULT '',
	author_email TEXT NOT NULL DEFAULT '',
	started_at TIMESTAMP NOT NULL,
	duration_ms INTEGER NOT NULL DEFAULT 0,
	outcome TEXT NOT NULL,
	reason TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_deploys_outcome ON deploys(outcome);
`

// recordMigrations tracks schema migrations for existing databases, in
// order, following the server/auth pattern. Empty while the schema is new.
var recordMigrations = []string{}

// Record is the deploy record: an append-only log of every deploy attempt,
// stored in SQLite so the CLI can read it with the server stopped.
type Record struct {
	db   *sql.DB
	path string
}

// OpenRecord opens (creating if needed) the deploy record at path -
// normally cfg.DeployDBPath(). WAL and a busy timeout are set because the
// CLI and a running server open the same file concurrently.
func OpenRecord(path string) (*Record, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating deploy record directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening deploy record: %w", err)
	}
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("%s: %w", pragma, err)
		}
	}
	if _, err := db.Exec(recordSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating deploy record schema: %w", err)
	}
	for _, migration := range recordMigrations {
		// Migrations are idempotent (CREATE IF NOT EXISTS, column exists);
		// errors are ignored, as in server/auth.
		db.Exec(migration)
	}
	return &Record{db: db, path: path}, nil
}

// Add appends one entry. It is called on every outcome - a deploy that is
// not recorded did not happen, and a failure that is not recorded cannot be
// diagnosed. Entry.Seq is ignored; the database assigns it.
func (r *Record) Add(e Entry) error {
	if e.CommitSHA == "" {
		return fmt.Errorf("recording deploy: commit SHA is required")
	}
	if e.Outcome == "" {
		return fmt.Errorf("recording deploy: outcome is required")
	}
	_, err := r.db.Exec(
		`INSERT INTO deploys (commit_sha, "trigger", publisher, author_name, author_email, started_at, duration_ms, outcome, reason)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.CommitSHA, e.Trigger, e.Publisher, e.AuthorName, e.AuthorEmail,
		e.StartedAt.UTC().Format(time.RFC3339Nano), e.Duration.Milliseconds(),
		e.Outcome, e.Reason,
	)
	if err != nil {
		return fmt.Errorf("recording deploy: %w", err)
	}
	return nil
}

// List returns entries newest-first. limit <= 0 means all of them.
func (r *Record) List(limit int) ([]Entry, error) {
	q := `SELECT seq, commit_sha, "trigger", publisher, author_name, author_email, started_at, duration_ms, outcome, reason
	      FROM deploys ORDER BY seq DESC`
	args := []any{}
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("listing deploys: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// LastDeployed returns the most recent entry that successfully activated a
// release - a deploy or a rollback - skipping rejections, failures and
// no-ops. This is what rollback rolls back to. Returns (nil, nil) when
// nothing has ever been activated.
func (r *Record) LastDeployed() (*Entry, error) {
	row := r.db.QueryRow(
		`SELECT seq, commit_sha, "trigger", publisher, author_name, author_email, started_at, duration_ms, outcome, reason
		 FROM deploys WHERE outcome IN (?, ?) ORDER BY seq DESC LIMIT 1`,
		OutcomeDeployed, OutcomeRolledBack,
	)
	e, err := scanEntry(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// Close closes the record.
func (r *Record) Close() error {
	return r.db.Close()
}

// Path returns the database file path.
func (r *Record) Path() string {
	return r.path
}

// scanner covers both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanEntry(s scanner) (Entry, error) {
	var e Entry
	var startedAt string
	var durationMS int64
	err := s.Scan(&e.Seq, &e.CommitSHA, &e.Trigger, &e.Publisher, &e.AuthorName,
		&e.AuthorEmail, &startedAt, &durationMS, &e.Outcome, &e.Reason)
	if err == sql.ErrNoRows {
		return e, err
	}
	if err != nil {
		return e, fmt.Errorf("scanning deploy entry: %w", err)
	}
	if t, perr := time.Parse(time.RFC3339Nano, startedAt); perr == nil {
		e.StartedAt = t
	}
	e.Duration = time.Duration(durationMS) * time.Millisecond
	return e, nil
}
