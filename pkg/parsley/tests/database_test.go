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

		if !strings.Contains(errObj.Message, "@DB is only available in Basil server handlers") {
			t.Fatalf("Unexpected error message: %s", errObj.Message)
		}
	})

	t.Run("returns basil connection", func(t *testing.T) {
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
			name: "SQL tag with params",
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
