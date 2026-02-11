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
] @keyword.operator

; Literals
(number) @number
(money) @number
(boolean) @constant.builtin
(null) @constant.builtin

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
(schema_literal) @type
(table_literal) @type
(query_literal) @function.builtin
(context_literal) @variable.builtin
(stdlib_import) @module
(stdio_literal) @constant.builtin
(path_literal) @string.special.path
(url_literal) @string.special.url
(path_template) @string.special.path

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

; Query DSL operators
[
  "|>"
  "|<"
  "?->"
  "??->"
  "?!->"
  "??!->"
  ".->"
  "<-"
] @operator

; Logical operators
[
  "&&"
  "||"
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

; Ternary
[
  "?"
  ":"
] @operator

; Tags
(tag_name) @tag
(attribute_name) @attribute
(tag_text) @string

; Tag punctuation
(self_closing_tag ["<" "/>"] @punctuation.bracket)
(open_tag ["<" ">"] @punctuation.bracket)
(close_tag ["</" ">"] @punctuation.bracket)

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
  pattern: (identifier) @variable)

; Import alias
(import_expression
  alias: (identifier) @variable)

; Interpolation
(interpolation ["{" "}"] @punctuation.special)
(raw_interpolation ["@{" "}"] @punctuation.special)
(tag_embedded_expression ["{" "}"] @punctuation.special)

; Comments
(comment) @comment

; Identifiers (lowest priority - catch-all)
(identifier) @variable
