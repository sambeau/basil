package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/basil/server/config"
)

// cloneShapedConfig is what a clone of a site actually looks like on a laptop:
// the legacy layout (no site root, DataDir == the working copy), carrying the
// release's basil.yaml — which came from the server, so auth is enabled — and
// no auth database, because a laptop has no data directory and no reason to
// have been given one.
func cloneShapedConfig(dir string, dev bool) *config.Config {
	return &config.Config{
		ReleaseDir: dir,
		DataDir:    dir,
		Server:     config.ServerConfig{Host: "localhost", Port: 8080, Dev: dev},
		Auth:       config.AuthConfig{Enabled: true},
		Logging:    config.LoggingConfig{Level: "error", Format: "text", Output: "stderr"},
	}
}

// TestDevCreatesMissingAuthDatabaseInAClone guards BUG-039.
//
// `git clone https://host/.git` is the documented way to set up a dev partner,
// and basil --init --server prints it as its own closing instruction. The
// clone carries the server's config, so auth.enabled is true; before this fix
// `basil --dev` in that folder refused to start at all:
//
//	error: creating server: initializing auth: opening auth database:
//	no authentication database found in this folder (.../.basil-auth.db)
//
// A hobbyist must be able to clone and run with no config and no flags.
func TestDevCreatesMissingAuthDatabaseInAClone(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, ".basil-auth.db")

	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("precondition: %s should not exist yet", dbPath)
	}

	srv, err := New(cloneShapedConfig(dir, true), "", "test", "test-commit", &noopBuffer{}, &noopBuffer{})
	if err != nil {
		t.Fatalf("basil --dev in a clone must start, got: %v", err)
	}
	if srv.authDB == nil {
		t.Fatal("dev mode started without an auth database handle")
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("dev mode did not create %s: %v", dbPath, err)
	}

	// It must not degrade to auth-off: a site using auth.protected_paths would
	// then serve its protected pages open locally and closed in production.
	if !srv.config.Auth.Enabled {
		t.Error("dev mode disabled authentication instead of creating the database; " +
			"protected paths would be open locally and closed in production")
	}
}

// TestProductionStillRefusesAMissingAuthDatabase is the other half of BUG-039.
//
// On a real server a missing auth database means the credentials are gone.
// That is an emergency to report, not a state to paper over by quietly making
// a new empty one — so only --dev creates.
func TestProductionStillRefusesAMissingAuthDatabase(t *testing.T) {
	dir := t.TempDir()

	_, err := New(cloneShapedConfig(dir, false), "", "test", "test-commit", &noopBuffer{}, &noopBuffer{})
	if err == nil {
		t.Fatal("production started with no auth database; it must refuse")
	}
	if !strings.Contains(err.Error(), "no authentication database") {
		t.Errorf("error should name the missing database, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".basil-auth.db")); !os.IsNotExist(statErr) {
		t.Error("production created an auth database; only --dev may do that")
	}
}
