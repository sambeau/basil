package repl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/peterh/liner"
	"github.com/sambeau/basil/pkg/parsley/errors"
	"github.com/sambeau/basil/pkg/parsley/evaluator"
	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
)

const PROMPT = ">> "
const PROMPT_RAW = ":> "
const CONTINUATION_PROMPT = ".. "

const PARSER_LOGO = `
█▀█ ▄▀█ █▀█ █▀ █░░ █▀▀ █▄█
█▀▀ █▀█ █▀▄ ▄█ █▄▄ ██▄ ░█░ `

// Parsley keywords and builtins for tab completion
var completionWords = []string{
	// Keywords
	"let", "if", "else", "for", "in", "fn", "return", "export", "import", "try",
	// Builtins - I/O
	"log", "logLine", "file", "dir", "JSON", "CSV", "MD", "SVG", "HTML",
	"text", "lines", "bytes", "SFTP", "Fetch", "SQL",
	// Builtins - Collections
	"len", "keys", "values", "type", "sort", "reverse", "join",
	// Builtins - Strings
	"split", "trim", "upper", "lower", "contains", "startsWith", "endsWith",
	"replace", "match", "test",
	// Builtins - Math
	"abs", "floor", "ceil", "round", "sqrt", "pow", "sin", "cos", "tan",
	"min", "max", "sum",
	// Builtins - DateTime
	"now", "date", "time", "duration", "format", "parse",
	// Builtins - Error Handling
	"fail",
	// Builtins - Other
	"range", "glob", "toString",
	// Common values
	"true", "false", "null",
}

// Start starts the REPL with line editing, history, and tab completion
func Start(in io.Reader, out io.Writer, version string) {
	line := liner.NewLiner()
	defer line.Close()

	// Enable Ctrl+C to abort current line
	line.SetCtrlCAborts(true)

	// Set up tab completion
	line.SetCompleter(func(line string) []string {
		return filterCompletions(line)
	})

	// Load command history from file
	historyFile := filepath.Join(os.TempDir(), ".parsley_history")
	if f, err := os.Open(historyFile); err == nil {
		line.ReadHistory(f)
		f.Close()
	}

	// Save history on exit
	defer func() {
		if f, err := os.Create(historyFile); err == nil {
			line.WriteHistory(f)
			f.Close()
		}
	}()

	env := evaluator.NewEnvironment()
	// Set up permissive security policy for REPL (allow reads and writes by default)
	env.Security = &evaluator.SecurityPolicy{
		AllowWriteAll:   true,  // Allow writes in REPL
		AllowExecuteAll: false, // Disallow executes for security
	}

	fmt.Fprintf(out, "%s", PARSER_LOGO)
	fmt.Fprintln(out, "v", version)
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Type 'exit' or Ctrl+D to quit")
	fmt.Fprintln(out, "Use Tab for completion, ↑↓ for history")
	fmt.Fprintln(out, "Type ':help' for REPL commands")
	fmt.Fprintln(out, "")

	var inputBuffer strings.Builder
	rawMode := false // When true, output is like running a .pars script
	basePrompt := PROMPT

	for {
		currentPrompt := basePrompt
		if inputBuffer.Len() > 0 {
			currentPrompt = CONTINUATION_PROMPT
		}
		input, err := line.Prompt(currentPrompt)
		if err != nil {
			// Ctrl+D or Ctrl+C
			if err == liner.ErrPromptAborted {
				// Ctrl+C - clear any buffered input and return to main prompt
				if inputBuffer.Len() > 0 {
					fmt.Fprintln(out, "^C (cleared)")
				} else {
					fmt.Fprintln(out, "^C")
				}
				inputBuffer.Reset()
				continue
			}
			if err == io.EOF {
				// Ctrl+D - exit
				fmt.Fprintln(out, "\nGoodbye!")
				return
			}
			fmt.Fprintf(out, "Error reading input: %v\n", err)
			continue
		}

		// Check for exit command
		trimmed := strings.TrimSpace(input)
		if inputBuffer.Len() == 0 && (trimmed == "exit" || trimmed == "quit") {
			fmt.Fprintln(out, "Goodbye!")
			return
		}

		// Handle REPL commands (start with :)
		if inputBuffer.Len() == 0 && strings.HasPrefix(trimmed, ":") {
			newRawMode, handled := handleReplCommand(trimmed, env, out, rawMode)
			if handled {
				rawMode = newRawMode
				if rawMode {
					basePrompt = PROMPT_RAW
				} else {
					basePrompt = PROMPT
				}
			}
			continue
		}

		// Skip empty lines when no input buffered
		if inputBuffer.Len() == 0 && trimmed == "" {
			continue
		}

		// Add to input buffer
		if inputBuffer.Len() > 0 {
			inputBuffer.WriteString("\n")
		}
		inputBuffer.WriteString(input)

		// Check if input is complete (no unclosed braces/brackets)
		fullInput := inputBuffer.String()
		if needsMoreInput(fullInput) {
			// Continue multi-line input
			continue
		}

		// Input is complete - parse and evaluate

		// Add complete input to history
		if trimmed != "" {
			line.AppendHistory(fullInput)
		}

		// Parse and evaluate
		l := lexer.New(fullInput)
		p := parser.New(l)
		program := p.ParseProgram()

		if errs := p.StructuredErrors(); len(errs) != 0 {
			printStructuredErrors(out, errs)
			inputBuffer.Reset()
			continue
		}

		evaluated := evaluator.Eval(program, env)
		if evaluated != nil {
			// Check if it's an error
			if errObj, ok := evaluated.(*evaluator.Error); ok {
				printRuntimeError(out, errObj)
			} else if evaluated.Type() == evaluator.NULL_OBJ {
				if !rawMode {
					io.WriteString(out, "OK\n")
				}
			} else {
				if rawMode {
					// Raw mode: output like running a .pars script
					result := evaluator.ObjectToPrintString(evaluated)
					if result != "" {
						io.WriteString(out, result)
						// Add newline if the output doesn't end with one
						if !strings.HasSuffix(result, "\n") {
							io.WriteString(out, "\n")
						}
					}
				} else {
					// Normal mode: pretty-printed Parsley literal output
					result := evaluator.ObjectToFormattedReprString(evaluated)
					if result != "" {
						io.WriteString(out, result)
						io.WriteString(out, "\n")
					} else {
						io.WriteString(out, "OK\n")
					}
				}
			}
		}

		// Clear buffer for next input
		inputBuffer.Reset()
	}
}

// handleReplCommand handles REPL meta-commands that start with ':'
// Returns (newRawMode, handled) - if handled is true, the command was recognized
func handleReplCommand(cmd string, env *evaluator.Environment, out io.Writer, rawMode bool) (bool, bool) {
	switch cmd {
	case ":help", ":h", ":?":
		fmt.Fprintln(out, "REPL Commands:")
		fmt.Fprintln(out, "  :help, :h, :?   Show this help")
		fmt.Fprintln(out, "  :env            Show variables in scope")
		fmt.Fprintln(out, "  :clear          Clear all user variables")
		fmt.Fprintln(out, "  :raw            Toggle raw output mode (script-style output)")
		fmt.Fprintln(out, "  exit, quit      Exit the REPL")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Output Modes:")
		fmt.Fprintln(out, "  >> (normal)     Shows Parsley literals (strings quoted, etc.)")
		fmt.Fprintln(out, "  :> (raw)        Shows output like running a .pars script")
		return rawMode, true

	case ":env":
		printEnvironment(env, out)
		return rawMode, true

	case ":clear":
		// Create a fresh environment
		*env = *evaluator.NewEnvironment()
		fmt.Fprintln(out, "Environment cleared")
		return rawMode, true

	case ":raw":
		newMode := !rawMode
		if newMode {
			fmt.Fprintln(out, "Raw output mode ON (script-style output)")
		} else {
			fmt.Fprintln(out, "Raw output mode OFF (Parsley literal output)")
		}
		return newMode, true

	default:
		fmt.Fprintf(out, "Unknown command: %s (type :help for commands)\n", cmd)
		return rawMode, true
	}
}

// printEnvironment displays all user-defined variables in the environment
func printEnvironment(env *evaluator.Environment, out io.Writer) {
	vars := env.UserVariables()
	if len(vars) == 0 {
		fmt.Fprintln(out, "(no user variables)")
		return
	}

	// Sort for consistent output
	names := make([]string, 0, len(vars))
	for name := range vars {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		obj := vars[name]
		typeStr := string(obj.Type())
		value := obj.Inspect()

		// For multi-line values, indent continuation lines by 2 spaces
		if strings.Contains(value, "\n") {
			lines := strings.Split(value, "\n")
			for i := 1; i < len(lines); i++ {
				lines[i] = "  " + lines[i]
			}
			value = strings.Join(lines, "\n")
		} else if len(value) > 60 {
			// Truncate long single-line values
			value = value[:57] + "..."
		}

		fmt.Fprintf(out, "  %s: %s = %s\n", name, typeStr, value)
	}
}

// filterCompletions returns completion suggestions based on current input
func filterCompletions(line string) []string {
	// Don't complete if line is empty or only whitespace
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	// Don't complete if line ends with whitespace (including tabs from pasting)
	if len(line) > 0 && (line[len(line)-1] == ' ' || line[len(line)-1] == '\t') {
		return nil
	}

	// Get the last word being typed
	words := strings.Fields(line)
	if len(words) == 0 {
		return nil
	}

	lastWord := words[len(words)-1]

	var matches []string
	for _, word := range completionWords {
		if strings.HasPrefix(word, lastWord) {
			matches = append(matches, word)
		}
	}
	return matches
}

// needsMoreInput checks if the input has unclosed braces, brackets, parentheses, or tags
func needsMoreInput(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return false
	}

	braceCount := 0
	bracketCount := 0
	parenCount := 0
	tagCount := 0
	inString := false
	escapeNext := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if escapeNext {
			escapeNext = false
			continue
		}

		if ch == '\\' {
			escapeNext = true
			continue
		}

		// Track string state to ignore braces inside strings
		if ch == '"' && (i == 0 || input[i-1] != '\\') {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch ch {
		case '{':
			braceCount++
		case '}':
			braceCount--
		case '[':
			bracketCount++
		case ']':
			bracketCount--
		case '(':
			parenCount++
		case ')':
			parenCount--
		case '<':
			// Check for tags: <tag or </tag (not comparison operators)
			if i+1 < len(input) {
				next := input[i+1]
				if next == '/' {
					// Closing tag </tag>
					if i+2 < len(input) && isTagNameStart(input[i+2]) {
						tagCount--
					}
				} else if isTagNameStart(next) {
					// Opening tag <tag> - but check for self-closing later
					// Find end of tag to check for />
					tagEnd := findTagEnd(input, i)
					if tagEnd > i && tagEnd >= 2 && input[tagEnd-1] == '/' {
						// Self-closing tag <tag/>, don't increment
					} else {
						tagCount++
					}
				}
			}
		}
	}

	// Need more input if any are unclosed
	return braceCount > 0 || bracketCount > 0 || parenCount > 0 || tagCount > 0
}

// isTagNameStart checks if a character can start a tag name (letter or underscore)
func isTagNameStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

// findTagEnd finds the position of the closing '>' for a tag starting at pos
func findTagEnd(input string, pos int) int {
	inQuote := false
	quoteChar := byte(0)
	for i := pos + 1; i < len(input); i++ {
		ch := input[i]
		if inQuote {
			if ch == quoteChar {
				inQuote = false
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			inQuote = true
			quoteChar = ch
			continue
		}
		if ch == '>' {
			return i
		}
	}
	return -1 // Tag not closed yet
}

// printStructuredErrors prints parser errors using structured error format
func printStructuredErrors(out io.Writer, errs []*errors.ParsleyError) {
	for _, err := range errs {
		io.WriteString(out, err.PrettyString())
		io.WriteString(out, "\n")
	}
}

// printRuntimeError prints a runtime error with structured formatting
func printRuntimeError(out io.Writer, err *evaluator.Error) {
	io.WriteString(out, "Runtime error")

	// Location info
	if err.Line > 0 {
		fmt.Fprintf(out, ": line %d, column %d\n  %s\n", err.Line, err.Column, err.Message)
	} else {
		io.WriteString(out, "\n  "+err.Message+"\n")
	}

	// Hints
	for _, hint := range err.Hints {
		io.WriteString(out, "  hint: "+hint+"\n")
	}
}
