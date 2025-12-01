package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sambeau/basil/config"
)

func TestDatabaseInit(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("no database configured", func(t *testing.T) {
		cfg := config.Defaults()
		cfg.BaseDir = tmpDir
		cfg.Server.Dev = true

		var stdout, stderr bytes.Buffer
		s, err := New(cfg, "", "test", &stdout, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if s.db != nil {
			t.Error("expected db to be nil when not configured")
		}
	})

	t.Run("sqlite database configured", func(t *testing.T) {
		cfg := config.Defaults()
		cfg.BaseDir = tmpDir
		cfg.Server.Dev = true
		cfg.Database = config.DatabaseConfig{
			Driver: "sqlite",
			Path:   "test.db",
		}

		var stdout, stderr bytes.Buffer
		s, err := New(cfg, "", "test", &stdout, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if s.db == nil {
			t.Error("expected db to be non-nil")
		}
		if s.dbDriver != "sqlite" {
			t.Errorf("expected driver 'sqlite', got %q", s.dbDriver)
		}

		// Clean up
		s.db.Close()
	})

	t.Run("sqlite absolute path", func(t *testing.T) {
		dbPath := filepath.Join(tmpDir, "absolute.db")
		cfg := config.Defaults()
		cfg.BaseDir = "/some/other/dir"
		cfg.Server.Dev = true
		cfg.Database = config.DatabaseConfig{
			Driver: "sqlite",
			Path:   dbPath, // Absolute path
		}

		var stdout, stderr bytes.Buffer
		s, err := New(cfg, "", "test", &stdout, &stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if s.db == nil {
			t.Error("expected db to be non-nil")
		}

		// Clean up
		s.db.Close()
	})

	t.Run("unknown driver", func(t *testing.T) {
		cfg := config.Defaults()
		cfg.BaseDir = tmpDir
		cfg.Server.Dev = true
		cfg.Database = config.DatabaseConfig{
			Driver: "mongodb",
			Path:   "test.db",
		}

		var stdout, stderr bytes.Buffer
		_, err := New(cfg, "", "test", &stdout, &stderr)
		if err == nil {
			t.Fatal("expected error for unknown driver")
		}
	})

	t.Run("postgres not yet supported", func(t *testing.T) {
		cfg := config.Defaults()
		cfg.BaseDir = tmpDir
		cfg.Server.Dev = true
		cfg.Database = config.DatabaseConfig{
			Driver: "postgres",
			DSN:    "host=localhost",
		}

		var stdout, stderr bytes.Buffer
		_, err := New(cfg, "", "test", &stdout, &stderr)
		if err == nil {
			t.Fatal("expected error for postgres (not yet supported)")
		}
	})

	t.Run("sqlite missing path", func(t *testing.T) {
		cfg := config.Defaults()
		cfg.BaseDir = tmpDir
		cfg.Server.Dev = true
		cfg.Database = config.DatabaseConfig{
			Driver: "sqlite",
			// Path is empty
		}

		var stdout, stderr bytes.Buffer
		_, err := New(cfg, "", "test", &stdout, &stderr)
		if err == nil {
			t.Fatal("expected error for missing sqlite path")
		}
	})
}

func TestDatabaseInHandler(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple database with test data
	dbPath := filepath.Join(tmpDir, "handler_test.db")

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true
	cfg.Database = config.DatabaseConfig{
		Driver: "sqlite",
		Path:   dbPath,
	}

	// Create a test handler script that queries the database
	handlersDir := filepath.Join(tmpDir, "handlers")
	if err := os.MkdirAll(handlersDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Script that creates a table and inserts data
	setupScript := `
// Create test table and insert data
let _ = db <=!=> "CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)"
let _ = db <=!=> "INSERT INTO users (name) VALUES ('Alice')"
let _ = db <=!=> "INSERT INTO users (name) VALUES ('Bob')"
<p>Setup complete</p>
`
	if err := os.WriteFile(filepath.Join(handlersDir, "setup.pars"), []byte(setupScript), 0o644); err != nil {
		t.Fatal(err)
	}

	// Script that queries users
	queryScript := `
let users = db <=??=> "SELECT id, name FROM users ORDER BY id"
<ul>
{for (user in users) {
    <li>{user.name}</li>
}}
</ul>
`
	if err := os.WriteFile(filepath.Join(handlersDir, "users.pars"), []byte(queryScript), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg.Routes = []config.Route{
		{Path: "/setup", Handler: filepath.Join(handlersDir, "setup.pars")},
		{Path: "/users", Handler: filepath.Join(handlersDir, "users.pars")},
	}

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer s.db.Close()

	// Run setup to create table and insert data
	req := httptest.NewRequest("GET", "/setup", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("setup: expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Query the users
	req = httptest.NewRequest("GET", "/users", nil)
	w = httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("users: expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !contains(body, "Alice") || !contains(body, "Bob") {
		t.Errorf("expected response to contain Alice and Bob, got: %s", body)
	}
}

func TestDatabaseShutdown(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "shutdown_test.db")

	cfg := config.Defaults()
	cfg.BaseDir = tmpDir
	cfg.Server.Dev = true
	cfg.Database = config.DatabaseConfig{
		Driver: "sqlite",
		Path:   dbPath,
	}

	var stdout, stderr bytes.Buffer
	s, err := New(cfg, "", "test", &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- s.Run(ctx)
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for shutdown with timeout
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("unexpected error during shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown timed out")
	}

	// Verify database connection was closed
	err = s.db.Ping()
	if err == nil {
		t.Error("expected database to be closed after shutdown")
	}
}
