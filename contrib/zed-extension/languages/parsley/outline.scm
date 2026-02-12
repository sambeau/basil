; Outline queries for Parsley code navigation
; Show exports and top-level functions

; Export statements
(export_statement
  name: (identifier) @name) @item

; Functions defined with direct assignment: name = fn(...)
(assignment_statement
  left: (identifier) @name
  right: (function_expression)) @item
