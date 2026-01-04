# Query Builder Investigation

**Date:** 2025-01-13  
**Status:** Research/Design Phase  
**Context:** Part 2 of TableBinding API consistency work (Part 1: FEAT-078)

## Design Principles (from user)

1. "Query-building is not the main objective; an interface that fetches data from a database is"
2. "We should not be afraid to add extra computation, or make multiple queries"
3. "Efficiency isn't the main goal—simplicity is"
4. "Composibility and expressivity are always an important goal"
5. "If we could find a simple way to do joins we will have hit the holy grail"

### Note on Magic
**Magic is acceptable for the Query Builder** — the "no magic" principle applies to the basic TableBinding API (FEAT-078), where predictability and transparency are paramount. A Query Builder is a higher-level abstraction where convenience and expressivity can justify some magic, as long as it remains debuggable.

## Fundamental Design Questions

### Are We Building the Wrong Abstraction?

**Problem 1: We're designing a JavaScript API, not a Parsley one**

The proposed syntax (`schema.relation("users", "user_id")`) is a library function call. But we control Parsley's grammar — we could add first-class syntax support:

```parsley
// Current proposal (function-based, JS-style)
let PostSchema = schema.define("Post", {
  id: schema.int(),
  title: schema.string(),
  user_id: schema.int(),
  author: schema.relation("users", "user_id"),
})

// Alternative: Grammar-level support
schema Post {
  id: int
  title: string
  user_id: int -> users.id    // FK syntax built into language
}

// Or even more declarative
model Post from "posts" {
  id: int, primary_key
  title: string
  author: User via user_id    // Relation as first-class concept
}
```

**Problem 2: The binding step becomes meaningless**

If schemas contain database table references, we have:
```parsley
// Schema already knows about "users" table
let PostSchema = schema.define("Post", {
  author: schema.relation("users", "user_id"),  // ← DB table here!
})

// Then we "bind" to a different table?
let Posts = schema.table(PostSchema, db, "posts")
```

This is conceptually confused. The schema is partially bound to the database before the actual binding. What exactly is `schema.table()` doing if the schema already contains `"users"`?

### Three Coherent Models

**Model A: Pure Schema (no DB knowledge)**

Schema only describes shape and validation. All database concepts live in the binding:

> ⚠️ **Note:** There's a variant of Model A worth exploring — see "Model A′: Schema-to-Schema Relations" below.

```parsley
// Schema = pure structure
let PostSchema = schema.define("Post", {
  id: schema.int(),
  title: schema.string(),
  user_id: schema.int(),
  // NO relations here — schema doesn't know about other tables
})

// Binding = adds database behavior AND relations
let Posts = schema.table(PostSchema, db, "posts", {
  relations: {
    author: {table: Users, via: "user_id"},  // Relation defined at binding time
  }
})
```

**Pros:** Clean separation. Schema is just a type. Binding adds all DB concerns.
**Cons:** Relations defined far from field definitions. Must have bindings before defining relations.

---

### Model A′: Schema-to-Schema Relations (Explored)

**The key insight:** Relations are about *data shape*, not database tables. "A Post has an Author" is true regardless of what database tables they're stored in.

```parsley
// Schema relates to SCHEMA, not table
let UserSchema = schema.define("User", {
  id: schema.int(),
  name: schema.string(),
  email: schema.string(),
})

let PostSchema = schema.define("Post", {
  id: schema.int(),
  title: schema.string(),
  user_id: schema.int(),
  author: schema.relation(UserSchema, "user_id"),  // ← References schema!
})
```

**What this means:**
- `author` is data shaped like `UserSchema`
- `user_id` is the FK field in PostSchema
- No database table mentioned — that's the binding's job

**The binding then provides the database mapping:**

```parsley
// Binding maps schemas to tables AND resolves relations
let Users = schema.table(UserSchema, db, "users")
let Posts = schema.table(PostSchema, db, "posts")

// When Posts.all({include: ["author"]}) runs:
// 1. Look at PostSchema.author relation → links to UserSchema
// 2. Find binding for UserSchema → Users, table "users"  
// 3. Execute: SELECT * FROM users WHERE id IN (...)
```

**Why this is conceptually clean:**
1. **Schemas are pure types** — They describe shape and relationships between shapes
2. **No table names in schemas** — Table knowledge lives only in bindings
3. **Same schema, different databases** — PostSchema works whether bound to "posts", "blog_posts", or a different DB
4. **Relations are type-level** — "Post has Author of type User" is a type relationship

**The circular dependency problem:**

```parsley
// This won't work — UserSchema doesn't exist yet
let PostSchema = schema.define("Post", {
  author: schema.relation(UserSchema, "user_id"),  // Error: UserSchema undefined
})

let UserSchema = schema.define("User", {
  posts: schema.relation(PostSchema, "user_id", {many: true}),  // This works
})
```

**Solutions to circular dependencies:**

**Solution 1: Forward declarations** ✅ **PREFERRED**

Verbose but explicit — you declare that a schema exists before you can reference it:

```parsley
// Step 1: Declare schemas exist (creates placeholder)
let UserSchema = schema.declare("User")
let PostSchema = schema.declare("Post")

// Step 2: Define fields and relations (both names now exist)
schema.define(UserSchema, {
  id: schema.int(),
  name: schema.string(),
  email: schema.string(),
  posts: schema.relation(PostSchema, "user_id", {many: true}),
})

schema.define(PostSchema, {
  id: schema.int(),
  title: schema.string(),
  user_id: schema.int(),
  author: schema.relation(UserSchema, "user_id"),
})

// Step 3: Bind to database (unchanged)
let Users = schema.table(UserSchema, db, "users")
let Posts = schema.table(PostSchema, db, "posts")
```

**Why this works well:**
- Two-phase: declare structure, then define content
- No magic resolution — if you forget to declare, you get a clear error
- Relations use actual schema objects, not strings
- IDE can autocomplete and catch typos
- Refactoring-friendly

**For schemas without circular deps, single step still works:**

```parsley
// No forward decl needed — TagSchema has no relations
let TagSchema = schema.define("Tag", {
  id: schema.int(),
  name: schema.string(),
})

// PostTagSchema references TagSchema (already defined)
let PostTagSchema = schema.define("PostTag", {
  post_id: schema.int(),
  tag_id: schema.int(),
  tag: schema.relation(TagSchema, "tag_id"),
})
```

**API shape:**
- `schema.declare(name)` → Returns a schema placeholder that can be used in relations
- `schema.define(placeholder, fields)` → Fills in the declared schema
- `schema.define(name, fields)` → Shorthand for declare + define (when no forward ref needed)

**Solution 2: String-based names (lazy resolution)**
```parsley
let PostSchema = schema.define("Post", {
  id: schema.int(),
  title: schema.string(),
  author: schema.relation("User", "user_id"),  // String = schema name
})

let UserSchema = schema.define("User", {
  id: schema.int(),
  name: schema.string(),
  posts: schema.relation("Post", "user_id", {many: true}),
})

// Resolution happens at query time, not definition time
```

**Solution 3: Post-definition mutation**
```parsley
let UserSchema = schema.define("User", {
  id: schema.int(),
  name: schema.string(),
})

let PostSchema = schema.define("Post", {
  id: schema.int(),
  title: schema.string(),
  user_id: schema.int(),
})

// Add relations after both exist
PostSchema.relation("author", UserSchema, "user_id")
UserSchema.relation("posts", PostSchema, "user_id", {many: true})
```

**Solution 4: Language-level syntax (lazy by nature)**
```parsley
// Grammar change: relations use identifiers, resolved at use-time
schema Post {
  id: int
  title: string
  author: User via user_id    // User resolved when Post is used
}

schema User {
  id: int
  name: string
  posts: [Post] via user_id   // Post resolved when User is used
}
```

**Comparison: Table names vs Schema references**

| Aspect | `schema.relation("users", ...)` | `schema.relation(UserSchema, ...)` |
|--------|--------------------------------|-----------------------------------|
| Schema purity | ❌ Contains DB table name | ✅ Pure type relationship |
| Circular deps | ✅ Strings always work | ❌ Needs solution (see above) |
| Typo safety | ❌ "usres" fails at runtime | ✅ Undefined variable fails early |
| Refactoring | ❌ Rename table = find all strings | ✅ Rename variable = IDE helps |
| Multi-DB | ❌ Assumes one table name | ✅ Different bindings, same schema |

**The binding's new role:**

With schema-to-schema relations, binding becomes the *only* place that knows about databases:

```parsley
// Schema world: pure types
let UserSchema = schema.define("User", {...})
let PostSchema = schema.define("Post", {
  author: schema.relation(UserSchema, "user_id"),
})

// Binding world: maps types to database
let Users = schema.table(UserSchema, db, "users")      // UserSchema → users table
let Posts = schema.table(PostSchema, db, "posts")      // PostSchema → posts table
// Relations auto-resolve: PostSchema.author → UserSchema → Users binding → "users" table
```

**What about tables without schemas?**

If you need to relate to a table that has no Parsley schema:

```parsley
// Option 1: Create a minimal schema just for the relation
let LegacyUserSchema = schema.define("LegacyUser", {
  id: schema.int(),
  // Only fields you need
})

// Option 2: Allow table name fallback in binding
let Posts = schema.table(PostSchema, db, "posts", {
  relations: {
    legacy_author: {table: "legacy_users", via: "legacy_user_id"},  // Raw table
  }
})
```

---

**Model B: Full Model (schema IS the model)**

Schema declares everything, including its table name:

```parsley
// Schema = complete model definition
let Post = schema.model("posts", {  // Table name in schema
  id: schema.int(),
  title: schema.string(),
  user_id: schema.int(),
  author: schema.belongsTo("users", "user_id"),
})

// "Binding" is just connecting to a database
Post.connect(db)
// or
let Posts = Post.using(db)
```

**Pros:** Everything in one place. No separate binding step.
**Cons:** Schema now coupled to database. Can't use same schema for non-DB validation.

**Model C: Language-Level Syntax**

Add grammar support for models as first-class citizens:

```parsley
// New syntax: 'model' keyword
model User from "users" {
  id: int, primary_key
  name: string
  email: string
  
  posts: [Post] via user_id
}

model Post from "posts" {
  id: int, primary_key
  title: string
  user_id: int
  
  author: User via user_id
}

// Usage
let users = User.all(db)
let post = Post.find(db, 1, include: [.author])
```

**Pros:** Clean, declarative, no function-call noise. Relations are syntax, not strings.
**Cons:** Significant language change. Schemas and models become different things.

### Recommendation

Before proceeding with any API design, we need to decide:

1. **Should schemas know about databases at all?** If yes, the binding step is just "connect to DB". If no, relations must live in bindings.

2. **If schemas have relations, should they reference tables or schemas?**
   - **Table names** (`"users"`) — Simple, no circular deps, but pollutes schema with DB knowledge
   - **Schema references** (`UserSchema`) — Pure types, needs forward decl or lazy resolution
   - ✅ **Decision: Schema references with forward declarations** — Verbose but clear

3. **Is this worth a language change?** If relations are important enough, grammar support would be cleaner than `schema.relation(...)`.

4. **What's the relationship between Schema and Model?** Are they the same thing? Is a Model a Schema + DB binding? Or are they completely separate?

---

## Research Summary

### Existing Approaches Reviewed

| Approach | Philosophy | Join Strategy | Parsley Fit |
|----------|-----------|---------------|-------------|
| **ActiveRecord** | ORM with associations | Declared relationships, lazy-load or eager-load | Medium - too magical |
| **Prisma** | Type-safe, declarative | `include`/`select` with nested objects | Medium - good ergonomics |
| **PostgREST** | REST API for Postgres | Query params for embedding | Low - HTTP-centric |
| **Malloy** | Semantic modeling | Convention-based via primary keys | High - simple join syntax |
| **LINQ** | Language-integrated | From...join...select | High - composable |
| **Datalog** | Logic programming | Implicit via shared variables | Medium - powerful but different |
| **Drizzle** | SQL-like, type-safe | Explicit join methods | Medium - mirrors SQL |

### Key Insights

#### From Malloy

- **Primary key declarations** enable convention-over-configuration joins
- `join_one: users with user_id` - the `with` clause names the FK, PK is inferred from target
- Joins are LEFT OUTER by default (return nulls, don't filter)
- Results preserve hierarchy (nested objects) vs SQL's flat rows

#### From LINQ

- **Deferred execution** - query doesn't run until results are needed
- Operators classified as streaming (process one element) vs buffering
- Query expressions (`from x in xs where p select f(x)`) feel natural
- Can mix query syntax with method chaining

#### From Datalog

- Variables unify across predicates: `user(X, Name), order(OrderId, X)` naturally joins on X
- Purely declarative - describe what you want, not how
- Rules can be recursive (e.g., ancestor queries)

#### From Drizzle

- `.leftJoin(posts, eq(posts.userId, users.id))` - explicit but verbose
- Select is composable: `db.select({ ...getTableColumns(users), extra: sql`...` })`
- Subqueries and CTEs are first-class
- Relations can be defined separately for relational queries

### ActiveRecord's Magic (Worth Stealing?)

ActiveRecord is often cited as "too magical" but some of its magic is genuinely useful:

#### 1. Association Declarations

```ruby
class User < ApplicationRecord
  has_many :posts
  has_one :profile
  belongs_to :organization
  has_many :comments, through: :posts
end
```
**What it buys you:**

- `user.posts` - lazy-loads associated records
- `User.includes(:posts)` - eager-loads to avoid N+1
- `user.posts.build(title: "...")` - creates associated record with FK set
- `user.posts.create(...)` - creates and saves in one call

**Worth stealing:** The declaration syntax is clean. The auto-FK inference (`user_id` for `belongs_to :user`) removes boilerplate.

#### 2. Scope Chaining

```ruby
User.active.recent.with_posts.limit(10)
```
Each scope returns a chainable relation object. Scopes compose.

**Worth stealing:** Deferred execution + composition. The query isn't run until you need the data.

#### 3. Dynamic Finders

```ruby
User.find_by_email("alice@example.com")
User.find_by_name_and_role("Alice", "admin")
```
Methods generated from column names.

**Worth stealing:** Maybe. It's convenient but feels too magical for Parsley's style.

#### 4. Automatic Timestamps

```ruby
user.created_at  # Set automatically on create
user.updated_at  # Set automatically on update
```

**Worth stealing:** Yes, but at the schema/migration level, not query builder.

#### 5. Counter Caches

```ruby
class Post < ApplicationRecord
  belongs_to :user, counter_cache: true
end
# user.posts_count is automatically maintained
```

**Worth stealing:** Interesting but complex. Deferred.

### What If Query Builder Wasn't a TableBinding Extension?

If we **decouple the Query Builder from TableBinding entirely**, we gain freedom:

#### Standalone Query Builder Possibilities

```parsley
// Import as separate module
import "@std/query"

// Build queries against any data source
let q = query.new(db)
  .from("users")
  .join("posts", "posts.user_id = users.id")
  .where({role: "admin"})
  .select(["users.name", "posts.title"])
  .orderBy("posts.created_at", "desc")

// Execute
let results = q.all()
let first = q.first()
let count = q.count()
```

#### Benefits of Decoupling

1. **Not limited by Schema** — Can query any table, join ad-hoc
2. **Raw SQL escape hatch** — Mix builder syntax with raw SQL fragments
3. **Cross-database** — Same query syntax, different backends
4. **Subqueries as values** — Pass queries around, compose them
5. **Aggregation pipelines** — Group, aggregate, window functions

#### Powerful Things We Could Add

**A. Query Variables**

```parsley
// Parameterized queries
let userPosts = query.new(db)
  .from("posts")
  .where({user_id: query.param("userId")})

// Later, bind and execute
userPosts.bind({userId: 42}).all()
```

**B. Named Subqueries / CTEs**

```parsley
let recentPosts = query.new(db)
  .from("posts")
  .where({created_at: {$gt: lastWeek}})
  .as("recent")

let q = query.new(db)
  .with(recentPosts)
  .from("recent")
  .join("users", "users.id = recent.user_id")
```

**C. Aggregation Pipelines**

```parsley
query.new(db)
  .from("orders")
  .groupBy("customer_id")
  .select({
    customer_id: "customer_id",
    total: query.sum("amount"),
    count: query.count(),
    avg_order: query.avg("amount")
  })
  .having({total: {$gt: 1000}})
```

**D. Window Functions**

```parsley
query.new(db)
  .from("sales")
  .select({
    date: "date",
    amount: "amount",
    running_total: query.sum("amount").over({orderBy: "date"}),
    rank: query.rowNumber().over({partitionBy: "region", orderBy: "amount desc"})
  })
```

**E. Union / Intersect / Except**

```parsley
let activeUsers = query.from("users").where({active: true})
let recentUsers = query.from("users").where({last_login: {$gt: lastMonth}})

activeUsers.union(recentUsers).all()
activeUsers.intersect(recentUsers).all()
```

#### Simplifications We Could Offer

1. **Auto-pluralization** — `from("user")` finds `users` table
2. **Smart joins** — Infer join conditions from FK naming conventions
3. **Nested results** — Automatic object nesting from joins
4. **Pagination built-in** — `.page(3, perPage: 20)` instead of limit/offset math
5. **Explain mode** — `.explain()` shows the generated SQL without running

## Proposed Design Directions

### Option A: Schema-Aware Joins (Malloy-inspired)

Leverage the Schema's structure to enable simple joins.

#### Current Syntax Proposal (with `_belongs_to`)

```parsley
let PostSchema = schema.new({
  id: schema.int(),
  title: schema.string(),
  user_id: schema.int(),
  _belongs_to: { user: { schema: UserSchema, key: "user_id" } }
})
```

**Problems with this syntax:**
- `_belongs_to` is Rails terminology, may confuse non-Rails users
- Mixing schema fields with relationship metadata feels wrong
- What about `has_many`? `has_one`?

#### Alternative Syntax Explorations

**Alt A1: Separate `relations` declaration**

```parsley
let PostSchema = schema.new({
  id: schema.int(),
  title: schema.string(),
  user_id: schema.int(),
})

// Relations declared separately
schema.relate(PostSchema, {
  author: schema.belongsTo(UserSchema, "user_id"),
})

schema.relate(UserSchema, {
  posts: schema.hasMany(PostSchema, "user_id"),
})
```
*Pros:* Clean separation. *Cons:* Scattered definitions.

**Alt A2: Inline with special field type**

```parsley
let PostSchema = schema.new({
  id: schema.int(),
  title: schema.string(),
  author: schema.ref(UserSchema, "user_id"),  // FK field + relation in one
})
```
*Pros:* Concise, obvious. *Cons:* Conflates storage (user_id) with relation (author).

**Alt A3: Declarative relations block**

```parsley
let PostSchema = schema.define("Post", {
  id: schema.int(),
  title: schema.string(),
  user_id: schema.int(),
}, {
  relations: {
    author: {to: UserSchema, via: "user_id"},
    comments: {to: CommentSchema, via: "post_id", many: true},
  }
})
```
*Pros:* All in one place, explicit direction. *Cons:* Verbose, adds third argument.

**Alt A3b: Relations inline with fields (overloaded)**

```parsley
let PostSchema = schema.define("Post", {
  id: schema.int(),
  title: schema.string(),
  user_id: schema.int(),
  // Relations detected by structure (has "to" and "via" keys)
  author: {to: "users", via: "user_id"},
  comments: {to: "comments", via: "post_id", many: true},
})
```
*Pros:* No extra argument, everything in one dict. *Cons:* Mixing fields and relations—how to distinguish?

**Analysis: Should relations be inline or separate?**

Current `schema.define(name, fields)` takes exactly 2 arguments. Options:

| Approach | Signature | Detection | Tradeoffs |
|----------|-----------|-----------|-----------|
| Third arg (A3) | `define(name, fields, options)` | Explicit `relations:` key | Clear separation, but more verbose |
| Inline (A3b) | `define(name, fields)` | Detect by `{to:, via:}` shape | Concise, but magic detection |
| Prefixed keys | `define(name, fields)` | Keys starting with `@` or `_` | e.g., `@author: {...}` — visible but no extra arg |

**What else might go in an options dict?**

If we add a third argument, what else could it contain?

```parsley
schema.define("Post", fields, {
  relations: {...},           // Relationships to other schemas
  primaryKey: "id",           // Override default PK assumption
  tableName: "blog_posts",    // Override table name for TableBinding
  timestamps: true,           // Auto-add created_at/updated_at
  softDelete: true,           // Add deleted_at for soft deletes
  indexes: [...],             // Hint for migrations/validation
})
```

If we anticipate needing these options, a third argument makes sense. If relations are the *only* metadata we'll ever need, inline is cleaner.

**Recommendation: Inline with explicit marker**

Use a special key prefix (`$` or `@`) to distinguish relations from fields:

```parsley
let PostSchema = schema.define("Post", {
  // Fields
  id: schema.int(),
  title: schema.string(),
  user_id: schema.int(),
  
  // Relations (prefixed with $)
  $author: {to: "users", via: "user_id"},
  $comments: {to: "comments", via: "post_id", many: true},
})
```

**Why this works:**

- No third argument needed (keeps API simple)
- `$` prefix is visually distinct and easy to detect programmatically
- Relations live with fields but are clearly marked
- If we later need options like `primaryKey`, we could add `$primaryKey: "id"` or switch to third arg then

**Alternative: Detection by shape (no prefix)**

```parsley
let PostSchema = schema.define("Post", {
  id: schema.int(),
  title: schema.string(),
  user_id: schema.int(),
  author: schema.relation("users", "user_id"),           // Relation type!
  comments: schema.relation("comments", "post_id", {many: true}),
})
```

This uses a `schema.relation()` type factory (like `schema.int()`), making detection trivial and keeping the pattern consistent. This may be the cleanest approach.

**Alt A4: Convention-based (Malloy style)**

```parsley
// UserSchema has primary_key: "id" (default)
let PostSchema = schema.new({
  id: schema.int(),
  title: schema.string(),
  user_id: schema.int(),  // Convention: *_id is a FK to * table
})

// Join syntax infers the relationship
Posts.all({join: "users"})  // Infers: posts.user_id = users.id
```
*Pros:* Zero config for conventional cases. *Cons:* Magic, breaks with non-standard naming.

**Alt A5: DSL for relations**

```parsley
let Post = schema.model("posts", {
  id: schema.int(),
  title: schema.string(),
  user_id: schema.int(),
})

Post.belongsTo("author", User, "user_id")
User.hasMany("posts", Post, "user_id")
```
*Pros:* Reads naturally, familiar to ORM users. *Cons:* Mutates schema object.

#### Recommended Syntax: `schema.relation()` type factory

The cleanest approach is to add a `schema.relation()` function that works like other type factories:

```parsley
let UserSchema = schema.define("User", {
  id: schema.int(),
  name: schema.string(),
  email: schema.string(),
  posts: schema.relation("posts", "user_id", {many: true}),
  profile: schema.relation("profiles", "user_id"),
})

let PostSchema = schema.define("Post", {
  id: schema.int(),
  title: schema.string(),
  user_id: schema.int(),
  author: schema.relation("users", "user_id"),
  comments: schema.relation("comments", "post_id", {many: true}),
})
```

**Critical question: What does `"users"` refer to?**

| Option | `schema.relation("users", ...)` means | Pros | Cons |
|--------|---------------------------------------|------|------|
| **Table name** | The `users` database table | Simple, no schema needed | Bypasses schema validation |
| **Schema name** | Schema defined as `schema.define("users", ...)` | Type-safe, validates FK | Circular dependency issues |
| **TableBinding name** | A TableBinding variable named `Users` | Links to actual binding | Requires binding to exist first |

**Recommendation: Table name (with optional schema lookup)**

Use the **database table name** as the primary identifier:

```parsley
// "users" = the database table name
author: schema.relation("users", "user_id")
```

**Why table name:**
1. **No circular dependencies** — Don't need to import UserSchema to define PostSchema
2. **Works without schemas** — Can relate to tables that don't have Parsley schemas
3. **Matches TableBinding** — `schema.table(PostSchema, db, "posts")` uses table name
4. **Simple mental model** — It's the database table you're joining to

---

### Aside: Do Schema Names Serve a Purpose?

**Current state:** `schema.define("User", {...})` requires a name, which is stored as `.name` and accessible via `UserSchema.name`, but **never used for anything functional**.

**This is potentially confusing because:**
- Users might expect `"User"` to be used for lookup/registration
- It's not clear what the name is *for*
- It adds a required parameter that does nothing

**Options:**

| Option | Change | Use Name For |
|--------|--------|--------------|
| **Remove name** | `schema.define({...})` | Nothing—remove it |
| **Make optional** | `schema.define({...}, {name: "User"})` | Error messages, debugging |
| **Auto-derive** | Infer from variable name | Introspection |
| **Keep, use for lookup** | Register schemas by name | Relations could use `"User"` |

**If we keep the name, potential uses:**
1. **Error messages**: "Validation failed for User schema: email is required"
2. **Debugging/logging**: Identify which schema in stack traces
3. **Schema registry**: `schema.get("User")` to retrieve by name
4. **Relation targets**: `schema.relation("User", ...)` instead of table name

**Recommendation: Keep name, use for error messages + optional registry**

The name becomes useful if validation errors say:
```
ValidationError in "User" schema: field "email" is required
```

Rather than just:
```
ValidationError: field "email" is required
```

For relations, we should still use **table names** (not schema names) because:
- Not all tables have schemas
- Table name is the actual join target
- Avoids needing a global schema registry for basic functionality

**Future consideration:** If we want `schema.relation("User", ...)` syntax, we could support both:
```parsley
author: schema.relation("users", "user_id"),        // Table name (always works)
author: schema.relation(UserSchema, "user_id"),     // Schema reference (if available)
```

---

**Schema lookup is automatic when available:**

When you create a TableBinding and use `include`, the system can:
1. Look up the table name from the relation (`"users"`)
2. Find a registered TableBinding for that table
3. Use that binding's schema for validation/structure

```parsley
let Users = schema.table(UserSchema, db, "users")  // Registers "users" → UserSchema
let Posts = schema.table(PostSchema, db, "posts")

// When including "author", system:
// 1. Reads relation: {table: "users", foreignKey: "user_id"}
// 2. Finds Users binding (registered for "users" table)
// 3. Uses UserSchema for the nested data
Posts.all({include: ["author"]})
```

**Alternative: Schema name (if we solve circular deps)**

If we used schema names, we'd need lazy/string-based references:

```parsley
// Schema name approach (requires "User" to be resolvable later)
author: schema.relation("User", "user_id")  // Capital = schema name
```

This could work if schemas are registered globally and resolved at query time, but adds complexity.

**Signature clarification:**

```
schema.relation(tableName, foreignKey, options?)
```
- `tableName`: String name of the **database table** (e.g., `"users"`, `"posts"`)
- `foreignKey`: Column holding the FK
- `options`: `{many: true}` for has-many

**Why this is best:**
- Consistent with existing API (`schema.int()`, `schema.string()`, etc.)
- No magic detection needed—it's a known type
- Relations are clearly part of the schema structure
- No third argument, no prefix syntax
- Easy to implement: just another type spec with special handling

**Signature:** `schema.relation(table, foreignKey, options?)`
- `table`: String name of related table
- `foreignKey`: Column that holds the FK (in this table for belongsTo, in related table for hasMany)
- `options`: `{many: true}` for has-many relationships

**Usage:**

```parsley
// Include related data
Posts.where({status: "published"}, {include: ["author"]})
// → [{id: 1, title: "...", user_id: 5, author: {id: 5, name: "Alice"}}]

// Nested includes
Posts.all({include: ["author", "comments"]})

// Deep nesting
Users.all({include: ["posts", "posts.comments"]})
```

**Why this syntax:**
- Uses `schema.relation()` which fits existing type factory pattern
- No need for separate relations block or third argument
- Relations are visible in schema definition
- String table names avoid circular import issues
- `many: true` option is explicit about cardinality

### Option B: Query Object Pattern (LINQ-inspired)

Build queries as data structures, execute on demand:

```parsley
// Query is a data structure, not immediately executed
let q = query.from(Users)
  |> query.where({status: "active"})
  |> query.select(["id", "name"])
  |> query.orderBy("name", "asc")

// Execute when needed
let results = query.run(q)
let count = query.count(q)
let first = query.first(q)

// Compose queries
let baseQuery = query.from(Users).where({role: "member"})
let recentQuery = baseQuery |> query.where({created_at: {$gt: yesterday}})
```

**Pros:**
- Queries are composable and reusable
- Clear separation between building and executing
- Can inspect/transform queries
- Naturally supports caching

**Cons:**
- Different paradigm from TableBinding
- More verbose for simple cases
- Joins still need explicit definition

### Option C: Fluent Multi-Table Syntax (Novel) — Expanded

This approach keeps queries explicit and composable without requiring schema-level relationship declarations.

#### Core Idea
Each TableBinding query can specify nested queries that run for each result row. The nested query can reference values from the parent row.

#### Basic Syntax

```parsley
// Parent query with nested child queries
Users.all({
  with: {
    posts: fn(user) { Posts.where({user_id: user.id}) },
    profile: fn(user) { Profiles.find(user.profile_id) }
  }
})
// Returns:
// [
//   {id: 1, name: "Alice", posts: [{...}, {...}], profile: {...}},
//   {id: 2, name: "Bob", posts: [{...}], profile: {...}},
// ]
```

#### Deep Nesting

```parsley
Users.all({
  with: {
    posts: fn(user) {
      Posts.where({user_id: user.id}, {
        with: {
          comments: fn(post) { Comments.where({post_id: post.id}) }
        }
      })
    }
  }
})
// Returns: users → posts → comments hierarchy
```

#### Aggregations in `with`

```parsley
Users.all({
  with: {
    postCount: fn(user) { Posts.count({user_id: user.id}) },
    recentPost: fn(user) { Posts.first({orderBy: "created_at", order: "desc"}) }
  }
})
// Returns:
// [{id: 1, name: "Alice", postCount: 5, recentPost: {...}}, ...]
```

#### Conditional Includes

```parsley
Users.all({
  with: {
    // Only fetch posts for active users
    posts: fn(user) {
      if user.active {
        Posts.where({user_id: user.id})
      } else {
        []
      }
    }
  }
})
```

#### Computed/Derived Fields

```parsley
Users.all({
  with: {
    fullName: fn(user) { user.first_name + " " + user.last_name },
    isAdmin: fn(user) { user.role == "admin" }
  }
})
```

#### Performance: The N+1 Problem

Naive implementation runs one query per row per `with` clause. Solutions:

**Option C1: Accept it (for small datasets)**
```parsley
// Fine for 10-50 rows
Users.all({limit: 20, with: {posts: fn(u) { Posts.where({user_id: u.id}) }}})
```

**Option C2: Batch mode**
```parsley
// Hint to batch queries
Users.all({
  with: {
    posts: fn(user) { Posts.where({user_id: user.id}) }
  },
  batch: true  // Collects all user.ids, does single WHERE user_id IN (...)
})
```

**Option C3: Explicit batching function**
```parsley
// User controls the batching
Users.all({
  with: {
    posts: batch(fn(users) {
      let ids = users.map(fn(u) { u.id })
      Posts.where({user_id: {$in: ids}})
        .groupBy(fn(p) { p.user_id })  // Returns {user_id: [posts...]}
    })
  }
})
```

#### Compared to Option A (Schema Relations)

| Aspect | Option A (Schema) | Option C (Fluent) |
|--------|-------------------|-------------------|
| Setup required | Declare relations in schema | None |
| Syntax | `{include: ["author"]}` | `{with: {author: fn(p) {...}}}` |
| Flexibility | Limited to declared relations | Any computation |
| Discoverability | Schema documents relations | Must read code |
| Reusability | Relation defined once | Must repeat or extract function |

#### Hybrid Approach

Allow both — schema relations for common cases, fluent syntax for ad-hoc:

```parsley
// Schema-declared relation
Posts.all({include: ["author"]})

// Ad-hoc relation for this query only
Posts.all({
  include: ["author"],  // Uses schema
  with: {
    relatedPosts: fn(post) {
      Posts.where({category: post.category, id: {$ne: post.id}}, {limit: 3})
    }
  }
})
```

### Option D: Relation Pipelines (Functional)

Treat relationships as lazy collections that can be piped:

```parsley
// Define relations as functions
let userPosts = fn(user) { Posts.where({user_id: user.id}) }
let postComments = fn(post) { Comments.where({post_id: post.id}) }

// Compose with pipes
Users.find(1)
  |> with_related("posts", userPosts)
  |> with_related("posts.comments", postComments)

// Bulk fetch
Users.all({limit: 10})
  |> with_related("posts", userPosts)  // Batches into single query
```

**Pros:**
- Relations are regular functions - testable, composable
- Works with Parsley's pipe operator
- Explicit data flow

**Cons:**
- Requires `with_related` helper
- Still need to define relation functions
- Less discoverable than schema-defined relations

### Option E: Natural Language-ish DSL

Optimize for readability over SQL mapping:

```parsley
// Very English-like
Users
  |> named("Alice")
  |> with_their("posts")
  |> sorted_by("created_at", "newest first")
  |> limited_to(10)

// Behind the scenes, these map to queries:
// - named("Alice") → WHERE name = 'Alice'
// - with_their("posts") → LEFT JOIN or subquery
// - sorted_by(..., "newest first") → ORDER BY ... DESC
// - limited_to(10) → LIMIT 10
```

**Pros:**
- Very readable
- Hides SQL complexity entirely
- Domain-specific language potential

**Cons:**
- Limited by predefined vocabulary
- May be frustrating when you need something not covered
- Loss of control

## Recommendation

Based on the design principles, I recommend **a combination of Option A and C**:

### Phase 1: Schema-Aware Relationships
Add relationship declarations to schemas:

```parsley
let PostSchema = schema.new({
  // ... fields ...
  _relations: {
    author: { table: "users", key: "user_id" }
  }
})
```

### Phase 2: Simple Include Syntax  
Enable pulling related data:
```parsley
Posts.where({status: "published"}, {
  include: ["author"]  // Uses declared relation
})
```

### Phase 3: Nested Query Fallback
For complex/ad-hoc joins:

```parsley
Posts.all({
  with: {
    stats: fn(post) { 
      // Use Parsley's database operators (<=?=> for single row)
      db <=?=> "SELECT count(*) as count FROM comments WHERE post_id = {post.id}"
    }
  }
})
```

## Open Questions — With Recommendations

### 1. Should relationships be in Schema or TableBinding?

**Options:**
- **Schema:** More declarative, single source of truth, documented structure
- **TableBinding:** More flexible, doesn't pollute schema, can vary by use-case

**Trade-offs:**

| Schema-level | TableBinding-level |
|--------------|-------------------|
| ✅ Relationships visible in schema definition | ✅ Same schema, different relationships per binding |
| ✅ Validates FK columns exist | ✅ Can relate tables without shared schema |
| ❌ Schema becomes heavier | ✅ Schema stays focused on validation |
| ❌ Circular reference issues (User→Post→User) | ❌ Relationships scattered across codebase |

**Recommendation: Schema, with lazy resolution**

Put relations in Schema, but use string table names (not schema references) to avoid circular dependencies:

```parsley
let UserSchema = schema.new({
  id: schema.int(),
  name: schema.string(),
}, {
  relations: {
    posts: {table: "posts", foreignKey: "user_id", many: true},
  }
})
```

The TableBinding resolves `"posts"` to the actual table at query time. This keeps schemas self-contained while avoiding import cycles.

---

### 2. How to handle nested includes?

**Options:**
- **Dot notation:** `include: ["author", "author.profile", "comments.author"]`
- **Nested objects:** `include: {author: {include: ["profile"]}, comments: {include: ["author"]}}`
- **Depth limit:** Only allow 1 level, require explicit nested queries for deeper

**Trade-offs:**

| Dot notation | Nested objects | Depth limit |
|--------------|----------------|-------------|
| ✅ Concise | ✅ More control (can add where/limit per level) | ✅ Simple, predictable |
| ❌ Ambiguous for options | ✅ Explicit structure | ❌ Frustrating for common cases |
| ❌ Hard to add per-relation options | ❌ Verbose | ❌ Pushes complexity to user |

**Recommendation: Dot notation + options object hybrid**

```parsley
// Simple case: dot notation
Posts.all({include: ["author", "author.profile", "comments"]})

// With options: object form
Posts.all({
  include: {
    author: true,  // Simple include
    comments: {
      where: {approved: true},
      orderBy: "created_at",
      limit: 10,
      include: ["author"]  // Nested include
    }
  }
})
```

Array form is shorthand for `{relationName: true}`. Object form allows per-relation options.

---

### 3. One query (JOIN) vs multiple queries?

**Options:**
- **Single JOIN:** One SQL query, faster network round-trip, but flattens results
- **Multiple queries:** N+1 pattern, preserves nesting, simpler result handling
- **Batched queries:** Collect IDs, single `WHERE IN (...)`, then stitch results
- **User choice:** Let user pick via option

**Trade-offs:**

| Single JOIN | Multiple queries | Batched |
|-------------|------------------|---------|
| ✅ One round-trip | ❌ N+1 problem | ✅ Constant queries (1 + N relations) |
| ❌ Flat results need reshaping | ✅ Natural nesting | ✅ Natural nesting |
| ❌ Duplicated parent data | ✅ No duplication | ✅ No duplication |
| ❌ Complex for has-many | ✅ Simple for has-many | ✅ Simple for has-many |

**Recommendation: Batched by default, with escape hatches**

```parsley
// Default: batched (good balance)
Posts.all({include: ["author", "comments"]})
// Executes:
//   1. SELECT * FROM posts
//   2. SELECT * FROM users WHERE id IN (collected author_ids)
//   3. SELECT * FROM comments WHERE post_id IN (collected post_ids)
// Then stitches results together

// Opt-in: single JOIN (for specific optimization needs)
Posts.all({include: ["author"], strategy: "join"})

// Opt-in: per-row queries (for very small datasets or complex logic)
Posts.all({include: ["author"], strategy: "lazy"})
```

Default to batched because:
- Constant number of queries (predictable)
- Results stay nested (natural to work with)
- No duplicate data in results
- Works well for both has-one and has-many

---

### 4. How does this interact with FEAT-078?

The `include` option should compose with FEAT-078's `orderBy`, `select`, `limit`, `offset` options.

**Behavior:**

```parsley
Posts.where({status: "published"}, {
  orderBy: "created_at",
  order: "desc",
  limit: 10,
  select: ["id", "title", "user_id"],  // Must include FK for relations!
  include: ["author"]
})
```

**Rules:**
1. `orderBy/limit/offset` apply to the **root query** only
2. `select` on root must include FK columns needed for includes (or we auto-add them)
3. Each included relation can have its own options (see nested objects form above)

**Recommendation:** Auto-add FK columns if `select` omits them but `include` needs them. Emit a warning in development mode:

```
Warning: 'select' omitted 'user_id' but 'include' needs it for 'author'. Auto-added.
```

## Next Steps

1. **Decide on relationship declaration syntax** - Recommendation: Alt A3 (relations in schema options)
2. **Decide on include syntax** - Recommendation: Array shorthand + object form for options
3. **Decide on query strategy** - Recommendation: Batched by default
4. **Prototype Option A** - Schema relations with include
5. **Prototype Option C** - Fluent `with` for ad-hoc relations
6. **Consider standalone Query Builder** - Separate from TableBinding for power users
7. **Create FEAT-079** - Formalize chosen approach

## Summary of Recommendations

**⚠️ BLOCKING QUESTION: See "Fundamental Design Questions" section above**

Before finalizing any API, we must decide:
1. Should schemas contain database knowledge (table names, relations)?
2. If yes, what's the point of a separate binding step?
3. Is this important enough to warrant language syntax changes?

The table below assumes we proceed with the library-function approach (Models in Schema, binding = connect to DB), but this may not be the right path.

| Decision | Recommendation | Rationale |
|----------|---------------|-----------|
| Relations location | In schema, inline with fields | Single source of truth, no extra arguments |
| Relation syntax | `schema.relation(table, fk, opts)` | Consistent with `schema.int()` etc. |
| Nested includes | Dot notation + object form | Concise for simple, powerful for complex |
| Query strategy | Batched by default | Best balance of performance and simplicity |
| Ad-hoc relations | `with:` option using functions | Flexible, explicit, composable |
| Standalone query builder | Worth exploring | Decoupling enables more power |

## Appendix: Detailed Research Notes

### Malloy Join Syntax

```malloy
source: flights is duckdb.table('flights') extend {
  primary_key: id
  join_one: carriers with carrier_id
  join_one: destinations is airports with destination_code
}
```
- `with` specifies the FK in this table
- Target table's `primary_key` is used automatically
- Can alias joins: `destinations is airports`

### LINQ Query Syntax

```csharp
// Query expression
from user in users
where user.Age > 18
orderby user.Name
select new { user.Name, user.Email }

// Method syntax (equivalent)
users
  .Where(u => u.Age > 18)
  .OrderBy(u => u.Name)
  .Select(u => new { u.Name, u.Email })
```
- Deferred execution until iteration/materialization
- `FirstOrDefault()`, `Count()`, `ToList()` trigger execution

### Datalog Variable Unification

```datalog
// These two facts share variable X, creating a natural join
user(X, Name, Email).
order(OrderId, X, Amount).

// Query: find user names with their order amounts
?- user(X, Name, _), order(_, X, Amount).
```
- No explicit JOIN keyword
- Variables with same name must unify (match)
- Very declarative but different mental model
