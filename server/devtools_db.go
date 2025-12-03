package server

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// TableInfo represents a database table's metadata.
type TableInfo struct {
	Name     string
	Columns  []ColumnInfo
	RowCount int64
}

// ColumnInfo represents a column in a database table.
type ColumnInfo struct {
	Name    string
	Type    string
	NotNull bool
	PK      bool
}

// getTableList returns all user tables (excludes sqlite_* system tables).
func getTableList(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT name FROM sqlite_master 
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("query tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan table name: %w", err)
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

// getTableInfo returns structure and row count for a table.
func getTableInfo(db *sql.DB, tableName string) (*TableInfo, error) {
	// Validate table name to prevent SQL injection
	if !isValidTableName(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	columns, err := getTableColumns(db, tableName)
	if err != nil {
		return nil, err
	}

	// Get row count
	var count int64
	// Table name is validated, safe to use in query
	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %q", tableName)).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("count rows: %w", err)
	}

	return &TableInfo{
		Name:     tableName,
		Columns:  columns,
		RowCount: count,
	}, nil
}

// getTableColumns returns column info using PRAGMA table_info.
func getTableColumns(db *sql.DB, tableName string) ([]ColumnInfo, error) {
	// Validate table name
	if !isValidTableName(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%q)", tableName))
	if err != nil {
		return nil, fmt.Errorf("pragma table_info: %w", err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue sql.NullString

		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("scan column info: %w", err)
		}

		columns = append(columns, ColumnInfo{
			Name:    name,
			Type:    colType,
			NotNull: notNull == 1,
			PK:      pk == 1,
		})
	}
	return columns, rows.Err()
}

// isValidTableName checks if a table name is safe to use in queries.
// Allows alphanumeric characters and underscores.
func isValidTableName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	// Cannot start with a digit
	if name[0] >= '0' && name[0] <= '9' {
		return false
	}
	return true
}

// exportTableCSV writes table data as CSV to the writer.
// BLOB columns are excluded from the export.
func exportTableCSV(db *sql.DB, tableName string, w io.Writer) error {
	if !isValidTableName(tableName) {
		return fmt.Errorf("invalid table name: %s", tableName)
	}

	// Get columns, filter out BLOB types
	columns, err := getTableColumns(db, tableName)
	if err != nil {
		return err
	}

	var exportCols []string
	for _, col := range columns {
		if strings.ToUpper(col.Type) != "BLOB" {
			exportCols = append(exportCols, col.Name)
		}
	}

	if len(exportCols) == 0 {
		return fmt.Errorf("no exportable columns (all BLOB)")
	}

	// Query data
	query := fmt.Sprintf("SELECT %s FROM %q", quoteColumns(exportCols), tableName)
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("query data: %w", err)
	}
	defer rows.Close()

	// Write CSV
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Header row
	if err := csvWriter.Write(exportCols); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Data rows
	values := make([]interface{}, len(exportCols))
	valuePtrs := make([]interface{}, len(exportCols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("scan row: %w", err)
		}

		record := make([]string, len(exportCols))
		for i, v := range values {
			record[i] = formatValue(v)
		}

		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}

	return rows.Err()
}

// quoteColumns returns column names quoted and joined for SQL.
func quoteColumns(cols []string) string {
	quoted := make([]string, len(cols))
	for i, col := range cols {
		quoted[i] = fmt.Sprintf("%q", col)
	}
	return strings.Join(quoted, ", ")
}

// formatValue converts a database value to a string for CSV.
func formatValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case []byte:
		return string(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

// InferredColumn represents a column with its inferred type.
type InferredColumn struct {
	Name string
	Type string // INTEGER, REAL, or TEXT
}

// inferColumnTypes scans all values and infers SQL types.
// Rules:
// - All values parse as int64 → INTEGER
// - All values parse as float64 (or mix) → REAL
// - Otherwise → TEXT
// - Empty strings are treated as NULL (compatible with any type)
func inferColumnTypes(headers []string, rows [][]string) []InferredColumn {
	result := make([]InferredColumn, len(headers))

	for i, header := range headers {
		result[i] = InferredColumn{
			Name: header,
			Type: inferColumnType(rows, i),
		}
	}

	return result
}

// inferColumnType infers the type for a single column.
func inferColumnType(rows [][]string, colIndex int) string {
	allInt := true
	allNumeric := true
	hasNonEmpty := false

	for _, row := range rows {
		if colIndex >= len(row) {
			continue
		}
		val := strings.TrimSpace(row[colIndex])
		if val == "" {
			continue // Empty = NULL, compatible with any type
		}
		hasNonEmpty = true

		// Try int
		if _, err := strconv.ParseInt(val, 10, 64); err != nil {
			allInt = false
		}

		// Try float
		if _, err := strconv.ParseFloat(val, 64); err != nil {
			allNumeric = false
		}
	}

	if !hasNonEmpty {
		// All empty - default to TEXT
		return "TEXT"
	}
	if allInt {
		return "INTEGER"
	}
	if allNumeric {
		return "REAL"
	}
	return "TEXT"
}

// replaceTableFromCSV drops existing table and creates new one from CSV data.
// The operation is atomic (uses transaction).
func replaceTableFromCSV(db *sql.DB, tableName string, r io.Reader) error {
	if !isValidTableName(tableName) {
		return fmt.Errorf("invalid table name: %s", tableName)
	}

	// Parse CSV
	csvReader := csv.NewReader(r)
	records, err := csvReader.ReadAll()
	if err != nil {
		return fmt.Errorf("parse CSV: %w", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("CSV file is empty")
	}

	headers := records[0]
	if len(headers) == 0 {
		return fmt.Errorf("CSV has no columns")
	}

	// Validate column names
	for _, h := range headers {
		if !isValidColumnName(h) {
			return fmt.Errorf("invalid column name: %s", h)
		}
	}

	dataRows := records[1:]

	// Infer column types
	columns := inferColumnTypes(headers, dataRows)

	// Build CREATE TABLE statement
	var colDefs []string
	for _, col := range columns {
		colDefs = append(colDefs, fmt.Sprintf("%q %s", col.Name, col.Type))
	}
	createSQL := fmt.Sprintf("CREATE TABLE %q (%s)", tableName, strings.Join(colDefs, ", "))

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Drop existing table
	if _, err := tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %q", tableName)); err != nil {
		return fmt.Errorf("drop table: %w", err)
	}

	// Create new table
	if _, err := tx.Exec(createSQL); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	// Insert data
	if len(dataRows) > 0 {
		placeholders := make([]string, len(headers))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		insertSQL := fmt.Sprintf("INSERT INTO %q (%s) VALUES (%s)",
			tableName, quoteColumns(headers), strings.Join(placeholders, ", "))

		stmt, err := tx.Prepare(insertSQL)
		if err != nil {
			return fmt.Errorf("prepare insert: %w", err)
		}
		defer stmt.Close()

		for rowNum, row := range dataRows {
			// Pad row if needed
			for len(row) < len(headers) {
				row = append(row, "")
			}

			// Convert values based on inferred types
			args := make([]interface{}, len(headers))
			for i, val := range row[:len(headers)] {
				args[i], err = convertValue(val, columns[i].Type)
				if err != nil {
					return fmt.Errorf("row %d, column %q: %w", rowNum+2, headers[i], err)
				}
			}

			if _, err := stmt.Exec(args...); err != nil {
				return fmt.Errorf("insert row %d: %w", rowNum+2, err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// isValidColumnName checks if a column name is safe.
func isValidColumnName(name string) bool {
	if name == "" {
		return false
	}
	// Allow alphanumeric, underscore, space (SQLite allows spaces in quoted identifiers)
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == ' ') {
			return false
		}
	}
	return true
}

// convertValue converts a CSV string value to the appropriate Go type.
func convertValue(val string, colType string) (interface{}, error) {
	val = strings.TrimSpace(val)
	if val == "" {
		return nil, nil // NULL
	}

	switch colType {
	case "INTEGER":
		return strconv.ParseInt(val, 10, 64)
	case "REAL":
		return strconv.ParseFloat(val, 64)
	default:
		return val, nil
	}
}

// createEmptyTable creates a new table with id INTEGER column and one row (value 0).
func createEmptyTable(db *sql.DB, tableName string) error {
	if !isValidTableName(tableName) {
		return fmt.Errorf("invalid table name: %s (use letters, numbers, underscore; cannot start with number)", tableName)
	}

	// Check if table already exists
	var exists int
	err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, tableName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check table exists: %w", err)
	}
	if exists > 0 {
		return fmt.Errorf("table %q already exists", tableName)
	}

	// Create table and insert initial row
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(fmt.Sprintf("CREATE TABLE %q (id INTEGER)", tableName)); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	if _, err := tx.Exec(fmt.Sprintf("INSERT INTO %q (id) VALUES (0)", tableName)); err != nil {
		return fmt.Errorf("insert initial row: %w", err)
	}

	return tx.Commit()
}
