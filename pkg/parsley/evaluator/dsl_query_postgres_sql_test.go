package evaluator

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func makeBinding(driver, tableName, softDeleteCol string) *TableBinding {
	return &TableBinding{
		DB:               &DBConnection{Driver: driver},
		TableName:        tableName,
		SoftDeleteColumn: softDeleteCol,
	}
}

func makeEnv() *Environment {
	return NewEnvironment()
}

// strExpr wraps a Go string as an AST StringLiteral.
func strExpr(s string) *ast.StringLiteral {
	return &ast.StringLiteral{
		Token: lexer.Token{Type: lexer.STRING, Literal: s},
		Value: s,
	}
}

// intExpr wraps a Go int64 as an AST IntegerLiteral.
func intExpr(v int64) *ast.IntegerLiteral {
	return &ast.IntegerLiteral{
		Token: lexer.Token{Type: lexer.INT, Literal: "0"},
		Value: v,
	}
}

// identExpr wraps a name as an AST Identifier.
func identExpr(name string) *ast.Identifier {
	return &ast.Identifier{
		Token: lexer.Token{Type: lexer.IDENT, Literal: name},
		Value: name,
	}
}

// interpExpr wraps a Parsley object as an interpolation node so the condition
// builder will evaluate it and emit a placeholder.
func interpExpr(obj Object) *ast.QueryInterpolation {
	// We wrap the object in an immediately-evaluable literal expression.
	// Using a StringLiteral whose value is the string representation is the
	// easiest way to stay within the evaluator's type system for these tests.
	switch v := obj.(type) {
	case *String:
		return &ast.QueryInterpolation{Expression: strExpr(v.Value)}
	case *Integer:
		return &ast.QueryInterpolation{Expression: intExpr(v.Value)}
	default:
		return &ast.QueryInterpolation{Expression: strExpr(obj.Inspect())}
	}
}

func eqCondition(column string, val Object) *ast.QueryCondition {
	return &ast.QueryCondition{
		Left:     identExpr(column),
		Operator: "==",
		Right:    interpExpr(val),
	}
}

func manyTerminal(projection ...string) *ast.QueryTerminal {
	if len(projection) == 0 {
		projection = []string{"*"}
	}
	return &ast.QueryTerminal{Type: "many", Projection: projection}
}

func oneTerminal(projection ...string) *ast.QueryTerminal {
	if len(projection) == 0 {
		projection = []string{"*"}
	}
	return &ast.QueryTerminal{Type: "one", Projection: projection}
}

func executeTerminal() *ast.QueryTerminal {
	return &ast.QueryTerminal{Type: "execute"}
}

// mockResult is a sql.Result that lets us control LastInsertId behaviour.
type mockResult struct {
	rowsAffected  int64
	lastInsertID  int64
	lastInsertErr error
}

func (m mockResult) LastInsertId() (int64, error) { return m.lastInsertID, m.lastInsertErr }
func (m mockResult) RowsAffected() (int64, error) { return m.rowsAffected, nil }

var _ sql.Result = mockResult{}

// buildLastIdDict builds the pairs map the same way evalExecuteStatement does,
// and returns the evaluated lastId object so tests can assert its type.
func buildLastIdDict(res sql.Result, env *Environment) (affectedObj, lastIdObj Object) {
	affected, _ := res.RowsAffected()
	lastIdVal, lastIdErr := res.LastInsertId()

	pairs := map[string]ast.Expression{
		"affected": &ast.IntegerLiteral{
			Token: lexer.Token{Type: lexer.INT},
			Value: affected,
		},
	}
	if lastIdErr == nil {
		pairs["lastId"] = &ast.IntegerLiteral{
			Token: lexer.Token{Type: lexer.INT},
			Value: lastIdVal,
		}
	} else {
		pairs["lastId"] = &ast.Identifier{
			Token: lexer.Token{Type: lexer.IDENT, Literal: "null"},
			Value: "null",
		}
	}

	affectedObj = Eval(pairs["affected"], env)
	lastIdObj = Eval(pairs["lastId"], env)
	return
}

// ─────────────────────────────────────────────────────────────────────────────
// P1-1: Soft delete timestamp dialect
// ─────────────────────────────────────────────────────────────────────────────

func TestPostgres_SoftDelete_UsesCurrentTimestamp(t *testing.T) {
	binding := makeBinding("postgres", "users", "deleted_at")
	node := &ast.DeleteExpression{
		Source:     identExpr("Users"),
		Conditions: nil,
		Terminal:   executeTerminal(),
	}

	sqlStr, _, err := buildDeleteSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sqlStr, "CURRENT_TIMESTAMP") {
		t.Errorf("expected CURRENT_TIMESTAMP in Postgres soft delete SQL, got: %s", sqlStr)
	}
	if strings.Contains(sqlStr, "datetime('now')") {
		t.Errorf("Postgres soft delete must not contain datetime('now'), got: %s", sqlStr)
	}
}

func TestSQLite_SoftDelete_UsesDatetimeNow(t *testing.T) {
	binding := makeBinding("sqlite", "users", "deleted_at")
	node := &ast.DeleteExpression{
		Source:     identExpr("Users"),
		Conditions: nil,
		Terminal:   executeTerminal(),
	}

	sqlStr, _, err := buildDeleteSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sqlStr, "datetime('now')") {
		t.Errorf("expected datetime('now') in SQLite soft delete SQL, got: %s", sqlStr)
	}
	if strings.Contains(sqlStr, "CURRENT_TIMESTAMP") {
		t.Errorf("SQLite soft delete must not contain CURRENT_TIMESTAMP, got: %s", sqlStr)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// P2-2 SELECT — placeholder dialect
// ─────────────────────────────────────────────────────────────────────────────

func TestPostgres_SelectCondition_UsesDollarPlaceholders(t *testing.T) {
	binding := makeBinding("postgres", "users", "")
	node := &ast.QueryExpression{
		Source:     identExpr("Users"),
		Conditions: []ast.QueryConditionNode{eqCondition("id", &Integer{Value: 42})},
		Terminal:   manyTerminal("*"),
	}

	sqlStr, params, err := buildSelectSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sqlStr, "$1") {
		t.Errorf("expected $1 placeholder in Postgres SELECT, got: %s", sqlStr)
	}
	if len(params) != 1 {
		t.Errorf("expected 1 param, got %d", len(params))
	}
}

func TestPostgres_SelectMultipleConditions_UsesDollarPlaceholders(t *testing.T) {
	binding := makeBinding("postgres", "orders", "")
	node := &ast.QueryExpression{
		Source: identExpr("Orders"),
		Conditions: []ast.QueryConditionNode{
			eqCondition("status", &String{Value: "open"}),
			&ast.QueryCondition{
				Left:     identExpr("amount"),
				Operator: ">",
				Right:    interpExpr(&Integer{Value: 100}),
				Logic:    "and",
			},
		},
		Terminal: manyTerminal("*"),
	}

	sqlStr, params, err := buildSelectSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sqlStr, "$1") {
		t.Errorf("expected $1 in SQL, got: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "$2") {
		t.Errorf("expected $2 in SQL, got: %s", sqlStr)
	}
	if len(params) != 2 {
		t.Errorf("expected 2 params, got %d", len(params))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// P2-2 INSERT — placeholders and RETURNING
// ─────────────────────────────────────────────────────────────────────────────

func TestPostgres_InsertValues_UsesDollarPlaceholders(t *testing.T) {
	binding := makeBinding("postgres", "users", "")
	node := &ast.InsertExpression{
		Source: identExpr("Users"),
		Writes: []*ast.InsertFieldWrite{
			{Field: "name", Value: strExpr("Alice")},
			{Field: "email", Value: strExpr("alice@example.com")},
		},
		Terminal: executeTerminal(),
	}

	sqlStr, params, err := buildInsertSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sqlStr, "$1") {
		t.Errorf("expected $1 in INSERT SQL, got: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "$2") {
		t.Errorf("expected $2 in INSERT SQL, got: %s", sqlStr)
	}
	if len(params) != 2 {
		t.Errorf("expected 2 params, got %d", len(params))
	}
}

func TestPostgres_InsertReturning_HasReturningClause(t *testing.T) {
	binding := makeBinding("postgres", "users", "")
	node := &ast.InsertExpression{
		Source: identExpr("Users"),
		Writes: []*ast.InsertFieldWrite{
			{Field: "name", Value: strExpr("Alice")},
		},
		Terminal: oneTerminal("*"),
	}

	sqlStr, _, err := buildInsertSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sqlStr, "RETURNING") {
		t.Errorf("expected RETURNING in INSERT SQL, got: %s", sqlStr)
	}
}

func TestPostgres_InsertNonReturning_NoReturningClause(t *testing.T) {
	binding := makeBinding("postgres", "users", "")
	node := &ast.InsertExpression{
		Source: identExpr("Users"),
		Writes: []*ast.InsertFieldWrite{
			{Field: "name", Value: strExpr("Alice")},
		},
		Terminal: executeTerminal(),
	}

	sqlStr, _, err := buildInsertSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(sqlStr, "RETURNING") {
		t.Errorf("execute terminal must not emit RETURNING, got: %s", sqlStr)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// P2-2 UPDATE — placeholders and RETURNING
// ─────────────────────────────────────────────────────────────────────────────

func TestPostgres_UpdateCondition_UsesDollarPlaceholders(t *testing.T) {
	binding := makeBinding("postgres", "users", "")
	node := &ast.UpdateExpression{
		Source: identExpr("Users"),
		Writes: []*ast.InsertFieldWrite{
			{Field: "name", Value: strExpr("Bob")},
		},
		Conditions: []*ast.QueryCondition{
			{Left: identExpr("id"), Operator: "==", Right: interpExpr(&Integer{Value: 1})},
		},
		Terminal: executeTerminal(),
	}

	sqlStr, params, err := buildUpdateSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// SET uses $1, WHERE uses $2
	if !strings.Contains(sqlStr, "$1") {
		t.Errorf("expected $1 in UPDATE SQL, got: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "$2") {
		t.Errorf("expected $2 in UPDATE SQL, got: %s", sqlStr)
	}
	if len(params) != 2 {
		t.Errorf("expected 2 params, got %d", len(params))
	}
}

func TestPostgres_UpdateReturning_HasReturningClause(t *testing.T) {
	binding := makeBinding("postgres", "users", "")
	node := &ast.UpdateExpression{
		Source: identExpr("Users"),
		Writes: []*ast.InsertFieldWrite{
			{Field: "name", Value: strExpr("Bob")},
		},
		Conditions: []*ast.QueryCondition{
			{Left: identExpr("id"), Operator: "==", Right: interpExpr(&Integer{Value: 1})},
		},
		Terminal: oneTerminal("*"),
	}

	sqlStr, _, err := buildUpdateSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sqlStr, "RETURNING") {
		t.Errorf("expected RETURNING in UPDATE SQL, got: %s", sqlStr)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// P2-2 DELETE — placeholders and RETURNING
// ─────────────────────────────────────────────────────────────────────────────

func TestPostgres_DeleteHard_UsesDollarPlaceholders(t *testing.T) {
	binding := makeBinding("postgres", "users", "")
	node := &ast.DeleteExpression{
		Source: identExpr("Users"),
		Conditions: []*ast.QueryCondition{
			{Left: identExpr("id"), Operator: "==", Right: interpExpr(&Integer{Value: 99})},
		},
		Terminal: executeTerminal(),
	}

	sqlStr, params, err := buildDeleteSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sqlStr, "$1") {
		t.Errorf("expected $1 in DELETE SQL, got: %s", sqlStr)
	}
	if len(params) != 1 {
		t.Errorf("expected 1 param, got %d", len(params))
	}
}

func TestPostgres_DeleteReturning_HasReturningClause(t *testing.T) {
	binding := makeBinding("postgres", "users", "")
	node := &ast.DeleteExpression{
		Source: identExpr("Users"),
		Conditions: []*ast.QueryCondition{
			{Left: identExpr("id"), Operator: "==", Right: interpExpr(&Integer{Value: 99})},
		},
		Terminal: oneTerminal("*"),
	}

	sqlStr, _, err := buildDeleteSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sqlStr, "RETURNING") {
		t.Errorf("expected RETURNING in DELETE SQL, got: %s", sqlStr)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// P2-2 Upsert — ON CONFLICT ... EXCLUDED
// ─────────────────────────────────────────────────────────────────────────────

func TestPostgres_Upsert_UsesOnConflictExcluded(t *testing.T) {
	binding := makeBinding("postgres", "users", "")
	node := &ast.InsertExpression{
		Source: identExpr("Users"),
		Writes: []*ast.InsertFieldWrite{
			{Field: "email", Value: strExpr("alice@example.com")},
			{Field: "name", Value: strExpr("Alice")},
		},
		UpsertKey: []string{"email"},
		Terminal:  executeTerminal(),
	}

	sqlStr, _, err := buildInsertSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sqlStr, "ON CONFLICT") {
		t.Errorf("expected ON CONFLICT in upsert SQL, got: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "EXCLUDED") {
		t.Errorf("expected EXCLUDED in upsert SQL, got: %s", sqlStr)
	}
	if strings.Contains(sqlStr, "ON DUPLICATE KEY") {
		t.Errorf("Postgres upsert must not use ON DUPLICATE KEY syntax, got: %s", sqlStr)
	}
}

func TestSQLite_Upsert_UsesOnConflictExcluded(t *testing.T) {
	binding := makeBinding("sqlite", "users", "")
	node := &ast.InsertExpression{
		Source: identExpr("Users"),
		Writes: []*ast.InsertFieldWrite{
			{Field: "email", Value: strExpr("alice@example.com")},
		},
		UpsertKey: []string{"email"},
		Terminal:  executeTerminal(),
	}

	sqlStr, _, err := buildInsertSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sqlStr, "ON CONFLICT") {
		t.Errorf("expected ON CONFLICT in SQLite upsert SQL, got: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "EXCLUDED") {
		t.Errorf("expected EXCLUDED in SQLite upsert SQL, got: %s", sqlStr)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// P2-2 IN clause
// ─────────────────────────────────────────────────────────────────────────────

func TestPostgres_InClause_UsesDollarPlaceholders(t *testing.T) {
	paramIdx := 1
	values := &Array{Elements: []Object{
		&Integer{Value: 1},
		&Integer{Value: 2},
		&Integer{Value: 3},
	}}

	sqlStr, params, err := buildInClause("id", values, &paramIdx, "postgres")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sqlStr, "$1") {
		t.Errorf("expected $1 in IN clause, got: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "$2") {
		t.Errorf("expected $2 in IN clause, got: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "$3") {
		t.Errorf("expected $3 in IN clause, got: %s", sqlStr)
	}
	if len(params) != 3 {
		t.Errorf("expected 3 params, got %d", len(params))
	}
	if paramIdx != 4 {
		t.Errorf("expected paramIdx to be 4 after 3 placeholders, got %d", paramIdx)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// P2-2 BETWEEN
// ─────────────────────────────────────────────────────────────────────────────

func TestPostgres_Between_UsesDollarPlaceholders(t *testing.T) {
	binding := makeBinding("postgres", "products", "")
	node := &ast.QueryExpression{
		Source: identExpr("Products"),
		Conditions: []ast.QueryConditionNode{
			&ast.QueryCondition{
				Left:     identExpr("price"),
				Operator: "between",
				Right:    interpExpr(&Integer{Value: 10}),
				RightEnd: interpExpr(&Integer{Value: 100}),
			},
		},
		Terminal: manyTerminal("*"),
	}

	sqlStr, params, err := buildSelectSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(sqlStr, "BETWEEN $1 AND $2") {
		t.Errorf("expected BETWEEN $1 AND $2 in Postgres SQL, got: %s", sqlStr)
	}
	if len(params) != 2 {
		t.Errorf("expected 2 params, got %d", len(params))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// P1-2: LastInsertId null/integer behaviour
// ─────────────────────────────────────────────────────────────────────────────

func TestLastInsertId_Error_ReturnsNull(t *testing.T) {
	res := mockResult{
		rowsAffected:  1,
		lastInsertErr: errors.New("driver does not support LastInsertId"),
	}
	env := makeEnv()

	affectedObj, lastIdObj := buildLastIdDict(res, env)

	if _, ok := affectedObj.(*Integer); !ok {
		t.Errorf("expected affected to be Integer, got %T", affectedObj)
	}

	// When LastInsertId errors, the expression stored is an Identifier("null")
	// which evaluates to NULL.
	if lastIdObj != NULL {
		t.Errorf("expected lastId to be NULL when LastInsertId errors, got %T: %v", lastIdObj, lastIdObj)
	}
}

func TestLastInsertId_Success_ReturnsInteger(t *testing.T) {
	res := mockResult{
		rowsAffected:  1,
		lastInsertID:  42,
		lastInsertErr: nil,
	}
	env := makeEnv()

	_, lastIdObj := buildLastIdDict(res, env)

	intObj, ok := lastIdObj.(*Integer)
	if !ok {
		t.Errorf("expected lastId to be Integer, got %T: %v", lastIdObj, lastIdObj)
		return
	}
	if intObj.Value != 42 {
		t.Errorf("expected lastId = 42, got %d", intObj.Value)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers verify driver sentinel values
// ─────────────────────────────────────────────────────────────────────────────

func TestCurrentTimestampSQL_Drivers(t *testing.T) {
	cases := []struct {
		driver string
		want   string
	}{
		{"sqlite", "datetime('now')"},
		{"postgres", "CURRENT_TIMESTAMP"},
		{"mysql", "CURRENT_TIMESTAMP"},
		{"", "CURRENT_TIMESTAMP"}, // unknown driver falls back to standard SQL
	}
	for _, c := range cases {
		got := currentTimestampSQL(c.driver)
		if got != c.want {
			t.Errorf("currentTimestampSQL(%q) = %q, want %q", c.driver, got, c.want)
		}
	}
}

func TestSQLPlaceholder_Drivers(t *testing.T) {
	cases := []struct {
		driver string
		idx    int
		want   string
	}{
		{"postgres", 1, "$1"},
		{"postgres", 3, "$3"},
		{"sqlite", 1, "$1"},
		{"sqlite", 5, "$5"},
		{"mysql", 1, "?"},
		{"mysql", 9, "?"},
		{"", 1, "$1"}, // unknown driver falls back to $N
	}
	for _, c := range cases {
		got := sqlPlaceholder(c.driver, c.idx)
		if got != c.want {
			t.Errorf("sqlPlaceholder(%q, %d) = %q, want %q", c.driver, c.idx, got, c.want)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// P3-1: MySQL ? placeholders
// ─────────────────────────────────────────────────────────────────────────────

func TestMySQL_SelectCondition_UsesQuestionMarkPlaceholder(t *testing.T) {
	binding := makeBinding("mysql", "users", "")

	node := &ast.QueryExpression{
		Source: identExpr("users"),
		Conditions: []ast.QueryConditionNode{
			&ast.QueryCondition{Left: identExpr("id"), Operator: "==", Right: interpExpr(&Integer{Value: 42})},
		},
		Terminal: manyTerminal(),
	}

	sqlStr, _, err := buildSelectSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sqlStr, "?") {
		t.Errorf("expected ? placeholder, got: %s", sqlStr)
	}
	if strings.Contains(sqlStr, "$1") {
		t.Errorf("expected no $N placeholder for MySQL, got: %s", sqlStr)
	}
}

func TestMySQL_InsertValues_UsesQuestionMarkPlaceholder(t *testing.T) {
	binding := makeBinding("mysql", "users", "")

	node := &ast.InsertExpression{
		Source: identExpr("users"),
		Writes: []*ast.InsertFieldWrite{
			{Field: "name", Value: strExpr("Alice")},
			{Field: "email", Value: strExpr("alice@example.com")},
		},
		Terminal: executeTerminal(),
	}

	sqlStr, _, err := buildInsertSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sqlStr, "?, ?") {
		t.Errorf("expected ?, ? placeholders, got: %s", sqlStr)
	}
	if strings.Contains(sqlStr, "$") {
		t.Errorf("expected no $N placeholder for MySQL, got: %s", sqlStr)
	}
}

func TestMySQL_UpdateCondition_UsesQuestionMarkPlaceholder(t *testing.T) {
	binding := makeBinding("mysql", "posts", "")

	node := &ast.UpdateExpression{
		Source: identExpr("posts"),
		Writes: []*ast.InsertFieldWrite{
			{Field: "title", Value: strExpr("New Title")},
		},
		Conditions: []*ast.QueryCondition{
			{Left: identExpr("id"), Operator: "==", Right: interpExpr(&Integer{Value: 7})},
		},
		Terminal: executeTerminal(),
	}

	sqlStr, _, err := buildUpdateSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(sqlStr, "$") {
		t.Errorf("expected no $N placeholder for MySQL, got: %s", sqlStr)
	}
	// Two parameters: SET title = ?, WHERE id = ?
	count := strings.Count(sqlStr, "?")
	if count != 2 {
		t.Errorf("expected 2 ? placeholders, got %d in: %s", count, sqlStr)
	}
}

func TestMySQL_DeleteHard_UsesQuestionMarkPlaceholder(t *testing.T) {
	binding := makeBinding("mysql", "users", "")

	node := &ast.DeleteExpression{
		Source: identExpr("users"),
		Conditions: []*ast.QueryCondition{
			{Left: identExpr("id"), Operator: "==", Right: interpExpr(&Integer{Value: 5})},
		},
		Terminal: executeTerminal(),
	}

	sqlStr, _, err := buildDeleteSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(sqlStr, "$") {
		t.Errorf("expected no $N placeholder for MySQL, got: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "?") {
		t.Errorf("expected ? placeholder, got: %s", sqlStr)
	}
}

func TestMySQL_SoftDelete_UsesCurrentTimestamp(t *testing.T) {
	binding := makeBinding("mysql", "users", "deleted_at")

	node := &ast.DeleteExpression{
		Source:   identExpr("users"),
		Terminal: executeTerminal(),
	}

	sqlStr, _, err := buildDeleteSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sqlStr, "CURRENT_TIMESTAMP") {
		t.Errorf("expected CURRENT_TIMESTAMP for MySQL soft delete, got: %s", sqlStr)
	}
	if strings.Contains(sqlStr, "datetime('now')") {
		t.Errorf("expected no datetime('now') for MySQL, got: %s", sqlStr)
	}
}

func TestMySQL_InClause_UsesQuestionMarkPlaceholders(t *testing.T) {
	paramIdx := 1

	values := &Array{Elements: []Object{
		&Integer{Value: 1},
		&Integer{Value: 2},
		&Integer{Value: 3},
	}}

	sqlStr, _, err := buildInClause("id", values, &paramIdx, "mysql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(sqlStr, "$") {
		t.Errorf("expected no $N placeholder for MySQL, got: %s", sqlStr)
	}
	if strings.Count(sqlStr, "?") != 3 {
		t.Errorf("expected 3 ? placeholders, got: %s", sqlStr)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// P3-2: MySQL RETURNING → error DB-0018
// ─────────────────────────────────────────────────────────────────────────────

func TestMySQL_InsertReturning_ReturnsError(t *testing.T) {
	binding := makeBinding("mysql", "users", "")

	node := &ast.InsertExpression{
		Source:   identExpr("users"),
		Writes:   []*ast.InsertFieldWrite{{Field: "name", Value: strExpr("Alice")}},
		Terminal: oneTerminal(),
	}

	_, _, err := buildInsertSQL(node, binding, makeEnv())
	if err == nil {
		t.Fatal("expected error for MySQL RETURNING, got nil")
	}
	if err.Code != "DB-0018" {
		t.Errorf("expected error code DB-0018, got %s", err.Code)
	}
	if !strings.Contains(err.Message, "RETURNING") {
		t.Errorf("expected error message to mention RETURNING, got: %s", err.Message)
	}
}

func TestMySQL_InsertManyReturning_ReturnsError(t *testing.T) {
	binding := makeBinding("mysql", "users", "")

	node := &ast.InsertExpression{
		Source:   identExpr("users"),
		Writes:   []*ast.InsertFieldWrite{{Field: "name", Value: strExpr("Bob")}},
		Terminal: manyTerminal(),
	}

	_, _, err := buildInsertSQL(node, binding, makeEnv())
	if err == nil {
		t.Fatal("expected error for MySQL RETURNING (many), got nil")
	}
	if err.Code != "DB-0018" {
		t.Errorf("expected error code DB-0018, got %s", err.Code)
	}
}

func TestMySQL_InsertNonReturning_Succeeds(t *testing.T) {
	binding := makeBinding("mysql", "users", "")

	node := &ast.InsertExpression{
		Source:   identExpr("users"),
		Writes:   []*ast.InsertFieldWrite{{Field: "name", Value: strExpr("Carol")}},
		Terminal: executeTerminal(),
	}

	sqlStr, _, err := buildInsertSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error for MySQL non-returning INSERT: %v", err)
	}
	if !strings.Contains(sqlStr, "INSERT INTO users") {
		t.Errorf("expected INSERT INTO users, got: %s", sqlStr)
	}
}

func TestMySQL_UpdateReturning_ReturnsError(t *testing.T) {
	binding := makeBinding("mysql", "users", "")

	node := &ast.UpdateExpression{
		Source:   identExpr("users"),
		Writes:   []*ast.InsertFieldWrite{{Field: "name", Value: strExpr("Dave")}},
		Terminal: oneTerminal(),
	}

	_, _, err := buildUpdateSQL(node, binding, makeEnv())
	if err == nil {
		t.Fatal("expected error for MySQL UPDATE RETURNING, got nil")
	}
	if err.Code != "DB-0018" {
		t.Errorf("expected error code DB-0018, got %s", err.Code)
	}
}

func TestMySQL_DeleteReturning_ReturnsError(t *testing.T) {
	binding := makeBinding("mysql", "users", "")

	node := &ast.DeleteExpression{
		Source:   identExpr("users"),
		Terminal: oneTerminal(),
	}

	_, _, err := buildDeleteSQL(node, binding, makeEnv())
	if err == nil {
		t.Fatal("expected error for MySQL DELETE RETURNING, got nil")
	}
	if err.Code != "DB-0018" {
		t.Errorf("expected error code DB-0018, got %s", err.Code)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// P3-3: MySQL upsert → ON DUPLICATE KEY UPDATE
// ─────────────────────────────────────────────────────────────────────────────

func TestMySQL_Upsert_UsesOnDuplicateKeyUpdate(t *testing.T) {
	binding := makeBinding("mysql", "users", "")

	node := &ast.InsertExpression{
		Source: identExpr("users"),
		Writes: []*ast.InsertFieldWrite{
			{Field: "email", Value: strExpr("alice@example.com")},
			{Field: "name", Value: strExpr("Alice")},
		},
		UpsertKey: []string{"email"},
		Terminal:  executeTerminal(),
	}

	sqlStr, _, err := buildInsertSQL(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sqlStr, "ON DUPLICATE KEY UPDATE") {
		t.Errorf("expected ON DUPLICATE KEY UPDATE for MySQL upsert, got: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "VALUES(") {
		t.Errorf("expected VALUES() references in MySQL upsert, got: %s", sqlStr)
	}
	if strings.Contains(sqlStr, "EXCLUDED") {
		t.Errorf("expected no EXCLUDED reference for MySQL, got: %s", sqlStr)
	}
	if strings.Contains(sqlStr, "ON CONFLICT") {
		t.Errorf("expected no ON CONFLICT for MySQL, got: %s", sqlStr)
	}
}

func TestMySQL_BatchUpsert_UsesOnDuplicateKeyUpdate(t *testing.T) {
	binding := makeBinding("mysql", "products", "")

	node := &ast.InsertExpression{
		Source: identExpr("products"),
		Writes: []*ast.InsertFieldWrite{
			{Field: "sku", Value: strExpr("ABC-001")},
			{Field: "price", Value: intExpr(999)},
		},
		UpsertKey: []string{"sku"},
		Terminal:  executeTerminal(),
	}

	sqlStr, _, err := buildInsertSQLForBatch(node, binding, makeEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sqlStr, "ON DUPLICATE KEY UPDATE") {
		t.Errorf("expected ON DUPLICATE KEY UPDATE for MySQL batch upsert, got: %s", sqlStr)
	}
	if strings.Contains(sqlStr, "ON CONFLICT") {
		t.Errorf("expected no ON CONFLICT for MySQL, got: %s", sqlStr)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// P3-4: MySQL DDL → CHAR_LENGTH in CHECK constraints
// ─────────────────────────────────────────────────────────────────────────────

func TestMySQL_CreateTable_UsesCharLength(t *testing.T) {
	minLen := 2
	maxLen := 100
	schema := &DSLSchema{
		Name: "users",
		Fields: map[string]*DSLSchemaField{
			"username": {
				Name:      "username",
				Type:      "string",
				MinLength: &minLen,
				MaxLength: &maxLen,
				Nullable:  true,
			},
		},
		FieldOrder: []string{"username"},
	}

	ddl := buildCreateTableSQL(schema, "users", "mysql")
	if !strings.Contains(ddl, "CHAR_LENGTH") {
		t.Errorf("expected CHAR_LENGTH in MySQL DDL, got: %s", ddl)
	}
	if strings.Contains(ddl, "length(") {
		t.Errorf("expected no length() in MySQL DDL, got: %s", ddl)
	}
}

func TestSQLite_CreateTable_UsesLength(t *testing.T) {
	minLen := 2
	maxLen := 100
	schema := &DSLSchema{
		Name: "users",
		Fields: map[string]*DSLSchemaField{
			"username": {
				Name:      "username",
				Type:      "string",
				MinLength: &minLen,
				MaxLength: &maxLen,
				Nullable:  true,
			},
		},
		FieldOrder: []string{"username"},
	}

	ddl := buildCreateTableSQL(schema, "users", "sqlite")
	if !strings.Contains(ddl, "length(") {
		t.Errorf("expected length() in SQLite DDL, got: %s", ddl)
	}
	if strings.Contains(ddl, "CHAR_LENGTH") {
		t.Errorf("expected no CHAR_LENGTH in SQLite DDL, got: %s", ddl)
	}
}

func TestSQLLengthFunc_Drivers(t *testing.T) {
	cases := []struct {
		driver string
		want   string
	}{
		{"mysql", "CHAR_LENGTH"},
		{"sqlite", "length"},
		{"postgres", "length"},
		{"", "length"},
	}
	for _, c := range cases {
		got := sqlLengthFunc(c.driver)
		if got != c.want {
			t.Errorf("sqlLengthFunc(%q) = %q, want %q", c.driver, got, c.want)
		}
	}
}
