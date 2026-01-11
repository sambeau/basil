## Create Parsley 'Cheat Sheet'

**tl;dr**: Write yourself a small help file / primer to quickly bring yourself up-to-date about Parsley's grammar and quirks—especially compared to Javascript and Python (and to a lesser degree to Rust and Go).  I will also find it useful.

- This document will be saved and refered to when writing tests and examples script during development. It ewill also be a handy guide when working on Plans and design documents.

- The two main tripping points you run up against are that Parsley uses 'log()' instead of 'print'. When debugging a multi-line script it is recommended to use logLine() as it prints out the line number.

- There are also major dirrerences with how 'for' and 'if' work, as Parslef is concatentative - you can assign to for, as-if it's ?: from other languages, and 'for' returns an array of values, so it is more akin to 'map' in other languages.

- Unlike JSX, Parsley does not use brackets to differentiate HTML from code. So not ``<div>{ a + 1 }</div>`` but ``<div>a + 1 </div>``

- Strings in HTML must be in quotes. ``<h3>"🌿Basil and 🦁Parsley FAQ."</h3>`` or <h3>`🌿Basil and 🦁Parsley FAQ.`</h3> — not ``<h3>🌿Basil and 🦁Parsley FAQ.</h3>``

- Functions are treated more like arrow functions, however, their syntax is fn(){...}

## Standard Patterns

Parsley's main use-case is generating HTML from data. This is a very standard pattern

```
{Page} = import @~/components/page/page.pars

let faq <== YAML(@./faq.yaml)

<Page title="🌿🦁 Herbaceous">

	<h3>`🌿Basil and 🦁Parsley FAQ.`</h3>

	for (qa in faq){
		<section>
			<h4>qa.Q</h4>
			<ul>
				<li>markdown(qa.A).html</li>
			</ul>
		</section>
	}
	
</Page>
```

Here is a very similar pattern:

```
	<details name="toc">
		<summary>"Table of contents"</summary>
		<ul>
			for (faq in faqs) {
				<li><a href={"#"+faq.name}>faq.title</a></li>
			}
		</ul>
	</details>
```

Here's an example of a simple module that exports <FancyTime time={time}/>. Note the function format, strings needing "", paths being @./foo.bar, interpolation in a tag's properties using foo={}

```
let clockIcon = (@./clock.svg).public()
export FancyTime = fn({time}){
	<div>"FANCY" {time}</div>
	<img src={clockIcon}/>
}
```


The Most Used Parsley Features (Tests & Examples Analysis) are:-

## Top Functions & Keywords

### 1. Core Output/Logging
- `log()` - 523 uses in examples, 710 in tests - **Most used function**
- `logLine()` - 259 uses

### 2. Control Flow
- `let` - 382 uses - **Primary variable declaration**
- `if` - 72 uses
- `for` - 44 uses

### 3. File I/O Factories
- `file()` - 46 uses (tests) + 8 (examples)
- `JSON()` - 20 uses
- `dir()` - 23 uses (tests) + 15 (examples)
- `text()` - 27 uses (tests) + 7 (examples)
- `SFTP()` - 6 uses

### 4. String & Collection Functions
- `len()` - 189 uses (tests) + 14 (examples)
- `split()` - 23 uses (tests) + 6 (examples)
- `sort()` - 7 uses
- `map()` - 22 uses (tests) + 1 (example)

### 5. DateTime
- `time()` - 159 uses (tests) + 22 (examples)
- `now()` - 28 uses (tests) + 8 (examples)

### 6. Operators
- File operators (`==>`, `<==`, `==>>`) - 50 uses in examples
- Network operators (`=/=>`, `<=/=`, `=/=>>`) for SFTP/HTTP

### 7. HTML/XML Tags
- `<p>` - 10 uses
- `<div>` - 6 uses
- Custom elements supported

### 8. Path Literals
- URL literals (`@https://...`) - Heavy usage for API testing
- File path literals (`@/path/to/file`)

### 9. Other Common Functions
- `replace()` - 4 uses
- `filter()` - 2 uses
- `import()` - 5 uses (module system)

## Parsely Summary
The data shows Parsley's tests heavily use **logging/output**, **file I/O**, **string/collection manipulation**, and **HTML templating** in the examples.

## Basil Server Cheatsheet

When developing and testing Basil needs to run in ``dev`` mode to turn HTTPS and caching off.

```
 ./basil -dev --config path/to/basil.yaml
 ```

 ## Working Code

 Please read and use the code in the link ``setup`` (/Users/samphillips/Dev/setup) which has some examples of working code using current basil grammar.

 see ``setup/site/``, ``setup/components/``, ``setup/parts/`` in particular.

