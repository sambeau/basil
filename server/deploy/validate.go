package deploy

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/lexer"
	"github.com/sambeau/basil/pkg/parsley/parser"
	"github.com/sambeau/basil/server/config"
)

// ValidationError is one reason a release cannot go live: a Parsley file
// that does not parse, or a config that does not load or validate. File is
// release-relative so the error reads the same wherever the site root lives.
type ValidationError struct {
	File    string // Release-relative path ("site/index.pars", "basil.yaml")
	Line    int    // 1-based, 0 when the error has no line (config errors)
	Column  int    // 1-based, 0 when unknown
	Message string
}

// String renders "file:line:col: message", dropping the parts that are
// unknown, so validation output is grep-able and diffable.
func (e ValidationError) String() string {
	var sb strings.Builder
	sb.WriteString(e.File)
	if e.Line > 0 {
		fmt.Fprintf(&sb, ":%d", e.Line)
		if e.Column > 0 {
			fmt.Fprintf(&sb, ":%d", e.Column)
		}
	}
	sb.WriteString(": ")
	sb.WriteString(e.Message)
	return sb.String()
}

// joinValidationErrors flattens a validation failure into one line per
// error, for the deploy record's reason column.
func joinValidationErrors(errs []ValidationError) string {
	parts := make([]string, len(errs))
	for i, e := range errs {
		parts[i] = e.String()
	}
	return strings.Join(parts, "; ")
}

// Validate is the gate between materialise and activate: it parses every
// .pars file in the release and loads the release's config, collecting ALL
// errors rather than stopping at the first — a rejected deploy should name
// everything wrong, not make the developer fix-push-fix-push.
//
// Validation is parse and config-load only (correctness, not style), and it
// walks the whole release: the server discovers handlers lazily anywhere
// under the site, and deploy.pars lives in the release root, so no directory
// is exempt. A nil result means the release may go live.
func Validate(releaseDir string) []ValidationError {
	var errs []ValidationError

	walkErr := filepath.WalkDir(releaseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// The release root itself failing means NOTHING was validated:
			// propagate it as a real walk error rather than one mislabelled
			// per-file entry. Errors below the root are collected and the
			// walk continues, so a rejection names everything wrong.
			if path == releaseDir {
				return err
			}
			errs = append(errs, ValidationError{
				File:    relOrSelf(releaseDir, path),
				Message: err.Error(),
			})
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".pars") {
			return nil
		}
		errs = append(errs, parseFile(releaseDir, path)...)
		return nil
	})
	if walkErr != nil {
		errs = append(errs, ValidationError{
			File:    releaseDir,
			Message: fmt.Sprintf("walking the release: %v", walkErr),
		})
	}

	errs = append(errs, validateConfig(releaseDir)...)
	return errs
}

// parseFile runs one .pars file through the same lexer/parser recipe the
// server uses to load it (server/handler.go parseScript), returning every
// structured error the parser found.
func parseFile(releaseDir, path string) []ValidationError {
	rel := relOrSelf(releaseDir, path)

	src, err := os.ReadFile(path)
	if err != nil {
		// Unreadable is unservable: a dangling symlink or permission hole
		// would surface as a 500 at request time, so refuse it now.
		return []ValidationError{{File: rel, Message: err.Error()}}
	}

	l := lexer.NewWithFilename(string(src), rel)
	p := parser.New(l)
	p.ParseProgram()

	var errs []ValidationError
	for _, perr := range p.StructuredErrors() {
		errs = append(errs, ValidationError{
			File:    rel,
			Line:    perr.Line,
			Column:  perr.Column,
			Message: perr.Message,
		})
	}
	return errs
}

// validateConfig loads and validates the release's basil.yaml exactly as
// the server will at startup (config.Load + config.Validate, the
// verifyGeneratedConfig recipe). A release whose config cannot load would
// take the site down on the next restart, so it is refused now.
func validateConfig(releaseDir string) []ValidationError {
	path := filepath.Join(releaseDir, config.ConfigFileName)
	cfg, err := config.Load(path, os.Getenv)
	if err != nil {
		return []ValidationError{{File: config.ConfigFileName, Message: err.Error()}}
	}
	if err := config.Validate(cfg); err != nil {
		return []ValidationError{{File: config.ConfigFileName, Message: err.Error()}}
	}
	return nil
}

// relOrSelf returns path relative to base, or path unchanged when it does
// not sit under base.
func relOrSelf(base, path string) string {
	if rel, err := filepath.Rel(base, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}
