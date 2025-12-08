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
let db = SQLITE(":memory:")
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
let db = SQLITE(":memory:")
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
let db = SQLITE(":memory:")
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

	filtered := evaluator.Eval(dict.Pairs["filtered"], dict.Env).(*evaluator.Array)
	if len(filtered.Elements) != 1 {
		t.Fatalf("expected 1 filtered row, got %d", len(filtered.Elements))
	}
	row := filtered.Elements[0].(*evaluator.Dictionary)
	age := evaluator.Eval(row.Pairs["age"], row.Env).(*evaluator.Integer).Value
	if age != 25 {
		t.Fatalf("expected age 25, got %d", age)
	}

	paged := evaluator.Eval(dict.Pairs["paged"], dict.Env).(*evaluator.Array)
	if len(paged.Elements) != 1 {
		t.Fatalf("expected paginated result length 1, got %d", len(paged.Elements))
	}
	second := paged.Elements[0].(*evaluator.Dictionary)
	name := evaluator.Eval(second.Pairs["name"], second.Env).(*evaluator.String).Value
	if name != "Bob" {
		t.Fatalf("expected second row Bob, got %s", name)
	}
}

func TestSchemaTableDelete(t *testing.T) {
	evaluator.ClearDBConnections()
	input := `
let schema = import @std/schema
let db = SQLITE(":memory:")
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
let db = SQLITE(":memory:")
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
