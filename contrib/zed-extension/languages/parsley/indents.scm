; Indentation rules for Parsley

; Indent within these constructs
(array_literal) @indent
(dictionary_literal) @indent
(function_expression) @indent
(tag_expression) @indent
(block) @indent
(for_expression) @indent
(if_expression) @indent
(try_expression) @indent

; Query DSL indentation
(query_expression) @indent
(query_body) @indent
(query_condition_group) @indent

; End markers for dedent
("]" @end)
("}" @end)
(")" @end)
