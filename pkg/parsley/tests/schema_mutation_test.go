package tests

import (
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/parsley"
)

// =============================================================================
// FEAT-093: Schema-Driven Database Mutations Tests
// =============================================================================
// Tests for Record/Table-based insert, update, save, and delete operations

func evalMutationTest(t *testing.T, input string) evaluator.Object {
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
// Insert Tests
// =============================================================================

func TestInsertRecord(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
    email: email
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, email TEXT)"
let users = db.bind(User, "users")

// Create a record and insert it
let user = User({name: "Alice", email: "alice@example.com"})
let inserted = users.insert(user)
inserted.name
`
	result := evalMutationTest(t, input)
	if str, ok := result.(*evaluator.String); !ok || str.Value != "Alice" {
		t.Errorf("Expected 'Alice', got %v", result)
	}
}

func TestInsertRecordWithExplicitID(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"
let users = db.bind(User, "users")

let user = User({id: "user-123", name: "Bob"})
let inserted = users.insert(user)
inserted.id
`
	result := evalMutationTest(t, input)
	if str, ok := result.(*evaluator.String); !ok || str.Value != "user-123" {
		t.Errorf("Expected 'user-123', got %v", result)
	}
}

func TestInsertTable(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"
let users = db.bind(User, "users")

let newUsers = table([
    {name: "Alice"},
    {name: "Bob"},
    {name: "Charlie"}
])
let result = users.insert(newUsers)
result.inserted
`
	result := evalMutationTest(t, input)
	if intVal, ok := result.(*evaluator.Integer); !ok || intVal.Value != 3 {
		t.Errorf("Expected 3 inserted, got %v", result)
	}
}

// =============================================================================
// Update Tests
// =============================================================================

func TestUpdateRecord(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
    email: email
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, email TEXT)"
let users = db.bind(User, "users")

// Insert initial user
let _ = users.insert({id: "user-1", name: "Alice", email: "alice@example.com"})

// Fetch and update using record.update({...})
let user = users.find("user-1")
let updated = users.update(user.update({name: "Alice Smith"}))
updated.name
`
	result := evalMutationTest(t, input)
	if str, ok := result.(*evaluator.String); !ok || str.Value != "Alice Smith" {
		t.Errorf("Expected 'Alice Smith', got %v", result)
	}
}

func TestUpdateTable(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
    verified: bool
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, verified INTEGER)"
let users = db.bind(User, "users")

// Insert users
let _ = users.insert({id: "1", name: "Alice", verified: false})
let _ = users.insert({id: "2", name: "Bob", verified: false})

// Create a table with updated records
let allUsers = users.all()
let u1 = allUsers[0].update({verified: true})
let u2 = allUsers[1].update({verified: true})
// Convert records to dicts for table()
let result = users.update(table([u1.data(), u2.data()]))
result.updated
`
	result := evalMutationTest(t, input)
	if intVal, ok := result.(*evaluator.Integer); !ok || intVal.Value != 2 {
		t.Errorf("Expected 2 updated, got %v", result)
	}
}

func TestUpdateRecordWithoutPrimaryKey(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"
let users = db.bind(User, "users")

// Try to update a record without id
let user = User({name: "Alice"})
users.update(user)
`
	evaluator.ClearDBConnections()
	result, err := parsley.Eval(input)
	if err == nil && result != nil && result.Value != nil {
		if errObj, ok := result.Value.(*evaluator.Error); ok {
			if !strings.Contains(errObj.Code, "DB-0016") {
				t.Errorf("Expected DB-0016 error, got %s", errObj.Code)
			}
		} else {
			t.Error("Expected an error for update without primary key")
		}
	}
}

// =============================================================================
// Save Tests (Upsert)
// =============================================================================

func TestSaveNewRecord(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"
let users = db.bind(User, "users")

// Save a new record
let user = User({id: "user-1", name: "Alice"})
let saved = users.save(user)
saved.name
`
	result := evalMutationTest(t, input)
	if str, ok := result.(*evaluator.String); !ok || str.Value != "Alice" {
		t.Errorf("Expected 'Alice', got %v", result)
	}
}

func TestSaveExistingRecord(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"
let users = db.bind(User, "users")

// Insert initial
let _ = users.insert({id: "user-1", name: "Alice"})

// Save (update) existing
let user = User({id: "user-1", name: "Alice Updated"})
let saved = users.save(user)
saved.name
`
	result := evalMutationTest(t, input)
	if str, ok := result.(*evaluator.String); !ok || str.Value != "Alice Updated" {
		t.Errorf("Expected 'Alice Updated', got %v", result)
	}
}

func TestSaveTable(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"
let users = db.bind(User, "users")

// Insert one user
let _ = users.insert({id: "1", name: "Alice"})

// Save table with mix of new and existing
let mixedUsers = table([
    {id: "1", name: "Alice Updated"},  // existing
    {id: "2", name: "Bob"}             // new
])
let result = users.save(mixedUsers)
result.total
`
	result := evalMutationTest(t, input)
	if intVal, ok := result.(*evaluator.Integer); !ok || intVal.Value != 2 {
		t.Errorf("Expected 2 total, got %v", result)
	}
}

// =============================================================================
// Delete Tests
// =============================================================================

func TestDeleteRecord(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"
let users = db.bind(User, "users")

// Insert and delete
let _ = users.insert({id: "user-1", name: "Alice"})
let user = users.find("user-1")
let result = users.delete(user)
result.affected
`
	result := evalMutationTest(t, input)
	if intVal, ok := result.(*evaluator.Integer); !ok || intVal.Value != 1 {
		t.Errorf("Expected 1 affected, got %v", result)
	}
}

func TestDeleteTable(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"
let users = db.bind(User, "users")

// Insert multiple
let _ = users.insert({id: "1", name: "Alice"})
let _ = users.insert({id: "2", name: "Bob"})
let _ = users.insert({id: "3", name: "Charlie"})

// Delete multiple by getting first two users
let toDelete = users.all().where(fn(u) { u.id == "1" || u.id == "2" })
let result = users.delete(toDelete)
result.affected
`
	result := evalMutationTest(t, input)
	if intVal, ok := result.(*evaluator.Integer); !ok || intVal.Value != 2 {
		t.Errorf("Expected 2 affected, got %v", result)
	}
}

func TestDeleteRecordWithoutPrimaryKey(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"
let users = db.bind(User, "users")

// Try to delete a record without id
let user = User({name: "Alice"})
users.delete(user)
`
	evaluator.ClearDBConnections()
	result, err := parsley.Eval(input)
	if err == nil && result != nil && result.Value != nil {
		if errObj, ok := result.Value.(*evaluator.Error); ok {
			if !strings.Contains(errObj.Code, "DB-0017") {
				t.Errorf("Expected DB-0017 error, got %s", errObj.Code)
			}
		} else {
			t.Error("Expected an error for delete without primary key")
		}
	}
}

// =============================================================================
// Schema Mismatch Tests
// =============================================================================

func TestInsertSchemaMismatch(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
}

@schema Product {
    id: id
    title: string
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"
let users = db.bind(User, "users")

// Try to insert a Product record into users table
let product = Product({id: "p1", title: "Widget"})
users.insert(product)
`
	evaluator.ClearDBConnections()
	result, err := parsley.Eval(input)
	if err == nil && result != nil && result.Value != nil {
		if errObj, ok := result.Value.(*evaluator.Error); ok {
			if !strings.Contains(errObj.Code, "VAL-0022") {
				t.Errorf("Expected VAL-0022 error, got %s", errObj.Code)
			}
		} else {
			t.Error("Expected schema mismatch error")
		}
	}
}

// =============================================================================
// Primary Key Detection Tests
// =============================================================================

func TestPrimaryKeyFieldDetection(t *testing.T) {
	// Verify that field named "id" is marked as primary key
	input := `
@schema User {
    id: id
    name: string
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"
let users = db.bind(User, "users")

// Insert without id - should auto-generate
let user = User({name: "Alice"})
let inserted = users.insert(user)
// If id was generated, it should be a non-empty string (not null)
inserted.id.length() > 0
`
	result := evalMutationTest(t, input)
	if boolVal, ok := result.(*evaluator.Boolean); !ok || !boolVal.Value {
		t.Errorf("Expected id to be non-empty string, got %v", result)
	}
}

// =============================================================================
// Backward Compatibility Tests
// =============================================================================

func TestInsertDictionaryStillWorks(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"
let users = db.bind(User, "users")

// Old-style dictionary insert still works
let inserted = users.insert({name: "Alice"})
inserted.name
`
	result := evalMutationTest(t, input)
	if str, ok := result.(*evaluator.String); !ok || str.Value != "Alice" {
		t.Errorf("Expected 'Alice', got %v", result)
	}
}

func TestUpdateByIdStillWorks(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"
let users = db.bind(User, "users")

let _ = users.insert({id: "user-1", name: "Alice"})

// Old-style update(id, dict) still works
let updated = users.update("user-1", {name: "Alice Smith"})
updated.name
`
	result := evalMutationTest(t, input)
	if str, ok := result.(*evaluator.String); !ok || str.Value != "Alice Smith" {
		t.Errorf("Expected 'Alice Smith', got %v", result)
	}
}

func TestDeleteByIdStillWorks(t *testing.T) {
	input := `
@schema User {
    id: id
    name: string
}

let db = @sqlite(":memory:")
let _ = db <=!=> "CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT)"
let users = db.bind(User, "users")

let _ = users.insert({id: "user-1", name: "Alice"})

// Old-style delete(id) still works
let result = users.delete("user-1")
result.affected
`
	result := evalMutationTest(t, input)
	if intVal, ok := result.(*evaluator.Integer); !ok || intVal.Value != 1 {
		t.Errorf("Expected 1 affected, got %v", result)
	}
}
