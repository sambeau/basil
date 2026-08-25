package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sambeau/basil/pkg/parsley/format"
)

// runFmtCommand implements `basil fmt`: format Parsley (.pars) source using the
// shared pkg/parsley/format pipeline (the exact same code path as `pars fmt` —
// it never shells out to the pars binary). Unlike `pars fmt`, arguments may be
// directories, which are walked for *.pars files, and with no arguments it walks
// the current directory tree.
func runFmtCommand(args []string, stdout, stderr io.Writer, getenv func(string) string) error {
	flags := flag.NewFlagSet("basil fmt", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	write := flags.Bool("w", false, "Write result to source file instead of stdout")
	list := flags.Bool("l", false, "List files whose formatting differs (exits non-zero if any)")
	diff := flags.Bool("d", false, "Display diffs instead of rewriting files")

	if err := flags.Parse(args); err != nil {
		printFmtUsage(stderr)
		return &usageError{err: err}
	}

	paths := flags.Args()

	// Collect the set of files to format. Explicit file arguments are taken as
	// given; directories (and the implicit cwd when no path is supplied) are
	// walked for *.pars files, skipping .git.
	targets, err := collectFmtTargets(paths)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		// Nothing to do (e.g. an empty tree). A clean no-op, not an error.
		return nil
	}

	// Decide the default behaviour when no mode flag is given. For a single
	// explicit file, print to stdout exactly like `pars fmt <file>`. For a
	// directory, multiple files, or a bare `basil fmt`, printing whole trees to
	// stdout is noise, so default to -l (list unformatted, CI-usable exit).
	stdoutSingle := !*write && !*list && !*diff && singleFileArg(paths)
	listMode := *list || (!*write && !*diff && !stdoutSingle)

	var (
		unformatted int
		parseErrors int
	)

	for _, filename := range targets {
		content, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(stderr, "error reading %s: %v\n", filename, err)
			parseErrors++
			continue
		}
		source := string(content)

		formatted, err := format.FormatSource(filename, source)
		if err != nil {
			// Report file:line:col and keep going so one bad file does not
			// abort a whole-tree run. The file is never touched.
			if pe, ok := format.AsParsleyError(err); ok {
				fmt.Fprintf(stderr, "%s:%d:%d: %s\n", filename, pe.Line, pe.Column, pe.Message)
			} else {
				fmt.Fprintf(stderr, "%s: %v\n", filename, err)
			}
			parseErrors++
			continue
		}

		changed := formatted != source
		if changed {
			unformatted++
		}

		switch {
		case listMode:
			if changed {
				fmt.Fprintln(stdout, filename)
			}
		case *diff:
			if changed {
				fmt.Fprint(stdout, format.Diff(filename, source, formatted))
			}
		case *write:
			if changed {
				if err := os.WriteFile(filename, []byte(formatted), 0644); err != nil {
					fmt.Fprintf(stderr, "error writing %s: %v\n", filename, err)
					parseErrors++
				}
			}
		default: // single-file stdout
			fmt.Fprint(stdout, formatted)
		}
	}

	if parseErrors > 0 {
		return fmt.Errorf("%s could not be formatted", plural(parseErrors, "file"))
	}
	// Under list mode (explicit -l or the tree default), an unformatted file is
	// a non-zero exit so the command is usable as a CI gate.
	if listMode && unformatted > 0 {
		return fmt.Errorf("%s not formatted", plural(unformatted, "file"))
	}
	return nil
}

// singleFileArg reports whether paths is exactly one entry naming a regular
// file (not a directory). Used to decide the flag-less default.
func singleFileArg(paths []string) bool {
	if len(paths) != 1 {
		return false
	}
	info, err := os.Stat(paths[0])
	return err == nil && !info.IsDir()
}

// collectFmtTargets expands the command's path arguments into a flat list of
// files to format. Explicit file arguments are included verbatim; directories
// are walked (filepath.WalkDir) for *.pars files, skipping any .git directory.
// No arguments means walk the current directory tree.
func collectFmtTargets(paths []string) ([]string, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	var targets []string
	seen := make(map[string]bool)
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			targets = append(targets, p)
		}
	}

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		if !info.IsDir() {
			add(p)
			continue
		}
		walkErr := filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) == ".pars" {
				add(path)
			}
			return nil
		})
		if walkErr != nil {
			return nil, fmt.Errorf("walking %s: %w", p, walkErr)
		}
	}
	return targets, nil
}

// plural renders "N thing" / "N things".
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func printFmtUsage(w io.Writer) {
	fmt.Fprintf(w, `basil fmt - format Parsley source files

Usage:
  basil fmt [options] [path...]

Paths may be files or directories. Directories are walked for *.pars files
(skipping .git). With no path, the current directory tree is formatted.

Options:
  -w    Write result to source file instead of stdout
  -l    List files whose formatting differs (exits non-zero if any)
  -d    Display diffs instead of rewriting files

With no option, a single file argument is printed formatted to stdout; a
directory (or no argument) defaults to -l.

Examples:
  basil fmt script.pars        Print formatted output to stdout
  basil fmt -w .               Format the whole tree in place
  basil fmt -l                 List unformatted files under cwd (CI gate)
  basil fmt -d src/            Show what would change under src/
`)
}
