; Keywords
[
  "let"
  "fn"
  "function"
  "if"
  "else"
  "for"
  "in"
  "return"
  "export"
  "import"
  "try"
  "check"
  "as"
  "computed"
] @keyword

[
  "and"
  "or"
  "not"
  "is"
] @keyword.operator

; Literals
(number) @number
(money) @number
(boolean) @constant.builtin

; Strings
(string) @string
(template_string) @string
(raw_string) @string
(escape_sequence) @string.escape

; Regex
(regex) @string.regexp

; At-literals
(datetime_literal) @number
(time_now_literal) @constant.builtin
(duration_literal) @number
(connection_literal) @function.builtin
(context_literal) @variable.builtin
(stdlib_import) @module
(stdio_literal) @constant.builtin
(path_literal) @string.special.path
(url_literal) @string.special.url
(path_template) @string.special.path

; Schema declarations
(schema_declaration
  "@schema" @keyword
  name: (identifier) @type)

; Schema fields
(schema_field
  name: (identifier) @property
  type: (identifier) @type)

; Schema type options: id(auto: true)
(type_option
  name: (identifier) @property)

; Arithmetic operators
[
  "+"
  "-"
  "*"
  "/"
  "%"
] @operator

; Comparison operators
[
  "=="
  "!="
  "<"
  ">"
  "<="
  ">="
] @operator

; Regex match operators
[
  "~"
  "!~"
] @operator

; Other operators
[
  "++"
  "??"
  ".."
  "="
  "!"
] @operator

; File I/O operators
[
  "<=="
  "==>"
  "==>>"
  "<=/="
  "=/=>"
  "=/=>>"
] @operator

; Database operators
[
  "<=?=>"
  "<=??=>"
  "<=!=>"
  "<=#=>"
] @operator

; Logical operators
[
  "&&"
  "||"
  "&"
  "|"
] @operator



; Spread/rest
"..." @operator

; Punctuation - brackets
[
  "("
  ")"
  "["
  "]"
  "{"
  "}"
] @punctuation.bracket

; Punctuation - delimiters
[
  ","
  ":"
  "."
] @punctuation.delimiter

; Tags
(tag_name) @tag
(tag_start) @tag
(attribute_name) @attribute

; Tag punctuation
(self_closing_tag "/>" @punctuation.bracket)
(open_tag ">" @punctuation.bracket)
(close_tag ["</" ">"] @punctuation.bracket)

; Style/script tags
(style_tag) @tag
(script_tag) @tag

; Functions
(function_expression ["fn" "function"] @keyword.function)

(call_expression
  function: (identifier) @function.call)

(call_expression
  function: (member_expression
    property: (identifier) @function.method.call))

; Variable declarations
(let_statement
  pattern: (identifier) @variable)

(export_statement
  name: (identifier) @variable)

; Parameters
(parameter_list
  (identifier) @variable.parameter)

; For loop variable
(for_expression
  variable: (identifier) @variable)

; Import alias
(import_expression
  alias: (identifier) @variable)

; Interpolation
(interpolation ["{" "}"] @punctuation.special)
(raw_interpolation "}" @punctuation.special)

; ==================== Query DSL ====================

; Query expression keyword
(query_expression "@query" @function.builtin)

; Query source table
(query_source
  table: (identifier) @type)

; Query source alias
(query_source
  alias: (identifier) @variable)

; Query field references
(query_field_ref
  (identifier) @property)

; Query condition operators
(query_operator) @operator

; Query null check (is null / is not null)
(query_null_check) @keyword.operator

; Query interpolation braces
(query_interpolation ["{" "}"] @punctuation.special)

; Query modifiers keywords
(query_order_modifier "order" @keyword)
(query_limit_modifier "limit" @keyword)
(query_offset_modifier "offset" @keyword)
(query_with_modifier "with" @keyword)

; Query order direction
(query_order_field ["asc" "desc"] @keyword)

; Query group by
(query_group_by ["+" "by"] @keyword)

; Query terminal operators
(query_terminal ["?->" "??->" "?!->" "??!->" ".->" "."] @operator)

; Query projection star
(query_projection "*" @constant.builtin)
(query_projection "toSQL" @function.builtin)

; Query computed field name
(query_computed_field
  name: (identifier) @property)

; Query aggregate functions
(query_aggregate ["count" "sum" "avg" "min" "max"] @function.builtin)

; Query condition keywords
(query_condition ["not" "!"] @keyword.operator)
(query_condition "between" @keyword.operator)
(query_condition_group ["not" "!" "and" "or"] @keyword.operator)

; Mutation expressions
(mutation_expression ["@insert" "@update" "@delete" "@transaction"] @function.builtin)

; Comments
(comment) @comment

; Identifiers (lowest priority - catch-all)
(identifier) @variable
