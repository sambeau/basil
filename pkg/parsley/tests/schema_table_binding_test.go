package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestSchemaTableInsertAndFind(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  age: schema.integer()
})
let Users = schema.table(User, db, "users")
let inserted = Users.insert({name: "Alice", age: 30})
{inserted: inserted, fetched: Users.find(inserted.id)}
`

	result := evalTest(t, input)
	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary result, got %s", result.Type())
	}

	inserted := evaluator.Eval(dict.Pairs["inserted"], dict.Env)
	fetched := evaluator.Eval(dict.Pairs["fetched"], dict.Env)

	insDict, ok := inserted.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("inserted should be dictionary, got %s", inserted.Type())
	}
	idVal := evaluator.Eval(insDict.Pairs["id"], insDict.Env).(*evaluator.String).Value
	if idVal == "" {
		t.Fatalf("expected generated id, got empty")
	}

	fetchedDict, ok := fetched.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("fetched should be dictionary, got %s", fetched.Type())
	}
	nameVal := evaluator.Eval(fetchedDict.Pairs["name"], fetchedDict.Env).(*evaluator.String).Value
	if nameVal != "Alice" {
		t.Fatalf("expected name Alice, got %s", nameVal)
	}
}

func TestSchemaTableValidationFailure(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  age: schema.integer()
})
let Users = schema.table(User, db, "users")
Users.insert({age: 20})
`

	result := evalTest(t, input)
	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("expected Dictionary result, got %s", result.Type())
	}

	validObj := evaluator.Eval(dict.Pairs["valid"], dict.Env)
	if validObj.Type() != evaluator.BOOLEAN_OBJ || validObj.(*evaluator.Boolean).Value {
		t.Fatalf("expected validation to fail")
	}
}

func TestSchemaTableWhereAndPagination(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  age: schema.integer()
})
let Users = schema.table(User, db, "users")
let _ = Users.insert({name: "Alice", age: 30})
let _ = Users.insert({name: "Bob", age: 25})
let _ = Users.insert({name: "Carol", age: 40})
let basil = {http: {request: {query: {limit: "1", offset: "1"}}}}
{
  filtered: Users.where({age: 25}),
  paged: Users.all()
}
`

	result := evalTest(t, input)
	dict := result.(*evaluator.Dictionary)

	filtered := evaluator.Eval(dict.Pairs["filtered"], dict.Env).(*evaluator.Table)
	if len(filtered.Rows) != 1 {
		t.Fatalf("expected 1 filtered row, got %d", len(filtered.Rows))
	}
	row := filtered.Rows[0]
	age := evaluator.Eval(row.Pairs["age"], row.Env).(*evaluator.Integer).Value
	if age != 25 {
		t.Fatalf("expected age 25, got %d", age)
	}

	paged := evaluator.Eval(dict.Pairs["paged"], dict.Env).(*evaluator.Table)
	if len(paged.Rows) != 1 {
		t.Fatalf("expected paginated result length 1, got %d", len(paged.Rows))
	}
	second := paged.Rows[0]
	name := evaluator.Eval(second.Pairs["name"], second.Env).(*evaluator.String).Value
	if name != "Bob" {
		t.Fatalf("expected second row Bob, got %s", name)
	}
}

func TestSchemaTableDelete(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  age: schema.integer()
})
let Users = schema.table(User, db, "users")
let _ = Users.insert({name: "Alice", age: 30})
Users.delete("non-existent")
`

	result := evalTest(t, input)
	dict := result.(*evaluator.Dictionary)
	affected := evaluator.Eval(dict.Pairs["affected"], dict.Env).(*evaluator.Integer).Value
	if affected != 0 {
		t.Fatalf("expected 0 rows affected, got %d", affected)
	}
}

func TestSchemaTableRejectsInvalidColumn(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  age: schema.integer()
})
let Users = schema.table(User, db, "users")
Users.where({"name; DROP TABLE": "x"})
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Fatalf("expected error for invalid column, got %s", result.Type())
	}
}

// === FEAT-078: Extended Query Methods Tests ===

func TestTableBindingAllWithOrderBy(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  age: schema.integer()
})
let Users = schema.table(User, db, "users")
let _ = Users.insert({name: "Charlie", age: 30})
let _ = Users.insert({name: "Alice", age: 25})
let _ = Users.insert({name: "Bob", age: 35})
{
  byName: Users.all({orderBy: "name", limit: 0}),
  byAgeDesc: Users.all({orderBy: "age", order: "desc", limit: 0})
}
`
	result := evalTest(t, input)
	dict := result.(*evaluator.Dictionary)

	byName := evaluator.Eval(dict.Pairs["byName"], dict.Env).(*evaluator.Table)
	if len(byName.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(byName.Rows))
	}
	firstName := evaluator.Eval(byName.Rows[0].Pairs["name"], nil).(*evaluator.String).Value
	if firstName != "Alice" {
		t.Fatalf("expected first row to be Alice, got %s", firstName)
	}

	byAgeDesc := evaluator.Eval(dict.Pairs["byAgeDesc"], dict.Env).(*evaluator.Table)
	firstAge := evaluator.Eval(byAgeDesc.Rows[0].Pairs["age"], nil).(*evaluator.Integer).Value
	if firstAge != 35 {
		t.Fatalf("expected first row age to be 35, got %d", firstAge)
	}
}

func TestTableBindingAllWithMultiColumnOrderBy(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  age: schema.integer()
})
let Users = schema.table(User, db, "users")
let _ = Users.insert({name: "Alice", age: 30})
let _ = Users.insert({name: "Bob", age: 30})
let _ = Users.insert({name: "Alice", age: 25})
Users.all({orderBy: [["name", "asc"], ["age", "desc"]], limit: 0})
`
	result := evalTest(t, input)
	tbl := result.(*evaluator.Table)
	if len(tbl.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(tbl.Rows))
	}

	// First should be Alice age 30 (name asc, then age desc)
	first := tbl.Rows[0]
	name := evaluator.Eval(first.Pairs["name"], nil).(*evaluator.String).Value
	age := evaluator.Eval(first.Pairs["age"], nil).(*evaluator.Integer).Value
	if name != "Alice" || age != 30 {
		t.Fatalf("expected Alice/30, got %s/%d", name, age)
	}
}

func TestTableBindingAllWithSelect(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  age: schema.integer()
})
let Users = schema.table(User, db, "users")
let _ = Users.insert({name: "Alice", age: 30})
Users.all({select: ["id", "name"], limit: 0})
`
	result := evalTest(t, input)
	tbl := result.(*evaluator.Table)
	if len(tbl.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(tbl.Rows))
	}

	row := tbl.Rows[0]
	if _, hasName := row.Pairs["name"]; !hasName {
		t.Fatal("expected 'name' in result")
	}
	if _, hasId := row.Pairs["id"]; !hasId {
		t.Fatal("expected 'id' in result")
	}
	// age should not be present (SQLite returns null for non-selected columns in our row scan)
}

func TestTableBindingAllWithLimitOffset(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  age: schema.integer()
})
let Users = schema.table(User, db, "users")
let _ = Users.insert({name: "Alice", age: 25})
let _ = Users.insert({name: "Bob", age: 30})
let _ = Users.insert({name: "Carol", age: 35})
Users.all({orderBy: "name", limit: 2, offset: 1})
`
	result := evalTest(t, input)
	tbl := result.(*evaluator.Table)
	if len(tbl.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(tbl.Rows))
	}

	// With orderBy name ASC, offset 1: should get Bob and Carol
	firstName := evaluator.Eval(tbl.Rows[0].Pairs["name"], nil).(*evaluator.String).Value
	if firstName != "Bob" {
		t.Fatalf("expected Bob, got %s", firstName)
	}
}

func TestTableBindingWhereWithOptions(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER, role TEXT)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  age: schema.integer(),
  role: schema.string()
})
let Users = schema.table(User, db, "users")
let _ = Users.insert({name: "Alice", age: 25, role: "admin"})
let _ = Users.insert({name: "Bob", age: 30, role: "admin"})
let _ = Users.insert({name: "Carol", age: 35, role: "user"})
Users.where({role: "admin"}, {orderBy: "age", order: "desc"})
`
	result := evalTest(t, input)
	tbl := result.(*evaluator.Table)
	if len(tbl.Rows) != 2 {
		t.Fatalf("expected 2 admins, got %d", len(tbl.Rows))
	}

	// Should be Bob (30) then Alice (25) due to age DESC
	firstName := evaluator.Eval(tbl.Rows[0].Pairs["name"], nil).(*evaluator.String).Value
	if firstName != "Bob" {
		t.Fatalf("expected Bob first (older), got %s", firstName)
	}
}

func TestTableBindingCount(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, role TEXT)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  role: schema.string()
})
let Users = schema.table(User, db, "users")
let _ = Users.insert({name: "Alice", role: "admin"})
let _ = Users.insert({name: "Bob", role: "user"})
let _ = Users.insert({name: "Carol", role: "admin"})
{
  total: Users.count(),
  admins: Users.count({role: "admin"})
}
`
	result := evalTest(t, input)
	dict := result.(*evaluator.Dictionary)

	total := evaluator.Eval(dict.Pairs["total"], dict.Env).(*evaluator.Integer).Value
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}

	admins := evaluator.Eval(dict.Pairs["admins"], dict.Env).(*evaluator.Integer).Value
	if admins != 2 {
		t.Fatalf("expected 2 admins, got %d", admins)
	}
}

func TestTableBindingSum(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE accounts (id TEXT PRIMARY KEY, name TEXT, balance INTEGER)"
let Account = schema.define("Account", {
  id: schema.id(),
  name: schema.string({required: true}),
  balance: schema.integer()
})
let Accounts = schema.table(Account, db, "accounts")
let _ = Accounts.insert({name: "Alice", balance: 100})
let _ = Accounts.insert({name: "Bob", balance: 200})
let _ = Accounts.insert({name: "Carol", balance: 50})
Accounts.sum("balance")
`
	result := evalTest(t, input)
	sum := result.(*evaluator.Integer).Value
	if sum != 350 {
		t.Fatalf("expected sum 350, got %d", sum)
	}
}

func TestTableBindingAvg(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  age: schema.integer()
})
let Users = schema.table(User, db, "users")
let _ = Users.insert({name: "Alice", age: 20})
let _ = Users.insert({name: "Bob", age: 30})
let _ = Users.insert({name: "Carol", age: 40})
Users.avg("age")
`
	result := evalTest(t, input)
	// SQLite AVG returns float
	avg := result.(*evaluator.Float).Value
	if avg != 30.0 {
		t.Fatalf("expected avg 30.0, got %f", avg)
	}
}

func TestTableBindingMinMax(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, score INTEGER)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  score: schema.integer()
})
let Users = schema.table(User, db, "users")
let _ = Users.insert({name: "Alice", score: 85})
let _ = Users.insert({name: "Bob", score: 92})
let _ = Users.insert({name: "Carol", score: 78})
{
  minScore: Users.min("score"),
  maxScore: Users.max("score")
}
`
	result := evalTest(t, input)
	dict := result.(*evaluator.Dictionary)

	minScore := evaluator.Eval(dict.Pairs["minScore"], dict.Env).(*evaluator.Integer).Value
	if minScore != 78 {
		t.Fatalf("expected min 78, got %d", minScore)
	}

	maxScore := evaluator.Eval(dict.Pairs["maxScore"], dict.Env).(*evaluator.Integer).Value
	if maxScore != 92 {
		t.Fatalf("expected max 92, got %d", maxScore)
	}
}

func TestTableBindingFirst(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  age: schema.integer()
})
let Users = schema.table(User, db, "users")
let u1 = Users.insert({name: "Alice", age: 25})
let _ = Users.insert({name: "Bob", age: 30})
let _ = Users.insert({name: "Carol", age: 35})
{
  single: Users.first(),
  multiple: Users.first(2),
  byAge: Users.first({orderBy: "age", order: "desc"})
}
`
	result := evalTest(t, input)
	dict := result.(*evaluator.Dictionary)

	// first() returns single record
	single := evaluator.Eval(dict.Pairs["single"], dict.Env).(*evaluator.Dictionary)
	if single == nil {
		t.Fatal("expected single result")
	}

	// first(2) returns table
	multiple := evaluator.Eval(dict.Pairs["multiple"], dict.Env).(*evaluator.Table)
	if len(multiple.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(multiple.Rows))
	}

	// first with orderBy
	byAge := evaluator.Eval(dict.Pairs["byAge"], dict.Env).(*evaluator.Dictionary)
	age := evaluator.Eval(byAge.Pairs["age"], nil).(*evaluator.Integer).Value
	if age != 35 {
		t.Fatalf("expected age 35 (oldest first with desc), got %d", age)
	}
}

func TestTableBindingLast(t *testing.T) {
	evaluator.ClearDBConnections()
	// Use explicit IDs to ensure deterministic ordering
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  age: schema.integer()
})
let Users = schema.table(User, db, "users")
let _ = db <=!=> "INSERT INTO users (id, name, age) VALUES ('001', 'Alice', 25)"
let _ = db <=!=> "INSERT INTO users (id, name, age) VALUES ('002', 'Bob', 30)"
let _ = db <=!=> "INSERT INTO users (id, name, age) VALUES ('003', 'Carol', 35)"
{
  single: Users.last(),
  byAge: Users.last({orderBy: "age"})
}
`
	result := evalTest(t, input)
	dict := result.(*evaluator.Dictionary)

	// last() returns last by id DESC (003 = Carol)
	single := evaluator.Eval(dict.Pairs["single"], dict.Env).(*evaluator.Dictionary)
	singleName := evaluator.Eval(single.Pairs["name"], nil).(*evaluator.String).Value
	if singleName != "Carol" {
		t.Fatalf("expected Carol (last by id), got %s", singleName)
	}

	// last({orderBy: "age"}) reverses to DESC, so gets oldest (35)
	byAge := evaluator.Eval(dict.Pairs["byAge"], dict.Env).(*evaluator.Dictionary)
	age := evaluator.Eval(byAge.Pairs["age"], nil).(*evaluator.Integer).Value
	if age != 35 {
		t.Fatalf("expected age 35 (last by age reversed), got %d", age)
	}
}

func TestTableBindingExists(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, email TEXT)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  email: schema.string()
})
let Users = schema.table(User, db, "users")
let _ = Users.insert({name: "Alice", email: "alice@example.com"})
{
  found: Users.exists({email: "alice@example.com"}),
  notFound: Users.exists({email: "nobody@example.com"})
}
`
	result := evalTest(t, input)
	dict := result.(*evaluator.Dictionary)

	found := evaluator.Eval(dict.Pairs["found"], dict.Env).(*evaluator.Boolean).Value
	if !found {
		t.Fatal("expected exists to return true")
	}

	notFound := evaluator.Eval(dict.Pairs["notFound"], dict.Env).(*evaluator.Boolean).Value
	if notFound {
		t.Fatal("expected exists to return false")
	}
}

func TestTableBindingFindBy(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, email TEXT, age INTEGER)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  email: schema.string(),
  age: schema.integer()
})
let Users = schema.table(User, db, "users")
let _ = Users.insert({name: "Alice", email: "alice@example.com", age: 25})
let _ = Users.insert({name: "Bob", email: "bob@example.com", age: 30})
{
  found: Users.findBy({email: "alice@example.com"}),
  notFound: Users.findBy({email: "nobody@example.com"}),
  withOrder: Users.findBy({name: "Alice"}, {select: ["name", "age"]})
}
`
	result := evalTest(t, input)
	dict := result.(*evaluator.Dictionary)

	found := evaluator.Eval(dict.Pairs["found"], dict.Env).(*evaluator.Dictionary)
	name := evaluator.Eval(found.Pairs["name"], nil).(*evaluator.String).Value
	if name != "Alice" {
		t.Fatalf("expected Alice, got %s", name)
	}

	notFound := evaluator.Eval(dict.Pairs["notFound"], dict.Env)
	if notFound.Type() != evaluator.NULL_OBJ {
		t.Fatalf("expected null for not found, got %s", notFound.Type())
	}

	withOrder := evaluator.Eval(dict.Pairs["withOrder"], dict.Env).(*evaluator.Dictionary)
	if _, hasName := withOrder.Pairs["name"]; !hasName {
		t.Fatal("expected 'name' in select result")
	}
}

func TestTableBindingInvalidOrderByColumn(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true})
})
let Users = schema.table(User, db, "users")
Users.all({orderBy: "name; DROP TABLE users"})
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if result.Type() != evaluator.ERROR_OBJ {
		t.Fatalf("expected error for invalid column, got %s", result.Type())
	}
}

func TestTableBindingAggregateOnEmptyTable(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, balance INTEGER)"
let User = schema.define("User", {
  id: schema.id(),
  name: schema.string({required: true}),
  balance: schema.integer()
})
let Users = schema.table(User, db, "users")
{
  count: Users.count(),
  sum: Users.sum("balance"),
  first: Users.first()
}
`
	result := evalTest(t, input)
	dict := result.(*evaluator.Dictionary)

	// count returns 0 on empty table
	count := evaluator.Eval(dict.Pairs["count"], dict.Env).(*evaluator.Integer).Value
	if count != 0 {
		t.Fatalf("expected count 0, got %d", count)
	}

	// sum returns null on empty table
	sum := evaluator.Eval(dict.Pairs["sum"], dict.Env)
	if sum.Type() != evaluator.NULL_OBJ {
		t.Fatalf("expected null for sum on empty, got %s", sum.Type())
	}

	// first returns null on empty table
	first := evaluator.Eval(dict.Pairs["first"], dict.Env)
	if first.Type() != evaluator.NULL_OBJ {
		t.Fatalf("expected null for first on empty, got %s", first.Type())
	}
}
