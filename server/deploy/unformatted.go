package deploy

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sambeau/basil/pkg/parsley/format"
)

// Unformatted reports the release-relative paths of every *.pars file whose
// source differs from its canonical `basil fmt` form. It is deliberately NOT
// part of Validate: formatting is style, not correctness, and the server never
// rewrites code (DESIGN-git-deploy D4b) — it only reports. So a caller uses
// this to WARN, never to reject, and the warning stays a separate non-fatal
// channel from Validate's rejecting []ValidationError.
//
// It reuses Validate's walk shape (skip directories, *.pars only, release
// root failure is fatal to the walk) but returns filenames instead of errors.
// A file that will not parse is NOT a formatting problem: Validate already
// rejected such a release before this runs, so in practice IsFormatted never
// sees one — but as a guard, a file whose formatting cannot be judged (read
// error or parse error) is skipped silently rather than warned about. The
// result is nil when everything is already formatted.
func Unformatted(releaseDir string) []string {
	var unformatted []string

	_ = filepath.WalkDir(releaseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// A walk error is not a formatting judgement: skip it. If the
			// release root itself is unreadable there is simply nothing to
			// report (Validate is the gate that turns that into a rejection).
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".pars") {
			return nil
		}

		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil // unreadable: not a formatting warning
		}
		rel := relOrSelf(releaseDir, path)
		formatted, fmtErr := format.IsFormatted(rel, string(src))
		if fmtErr != nil {
			return nil // a parse error is not a formatting warning; skip it
		}
		if !formatted {
			unformatted = append(unformatted, rel)
		}
		return nil
	})

	return unformatted
}
