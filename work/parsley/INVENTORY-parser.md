# Parser Inventory

**Source**: `pkg/parsley/parser/parser.go`  
**Extracted**: 2026-01-11

---

## Precedence Table (Lowest to Highest)

| Level | Name | Operators |
|-------|------|-----------|
| 1 | LOWEST | (base) |
| 2 | COMMA_PREC | `,` |
| 3 | LOGIC_OR | `\|`, `\|\|`, `or`, `??` |
| 4 | LOGIC_AND | `&`, `&&`, `and` |
| 5 | EQUALS | `==`, `!=`, `~`, `!~`, `in`, `not in`, `<=?=>`, `<=??=>`, `<=!=>` |
| 6 | LESSGREATER | `<`, `>`, `<=`, `>=` |
| 7 | SUM | `+`, `-`, `..` |
| 8 | CONCAT | `++` |
| 9 | PRODUCT | `*`, `/`, `%` |
| 10 | PREFIX | `-x`, `!x` |
| 11 | INDEX | `[]`, `.` |
| 12 | CALL | `()` |

---

## Statements

### Let Statement
```
let_stmt := "let" pattern "=" expr
pattern  := IDENT | dict_pattern | array_pattern

// Examples
let x = 5
let {a, b} = dict
let [x, y, ...rest] = arr
```

### Assignment Statement
```
assign_stmt := IDENT "=" expr
            | dict_pattern "=" expr

// Examples
x = 5
{a, b} = dict
```

### Index/Property Assignment
```
index_assign := expr "[" expr "]" "=" expr
             | expr "." IDENT "=" expr

// Examples
arr[0] = 5
obj.name = "Alice"
```

### Export Statement
```
export_stmt := "export" let_stmt
            | "export" assign_stmt

// Examples
export let PI = 3.14
export name = "Alice"
```

### Return Statement
```
return_stmt := "return" expr

// Example
return x + y
```

### Check Statement (Guard)
```
check_stmt := "check" expr "else" expr

// Example
check x > 0 else "x must be positive"
```

### Stop/Skip Statements (Loop Control)
```
stop_stmt := "stop"
skip_stmt := "skip"

// Examples
if (x > 10) stop
if (x == 0) skip
```

### Expression Statement
```
expr_stmt := expr

// Examples
log("hello")
x + y
```

### Read Statement
```
read_stmt := pattern "<==" expr

// Examples
let data <== JSON(@./file.json)
let {name, age} <== JSON(@./person.json)
```

### Fetch Statement
```
fetch_stmt := pattern "<=/=" expr

// Examples
let data <=/= JSON(@https://api.example.com/data)
let {result, error} <=/= JSON(@https://api.example.com)
```

### Write Statement
```
write_stmt := expr "==>" expr
           | expr "==>>" expr

// Examples
data ==> JSON(@./output.json)
logLine ==>> text(@./log.txt)
```

---

## Expressions

### Literals

#### Integer & Float
```
int_lit   := DIGIT+
float_lit := DIGIT+ "." DIGIT+

// Examples: 42, 3.14159
```

#### String (No Interpolation)
```
string_lit := '"' chars '"'

// Escapes: \n, \t, \\, \"
// Example: "Hello\nWorld"
```

#### Template String (With Interpolation)
```
template_lit := '`' (chars | "{" expr "}")* '`'

// Example: `Hello, {name}!`
```

#### Raw String (With @{} Interpolation)
```
raw_lit := "'" (chars | "@{" expr "}")* "'"

// Backslashes literal, @{} for interpolation
// Example: 'Parts.refresh("x", {id: @{id}})'
```

#### Boolean
```
bool_lit := "true" | "false"
```

#### Null
```
null_lit := "null"
```

#### Regex
```
regex_lit := "/" pattern "/" flags?
flags     := [imsg]*

// Example: /\d+/g
```

### Path & URL Literals
```
path_lit     := "@" path
url_lit      := "@" scheme "://" url_body
stdlib_lit   := "@std/" module | "@basil/" module
path_template := "@(" path_with_interpolation ")"
url_template  := "@(" url_with_interpolation ")"

// Examples
@./config.json
@~/components/header
@https://api.example.com
@std/math
@basil/http
@(./data/{filename}.json)
@(https://api.com/users/{id})
```

### DateTime Literals
```
datetime_lit := "@" date ["T" time] ["Z"]
date         := YYYY "-" MM "-" DD
time         := HH ":" MM [":" SS]
duration_lit := "@" ["-"] number unit+
datetime_now := "@now" | "@today" | "@dateNow" | "@timeNow"

// Examples
@2024-12-25
@2024-12-25T14:30:00
@14:30
@now
@today
@2h30m
@-1w
```

### Money Literals
```
money_lit := currency_symbol number
          | currency_code "#" number

// Examples: $12.34, £99.99, EUR#50.00
```

### Connection Literals
```
connection_lit := "@sqlite" | "@postgres" | "@mysql"
              | "@sftp" | "@shell" | "@DB" | "@SEARCH"

// Example: let db = SQLITE(@sqlite)
```

### Array Literal
```
array_lit := "[" [expr ("," expr)*] "]"

// Examples
[]
[1, 2, 3]
[a, b, c]
```

### Dictionary Literal
```
dict_lit := "{" [dict_entry ("," dict_entry)*] "}"
dict_entry := IDENT ":" expr
           | STRING ":" expr
           | "[" expr "]" ":" expr  // computed key

// Examples
{}
{name: "Alice", age: 30}
{[dynamicKey]: value}
```

### Function Literal
```
fn_lit  := "fn" "(" params? ")" block
params  := param ("," param)*
param   := IDENT ["=" expr]  // with default value

// Examples
fn() { 42 }
fn(x) { x * 2 }
fn(a, b = 10) { a + b }
fn({name, age}) { ... }  // destructuring param
```

### Tag Literals (HTML/XML)
```
tag_singleton := "<" tag_name attrs "/>"
tag_pair      := "<" tag_name attrs ">" contents "</" tag_name ">"
attrs         := (IDENT ["=" attr_value] | "..." IDENT)*
attr_value    := STRING | "{" expr "}"
contents      := (tag | expr | string)*

// Examples
<br/>
<img src="photo.jpg" alt={altText}/>
<div class="container" ...props>
    <h1>"Title"</h1>
    {content}
</div>
```

**Tag Content Rules:**
- Text must be quoted: `<p>"Hello"</p>`
- Expressions don't need braces: `<p>name</p>` (variable)
- But expressions CAN use braces: `<p>{name}</p>`
- Spread attributes: `<div ...props/>`

### If Expression
```
if_expr := "if" ["("] condition [")"] block ["else" (block | if_expr)]

// Parentheses optional but recommended
// Examples
if (x > 0) { "positive" }
if x > 0 { "positive" } else { "non-positive" }
if (x > 0) "positive" else "negative"  // single expression
```

### For Expression
```
for_expr := "for" ["("] binding "in" iterable [")"] block
         | "for" "(" iterable ")" fn
binding  := IDENT | IDENT "," IDENT

// Returns array of results (null values filtered out)
// Examples
for (x in arr) { x * 2 }
for (i, x in arr) { `{i}: {x}` }
for (key, val in dict) { ... }
for x in 1..10 { x * x }
```

### Try Expression
```
try_expr := "try" expr

// Returns {result, error}
// Example
let {result, error} = try riskyOperation()
```

### Import Expression
```
import_expr := "import" path ["as" IDENT]
            | "let" pattern "=" "import" path

// Examples
import @std/math
import @std/math as M
let {floor, ceil} = import @std/math
```

### Check Expression (Inline Guard)
```
check_expr := "check" condition "else" value

// Returns null if condition true, else returns value
// Example
check x > 0 else fail("must be positive")
```

### Range Expression
```
range_expr := expr ".." expr

// Creates array from start to end (inclusive)
// Example: 1..5 → [1, 2, 3, 4, 5]
```

### Infix Expressions
```
infix_expr := expr op expr

// All binary operators
+, -, *, /, %
==, !=, <, >, <=, >=
&&, ||, &, |
~, !~
in, not in
??
..
++
<=?=>, <=??=>, <=!=>
<=#=>
```

### Prefix Expressions
```
prefix_expr := op expr

// Unary operators
!expr    // logical not
-expr    // negation
```

### Index Expression
```
index_expr := expr "[" expr "]"

// Examples
arr[0]
dict["key"]
str[5]
```

### Slice Expression
```
slice_expr := expr "[" [start] ":" [end] "]"

// Examples
arr[1:3]   // elements 1 to 2
arr[:3]    // first 3 elements
arr[2:]    // from index 2 to end
str[0:5]   // first 5 characters
```

### Dot Expression
```
dot_expr := expr "." IDENT

// Examples
obj.name
arr.length()
str.toUpper()
```

### Call Expression
```
call_expr := expr "(" [args] ")"
args      := expr ("," expr)*

// Examples
log("hello")
arr.map(fn(x) { x * 2 })
Math.floor(3.7)
```

### Grouped Expression
```
grouped_expr := "(" expr ")"

// Example: (a + b) * c
```

---

## Destructuring Patterns

### Array Destructuring
```
array_pattern := "[" IDENT ("," IDENT)* ["," "..." IDENT] "]"

// Examples
let [a, b] = [1, 2]
let [first, ...rest] = arr
let [x, y, z] = coords
```

### Dictionary Destructuring
```
dict_pattern := "{" dict_key ("," dict_key)* ["," "..." IDENT] "}"
dict_key     := IDENT ["as" IDENT] [":" nested_pattern]

// Examples
let {name, age} = person
let {name as n, age as a} = person
let {user: {name}} = data  // nested
let {a, b, ...rest} = dict
```

---

## Query DSL

### Schema Declaration
```
schema_decl := "@schema" IDENT "{" schema_fields "}"
schema_field := IDENT ":" type [type_options]
type         := "text" | "int" | "real" | "bool" | "datetime" | "money" | "json"
             | "enum" "(" enum_values ")"
             | IDENT  // reference to another schema
type_options := "(" option ("," option)* ")"
option       := IDENT "=" expr

// Example
@schema User {
    id: int(primary, auto)
    name: text(required, minLen=1)
    email: text(unique)
    role: enum("admin", "user")
    profile: Profile  // relation
}
```

### Query Expression
```
query_expr := "@query" table_ref [conditions] [modifiers] terminal

table_ref  := IDENT ["as" IDENT]
conditions := condition+
condition  := field_ref op value
            | "(" condition_group ")"
modifiers  := order_by | limit | offset | group_by
terminal   := "??->" | "?->" | ".->"

// Examples
@query users name == "Alice" ??->
@query users age > 18 orderBy age desc limit 10 ??->
@query users (role == "admin" or role == "mod") ?->
```

### Insert Expression
```
insert_expr := "@insert" table_ref "{" field_writes "}" terminal
field_write := field_name "|<" expr

// Example
@insert users {
    name |< userName
    email |< userEmail
} ?->
```

### Update Expression
```
update_expr := "@update" table_ref conditions "{" field_writes "}" terminal

// Example
@update users id == userId {
    name |< newName
} .->
```

### Delete Expression
```
delete_expr := "@delete" table_ref conditions terminal

// Example
@delete users id == userId .->
```

### Transaction Expression
```
transaction_expr := "@transaction" block

// Example
@transaction {
    @insert accounts {...} ?->
    @update balances {...} .->
}
```

---

## Special Syntax

### Spread in Tags
```
<div ...props/>
<input type="text" ...attrs/>
```

### Spread in Dictionaries
```
{...baseConfig, name: "custom"}
```

### Rest in Destructuring
```
let [first, ...rest] = arr
let {a, ...others} = dict
```

### Default Parameters
```
fn(x, y = 10) { x + y }
```

### Computed Dictionary Keys
```
{[dynamicKey]: value}
```

### Null Coalescing
```
value ?? defaultValue
```

---

## Parser Notes

1. **Pratt Parser**: Uses precedence climbing for operator parsing
2. **Backtracking**: Limited backtracking for ambiguous syntax (e.g., `{` could be dict or block)
3. **Optional Semicolons**: Semicolons are optional statement terminators
4. **Optional Parentheses**: `if` and `for` conditions allow optional parentheses
5. **Tag Parsing**: Special lexer mode for tag content in `<style>` and `<script>`
6. **Error Recovery**: Only first error kept to avoid cascading noise
