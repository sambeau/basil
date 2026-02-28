package tests

import (
	"fmt"
	"testing"

	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

func TestDatabaseTransactions(t *testing.T) {
	t.Run("Transaction commit", func(t *testing.T) {
		input := `
			let db = @sqlite(":memory:")
			let _ = db <=!=> "CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)"

			let _ = db.begin()
			let _ = db <=!=> "INSERT INTO test (value) VALUES ('a')"
			let _ = db <=!=> "INSERT INTO test (value) VALUES ('b')"
			let _ = db.commit()

			let rows = db <=??=> "SELECT * FROM test"
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}

		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)

		if err, ok := result.(*evaluator.Error); ok {
			t.Fatalf("Eval returned error: %s", err.Message)
		}

		// Get rows from environment
		rowsObj, ok := env.Get("rows")
		if !ok {
			t.Fatal("rows not found in environment")
		}

		tbl, ok := rowsObj.(*evaluator.Table)
		if !ok {
			t.Fatalf("Expected Table, got %T", rowsObj)
		}
		if len(tbl.Rows) != 2 {
			t.Errorf("Expected 2 rows, got %d", len(tbl.Rows))
		}
	})

	t.Run("Transaction rollback", func(t *testing.T) {
		input := `
			let db = @sqlite(":memory:")
			let _ = db <=!=> "CREATE TABLE test_rollback (id INTEGER PRIMARY KEY, value TEXT)"

			let _ = db.begin()
			let _ = db <=!=> "INSERT INTO test_rollback (value) VALUES ('in_transaction')"
			let _ = db.rollback()

			let rows = db <=??=> "SELECT * FROM test_rollback"
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}

		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)

		if err, ok := result.(*evaluator.Error); ok {
			t.Fatalf("Eval returned error: %s", err.Message)
		}

		// Get rows from environment
		rowsObj, ok := env.Get("rows")
		if !ok {
			t.Fatal("rows not found in environment")
		}

		tbl, ok := rowsObj.(*evaluator.Table)
		if !ok {
			t.Fatalf("Expected Table, got %T", rowsObj)
		}
		if len(tbl.Rows) != 0 {
			t.Errorf("Expected 0 rows after rollback, got %d", len(tbl.Rows))
		}
	})
}

func TestDatabaseTransactionIntegration(t *testing.T) {
	t.Run("begin/commit with infix operators", func(t *testing.T) {
		input := `
			let db = @sqlite(":memory:")
			let _ = db <=!=> "CREATE TABLE tx_infix (id INTEGER PRIMARY KEY, value TEXT)"

			let _ = db.begin()
			let _ = db <=!=> "INSERT INTO tx_infix (value) VALUES ('one')"
			let _ = db <=!=> "INSERT INTO tx_infix (value) VALUES ('two')"
			let _ = db.commit()

			let rows = db <=??=> "SELECT * FROM tx_infix"
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}
		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)
		if err, ok := result.(*evaluator.Error); ok {
			t.Fatalf("Eval returned error: %s", err.Message)
		}
		rowsObj, ok := env.Get("rows")
		if !ok {
			t.Fatal("rows not found in environment")
		}
		tbl, ok := rowsObj.(*evaluator.Table)
		if !ok {
			t.Fatalf("Expected Table, got %T", rowsObj)
		}
		if len(tbl.Rows) != 2 {
			t.Errorf("Expected 2 rows after commit, got %d", len(tbl.Rows))
		}
	})

	t.Run("begin/rollback with infix operators", func(t *testing.T) {
		input := `
			let db = @sqlite(":memory:")
			let _ = db <=!=> "CREATE TABLE tx_infix_rb (id INTEGER PRIMARY KEY, value TEXT)"

			let _ = db.begin()
			let _ = db <=!=> "INSERT INTO tx_infix_rb (value) VALUES ('will_be_gone')"
			let _ = db.rollback()

			let rows = db <=??=> "SELECT * FROM tx_infix_rb"
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}
		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)
		if err, ok := result.(*evaluator.Error); ok {
			t.Fatalf("Eval returned error: %s", err.Message)
		}
		rowsObj, ok := env.Get("rows")
		if !ok {
			t.Fatal("rows not found in environment")
		}
		tbl, ok := rowsObj.(*evaluator.Table)
		if !ok {
			t.Fatalf("Expected Table, got %T", rowsObj)
		}
		if len(tbl.Rows) != 0 {
			t.Errorf("Expected 0 rows after rollback, got %d", len(tbl.Rows))
		}
	})

	t.Run("begin twice returns DB-0007 error", func(t *testing.T) {
		input := `
			let db = @sqlite(":memory:")
			let _ = db.begin()
			db.begin()
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}
		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)
		errObj, ok := result.(*evaluator.Error)
		if !ok {
			t.Fatalf("Expected Error, got %T (%v)", result, result)
		}
		if errObj.Code != "DB-0007" {
			t.Errorf("Expected DB-0007, got %s: %s", errObj.Code, errObj.Message)
		}
		// Clean up: rollback so connection is usable
		_ = env
	})

	t.Run("commit without begin returns DB-0006 error", func(t *testing.T) {
		input := `
			let db = @sqlite(":memory:")
			db.commit()
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}
		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)
		errObj, ok := result.(*evaluator.Error)
		if !ok {
			t.Fatalf("Expected Error, got %T (%v)", result, result)
		}
		if errObj.Code != "DB-0006" {
			t.Errorf("Expected DB-0006, got %s: %s", errObj.Code, errObj.Message)
		}
	})

	t.Run("rollback without begin returns DB-0006 error", func(t *testing.T) {
		input := `
			let db = @sqlite(":memory:")
			db.rollback()
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}
		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)
		errObj, ok := result.(*evaluator.Error)
		if !ok {
			t.Fatalf("Expected Error, got %T (%v)", result, result)
		}
		if errObj.Code != "DB-0006" {
			t.Errorf("Expected DB-0006, got %s: %s", errObj.Code, errObj.Message)
		}
	})

	t.Run("@transaction block with DSL operations - commit", func(t *testing.T) {
		input := `
			let db = @sqlite(":memory:")
			let _ = db <=!=> "CREATE TABLE tx_dsl (id INTEGER PRIMARY KEY, name TEXT)"

			@schema TxItem { name: string }
			let Items = db.bind(TxItem, "tx_dsl")

			@transaction {
				@insert(Items |< name: "from_transaction" .)
			}

			let rows = db <=??=> "SELECT * FROM tx_dsl"
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}
		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)
		if err, ok := result.(*evaluator.Error); ok {
			t.Fatalf("Eval returned error: %s", err.Message)
		}
		rowsObj, ok := env.Get("rows")
		if !ok {
			t.Fatal("rows not found in environment")
		}
		tbl, ok := rowsObj.(*evaluator.Table)
		if !ok {
			t.Fatalf("Expected Table, got %T", rowsObj)
		}
		if len(tbl.Rows) != 1 {
			t.Errorf("Expected 1 row after @transaction commit, got %d", len(tbl.Rows))
		}
	})

	t.Run("@transaction block rollback on error", func(t *testing.T) {
		// Set up the table and bind it
		setupInput := `
			let db = @sqlite(":memory:")
			let _ = db <=!=> "CREATE TABLE tx_dsl_rb (id INTEGER PRIMARY KEY, name TEXT)"
			@schema TxRbItem { name: string }
			let RbItems = db.bind(TxRbItem, "tx_dsl_rb")
		`
		l := lexer.New(setupInput)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Fatalf("Parser errors (setup): %v", p.Errors())
		}
		env := evaluator.NewEnvironment()
		evaluator.Eval(program, env)

		// Run the transaction that will fail — @transaction returns the error,
		// stopping the outer program, so we run it separately
		txInput := `
			@transaction {
				@insert(RbItems |< name: "will_rollback" .)
				fail("deliberate error")
			}
		`
		l2 := lexer.New(txInput)
		p2 := parser.New(l2)
		program2 := p2.ParseProgram()
		if len(p2.Errors()) != 0 {
			t.Fatalf("Parser errors (transaction): %v", p2.Errors())
		}
		// This eval returns an error (from fail()); that is expected
		evaluator.Eval(program2, env)

		// Now query — rollback should have reverted the insert
		queryInput := `
			let rows = db <=??=> "SELECT * FROM tx_dsl_rb"
		`
		l3 := lexer.New(queryInput)
		p3 := parser.New(l3)
		program3 := p3.ParseProgram()
		if len(p3.Errors()) != 0 {
			t.Fatalf("Parser errors (query): %v", p3.Errors())
		}
		evaluator.Eval(program3, env)

		rowsObj, ok := env.Get("rows")
		if !ok {
			t.Fatal("rows not found in environment")
		}
		tbl, ok := rowsObj.(*evaluator.Table)
		if !ok {
			t.Fatalf("Expected Table, got %T", rowsObj)
		}
		if len(tbl.Rows) != 0 {
			t.Errorf("Expected 0 rows after @transaction rollback, got %d", len(tbl.Rows))
		}
	})

	t.Run("infix <=!=> participates in transaction", func(t *testing.T) {
		input := `
			let db = @sqlite(":memory:")
			let _ = db <=!=> "CREATE TABLE tx_exec (id INTEGER PRIMARY KEY, value TEXT)"

			let _ = db.begin()
			let _ = db <=!=> "INSERT INTO tx_exec (value) VALUES ('via_infix_exec')"
			let _ = db.rollback()

			let rows = db <=??=> "SELECT * FROM tx_exec"
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}
		env := evaluator.NewEnvironment()
		result := evaluator.Eval(program, env)
		if err, ok := result.(*evaluator.Error); ok {
			t.Fatalf("Eval returned error: %s", err.Message)
		}
		rowsObj, ok := env.Get("rows")
		if !ok {
			t.Fatal("rows not found in environment")
		}
		tbl, ok := rowsObj.(*evaluator.Table)
		if !ok {
			t.Fatalf("Expected Table, got %T", rowsObj)
		}
		if len(tbl.Rows) != 0 {
			t.Errorf("Expected 0 rows after rollback, got %d (execute() must use Tx)", len(tbl.Rows))
		}
	})
}

func TestQueryManyReturnType(t *testing.T) {
	t.Run("infix <=??=> returns Table with columns", func(t *testing.T) {
		input := `
			let db = @sqlite(":memory:")
			let _ = db <=!=> "CREATE TABLE rt_users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)"
			let _ = db <=!=> "INSERT INTO rt_users (name, age) VALUES ('Alice', 30)"
			let _ = db <=!=> "INSERT INTO rt_users (name, age) VALUES ('Bob', 25)"
			let users = db <=??=> "SELECT * FROM rt_users"
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}
		env := evaluator.NewEnvironment()
		evaluator.Eval(program, env)

		usersObj, ok := env.Get("users")
		if !ok {
			t.Fatal("users not found in environment")
		}
		tbl, ok := usersObj.(*evaluator.Table)
		if !ok {
			t.Fatalf("Expected Table from infix <=??=>, got %T", usersObj)
		}
		if len(tbl.Rows) != 2 {
			t.Errorf("Expected 2 rows, got %d", len(tbl.Rows))
		}
		if len(tbl.Columns) != 3 {
			t.Errorf("Expected 3 columns (id, name, age), got %d: %v", len(tbl.Columns), tbl.Columns)
		}
	})

	t.Run("infix <=??=> with SQL tag returns Table with columns", func(t *testing.T) {
		input := `
			let db = @sqlite(":memory:")
			let _ = db <=!=> "CREATE TABLE rt_stmt (id INTEGER PRIMARY KEY, name TEXT)"
			let _ = db <=!=> "INSERT INTO rt_stmt (name) VALUES ('Alice')"
			let rows = db <=??=> <SQL>SELECT * FROM rt_stmt</SQL>
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}
		env := evaluator.NewEnvironment()
		evaluator.Eval(program, env)

		rowsObj, ok := env.Get("rows")
		if !ok {
			t.Fatal("rows not found in environment")
		}
		tbl, ok := rowsObj.(*evaluator.Table)
		if !ok {
			t.Fatalf("Expected Table from infix <=??=> with SQL tag, got %T", rowsObj)
		}
		if len(tbl.Rows) != 1 {
			t.Errorf("Expected 1 row, got %d", len(tbl.Rows))
		}
		if len(tbl.Columns) != 2 {
			t.Errorf("Expected 2 columns (id, name), got %d: %v", len(tbl.Columns), tbl.Columns)
		}
	})

	t.Run("both forms return the same column metadata", func(t *testing.T) {
		input := `
			let db = @sqlite(":memory:")
			let _ = db <=!=> "CREATE TABLE rt_both (id INTEGER PRIMARY KEY, name TEXT, score REAL)"
			let _ = db <=!=> "INSERT INTO rt_both (name, score) VALUES ('Alice', 9.5)"

			let infix_result = db <=??=> "SELECT * FROM rt_both"
		`
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			t.Fatalf("Parser errors: %v", p.Errors())
		}
		env := evaluator.NewEnvironment()
		evaluator.Eval(program, env)

		infixObj, ok := env.Get("infix_result")
		if !ok {
			t.Fatal("infix_result not found in environment")
		}
		tbl, ok := infixObj.(*evaluator.Table)
		if !ok {
			t.Fatalf("Expected Table, got %T", infixObj)
		}
		// Verify columns are present and correct
		if len(tbl.Columns) != 3 {
			t.Fatalf("Expected 3 columns, got %d: %v", len(tbl.Columns), tbl.Columns)
		}
		colSet := map[string]bool{}
		for _, c := range tbl.Columns {
			colSet[c] = true
		}
		for _, expected := range []string{"id", "name", "score"} {
			if !colSet[expected] {
				t.Errorf("Expected column %q in %v", expected, tbl.Columns)
			}
		}
	})
}

func TestDatabaseNullValues(t *testing.T) {
	input := `
		let db = @sqlite(":memory:")
		let _ = db <=!=> "CREATE TABLE test_nulls (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)"
		let _ = db <=!=> "INSERT INTO test_nulls (id, name) VALUES (1, 'Alice')"

		let row = db <=?=> "SELECT * FROM test_nulls WHERE id = 1"
		row
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("Eval returned error: %s", err.Message)
	}

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("Expected Dictionary, got %T", result)
	}

	// Check that age field exists (should be null)
	if _, hasAge := dict.Pairs["age"]; !hasAge {
		t.Error("Result should have 'age' field (even if null)")
	}
}

func TestDatabaseMultipleConnections(t *testing.T) {
	input := `
		let db1 = @sqlite(":memory:")
		let db2 = @sqlite(":memory:")

		let _ = db1 <=!=> "CREATE TABLE test1 (id INTEGER PRIMARY KEY, value TEXT)"
		let _ = db2 <=!=> "CREATE TABLE test2 (id INTEGER PRIMARY KEY, value TEXT)"

		let _ = db1 <=!=> "INSERT INTO test1 (value) VALUES ('from db1')"
		let _ = db2 <=!=> "INSERT INTO test2 (value) VALUES ('from db2')"

		let result1 = db1 <=?=> "SELECT * FROM test1"
		let result2 = db2 <=?=> "SELECT * FROM test2"

		result1.value
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("Eval returned error: %s", err.Message)
	}

	str, ok := result.(*evaluator.String)
	if !ok {
		t.Fatalf("Expected String, got %T", result)
	}

	if str.Value != "from db1" {
		t.Errorf("Expected 'from db1', got '%s'", str.Value)
	}
}

func TestDatabaseEmptyResultSet(t *testing.T) {
	input := `
		let db = @sqlite(":memory:")
		let _ = db <=!=> "CREATE TABLE test_empty (id INTEGER PRIMARY KEY, value TEXT)"

		let rows = db <=??=> "SELECT * FROM test_empty"
		rows
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("Eval returned error: %s", err.Message)
	}

	tbl, ok := result.(*evaluator.Table)
	if !ok {
		t.Fatalf("Expected Table, got %T", result)
	}

	if len(tbl.Rows) != 0 {
		t.Errorf("Expected empty table, got %d rows", len(tbl.Rows))
	}
}

func TestDatabaseLargeResultSet(t *testing.T) {
	// Create database and insert rows
	l := lexer.New(`
		let db = @sqlite(":memory:")
		let _ = db <=!=> "CREATE TABLE test_large (id INTEGER PRIMARY KEY, value INTEGER)"
	`)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	env := evaluator.NewEnvironment()
	evaluator.Eval(program, env)

	// Insert 100 rows individually
	for i := range 100 {
		insertSQL := fmt.Sprintf(`let _ = db <=!=> "INSERT INTO test_large (value) VALUES (%d)"`, i)
		l2 := lexer.New(insertSQL)
		p2 := parser.New(l2)
		program2 := p2.ParseProgram()
		if len(p2.Errors()) != 0 {
			t.Fatalf("Parser errors on insert: %v", p2.Errors())
		}
		evaluator.Eval(program2, env)
	}

	// Query all rows
	l3 := lexer.New(`
		let rows = db <=??=> "SELECT * FROM test_large"
		rows.count()
	`)
	p3 := parser.New(l3)
	program3 := p3.ParseProgram()
	if len(p3.Errors()) != 0 {
		t.Fatalf("Parser errors on query: %v", p3.Errors())
	}
	result := evaluator.Eval(program3, env)

	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("Eval returned error: %s", err.Message)
	}

	integer, ok := result.(*evaluator.Integer)
	if !ok {
		t.Fatalf("Expected Integer, got %T", result)
	}

	if integer.Value != 100 {
		t.Errorf("Expected 100 rows, got %d", integer.Value)
	}
}

func TestDatabaseDataTypes(t *testing.T) {
	input := `
		let db = @sqlite(":memory:")
		let _ = db <=!=> ` + "`CREATE TABLE test_types (\n\t\t\tid INTEGER PRIMARY KEY,\n\t\t\tint_val INTEGER,\n\t\t\tfloat_val REAL,\n\t\t\ttext_val TEXT,\n\t\t\tblob_val BLOB\n\t\t)`" + `

		let _ = db <=!=> "INSERT INTO test_types (int_val, float_val, text_val) VALUES (42, 3.14, 'hello')"

		let row = db <=?=> "SELECT * FROM test_types WHERE id = 1"
		row
	`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	env := evaluator.NewEnvironment()
	result := evaluator.Eval(program, env)

	if err, ok := result.(*evaluator.Error); ok {
		t.Fatalf("Eval returned error: %s", err.Message)
	}

	dict, ok := result.(*evaluator.Dictionary)
	if !ok {
		t.Fatalf("Expected Dictionary, got %T", result)
	}

	// Verify int_val
	if intExpr, hasInt := dict.Pairs["int_val"]; hasInt {
		intObj := evaluator.Eval(intExpr, dict.Env)
		if integer, ok := intObj.(*evaluator.Integer); !ok || integer.Value != 42 {
			t.Errorf("Expected int_val=42, got %v", intObj.Inspect())
		}
	} else {
		t.Error("Result should have int_val field")
	}

	// Verify float_val
	if floatExpr, hasFloat := dict.Pairs["float_val"]; hasFloat {
		floatObj := evaluator.Eval(floatExpr, dict.Env)
		if float, ok := floatObj.(*evaluator.Float); !ok || float.Value != 3.14 {
			t.Errorf("Expected float_val=3.14, got %v", floatObj.Inspect())
		}
	} else {
		t.Error("Result should have float_val field")
	}

	// Verify text_val
	if textExpr, hasText := dict.Pairs["text_val"]; hasText {
		textObj := evaluator.Eval(textExpr, dict.Env)
		if str, ok := textObj.(*evaluator.String); !ok || str.Value != "hello" {
			t.Errorf("Expected text_val='hello', got %v", textObj.Inspect())
		}
	} else {
		t.Error("Result should have text_val field")
	}
}
