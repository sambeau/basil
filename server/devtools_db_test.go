package server

import (
	"bytes"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestIsValidTableName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"users", true},
		{"my_table", true},
		{"Table123", true},
		{"_private", true},
		{"123table", false}, // Can't start with digit
		{"my-table", false}, // Hyphen not allowed
		{"my table", false}, // Space not allowed
		{"", false},         // Empty not allowed
		{"select", true},    // Reserved words are technically allowed
		{"user's", false},   // Apostrophe not allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidTableName(tt.name)
			if got != tt.valid {
				t.Errorf("isValidTableName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

func TestIsValidColumnName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"id", true},
		{"user_name", true},
		{"First Name", true}, // Spaces allowed (SQLite supports in quoted identifiers)
		{"age", true},
		{"", false},
		{"col;drop", false}, // Semicolon not allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidColumnName(tt.name)
			if got != tt.valid {
				t.Errorf("isValidColumnName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

func TestGetTableData(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Create and populate test table
	_, err = db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, score REAL);
		INSERT INTO users (id, name, score) VALUES (1, 'Alice', 95.5);
		INSERT INTO users (id, name, score) VALUES (2, 'Bob', NULL);
		INSERT INTO users (id, name, score) VALUES (3, 'Charlie', 87.0);
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Test getTableData
	columns, rows, err := getTableData(db, "users")
	if err != nil {
		t.Fatalf("getTableData failed: %v", err)
	}

	// Check columns
	expectedCols := []string{"id", "name", "score"}
	if len(columns) != len(expectedCols) {
		t.Errorf("expected %d columns, got %d", len(expectedCols), len(columns))
	}
	for i, col := range expectedCols {
		if columns[i] != col {
			t.Errorf("column %d: expected %q, got %q", i, col, columns[i])
		}
	}

	// Check rows
	if len(rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(rows))
	}

	// Check first row values
	if rows[0][0] != int64(1) {
		t.Errorf("row 0 col 0: expected 1, got %v", rows[0][0])
	}
	if rows[0][1] != "Alice" {
		t.Errorf("row 0 col 1: expected 'Alice', got %v", rows[0][1])
	}

	// Check NULL value
	if rows[1][2] != nil {
		t.Errorf("row 1 col 2: expected nil, got %v", rows[1][2])
	}

	// Test invalid table name
	_, _, err = getTableData(db, "invalid-table")
	if err == nil {
		t.Error("expected error for invalid table name")
	}
}

func TestInferColumnType(t *testing.T) {
	tests := []struct {
		name     string
		values   [][]string
		colIndex int
		expected string
	}{
		{
			name:     "all integers",
			values:   [][]string{{"1"}, {"2"}, {"3"}},
			colIndex: 0,
			expected: "INTEGER",
		},
		{
			name:     "all floats",
			values:   [][]string{{"1.5"}, {"2.7"}, {"3.14"}},
			colIndex: 0,
			expected: "REAL",
		},
		{
			name:     "mixed int and float",
			values:   [][]string{{"1"}, {"2.5"}, {"3"}},
			colIndex: 0,
			expected: "REAL",
		},
		{
			name:     "text values",
			values:   [][]string{{"hello"}, {"world"}},
			colIndex: 0,
			expected: "TEXT",
		},
		{
			name:     "mixed text and numbers",
			values:   [][]string{{"hello"}, {"123"}},
			colIndex: 0,
			expected: "TEXT",
		},
		{
			name:     "empty values only",
			values:   [][]string{{""}, {""}},
			colIndex: 0,
			expected: "TEXT",
		},
		{
			name:     "integers with empty",
			values:   [][]string{{"1"}, {""}, {"3"}},
			colIndex: 0,
			expected: "INTEGER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferColumnType(tt.values, tt.colIndex)
			if got != tt.expected {
				t.Errorf("inferColumnType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestInferColumnTypes(t *testing.T) {
	headers := []string{"id", "name", "score"}
	rows := [][]string{
		{"1", "Alice", "95.5"},
		{"2", "Bob", "87.0"},
		{"3", "Charlie", "92.3"},
	}

	result := inferColumnTypes(headers, rows)

	if len(result) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(result))
	}

	expected := []InferredColumn{
		{Name: "id", Type: "INTEGER"},
		{Name: "name", Type: "TEXT"},
		{Name: "score", Type: "REAL"},
	}

	for i, col := range result {
		if col.Name != expected[i].Name || col.Type != expected[i].Type {
			t.Errorf("column %d: got {%s, %s}, want {%s, %s}",
				i, col.Name, col.Type, expected[i].Name, expected[i].Type)
		}
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	return db
}

func TestGetTableList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create some tables
	db.Exec("CREATE TABLE users (id INTEGER, name TEXT)")
	db.Exec("CREATE TABLE posts (id INTEGER, title TEXT)")

	tables, err := getTableList(db)
	if err != nil {
		t.Fatalf("getTableList failed: %v", err)
	}

	if len(tables) != 2 {
		t.Errorf("expected 2 tables, got %d", len(tables))
	}

	// Should be sorted alphabetically
	if tables[0] != "posts" || tables[1] != "users" {
		t.Errorf("expected [posts, users], got %v", tables)
	}
}

func TestGetTableListExcludesSQLiteTables(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	db.Exec("CREATE TABLE users (id INTEGER)")

	tables, err := getTableList(db)
	if err != nil {
		t.Fatalf("getTableList failed: %v", err)
	}

	// Should only have users, not sqlite_* tables
	for _, name := range tables {
		if strings.HasPrefix(name, "sqlite_") {
			t.Errorf("should not include sqlite table: %s", name)
		}
	}
}

func TestGetTableInfo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)")
	db.Exec("INSERT INTO users VALUES (1, 'Alice', 'alice@example.com')")
	db.Exec("INSERT INTO users VALUES (2, 'Bob', 'bob@example.com')")

	info, err := getTableInfo(db, "users")
	if err != nil {
		t.Fatalf("getTableInfo failed: %v", err)
	}

	if info.Name != "users" {
		t.Errorf("expected name 'users', got %q", info.Name)
	}

	if info.RowCount != 2 {
		t.Errorf("expected 2 rows, got %d", info.RowCount)
	}

	if len(info.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(info.Columns))
	}

	// Check id column
	if info.Columns[0].Name != "id" || !info.Columns[0].PK {
		t.Errorf("expected id column to be PK")
	}

	// Check name column
	if info.Columns[1].Name != "name" || !info.Columns[1].NotNull {
		t.Errorf("expected name column to be NOT NULL")
	}
}

func TestExportTableCSV(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	db.Exec("CREATE TABLE users (id INTEGER, name TEXT, score REAL)")
	db.Exec("INSERT INTO users VALUES (1, 'Alice', 95.5)")
	db.Exec("INSERT INTO users VALUES (2, 'Bob', 87.0)")

	var buf bytes.Buffer
	err := exportTableCSV(db, "users", &buf)
	if err != nil {
		t.Fatalf("exportTableCSV failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 rows), got %d", len(lines))
	}

	if lines[0] != "id,name,score" {
		t.Errorf("expected header 'id,name,score', got %q", lines[0])
	}

	if lines[1] != "1,Alice,95.5" {
		t.Errorf("expected first row '1,Alice,95.5', got %q", lines[1])
	}
}

func TestExportTableCSVExcludesBlob(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	db.Exec("CREATE TABLE files (id INTEGER, name TEXT, data BLOB)")
	db.Exec("INSERT INTO files VALUES (1, 'test.txt', X'48454C4C4F')")

	var buf bytes.Buffer
	err := exportTableCSV(db, "files", &buf)
	if err != nil {
		t.Fatalf("exportTableCSV failed: %v", err)
	}

	output := buf.String()
	// Should not include 'data' column
	if strings.Contains(output, "data") {
		t.Errorf("expected BLOB column to be excluded, got: %s", output)
	}

	if !strings.HasPrefix(output, "id,name") {
		t.Errorf("expected header 'id,name', got: %s", output)
	}
}

func TestReplaceTableFromCSV(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create initial table
	db.Exec("CREATE TABLE users (old_col TEXT)")
	db.Exec("INSERT INTO users VALUES ('old_data')")

	// Replace with CSV
	csv := "id,name,score\n1,Alice,95.5\n2,Bob,87"
	err := replaceTableFromCSV(db, "users", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("replaceTableFromCSV failed: %v", err)
	}

	// Verify new structure
	columns, err := getTableColumns(db, "users")
	if err != nil {
		t.Fatalf("getTableColumns failed: %v", err)
	}

	if len(columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(columns))
	}

	// Verify types were inferred correctly
	expectedTypes := map[string]string{
		"id":    "INTEGER",
		"name":  "TEXT",
		"score": "REAL",
	}
	for _, col := range columns {
		if expectedTypes[col.Name] != col.Type {
			t.Errorf("column %s: expected type %s, got %s", col.Name, expectedTypes[col.Name], col.Type)
		}
	}

	// Verify data
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 rows, got %d", count)
	}
}

func TestReplaceTableFromCSVWithEmptyValues(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	csv := "id,name\n1,Alice\n2,\n3,Charlie"
	err := replaceTableFromCSV(db, "users", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("replaceTableFromCSV failed: %v", err)
	}

	// Verify empty string became NULL
	var name sql.NullString
	db.QueryRow("SELECT name FROM users WHERE id = 2").Scan(&name)
	if name.Valid {
		t.Errorf("expected NULL for empty value, got %q", name.String)
	}
}

func TestCreateEmptyTable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := createEmptyTable(db, "new_table")
	if err != nil {
		t.Fatalf("createEmptyTable failed: %v", err)
	}

	// Verify table exists with correct structure
	info, err := getTableInfo(db, "new_table")
	if err != nil {
		t.Fatalf("getTableInfo failed: %v", err)
	}

	if len(info.Columns) != 1 || info.Columns[0].Name != "id" {
		t.Errorf("expected single 'id' column, got %v", info.Columns)
	}

	if info.Columns[0].Type != "INTEGER" {
		t.Errorf("expected INTEGER type, got %s", info.Columns[0].Type)
	}

	if info.RowCount != 1 {
		t.Errorf("expected 1 row, got %d", info.RowCount)
	}

	// Verify the value is 0
	var id int
	db.QueryRow("SELECT id FROM new_table").Scan(&id)
	if id != 0 {
		t.Errorf("expected id=0, got %d", id)
	}
}

func TestCreateEmptyTableDuplicate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := createEmptyTable(db, "users")
	if err != nil {
		t.Fatalf("first createEmptyTable failed: %v", err)
	}

	err = createEmptyTable(db, "users")
	if err == nil {
		t.Error("expected error for duplicate table")
	}
}

func TestCreateEmptyTableInvalidName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := createEmptyTable(db, "invalid-name")
	if err == nil {
		t.Error("expected error for invalid table name")
	}
}

func TestReplaceTableFromCSVPreservesSchema(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a table with specific schema (PRIMARY KEY, NOT NULL, custom types)
	_, err := db.Exec(`CREATE TABLE people (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		email VARCHAR(255),
		score REAL
	)`)
	if err != nil {
		t.Fatalf("create table failed: %v", err)
	}

	// Upload CSV with same columns
	csv := "id,name,email,score\n1,Alice,alice@example.com,95.5\n2,Bob,bob@example.com,87.0"
	err = replaceTableFromCSV(db, "people", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("replaceTableFromCSV failed: %v", err)
	}

	// Verify schema was preserved
	columns, err := getTableColumns(db, "people")
	if err != nil {
		t.Fatalf("getTableColumns failed: %v", err)
	}

	// Check id column
	idCol := findColumn(columns, "id")
	if idCol == nil {
		t.Fatal("id column not found")
	}
	if !idCol.PK {
		t.Error("id column should be PRIMARY KEY")
	}

	// Check name column - should preserve NOT NULL
	nameCol := findColumn(columns, "name")
	if nameCol == nil {
		t.Fatal("name column not found")
	}
	if !nameCol.NotNull {
		t.Error("name column should be NOT NULL")
	}

	// Check email column - should preserve VARCHAR(255)
	emailCol := findColumn(columns, "email")
	if emailCol == nil {
		t.Fatal("email column not found")
	}
	if emailCol.Type != "VARCHAR(255)" {
		t.Errorf("email type should be VARCHAR(255), got %s", emailCol.Type)
	}

	// Check score column - should preserve REAL
	scoreCol := findColumn(columns, "score")
	if scoreCol == nil {
		t.Fatal("score column not found")
	}
	if scoreCol.Type != "REAL" {
		t.Errorf("score type should be REAL, got %s", scoreCol.Type)
	}

	// Verify data was inserted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM people").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 rows, got %d", count)
	}
}

func TestReplaceTableFromCSVAddsIDColumn(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Upload CSV without id column
	csv := "name,email\nAlice,alice@example.com\nBob,bob@example.com"
	err := replaceTableFromCSV(db, "people", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("replaceTableFromCSV failed: %v", err)
	}

	// Verify id column was added
	columns, err := getTableColumns(db, "people")
	if err != nil {
		t.Fatalf("getTableColumns failed: %v", err)
	}

	idCol := findColumn(columns, "id")
	if idCol == nil {
		t.Fatal("id column should have been added")
	}
	if !idCol.PK {
		t.Error("id column should be PRIMARY KEY")
	}

	// Verify auto-increment worked (ids should be 1 and 2)
	var id1, id2 int
	db.QueryRow("SELECT id FROM people WHERE name = 'Alice'").Scan(&id1)
	db.QueryRow("SELECT id FROM people WHERE name = 'Bob'").Scan(&id2)
	if id1 != 1 || id2 != 2 {
		t.Errorf("expected ids 1 and 2, got %d and %d", id1, id2)
	}
}

func findColumn(columns []ColumnInfo, name string) *ColumnInfo {
	for i := range columns {
		if strings.EqualFold(columns[i].Name, name) {
			return &columns[i]
		}
	}
	return nil
}
