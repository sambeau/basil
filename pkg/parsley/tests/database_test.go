package tests

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/ast"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestSQLiteConnection(t *testing.T) {
	t.Run("Create SQLite connection", func(t *testing.T) {
		l := lexer.New(`let db = @sqlite(":memory:")`)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}

		env := evaluator.NewEnvironment()
		evaluator.Eval(program, env)

		db, ok := env.Get("db")
		if !ok {
			t.Fatal("db not found in environment")
		}

		if db.Type() != "DB_CONNECTION" {
			t.Errorf("Expected DB_CONNECTION object, got %s", db.Type())
		}
	})

	t.Run("Check connection type", func(t *testing.T) {
		l := lexer.New(`let db = @sqlite(":memory:")`)
		p := parser.New(l)
		program := p.ParseProgram()
		env := evaluator.NewEnvironment()
		evaluator.Eval(program, env)

		db, _ := env.Get("db")
		if db.Inspect() != "<DBConnection driver=sqlite>" {
			t.Errorf("Expected %q, got %q", "<DBConnection driver=sqlite>", db.Inspect())
		}
	})

	t.Run("Ping connection", func(t *testing.T) {
		l := lexer.New(`let db = @sqlite(":memory:")` + "\n" + `db.ping()`)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}

		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)

		if err, ok := result.(*evaluator.Error); ok {
			t.Fatalf("Eval error: %s", err.Message)
		}

		// db.ping() should be the only non-NULL result
		boolean, ok := result.(*evaluator.Boolean)
		if !ok {
			t.Errorf("Expected Boolean, got %T: %v", result, result)
			return
		}
		if !boolean.Value {
			t.Errorf("Expected true, got %v", result.Inspect())
		}
	})

	t.Run("Begin transaction", func(t *testing.T) {
		l := lexer.New(`let db = @sqlite(":memory:")` + "\n" + `db.begin()`)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}

		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)

		if err, ok := result.(*evaluator.Error); ok {
			t.Fatalf("Eval error: %s", err.Message)
		}

		// db.begin() should be the only non-NULL result
		boolean, ok := result.(*evaluator.Boolean)
		if !ok {
			t.Errorf("Expected Boolean, got %T: %v", result, result)
			return
		}
		if !boolean.Value {
			t.Errorf("Expected true, got %v", result.Inspect())
		}
	})

	t.Run("Begin and commit transaction", func(t *testing.T) {
		l := lexer.New(`let db = @sqlite(":memory:")` + "\n" + `let _ = db.begin()` + "\n" + `db.commit()`)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}

		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)

		if err, ok := result.(*evaluator.Error); ok {
			t.Fatalf("Eval error: %s", err.Message)
		}

		// Only db.commit() should be in output (db.begin() assigned to _)
		boolean, ok := result.(*evaluator.Boolean)
		if !ok || !boolean.Value {
			t.Errorf("Expected true, got %v", result.Inspect())
		}
	})

	t.Run("Begin and rollback transaction", func(t *testing.T) {
		l := lexer.New(`let db = @sqlite(":memory:")` + "\n" + `let _ = db.begin()` + "\n" + `db.rollback()`)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}

		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)

		if err, ok := result.(*evaluator.Error); ok {
			t.Fatalf("Eval error: %s", err.Message)
		}

		// Only db.rollback() should be in output
		boolean, ok := result.(*evaluator.Boolean)
		if !ok || !boolean.Value {
			t.Errorf("Expected true, got %v", result.Inspect())
		}
	})
}

func TestDBLiteralBasilOnly(t *testing.T) {
	t.Run("errors outside basil", func(t *testing.T) {
		l := lexer.New(`@DB`)
		p := parser.New(l)
		program := p.ParseProgram()

		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)

		errObj, ok := result.(*evaluator.Error)
		if !ok {
			t.Fatalf("Expected error, got %T", result)
		}

		if !strings.Contains(errObj.Message, "@DB is only available in Basil server context") {
			t.Fatalf("Unexpected error message: %s", errObj.Message)
		}
	})

	t.Run("returns ServerDB when set", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatalf("Failed to open sqlite: %v", err)
		}
		defer db.Close()

		env := evaluator.NewEnvironment()
		conn := evaluator.NewManagedDBConnection(db, "sqlite")
		env.ServerDB = conn

		l := lexer.New(`@DB`)
		p := parser.New(l)
		program := p.ParseProgram()

		result := evaluator.Eval(program, env)
		dbConn, ok := result.(*evaluator.DBConnection)
		if !ok {
			t.Fatalf("Expected DBConnection, got %T: %v", result, result.Inspect())
		}

		if dbConn.Driver != "sqlite" {
			t.Errorf("Expected driver 'sqlite', got %s", dbConn.Driver)
		}
		if !dbConn.Managed {
			t.Errorf("Expected managed connection for @DB")
		}
	})

	t.Run("falls back to BasilCtx when ServerDB is nil", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatalf("Failed to open sqlite: %v", err)
		}
		defer db.Close()

		env := evaluator.NewEnvironment()
		conn := evaluator.NewManagedDBConnection(db, "sqlite")
		basilDict := &evaluator.Dictionary{
			Pairs: map[string]ast.Expression{
				"sqlite": &ast.ObjectLiteralExpression{Obj: conn},
			},
			Env: env,
		}
		env.BasilCtx = basilDict
		env.Set("basil", basilDict)

		l := lexer.New(`@DB`)
		p := parser.New(l)
		program := p.ParseProgram()

		result := evaluator.Eval(program, env)
		dbConn, ok := result.(*evaluator.DBConnection)
		if !ok {
			t.Fatalf("Expected DBConnection, got %T", result)
		}

		if dbConn.Driver != "sqlite" {
			t.Errorf("Expected driver 'sqlite', got %s", dbConn.Driver)
		}
		if !dbConn.Managed {
			t.Errorf("Expected managed connection for @DB")
		}
	})
}

func TestSQLiteQueries(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(*testing.T, evaluator.Object)
	}{
		{
			name: "Execute CREATE TABLE",
			input: `
				let db = @sqlite(":memory:")
				let result = db <=!=> "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)"
				result
			`,
			check: func(t *testing.T, result evaluator.Object) {
				dict, ok := result.(*evaluator.Dictionary)
				if !ok {
					t.Fatalf("Expected Dictionary, got %T", result)
				}
				// Check that affected exists
				if _, hasAffected := dict.Pairs["affected"]; !hasAffected {
					t.Error("Result should have 'affected' property")
				}
			},
		},
		{
			name: "Execute INSERT",
			input: `
				let db = @sqlite(":memory:")
				let _ = db <=!=> "DROP TABLE IF EXISTS test_users"
				let _ = db <=!=> "CREATE TABLE test_users (id INTEGER PRIMARY KEY, name TEXT)"
				let result = db <=!=> "INSERT INTO test_users (name) VALUES ('Alice')"
				result
			`,
			check: func(t *testing.T, result evaluator.Object) {
				dict, ok := result.(*evaluator.Dictionary)
				if !ok {
					t.Fatalf("Expected Dictionary, got %T", result)
				}
				// Check for affected rows
				affectedExpr, ok := dict.Pairs["affected"]
				if !ok {
					t.Fatal("Result should have 'affected' property")
				}
				affected := evaluator.Eval(affectedExpr, dict.Env)
				affectedInt, ok := affected.(*evaluator.Integer)
				if !ok || affectedInt.Value != 1 {
					t.Errorf("Expected affected=1, got %v", affected.Inspect())
				}
			},
		},
		{
			name: "Query single row with <=?=>",
			input: `
				let db = @sqlite(":memory:")
				let _ = db <=!=> "DROP TABLE IF EXISTS query_users"
				let _ = db <=!=> "CREATE TABLE query_users (id INTEGER PRIMARY KEY, name TEXT)"
				let _ = db <=!=> "INSERT INTO query_users (name) VALUES ('Alice')"
				let user = db <=?=> "SELECT * FROM query_users WHERE name = 'Alice'"
				user
			`,
			check: func(t *testing.T, result evaluator.Object) {
				dict, ok := result.(*evaluator.Dictionary)
				if !ok {
					t.Fatalf("Expected Dictionary, got %T", result)
				}
				// Check for name field
				if _, hasName := dict.Pairs["name"]; !hasName {
					t.Error("Result should have 'name' field")
				}
			},
		},
		{
			name: "Query multiple rows with <=??=>",
			input: `
				let db = @sqlite(":memory:")
				let _ = db <=!=> "DROP TABLE IF EXISTS many_users"
				let _ = db <=!=> "CREATE TABLE many_users (id INTEGER PRIMARY KEY, name TEXT)"
				let _ = db <=!=> "INSERT INTO many_users (name) VALUES ('Alice')"
				let _ = db <=!=> "INSERT INTO many_users (name) VALUES ('Bob')"
				let users = db <=??=> "SELECT * FROM many_users"
				users
			`,
			check: func(t *testing.T, result evaluator.Object) {
				arr, ok := result.(*evaluator.Array)
				if !ok {
					t.Fatalf("Expected Array, got %T", result)
				}
				if len(arr.Elements) != 2 {
					t.Errorf("Expected 2 users, got %d", len(arr.Elements))
				}
			},
		},
		{
			name: "Query non-existent row returns null",
			input: `
				let db = @sqlite(":memory:")
				let _ = db <=!=> "DROP TABLE IF EXISTS empty_users"
				let _ = db <=!=> "CREATE TABLE empty_users (id INTEGER PRIMARY KEY, name TEXT)"
				let user = db <=?=> "SELECT * FROM empty_users WHERE id = 999"
				user
			`,
			check: func(t *testing.T, result evaluator.Object) {
				if result.Type() != "NULL" {
					t.Errorf("Expected NULL, got %s", result.Type())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if result == nil {
				t.Fatalf("Eval returned nil")
			}

			if err, ok := result.(*evaluator.Error); ok {
				t.Fatalf("Eval returned error: %s", err.Message)
			}

			tt.check(t, result)
		})
	}
}

func TestSQLTag(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(*testing.T, evaluator.Object)
	}{
		{
			name: "SQL tag without params in component",
			input: `
				let db = @sqlite(":memory:")
				let _ = db <=!=> "DROP TABLE IF EXISTS tag_users"
				let _ = db <=!=> "CREATE TABLE tag_users (id INTEGER PRIMARY KEY, name TEXT)"

				let InsertUser = fn(props) {
					<SQL>
						INSERT INTO tag_users (name) VALUES ('Alice')
					</SQL>
				}

				let result = db <=!=> <InsertUser />
				result
			`,
			check: func(t *testing.T, result evaluator.Object) {
				dict, ok := result.(*evaluator.Dictionary)
				if !ok {
					t.Fatalf("Expected Dictionary, got %T", result)
				}
				affectedExpr, ok := dict.Pairs["affected"]
				if !ok {
					t.Fatal("Result should have 'affected' property")
				}
				affected := evaluator.Eval(affectedExpr, dict.Env)
				affectedInt, ok := affected.(*evaluator.Integer)
				if !ok || affectedInt.Value != 1 {
					t.Errorf("Expected affected=1, got %v", affected.Inspect())
				}
			},
		},
		{
			name: "SQL tag with params in component - insert",
			input: `
				let db = @sqlite(":memory:")
				let _ = db <=!=> "DROP TABLE IF EXISTS tag_users"
				let _ = db <=!=> "CREATE TABLE tag_users (id INTEGER PRIMARY KEY, name TEXT)"

				let InsertUser = fn(props) {
					<SQL name={props.name}>
						INSERT INTO tag_users (name) VALUES (?)
					</SQL>
				}

				let _ = db <=!=> <InsertUser name="Alice" />
				let users = db <=??=> "SELECT * FROM tag_users"
				users
			`,
			check: func(t *testing.T, result evaluator.Object) {
				arr, ok := result.(*evaluator.Array)
				if !ok {
					t.Fatalf("Expected Array, got %T", result)
				}
				if len(arr.Elements) != 1 {
					t.Fatalf("Expected 1 row, got %d", len(arr.Elements))
				}
				row, ok := arr.Elements[0].(*evaluator.Dictionary)
				if !ok {
					t.Fatalf("Expected Dictionary row, got %T", arr.Elements[0])
				}
				nameExpr, ok := row.Pairs["name"]
				if !ok {
					t.Fatal("Row should have 'name' column")
				}
				name := evaluator.Eval(nameExpr, row.Env)
				nameStr, ok := name.(*evaluator.String)
				if !ok || nameStr.Value != "Alice" {
					t.Errorf("Expected name='Alice', got %v", name.Inspect())
				}
			},
		},
		{
			name: "SQL tag with params in component - query",
			input: `
				let db = @sqlite(":memory:")
				let _ = db <=!=> "DROP TABLE IF EXISTS tag_users"
				let _ = db <=!=> "CREATE TABLE tag_users (id INTEGER PRIMARY KEY, name TEXT)"
				let _ = db <=!=> "INSERT INTO tag_users (id, name) VALUES (1, 'Alice')"
				let _ = db <=!=> "INSERT INTO tag_users (id, name) VALUES (2, 'Bob')"

				let GetUser = fn(props) {
					<SQL id={props.id}>
						SELECT * FROM tag_users WHERE id = ?
					</SQL>
				}

				let user = db <=?=> <GetUser id={2} />
				user
			`,
			check: func(t *testing.T, result evaluator.Object) {
				dict, ok := result.(*evaluator.Dictionary)
				if !ok {
					t.Fatalf("Expected Dictionary, got %T", result)
				}
				nameExpr, ok := dict.Pairs["name"]
				if !ok {
					t.Fatal("Result should have 'name' column")
				}
				name := evaluator.Eval(nameExpr, dict.Env)
				nameStr, ok := name.(*evaluator.String)
				if !ok || nameStr.Value != "Bob" {
					t.Errorf("Expected name='Bob', got %v", name.Inspect())
				}
			},
		},
		{
			name: "SQL tag with multiple params in component",
			input: `
				let db = @sqlite(":memory:")
				let _ = db <=!=> "DROP TABLE IF EXISTS tag_users"
				let _ = db <=!=> "CREATE TABLE tag_users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)"

				let InsertUser = fn(props) {
					<SQL age={props.age} name={props.name}>
						INSERT INTO tag_users (age, name) VALUES (?, ?)
					</SQL>
				}

				let _ = db <=!=> <InsertUser name="Alice" age={30} />
				let user = db <=?=> "SELECT * FROM tag_users WHERE id = 1"
				user
			`,
			check: func(t *testing.T, result evaluator.Object) {
				dict, ok := result.(*evaluator.Dictionary)
				if !ok {
					t.Fatalf("Expected Dictionary, got %T", result)
				}
				nameExpr, ok := dict.Pairs["name"]
				if !ok {
					t.Fatal("Result should have 'name' column")
				}
				name := evaluator.Eval(nameExpr, dict.Env)
				nameStr, ok := name.(*evaluator.String)
				if !ok || nameStr.Value != "Alice" {
					t.Errorf("Expected name='Alice', got %v", name.Inspect())
				}
				ageExpr, ok := dict.Pairs["age"]
				if !ok {
					t.Fatal("Result should have 'age' column")
				}
				age := evaluator.Eval(ageExpr, dict.Env)
				ageInt, ok := age.(*evaluator.Integer)
				if !ok || ageInt.Value != 30 {
					t.Errorf("Expected age=30, got %v", age.Inspect())
				}
			},
		},
		{
			name: "SQL tag with multi-line content and whitespace trimming",
			input: `
				let query = <SQL>
					SELECT id, name
					FROM users
					WHERE id = 1
				</SQL>
				query.sql
			`,
			check: func(t *testing.T, result evaluator.Object) {
				str, ok := result.(*evaluator.String)
				if !ok {
					t.Fatalf("Expected String, got %T", result)
				}
				// Verify leading/trailing whitespace is trimmed
				if str.Value[0] == '\n' || str.Value[0] == ' ' || str.Value[0] == '\t' {
					t.Errorf("Leading whitespace should be trimmed, got: %q", str.Value)
				}
				lastChar := str.Value[len(str.Value)-1]
				if lastChar == '\n' || lastChar == ' ' || lastChar == '\t' {
					t.Errorf("Trailing whitespace should be trimmed, got: %q", str.Value)
				}
				// Verify content is present
				if !strings.Contains(str.Value, "SELECT id, name") {
					t.Errorf("Expected SQL content, got %s", str.Value)
				}
				if !strings.Contains(str.Value, "FROM users") {
					t.Errorf("Expected FROM clause, got %s", str.Value)
				}
			},
		},
		{
			name: "SQL tag with SQL comments preserved",
			input: `
				let query = <SQL>
					-- This is a SQL comment
					SELECT * FROM users
				</SQL>
				query.sql
			`,
			check: func(t *testing.T, result evaluator.Object) {
				str, ok := result.(*evaluator.String)
				if !ok {
					t.Fatalf("Expected String, got %T", result)
				}
				if !strings.Contains(str.Value, "-- This is a SQL comment") {
					t.Errorf("SQL comments should be preserved, got: %s", str.Value)
				}
				if !strings.Contains(str.Value, "SELECT * FROM users") {
					t.Errorf("SQL query should be present, got: %s", str.Value)
				}
			},
		},
		{
			name: "SQL tag simple inline",
			input: `
				let query = <SQL>SELECT * FROM users</SQL>
				query.sql
			`,
			check: func(t *testing.T, result evaluator.Object) {
				str, ok := result.(*evaluator.String)
				if !ok {
					t.Fatalf("Expected String, got %T", result)
				}
				if str.Value != "SELECT * FROM users" {
					t.Errorf("Expected 'SELECT * FROM users', got %q", str.Value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) != 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}

			env := evaluator.NewEnvironment()
			result := evaluator.Eval(program, env)

			if result == nil {
				t.Fatalf("Eval returned nil")
			}

			if err, ok := result.(*evaluator.Error); ok {
				t.Fatalf("Eval returned error: %s", err.Message)
			}

			tt.check(t, result)
		})
	}
}
