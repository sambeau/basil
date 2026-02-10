package evaluator

// Database driver imports for side-effect registration with database/sql.
// These drivers are required for @postgres() and @mysql() to function.

import (
	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/lib/pq"              // PostgreSQL driver
)
