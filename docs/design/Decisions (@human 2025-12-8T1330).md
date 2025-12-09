Decisions (@human 2025-12-8T1330):
- remove len() it doesn’t work on pseudo types and it makes little sense on an int or a float (does it work on anything other than an array?
- Type Constructors should stay but we shouldn’t make a big thing of them to encourage people to use @({interpolation}), as having two ways to do something is usually bad UX.
- Consider moving database/SFTP/Command connectors
- Clarify difference between repr() and toDebug()
- Consider removing any global builtin formatter for a type that already has the identical formatter as a method
- If JSON, CSV, MD etc are going to fs then the database connectors should also be a library
	- But maybe all should be global?
- do we want std/fs?
	- just in main namespace?
	- or: CSV(path) -\> std/csv, JSON(path) -\> std/json etc?
	- should we capitalise TEXT, BYTES, FILES?
	- or: csv(path), json(path), yaml(path), svg(path)
	- files should be global
- Regex are fundamental type so keep global
-sqliteDB, postgresDB, mysqlDB, SftpClient, ShellCommand
- Clarify that everything not in std/basil is in Parsley

format() template function - Is this used enough to stay global? It's powerful for format("Hello {name}", {name: "World"}).
- I need to look at format again to decide. I have an item on my list to look at all formatters to make sure we can print() as well as we can produce tags.
match() placement - Currently does URL path matching. Should it move to std/path or std/url, or stay global?

File readers (JSON, CSV, etc.) - These use uppercase by convention. Should they stay as-is for familiarity, or lowercase in std/fs?
- rename to JSONFile() CSVFile(etc.)

RE: Defer stdlib moves to post-alpha (non-breaking, can add modules alongside existing builtins) — Current plan ❌
- No. We need to break things PRE alpha The point
-- The point of this exercise is to create the std/lib structure /global namespace now!
 