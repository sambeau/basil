package evaluator

import (
	"fmt"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/ast"
)

// evalQueryExpression evaluates a @query(...) expression
func evalQueryExpression(node *ast.QueryExpression, env *Environment) Object {
	// 1. Resolve the source binding from the environment
	sourceObj, ok := env.Get(node.Source.Value)
	if !ok {
		return &Error{
			Message: fmt.Sprintf("undefined binding: %s", node.Source.Value),
			Class:   ClassUndefined,
			Code:    "REF-0001",
		}
	}

	binding, ok := sourceObj.(*TableBinding)
	if !ok {
		return &Error{
			Message: fmt.Sprintf("@query source must be a table binding, got %s", sourceObj.Type()),
			Class:   ClassType,
			Code:    "TYPE-0001",
		}
	}

	// 2. Build SQL from conditions and modifiers
	sql, params, err := buildSelectSQL(node, binding, env)
	if err != nil {
		return err
	}

	// 3. Execute the query against the database
	if binding.DB == nil {
		return &Error{
			Message: "table binding has no database connection",
			Class:   ClassDatabase,
			Code:    "DB-0001",
		}
	}

	// 4. Execute and transform results based on terminal type
	terminal := node.Terminal
	if terminal == nil {
		return &Error{
			Message: "@query requires a terminal (?->, ??->, or .)",
			Class:   ClassParse,
			Code:    "SYN-0001",
		}
	}

	// Extract "with" relations for eager loading
	var withRelations []*ast.RelationPath
	for _, mod := range node.Modifiers {
		if mod.Kind == "with" {
			if len(mod.RelationPaths) > 0 {
				withRelations = append(withRelations, mod.RelationPaths...)
			} else {
				// Backward compatibility: convert Fields to RelationPaths
				for _, field := range mod.Fields {
					withRelations = append(withRelations, &ast.RelationPath{Path: field})
				}
			}
		}
	}

	var result Object
	switch terminal.Type {
	case "many":
		result = executeQueryMany(binding, sql, params, terminal.Projection, env)
	case "one":
		// Check for special projections: count, exists, toSQL
		if len(terminal.Projection) == 1 {
			switch terminal.Projection[0] {
			case "count":
				return executeQueryCount(binding, sql, params, env)
			case "exists":
				return executeQueryExists(binding, sql, params, env)
			case "toSQL":
				return executeQueryToSQL(binding, sql, params, env)
			}
		}
		result = executeQueryOne(binding, sql, params, terminal.Projection, env)
	case "execute":
		return executeQueryExecute(binding, sql, params, env)
	case "count":
		return executeQueryAffectedCount(binding, sql, params, env)
	default:
		return &Error{
			Message: fmt.Sprintf("unknown query terminal type: %s", terminal.Type),
			Class:   ClassParse,
			Code:    "SYN-0002",
		}
	}

	// If there are relations to eager load and result is not an error
	if len(withRelations) > 0 && !isError(result) && result != NULL {
		result = loadRelations(result, binding, withRelations, env)
	}

	return result
}

// buildSelectSQL builds a SELECT statement from a QueryExpression
func buildSelectSQL(node *ast.QueryExpression, binding *TableBinding, env *Environment) (string, []Object, *Error) {
	var sql strings.Builder
	var params []Object
	paramIdx := 1

	// Build CTE names map for resolving CTE references in conditions
	// We build this incrementally as we process CTEs, so earlier CTEs can be referenced by later ones
	cteNames := make(map[string]*ast.QueryCTE)

	// Build WITH clause from CTEs
	if len(node.CTEs) > 0 {
		sql.WriteString("WITH ")
		for i, cte := range node.CTEs {
			if i > 0 {
				sql.WriteString(", ")
			}
			// Pass the CTEs defined so far, so later CTEs can reference earlier ones
			cteSql, cteParams, err := buildCTESQL(cte, env, &paramIdx, cteNames)
			if err != nil {
				return "", nil, err
			}
			sql.WriteString(cte.Name)
			sql.WriteString(" AS (")
			sql.WriteString(cteSql)
			sql.WriteString(")")
			params = append(params, cteParams...)

			// Add this CTE to the map for subsequent CTEs and the main query
			cteNames[cte.Name] = cte
		}
		sql.WriteString(" ")
	}

	// Check if this is an aggregation query (has GROUP BY or computed fields)
	hasGroupBy := len(node.GroupBy) > 0
	hasComputedFields := len(node.ComputedFields) > 0
	isAggregation := hasGroupBy || hasComputedFields

	// Check if any computed fields are correlated subqueries or join subqueries
	hasCorrelatedSubquery := false
	hasJoinSubquery := false
	var joinSubqueries []*ast.QueryComputedField
	for _, cf := range node.ComputedFields {
		if cf.Subquery != nil {
			if cf.IsJoinSubquery {
				hasJoinSubquery = true
				joinSubqueries = append(joinSubqueries, cf)
			} else {
				hasCorrelatedSubquery = true
			}
		}
	}

	// Determine columns to select
	var selectCols []string

	// Determine the outer table alias (for join subqueries)
	outerTableAlias := binding.TableName
	if node.SourceAlias != nil {
		outerTableAlias = node.SourceAlias.Value
	}

	if hasJoinSubquery {
		// For join subqueries, we need to select from outer table and joined tables
		// SELECT outer.*, joined_alias.* FROM outer JOIN ... AS joined_alias ON ...
		if node.Terminal != nil && len(node.Terminal.Projection) > 0 &&
			!(len(node.Terminal.Projection) == 1 && node.Terminal.Projection[0] == "*") {
			// Specific columns requested - qualify them with outer table alias
			for _, col := range node.Terminal.Projection {
				selectCols = append(selectCols, fmt.Sprintf("%s.%s", outerTableAlias, col))
			}
		} else {
			selectCols = []string{fmt.Sprintf("%s.*", outerTableAlias)}
		}

		// Add columns from each join subquery
		for _, cf := range joinSubqueries {
			if cf.Subquery != nil && cf.Subquery.Terminal != nil {
				proj := cf.Subquery.Terminal.Projection
				if len(proj) == 1 && proj[0] == "*" {
					selectCols = append(selectCols, fmt.Sprintf("%s.*", cf.Name))
				} else {
					for _, col := range proj {
						selectCols = append(selectCols, fmt.Sprintf("%s.%s", cf.Name, col))
					}
				}
			}
		}

		// Also add non-join correlated subquery fields as scalar selects
		for _, cf := range node.ComputedFields {
			if cf.Subquery != nil && !cf.IsJoinSubquery {
				cfSQL, cfParams, err := buildComputedFieldSQL(cf, binding.TableName, env, &paramIdx)
				if err != nil {
					return "", nil, err
				}
				selectCols = append(selectCols, cfSQL)
				params = append(params, cfParams...)
			}
		}
	} else if isAggregation && !hasCorrelatedSubquery {
		// For aggregation queries, build SELECT from GROUP BY fields and computed fields
		// First add GROUP BY fields
		selectCols = append(selectCols, node.GroupBy...)

		// Then add computed fields (simple aggregates, no correlated subqueries)
		for _, cf := range node.ComputedFields {
			cfSQL, cfParams, err := buildComputedFieldSQL(cf, binding.TableName, env, &paramIdx)
			if err != nil {
				return "", nil, err
			}
			selectCols = append(selectCols, cfSQL)
			params = append(params, cfParams...)
		}

		// If terminal specifies projection, filter to only those columns
		if node.Terminal != nil && len(node.Terminal.Projection) > 0 {
			if !(len(node.Terminal.Projection) == 1 && node.Terminal.Projection[0] == "*") {
				// User specified specific columns - validate they exist in our select
				// For now, trust the user knows what they're doing
				// The database will error if columns don't exist
			}
		}
	} else if hasCorrelatedSubquery {
		// Correlated subquery computed fields: SELECT *, (SELECT ...) AS alias, ...
		// Start with base columns
		if node.Terminal != nil && len(node.Terminal.Projection) > 0 &&
			!(len(node.Terminal.Projection) == 1 && node.Terminal.Projection[0] == "*") {
			selectCols = node.Terminal.Projection
		} else {
			selectCols = []string{"*"}
		}

		// Add correlated subquery computed fields
		for _, cf := range node.ComputedFields {
			cfSQL, cfParams, err := buildComputedFieldSQL(cf, binding.TableName, env, &paramIdx)
			if err != nil {
				return "", nil, err
			}
			selectCols = append(selectCols, cfSQL)
			params = append(params, cfParams...)
		}
	} else if node.Terminal != nil && len(node.Terminal.Projection) > 0 {
		// Check for special projections
		if len(node.Terminal.Projection) == 1 {
			switch node.Terminal.Projection[0] {
			case "count":
				selectCols = []string{"COUNT(*) as count"}
			case "exists":
				selectCols = []string{"1"}
			case "toSQL":
				// For toSQL, generate a normal SELECT * query
				// The actual toSQL behavior is handled in evalQueryExpression
				selectCols = []string{"*"}
			default:
				selectCols = node.Terminal.Projection
			}
		} else {
			selectCols = node.Terminal.Projection
		}
	} else {
		selectCols = []string{"*"}
	}

	// Build SELECT clause
	sql.WriteString("SELECT ")
	sql.WriteString(strings.Join(selectCols, ", "))
	sql.WriteString(" FROM ")
	sql.WriteString(binding.TableName)

	// Add table alias for join subqueries
	if hasJoinSubquery && node.SourceAlias != nil {
		sql.WriteString(" ")
		sql.WriteString(node.SourceAlias.Value)
	}

	// Build JOIN clauses for join subqueries
	for _, cf := range joinSubqueries {
		joinSQL, joinParams, err := buildJoinSubquerySQL(cf, outerTableAlias, env, &paramIdx)
		if err != nil {
			return "", nil, err
		}
		sql.WriteString(joinSQL)
		params = append(params, joinParams...)
	}

	// Build WHERE clause from conditions (these are pre-aggregation conditions)
	var whereClauses []string

	// Add soft delete filter if configured
	if binding.SoftDeleteColumn != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", binding.SoftDeleteColumn))
	}

	// Add user conditions (only non-computed field conditions for WHERE)
	// Conditions on computed fields become HAVING clauses (or WHERE for correlated subqueries)
	var havingClauses []string
	computedFieldNames := make(map[string]bool)
	correlatedFieldDefs := make(map[string]*ast.QueryComputedField)
	for _, cf := range node.ComputedFields {
		computedFieldNames[cf.Name] = true
		if cf.Subquery != nil {
			correlatedFieldDefs[cf.Name] = cf
		}
	}

	for _, cond := range node.Conditions {
		// Check if this condition is on a computed field
		leftName := getConditionLeft(cond)

		if correlatedFieldDefs[leftName] != nil {
			// This is a condition on a correlated subquery field
			// Generate WHERE clause with inline subquery
			cf := correlatedFieldDefs[leftName]
			clause, condParams, err := buildCorrelatedConditionWhereClause(cond, cf, binding.TableName, env, &paramIdx)
			if err != nil {
				return "", nil, err
			}
			params = append(params, condParams...)
			whereClauses = append(whereClauses, clause)
		} else if computedFieldNames[leftName] {
			// This is a HAVING condition (condition on non-correlated computed field)
			clause, condParams, _, _, err := buildConditionNodeSQLWithCTEs(cond, env, &paramIdx, cteNames)
			if err != nil {
				return "", nil, err
			}
			params = append(params, condParams...)
			havingClauses = append(havingClauses, clause)
		} else {
			// This is a WHERE condition
			clause, condParams, logic, _, err := buildConditionNodeSQLWithCTEs(cond, env, &paramIdx, cteNames)
			if err != nil {
				return "", nil, err
			}
			params = append(params, condParams...)

			if len(whereClauses) == 0 || logic == "" {
				whereClauses = append(whereClauses, clause)
			} else {
				// Combine with previous using AND/OR
				logicStr := strings.ToUpper(logic)
				if logicStr != "AND" && logicStr != "OR" {
					logicStr = "AND"
				}
				lastIdx := len(whereClauses) - 1
				whereClauses[lastIdx] = fmt.Sprintf("(%s %s %s)", whereClauses[lastIdx], logicStr, clause)
			}
		}
	}

	if len(whereClauses) > 0 {
		sql.WriteString(" WHERE ")
		sql.WriteString(strings.Join(whereClauses, " AND "))
	}

	// Build GROUP BY clause
	if hasGroupBy {
		sql.WriteString(" GROUP BY ")
		sql.WriteString(strings.Join(node.GroupBy, ", "))
	}

	// Build HAVING clause for conditions on computed fields
	if len(havingClauses) > 0 {
		sql.WriteString(" HAVING ")
		sql.WriteString(strings.Join(havingClauses, " AND "))
	}

	// Build ORDER BY, LIMIT, OFFSET from modifiers
	var orderBy []string
	var limit, offset int64
	hasLimit := false

	for _, mod := range node.Modifiers {
		switch mod.Kind {
		case "order":
			for _, of := range mod.OrderFields {
				orderSpec := of.Field
				if of.Direction != "" {
					orderSpec += " " + strings.ToUpper(of.Direction)
				}
				orderBy = append(orderBy, orderSpec)
			}
		case "limit":
			limit = mod.Value
			hasLimit = true
		case "offset":
			offset = mod.Value
		}
	}

	if len(orderBy) > 0 {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(strings.Join(orderBy, ", "))
	}

	if hasLimit {
		sql.WriteString(fmt.Sprintf(" LIMIT %d", limit))
	}

	if offset > 0 {
		sql.WriteString(fmt.Sprintf(" OFFSET %d", offset))
	}

	// For "one" terminal without explicit limit, add LIMIT 1
	if node.Terminal != nil && node.Terminal.Type == "one" && !hasLimit {
		// Don't add LIMIT 1 for count/exists
		if len(node.Terminal.Projection) != 1 ||
			(node.Terminal.Projection[0] != "count" && node.Terminal.Projection[0] != "exists") {
			sql.WriteString(" LIMIT 1")
		}
	}

	// For exists, add LIMIT 1
	if node.Terminal != nil && len(node.Terminal.Projection) == 1 && node.Terminal.Projection[0] == "exists" {
		sql.WriteString(" LIMIT 1")
	}

	return sql.String(), params, nil
}

// buildCTESQL builds a SELECT statement for a Common Table Expression
// cteNames contains CTEs defined before this one, for inter-CTE references
func buildCTESQL(cte *ast.QueryCTE, env *Environment, paramIdx *int, cteNames map[string]*ast.QueryCTE) (string, []Object, *Error) {
	var sql strings.Builder
	var params []Object

	// Get the source table name - resolve from environment if it's a binding
	tableName := cte.Source.Value
	sourceObj, ok := env.Get(tableName)
	if ok {
		if binding, isBinding := sourceObj.(*TableBinding); isBinding {
			tableName = binding.TableName
		}
	}

	// Determine columns to select
	var selectCols []string
	if cte.Terminal != nil && len(cte.Terminal.Projection) > 0 {
		selectCols = cte.Terminal.Projection
	} else {
		selectCols = []string{"*"}
	}

	// Build SELECT clause
	sql.WriteString("SELECT ")
	sql.WriteString(strings.Join(selectCols, ", "))
	sql.WriteString(" FROM ")
	sql.WriteString(tableName)

	// Build WHERE clause from conditions (use CTE-aware version)
	var whereClauses []string
	for _, cond := range cte.Conditions {
		clause, condParams, logic, _, err := buildConditionNodeSQLWithCTEs(cond, env, paramIdx, cteNames)
		if err != nil {
			return "", nil, err
		}
		params = append(params, condParams...)

		if len(whereClauses) == 0 || logic == "" {
			whereClauses = append(whereClauses, clause)
		} else {
			// Combine with previous using AND/OR
			logicStr := strings.ToUpper(logic)
			if logicStr != "AND" && logicStr != "OR" {
				logicStr = "AND"
			}
			lastIdx := len(whereClauses) - 1
			whereClauses[lastIdx] = fmt.Sprintf("(%s %s %s)", whereClauses[lastIdx], logicStr, clause)
		}
	}

	if len(whereClauses) > 0 {
		sql.WriteString(" WHERE ")
		sql.WriteString(strings.Join(whereClauses, " AND "))
	}

	// Build ORDER BY, LIMIT from modifiers
	var orderBy []string
	var limit int64
	hasLimit := false

	for _, mod := range cte.Modifiers {
		switch mod.Kind {
		case "order":
			for _, of := range mod.OrderFields {
				orderSpec := of.Field
				if of.Direction != "" {
					orderSpec += " " + strings.ToUpper(of.Direction)
				}
				orderBy = append(orderBy, orderSpec)
			}
		case "limit":
			limit = mod.Value
			hasLimit = true
		}
	}

	if len(orderBy) > 0 {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(strings.Join(orderBy, ", "))
	}

	if hasLimit {
		sql.WriteString(fmt.Sprintf(" LIMIT %d", limit))
	}

	return sql.String(), params, nil
}

// buildComputedFieldSQL converts a QueryComputedField to SQL SELECT expression
// outerTableName is used for correlated subqueries to qualify column references
func buildComputedFieldSQL(cf *ast.QueryComputedField, outerTableName string, env *Environment, paramIdx *int) (string, []Object, *Error) {
	var expr string
	var params []Object

	// Check for correlated subquery
	if cf.Subquery != nil {
		subExpr, subParams, err := buildCorrelatedSubquerySQL(cf.Subquery, outerTableName, env, paramIdx)
		if err != nil {
			return "", nil, err
		}
		params = append(params, subParams...)
		// Return as "(SUBQUERY) as alias"
		return fmt.Sprintf("(%s) as %s", subExpr, cf.Name), params, nil
	}

	// Simple computed field (aggregate or field reference)
	switch cf.Function {
	case "count":
		if cf.Field != "" {
			expr = fmt.Sprintf("COUNT(%s)", cf.Field)
		} else {
			expr = "COUNT(*)"
		}
	case "sum":
		expr = fmt.Sprintf("SUM(%s)", cf.Field)
	case "avg":
		expr = fmt.Sprintf("AVG(%s)", cf.Field)
	case "min":
		expr = fmt.Sprintf("MIN(%s)", cf.Field)
	case "max":
		expr = fmt.Sprintf("MAX(%s)", cf.Field)
	default:
		// Just a field reference (no aggregation)
		expr = cf.Field
	}

	// Return as "EXPR as alias"
	return fmt.Sprintf("%s as %s", expr, cf.Name), nil, nil
}

// buildCorrelatedSubquerySQL builds a correlated subquery that references the outer query
// Example: SELECT COUNT(*) FROM comments WHERE post_id = posts.id
func buildCorrelatedSubquerySQL(subquery *ast.QuerySubquery, outerTableName string, env *Environment, paramIdx *int) (string, []Object, *Error) {
	var sql strings.Builder
	var params []Object

	// Get the subquery table name
	tableName := subquery.Source.Value

	// Determine SELECT clause based on terminal
	selectExpr := "*"
	if subquery.Terminal != nil && len(subquery.Terminal.Projection) > 0 {
		proj := subquery.Terminal.Projection[0]
		switch proj {
		case "count":
			selectExpr = "COUNT(*)"
		case "sum", "avg", "min", "max":
			// If we have a field like sum(amount), we'd need to parse it
			// For now, treat as-is
			selectExpr = strings.ToUpper(proj) + "(*)"
		default:
			selectExpr = proj
		}
	}

	sql.WriteString("SELECT ")
	sql.WriteString(selectExpr)
	sql.WriteString(" FROM ")
	sql.WriteString(tableName)

	// Build WHERE clause from conditions
	// Conditions can reference outer table columns with table.column syntax
	if len(subquery.Conditions) > 0 {
		sql.WriteString(" WHERE ")
		for i, cond := range subquery.Conditions {
			if i > 0 {
				// Get logic from the condition
				logic := "AND"
				if qc, ok := cond.(*ast.QueryCondition); ok && qc.Logic != "" {
					logic = strings.ToUpper(qc.Logic)
				}
				sql.WriteString(" " + logic + " ")
			}
			clause, condParams, err := buildCorrelatedConditionSQL(cond, outerTableName, env, paramIdx)
			if err != nil {
				return "", nil, err
			}
			sql.WriteString(clause)
			params = append(params, condParams...)
		}
	}

	// Build ORDER BY, LIMIT from modifiers
	for _, mod := range subquery.Modifiers {
		switch mod.Kind {
		case "order":
			var orderParts []string
			for _, of := range mod.OrderFields {
				orderSpec := of.Field
				if of.Direction != "" {
					orderSpec += " " + strings.ToUpper(of.Direction)
				}
				orderParts = append(orderParts, orderSpec)
			}
			if len(orderParts) > 0 {
				sql.WriteString(" ORDER BY ")
				sql.WriteString(strings.Join(orderParts, ", "))
			}
		case "limit":
			sql.WriteString(fmt.Sprintf(" LIMIT %d", mod.Value))
		case "offset":
			sql.WriteString(fmt.Sprintf(" OFFSET %d", mod.Value))
		}
	}

	return sql.String(), params, nil
}

// buildJoinSubquerySQL builds a JOIN clause for a join-like subquery (??-> terminal)
// Example: JOIN order_items items ON items.order_id = orders.id
// This produces row multiplication - each outer row expands to multiple rows based on the joined table
func buildJoinSubquerySQL(cf *ast.QueryComputedField, outerTableAlias string, env *Environment, paramIdx *int) (string, []Object, *Error) {
	var sql strings.Builder
	var params []Object

	if cf.Subquery == nil {
		return "", nil, &Error{Message: "join subquery requires a subquery definition"}
	}

	subquery := cf.Subquery
	joinedTableName := subquery.Source.Value
	joinAlias := cf.Name // The computed field name becomes the join alias

	// Build JOIN clause
	// We use INNER JOIN to match the LATERAL JOIN semantics (only include rows that have matches)
	// Use LEFT JOIN if we want to include outer rows without matches
	sql.WriteString(" JOIN ")
	sql.WriteString(joinedTableName)
	sql.WriteString(" ")
	sql.WriteString(joinAlias)

	// Build ON clause from subquery conditions
	if len(subquery.Conditions) > 0 {
		sql.WriteString(" ON ")
		for i, cond := range subquery.Conditions {
			if i > 0 {
				// Get logic from the condition
				logic := "AND"
				if qc, ok := cond.(*ast.QueryCondition); ok && qc.Logic != "" {
					logic = strings.ToUpper(qc.Logic)
				}
				sql.WriteString(" " + logic + " ")
			}
			clause, condParams, err := buildJoinConditionSQL(cond, outerTableAlias, joinAlias, env, paramIdx)
			if err != nil {
				return "", nil, err
			}
			sql.WriteString(clause)
			params = append(params, condParams...)
		}
	} else {
		// No conditions - cross join (all combinations)
		sql.WriteString(" ON 1=1")
	}

	return sql.String(), params, nil
}

// buildJoinConditionSQL builds a condition for a JOIN ON clause
// It translates outer.field and inner.field references appropriately
func buildJoinConditionSQL(node ast.QueryConditionNode, outerTableAlias string, joinAlias string, env *Environment, paramIdx *int) (string, []Object, *Error) {
	switch cond := node.(type) {
	case *ast.QueryCondition:
		return buildJoinCondition(cond, outerTableAlias, joinAlias, env, paramIdx)
	case *ast.QueryConditionGroup:
		// Handle condition groups
		var parts []string
		var allParams []Object
		for i, child := range cond.Conditions {
			part, partParams, err := buildJoinConditionSQL(child, outerTableAlias, joinAlias, env, paramIdx)
			if err != nil {
				return "", nil, err
			}
			if i > 0 {
				// Get logic from child
				logic := "AND"
				if qc, ok := child.(*ast.QueryCondition); ok && qc.Logic != "" {
					logic = strings.ToUpper(qc.Logic)
				}
				parts = append(parts, logic+" "+part)
			} else {
				parts = append(parts, part)
			}
			allParams = append(allParams, partParams...)
		}
		result := "(" + strings.Join(parts, " ") + ")"
		if cond.Negated {
			result = "NOT " + result
		}
		return result, allParams, nil
	default:
		return "", nil, &Error{Message: "unknown condition node type in join subquery"}
	}
}

// buildJoinCondition builds a single condition for a JOIN ON clause
// Example: order_id == o.id becomes items.order_id = o.id
func buildJoinCondition(cond *ast.QueryCondition, outerTableAlias string, joinAlias string, env *Environment, paramIdx *int) (string, []Object, *Error) {
	var params []Object

	// Get left side - bare identifier is from the joined table
	leftStr := ""
	if ident, ok := cond.Left.(*ast.Identifier); ok {
		// Bare identifier belongs to the joined table
		leftStr = joinAlias + "." + ident.Value
	} else if dotExpr, ok := cond.Left.(*ast.DotExpression); ok {
		// Dot expression like "outer.id" - use as-is
		if objIdent, ok := dotExpr.Left.(*ast.Identifier); ok {
			leftStr = objIdent.Value + "." + dotExpr.Key
		}
	}

	// Map operator
	sqlOp := cond.Operator
	switch cond.Operator {
	case "==":
		sqlOp = "="
	case "!=":
		sqlOp = "<>"
	}

	// Get right side - check if it's an outer table reference
	rightStr := ""
	switch right := cond.Right.(type) {
	case *ast.Identifier:
		// Simple identifier - could be from join table (unusual in ON clause)
		rightStr = right.Value
	case *ast.DotExpression:
		// Dot expression like "o.id" - references outer table
		if objIdent, ok := right.Left.(*ast.Identifier); ok {
			// The prefix identifies which table the column is from
			rightStr = objIdent.Value + "." + right.Key
		} else {
			// Complex property access - try to evaluate
			val := Eval(right, env)
			if isError(val) {
				return "", nil, val.(*Error)
			}
			placeholder := fmt.Sprintf("$%d", *paramIdx)
			*paramIdx++
			rightStr = placeholder
			params = append(params, val)
		}
	case *ast.IntegerLiteral:
		placeholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		rightStr = placeholder
		params = append(params, &Integer{Value: right.Value})
	case *ast.StringLiteral:
		placeholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		rightStr = placeholder
		params = append(params, &String{Value: right.Value})
	case *ast.Boolean:
		placeholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		rightStr = placeholder
		params = append(params, &Boolean{Value: right.Value})
	default:
		// Try to evaluate as expression
		val := Eval(right, env)
		if isError(val) {
			return "", nil, val.(*Error)
		}
		placeholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		rightStr = placeholder
		params = append(params, val)
	}

	result := fmt.Sprintf("%s %s %s", leftStr, sqlOp, rightStr)
	if cond.Negated {
		result = "NOT " + result
	}
	return result, params, nil
}

// buildCorrelatedConditionSQL builds a condition that may reference outer query columns
// It handles column references like "post.id" which should resolve to the outer table
func buildCorrelatedConditionSQL(node ast.QueryConditionNode, outerTableName string, env *Environment, paramIdx *int) (string, []Object, *Error) {
	switch cond := node.(type) {
	case *ast.QueryCondition:
		return buildCorrelatedCondition(cond, outerTableName, env, paramIdx)
	case *ast.QueryConditionGroup:
		// Handle condition groups
		var parts []string
		var allParams []Object
		for i, child := range cond.Conditions {
			part, partParams, err := buildCorrelatedConditionSQL(child, outerTableName, env, paramIdx)
			if err != nil {
				return "", nil, err
			}
			if i > 0 {
				// Get logic from child
				logic := "AND"
				if qc, ok := child.(*ast.QueryCondition); ok && qc.Logic != "" {
					logic = strings.ToUpper(qc.Logic)
				}
				parts = append(parts, logic+" "+part)
			} else {
				parts = append(parts, part)
			}
			allParams = append(allParams, partParams...)
		}
		result := "(" + strings.Join(parts, " ") + ")"
		if cond.Negated {
			result = "NOT " + result
		}
		return result, allParams, nil
	default:
		return "", nil, &Error{Message: "unknown condition node type in correlated subquery"}
	}
}

// buildCorrelatedCondition builds a single condition with outer table reference support
func buildCorrelatedCondition(cond *ast.QueryCondition, outerTableName string, env *Environment, paramIdx *int) (string, []Object, *Error) {
	var params []Object

	// Get left side - if it's "table.column" and table matches outer, don't parameterize
	leftStr := ""
	if ident, ok := cond.Left.(*ast.Identifier); ok {
		leftStr = ident.Value
	} else if dotExpr, ok := cond.Left.(*ast.DotExpression); ok {
		// This is a dot expression like "post.id"
		if objIdent, ok := dotExpr.Left.(*ast.Identifier); ok {
			leftStr = objIdent.Value + "." + dotExpr.Key
		}
	}

	// Map operator
	sqlOp := cond.Operator
	switch cond.Operator {
	case "==":
		sqlOp = "="
	case "!=":
		sqlOp = "<>"
	}

	// Get right side - check if it's an outer table reference
	rightStr := ""
	switch right := cond.Right.(type) {
	case *ast.Identifier:
		// Simple identifier - treat as column in subquery table
		rightStr = right.Value
	case *ast.DotExpression:
		// Dot expression like "post.id" - qualify with outer table
		if _, ok := right.Left.(*ast.Identifier); ok {
			// Check if this references the outer table alias
			// The outer table reference should use the table name, not the alias
			rightStr = outerTableName + "." + right.Key
		} else {
			// Complex property access - try to evaluate
			val := Eval(right, env)
			if isError(val) {
				return "", nil, val.(*Error)
			}
			placeholder := fmt.Sprintf("$%d", *paramIdx)
			*paramIdx++
			rightStr = placeholder
			params = append(params, val)
		}
	case *ast.IntegerLiteral:
		placeholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		rightStr = placeholder
		params = append(params, &Integer{Value: right.Value})
	case *ast.StringLiteral:
		placeholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		rightStr = placeholder
		params = append(params, &String{Value: right.Value})
	case *ast.Boolean:
		placeholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		rightStr = placeholder
		params = append(params, &Boolean{Value: right.Value})
	default:
		// Try to evaluate as expression
		val := Eval(right, env)
		if isError(val) {
			return "", nil, val.(*Error)
		}
		placeholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		rightStr = placeholder
		params = append(params, val)
	}

	result := fmt.Sprintf("%s %s %s", leftStr, sqlOp, rightStr)
	if cond.Negated {
		result = "NOT " + result
	}
	return result, params, nil
}

// buildCorrelatedConditionWhereClause builds a WHERE clause condition for a correlated subquery field
// Example: (SELECT COUNT(*) FROM comments WHERE post_id = posts.id) > 5
func buildCorrelatedConditionWhereClause(cond ast.QueryConditionNode, cf *ast.QueryComputedField, outerTableName string, env *Environment, paramIdx *int) (string, []Object, *Error) {
	qc, ok := cond.(*ast.QueryCondition)
	if !ok {
		return "", nil, &Error{Message: "correlated subquery conditions must be simple conditions"}
	}

	// Build the subquery SQL
	subSQL, subParams, err := buildCorrelatedSubquerySQL(cf.Subquery, outerTableName, env, paramIdx)
	if err != nil {
		return "", nil, err
	}

	// Map operator
	sqlOp := qc.Operator
	switch qc.Operator {
	case "==":
		sqlOp = "="
	case "!=":
		sqlOp = "<>"
	}

	// Get right side value
	var rightStr string
	var params []Object
	params = append(params, subParams...)

	switch right := qc.Right.(type) {
	case *ast.IntegerLiteral:
		placeholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		rightStr = placeholder
		params = append(params, &Integer{Value: right.Value})
	case *ast.StringLiteral:
		placeholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		rightStr = placeholder
		params = append(params, &String{Value: right.Value})
	case *ast.Boolean:
		placeholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		rightStr = placeholder
		params = append(params, &Boolean{Value: right.Value})
	default:
		// Try to evaluate as expression
		val := Eval(right, env)
		if isError(val) {
			return "", nil, val.(*Error)
		}
		placeholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		rightStr = placeholder
		params = append(params, val)
	}

	result := fmt.Sprintf("(%s) %s %s", subSQL, sqlOp, rightStr)
	if qc.Negated {
		result = "NOT " + result
	}
	return result, params, nil
}

// buildConditionNodeSQL converts a QueryConditionNode (either QueryCondition or QueryConditionGroup) to SQL
// Returns the SQL clause, parameters, logic operator (and/or), and negated flag
func buildConditionNodeSQL(node ast.QueryConditionNode, env *Environment, paramIdx *int) (string, []Object, string, bool, *Error) {
	switch cond := node.(type) {
	case *ast.QueryCondition:
		clause, params, err := buildConditionSQL(cond, env, paramIdx)
		if err != nil {
			return "", nil, "", false, err
		}
		// Handle negation
		if cond.Negated {
			clause = "NOT " + clause
		}
		return clause, params, cond.Logic, cond.Negated, nil
	case *ast.QueryConditionGroup:
		clause, params, err := buildConditionGroupSQL(cond, env, paramIdx)
		if err != nil {
			return "", nil, "", false, err
		}
		return clause, params, cond.Logic, cond.Negated, nil
	default:
		return "", nil, "", false, &Error{
			Message: fmt.Sprintf("unknown condition node type: %T", node),
			Class:   ClassParse,
			Code:    "SYN-0099",
		}
	}
}

// buildConditionGroupSQL converts a QueryConditionGroup to SQL (a parenthesized group of conditions)
func buildConditionGroupSQL(group *ast.QueryConditionGroup, env *Environment, paramIdx *int) (string, []Object, *Error) {
	var params []Object
	var clauses []string

	for i, node := range group.Conditions {
		clause, condParams, logic, _, err := buildConditionNodeSQL(node, env, paramIdx)
		if err != nil {
			return "", nil, err
		}
		params = append(params, condParams...)

		if i == 0 {
			clauses = append(clauses, clause)
		} else {
			// Use logic from the condition/group to combine with previous
			logicStr := "AND"
			if logic != "" {
				logicStr = strings.ToUpper(logic)
			}
			clauses = append(clauses, fmt.Sprintf("%s %s", logicStr, clause))
		}
	}

	// Wrap in parentheses and optionally negate
	result := "(" + strings.Join(clauses, " ") + ")"
	if group.Negated {
		result = "NOT " + result
	}
	return result, params, nil
}

// buildConditionNodeSQLWithCTEs is like buildConditionNodeSQL but handles CTE references
func buildConditionNodeSQLWithCTEs(node ast.QueryConditionNode, env *Environment, paramIdx *int, cteNames map[string]*ast.QueryCTE) (string, []Object, string, bool, *Error) {
	switch cond := node.(type) {
	case *ast.QueryCondition:
		clause, params, err := buildConditionSQLWithCTEs(cond, env, paramIdx, cteNames)
		if err != nil {
			return "", nil, "", false, err
		}
		// Handle negation
		if cond.Negated {
			clause = "NOT " + clause
		}
		return clause, params, cond.Logic, cond.Negated, nil
	case *ast.QueryConditionGroup:
		clause, params, err := buildConditionGroupSQLWithCTEs(cond, env, paramIdx, cteNames)
		if err != nil {
			return "", nil, "", false, err
		}
		return clause, params, cond.Logic, cond.Negated, nil
	default:
		return "", nil, "", false, &Error{
			Message: fmt.Sprintf("unknown condition node type: %T", node),
			Class:   ClassParse,
			Code:    "SYN-0099",
		}
	}
}

// buildConditionGroupSQLWithCTEs is like buildConditionGroupSQL but handles CTE references
func buildConditionGroupSQLWithCTEs(group *ast.QueryConditionGroup, env *Environment, paramIdx *int, cteNames map[string]*ast.QueryCTE) (string, []Object, *Error) {
	var params []Object
	var clauses []string

	for i, node := range group.Conditions {
		clause, condParams, logic, _, err := buildConditionNodeSQLWithCTEs(node, env, paramIdx, cteNames)
		if err != nil {
			return "", nil, err
		}
		params = append(params, condParams...)

		if i == 0 {
			clauses = append(clauses, clause)
		} else {
			// Use logic from the condition/group to combine with previous
			logicStr := "AND"
			if logic != "" {
				logicStr = strings.ToUpper(logic)
			}
			clauses = append(clauses, fmt.Sprintf("%s %s", logicStr, clause))
		}
	}

	// Wrap in parentheses and optionally negate
	result := "(" + strings.Join(clauses, " ") + ")"
	if group.Negated {
		result = "NOT " + result
	}
	return result, params, nil
}

// buildConditionSQLWithCTEs is like buildConditionSQL but handles CTE references
// When the right side is an identifier that matches a CTE name, it generates a subquery reference
func buildConditionSQLWithCTEs(cond *ast.QueryCondition, env *Environment, paramIdx *int, cteNames map[string]*ast.QueryCTE) (string, []Object, *Error) {
	var params []Object

	// Get the column name from the left side
	leftStr := ""
	if ident, ok := cond.Left.(*ast.Identifier); ok {
		leftStr = ident.Value
	} else {
		leftStr = cond.Left.String()
	}

	// Handle different operators
	switch cond.Operator {
	case "is null":
		return fmt.Sprintf("%s IS NULL", leftStr), nil, nil
	case "is not null":
		return fmt.Sprintf("%s IS NOT NULL", leftStr), nil, nil
	case "between":
		// Handle "between X and Y"
		if cond.Right == nil || cond.RightEnd == nil {
			return "", nil, &Error{
				Message: "between operator requires two values",
				Class:   ClassParse,
				Code:    "SYN-0003",
			}
		}
		startVal, startErr := evalConditionValue(cond.Right, env)
		if startErr != nil {
			return "", nil, startErr
		}
		endVal, endErr := evalConditionValue(cond.RightEnd, env)
		if endErr != nil {
			return "", nil, endErr
		}
		startPlaceholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		endPlaceholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		params = append(params, startVal, endVal)
		return fmt.Sprintf("%s BETWEEN %s AND %s", leftStr, startPlaceholder, endPlaceholder), params, nil
	}

	// Check for subquery on the right side
	if subquery, ok := cond.Right.(*ast.QuerySubquery); ok {
		return buildSubqueryCondition(leftStr, cond.Operator, subquery, env, paramIdx)
	}

	// Check for CTE reference on the right side (for "in" or "not in" operators)
	// CTE references can come from:
	// 1. QueryCTERef (explicit CTE reference from parser)
	// 2. QueryColumnRef where the column name matches a CTE name
	// 3. Legacy Identifier (for backward compatibility during transition)
	cteName := ""
	if cteRef, ok := cond.Right.(*ast.QueryCTERef); ok {
		cteName = cteRef.Name
	} else if colRef, ok := cond.Right.(*ast.QueryColumnRef); ok {
		// Check if this column ref is actually a CTE name
		if _, exists := cteNames[colRef.Column]; exists {
			cteName = colRef.Column
		}
	} else if ident, ok := cond.Right.(*ast.Identifier); ok {
		// Legacy path - bare identifier might be a CTE reference
		if _, exists := cteNames[ident.Value]; exists {
			cteName = ident.Value
		}
	}

	if cteName != "" {
		if cte, exists := cteNames[cteName]; exists {
			// This is a CTE reference - generate "column IN (SELECT * FROM cte_name)"
			// Determine what column to select from the CTE based on its terminal
			selectCol := "*"
			if cte.Terminal != nil && len(cte.Terminal.Projection) > 0 {
				// Use the first projected column
				selectCol = cte.Terminal.Projection[0]
			}

			sqlOp := "IN"
			if cond.Operator == "not in" {
				sqlOp = "NOT IN"
			}

			return fmt.Sprintf("%s %s (SELECT %s FROM %s)", leftStr, sqlOp, selectCol, cte.Name), nil, nil
		}
	}

	// Check if right side is a column reference (column-to-column comparison)
	if colRef, ok := cond.Right.(*ast.QueryColumnRef); ok {
		sqlOp := conditionOperatorToSQL(cond.Operator)
		return fmt.Sprintf("%s %s %s", leftStr, sqlOp, colRef.Column), nil, nil
	}

	// Handle the right side value
	if cond.Right == nil {
		return "", nil, &Error{
			Message: fmt.Sprintf("condition '%s %s' requires a value", leftStr, cond.Operator),
			Class:   ClassParse,
			Code:    "SYN-0003",
		}
	}

	// Evaluate the right side (interpolation, literal, etc.)
	rightVal, evalErr := evalConditionValue(cond.Right, env)
	if evalErr != nil {
		return "", nil, evalErr
	}

	// Convert operator to SQL
	sqlOp := ""
	switch cond.Operator {
	case "==":
		sqlOp = "="
	case "!=":
		sqlOp = "<>"
	case ">", "<", ">=", "<=":
		sqlOp = cond.Operator
	case "in":
		return buildInClause(leftStr, rightVal, paramIdx)
	case "not in":
		clause, inParams, err := buildInClause(leftStr, rightVal, paramIdx)
		if err != nil {
			return "", nil, err
		}
		// Convert "IN" to "NOT IN"
		clause = strings.Replace(clause, " IN (", " NOT IN (", 1)
		return clause, inParams, nil
	case "like":
		sqlOp = "LIKE"
	default:
		return "", nil, &Error{
			Message: fmt.Sprintf("unknown operator: %s", cond.Operator),
			Class:   ClassParse,
			Code:    "SYN-0004",
		}
	}

	placeholder := fmt.Sprintf("$%d", *paramIdx)
	*paramIdx++
	params = append(params, rightVal)

	return fmt.Sprintf("%s %s %s", leftStr, sqlOp, placeholder), params, nil
}

// getConditionLeft extracts the left identifier name from a condition node (for computed field check)
func getConditionLeft(node ast.QueryConditionNode) string {
	if cond, ok := node.(*ast.QueryCondition); ok {
		if ident, ok := cond.Left.(*ast.Identifier); ok {
			return ident.Value
		}
	}
	// Groups don't have a single left identifier
	return ""
}

// getConditionLogic extracts the logic operator from a condition node
func getConditionLogic(node ast.QueryConditionNode) string {
	switch cond := node.(type) {
	case *ast.QueryCondition:
		return cond.Logic
	case *ast.QueryConditionGroup:
		return cond.Logic
	}
	return ""
}

// buildConditionSQL converts a QueryCondition to SQL
func buildConditionSQL(cond *ast.QueryCondition, env *Environment, paramIdx *int) (string, []Object, *Error) {
	var params []Object

	// Get the column name from the left side
	leftStr := ""
	if ident, ok := cond.Left.(*ast.Identifier); ok {
		leftStr = ident.Value
	} else {
		leftStr = cond.Left.String()
	}

	// Handle different operators
	switch cond.Operator {
	case "is null":
		return fmt.Sprintf("%s IS NULL", leftStr), nil, nil
	case "is not null":
		return fmt.Sprintf("%s IS NOT NULL", leftStr), nil, nil
	case "between":
		// Handle "between X and Y"
		if cond.Right == nil || cond.RightEnd == nil {
			return "", nil, &Error{
				Message: "between operator requires two values",
				Class:   ClassParse,
				Code:    "SYN-0003",
			}
		}
		startVal, startErr := evalConditionValue(cond.Right, env)
		if startErr != nil {
			return "", nil, startErr
		}
		endVal, endErr := evalConditionValue(cond.RightEnd, env)
		if endErr != nil {
			return "", nil, endErr
		}
		startPlaceholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		endPlaceholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		params = append(params, startVal, endVal)
		return fmt.Sprintf("%s BETWEEN %s AND %s", leftStr, startPlaceholder, endPlaceholder), params, nil
	}

	// Check for subquery on the right side
	if subquery, ok := cond.Right.(*ast.QuerySubquery); ok {
		return buildSubqueryCondition(leftStr, cond.Operator, subquery, env, paramIdx)
	}

	// Handle the right side value
	if cond.Right == nil {
		return "", nil, &Error{
			Message: fmt.Sprintf("condition '%s %s' requires a value", leftStr, cond.Operator),
			Class:   ClassParse,
			Code:    "SYN-0003",
		}
	}

	// Check if right side is a column reference (bare identifier = column-to-column comparison)
	if colRef, ok := cond.Right.(*ast.QueryColumnRef); ok {
		// Column-to-column comparison: price > cost
		sqlOp := conditionOperatorToSQL(cond.Operator)
		return fmt.Sprintf("%s %s %s", leftStr, sqlOp, colRef.Column), nil, nil
	}

	// Evaluate the right side (interpolation, literal, etc.)
	rightVal, evalErr := evalConditionValue(cond.Right, env)
	if evalErr != nil {
		return "", nil, evalErr
	}

	// Convert operator to SQL
	sqlOp := ""
	switch cond.Operator {
	case "==":
		sqlOp = "="
	case "!=":
		sqlOp = "<>"
	case ">", "<", ">=", "<=":
		sqlOp = cond.Operator
	case "in":
		return buildInClause(leftStr, rightVal, paramIdx)
	case "not in":
		clause, inParams, err := buildInClause(leftStr, rightVal, paramIdx)
		if err != nil {
			return "", nil, err
		}
		// Convert "IN" to "NOT IN"
		clause = strings.Replace(clause, " IN (", " NOT IN (", 1)
		return clause, inParams, nil
	case "like":
		sqlOp = "LIKE"
	default:
		return "", nil, &Error{
			Message: fmt.Sprintf("unknown operator: %s", cond.Operator),
			Class:   ClassParse,
			Code:    "SYN-0004",
		}
	}

	placeholder := fmt.Sprintf("$%d", *paramIdx)
	*paramIdx++
	params = append(params, rightVal)

	return fmt.Sprintf("%s %s %s", leftStr, sqlOp, placeholder), params, nil
}

// conditionOperatorToSQL converts a condition operator to SQL
func conditionOperatorToSQL(op string) string {
	switch op {
	case "==":
		return "="
	case "!=":
		return "<>"
	case "like":
		return "LIKE"
	default:
		return op
	}
}

// evalConditionValue evaluates a condition value expression
// - QueryInterpolation: evaluate the contained Parsley expression
// - QueryColumnRef: error (should not be evaluated, used as SQL column)
// - Literals: convert to Object
// - Other expressions: evaluate as Parsley expression
func evalConditionValue(expr ast.Expression, env *Environment) (Object, *Error) {
	switch v := expr.(type) {
	case *ast.QueryInterpolation:
		// Evaluate the interpolated expression
		result := Eval(v.Expression, env)
		if isError(result) {
			return nil, result.(*Error)
		}
		return result, nil
	case *ast.QueryColumnRef:
		// Column references should not be evaluated - they're used directly in SQL
		return nil, &Error{
			Message: fmt.Sprintf("column reference '%s' cannot be used as a value; did you mean {%s}?", v.Column, v.Column),
			Class:   ClassParse,
			Code:    "SYN-0005",
		}
	case *ast.StringLiteral:
		return &String{Value: v.Value}, nil
	case *ast.IntegerLiteral:
		return &Integer{Value: v.Value}, nil
	case *ast.FloatLiteral:
		return &Float{Value: v.Value}, nil
	case *ast.Boolean:
		return &Boolean{Value: v.Value}, nil
	case *ast.ArrayLiteral:
		// Evaluate array elements
		elements := make([]Object, len(v.Elements))
		for i, el := range v.Elements {
			val := Eval(el, env)
			if isError(val) {
				return nil, val.(*Error)
			}
			elements[i] = val
		}
		return &Array{Elements: elements}, nil
	default:
		// Fall back to general evaluation for other expression types
		result := Eval(expr, env)
		if isError(result) {
			return nil, result.(*Error)
		}
		return result, nil
	}
}

// buildSubqueryCondition builds a subquery condition (e.g., author_id IN (SELECT id FROM users WHERE role = 'admin'))
func buildSubqueryCondition(column string, operator string, subquery *ast.QuerySubquery, env *Environment, paramIdx *int) (string, []Object, *Error) {
	var params []Object

	// Get the table name from the subquery source
	tableName := subquery.Source.Value

	// Determine SELECT column from terminal projection
	selectColumn := "*"
	if subquery.Terminal != nil && len(subquery.Terminal.Projection) > 0 {
		selectColumn = subquery.Terminal.Projection[0]
	}

	// Build the subquery SQL
	subSQL := fmt.Sprintf("SELECT %s FROM %s", selectColumn, tableName)

	// Build WHERE clause from conditions
	if len(subquery.Conditions) > 0 {
		var whereClauses []string
		for i, cond := range subquery.Conditions {
			clause, condParams, logic, _, err := buildConditionNodeSQL(cond, env, paramIdx)
			if err != nil {
				return "", nil, err
			}
			if i == 0 {
				whereClauses = append(whereClauses, clause)
			} else {
				logicStr := "AND"
				if logic != "" {
					logicStr = strings.ToUpper(logic)
				}
				whereClauses = append(whereClauses, fmt.Sprintf("%s %s", logicStr, clause))
			}
			params = append(params, condParams...)
		}
		subSQL += " WHERE " + strings.Join(whereClauses, " ")
	}

	// Build ORDER BY, LIMIT from modifiers
	for _, mod := range subquery.Modifiers {
		switch mod.Kind {
		case "order":
			var orderParts []string
			for _, of := range mod.OrderFields {
				orderSpec := of.Field
				if of.Direction != "" {
					orderSpec += " " + strings.ToUpper(of.Direction)
				}
				orderParts = append(orderParts, orderSpec)
			}
			if len(orderParts) > 0 {
				subSQL += " ORDER BY " + strings.Join(orderParts, ", ")
			}
		case "limit":
			subSQL += fmt.Sprintf(" LIMIT %d", mod.Value)
		case "offset":
			subSQL += fmt.Sprintf(" OFFSET %d", mod.Value)
		}
	}

	// Build the full condition
	sqlOp := "IN"
	if operator == "not in" {
		sqlOp = "NOT IN"
	}

	return fmt.Sprintf("%s %s (%s)", column, sqlOp, subSQL), params, nil
}

// buildInClause builds an IN clause for arrays
func buildInClause(column string, value Object, paramIdx *int) (string, []Object, *Error) {
	arr, ok := value.(*Array)
	if !ok {
		// Single value - treat as array of one
		placeholder := fmt.Sprintf("$%d", *paramIdx)
		*paramIdx++
		return fmt.Sprintf("%s IN (%s)", column, placeholder), []Object{value}, nil
	}

	if len(arr.Elements) == 0 {
		// Empty array - always false
		return "1 = 0", nil, nil
	}

	var placeholders []string
	var params []Object
	for _, elem := range arr.Elements {
		placeholders = append(placeholders, fmt.Sprintf("$%d", *paramIdx))
		*paramIdx++
		params = append(params, elem)
	}

	return fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ", ")), params, nil
}

// executeQueryMany executes a query and returns an array of dictionaries
func executeQueryMany(binding *TableBinding, sql string, params []Object, projection []string, env *Environment) Object {
	rows, err := binding.query(sql, params)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, colErr := rows.Rows.Columns()
	if colErr != nil {
		return &Error{
			Message: fmt.Sprintf("error getting columns: %s", colErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0004",
		}
	}

	results := []Object{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}

		if scanErr := rows.Rows.Scan(ptrs...); scanErr != nil {
			return &Error{
				Message: fmt.Sprintf("error scanning row: %s", scanErr.Error()),
				Class:   ClassDatabase,
				Code:    "DB-0005",
			}
		}

		results = append(results, rowToDict(columns, values, env))
	}

	if rows.Err() != nil {
		return &Error{
			Message: fmt.Sprintf("error iterating rows: %s", rows.Err().Error()),
			Class:   ClassDatabase,
			Code:    "DB-0002",
		}
	}

	return &Array{Elements: results}
}

// executeQueryOne executes a query and returns a single dictionary or null
func executeQueryOne(binding *TableBinding, sql string, params []Object, projection []string, env *Environment) Object {
	rows, err := binding.query(sql, params)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, colErr := rows.Rows.Columns()
	if colErr != nil {
		return &Error{
			Message: fmt.Sprintf("error getting columns: %s", colErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0004",
		}
	}

	if !rows.Next() {
		if rows.Err() != nil {
			return &Error{
				Message: fmt.Sprintf("error reading row: %s", rows.Err().Error()),
				Class:   ClassDatabase,
				Code:    "DB-0002",
			}
		}
		return NULL
	}

	values := make([]interface{}, len(columns))
	ptrs := make([]interface{}, len(columns))
	for i := range values {
		ptrs[i] = &values[i]
	}

	if scanErr := rows.Rows.Scan(ptrs...); scanErr != nil {
		return &Error{
			Message: fmt.Sprintf("error scanning row: %s", scanErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0005",
		}
	}

	return rowToDict(columns, values, env)
}

// executeQueryCount executes a COUNT query and returns an integer
func executeQueryCount(binding *TableBinding, sql string, params []Object, env *Environment) Object {
	rows, err := binding.query(sql, params)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return &Integer{Value: 0}
	}

	var count int64
	if scanErr := rows.Rows.Scan(&count); scanErr != nil {
		return &Error{
			Message: fmt.Sprintf("error scanning count: %s", scanErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0003",
		}
	}

	return &Integer{Value: count}
}

// executeQueryExists executes an EXISTS-style query and returns a boolean
func executeQueryExists(binding *TableBinding, sql string, params []Object, env *Environment) Object {
	rows, err := binding.query(sql, params)
	if err != nil {
		return err
	}
	defer rows.Close()

	exists := rows.Next()
	return &Boolean{Value: exists}
}

// executeQueryToSQL returns the generated SQL and parameters without executing the query
func executeQueryToSQL(binding *TableBinding, sql string, params []Object, env *Environment) Object {
	// Build params array
	paramsArray := &Array{Elements: make([]Object, len(params))}
	copy(paramsArray.Elements, params)

	// Return dictionary with sql and params
	result := &Dictionary{
		Pairs: make(map[string]ast.Expression),
		Env:   env,
	}
	result.SetKey("sql", objectToExpression(&String{Value: sql}))
	result.SetKey("params", objectToExpression(paramsArray))
	return result
}

// executeQueryExecute executes a query without returning results
func executeQueryExecute(binding *TableBinding, sql string, params []Object, env *Environment) Object {
	_, err := binding.exec(sql, params)
	if err != nil {
		return err
	}
	return NULL
}

// executeQueryAffectedCount executes a query and returns the number of affected rows
func executeQueryAffectedCount(binding *TableBinding, sql string, params []Object, env *Environment) Object {
	result, err := binding.exec(sql, params)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	return &Integer{Value: count}
}

// loadRelations performs eager loading of related records
func loadRelations(result Object, binding *TableBinding, relations []*ast.RelationPath, env *Environment) Object {
	// Make sure we have a DSL schema with relation info
	if binding.DSLSchema == nil {
		// No schema, can't do eager loading
		return result
	}

	// Handle single result (dictionary)
	if dict, ok := result.(*Dictionary); ok {
		return loadRelationsForRecord(dict, binding, relations, env)
	}

	// Handle array of results
	if arr, ok := result.(*Array); ok {
		if len(arr.Elements) == 0 {
			return arr
		}
		// Load relations for each record
		for i, elem := range arr.Elements {
			if dict, ok := elem.(*Dictionary); ok {
				arr.Elements[i] = loadRelationsForRecord(dict, binding, relations, env)
				if isError(arr.Elements[i]) {
					return arr.Elements[i]
				}
			}
		}
		return arr
	}

	return result
}

// loadRelationsForRecord loads relations for a single record
func loadRelationsForRecord(record *Dictionary, binding *TableBinding, relations []*ast.RelationPath, env *Environment) Object {
	schema := binding.DSLSchema

	for _, relationPathObj := range relations {
		// Handle nested relations like "comments.author"
		parts := strings.Split(relationPathObj.Path, ".")
		firstRelation := parts[0]
		nestedPath := parts[1:] // remaining path segments

		// Find the relation in the schema
		relation, exists := schema.Relations[firstRelation]
		if !exists {
			return &Error{
				Message: fmt.Sprintf("unknown relation '%s' in schema %s", firstRelation, schema.Name),
				Class:   ClassUndefined,
				Code:    "REF-0002",
			}
		}

		// We need to find the binding for the related table
		// The target schema name tells us what schema to look for
		// We need to find a TableBinding that uses this schema

		// Get the foreign key value from the record for belongs-to relations
		// or the primary key for has-many relations
		if relation.IsMany {
			// Has-many: look up related table by foreign key pointing to this record's ID
			// Get this record's ID
			idExpr, hasID := record.Pairs["id"]
			if !hasID {
				// Skip if no ID
				record.Pairs[firstRelation] = &ast.ObjectLiteralExpression{Obj: &Array{Elements: []Object{}}}
				record.KeyOrder = append(record.KeyOrder, firstRelation)
				continue
			}

			// Evaluate the ID expression to get the actual value
			idObj := Eval(idExpr, env)
			if isError(idObj) {
				return idObj.(*Error)
			}

			// Find related records where foreign_key = this.id
			// Pass filter conditions, order, and limit from the RelationPath
			relatedRecords, err := loadHasManyRelation(binding, relation, idObj, relationPathObj.Conditions, relationPathObj.Order, relationPathObj.Limit, env)
			if err != nil {
				return err
			}

			// If there are nested relations, load them on the related records
			if len(nestedPath) > 0 {
				relatedRecords = loadNestedRelations(relatedRecords, relation, nestedPath, binding, env)
				if isError(relatedRecords) {
					return relatedRecords
				}
			}

			record.Pairs[firstRelation] = &ast.ObjectLiteralExpression{Obj: relatedRecords}
			record.KeyOrder = append(record.KeyOrder, firstRelation)
		} else {
			// Belongs-to: look up related record by foreign key on this record
			fkExpr, hasFK := record.Pairs[relation.ForeignKey]
			if !hasFK || fkExpr == nil {
				// Skip if no foreign key value
				record.Pairs[firstRelation] = &ast.ObjectLiteralExpression{Obj: NULL}
				record.KeyOrder = append(record.KeyOrder, firstRelation)
				continue
			}

			// Evaluate the foreign key expression
			fkObj := Eval(fkExpr, env)
			if isError(fkObj) {
				return fkObj.(*Error)
			}
			if fkObj == NULL {
				record.Pairs[firstRelation] = &ast.ObjectLiteralExpression{Obj: NULL}
				record.KeyOrder = append(record.KeyOrder, firstRelation)
				continue
			}

			// Find the related record
			relatedRecord, err := loadBelongsToRelation(binding, relation, fkObj, env)
			if err != nil {
				return err
			}

			// If there are nested relations, load them on the related record
			if len(nestedPath) > 0 && relatedRecord != NULL {
				relatedRecord = loadNestedRelations(relatedRecord, relation, nestedPath, binding, env)
				if isError(relatedRecord) {
					return relatedRecord
				}
			}

			record.Pairs[firstRelation] = &ast.ObjectLiteralExpression{Obj: relatedRecord}
			record.KeyOrder = append(record.KeyOrder, firstRelation)
		}
	}

	return record
}

// loadNestedRelations loads nested relations on already-loaded related records
// e.g., for "comments.author", after loading comments, this loads author on each comment
func loadNestedRelations(records Object, parentRelation *DSLSchemaRelation, nestedPath []string, parentBinding *TableBinding, env *Environment) Object {
	// Find the binding for the parent relation's target schema
	relatedBinding := findBindingForSchema(parentBinding.DB, parentRelation.TargetSchema, env)
	if relatedBinding == nil || relatedBinding.DSLSchema == nil {
		// No binding or schema found, can't load nested relations
		return records
	}

	// Build the nested relation path (e.g., "author" or "author.profile")
	nestedRelationPath := &ast.RelationPath{Path: strings.Join(nestedPath, ".")}

	// Handle array of records
	if arr, ok := records.(*Array); ok {
		for i, elem := range arr.Elements {
			if dict, ok := elem.(*Dictionary); ok {
				result := loadRelationsForRecord(dict, relatedBinding, []*ast.RelationPath{nestedRelationPath}, env)
				if isError(result) {
					return result
				}
				arr.Elements[i] = result
			}
		}
		return arr
	}

	// Handle single record
	if dict, ok := records.(*Dictionary); ok {
		return loadRelationsForRecord(dict, relatedBinding, []*ast.RelationPath{nestedRelationPath}, env)
	}

	return records
}

// loadHasManyRelation loads a has-many relation (e.g., author has many posts)
// conditions are optional filters to apply to the related records
// orderFields are optional ordering fields
// limit is an optional limit on the number of related records
func loadHasManyRelation(parentBinding *TableBinding, relation *DSLSchemaRelation, parentID Object, conditions []ast.QueryConditionNode, orderFields []ast.QueryOrderField, limit *int64, env *Environment) (Object, *Error) {
	// Find a binding for the related schema
	// This is a simplified approach - we look for a table binding with matching schema name
	relatedBinding := findBindingForSchema(parentBinding.DB, relation.TargetSchema, env)
	if relatedBinding == nil {
		// No binding found, return empty array
		return &Array{Elements: []Object{}}, nil
	}

	// Build query: SELECT * FROM related_table WHERE foreign_key = parent_id
	sql := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", relatedBinding.TableName, relation.ForeignKey)
	params := []Object{parentID}
	paramIndex := 2

	// Add soft delete filter if configured
	if relatedBinding.SoftDeleteColumn != "" {
		sql += fmt.Sprintf(" AND %s IS NULL", relatedBinding.SoftDeleteColumn)
	}

	// Add filter conditions
	for _, cond := range conditions {
		condSQL, condParams, _, _, err := buildConditionNodeSQL(cond, env, &paramIndex)
		if err != nil {
			return nil, err
		}
		sql += " AND " + condSQL
		params = append(params, condParams...)
	}

	// Add ORDER BY if specified
	if len(orderFields) > 0 {
		var orderParts []string
		for _, of := range orderFields {
			orderStr := of.Field
			if of.Direction != "" {
				orderStr += " " + strings.ToUpper(of.Direction)
			}
			orderParts = append(orderParts, orderStr)
		}
		sql += " ORDER BY " + strings.Join(orderParts, ", ")
	}

	// Add LIMIT if specified
	if limit != nil {
		sql += fmt.Sprintf(" LIMIT %d", *limit)
	}

	rows, err := relatedBinding.query(sql, params)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, colErr := rows.Rows.Columns()
	if colErr != nil {
		return nil, &Error{
			Message: fmt.Sprintf("error getting columns: %s", colErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0004",
		}
	}

	results := []Object{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}

		if scanErr := rows.Rows.Scan(ptrs...); scanErr != nil {
			return nil, &Error{
				Message: fmt.Sprintf("error scanning row: %s", scanErr.Error()),
				Class:   ClassDatabase,
				Code:    "DB-0005",
			}
		}

		results = append(results, rowToDict(columns, values, env))
	}

	return &Array{Elements: results}, nil
}

// loadBelongsToRelation loads a belongs-to relation (e.g., post belongs to author)
func loadBelongsToRelation(parentBinding *TableBinding, relation *DSLSchemaRelation, foreignKeyValue Object, env *Environment) (Object, *Error) {
	// Find a binding for the related schema
	relatedBinding := findBindingForSchema(parentBinding.DB, relation.TargetSchema, env)
	if relatedBinding == nil {
		return NULL, nil
	}

	// Build query: SELECT * FROM related_table WHERE id = foreign_key_value LIMIT 1
	sql := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", relatedBinding.TableName)

	// Add soft delete filter if configured
	if relatedBinding.SoftDeleteColumn != "" {
		sql += fmt.Sprintf(" AND %s IS NULL", relatedBinding.SoftDeleteColumn)
	}
	sql += " LIMIT 1"

	rows, err := relatedBinding.query(sql, []Object{foreignKeyValue})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, colErr := rows.Rows.Columns()
	if colErr != nil {
		return nil, &Error{
			Message: fmt.Sprintf("error getting columns: %s", colErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0004",
		}
	}

	if !rows.Next() {
		return NULL, nil
	}

	values := make([]interface{}, len(columns))
	ptrs := make([]interface{}, len(columns))
	for i := range values {
		ptrs[i] = &values[i]
	}

	if scanErr := rows.Rows.Scan(ptrs...); scanErr != nil {
		return nil, &Error{
			Message: fmt.Sprintf("error scanning row: %s", scanErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0005",
		}
	}

	return rowToDict(columns, values, env), nil
}

// findBindingForSchema finds a TableBinding that uses the given schema name
// It searches the environment for bindings with matching DSLSchema
func findBindingForSchema(db *DBConnection, schemaName string, env *Environment) *TableBinding {
	// Search the environment for a binding with matching schema
	// This is a simplified approach - in a real implementation, you might want to
	// maintain a registry of bindings by schema name
	for varName, obj := range env.store {
		if binding, ok := obj.(*TableBinding); ok {
			if binding.DB == db && binding.DSLSchema != nil && binding.DSLSchema.Name == schemaName {
				_ = varName // Used for debugging
				return binding
			}
		}
	}

	// Also check the outer environment
	if env.outer != nil {
		return findBindingForSchema(db, schemaName, env.outer)
	}

	return nil
}

// ============================================================================
// INSERT Expression
// ============================================================================

// evalInsertExpression evaluates a @insert(...) expression
func evalInsertExpression(node *ast.InsertExpression, env *Environment) Object {
	// 1. Resolve the source binding
	sourceObj, ok := env.Get(node.Source.Value)
	if !ok {
		return &Error{
			Message: fmt.Sprintf("undefined binding: %s", node.Source.Value),
			Class:   ClassUndefined,
			Code:    "REF-0001",
		}
	}

	binding, ok := sourceObj.(*TableBinding)
	if !ok {
		return &Error{
			Message: fmt.Sprintf("@insert source must be a table binding, got %s", sourceObj.Type()),
			Class:   ClassType,
			Code:    "TYPE-0001",
		}
	}

	// 2. Check for batch insert
	if node.Batch != nil {
		return evalBatchInsert(node, binding, env)
	}

	// 3. Validate field values against schema (if DSL schema is bound)
	if binding.DSLSchema != nil {
		values := make(map[string]Object)
		for _, write := range node.Writes {
			val := Eval(write.Value, env)
			if isError(val) {
				return val
			}
			values[write.Field] = val
		}
		if validationErr := ValidateSchemaFields(values, binding.DSLSchema); validationErr != nil {
			return validationErr
		}
	}

	// 4. Build INSERT SQL
	sql, params, err := buildInsertSQL(node, binding, env)
	if err != nil {
		return err
	}

	// 5. Execute based on terminal
	if node.Terminal == nil {
		return &Error{
			Message: "@insert requires a terminal (., ?->, or ??->)",
			Class:   ClassParse,
			Code:    "SYN-0001",
		}
	}

	switch node.Terminal.Type {
	case "execute":
		return executeInsert(binding, sql, params, env)
	case "one":
		return executeInsertReturning(binding, sql, params, node.Terminal.Projection, env)
	case "count":
		return executeInsertCount(binding, sql, params, env)
	default:
		return &Error{
			Message: fmt.Sprintf("invalid terminal type for @insert: %s", node.Terminal.Type),
			Class:   ClassParse,
			Code:    "SYN-0002",
		}
	}
}

// buildInsertSQL builds an INSERT statement from an InsertExpression
func buildInsertSQL(node *ast.InsertExpression, binding *TableBinding, env *Environment) (string, []Object, *Error) {
	var sql strings.Builder
	var params []Object
	paramIdx := 1

	// Collect columns and values
	var columns []string
	var placeholders []string

	for _, write := range node.Writes {
		columns = append(columns, write.Field)

		// Evaluate the value
		val := Eval(write.Value, env)
		if isError(val) {
			return "", nil, val.(*Error)
		}

		placeholders = append(placeholders, fmt.Sprintf("$%d", paramIdx))
		paramIdx++
		params = append(params, val)
	}

	if len(columns) == 0 {
		return "", nil, &Error{
			Message: "@insert requires at least one field to set",
			Class:   ClassParse,
			Code:    "SYN-0003",
		}
	}

	sql.WriteString("INSERT INTO ")
	sql.WriteString(binding.TableName)
	sql.WriteString(" (")
	sql.WriteString(strings.Join(columns, ", "))
	sql.WriteString(") VALUES (")
	sql.WriteString(strings.Join(placeholders, ", "))
	sql.WriteString(")")

	// Handle upsert (ON CONFLICT)
	if len(node.UpsertKey) > 0 {
		sql.WriteString(" ON CONFLICT (")
		sql.WriteString(strings.Join(node.UpsertKey, ", "))
		sql.WriteString(") DO UPDATE SET ")

		var updates []string
		for _, col := range columns {
			updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
		}
		sql.WriteString(strings.Join(updates, ", "))
	}

	// Add RETURNING clause if terminal requests data
	if node.Terminal != nil && (node.Terminal.Type == "one" || node.Terminal.Type == "many") {
		sql.WriteString(" RETURNING ")
		if len(node.Terminal.Projection) == 0 || node.Terminal.Projection[0] == "*" {
			sql.WriteString("*")
		} else {
			sql.WriteString(strings.Join(node.Terminal.Projection, ", "))
		}
	}

	return sql.String(), params, nil
}

// executeInsert executes an INSERT without returning data
func executeInsert(binding *TableBinding, sql string, params []Object, env *Environment) Object {
	_, err := binding.exec(sql, params)
	if err != nil {
		return err
	}
	return NULL
}

// executeInsertReturning executes an INSERT and returns the created row
func executeInsertReturning(binding *TableBinding, sql string, params []Object, projection []string, env *Environment) Object {
	rows, err := binding.query(sql, params)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, colErr := rows.Rows.Columns()
	if colErr != nil {
		return &Error{
			Message: fmt.Sprintf("error getting columns: %s", colErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0004",
		}
	}

	if !rows.Next() {
		// No row returned - shouldn't happen with RETURNING
		return NULL
	}

	values := make([]interface{}, len(columns))
	ptrs := make([]interface{}, len(columns))
	for i := range values {
		ptrs[i] = &values[i]
	}

	if scanErr := rows.Rows.Scan(ptrs...); scanErr != nil {
		return &Error{
			Message: fmt.Sprintf("error scanning row: %s", scanErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0005",
		}
	}

	return rowToDict(columns, values, env)
}

// executeInsertCount executes an INSERT and returns count of affected rows
func executeInsertCount(binding *TableBinding, sql string, params []Object, env *Environment) Object {
	result, err := binding.exec(sql, params)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	return &Integer{Value: count}
}

// evalBatchInsert handles batch inserts: @insert(Table * each {items} -> item |< ... .)
func evalBatchInsert(node *ast.InsertExpression, binding *TableBinding, env *Environment) Object {
	// Evaluate the collection
	collection := Eval(node.Batch.Collection, env)
	if isError(collection) {
		return collection
	}

	arr, ok := collection.(*Array)
	if !ok {
		return &Error{
			Message: fmt.Sprintf("batch insert collection must be an array, got %s", collection.Type()),
			Class:   ClassType,
			Code:    "TYPE-0002",
		}
	}

	if len(arr.Elements) == 0 {
		// Nothing to insert
		if node.Terminal != nil && node.Terminal.Type == "count" {
			return &Integer{Value: 0}
		}
		return NULL
	}

	// For batch inserts, we'll execute each insert in sequence
	// TODO: Optimize with bulk insert statement
	var results []Object
	var totalCount int64

	for i, item := range arr.Elements {
		// Create a new scope with the alias bound to the current item
		innerEnv := NewEnclosedEnvironment(env)
		innerEnv.Set(node.Batch.Alias.Value, item)
		if node.Batch.IndexAlias != nil {
			innerEnv.Set(node.Batch.IndexAlias.Value, &Integer{Value: int64(i)})
		}

		// Build SQL for this item
		sql, params, err := buildInsertSQLForBatch(node, binding, innerEnv)
		if err != nil {
			return err
		}

		// Execute based on terminal
		if node.Terminal != nil && (node.Terminal.Type == "one" || node.Terminal.Type == "many") {
			result := executeInsertReturning(binding, sql, params, node.Terminal.Projection, innerEnv)
			if isError(result) {
				return result
			}
			results = append(results, result)
		} else {
			result, execErr := binding.exec(sql, params)
			if execErr != nil {
				return execErr
			}
			count, _ := result.RowsAffected()
			totalCount += count
		}
	}

	// Return based on terminal type
	if node.Terminal == nil || node.Terminal.Type == "execute" {
		return NULL
	}
	if node.Terminal.Type == "count" {
		return &Integer{Value: totalCount}
	}
	if len(results) > 0 {
		return &Array{Elements: results}
	}
	return NULL
}

// buildInsertSQLForBatch builds INSERT SQL for batch operations (no RETURNING needed in loop)
func buildInsertSQLForBatch(node *ast.InsertExpression, binding *TableBinding, env *Environment) (string, []Object, *Error) {
	var sql strings.Builder
	var params []Object
	paramIdx := 1

	var columns []string
	var placeholders []string

	for _, write := range node.Writes {
		columns = append(columns, write.Field)

		val := Eval(write.Value, env)
		if isError(val) {
			return "", nil, val.(*Error)
		}

		placeholders = append(placeholders, fmt.Sprintf("$%d", paramIdx))
		paramIdx++
		params = append(params, val)
	}

	sql.WriteString("INSERT INTO ")
	sql.WriteString(binding.TableName)
	sql.WriteString(" (")
	sql.WriteString(strings.Join(columns, ", "))
	sql.WriteString(") VALUES (")
	sql.WriteString(strings.Join(placeholders, ", "))
	sql.WriteString(")")

	// Handle upsert
	if len(node.UpsertKey) > 0 {
		sql.WriteString(" ON CONFLICT (")
		sql.WriteString(strings.Join(node.UpsertKey, ", "))
		sql.WriteString(") DO UPDATE SET ")

		var updates []string
		for _, col := range columns {
			updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
		}
		sql.WriteString(strings.Join(updates, ", "))
	}

	// Add RETURNING for batch with results
	if node.Terminal != nil && (node.Terminal.Type == "one" || node.Terminal.Type == "many") {
		sql.WriteString(" RETURNING ")
		if len(node.Terminal.Projection) == 0 || node.Terminal.Projection[0] == "*" {
			sql.WriteString("*")
		} else {
			sql.WriteString(strings.Join(node.Terminal.Projection, ", "))
		}
	}

	return sql.String(), params, nil
}

// ============================================================================
// UPDATE Expression
// ============================================================================

// evalUpdateExpression evaluates a @update(...) expression
func evalUpdateExpression(node *ast.UpdateExpression, env *Environment) Object {
	// 1. Resolve the source binding
	sourceObj, ok := env.Get(node.Source.Value)
	if !ok {
		return &Error{
			Message: fmt.Sprintf("undefined binding: %s", node.Source.Value),
			Class:   ClassUndefined,
			Code:    "REF-0001",
		}
	}

	binding, ok := sourceObj.(*TableBinding)
	if !ok {
		return &Error{
			Message: fmt.Sprintf("@update source must be a table binding, got %s", sourceObj.Type()),
			Class:   ClassType,
			Code:    "TYPE-0001",
		}
	}

	// 2. Validate field values against schema (if DSL schema is bound)
	if binding.DSLSchema != nil && len(node.Writes) > 0 {
		values := make(map[string]Object)
		for _, write := range node.Writes {
			val := Eval(write.Value, env)
			if isError(val) {
				return val
			}
			values[write.Field] = val
		}
		if validationErr := ValidateSchemaFields(values, binding.DSLSchema); validationErr != nil {
			return validationErr
		}
	}

	// 3. Build UPDATE SQL
	sql, params, err := buildUpdateSQL(node, binding, env)
	if err != nil {
		return err
	}

	// 4. Execute based on terminal
	if node.Terminal == nil {
		return &Error{
			Message: "@update requires a terminal (., .-> count, ?->, or ??->)",
			Class:   ClassParse,
			Code:    "SYN-0001",
		}
	}

	switch node.Terminal.Type {
	case "execute":
		return executeUpdate(binding, sql, params, env)
	case "count":
		return executeUpdateCount(binding, sql, params, env)
	case "one":
		return executeUpdateReturningOne(binding, sql, params, node.Terminal.Projection, env)
	case "many":
		return executeUpdateReturningMany(binding, sql, params, node.Terminal.Projection, env)
	default:
		return &Error{
			Message: fmt.Sprintf("invalid terminal type for @update: %s", node.Terminal.Type),
			Class:   ClassParse,
			Code:    "SYN-0002",
		}
	}
}

// buildUpdateSQL builds an UPDATE statement from an UpdateExpression
func buildUpdateSQL(node *ast.UpdateExpression, binding *TableBinding, env *Environment) (string, []Object, *Error) {
	var sql strings.Builder
	var params []Object
	paramIdx := 1

	sql.WriteString("UPDATE ")
	sql.WriteString(binding.TableName)
	sql.WriteString(" SET ")

	// Build SET clause
	var setClauses []string
	for _, write := range node.Writes {
		val := Eval(write.Value, env)
		if isError(val) {
			return "", nil, val.(*Error)
		}

		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", write.Field, paramIdx))
		paramIdx++
		params = append(params, val)
	}

	if len(setClauses) == 0 {
		return "", nil, &Error{
			Message: "@update requires at least one field to set",
			Class:   ClassParse,
			Code:    "SYN-0003",
		}
	}

	sql.WriteString(strings.Join(setClauses, ", "))

	// Build WHERE clause
	var whereClauses []string

	// Add soft delete filter if configured
	if binding.SoftDeleteColumn != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", binding.SoftDeleteColumn))
	}

	// Add user conditions
	for _, cond := range node.Conditions {
		clause, condParams, err := buildConditionSQL(cond, env, &paramIdx)
		if err != nil {
			return "", nil, err
		}
		params = append(params, condParams...)

		if len(whereClauses) == 0 || cond.Logic == "" {
			whereClauses = append(whereClauses, clause)
		} else {
			logic := strings.ToUpper(cond.Logic)
			if logic != "AND" && logic != "OR" {
				logic = "AND"
			}
			lastIdx := len(whereClauses) - 1
			whereClauses[lastIdx] = fmt.Sprintf("(%s %s %s)", whereClauses[lastIdx], logic, clause)
		}
	}

	if len(whereClauses) > 0 {
		sql.WriteString(" WHERE ")
		sql.WriteString(strings.Join(whereClauses, " AND "))
	}

	// Add RETURNING clause if terminal requests data
	if node.Terminal != nil && (node.Terminal.Type == "one" || node.Terminal.Type == "many") {
		sql.WriteString(" RETURNING ")
		if len(node.Terminal.Projection) == 0 || node.Terminal.Projection[0] == "*" {
			sql.WriteString("*")
		} else {
			sql.WriteString(strings.Join(node.Terminal.Projection, ", "))
		}
	}

	return sql.String(), params, nil
}

// executeUpdate executes an UPDATE without returning data
func executeUpdate(binding *TableBinding, sql string, params []Object, env *Environment) Object {
	_, err := binding.exec(sql, params)
	if err != nil {
		return err
	}
	return NULL
}

// executeUpdateCount executes an UPDATE and returns count of affected rows
func executeUpdateCount(binding *TableBinding, sql string, params []Object, env *Environment) Object {
	result, err := binding.exec(sql, params)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	return &Integer{Value: count}
}

// executeUpdateReturningOne executes an UPDATE and returns the first modified row
func executeUpdateReturningOne(binding *TableBinding, sql string, params []Object, projection []string, env *Environment) Object {
	rows, err := binding.query(sql, params)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, colErr := rows.Rows.Columns()
	if colErr != nil {
		return &Error{
			Message: fmt.Sprintf("error getting columns: %s", colErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0004",
		}
	}

	if !rows.Next() {
		return NULL
	}

	values := make([]interface{}, len(columns))
	ptrs := make([]interface{}, len(columns))
	for i := range values {
		ptrs[i] = &values[i]
	}

	if scanErr := rows.Rows.Scan(ptrs...); scanErr != nil {
		return &Error{
			Message: fmt.Sprintf("error scanning row: %s", scanErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0005",
		}
	}

	return rowToDict(columns, values, env)
}

// executeUpdateReturningMany executes an UPDATE and returns all modified rows
func executeUpdateReturningMany(binding *TableBinding, sql string, params []Object, projection []string, env *Environment) Object {
	rows, err := binding.query(sql, params)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, colErr := rows.Rows.Columns()
	if colErr != nil {
		return &Error{
			Message: fmt.Sprintf("error getting columns: %s", colErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0004",
		}
	}

	results := []Object{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}

		if scanErr := rows.Rows.Scan(ptrs...); scanErr != nil {
			return &Error{
				Message: fmt.Sprintf("error scanning row: %s", scanErr.Error()),
				Class:   ClassDatabase,
				Code:    "DB-0005",
			}
		}

		results = append(results, rowToDict(columns, values, env))
	}

	if rows.Err() != nil {
		return &Error{
			Message: fmt.Sprintf("error iterating rows: %s", rows.Err().Error()),
			Class:   ClassDatabase,
			Code:    "DB-0002",
		}
	}

	return &Array{Elements: results}
}

// ============================================================================
// DELETE Expression
// ============================================================================

// evalDeleteExpression evaluates a @delete(...) expression
func evalDeleteExpression(node *ast.DeleteExpression, env *Environment) Object {
	// 1. Resolve the source binding
	sourceObj, ok := env.Get(node.Source.Value)
	if !ok {
		return &Error{
			Message: fmt.Sprintf("undefined binding: %s", node.Source.Value),
			Class:   ClassUndefined,
			Code:    "REF-0001",
		}
	}

	binding, ok := sourceObj.(*TableBinding)
	if !ok {
		return &Error{
			Message: fmt.Sprintf("@delete source must be a table binding, got %s", sourceObj.Type()),
			Class:   ClassType,
			Code:    "TYPE-0001",
		}
	}

	// 2. Build DELETE SQL (or UPDATE for soft delete)
	sql, params, err := buildDeleteSQL(node, binding, env)
	if err != nil {
		return err
	}

	// 3. Execute based on terminal
	if node.Terminal == nil {
		return &Error{
			Message: "@delete requires a terminal (., .-> count, ?->, or ??->)",
			Class:   ClassParse,
			Code:    "SYN-0001",
		}
	}

	switch node.Terminal.Type {
	case "execute":
		return executeDelete(binding, sql, params, env)
	case "count":
		return executeDeleteCount(binding, sql, params, env)
	case "one":
		return executeDeleteReturningOne(binding, sql, params, node.Terminal.Projection, env)
	case "many":
		return executeDeleteReturningMany(binding, sql, params, node.Terminal.Projection, env)
	default:
		return &Error{
			Message: fmt.Sprintf("invalid terminal type for @delete: %s", node.Terminal.Type),
			Class:   ClassParse,
			Code:    "SYN-0002",
		}
	}
}

// buildDeleteSQL builds a DELETE statement (or UPDATE for soft delete)
func buildDeleteSQL(node *ast.DeleteExpression, binding *TableBinding, env *Environment) (string, []Object, *Error) {
	var sql strings.Builder
	var params []Object
	paramIdx := 1

	// Check if this binding uses soft deletes
	if binding.SoftDeleteColumn != "" {
		// Soft delete: UPDATE ... SET deleted_at = NOW()
		sql.WriteString("UPDATE ")
		sql.WriteString(binding.TableName)
		sql.WriteString(" SET ")
		sql.WriteString(binding.SoftDeleteColumn)
		sql.WriteString(" = datetime('now')")
	} else {
		// Hard delete: DELETE FROM ...
		sql.WriteString("DELETE FROM ")
		sql.WriteString(binding.TableName)
	}

	// Build WHERE clause
	var whereClauses []string

	// For soft delete, only delete rows that aren't already deleted
	if binding.SoftDeleteColumn != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", binding.SoftDeleteColumn))
	}

	// Add user conditions
	for _, cond := range node.Conditions {
		clause, condParams, err := buildConditionSQL(cond, env, &paramIdx)
		if err != nil {
			return "", nil, err
		}
		params = append(params, condParams...)

		if len(whereClauses) == 0 || cond.Logic == "" {
			whereClauses = append(whereClauses, clause)
		} else {
			logic := strings.ToUpper(cond.Logic)
			if logic != "AND" && logic != "OR" {
				logic = "AND"
			}
			lastIdx := len(whereClauses) - 1
			whereClauses[lastIdx] = fmt.Sprintf("(%s %s %s)", whereClauses[lastIdx], logic, clause)
		}
	}

	if len(whereClauses) > 0 {
		sql.WriteString(" WHERE ")
		sql.WriteString(strings.Join(whereClauses, " AND "))
	}

	// Add RETURNING clause if terminal requests data
	if node.Terminal != nil && (node.Terminal.Type == "one" || node.Terminal.Type == "many") {
		sql.WriteString(" RETURNING ")
		if len(node.Terminal.Projection) == 0 || node.Terminal.Projection[0] == "*" {
			sql.WriteString("*")
		} else {
			sql.WriteString(strings.Join(node.Terminal.Projection, ", "))
		}
	}

	return sql.String(), params, nil
}

// executeDelete executes a DELETE without returning data
func executeDelete(binding *TableBinding, sql string, params []Object, env *Environment) Object {
	_, err := binding.exec(sql, params)
	if err != nil {
		return err
	}
	return NULL
}

// executeDeleteCount executes a DELETE and returns count of affected rows
func executeDeleteCount(binding *TableBinding, sql string, params []Object, env *Environment) Object {
	result, err := binding.exec(sql, params)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	return &Integer{Value: count}
}

// executeDeleteReturningOne executes a DELETE and returns the first deleted row
func executeDeleteReturningOne(binding *TableBinding, sql string, params []Object, projection []string, env *Environment) Object {
	rows, err := binding.query(sql, params)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, colErr := rows.Rows.Columns()
	if colErr != nil {
		return &Error{
			Message: fmt.Sprintf("error getting columns: %s", colErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0004",
		}
	}

	if !rows.Next() {
		return NULL
	}

	values := make([]interface{}, len(columns))
	ptrs := make([]interface{}, len(columns))
	for i := range values {
		ptrs[i] = &values[i]
	}

	if scanErr := rows.Rows.Scan(ptrs...); scanErr != nil {
		return &Error{
			Message: fmt.Sprintf("error scanning row: %s", scanErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0005",
		}
	}

	return rowToDict(columns, values, env)
}

// executeDeleteReturningMany executes a DELETE and returns all deleted rows
func executeDeleteReturningMany(binding *TableBinding, sql string, params []Object, projection []string, env *Environment) Object {
	rows, err := binding.query(sql, params)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, colErr := rows.Rows.Columns()
	if colErr != nil {
		return &Error{
			Message: fmt.Sprintf("error getting columns: %s", colErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0004",
		}
	}

	results := []Object{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}

		if scanErr := rows.Rows.Scan(ptrs...); scanErr != nil {
			return &Error{
				Message: fmt.Sprintf("error scanning row: %s", scanErr.Error()),
				Class:   ClassDatabase,
				Code:    "DB-0005",
			}
		}

		results = append(results, rowToDict(columns, values, env))
	}

	if rows.Err() != nil {
		return &Error{
			Message: fmt.Sprintf("error iterating rows: %s", rows.Err().Error()),
			Class:   ClassDatabase,
			Code:    "DB-0002",
		}
	}

	return &Array{Elements: results}
}

// evalTransactionExpression evaluates a @transaction { ... } expression
func evalTransactionExpression(node *ast.TransactionExpression, env *Environment) Object {
	// First, find all TableBindings used in the transaction to get the DB connection
	// For now, we'll get the DB from the first DSL operation we find
	var dbConn *DBConnection

	// Walk through statements to find a TableBinding
	for _, stmt := range node.Statements {
		conn := findDBConnectionInStatement(stmt, env)
		if conn != nil {
			dbConn = conn
			break
		}
	}

	if dbConn == nil {
		return &Error{
			Message: "@transaction requires at least one database operation",
			Class:   ClassDatabase,
			Code:    "DB-0012",
		}
	}

	// Check if already in a transaction
	if dbConn.Tx != nil {
		return &Error{
			Message: "nested transactions are not supported",
			Class:   ClassDatabase,
			Code:    "DB-0013",
		}
	}

	// Begin transaction
	tx, err := dbConn.DB.Begin()
	if err != nil {
		dbConn.LastError = err.Error()
		return &Error{
			Message: fmt.Sprintf("failed to begin transaction: %s", err.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0014",
		}
	}

	// Set transaction on connection so all queries use it
	dbConn.Tx = tx
	dbConn.InTransaction = true

	// Execute statements in a new environment scope
	transactionEnv := NewEnclosedEnvironment(env)
	var lastResult Object = NULL

	for _, stmt := range node.Statements {
		result := Eval(stmt, transactionEnv)

		// Check for errors - rollback on any error
		if isError(result) {
			rollbackErr := tx.Rollback()
			dbConn.Tx = nil
			dbConn.InTransaction = false
			if rollbackErr != nil {
				// Include rollback error in message
				if errObj, ok := result.(*Error); ok {
					errObj.Message = fmt.Sprintf("%s (rollback also failed: %s)", errObj.Message, rollbackErr.Error())
				}
			}
			return result
		}

		// Check for return statements
		if returnVal, ok := result.(*ReturnValue); ok {
			// Commit before returning
			if commitErr := tx.Commit(); commitErr != nil {
				dbConn.Tx = nil
				dbConn.InTransaction = false
				return &Error{
					Message: fmt.Sprintf("transaction commit failed: %s", commitErr.Error()),
					Class:   ClassDatabase,
					Code:    "DB-0015",
				}
			}
			dbConn.Tx = nil
			dbConn.InTransaction = false
			return returnVal.Value
		}

		lastResult = result
	}

	// Commit transaction
	if commitErr := tx.Commit(); commitErr != nil {
		dbConn.Tx = nil
		dbConn.InTransaction = false
		return &Error{
			Message: fmt.Sprintf("transaction commit failed: %s", commitErr.Error()),
			Class:   ClassDatabase,
			Code:    "DB-0015",
		}
	}

	dbConn.Tx = nil
	dbConn.InTransaction = false
	return lastResult
}

// findDBConnectionInStatement finds a DBConnection from a statement by looking for TableBinding references
func findDBConnectionInStatement(stmt ast.Statement, env *Environment) *DBConnection {
	switch s := stmt.(type) {
	case *ast.LetStatement:
		return findDBConnectionInExpression(s.Value, env)
	case *ast.ExpressionStatement:
		return findDBConnectionInExpression(s.Expression, env)
	}
	return nil
}

// findDBConnectionInExpression finds a DBConnection from an expression
func findDBConnectionInExpression(expr ast.Expression, env *Environment) *DBConnection {
	switch e := expr.(type) {
	case *ast.QueryExpression:
		if e.Source != nil {
			if obj, ok := env.Get(e.Source.Value); ok {
				if binding, ok := obj.(*TableBinding); ok {
					return binding.DB
				}
			}
		}
	case *ast.InsertExpression:
		if e.Source != nil {
			if obj, ok := env.Get(e.Source.Value); ok {
				if binding, ok := obj.(*TableBinding); ok {
					return binding.DB
				}
			}
		}
	case *ast.UpdateExpression:
		if e.Source != nil {
			if obj, ok := env.Get(e.Source.Value); ok {
				if binding, ok := obj.(*TableBinding); ok {
					return binding.DB
				}
			}
		}
	case *ast.DeleteExpression:
		if e.Source != nil {
			if obj, ok := env.Get(e.Source.Value); ok {
				if binding, ok := obj.(*TableBinding); ok {
					return binding.DB
				}
			}
		}
	}
	return nil
}
