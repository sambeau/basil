package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDevLogCreate(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultDevLogConfig()
	dl, err := NewDevLog(tmpDir, cfg)
	if err != nil {
		t.Fatalf("failed to create dev log: %v", err)
	}
	defer dl.Close()

	// Check that database file was created
	if _, err := os.Stat(dl.Path()); os.IsNotExist(err) {
		t.Error("database file was not created")
	}

	// Check path uses the fixed default filename
	base := filepath.Base(dl.Path())
	if base != "dev_logs.db" {
		t.Errorf("expected 'dev_logs.db', got: %s", base)
	}
}

func TestDevLogCreateWithCustomPath(t *testing.T) {
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "custom", "logs.db")

	cfg := DevLogConfig{
		Path:        customPath,
		MaxSize:     1024 * 1024,
		TruncatePct: 10,
	}
	dl, err := NewDevLog(tmpDir, cfg)
	if err != nil {
		t.Fatalf("failed to create dev log: %v", err)
	}
	defer dl.Close()

	if dl.Path() != customPath {
		t.Errorf("expected path %s, got %s", customPath, dl.Path())
	}

	// Check that subdirectory was created
	if _, err := os.Stat(filepath.Dir(customPath)); os.IsNotExist(err) {
		t.Error("custom directory was not created")
	}
}

func TestDevLogWrite(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultDevLogConfig()
	dl, err := NewDevLog(tmpDir, cfg)
	if err != nil {
		t.Fatalf("failed to create dev log: %v", err)
	}
	defer dl.Close()

	// Write a log entry
	entry := LogEntry{
		Route:     "",
		Level:     "info",
		Filename:  "test.pars",
		Line:      42,
		CallRepr:  "dev.log(x)",
		ValueRepr: "[1, 2, 3]",
	}

	if err := dl.Log(entry); err != nil {
		t.Fatalf("failed to log entry: %v", err)
	}

	// Verify it was written
	count, err := dl.Count("")
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}
}

func TestDevLogRead(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultDevLogConfig()
	dl, err := NewDevLog(tmpDir, cfg)
	if err != nil {
		t.Fatalf("failed to create dev log: %v", err)
	}
	defer dl.Close()

	// Write multiple entries
	for i := range 3 {
		entry := LogEntry{
			Route:     "",
			Level:     "info",
			Filename:  "test.pars",
			Line:      i + 1,
			CallRepr:  "dev.log(x)",
			ValueRepr: "value",
		}
		if err := dl.Log(entry); err != nil {
			t.Fatalf("failed to log entry %d: %v", i, err)
		}
	}

	// Read back
	entries, err := dl.GetLogs("", 10)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}

	// Verify order (newest first)
	if entries[0].Line != 3 {
		t.Errorf("expected newest entry first, got line %d", entries[0].Line)
	}
}

func TestDevLogReadByRoute(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultDevLogConfig()
	dl, err := NewDevLog(tmpDir, cfg)
	if err != nil {
		t.Fatalf("failed to create dev log: %v", err)
	}
	defer dl.Close()

	// Write entries to different routes
	routes := []string{"", "users", "orders", "users"}
	for i, route := range routes {
		entry := LogEntry{
			Route:     route,
			Level:     "info",
			Filename:  "test.pars",
			Line:      i + 1,
			CallRepr:  "dev.log(x)",
			ValueRepr: "value",
		}
		if err := dl.Log(entry); err != nil {
			t.Fatalf("failed to log entry %d: %v", i, err)
		}
	}

	// Read all
	all, err := dl.GetLogs("", 10)
	if err != nil {
		t.Fatalf("failed to get all logs: %v", err)
	}
	if len(all) != 4 {
		t.Errorf("expected 4 entries total, got %d", len(all))
	}

	// Read users route only
	users, err := dl.GetLogs("users", 10)
	if err != nil {
		t.Fatalf("failed to get users logs: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 entries for users route, got %d", len(users))
	}

	// Read orders route only
	orders, err := dl.GetLogs("orders", 10)
	if err != nil {
		t.Fatalf("failed to get orders logs: %v", err)
	}
	if len(orders) != 1 {
		t.Errorf("expected 1 entry for orders route, got %d", len(orders))
	}
}

func TestDevLogClear(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultDevLogConfig()
	dl, err := NewDevLog(tmpDir, cfg)
	if err != nil {
		t.Fatalf("failed to create dev log: %v", err)
	}
	defer dl.Close()

	// Write some entries
	for i := range 5 {
		entry := LogEntry{
			Route:     "",
			Level:     "info",
			Filename:  "test.pars",
			Line:      i + 1,
			CallRepr:  "dev.log(x)",
			ValueRepr: "value",
		}
		if err := dl.Log(entry); err != nil {
			t.Fatalf("failed to log entry %d: %v", i, err)
		}
	}

	// Clear all
	if err := dl.ClearLogs(""); err != nil {
		t.Fatalf("failed to clear logs: %v", err)
	}

	// Verify empty
	count, err := dl.Count("")
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0 after clear, got %d", count)
	}
}

func TestDevLogClearByRoute(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultDevLogConfig()
	dl, err := NewDevLog(tmpDir, cfg)
	if err != nil {
		t.Fatalf("failed to create dev log: %v", err)
	}
	defer dl.Close()

	// Write entries to different routes
	routes := []string{"", "users", "orders", "users"}
	for i, route := range routes {
		entry := LogEntry{
			Route:     route,
			Level:     "info",
			Filename:  "test.pars",
			Line:      i + 1,
			CallRepr:  "dev.log(x)",
			ValueRepr: "value",
		}
		if err := dl.Log(entry); err != nil {
			t.Fatalf("failed to log entry %d: %v", i, err)
		}
	}

	// Clear only users route
	if err := dl.ClearLogs("users"); err != nil {
		t.Fatalf("failed to clear users logs: %v", err)
	}

	// Verify users cleared
	users, err := dl.GetLogs("users", 10)
	if err != nil {
		t.Fatalf("failed to get users logs: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 entries for users route after clear, got %d", len(users))
	}

	// Verify others still exist
	total, err := dl.Count("")
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if total != 2 { // empty route + orders
		t.Errorf("expected 2 entries remaining, got %d", total)
	}
}

func TestDevLogLevels(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultDevLogConfig()
	dl, err := NewDevLog(tmpDir, cfg)
	if err != nil {
		t.Fatalf("failed to create dev log: %v", err)
	}
	defer dl.Close()

	// Write info and warn entries
	infoEntry := LogEntry{
		Route:     "",
		Level:     "info",
		Filename:  "test.pars",
		Line:      1,
		CallRepr:  "dev.log(x)",
		ValueRepr: "info value",
	}
	if err := dl.Log(infoEntry); err != nil {
		t.Fatalf("failed to log info: %v", err)
	}

	warnEntry := LogEntry{
		Route:     "",
		Level:     "warn",
		Filename:  "test.pars",
		Line:      2,
		CallRepr:  "dev.log(x, {level: \"warn\"})",
		ValueRepr: "warn value",
	}
	if err := dl.Log(warnEntry); err != nil {
		t.Fatalf("failed to log warn: %v", err)
	}

	// Read back and verify levels
	entries, err := dl.GetLogs("", 10)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Entries are newest first
	if entries[0].Level != "warn" {
		t.Errorf("expected warn level, got %s", entries[0].Level)
	}
	if entries[1].Level != "info" {
		t.Errorf("expected info level, got %s", entries[1].Level)
	}
}

func TestDevLogTimestamp(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultDevLogConfig()
	dl, err := NewDevLog(tmpDir, cfg)
	if err != nil {
		t.Fatalf("failed to create dev log: %v", err)
	}
	defer dl.Close()

	before := time.Now().Add(-time.Second)

	entry := LogEntry{
		Route:     "",
		Level:     "info",
		Filename:  "test.pars",
		Line:      1,
		CallRepr:  "dev.log(x)",
		ValueRepr: "value",
	}
	if err := dl.Log(entry); err != nil {
		t.Fatalf("failed to log: %v", err)
	}

	after := time.Now().Add(time.Second)

	entries, err := dl.GetLogs("", 10)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	ts := entries[0].Timestamp
	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v not in expected range [%v, %v]", ts, before, after)
	}
}

func TestDevLogFromEvaluator(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultDevLogConfig()
	dl, err := NewDevLog(tmpDir, cfg)
	if err != nil {
		t.Fatalf("failed to create dev log: %v", err)
	}
	defer dl.Close()

	// Test the interface method used by the evaluator
	err = dl.LogFromEvaluator("myroute", "warn", "/path/to/handler.pars", 42, "dev.log(x)", "[1, 2, 3]")
	if err != nil {
		t.Fatalf("failed to log from evaluator: %v", err)
	}

	entries, err := dl.GetLogs("myroute", 10)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Route != "myroute" {
		t.Errorf("expected route 'myroute', got '%s'", e.Route)
	}
	if e.Level != "warn" {
		t.Errorf("expected level 'warn', got '%s'", e.Level)
	}
	if e.Filename != "/path/to/handler.pars" {
		t.Errorf("expected filename '/path/to/handler.pars', got '%s'", e.Filename)
	}
	if e.Line != 42 {
		t.Errorf("expected line 42, got %d", e.Line)
	}
}
