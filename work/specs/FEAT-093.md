# FEAT-093: Schema-Driven Database Mutations

**Status**: Implemented  
**Created**: 2026-01-16  
**Branch**: `feat/FEAT-093-schema-driven-mutations`

## Summary

Add support for passing Record or Table objects directly to `insert`, `update`, `save`, and `delete` methods on bound tables, using the schema to determine field mappings and primary key.

## Motivation

Current API requires extracting data into dictionaries:
```parsley
let user = User({name: "Alice", email: "alice@example.com"})
users.insert({name: user.name, email: user.email})  // tedious
```

With schema-driven mutations:
```parsley
let user = User({name: "Alice", email: "alice@example.com"})
users.insert(user)  // simple!
```

## Design Decisions

1. **Overloaded methods** - Same method names (`insert`, `update`, `delete`) detect argument type
2. **New `save` method** - Upsert semantics (INSERT ... ON CONFLICT DO UPDATE)
3. **Primary key convention** - Field named "id" is the primary key
4. **Bulk operations** - Table argument processes all rows as a single transaction
5. **Schema matching** - Record/Table schema must match TableBinding schema (or be untyped)

## API Specification

### insert(record: Record)
Insert a single record. Returns the inserted row.

```parsley
let user = User({name: "Alice", email: "alice@example.com"})
let inserted = users.insert(user)
// inserted.id is auto-generated if not provided
```

### insert(table: Table)
Insert all rows from a table. Returns count of inserted rows.

```parsley
let newUsers = Table([
    {name: "Alice", email: "alice@example.com"},
    {name: "Bob", email: "bob@example.com"}
])
let result = users.insert(newUsers)
// result.inserted == 2
```

### update(record: Record)
Update a row using the record's primary key. Returns the updated row.

```parsley
let user = users.find(1)
let updated = users.update(user.set("name", "Alice Smith"))
```

**Error cases:**
- Record missing primary key: `"Cannot update: record has no primary key value"`
- Primary key doesn't exist: Returns `null`

### update(table: Table)
Update multiple rows. Returns count of updated rows.

```parsley
let admins = users.where({role: "admin"})
let updated = admins.map(fn(u) { u.set("verified", true) })
let result = users.update(Table(updated))
// result.updated == N
```

### save(record: Record)
Upsert: insert if new, update if exists (based on primary key).

```parsley
let user = User({id: "123", name: "Alice"})
let saved = users.save(user)
// Inserts if id "123" doesn't exist, updates if it does
```

### save(table: Table)
Bulk upsert. Returns summary.

```parsley
let result = users.save(usersTable)
// result.inserted, result.updated, result.total
```

### delete(record: Record)
Delete by primary key. Returns affected count.

```parsley
let user = users.find(1)
users.delete(user)  // Deletes user with id=1
```

### delete(table: Table)
Delete multiple rows. Returns total affected count.

```parsley
let inactive = users.where({active: false})
let result = users.delete(inactive)
// result.affected == N
```

## Schema Matching Rules

1. If Record has a schema, it must match the TableBinding's schema
2. Untyped Tables are allowed (their rows are treated as dictionaries)
3. Schema mismatch returns an error: `"Schema mismatch: expected User, got Product"`

## Primary Key Detection

1. TableBinding uses the DSLSchema's primary key field
2. By convention, a field named "id" is the primary key
3. Future: Add explicit `primary: true` metadata support

## Implementation Notes

### Phase 1: Record Support (this feature)
- Overload `insert`, `update`, `delete` to accept Record
- Add `save` method
- Primary key = field named "id"

### Future Enhancement
- Add `Primary bool` field to DSLSchemaField
- Support `id (primary: true)` constraint syntax
- Support composite primary keys

## Error Codes

| Code | Message |
|------|---------|
| DB-0016 | Cannot update: record has no primary key value |
| DB-0017 | Cannot delete: record has no primary key value |
| VAL-0022 | Schema mismatch: expected {expected}, got {actual} |

## Test Cases

1. Insert single Record (with/without id)
2. Insert Table (multiple rows)
3. Update Record by primary key
4. Update Table (multiple rows)
5. Save Record (insert new)
6. Save Record (update existing)
7. Save Table (mixed insert/update)
8. Delete Record
9. Delete Table
10. Error: update/delete without primary key
11. Error: schema mismatch

## Implementation Details

### Files Modified

1. **pkg/parsley/evaluator/stdlib_dsl_schema.go**
   - Added `Primary bool` field to `DSLSchemaField` struct
   - Added `PrimaryKey()` method to `DSLSchema` - returns field name with `Primary=true`
   - Schema parsing auto-sets `Primary: true` for field named "id"

2. **pkg/parsley/evaluator/stdlib_schema_table_binding.go**
   - `executeInsert()` - Added type dispatch for Record/Table arguments
   - `executeInsertRecord()` - Converts Record to dict, validates schema match
   - `executeInsertTable()` - Iterates rows with individual inserts, returns count
   - `executeUpdate()` - Added type dispatch for Record/Table arguments  
   - `executeUpdateRecord()` - Extracts PK from record, calls executeUpdateByID
   - `executeUpdateTable()` - Batch update with counts
   - `executeUpdateByID()` - Fixed to support DSLSchema (not just legacy Schema)
   - Added `save` method dispatcher
   - `executeSave()` - Type dispatch for save operation
   - `executeSaveRecord()` - Single record upsert
   - `executeSaveTable()` - Batch upsert with counts
   - `executeSaveDictionary()` - Core upsert using INSERT...ON CONFLICT
   - `executeDelete()` - Added type dispatch for Record/Table arguments
   - `executeDeleteRecord()` - Extract PK and delete
   - `executeDeleteTable()` - Batch delete with count

3. **pkg/parsley/evaluator/errors.go**
   - DB-0016: "Cannot update: record has no primary key value"
   - DB-0017: "Cannot delete: record has no primary key value"
   - VAL-0022: "Schema mismatch: expected {{.Expected}}, got {{.Got}}"

4. **pkg/parsley/tests/schema_mutation_test.go** (new file)
   - 15+ test cases covering all scenarios

### Key Implementation Decisions

1. **Partial update validation**: `executeUpdateByID` skips DSLSchema validation
   because partial updates don't include all required fields. Type validation
   only applies if legacy `Schema` is present.

2. **Null ID handling**: When inserting a Record with explicit `id: null`, we
   treat it as missing and auto-generate. Check for null value, not just key existence.

3. **Save return value**: For Table saves, we return `{saved: N, total: N}` rather
   than distinguishing inserts vs updates since SQLite upsert doesn't easily tell us.

4. **Record vs Dictionary detection**: Record has `ToDictionary()` method. Table
   iteration returns raw Dictionaries (not Records) - this is a design decision
   in the Table type that affects how users work with table data.
