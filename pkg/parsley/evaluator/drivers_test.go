package evaluator

import (
	"database/sql"
	"slices"
	"testing"
)

func TestDriverRegistration(t *testing.T) {
	// Get list of registered SQL drivers
	drivers := sql.Drivers()

	// Check for PostgreSQL driver
	if !slices.Contains(drivers, "postgres") {
		t.Error("PostgreSQL driver 'postgres' is not registered")
	}

	// Check for MySQL driver
	if !slices.Contains(drivers, "mysql") {
		t.Error("MySQL driver 'mysql' is not registered")
	}

	// Check for SQLite driver (should already be registered)
	if !slices.Contains(drivers, "sqlite") {
		t.Error("SQLite driver 'sqlite' is not registered")
	}
}
