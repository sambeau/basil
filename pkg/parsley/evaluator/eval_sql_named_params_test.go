package evaluator

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/lexer"
)

// ─── scanSQLNamedParams tests ───

func TestScanSQLNamedParams_NoPlaceholders(t *testing.T) {
	result := scanSQLNamedParams("SELECT * FROM users")
	if result.hasNamed {
		t.Error("Expected hasNamed=false")
	}
	if result.hasPositional {
		t.Error("Expected hasPositional=false")
	}
	if len(result.names) != 0 {
		t.Errorf("Expected 0 names, got %d", len(result.names))
	}
}

func TestScanSQLNamedParams_PositionalOnly(t *testing.T) {
	result := scanSQLNamedParams("SELECT * FROM users WHERE id = ? AND age > ?")
	if result.hasNamed {
		t.Error("Expected hasNamed=false")
	}
	if !result.hasPositional {
		t.Error("Expected hasPositional=true")
	}
	if len(result.names) != 0 {
		t.Errorf("Expected 0 names, got %d", len(result.names))
	}
}

func TestScanSQLNamedParams_BasicNamed(t *testing.T) {
	result := scanSQLNamedParams("SELECT * FROM users WHERE id = :id")
	if !result.hasNamed {
		t.Error("Expected hasNamed=true")
	}
	if result.hasPositional {
		t.Error("Expected hasPositional=false")
	}
	if len(result.names) != 1 || result.names[0] != "id" {
		t.Errorf("Expected names=[id], got %v", result.names)
	}
}

func TestScanSQLNamedParams_MultipleNamed(t *testing.T) {
	result := scanSQLNamedParams("INSERT INTO users (name, age) VALUES (:name, :age)")
	if !result.hasNamed {
		t.Error("Expected hasNamed=true")
	}
	if len(result.names) != 2 {
		t.Fatalf("Expected 2 names, got %d", len(result.names))
	}
	if result.names[0] != "name" || result.names[1] != "age" {
		t.Errorf("Expected names=[name, age], got %v", result.names)
	}
}

func TestScanSQLNamedParams_RepeatedName(t *testing.T) {
	result := scanSQLNamedParams("SELECT * FROM users WHERE name LIKE :term OR email LIKE :term")
	if len(result.names) != 2 {
		t.Fatalf("Expected 2 names (repeated), got %d", len(result.names))
	}
	if result.names[0] != "term" || result.names[1] != "term" {
		t.Errorf("Expected names=[term, term], got %v", result.names)
	}
}

func TestScanSQLNamedParams_UnderscorePrefixed(t *testing.T) {
	result := scanSQLNamedParams("SELECT * FROM t WHERE x = :_private")
	if !result.hasNamed {
		t.Error("Expected hasNamed=true")
	}
	if len(result.names) != 1 || result.names[0] != "_private" {
		t.Errorf("Expected names=[_private], got %v", result.names)
	}
}

func TestScanSQLNamedParams_MixedPlaceholders(t *testing.T) {
	result := scanSQLNamedParams("SELECT * FROM users WHERE id = :id AND age > ?")
	if !result.hasNamed {
		t.Error("Expected hasNamed=true")
	}
	if !result.hasPositional {
		t.Error("Expected hasPositional=true")
	}
}

func TestScanSQLNamedParams_PostgresDoubleCast(t *testing.T) {
	result := scanSQLNamedParams("SELECT id::text, name::varchar FROM users WHERE id = :id")
	if !result.hasNamed {
		t.Error("Expected hasNamed=true")
	}
	if len(result.names) != 1 || result.names[0] != "id" {
		t.Errorf("Expected names=[id], got %v — :: cast was treated as named param", result.names)
	}
}

func TestScanSQLNamedParams_TripleColon(t *testing.T) {
	// Degenerate case: :::name — first two colons form ::, third + name is :name
	result := scanSQLNamedParams("SELECT :::foo FROM t")
	if !result.hasNamed {
		t.Error("Expected hasNamed=true for :::foo")
	}
	if len(result.names) != 1 || result.names[0] != "foo" {
		t.Errorf("Expected names=[foo], got %v", result.names)
	}
}

func TestScanSQLNamedParams_SingleQuotedStringSkipped(t *testing.T) {
	result := scanSQLNamedParams("SELECT * FROM users WHERE name = ':not_a_param'")
	if result.hasNamed {
		t.Error("Expected hasNamed=false — :not_a_param is inside string literal")
	}
	if len(result.names) != 0 {
		t.Errorf("Expected 0 names, got %v", result.names)
	}
}

func TestScanSQLNamedParams_EscapedQuoteInString(t *testing.T) {
	result := scanSQLNamedParams("SELECT * FROM t WHERE x = 'it''s :not_this' AND y = :real")
	if len(result.names) != 1 || result.names[0] != "real" {
		t.Errorf("Expected names=[real], got %v", result.names)
	}
}

func TestScanSQLNamedParams_DollarQuotingSkipped(t *testing.T) {
	result := scanSQLNamedParams("SELECT $$ body with :stuff $$ FROM t WHERE id = :id")
	if len(result.names) != 1 || result.names[0] != "id" {
		t.Errorf("Expected names=[id], got %v — dollar-quoted content should be skipped", result.names)
	}
}

func TestScanSQLNamedParams_UnterminatedDollarQuote(t *testing.T) {
	// Unterminated dollar-quote — scanner should not panic
	result := scanSQLNamedParams("SELECT $$ body with :stuff")
	if result.hasNamed {
		t.Error("Expected hasNamed=false — :stuff is inside unterminated dollar-quote")
	}
}

func TestScanSQLNamedParams_LineCommentSkipped(t *testing.T) {
	result := scanSQLNamedParams("SELECT * FROM t -- WHERE x = :commented_out\nWHERE y = :real")
	if len(result.names) != 1 || result.names[0] != "real" {
		t.Errorf("Expected names=[real], got %v — line comment content should be skipped", result.names)
	}
}

func TestScanSQLNamedParams_BlockCommentSkipped(t *testing.T) {
	result := scanSQLNamedParams("SELECT * FROM t /* :ignored */ WHERE id = :id")
	if len(result.names) != 1 || result.names[0] != "id" {
		t.Errorf("Expected names=[id], got %v — block comment content should be skipped", result.names)
	}
}

func TestScanSQLNamedParams_NumericSuffix(t *testing.T) {
	// :1 and :123 should NOT be treated as named params
	result := scanSQLNamedParams("SELECT * FROM t WHERE x = :1 OR y = :123")
	if result.hasNamed {
		t.Error("Expected hasNamed=false — :1 and :123 are not valid identifiers")
	}
}

func TestScanSQLNamedParams_ColonAtEnd(t *testing.T) {
	result := scanSQLNamedParams("SELECT * FROM t WHERE x = :")
	if result.hasNamed {
		t.Error("Expected hasNamed=false — bare : at end of string")
	}
}

func TestScanSQLNamedParams_IdentifierWithDigits(t *testing.T) {
	result := scanSQLNamedParams("SELECT * FROM t WHERE x = :param1 AND y = :p2_val")
	if len(result.names) != 2 {
		t.Fatalf("Expected 2 names, got %d", len(result.names))
	}
	if result.names[0] != "param1" || result.names[1] != "p2_val" {
		t.Errorf("Expected names=[param1, p2_val], got %v", result.names)
	}
}

func TestScanSQLNamedParams_EmptySQL(t *testing.T) {
	result := scanSQLNamedParams("")
	if result.hasNamed || result.hasPositional {
		t.Error("Expected no placeholders for empty string")
	}
}

// ─── rewriteNamedParams tests ───

// helper to make a Dictionary with pre-evaluated literal values
func makeParamsDict(pairs map[string]Object, keyOrder []string) *Dictionary {
	env := NewEnvironment()
	exprPairs := make(map[string]ast.Expression, len(pairs))
	for k, v := range pairs {
		exprPairs[k] = makeLiteralExpr(v)
	}
	return &Dictionary{Pairs: exprPairs, KeyOrder: keyOrder, Env: env}
}

func makeLiteralExpr(obj Object) ast.Expression {
	switch v := obj.(type) {
	case *Integer:
		return &ast.IntegerLiteral{
			Token: lexer.Token{Type: lexer.INT, Literal: "0"},
			Value: v.Value,
		}
	case *String:
		return &ast.StringLiteral{
			Token: lexer.Token{Type: lexer.STRING, Literal: v.Value},
			Value: v.Value,
		}
	case *Boolean:
		return &ast.Boolean{
			Token: lexer.Token{Type: lexer.TRUE, Literal: "true"},
			Value: v.Value,
		}
	default:
		return &ast.StringLiteral{Value: "unknown"}
	}
}

func TestRewriteNamedParams_SQLite(t *testing.T) {
	sql := "SELECT * FROM users WHERE id = :id AND name = :name"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{
		"id":   &Integer{Value: 42},
		"name": &String{Value: "Alice"},
	}, []string{"id", "name"})

	rewritten, params, err := rewriteNamedParams(sql, scan, dict, dict.Env, "sqlite")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Message)
	}
	if rewritten != "SELECT * FROM users WHERE id = $1 AND name = $2" {
		t.Errorf("Unexpected rewrite for sqlite: %s", rewritten)
	}
	if len(params) != 2 {
		t.Fatalf("Expected 2 params, got %d", len(params))
	}
	if params[0] != int64(42) {
		t.Errorf("Expected params[0]=42, got %v", params[0])
	}
	if params[1] != "Alice" {
		t.Errorf("Expected params[1]='Alice', got %v", params[1])
	}
}

func TestRewriteNamedParams_Postgres(t *testing.T) {
	sql := "INSERT INTO t (a, b) VALUES (:a, :b)"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{
		"a": &String{Value: "x"},
		"b": &Integer{Value: 7},
	}, []string{"a", "b"})

	rewritten, params, err := rewriteNamedParams(sql, scan, dict, dict.Env, "postgres")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Message)
	}
	if rewritten != "INSERT INTO t (a, b) VALUES ($1, $2)" {
		t.Errorf("Unexpected rewrite for postgres: %s", rewritten)
	}
	if len(params) != 2 {
		t.Fatalf("Expected 2 params, got %d", len(params))
	}
	if params[0] != "x" {
		t.Errorf("Expected params[0]='x', got %v", params[0])
	}
	if params[1] != int64(7) {
		t.Errorf("Expected params[1]=7, got %v", params[1])
	}
}

func TestRewriteNamedParams_MySQL(t *testing.T) {
	sql := "SELECT * FROM users WHERE id = :id AND name = :name"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{
		"id":   &Integer{Value: 1},
		"name": &String{Value: "Bob"},
	}, []string{"id", "name"})

	rewritten, params, err := rewriteNamedParams(sql, scan, dict, dict.Env, "mysql")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Message)
	}
	if rewritten != "SELECT * FROM users WHERE id = ? AND name = ?" {
		t.Errorf("Unexpected rewrite for mysql: %s", rewritten)
	}
	if len(params) != 2 {
		t.Fatalf("Expected 2 params, got %d", len(params))
	}
}

func TestRewriteNamedParams_RepeatedParam(t *testing.T) {
	sql := "SELECT * FROM t WHERE a LIKE :term OR b LIKE :term"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{
		"term": &String{Value: "%foo%"},
	}, []string{"term"})

	rewritten, params, err := rewriteNamedParams(sql, scan, dict, dict.Env, "postgres")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Message)
	}
	if rewritten != "SELECT * FROM t WHERE a LIKE $1 OR b LIKE $2" {
		t.Errorf("Unexpected rewrite: %s", rewritten)
	}
	if len(params) != 2 {
		t.Fatalf("Expected 2 params (repeated), got %d", len(params))
	}
	if params[0] != "%foo%" || params[1] != "%foo%" {
		t.Errorf("Expected both params to be '%%foo%%', got %v, %v", params[0], params[1])
	}
}

func TestRewriteNamedParams_PreservesDoubleColon(t *testing.T) {
	sql := "SELECT name::text FROM users WHERE id = :id"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{
		"id": &Integer{Value: 1},
	}, []string{"id"})

	rewritten, _, err := rewriteNamedParams(sql, scan, dict, dict.Env, "postgres")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Message)
	}
	if !strings.Contains(rewritten, "::text") {
		t.Errorf("Expected :: cast to be preserved, got: %s", rewritten)
	}
	if !strings.Contains(rewritten, "$1") {
		t.Errorf("Expected $1 placeholder, got: %s", rewritten)
	}
}

func TestRewriteNamedParams_PreservesSingleQuotedString(t *testing.T) {
	sql := "SELECT * FROM t WHERE a = ':not_a_param' AND b = :real"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{
		"real": &Integer{Value: 5},
	}, []string{"real"})

	rewritten, params, err := rewriteNamedParams(sql, scan, dict, dict.Env, "sqlite")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Message)
	}
	if !strings.Contains(rewritten, "':not_a_param'") {
		t.Errorf("Expected string literal to be preserved, got: %s", rewritten)
	}
	if len(params) != 1 {
		t.Fatalf("Expected 1 param, got %d", len(params))
	}
}

func TestRewriteNamedParams_PreservesDollarQuoting(t *testing.T) {
	sql := "SELECT $$body :stuff$$ FROM t WHERE id = :id"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{
		"id": &Integer{Value: 1},
	}, []string{"id"})

	rewritten, params, err := rewriteNamedParams(sql, scan, dict, dict.Env, "postgres")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Message)
	}
	if !strings.Contains(rewritten, "$$body :stuff$$") {
		t.Errorf("Expected dollar-quoted content to be preserved verbatim, got: %s", rewritten)
	}
	if len(params) != 1 {
		t.Fatalf("Expected 1 param, got %d", len(params))
	}
}

func TestRewriteNamedParams_PreservesLineComment(t *testing.T) {
	sql := "SELECT * FROM t -- :commented\nWHERE id = :id"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{
		"id": &Integer{Value: 1},
	}, []string{"id"})

	rewritten, params, err := rewriteNamedParams(sql, scan, dict, dict.Env, "sqlite")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Message)
	}
	if !strings.Contains(rewritten, "-- :commented") {
		t.Errorf("Expected line comment to be preserved, got: %s", rewritten)
	}
	if len(params) != 1 {
		t.Fatalf("Expected 1 param, got %d", len(params))
	}
}

func TestRewriteNamedParams_PreservesBlockComment(t *testing.T) {
	sql := "SELECT * FROM t /* :ignored */ WHERE id = :id"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{
		"id": &Integer{Value: 1},
	}, []string{"id"})

	rewritten, params, err := rewriteNamedParams(sql, scan, dict, dict.Env, "sqlite")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Message)
	}
	if !strings.Contains(rewritten, "/* :ignored */") {
		t.Errorf("Expected block comment to be preserved, got: %s", rewritten)
	}
	if len(params) != 1 {
		t.Fatalf("Expected 1 param, got %d", len(params))
	}
}

func TestRewriteNamedParams_MissingParam(t *testing.T) {
	sql := "SELECT * FROM t WHERE id = :id"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{}, nil)

	_, _, err := rewriteNamedParams(sql, scan, dict, dict.Env, "sqlite")
	if err == nil {
		t.Fatal("Expected SQL-0005 error for missing param")
	}
	if err.Code != "SQL-0005" {
		t.Errorf("Expected error code SQL-0005, got %s", err.Code)
	}
	if !strings.Contains(err.Message, ":id") {
		t.Errorf("Expected error message to mention :id, got: %s", err.Message)
	}
}

func TestRewriteNamedParams_MissingParamAvailableListSorted(t *testing.T) {
	sql := "SELECT * FROM t WHERE id = :missing"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{
		"zebra": &Integer{Value: 1},
		"alpha": &Integer{Value: 2},
		"mike":  &Integer{Value: 3},
	}, []string{"zebra", "alpha", "mike"})

	_, _, err := rewriteNamedParams(sql, scan, dict, dict.Env, "sqlite")
	if err == nil {
		t.Fatal("Expected SQL-0005 error")
	}
	// The available list should be sorted alphabetically
	if !strings.Contains(err.Message, ":alpha, :mike, :zebra") {
		t.Errorf("Expected sorted available list ':alpha, :mike, :zebra' in message, got: %s", err.Message)
	}
}

func TestRewriteNamedParams_AttributeOrderIrrelevant(t *testing.T) {
	// Attributes listed as name, age but SQL uses :age, :name order
	sql := "INSERT INTO t (age, name) VALUES (:age, :name)"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{
		"name": &String{Value: "Alice"},
		"age":  &Integer{Value: 30},
	}, []string{"name", "age"})

	_, params, err := rewriteNamedParams(sql, scan, dict, dict.Env, "sqlite")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Message)
	}
	if len(params) != 2 {
		t.Fatalf("Expected 2 params, got %d", len(params))
	}
	// Params should follow SQL order: age first, then name
	if params[0] != int64(30) {
		t.Errorf("Expected params[0]=30 (age, per SQL order), got %v", params[0])
	}
	if params[1] != "Alice" {
		t.Errorf("Expected params[1]='Alice' (name, per SQL order), got %v", params[1])
	}
}

func TestRewriteNamedParams_NoPlaceholders(t *testing.T) {
	sql := "SELECT * FROM users"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{}, nil)

	rewritten, params, err := rewriteNamedParams(sql, scan, dict, dict.Env, "sqlite")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Message)
	}
	if rewritten != sql {
		t.Errorf("Expected SQL unchanged, got: %s", rewritten)
	}
	if len(params) != 0 {
		t.Errorf("Expected 0 params, got %d", len(params))
	}
}

func TestRewriteNamedParams_EscapedQuoteInString(t *testing.T) {
	sql := "SELECT * FROM t WHERE x = 'it''s :not_this' AND y = :real"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{
		"real": &Integer{Value: 9},
	}, []string{"real"})

	rewritten, params, err := rewriteNamedParams(sql, scan, dict, dict.Env, "sqlite")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Message)
	}
	if !strings.Contains(rewritten, "it''s :not_this") {
		t.Errorf("Expected escaped quote and :not_this preserved in string literal, got: %s", rewritten)
	}
	if len(params) != 1 {
		t.Fatalf("Expected 1 param, got %d", len(params))
	}
}

func TestRewriteNamedParams_AllDriverPlaceholders(t *testing.T) {
	sql := "SELECT * FROM t WHERE a = :x AND b = :y AND c = :z"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{
		"x": &Integer{Value: 1},
		"y": &Integer{Value: 2},
		"z": &Integer{Value: 3},
	}, []string{"x", "y", "z"})

	cases := []struct {
		driver string
		want   string
	}{
		{"sqlite", "SELECT * FROM t WHERE a = $1 AND b = $2 AND c = $3"},
		{"postgres", "SELECT * FROM t WHERE a = $1 AND b = $2 AND c = $3"},
		{"mysql", "SELECT * FROM t WHERE a = ? AND b = ? AND c = ?"},
	}

	for _, c := range cases {
		rewritten, _, err := rewriteNamedParams(sql, scan, dict, dict.Env, c.driver)
		if err != nil {
			t.Fatalf("[%s] Unexpected error: %s", c.driver, err.Message)
		}
		if rewritten != c.want {
			t.Errorf("[%s] Expected %q, got %q", c.driver, c.want, rewritten)
		}
	}
}

func TestScanSQLNamedParams_UnterminatedBlockComment(t *testing.T) {
	// Unterminated block comment — scanner should not panic
	result := scanSQLNamedParams("SELECT /* unterminated")
	if result.hasNamed {
		t.Error("Expected hasNamed=false — nothing outside a comment")
	}
	if result.hasPositional {
		t.Error("Expected hasPositional=false")
	}
}

func TestScanSQLNamedParams_UnterminatedBlockCommentWithParam(t *testing.T) {
	// :x inside an unterminated block comment must not be detected
	result := scanSQLNamedParams("SELECT /* comment with :x")
	if result.hasNamed {
		t.Error("Expected hasNamed=false — :x is inside unterminated block comment")
	}
}

func TestScanSQLNamedParams_UnterminatedBlockCommentLastChar(t *testing.T) {
	// The last character before EOF (inside the comment) must not be lost/panic
	result := scanSQLNamedParams("SELECT /*x")
	if result.hasNamed {
		t.Error("Expected hasNamed=false")
	}
}

func TestScanSQLNamedParams_TerminatedBlockCommentRegression(t *testing.T) {
	// Regression: properly terminated block comment still works after the fix
	result := scanSQLNamedParams("SELECT /* comment */ :id FROM t")
	if !result.hasNamed {
		t.Error("Expected hasNamed=true")
	}
	if len(result.names) != 1 || result.names[0] != "id" {
		t.Errorf("Expected names=[id], got %v", result.names)
	}
}

func TestRewriteNamedParams_UnterminatedBlockComment(t *testing.T) {
	// An unterminated block comment must copy all characters verbatim (no loss)
	sql := "SELECT /* unterminated"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{}, []string{})

	rewritten, params, err := rewriteNamedParams(sql, scan, dict, dict.Env, "sqlite")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Message)
	}
	if rewritten != sql {
		t.Errorf("Expected output to equal input %q, got %q", sql, rewritten)
	}
	if len(params) != 0 {
		t.Errorf("Expected 0 params, got %d", len(params))
	}
}

func TestRewriteNamedParams_UnterminatedBlockCommentLastChar(t *testing.T) {
	// The last character inside an unterminated comment must not be lost
	sql := "SELECT /*x"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{}, []string{})

	rewritten, _, err := rewriteNamedParams(sql, scan, dict, dict.Env, "sqlite")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Message)
	}
	if rewritten != sql {
		t.Errorf("Expected output %q, got %q", sql, rewritten)
	}
}

func TestRewriteNamedParams_TerminatedBlockCommentRegression(t *testing.T) {
	// Regression: properly terminated block comment with a named param after it
	sql := "SELECT /* comment */ :id FROM t"
	scan := scanSQLNamedParams(sql)
	dict := makeParamsDict(map[string]Object{
		"id": &Integer{Value: 7},
	}, []string{"id"})

	rewritten, params, err := rewriteNamedParams(sql, scan, dict, dict.Env, "sqlite")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Message)
	}
	want := "SELECT /* comment */ $1 FROM t"
	if rewritten != want {
		t.Errorf("Expected %q, got %q", want, rewritten)
	}
	if len(params) != 1 || params[0] != int64(7) {
		t.Errorf("Expected params=[7], got %v", params)
	}
}
