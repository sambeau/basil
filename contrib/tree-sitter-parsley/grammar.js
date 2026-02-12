/**
 * @file Tree-sitter grammar for Parsley — direct translation from Go source
 * @author Basil Contributors
 * @license MIT
 *
 * Source of truth:
 *   pkg/parsley/lexer/lexer.go  — tokens, keywords, operator patterns
 *   pkg/parsley/parser/parser.go — grammar rules, precedence, all parse* functions
 */

/// <reference types="tree-sitter-cli/dsl" />
// @ts-check

// Precedence levels — direct from parser.go L17-29
const PREC = {
  LOWEST: 0,
  COMMA: 1, // , ==> ==>> =/=> =/=>>
  OR: 2, // or | || ??
  AND: 3, // and & &&
  EQUALS: 4, // == != ~ !~ in is <=?=> <=??=> <=!=> <=#=>
  COMPARE: 5, // < > <= >=
  SUM: 6, // + - ..
  CONCAT: 7, // ++
  PRODUCT: 8, // * / %
  PREFIX: 9, // -x !x not x
  INDEX: 10, // a[i] a.b
  CALL: 11, // f(x)
  TAG: 12, // tags get highest so <tag> is preferred over < comparison
};

module.exports = grammar({
  name: "parsley",

  extras: ($) => [/[\s;]/, $.comment],

  word: ($) => $.identifier,

  // External scanner tokens — see src/scanner.c
  externals: ($) => [$.raw_text, $.raw_text_interpolation_start],

  conflicts: ($) => [
    // { — dictionary literal vs block
    [$.dictionary_literal, $.block],
    // [ — array pattern vs array literal
    [$.array_pattern, $.array_literal],
    // identifier — could be pattern or primary expression (also covers for iteration vs mapping)
    [$._primary_expression, $._pattern],
    // dict pattern in assignment vs dict literal in expression vs block
    [$.dictionary_pattern, $.dictionary_literal, $.block],
    // identifier followed by = or <== — assignment vs expression
    [$.assignment_statement, $._primary_expression],
    // assignment target can be index/member expression
    [$.assignment_statement, $._expression],
    // { identifier } — dict pattern vs expression in braces
    [$._primary_expression, $.dictionary_pattern],
    // if (expr) — parenthesized_expression vs if condition in parens
    [$.parenthesized_expression, $.if_expression],
    // if (expr) block — block as expression vs if consequence
    [$._expression, $.if_expression],
  ],

  rules: {
    // ================================================================
    // Program structure — parser.go L276-289 (ParseProgram)
    // ================================================================

    source_file: ($) => repeat($._statement),

    // ================================================================
    // Statements — parser.go L292-370 (parseStatement)
    // ================================================================

    _statement: ($) =>
      choice(
        $.let_statement,
        $.assignment_statement,
        $.dict_destructuring_assignment,
        $.export_statement,
        $.return_statement,
        $.check_statement,
        $.expression_statement,
      ),

    // parser.go L499-669 (parseLetStatement)
    // Forms: let x = expr, let {a,b} = expr, let [a,b] = expr
    // Also: let x <== expr, let x <=/= expr
    let_statement: ($) =>
      seq(
        "let",
        field("pattern", $._pattern),
        field("operator", choice("=", "<==", "<=/=")),
        field("value", $._expression),
      ),

    // parser.go L673-730 (parseAssignmentStatement)
    // Forms: x = expr, obj.name = expr, arr[0] = expr
    // Also: x <== expr, x <=/= expr
    assignment_statement: ($) =>
      prec.right(
        PREC.LOWEST,
        seq(
          field(
            "left",
            choice($.identifier, $.member_expression, $.index_expression),
          ),
          field("operator", choice("=", "<==", "<=/=")),
          field("right", $._expression),
        ),
      ),

    // parser.go L733-790 (parseDictDestructuringAssignment)
    // Form: {a, b} = expr, {a, b} <== expr
    dict_destructuring_assignment: ($) =>
      prec.dynamic(
        1,
        seq(
          field("pattern", $.dictionary_pattern),
          field("operator", choice("=", "<==", "<=/=")),
          field("value", $._expression),
        ),
      ),

    // parser.go L379-496 (parseExportStatement, parseComputedExportStatement)
    // Forms:
    //   export let pattern = expr
    //   export name = expr
    //   export name              (bare export)
    //   export computed name = expr
    //   export computed name { body }
    //   export @schema Name { ... }
    //   export {a,b} = expr
    export_statement: ($) =>
      prec.right(
        seq(
          "export",
          choice(
            // export let pattern = expr
            seq(
              "let",
              field("pattern", $._pattern),
              "=",
              field("value", $._expression),
            ),
            // export computed name = expr | export computed name { body }
            seq(
              "computed",
              field("name", $.identifier),
              choice(
                seq("=", field("value", $._expression)),
                field("value", $.block),
              ),
            ),
            // export @schema — passes through to expression
            seq(field("value", $.schema_declaration)),
            // export name = expr
            seq(
              field("name", $.identifier),
              "=",
              field("value", $._expression),
            ),
            // export {a,b} = expr
            seq(
              field("pattern", $.dictionary_pattern),
              "=",
              field("value", $._expression),
            ),
            // export name (bare export)
            field("name", $.identifier),
          ),
        ),
      ),

    // parser.go L793-805 (parseReturnStatement)
    return_statement: ($) =>
      prec.right(seq("return", optional(field("value", $._expression)))),

    // parser.go L808-828 (parseCheckStatement)
    // check CONDITION else VALUE — else is REQUIRED
    check_statement: ($) =>
      seq(
        "check",
        field("condition", $._expression),
        "else",
        field("fallback", $._expression),
      ),

    // parser.go L863-945 (parseExpressionStatement)
    expression_statement: ($) => $._expression,

    // ================================================================
    // Expressions — parser.go L957-1012 (parseExpression)
    // ================================================================

    _expression: ($) =>
      choice(
        $._primary_expression,
        $.prefix_expression,
        $.read_expression,
        $.fetch_expression,
        $.infix_expression,
        $.not_in_expression,
        $.is_expression,
        $.write_expression,
        $.call_expression,
        $.index_expression,
        $.slice_expression,
        $.member_expression,
        $.function_expression,
        $.if_expression,
        $.for_expression,
        $.try_expression,
        $.import_expression,
        $.tag_expression,
        $.parenthesized_expression,
        $.block,
        $.schema_declaration,
        $.table_expression,
        $.query_expression,
        $.mutation_expression,
      ),

    _primary_expression: ($) =>
      choice($.identifier, $._literal, $.array_literal, $.dictionary_literal),

    // parser.go L2079-2137 (parseGroupedExpression)
    parenthesized_expression: ($) => seq("(", $._expression, ")"),

    // ================================================================
    // Prefix expressions — parser.go L1963-2000
    // ================================================================

    // parsePrefixExpression: -x, !x, not x
    prefix_expression: ($) =>
      prec.right(
        PREC.PREFIX,
        seq(
          field("operator", choice("-", "!", "not")),
          field("operand", $._expression),
        ),
      ),

    // parser.go L1977-1987 (parseReadExpression): <== expr
    read_expression: ($) =>
      prec.right(PREC.PREFIX, seq("<==", field("source", $._expression))),

    // parser.go L1990-2000 (parseFetchExpression): <=/= expr
    fetch_expression: ($) =>
      prec.right(PREC.PREFIX, seq("<=/=", field("source", $._expression))),

    // ================================================================
    // Infix expressions — parser.go L2002-2014, precedences L33-67
    // ================================================================

    infix_expression: ($) =>
      choice(
        // PRODUCT: * / % — parser.go precedences map
        ...[
          ["*", PREC.PRODUCT],
          ["/", PREC.PRODUCT],
          ["%", PREC.PRODUCT],
        ].map(([op, prec_val]) =>
          prec.left(
            prec_val,
            seq(
              field("left", $._expression),
              field("operator", op),
              field("right", $._expression),
            ),
          ),
        ),
        // CONCAT: ++
        prec.left(
          PREC.CONCAT,
          seq(
            field("left", $._expression),
            field("operator", "++"),
            field("right", $._expression),
          ),
        ),
        // SUM: + - ..
        ...["+", "-", ".."].map((op) =>
          prec.left(
            PREC.SUM,
            seq(
              field("left", $._expression),
              field("operator", op),
              field("right", $._expression),
            ),
          ),
        ),
        // COMPARE: < > <= >=
        ...["<", ">", "<=", ">="].map((op) =>
          prec.left(
            PREC.COMPARE,
            seq(
              field("left", $._expression),
              field("operator", op),
              field("right", $._expression),
            ),
          ),
        ),
        // EQUALS: == != ~ !~ in
        ...["==", "!=", "~", "!~", "in"].map((op) =>
          prec.left(
            PREC.EQUALS,
            seq(
              field("left", $._expression),
              field("operator", op),
              field("right", $._expression),
            ),
          ),
        ),
        // AND: and && &
        ...["and", "&&", "&"].map((op) =>
          prec.left(
            PREC.AND,
            seq(
              field("left", $._expression),
              field("operator", op),
              field("right", $._expression),
            ),
          ),
        ),
        // OR: or || | ??
        ...["or", "||", "|", "??"].map((op) =>
          prec.left(
            PREC.OR,
            seq(
              field("left", $._expression),
              field("operator", op),
              field("right", $._expression),
            ),
          ),
        ),
        // Database operators at EQUALS: <=?=> <=??=> <=!=>
        ...["<=?=>", "<=??=>", "<=!=>"].map((op) =>
          prec.left(
            PREC.EQUALS,
            seq(
              field("left", $._expression),
              field("operator", op),
              field("right", $._expression),
            ),
          ),
        ),
        // Process execution at EQUALS: <=#=>
        prec.left(
          PREC.EQUALS,
          seq(
            field("left", $._expression),
            field("operator", "<=#=>"),
            field("right", $._expression),
          ),
        ),
      ),

    // File I/O write operators at COMMA precedence — parser.go parseExpression special handling
    // ==> ==>> =/=> =/=>>
    write_expression: ($) =>
      prec.left(
        PREC.COMMA,
        seq(
          field("value", $._expression),
          field("operator", choice("==>", "==>>", "=/=>", "=/=>>")),
          field("target", $._expression),
        ),
      ),

    // parser.go L2019-2041 (parseNotInExpression)
    // expr not in expr — compound operator
    not_in_expression: ($) =>
      prec.left(
        PREC.EQUALS,
        seq(
          field("left", $._expression),
          "not",
          "in",
          field("right", $._expression),
        ),
      ),

    // parser.go L2045-2064 (parseIsExpression)
    // expr is expr, expr is not expr
    // Note: "is not" parses as is_expression with prefix_expression(not, schema).
    // This is a known limitation — tree-sitter cannot disambiguate "is not" from
    // "is (not expr)" without an external scanner. Functionally equivalent for highlighting.
    is_expression: ($) =>
      prec.left(
        PREC.EQUALS,
        seq(
          field("value", $._expression),
          "is",
          field("schema", $._expression),
        ),
      ),

    // ================================================================
    // Call, index, member — parser.go L2498-2637, L3082-3097
    // ================================================================

    // parser.go L2498-2531 (parseCallExpression)
    call_expression: ($) =>
      prec(
        PREC.CALL,
        seq(field("function", $._expression), field("arguments", $.arguments)),
      ),

    arguments: ($) => seq("(", commaSep($._expression), ")"),

    // parser.go L2579-2613 (parseIndexOrSliceExpression)
    index_expression: ($) =>
      prec(
        PREC.INDEX,
        seq(
          field("object", $._expression),
          "[",
          optional("?"),
          field("index", $._expression),
          "]",
        ),
      ),

    // parser.go L2615-2637 (parseSliceExpression)
    slice_expression: ($) =>
      prec(
        PREC.INDEX,
        seq(
          field("object", $._expression),
          "[",
          optional(field("start", $._expression)),
          ":",
          optional(field("end", $._expression)),
          "]",
        ),
      ),

    // parser.go L3082-3097 (parseDotExpression)
    member_expression: ($) =>
      prec(
        PREC.INDEX,
        seq(
          field("object", $._expression),
          ".",
          field("property", choice($.identifier, alias("as", $.identifier))),
        ),
      ),

    // ================================================================
    // Functions and blocks — parser.go L2286-2385
    // ================================================================

    // parser.go L2286-2301 (parseBlockStatement)
    block: ($) => seq("{", repeat($._statement), "}"),

    // parser.go L2303-2327 (parseFunctionLiteral)
    // Body is ALWAYS a block — no expression bodies
    function_expression: ($) =>
      seq(
        choice("fn", "function"),
        optional(field("parameters", $.parameter_list)),
        field("body", $.block),
      ),

    // parser.go L2330-2360 (parseFunctionParametersNew)
    parameter_list: ($) =>
      seq(
        "(",
        commaSep(
          choice(
            $.identifier,
            $._default_param,
            $._rest_param,
            $.array_pattern,
            $.dictionary_pattern,
          ),
        ),
        ")",
      ),

    // parser.go L2363-2385 (parseFunctionParameter — default value)
    _default_param: ($) =>
      seq(field("name", $.identifier), "=", field("default", $._expression)),

    // parser.go L2363-2385 (parseFunctionParameter — rest)
    _rest_param: ($) => seq("...", field("name", $.identifier)),

    // ================================================================
    // Control flow — parser.go L2171-2496
    // ================================================================

    // parser.go L2171-2284 (parseIfExpression)
    // Two forms:
    //   With parens: if (cond) consequence [else alternative]
    //     consequence can be block or single expression/statement
    //   Without parens: if cond { body } [else ...]
    //     consequence must be a block
    if_expression: ($) =>
      prec.dynamic(
        10,
        prec.right(
          PREC.LOWEST,
          seq(
            "if",
            choice(
              prec.dynamic(
                20,
                seq(
                  "(",
                  field("condition", $._expression),
                  ")",
                  field(
                    "consequence",
                    choice(prec.dynamic(10, $.block), $._statement),
                  ),
                ),
              ),
              seq(
                field("condition", $._expression),
                field("consequence", $.block),
              ),
            ),
            optional(
              seq(
                "else",
                field(
                  "alternative",
                  choice(
                    prec.dynamic(10, $.block),
                    prec.dynamic(5, $.if_expression),
                    $._statement,
                  ),
                ),
              ),
            ),
          ),
        ),
      ),

    // parser.go L2390-2496 (parseForExpression)
    // Iteration form: for [var|key,value] in iterable { body }
    // Mapping form:   for (iterable) function
    for_expression: ($) =>
      prec.right(
        PREC.LOWEST,
        seq(
          "for",
          choice(
            // Iteration with parens: for (x in arr) { body } or for (k, v in dict) { body }
            // prec.dynamic(10) to prefer iteration over mapping when ambiguous
            prec.dynamic(
              10,
              seq(
                "(",
                optional(seq(field("key", $.identifier), ",")),
                field("variable", $._pattern),
                "in",
                field("iterable", $._expression),
                ")",
                field("body", $.block),
              ),
            ),
            // Iteration without parens: for x in arr { body }
            prec.dynamic(
              10,
              seq(
                optional(seq(field("key", $.identifier), ",")),
                field("variable", $._pattern),
                "in",
                field("iterable", $._expression),
                field("body", $.block),
              ),
            ),
            // Mapping form: for (iterable) function — REQUIRES parens
            seq(
              "(",
              field("iterable", $._expression),
              ")",
              field("mapper", $._expression),
            ),
          ),
        ),
      ),

    // parser.go L1838-1863 (parseTryExpression)
    // try must be followed by a call expression
    try_expression: ($) =>
      prec.right(PREC.PREFIX, seq("try", field("call", $._expression))),

    // parser.go L1871-1931 (parseImportExpression)
    import_expression: ($) =>
      prec.right(
        seq(
          "import",
          field("source", $._expression),
          optional(seq("as", field("alias", $.identifier))),
        ),
      ),

    // ================================================================
    // Tags — parser.go L1354-1528
    // ================================================================

    // parser.go L1354-1365 (parseTagLiteral), L1367-1528 (parseTagPair, parseTagContents)
    // Tag content is repeat(_statement) — full Parsley code, NOT text with interpolation
    // Exception: <style> and <script> tags use raw text mode with @{} interpolation
    tag_expression: ($) =>
      prec(
        PREC.TAG,
        choice(
          $.self_closing_tag,
          $.style_tag,
          $.script_tag,
          seq($.open_tag, repeat($._tag_child), $.close_tag),
          // Grouping tags: <>...</>
          seq("<>", repeat($._tag_child), "</>"),
        ),
      ),

    // Style tag with raw text content and @{} interpolation
    // Uses token with higher precedence to ensure <style wins over generic tag_start
    style_tag: ($) =>
      seq(
        token(prec(PREC.TAG + 1, "<style")),
        repeat(choice($.tag_attribute, $.tag_spread_attribute)),
        ">",
        repeat($._raw_text_content),
        token(prec(PREC.TAG + 1, "</style>")),
      ),

    // Script tag with raw text content and @{} interpolation
    // Uses token with higher precedence to ensure <script wins over generic tag_start
    script_tag: ($) =>
      seq(
        token(prec(PREC.TAG + 1, "<script")),
        repeat(choice($.tag_attribute, $.tag_spread_attribute)),
        ">",
        repeat($._raw_text_content),
        token(prec(PREC.TAG + 1, "</script>")),
      ),

    // Raw text content: either literal text or @{expr} interpolation
    _raw_text_content: ($) => choice($.raw_text, $.raw_text_interpolation),

    // @{expr} interpolation in raw text (style/script tags)
    raw_text_interpolation: ($) =>
      seq($.raw_text_interpolation_start, $._expression, "}"),

    self_closing_tag: ($) =>
      seq(
        $.tag_start,
        repeat(choice($.tag_attribute, $.tag_spread_attribute)),
        "/>",
      ),

    open_tag: ($) =>
      seq(
        $.tag_start,
        repeat(choice($.tag_attribute, $.tag_spread_attribute)),
        ">",
      ),

    close_tag: ($) => seq("</", field("name", $.tag_name), ">"),

    // Tag start: < immediately followed by tag name (no whitespace)
    // This token disambiguates from the less-than operator
    // The token captures <tagname as a single unit so tree-sitter can
    // distinguish it from < followed by an identifier in expression context
    tag_start: ($) => token(prec(PREC.TAG, /<[a-zA-Z][a-zA-Z0-9.-]*/)),

    // Tag name: letters, digits, hyphens (for HTML elements and components)
    // Used in close_tag where we need just the name without <
    tag_name: ($) => /[a-zA-Z][a-zA-Z0-9.-]*/,

    // Tag children are Parsley code — parser.go L1421-1528 (parseTagContents)
    // tag_expression is reachable via _expression → expression_statement
    _tag_child: ($) => $._statement,

    // parser.go parseTagAttributes (L1645-1814)
    tag_attribute: ($) =>
      choice(
        // name="value" or name={expr} or name=number or name=identifier
        seq(
          field("name", $.attribute_name),
          "=",
          field(
            "value",
            choice(
              $.string,
              $.template_string,
              $.raw_string,
              $.tag_expression_value,
              $.number,
              $.identifier,
            ),
          ),
        ),
        // bare boolean attribute
        field("name", $.attribute_name),
      ),

    // Attribute names allow @field, @record, hyphens, colons (for namespaced attrs like xlink:href), etc.
    attribute_name: ($) => /[a-zA-Z@][a-zA-Z0-9_:-]*/,

    // Expression value in tag attribute: {expr}
    tag_expression_value: ($) => seq("{", $._expression, "}"),

    // Spread attribute: ...identifier
    tag_spread_attribute: ($) => seq("...", field("expression", $.identifier)),

    // ================================================================
    // Patterns (destructuring) — parser.go L3100-3250
    // ================================================================

    _pattern: ($) =>
      choice(
        $.identifier,
        $.array_pattern,
        $.dictionary_pattern,
        alias("_", $.identifier),
      ),

    // parser.go L3193-3250 (parseArrayDestructuringPattern)
    array_pattern: ($) =>
      seq(
        "[",
        commaSep(choice($._pattern, seq("...", optional($.identifier)))),
        "]",
      ),

    // parser.go L3100-3190 (parseDictDestructuringPattern)
    dictionary_pattern: ($) =>
      seq(
        "{",
        commaSep(
          choice(
            // key: nested_pattern
            seq(field("key", $.identifier), ":", field("value", $._pattern)),
            // key as alias
            seq(field("key", $.identifier), "as", field("alias", $.identifier)),
            // simple key (shorthand)
            $.identifier,
            // rest: ...identifier
            seq("...", optional($.identifier)),
          ),
        ),
        "}",
      ),

    // ================================================================
    // Schema declarations — parser.go L3253-3410 (Phase 2 basic support)
    // ================================================================

    // parser.go L3253-3293 (parseSchemaDeclaration)
    schema_declaration: ($) =>
      seq(
        "@schema",
        field("name", $.identifier),
        "{",
        repeat($.schema_field),
        "}",
      ),

    // parser.go L3302-3410 (parseSchemaField)
    schema_field: ($) =>
      seq(
        field("name", $.identifier),
        ":",
        choice(
          // Array type: [Type]
          seq("[", field("type", $.identifier), "]"),
          // Regular type
          field("type", $.identifier),
        ),
        optional("?"),
        optional($.enum_values),
        optional($.type_options),
        optional(seq("=", field("default", $._expression))),
        optional(seq("|", field("metadata", $._expression))),
        optional(seq("via", field("relation", $.identifier))),
      ),

    // enum["a", "b", "c"] — uses square brackets
    enum_values: ($) =>
      seq("[", commaSep(choice($.string, $.number, $.identifier)), "]"),

    // type(auto: true, min: 1) or type(required) — uses parentheses
    type_options: ($) => seq("(", commaSep($.type_option), ")"),

    type_option: ($) =>
      choice(
        // key: value pair
        seq(field("name", $.identifier), ":", field("value", $._expression)),
        // bare flag (treated as boolean true)
        field("name", $.identifier),
      ),

    // ================================================================
    // Table literals — parser.go L3413-3523
    // ================================================================

    table_expression: ($) =>
      seq(
        "@table",
        optional(seq("(", field("schema", $.identifier), ")")),
        "[",
        commaSep($._expression),
        "]",
      ),

    // ================================================================
    // Query/mutation expressions — parser.go L3611-3764, L4588-4800
    // ================================================================

    // @query(Source | conditions | modifiers + by group ??-> projection)
    query_expression: ($) => seq("@query", "(", optional($.query_body), ")"),

    query_body: ($) =>
      seq(
        $.query_source,
        repeat($.query_clause),
        optional($.query_group_by),
        optional($.query_terminal),
      ),

    query_source: ($) =>
      seq(
        field("table", $.identifier),
        optional(seq("as", field("alias", $.identifier))),
      ),

    // Pipe-separated clauses: conditions, modifiers, computed fields
    query_clause: ($) =>
      seq(
        "|",
        choice(
          $.query_condition_group,
          $.query_condition,
          $.query_modifier,
          $.query_computed_field,
        ),
      ),

    // Condition: field op value OR field between X and Y OR field is [not] null
    query_condition: ($) =>
      choice(
        seq(
          optional(choice("not", "!")),
          $.query_field_ref,
          $.query_operator,
          $.query_value,
        ),
        // between has special syntax: field between X and Y
        seq(
          optional(choice("not", "!")),
          $.query_field_ref,
          "between",
          $.query_value,
          "and",
          $.query_value,
        ),
        // is null / is not null don't take a value
        seq(
          optional(choice("not", "!")),
          $.query_field_ref,
          $.query_null_check,
        ),
      ),

    // is null / is not null - separate from query_operator since no value follows
    query_null_check: ($) =>
      choice(seq("is", "null"), seq("is", "not", "null")),

    // Grouped conditions: (cond1 or cond2 and cond3)
    query_condition_group: ($) =>
      seq(
        optional(choice("not", "!")),
        "(",
        $.query_condition,
        repeat(seq(choice("and", "or"), $.query_condition)),
        ")",
      ),

    // Field reference: field or table.field
    query_field_ref: ($) =>
      choice($.identifier, seq($.identifier, ".", $.identifier)),

    // Comparison operators (excluding between and is null which have special syntax)
    query_operator: ($) =>
      choice("==", "!=", "<", ">", "<=", ">=", "in", seq("not", "in"), "like"),

    // Value in a condition: interpolation, literal, or column ref
    query_value: ($) =>
      choice(
        $.query_interpolation,
        $.string,
        $.number,
        $.boolean,
        $.array_literal,
        $.query_column_ref,
      ),

    // Interpolated Parsley expression: {expr}
    query_interpolation: ($) => seq("{", $._expression, "}"),

    // Column reference (bare identifier or table.field in query context)
    query_column_ref: ($) =>
      prec.right(choice($.identifier, seq($.identifier, ".", $.identifier))),

    // Modifiers: order, limit, offset, with
    query_modifier: ($) =>
      choice(
        $.query_order_modifier,
        $.query_limit_modifier,
        $.query_offset_modifier,
        $.query_with_modifier,
      ),

    query_order_modifier: ($) =>
      seq("order", $.query_order_field, repeat(seq(",", $.query_order_field))),

    query_order_field: ($) =>
      seq($.identifier, optional(choice("asc", "desc"))),

    query_limit_modifier: ($) => seq("limit", $.number),

    query_offset_modifier: ($) => seq("offset", $.number),

    query_with_modifier: ($) =>
      seq("with", $.identifier, repeat(seq(",", $.identifier))),

    // Computed fields: name: aggregate(field) or name <-Table | cond ?-> agg
    query_computed_field: ($) =>
      choice(
        // Aggregate: total: count or total: sum(amount)
        seq(field("name", $.identifier), ":", $.query_aggregate),
        // Correlated subquery: items <-OrderItems | order.id == orderId ?-> count
        seq(field("name", $.identifier), "<-", $.query_subquery),
      ),

    query_aggregate: ($) =>
      choice(
        "count",
        seq(
          choice("count", "sum", "avg", "min", "max"),
          "(",
          $.identifier,
          ")",
        ),
        $.identifier, // bare field reference
      ),

    query_subquery: ($) =>
      prec.right(
        seq(
          field("table", $.identifier),
          repeat($.query_clause),
          optional($.query_terminal),
        ),
      ),

    // Group by: + by field1, field2
    query_group_by: ($) =>
      seq("+", "by", $.identifier, repeat(seq(",", $.identifier))),

    // Terminal: ?-> or ??-> followed by projection
    query_terminal: ($) =>
      seq(
        choice("?->", "??->", "?!->", "??!->", ".->", "."),
        optional($.query_projection),
      ),

    query_projection: ($) =>
      choice("*", "toSQL", seq($.identifier, repeat(seq(",", $.identifier)))),

    mutation_expression: ($) =>
      seq(
        choice("@insert", "@update", "@delete", "@transaction"),
        "(",
        commaSep($._expression),
        ")",
      ),

    // ================================================================
    // Literals
    // ================================================================

    _literal: ($) =>
      choice(
        $.number,
        $.string,
        $.template_string,
        $.raw_string,
        $.regex,
        $.boolean,
        $.money,
        $._at_literal,
      ),

    // -------------------- Numbers --------------------
    // lexer.go L1257-1272 (readNumber)
    number: ($) => /\d+(\.\d+)?/,

    // -------------------- Booleans --------------------
    // parser.go L1834-1836 — true/false are keywords
    // NOTE: null is NOT a keyword — it's an identifier that the evaluator special-cases
    boolean: ($) => choice("true", "false"),

    // -------------------- Strings --------------------
    // lexer.go L1570-1600 (readString) — double-quoted, NO interpolation
    // Note: Only template strings (backticks) and raw strings (single quotes with @{}) support interpolation
    string: ($) =>
      seq('"', repeat(choice($.escape_sequence, $._string_content)), '"'),

    // Higher precedence than TAG (12) to prevent <tag> inside strings from being tokenized as tag_start
    // Includes { since double-quoted strings don't have interpolation
    _string_content: ($) => token.immediate(prec(PREC.TAG + 2, /[^"\\]+/)),

    // lexer.go L1642-1664 (readTemplate) — backtick strings with {expr} interpolation
    template_string: ($) =>
      seq(
        "`",
        repeat(choice($.escape_sequence, $.interpolation, $._template_content)),
        "`",
      ),

    // Higher precedence than TAG (12) to prevent <tag> inside strings from being tokenized as tag_start
    _template_content: ($) => token.immediate(prec(PREC.TAG + 2, /[^`\\{]+/)),

    // lexer.go L1605-1639 (readRawString) — single-quoted, with @{expr} interpolation
    raw_string: ($) =>
      seq(
        "'",
        repeat(
          choice($.escape_sequence, $.raw_interpolation, $._raw_string_content),
        ),
        "'",
      ),

    // Higher precedence than TAG (12) to prevent <tag> inside strings from being tokenized as tag_start
    _raw_string_content: ($) =>
      token.immediate(prec(PREC.TAG + 2, /([^'\\@]|@[^{'])+/)),

    // Shared string internals
    escape_sequence: ($) => token.immediate(prec(1, /\\./)),

    interpolation: ($) => seq(token.immediate("{"), $._expression, "}"),

    raw_interpolation: ($) =>
      seq(token.immediate(seq("@", "{")), $._expression, "}"),

    // -------------------- Regex --------------------
    // lexer.go L2293-2327 (readRegex), shouldTreatAsRegex L2879-2896
    // Regex only valid in prefix position (where division isn't expected)
    regex: ($) =>
      token(seq("/", /[^/\n*][^/\n]*/, "/", optional(/[gimsuvy]+/))),

    // -------------------- Money --------------------
    // lexer.go L1275-1565 (readMoneyLiteral, isCurrencyCodeStart, isCompoundCurrencySymbol)
    money: ($) =>
      token(
        seq(
          choice(
            /[$\u00A3\u20AC\u00A5]/, // $ £ € ¥
            /[A-Z]{1,2}[$\u00A3\u20AC\u00A5]/, // CA$ AU$ HK$ S$ CN¥
            /[A-Z]{3}#/, // USD# GBP# EUR# etc.
          ),
          /\d+(\.\d{1,2})?/,
        ),
      ),

    // -------------------- @-literals --------------------
    // lexer.go L2474-2655 (detectAtLiteralType)
    _at_literal: ($) =>
      choice(
        $.datetime_literal,
        $.time_now_literal,
        $.duration_literal,
        $.connection_literal,
        $.context_literal,
        $.stdlib_import,
        $.stdio_literal,
        $.path_literal,
        $.url_literal,
        $.path_template,
      ),

    // @2024-01-15, @2024-01-15T10:30:00Z, @12:30:00
    // lexer.go L2332-2422 (readDatetimeLiteral)
    datetime_literal: ($) =>
      token(
        prec(
          2,
          seq(
            "@",
            choice(
              // Full date or datetime
              seq(
                /\d{4}-\d{2}-\d{2}/,
                optional(
                  seq(
                    "T",
                    /\d{2}:\d{2}/,
                    optional(seq(":", /\d{2}/)),
                    optional(/(\.\d+)?(Z|[+-]\d{2}:\d{2})?/),
                  ),
                ),
              ),
              // Time only: HH:MM or HH:MM:SS
              seq(/\d{1,2}:\d{2}/, optional(seq(":", /\d{2}/))),
            ),
          ),
        ),
      ),

    // @now, @today, @timeNow, @dateNow
    // lexer.go detectAtLiteralType — keyword checks
    time_now_literal: ($) =>
      token(prec(3, seq("@", choice("now", "today", "timeNow", "dateNow")))),

    // @2h30m, @-7d, @1y6mo
    // lexer.go L2427-2465 (readDurationLiteral)
    duration_literal: ($) =>
      token(prec(4, seq("@", /-?\d+[yMwdhms]([0-9yMwdhms]|mo)*/))),

    // @sqlite, @postgres, @mysql, @sftp, @shell, @DB
    // lexer.go detectAtLiteralType — connection keywords
    connection_literal: ($) =>
      token(
        prec(
          3,
          seq(
            "@",
            choice("sqlite", "postgres", "mysql", "sftp", "shell", "DB"),
          ),
        ),
      ),

    // @SEARCH, @env, @args, @params
    // lexer.go detectAtLiteralType — builtin globals / context
    context_literal: ($) =>
      token(prec(3, seq("@", choice("SEARCH", "env", "args", "params")))),

    // @std/math, @basil/http, @basil/auth, @std, @basil
    // lexer.go L2787-2802 (readStdlibPath)
    stdlib_import: ($) =>
      token(
        prec(
          3,
          seq(
            "@",
            choice("std", "basil"),
            optional(seq("/", /[a-zA-Z][a-zA-Z0-9_]*/)),
          ),
        ),
      ),

    // @-, @stdin, @stdout, @stderr
    // lexer.go readPathLiteral — stdio detection
    stdio_literal: ($) =>
      token(prec(3, seq("@", choice("-", "stdin", "stdout", "stderr")))),

    // @./file, @../dir, @/usr/local, @~/home, @.config
    // lexer.go L2715-2782 (readPathLiteral)
    path_literal: ($) =>
      token(
        prec(
          1,
          seq(
            "@",
            choice(
              seq(".", /[./]?/, /[^\s<>"{}|\\^`\[\]),:;]*/),
              seq("/", /[^\s<>"{}|\\^`\[\]),:;]*/),
              seq("~/", /[^\s<>"{}|\\^`\[\]),:;]*/),
            ),
          ),
        ),
      ),

    // @https://example.com, @http://..., @ftp://...
    // lexer.go L2841-2874 (readUrlLiteral)
    url_literal: ($) =>
      token(
        prec(
          2,
          seq("@", /(https?|ftp|file|wss?|ssh):\/\/[^\s<>"{}|\\^`\[\]),;]*/),
        ),
      ),

    // @(./path/{expr}), @(./{name}/file)
    // lexer.go L2972-3017 (readPathTemplate)
    path_template: ($) =>
      seq("@(", repeat(choice(/[^{}()]+/, seq("{", $._expression, "}"))), ")"),

    // url_template and datetime_template use the same @(...) syntax
    // but are semantically distinct. Since tree-sitter alias causes conflicts,
    // we fold them into path_template and distinguish in highlights.scm.
    // The _at_literal choice only includes path_template.

    // ================================================================
    // Collections — parser.go L2139-2169, L2988-3079
    // ================================================================

    // parser.go L2139-2169 (parseSquareBracketArrayLiteral)
    // No spread in arrays — parser just calls parseExpression for each element
    array_literal: ($) => seq("[", commaSep($._expression), "]"),

    // parser.go L2988-3079 (parseDictionaryLiteral)
    // No spread, no shorthand properties — always key: value or [key]: value
    dictionary_literal: ($) =>
      prec.dynamic(
        -1,
        seq("{", commaSep(choice($.pair, $.computed_property)), "}"),
      ),

    pair: ($) =>
      seq(
        field("key", choice($.identifier, $.string)),
        ":",
        field("value", $._expression),
      ),

    // parser.go L2988-3079 — computed key: [expr]: value
    computed_property: ($) =>
      seq(
        "[",
        field("key", $._expression),
        "]",
        ":",
        field("value", $._expression),
      ),

    // ================================================================
    // Basic tokens
    // ================================================================

    // lexer.go L1248-1254 (readIdentifier)
    // Supports ASCII identifiers (Unicode handled via external scanner in Phase 2)
    identifier: ($) => /[a-zA-Z_][a-zA-Z0-9_]*/,

    // lexer.go collectTrivia, skipAndCaptureComment
    comment: ($) => token(seq("//", /.*/)),
  },
});

/**
 * Comma-separated list with optional trailing comma
 * Used throughout: arrays, dicts, parameters, arguments
 */
function commaSep(rule) {
  return optional(seq(rule, repeat(seq(",", rule)), optional(",")));
}
