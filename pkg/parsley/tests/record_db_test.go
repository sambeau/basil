package tests

import (
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
)

// =============================================================================
// Phase 5: Database Integration Tests
// =============================================================================
// Tests for FEAT-091 Phase 5 requirements:
// - TEST-DB-001: Query returns Record (find, first, last, findBy)
// - TEST-DB-002: Query returns Table of Records (all, where)
// - TEST-DB-003: Auto-validation on query return
// - TEST-DB-004: Table row access returns Record
// - TEST-DB-005: Record from DB has no errors

// evalRecordDBTest helper that evaluates Parsley code using the full evaluator
func evalRecordDBTest(t *testing.T, input string) evaluator.Object {
	t.Helper()
	evaluator.ClearDBConnections()
	result, err := parsley.Eval(input)
	if err != nil {
		t.Fatalf("evaluation error: %v", err)
	}
	if result == nil || result.Value == nil {
		t.Fatal("result is nil")
	}
	return result.Value
}

// =============================================================================
// TEST-DB-001: Query ?-> * returns Record
// =============================================================================

func TestDBQueryReturnsRecord(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "find() returns Record",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"

@schema User {
    id: uuid
    name: string
    age: int
}

let Users = db.bind(User, "users")
let _ = Users.insert({name: "Alice", age: 30})
let all = Users.all()
let id = all[0].id
let record = Users.find(id)
record.type()`,
			expected: "record",
		},
		{
			name: "find() Record has correct name",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"

@schema User2 {
    id: uuid
    name: string
    age: int
}

let Users = db.bind(User2, "users")
let _ = Users.insert({name: "Alice", age: 30})
let all = Users.all()
let id = all[0].id
let record = Users.find(id)
record.name`,
			expected: "Alice",
		},
		{
			name: "first() returns Record",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT, priority INTEGER)"

@schema Item {
    id: uuid
    name: string
    priority: int
}

let Items = db.bind(Item, "items")
let _ = Items.insert({name: "First", priority: 1})
let _ = Items.insert({name: "Second", priority: 2})
Items.first().type()`,
			expected: "record",
		},
		{
			name: "first() Record has correct name",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT, priority INTEGER)"

@schema Item2 {
    id: uuid
    name: string
    priority: int
}

let Items = db.bind(Item2, "items")
let _ = Items.insert({name: "OnlyItem", priority: 1})
Items.first().name`,
			expected: "OnlyItem",
		},
		{
			name: "last() returns Record",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT, priority INTEGER)"

@schema Item3 {
    id: uuid
    name: string
    priority: int
}

let Items = db.bind(Item3, "items")
let _ = Items.insert({name: "First", priority: 1})
let _ = Items.insert({name: "Last", priority: 2})
Items.last().type()`,
			expected: "record",
		},
		{
			name: "last() Record has correct name",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT, priority INTEGER)"

@schema Item4 {
    id: uuid
    name: string
    priority: int
}

let Items = db.bind(Item4, "items")
let _ = Items.insert({name: "OnlyItem", priority: 1})
Items.last().name`,
			expected: "OnlyItem",
		},
		{
			name: "findBy() returns Record",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE people (id TEXT PRIMARY KEY, name TEXT, city TEXT)"

@schema Person {
    id: uuid
    name: string
    city: string
}

let People = db.bind(Person, "people")
let _ = People.insert({name: "Bob", city: "NYC"})
let _ = People.insert({name: "Carol", city: "LA"})
People.findBy({city: "NYC"}).type()`,
			expected: "record",
		},
		{
			name: "findBy() Record has correct name",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE people (id TEXT PRIMARY KEY, name TEXT, city TEXT)"

@schema Person2 {
    id: uuid
    name: string
    city: string
}

let People = db.bind(Person2, "people")
let _ = People.insert({name: "Bob", city: "NYC"})
let _ = People.insert({name: "Carol", city: "LA"})
People.findBy({city: "NYC"}).name`,
			expected: "Bob",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordDBTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// =============================================================================
// TEST-DB-002: Query ??-> * returns Table of Records
// =============================================================================

func TestDBQueryReturnsTableOfRecords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "all() returns Table",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE products (id TEXT PRIMARY KEY, name TEXT, price REAL)"

@schema Product {
    id: uuid
    name: string
    price: float
}

let Products = db.bind(Product, "products")
let _ = Products.insert({name: "Widget", price: 9.99})
let _ = Products.insert({name: "Gadget", price: 19.99})
Products.all().type()`,
			expected: "table",
		},
		{
			name: "all() Table has correct row count",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE products (id TEXT PRIMARY KEY, name TEXT, price REAL)"

@schema Product2 {
    id: uuid
    name: string
    price: float
}

let Products = db.bind(Product2, "products")
let _ = Products.insert({name: "Widget", price: 9.99})
let _ = Products.insert({name: "Gadget", price: 19.99})
Products.all().length`,
			expected: "2",
		},
		{
			name: "all() Table has schema",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE products (id TEXT PRIMARY KEY, name TEXT, price REAL)"

@schema Product3 {
    id: uuid
    name: string
    price: float
}

let Products = db.bind(Product3, "products")
let _ = Products.insert({name: "Widget", price: 9.99})
Products.all().schema.name`,
			expected: "Product3",
		},
		{
			name: "where() returns Table",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE products (id TEXT PRIMARY KEY, name TEXT, price REAL)"

@schema Product4 {
    id: uuid
    name: string
    price: float
}

let Products = db.bind(Product4, "products")
let _ = Products.insert({name: "Cheap", price: 5.00})
let _ = Products.insert({name: "Expensive", price: 100.00})
Products.where({price: 5.00}).type()`,
			expected: "table",
		},
		{
			name: "where() returns filtered Table",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE products (id TEXT PRIMARY KEY, name TEXT, price REAL)"

@schema Product5 {
    id: uuid
    name: string
    price: float
}

let Products = db.bind(Product5, "products")
let _ = Products.insert({name: "Cheap", price: 5.00})
let _ = Products.insert({name: "Expensive", price: 100.00})
Products.where({price: 5.00}).length`,
			expected: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordDBTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// =============================================================================
// TEST-DB-003: Auto-validation on query return
// =============================================================================

func TestDBRecordAutoValidation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "find() returns valid Record (isValid)",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"

@schema UserVal {
    id: uuid
    name: string
    age: int
}

let Users = db.bind(UserVal, "users")
let _ = Users.insert({name: "Alice", age: 30})
let all = Users.all()
let id = all[0].id
let record = Users.find(id)
record.isValid()`,
			expected: "true",
		},
		{
			name: "find() returns validated Record (isValid again)",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, age INTEGER)"

@schema UserVal2 {
    id: uuid
    name: string
    age: int
}

let Users = db.bind(UserVal2, "users")
let _ = Users.insert({name: "Alice", age: 30})
let all = Users.all()
let id = all[0].id
let record = Users.find(id)
record.isValid()`,
			expected: "true",
		},
		{
			name: "first() returns valid Record",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT)"

@schema ItemVal {
    id: uuid
    name: string
}

let Items = db.bind(ItemVal, "items")
let _ = Items.insert({name: "Test"})
Items.first().isValid()`,
			expected: "true",
		},
		{
			name: "last() returns valid Record",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT)"

@schema ItemVal2 {
    id: uuid
    name: string
}

let Items = db.bind(ItemVal2, "items")
let _ = Items.insert({name: "Test"})
Items.last().isValid()`,
			expected: "true",
		},
		{
			name: "findBy() returns valid Record",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT)"

@schema ItemVal3 {
    id: uuid
    name: string
}

let Items = db.bind(ItemVal3, "items")
let _ = Items.insert({name: "Test"})
Items.findBy({name: "Test"}).isValid()`,
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordDBTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// =============================================================================
// TEST-DB-004: Table row access returns Record
// =============================================================================

func TestDBTableRowAccessReturnsRecord(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "table[n] returns Record when table has schema",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT, qty INTEGER)"

@schema ItemRow {
    id: uuid
    name: string
    qty: int
}

let Items = db.bind(ItemRow, "items")
let _ = Items.insert({name: "Apple", qty: 10})
let _ = Items.insert({name: "Banana", qty: 20})
let table = Items.all()
table[0].type()`,
			expected: "record",
		},
		{
			name: "table[n] Record has correct field values",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT, qty INTEGER)"

@schema ItemRow2 {
    id: uuid
    name: string
    qty: int
}

let Items = db.bind(ItemRow2, "items")
let _ = Items.insert({name: "Apple", qty: 10})
let _ = Items.insert({name: "Banana", qty: 20})
let table = Items.all()
table[0].name`,
			expected: "Apple",
		},
		{
			name: "table[n] Record is valid",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT, qty INTEGER)"

@schema ItemRow3 {
    id: uuid
    name: string
    qty: int
}

let Items = db.bind(ItemRow3, "items")
let _ = Items.insert({name: "Apple", qty: 10})
let table = Items.all()
table[0].isValid()`,
			expected: "true",
		},
		{
			name: "table[n] Record has schema reference",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT)"

@schema ItemSchema {
    id: uuid
    name: string
}

let Items = db.bind(ItemSchema, "items")
let _ = Items.insert({name: "Test"})
let table = Items.all()
let record = table[0]
record.schema().name`,
			expected: "ItemSchema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordDBTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// =============================================================================
// TEST-DB-005: Record from DB has no errors
// =============================================================================

func TestDBRecordHasNoErrors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "find() returns Record with empty errors",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"

@schema UserErr {
    id: uuid
    name: string
}

let Users = db.bind(UserErr, "users")
let _ = Users.insert({name: "Alice"})
let all = Users.all()
let id = all[0].id
let record = Users.find(id)
record.errors().keys().length()`,
			expected: "0",
		},
		{
			name: "Record from DB has no error for name field",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"

@schema UserErr2 {
    id: uuid
    name: string
}

let Users = db.bind(UserErr2, "users")
let _ = Users.insert({name: "Bob"})
let record = Users.first()
record.hasError("name")`,
			expected: "false",
		},
		{
			name: "Record from DB has null error for valid field",
			input: `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"

@schema UserErr3 {
    id: uuid
    name: string
}

let Users = db.bind(UserErr3, "users")
let _ = Users.insert({name: "Carol"})
let record = Users.first()
record.error("name")`,
			expected: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evalRecordDBTest(t, tt.input)
			if result.Type() == evaluator.ERROR_OBJ {
				t.Fatalf("evaluation error: %s", result.Inspect())
			}
			if result.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Inspect())
			}
		})
	}
}

// =============================================================================
// TEST-DB-006: Record type() method returns "record"
// =============================================================================

func TestDBRecordTypeMethod(t *testing.T) {
	input := `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"

@schema UserType {
    id: uuid
    name: string
}

let Users = db.bind(UserType, "users")
let _ = Users.insert({name: "Test"})
let record = Users.first()
record.type()`

	result := evalRecordDBTest(t, input)
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}
	if result.Inspect() != "record" {
		t.Errorf("expected type() to return 'record', got %s", result.Inspect())
	}
}

// =============================================================================
// TEST-DB-007: find() returns null for non-existent record
// =============================================================================

func TestDBFindReturnsNullForMissing(t *testing.T) {
	input := `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"

@schema UserMissing {
    id: uuid
    name: string
}

let Users = db.bind(UserMissing, "users")
Users.find("non-existent-id")`

	result := evalRecordDBTest(t, input)
	if result.Type() != evaluator.NULL_OBJ {
		t.Errorf("find() should return null for missing record, got %s", result.Type())
	}
}

// =============================================================================
// TEST-DB-008: Table.schema property returns schema
// =============================================================================

func TestDBTableSchemaMethod(t *testing.T) {
	input := `
let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE items (id TEXT PRIMARY KEY, name TEXT)"

@schema ItemTableSchema {
    id: uuid
    name: string
}

let Items = db.bind(ItemTableSchema, "items")
let _ = Items.insert({name: "Test"})
let table = Items.all()
table.schema.name`

	result := evalRecordDBTest(t, input)
	if result.Type() == evaluator.ERROR_OBJ {
		t.Fatalf("evaluation error: %s", result.Inspect())
	}
	if result.Inspect() != "ItemTableSchema" {
		t.Errorf("expected schema name ItemTableSchema, got %s", result.Inspect())
	}
}
