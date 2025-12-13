/*
Language: Parsley
Description: Parsley language syntax highlighting for highlight.js
Author: Basil Contributors
Website: https://github.com/sambeau/basil
Category: web
*/

function hljsDefineParsley ( hljs ) {
	const IDENT_RE = '[a-zA-Z_][a-zA-Z0-9_]*'

	// Note: highlight.js uses $pattern to match keyword candidates
	// Keywords must be exact matches against the $pattern
	const KEYWORDS = {
		$pattern: /\b[a-zA-Z_][a-zA-Z0-9_]*\b/,
		keyword: [
			'fn',
			'function',
			'let',
			'for',
			'in',
			'as',
			'if',
			'else',
			'return',
			'export',
			'try',
			'import'
			// Note: 'and', 'or', 'not' are NOT keywords - they are operators
		],
		literal: [
			'true',
			'false',
			'null',
			'OK' // NULL display value
		]
	}

	// At-literals (@-prefixed values)
	const AT_LITERAL = {
		scope: 'symbol',
		variants: [
			// DateTime literals: @2024-12-25T14:30:00Z, @now, @today, @timeNow, @dateNow
			{
				match: /@\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z|[+-]\d{2}:\d{2})?)?/
			},
			{
				match: /@(now|today|timeNow|dateNow)\b/
			},
			// Duration literals: @2h30m, @7d, @1y6mo
			{
				match: /@-?\d+[yMwdhms]([0-9yMwdhms]|mo)*/
			},
			// Database connection literals
			{
				match: /@(sqlite|postgres|mysql|DB)\b/
			},
			// Other literals
			{
				match: /@(sftp|shell)\b/
			},
			// Standard streams
			{
				match: /@(stdin|stdout|stderr|-)\b/
			},
			// Stdlib imports: @std/module
			{
				match: /@std\/[a-z]+\b/
			},
			// URL literals: @https://example.com (must come before path literals)
			{
				match: /@https?:\/\/[^\s<>"{}|\\^`\[\]]*/
			},
			// Path literals: @./config, @../config, @/usr/local, @~/home
			{
				match: /@\.\.?\/[^\s<>"{}|\\^`\[\]]*/
			},
			{
				match: /@\/[^\s<>"{}|\\^`\[\]]*/
			},
			{
				match: /@~\/[^\s<>"{}|\\^`\[\]]*/
			},
			// Templated at-literals: @(expr)
			{
				begin: /@\(/,
				end: /\)/
				// Contains expressions - will be populated later
			}
		]
	}

	// Money literals: $12.34, £99.99, EUR#50.00
	const MONEY = {
		scope: 'number',
		variants: [
			{
				match: /[$£€¥][\d,]+\.?\d*/
			},
			{
				match: /[A-Z]{3}#[\d,]+\.?\d*/
			}
		],
		relevance: 5
	}

	// Operators - must use word boundaries for text operators
	const OPERATORS = {
		scope: 'operator',
		variants: [
			// File I/O operators
			{ match: /<==|<=\/=|==>|==>>/ },
			// Database operators
			{ match: /<=\?=>|<=\?\?=>|<=!=>/ },
			// Process execution
			{ match: /<=#=>/ },
			// Comparison operators
			{ match: /<=|>=|==|!=|!~|&&|\|\|/ },
			// Text operators with word boundaries (must not match inside identifiers)
			{ match: /\band\b/ },
			{ match: /\bor\b/ },
			{ match: /\bnot\b/ },
			// Nullish coalescing
			{ match: /\?\?/ },
			// Concatenation
			{ match: /\+\+/ },
			// Range operators
			{ match: /\.\.\./ },
			{ match: /\.\./ },
			// Basic operators
			{ match: /[+\-*\/%<>=!&|~?]/ },
			// Dot accessor (last so it doesn't interfere)
			{ match: /\./ }
		]
	}

	// Template strings with ${expr} interpolation
	const TEMPLATE_STRING = {
		scope: 'string',
		begin: '`',
		end: '`',
		contains: [
			hljs.BACKSLASH_ESCAPE,
			{
				scope: 'subst',
				begin: /\${/,
				end: /}/,
				keywords: KEYWORDS,
				contains: [] // Will be populated below
			}
		]
	}

	// Regular strings
	const STRING = {
		scope: 'string',
		variants: [
			hljs.QUOTE_STRING_MODE,
			hljs.APOS_STRING_MODE
		]
	}

	// Regex literals: /pattern/flags
	const REGEX = {
		scope: 'regexp',
		begin: /\//,
		end: /\/[gimsuvy]*/,
		contains: [
			hljs.BACKSLASH_ESCAPE,
			{
				begin: /\[/,
				end: /\]/,
				contains: [ hljs.BACKSLASH_ESCAPE ]
			}
		]
	}

	// Numbers (integer and float)
	const NUMBER = {
		scope: 'number',
		variants: [
			{ match: /\b\d+\.\d+\b/ },
			{ match: /\b\d+\b/ }
		]
	}

	// HTML/XML tags (Parsley's JSX-like syntax)
	const TAG = {
		begin: /<[A-Z][a-zA-Z0-9.]*/,
		end: />/,
		scope: 'tag',
		contains: [
			{
				scope: 'name',
				begin: /<[A-Z][a-zA-Z0-9.]*/,
				end: /(?=\s|>|\/)/
			},
			{
				scope: 'attr',
				match: /[a-zA-Z_:][a-zA-Z0-9_:.-]*/
			},
			{
				begin: /=/,
				contains: [
					STRING,
					{
						// Attribute expression: prop={expr}
						begin: /{/,
						end: /}/
						// Contains expressions - will be populated later
					}
				]
			}
		]
	}

	const LOWERCASE_TAG = {
		begin: /<[a-z][a-z0-9-]*/,
		end: />/,
		scope: 'tag',
		contains: [
			{
				scope: 'name',
				begin: /<[a-z][a-z0-9-]*/,
				end: /(?=\s|>|\/)/
			},
			{
				scope: 'attr',
				match: /[a-zA-Z_:][a-zA-Z0-9_:.-]*/
			},
			{
				begin: /=/,
				contains: [
					STRING,
					{
						// Attribute expression: prop={expr}
						begin: /{/,
						end: /}/
						// Contains expressions - will be populated later
					}
				]
			}
		]
	}

	// Closing tags
	const TAG_CLOSE = {
		scope: 'tag',
		begin: /<\/[A-Za-z][A-Za-z0-9.-]*/,
		end: />/,
		contains: [
			{
				scope: 'name',
				begin: /<\/[A-Za-z][A-Za-z0-9.-]*/,
				end: /(?=>)/
			}
		]
	}

	// Function definitions
	const FUNCTION_DEF = {
		scope: 'function',
		begin: /\b(fn|function)\s*\(/,
		end: /\)/,
		keywords: KEYWORDS,
		contains: [
			{
				scope: 'params',
				begin: /\(/,
				end: /\)/,
				contains: [
					{
						scope: 'variable',
						match: IDENT_RE
					}
				]
			}
		]
	}

	// Comments
	const COMMENT = hljs.COMMENT( '//', '$' )

	// Define all modes that can appear in expressions
	const EXPRESSION_MODES = [
		COMMENT,
		AT_LITERAL,
		MONEY,
		TEMPLATE_STRING,
		STRING,
		NUMBER,
		OPERATORS
	]

	// Populate template string substitution
	TEMPLATE_STRING.contains[ 1 ].contains = EXPRESSION_MODES

	return {
		name: 'Parsley',
		aliases: [ 'pars' ],
		case_insensitive: false,
		keywords: KEYWORDS,
		contains: [
			COMMENT,
			AT_LITERAL,
			MONEY,
			TEMPLATE_STRING,
			STRING,
			REGEX,
			NUMBER,
			TAG,
			LOWERCASE_TAG,
			TAG_CLOSE,
			FUNCTION_DEF,
			OPERATORS,
			{
				// Destructuring and object patterns
				scope: 'variable',
				match: /\{[a-zA-Z_][a-zA-Z0-9_,\s]*\}/
			},
			{
				// Built-in functions
				scope: 'built_in',
				match: /\b(print|println|len|keys|values|type|inspect|describe|money|tag|toString|text|json|csv|sql|markdown)\b/
			}
		]
	}
}

// Export for different module systems
if ( typeof module !== 'undefined' && module.exports ) {
	module.exports = hljsDefineParsley
}
if ( typeof exports !== 'undefined' ) {
	exports.default = hljsDefineParsley
}
if ( typeof window !== 'undefined' ) {
	window.hljsDefineParsley = hljsDefineParsley
}
