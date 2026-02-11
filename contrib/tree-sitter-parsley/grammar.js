/**
 * @file Tree-sitter grammar for Parsley
 * @author Basil Contributors
 * @license MIT
 */

/// <reference types="tree-sitter-cli/dsl" />
// @ts-check

const PREC = {
  ASSIGN: 1,
  NULLISH: 2,
  OR: 3,
  AND: 4,
  COMPARE: 5,
  REGEX_MATCH: 6,
  RANGE: 7,
  ADD: 8,
  MULT: 9,
  CONCAT: 10,
  UNARY: 11,
  CALL: 12,
  MEMBER: 13,
};

module.exports = grammar({
  name: "parsley",

  extras: ($) => [/\s/, $.comment],

  conflicts: ($) => [
    [$._primary_expression, $._pattern],
    [$.dictionary_literal, $.dictionary_pattern],
    [$.array_literal, $.array_pattern],
    [$.dictionary_pattern, $._primary_expression],
    [$.array_pattern, $._primary_expression],
    [$.binary_expression, $.tag_expression],
  ],

  word: ($) => $.identifier,

  rules: {
    source_file: ($) => repeat($._statement),

    // ==================== STATEMENTS ====================

    _statement: ($) =>
      choice(
        $.let_statement,
        $.export_statement,
        $.return_statement,
        $.check_statement,
        $.expression_statement,
      ),

    let_statement: ($) =>
      seq(
        "let",
        field("pattern", $._pattern),
        "=",
        field("value", $._expression),
      ),

    export_statement: ($) =>
      seq(
        "export",
        optional("computed"),
        field("name", $.identifier),
        "=",
        field("value", $._expression),
      ),

    return_statement: ($) => prec.right(seq("return", optional($._expression))),

    check_statement: ($) =>
      prec.right(
        seq(
          "check",
          field("condition", $._expression),
          optional(field("body", $.block)),
        ),
      ),

    expression_statement: ($) => $._expression,

    block: ($) => seq("{", repeat($._statement), "}"),

    // ==================== PATTERNS ====================

    _pattern: ($) =>
      choice($.identifier, $.array_pattern, $.dictionary_pattern, "_"),

    array_pattern: ($) =>
      seq(
        "[",
        commaSep(choice($._pattern, seq("...", optional($.identifier)))),
        "]",
      ),

    dictionary_pattern: ($) =>
      seq(
        "{",
        commaSep(
          choice(
            $.identifier,
            seq(field("key", $.identifier), ":", field("value", $._pattern)),
            seq("...", optional($.identifier)),
          ),
        ),
        "}",
      ),

    // ==================== EXPRESSIONS ====================

    _expression: ($) =>
      choice(
        $._primary_expression,
        $.unary_expression,
        $.binary_expression,
        $.ternary_expression,
        $.assignment_expression,
        $.call_expression,
        $.index_expression,
        $.member_expression,
        $.function_expression,
        $.for_expression,
        $.if_expression,
        $.try_expression,
        $.import_expression,
        $.tag_expression,
        $.parenthesized_expression,
      ),

    _primary_expression: ($) =>
      choice($.identifier, $._literal, $.array_literal, $.dictionary_literal),

    parenthesized_expression: ($) => seq("(", $._expression, ")"),

    unary_expression: ($) =>
      prec.right(
        PREC.UNARY,
        seq(
          field("operator", choice("-", "!", "not")),
          field("operand", $._expression),
        ),
      ),

    binary_expression: ($) =>
      choice(
        // Nullish coalescing (right-associative)
        prec.right(
          PREC.NULLISH,
          seq(
            field("left", $._expression),
            field("operator", "??"),
            field("right", $._expression),
          ),
        ),
        // Logical OR
        prec.left(
          PREC.OR,
          seq(
            field("left", $._expression),
            field("operator", choice("or", "||")),
            field("right", $._expression),
          ),
        ),
        // Logical AND
        prec.left(
          PREC.AND,
          seq(
            field("left", $._expression),
            field("operator", choice("and", "&&")),
            field("right", $._expression),
          ),
        ),
        // Comparison
        prec.left(
          PREC.COMPARE,
          seq(
            field("left", $._expression),
            field("operator", choice("==", "!=", "<", ">", "<=", ">=")),
            field("right", $._expression),
          ),
        ),
        // Regex match
        prec.left(
          PREC.REGEX_MATCH,
          seq(
            field("left", $._expression),
            field("operator", choice("~", "!~")),
            field("right", $._expression),
          ),
        ),
        // Range
        prec.left(
          PREC.RANGE,
          seq(
            field("left", $._expression),
            field("operator", ".."),
            field("right", $._expression),
          ),
        ),
        // Addition/Subtraction
        prec.left(
          PREC.ADD,
          seq(
            field("left", $._expression),
            field("operator", choice("+", "-")),
            field("right", $._expression),
          ),
        ),
        // Multiplication/Division
        prec.left(
          PREC.MULT,
          seq(
            field("left", $._expression),
            field("operator", choice("*", "/", "%")),
            field("right", $._expression),
          ),
        ),
        // Concatenation
        prec.left(
          PREC.CONCAT,
          seq(
            field("left", $._expression),
            field("operator", "++"),
            field("right", $._expression),
          ),
        ),
        // File I/O operators
        prec.left(
          PREC.COMPARE,
          seq(
            field("left", $._expression),
            field(
              "operator",
              choice("<==", "<=/=", "==>", "==>>", "=/=>", "=/=>>"),
            ),
            field("right", $._expression),
          ),
        ),
        // Database operators
        prec.left(
          PREC.COMPARE,
          seq(
            field("left", $._expression),
            field("operator", choice("<=?=>", "<=??=>", "<=!=>", "<=#=>")),
            field("right", $._expression),
          ),
        ),
        // Query DSL operators
        prec.left(
          PREC.COMPARE,
          seq(
            field("left", $._expression),
            field(
              "operator",
              choice("|>", "|<", "?->", "??->", "?!->", "??!->", ".->", "<-"),
            ),
            field("right", $._expression),
          ),
        ),
      ),

    ternary_expression: ($) =>
      prec.right(
        PREC.NULLISH,
        seq(
          field("condition", $._expression),
          "?",
          field("consequence", $._expression),
          ":",
          field("alternative", $._expression),
        ),
      ),

    assignment_expression: ($) =>
      prec.right(
        PREC.ASSIGN,
        seq(
          field(
            "left",
            choice($.identifier, $.member_expression, $.index_expression),
          ),
          "=",
          field("right", $._expression),
        ),
      ),

    call_expression: ($) =>
      prec(
        PREC.CALL,
        seq(field("function", $._expression), field("arguments", $.arguments)),
      ),

    arguments: ($) =>
      seq("(", commaSep(choice($._expression, $.spread_element)), ")"),

    spread_element: ($) => seq("...", $._expression),

    index_expression: ($) =>
      prec(
        PREC.MEMBER,
        seq(
          field("object", $._expression),
          "[",
          choice(
            // Slice: arr[start:end]
            seq(
              field("start", optional($._expression)),
              ":",
              field("end", optional($._expression)),
            ),
            // Regular index: arr[index]
            field("index", $._expression),
          ),
          "]",
        ),
      ),

    member_expression: ($) =>
      prec(
        PREC.MEMBER,
        seq(
          field("object", $._expression),
          ".",
          field("property", $.identifier),
        ),
      ),

    function_expression: ($) =>
      seq(
        choice("fn", "function"),
        field("parameters", $.parameter_list),
        field("body", $._expression),
      ),

    parameter_list: ($) =>
      seq(
        "(",
        commaSep(
          choice(
            $.identifier,
            seq($.identifier, "=", $._expression), // default value
            seq("...", $.identifier), // rest parameter
          ),
        ),
        ")",
      ),

    for_expression: ($) =>
      prec.right(
        seq(
          "for",
          choice(
            // for item in collection { body }
            seq(
              field("pattern", $._pattern),
              "in",
              field("iterable", $._expression),
              field("body", $.block),
            ),
            // for collection (shorthand)
            seq(field("iterable", $._expression)),
          ),
        ),
      ),

    if_expression: ($) =>
      seq(
        "if",
        field("condition", $._expression),
        field("consequence", $.block),
        optional(
          seq("else", field("alternative", choice($.block, $.if_expression))),
        ),
      ),

    try_expression: ($) => seq("try", $._expression),

    import_expression: ($) =>
      prec.right(
        seq(
          "import",
          field("source", $._expression),
          optional(seq("as", field("alias", $.identifier))),
        ),
      ),

    // ==================== LITERALS ====================

    _literal: ($) =>
      choice(
        $.number,
        $.string,
        $.template_string,
        $.raw_string,
        $.regex,
        $.boolean,
        $.null,
        $.money,
        $._at_literal,
      ),

    number: ($) => /\d+(\.\d+)?/,

    boolean: ($) => choice("true", "false"),

    null: ($) => "null",

    string: ($) =>
      seq(
        '"',
        repeat(choice($.escape_sequence, $.interpolation, /[^"\\{]+/)),
        '"',
      ),

    template_string: ($) =>
      seq(
        "`",
        repeat(choice($.escape_sequence, $.interpolation, /[^`\\{]+/)),
        "`",
      ),

    raw_string: ($) =>
      seq(
        "'",
        repeat(
          choice($.escape_sequence, $.raw_interpolation, $._raw_string_content),
        ),
        "'",
      ),

    // Match non-special characters, or @ followed by non-{ character
    _raw_string_content: ($) => token(prec(-1, /([^'\\@]|@[^{'])+/)),

    escape_sequence: ($) => /\\./,

    interpolation: ($) => seq("{", $._expression, "}"),

    raw_interpolation: ($) => seq("@{", $._expression, "}"),

    regex: ($) => token(seq("/", /[^/\n]+/, "/", optional(/[gimsuvy]+/))),

    money: ($) => /([$£€¥]|[A-Z]{3}#)\d+(\.\d{1,2})?/,

    // ==================== AT-LITERALS ====================

    _at_literal: ($) =>
      choice(
        $.datetime_literal,
        $.time_now_literal,
        $.duration_literal,
        $.connection_literal,
        $.schema_literal,
        $.table_literal,
        $.query_literal,
        $.context_literal,
        $.stdlib_import,
        $.path_literal,
        $.url_literal,
        $.path_template,
        $.stdio_literal,
      ),

    // @2024-01-15, @2024-01-15T10:30:00Z, @12:30:00
    datetime_literal: ($) =>
      token(
        seq(
          "@",
          choice(
            // Full datetime
            seq(
              /\d{4}-\d{2}-\d{2}/,
              optional(
                seq(
                  "T",
                  /\d{2}:\d{2}(:\d{2})?/,
                  optional(/(\.\d+)?(Z|[+-]\d{2}:\d{2})?/),
                ),
              ),
            ),
            // Time only
            /\d{1,2}:\d{2}(:\d{2})?/,
          ),
        ),
      ),

    // @now, @today, @timeNow, @dateNow
    time_now_literal: ($) =>
      token(seq("@", choice("now", "today", "timeNow", "dateNow"))),

    // @2h30m, @-7d, @1y6mo
    duration_literal: ($) =>
      token(seq("@", /-?\d+[yMwdhms]([0-9yMwdhms]|mo)*/)),

    // @sqlite, @postgres, @mysql, @sftp, @shell, @DB
    connection_literal: ($) =>
      token(
        seq("@", choice("sqlite", "postgres", "mysql", "sftp", "shell", "DB")),
      ),

    // @schema
    schema_literal: ($) => token("@schema"),

    // @table
    table_literal: ($) => token("@table"),

    // @query, @insert, @update, @delete, @transaction
    query_literal: ($) =>
      token(
        seq("@", choice("query", "insert", "update", "delete", "transaction")),
      ),

    // @SEARCH, @env, @args, @params
    context_literal: ($) =>
      token(seq("@", choice("SEARCH", "env", "args", "params"))),

    // @std, @std/module, @basil, @basil/http, @basil/auth
    stdlib_import: ($) =>
      token(
        seq(
          "@",
          choice(
            seq("std", optional(seq("/", /[a-zA-Z]+/))),
            seq("basil", optional(seq("/", /[a-zA-Z]+/))),
          ),
        ),
      ),

    // @-, @stdin, @stdout, @stderr
    stdio_literal: ($) =>
      token(seq("@", choice("-", "stdin", "stdout", "stderr"))),

    // @./file, @../dir, @/usr/local, @~/home, @.config
    path_literal: ($) =>
      token(
        seq(
          "@",
          choice(
            seq(".", optional(seq(/\.?\//, /[^\s<>"{}|\\^`\[\]]*/))),
            seq("/", /[^\s<>"{}|\\^`\[\]]*/),
            seq("~/", /[^\s<>"{}|\\^`\[\]]*/),
          ),
        ),
      ),

    // @https://..., @http://..., @ftp://..., @file://...
    url_literal: ($) =>
      token(seq("@", /(https?|ftp|file):\/\/[^\s<>"{}|\\^`\[\]]*/)),

    // @(./path/{expr}) or @(https://api.com/{expr})
    path_template: ($) =>
      seq("@(", repeat(choice(/[^{}()]+/, $.interpolation)), ")"),

    // ==================== ARRAYS AND DICTIONARIES ====================

    array_literal: ($) =>
      seq("[", commaSep(choice($._expression, $.spread_element)), "]"),

    dictionary_literal: ($) =>
      seq(
        "{",
        commaSep(
          choice(
            $.pair,
            $.shorthand_property,
            $.spread_element,
            $.computed_property,
          ),
        ),
        "}",
      ),

    pair: ($) =>
      seq(
        field("key", choice($.identifier, $.string, $.number)),
        ":",
        field("value", $._expression),
      ),

    shorthand_property: ($) => prec(-1, $.identifier),

    computed_property: ($) =>
      seq(
        "[",
        field("key", $._expression),
        "]",
        ":",
        field("value", $._expression),
      ),

    // ==================== TAGS (JSX-like) ====================

    tag_expression: ($) =>
      prec(
        PREC.MEMBER + 1,
        choice(
          $.self_closing_tag,
          seq($.open_tag, repeat($._tag_child), $.close_tag),
        ),
      ),

    self_closing_tag: ($) =>
      seq("<", field("name", $.tag_name), repeat($.tag_attribute), "/>"),

    open_tag: ($) =>
      seq("<", field("name", $.tag_name), repeat($.tag_attribute), ">"),

    close_tag: ($) => seq("</", field("name", $.tag_name), ">"),

    tag_name: ($) => /[a-zA-Z][a-zA-Z0-9-]*/,

    tag_attribute: ($) =>
      choice(
        // name="value" or name={expr}
        seq(
          field("name", $.attribute_name),
          "=",
          field("value", choice($.string, $.tag_embedded_expression)),
        ),
        // bare attribute
        field("name", $.attribute_name),
        // spread: ...props
        $.tag_spread_attribute,
      ),

    attribute_name: ($) => /[a-zA-Z][a-zA-Z0-9_-]*/,

    tag_embedded_expression: ($) => seq("{", $._expression, "}"),

    tag_spread_attribute: ($) => seq("...", $.identifier),

    _tag_child: ($) =>
      choice($.tag_expression, $.tag_embedded_expression, $.string, $.tag_text),

    tag_text: ($) => /[^<{"]+/,

    // ==================== BASIC TOKENS ====================

    identifier: ($) => /[a-zA-Z_][a-zA-Z0-9_]*/,

    comment: ($) => token(seq("//", /.*/)),
  },
});

/**
 * Helper function for comma-separated lists
 */
function commaSep(rule) {
  return optional(seq(rule, repeat(seq(",", rule)), optional(",")));
}
