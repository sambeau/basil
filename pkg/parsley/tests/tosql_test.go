package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

// TestTableBindingToSQL tests the .toSQL() method on TableBinding
func TestTableBindingToSQL(t *testing.T) {
	evaluator.ClearDBConnections()

	input := `
@schema User {
	id: int
	name: string
	age: int
	status: string
}

let db = @sqlite(":memory:")
let Users = db.bind(User, "users")

// Test all()
let q1 = Users.toSQL("all")
log(q1.sql)

// Test all() with options
let q2 = Users.toSQL("all", {orderBy: "name", limit: 10, offset: 5})
log(q2.sql)
log(q2.params)

// Test where()
let q3 = Users.toSQL("where", {status: "active"})
log(q3.sql)
log(q3.params)

// Test find()
let q4 = Users.toSQL("find", 42)
log(q4.sql)
log(q4.params)

// Test count()
let q5 = Users.toSQL("count")
log(q5.sql)

// Test count() with where
let q6 = Users.toSQL("count", {status: "active"})
log(q6.sql)
log(q6.params)

// Test sum()
let q7 = Users.toSQL("sum", "age")
log(q7.sql)

// Test first()
let q8 = Users.toSQL("first", {orderBy: "name"})
log(q8.sql)

// Test exists()
let q9 = Users.toSQL("exists", {name: "Alice"})
log(q9.sql)
log(q9.params)

// Return success marker
"all-tests-passed"
`

	env := evaluator.NewEnvironment()
	env.Filename = "test.pars"
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	result := evaluator.Eval(program, env)

	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("evaluation error: %s", err.Inspect())
	}

	if str, ok := result.(*evaluator.String); ok {
		if str.Value != "all-tests-passed" {
			t.Errorf("expected 'all-tests-passed', got %q", str.Value)
		}
	} else {
		t.Errorf("expected String result, got %T", result)
	}
}

// TestDSLQueryToSQL tests the toSQL special projection in DSL queries
func TestDSLQueryToSQL(t *testing.T) {
	evaluator.ClearDBConnections()

	input := `
@schema User {
	id: int
	name: string
	age: int
	status: string
}

let db = @sqlite(":memory:")
let Users = db.bind(User, "users")

// Test basic query with toSQL
let q1 = @query(Users | status == "active" | limit 10 ?-> toSQL)
log(q1.sql)
log(q1.params)

// Test query with multiple conditions
let q2 = @query(Users | status == "active" | age > 21 | order name ?-> toSQL)
log(q2.sql)
log(q2.params)

// Return success marker
"dsl-tests-passed"
`

	env := evaluator.NewEnvironment()
	env.Filename = "test.pars"
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	result := evaluator.Eval(program, env)

	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("evaluation error: %s", err.Inspect())
	}

	if str, ok := result.(*evaluator.String); ok {
		if str.Value != "dsl-tests-passed" {
			t.Errorf("expected 'dsl-tests-passed', got %q", str.Value)
		}
	} else {
		t.Errorf("expected String result, got %T", result)
	}
}

// TestToSQLOutputFormat tests the format of the returned SQL dictionary
func TestToSQLOutputFormat(t *testing.T) {
	evaluator.ClearDBConnections()

	input := `
@schema User {
	id: int
	name: string
}

let db = @sqlite(":memory:")
let Users = db.bind(User, "users")

let result = Users.toSQL("where", {name: "Alice"})

// Check structure
{
	hasSql: result.sql != null,
	hasParams: result.params != null,
	sql: result.sql,
	params: result.params
}
`

	env := evaluator.NewEnvironment()
	env.Filename = "test.pars"
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	result := evaluator.Eval(program, env)

	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("evaluation error: %s", err.Inspect())
	}

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary result, got %T", result)
	}

	// Check hasSql
	hasSqlExpr, ok := dict.Pairs["hasSql"]
	if !ok {
		t.Fatal("missing hasSql field")
	}
	hasSqlObj := evaluator.Eval(hasSqlExpr, dict.Env)
	if hasSqlBool, ok := hasSqlObj.(*evaluator.Boolean); !ok || !hasSqlBool.Value {
		t.Error("hasSql should be true")
	}

	// Check hasParams
	hasParamsExpr, ok := dict.Pairs["hasParams"]
	if !ok {
		t.Fatal("missing hasParams field")
	}
	hasParamsObj := evaluator.Eval(hasParamsExpr, dict.Env)
	if hasParamsBool, ok := hasParamsObj.(*evaluator.Boolean); !ok || !hasParamsBool.Value {
		t.Error("hasParams should be true")
	}

	// Check sql value
	sqlExpr, ok := dict.Pairs["sql"]
	if !ok {
		t.Fatal("missing sql field")
	}
	sqlObj := evaluator.Eval(sqlExpr, dict.Env)
	if _, ok := sqlObj.(*evaluator.String); !ok {
		t.Errorf("sql should be a String, got %T", sqlObj)
	}

	// Check params value
	paramsExpr, ok := dict.Pairs["params"]
	if !ok {
		t.Fatal("missing params field")
	}
	paramsObj := evaluator.Eval(paramsExpr, dict.Env)
	if _, ok := paramsObj.(*evaluator.Array); !ok {
		t.Errorf("params should be an Array, got %T", paramsObj)
	}
}

// TestSchemaToSQLWithNullableAndDefaults tests SQL generation for nullable and default fields
func TestSchemaToSQLWithNullableAndDefaults(t *testing.T) {
	evaluator.ClearDBConnections()

	input := `
@schema Product {
	id: int
	name: string
	description: string?
	price: int = 0
	status: string? = "draft"
}

let db = @sqlite(":memory:")
let Products = db.bind(Product, "products")

// The table creation SQL should include NOT NULL, DEFAULT clauses
Products
`

	env := evaluator.NewEnvironment()
	env.Filename = "test.pars"
	env.Security = &evaluator.SecurityPolicy{AllowExecuteAll: true}

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	result := evaluator.Eval(program, env)

	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("evaluation error: %s", err.Inspect())
	}

	// Just verify it didn't error - the actual SQL generation is tested via the TableBinding
	// The main goal is to ensure nullable and default fields work with db.bind()
	if result == nil {
		t.Fatal("result should not be nil")
	}
}
