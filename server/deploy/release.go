package deploy

import (
	"archive/tar"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sambeau/basil/server/config"
)

// Materialise extracts commit sha from the repository at repoDir into
// releasesDir/<sha> and returns that directory. The release is byte-identical
// to the commit and contains no .git: `git archive` is streamed straight into
// Go's tar reader, so no external tar binary is involved.
//
// Extraction happens in releasesDir/.tmp-<sha> followed by an atomic rename,
// so a crash mid-extract leaves only a dot-prefixed temp directory (which
// Prune ignores and the next Materialise of the same sha sweeps away), never
// a half-built release under a real release name.
//
// Materialise is idempotent: an existing releasesDir/<sha> is returned as-is
// without re-extracting - a release, being byte-identical to its commit,
// never needs rebuilding.
func Materialise(repoDir, sha, releasesDir string) (string, error) {
	target := filepath.Join(releasesDir, sha)
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		return target, nil
	}

	if err := os.MkdirAll(releasesDir, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", releasesDir, err)
	}

	tmp := filepath.Join(releasesDir, ".tmp-"+sha)
	if err := os.RemoveAll(tmp); err != nil {
		return "", fmt.Errorf("clearing stale %s: %w", tmp, err)
	}
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", tmp, err)
	}

	if err := extractCommit(repoDir, sha, tmp); err != nil {
		os.RemoveAll(tmp)
		return "", err
	}

	if err := os.Rename(tmp, target); err != nil {
		// A concurrent Materialise of the same sha may have won the rename;
		// its result is byte-identical, so use it.
		if info, statErr := os.Stat(target); statErr == nil && info.IsDir() {
			os.RemoveAll(tmp)
			return target, nil
		}
		os.RemoveAll(tmp)
		return "", fmt.Errorf("activating extracted release: %w", err)
	}
	return target, nil
}

// extractCommit streams `git archive` for sha into dest.
func extractCommit(repoDir, sha, dest string) error {
	cmd := exec.Command("git", "archive", "--format=tar", sha)
	cmd.Dir = repoDir
	// Same env hygiene as cmd/basil runGit: no system config surprises, and
	// git must fail rather than prompt when run from a server.
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("git archive %s: %w", sha, err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("git archive %s: %w", sha, err)
	}

	extractErr := extractTar(stdout, dest)
	if extractErr != nil {
		// Drain so git can exit, then surface the extraction error.
		io.Copy(io.Discard, stdout)
	}
	waitErr := cmd.Wait()

	if waitErr != nil {
		return fmt.Errorf("git archive %s: %v: %s", sha, waitErr, strings.TrimSpace(stderr.String()))
	}
	return extractErr
}

// extractTar unpacks a tar stream into dest, preserving file modes, dirs and
// symlinks as git archive emitted them. Any entry whose cleaned path would
// escape dest - path traversal - is rejected outright: the archive normally
// comes from the site's own repository, but the extractor must not trust it.
func extractTar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading archive: %w", err)
		}

		name := filepath.FromSlash(hdr.Name)
		if !filepath.IsLocal(name) {
			return fmt.Errorf("refusing archive entry %q: path escapes the release directory", hdr.Name)
		}
		target := filepath.Join(dest, name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, hdr.FileInfo().Mode().Perm()); err != nil {
				return fmt.Errorf("extracting %s: %w", hdr.Name, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("extracting %s: %w", hdr.Name, err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode().Perm())
			if err != nil {
				return fmt.Errorf("extracting %s: %w", hdr.Name, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("extracting %s: %w", hdr.Name, err)
			}
			if err := f.Close(); err != nil {
				return fmt.Errorf("extracting %s: %w", hdr.Name, err)
			}
		case tar.TypeSymlink:
			// The link target is preserved verbatim - a release is
			// byte-identical to its commit, dangling or absolute links
			// included. Only entry paths are traversal-checked; the server
			// never follows links out of the release on behalf of a request.
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("extracting %s: %w", hdr.Name, err)
			}
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return fmt.Errorf("extracting %s: %w", hdr.Name, err)
			}
		case tar.TypeXGlobalHeader:
			// git archive stamps the commit id in a pax global header.
		default:
			return fmt.Errorf("refusing archive entry %q: unsupported type %q", hdr.Name, hdr.Typeflag)
		}
	}
}

// SetCurrent atomically re-points <siteRoot>/current at releaseDir, which
// must be a directory under <siteRoot>/releases. The link target is written
// RELATIVE (releases/<sha>), matching what `basil --init` creates, so the
// site root can be moved or mounted elsewhere without breaking the link.
//
// The swap is symlink-then-rename: a reader sees either the old target or
// the new one, never a missing or half-written link.
func SetCurrent(siteRoot, releaseDir string) error {
	rel := filepath.Join(config.ReleasesDirName, filepath.Base(releaseDir))
	if info, err := os.Stat(filepath.Join(siteRoot, rel)); err != nil || !info.IsDir() {
		return fmt.Errorf("cannot activate %s: %s is not a release directory under %s", releaseDir, rel, siteRoot)
	}

	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		return fmt.Errorf("activating release: %w", err)
	}
	tmp := filepath.Join(siteRoot, config.CurrentLinkName+".tmp-"+hex.EncodeToString(suffix))
	if err := os.Symlink(rel, tmp); err != nil {
		return fmt.Errorf("activating release: %w", err)
	}
	if err := os.Rename(tmp, filepath.Join(siteRoot, config.CurrentLinkName)); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("activating release: %w", err)
	}
	return nil
}

// CurrentRelease returns the absolute path of the active release, read
// through <siteRoot>/current.
func CurrentRelease(siteRoot string) (string, error) {
	link := filepath.Join(siteRoot, config.CurrentLinkName)
	target, err := os.Readlink(link)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", link, err)
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(siteRoot, target)
	}
	return filepath.Clean(target), nil
}

// Prune removes the oldest release directories (by directory mtime) beyond
// keep, and returns the paths it removed. The active release is never
// removed, even when it is old enough to qualify; dot-prefixed entries
// (in-flight .tmp-<sha> extractions) and plain files are never touched.
//
// keep <= 0 is treated as keep-everything, not delete-everything: a zero
// that reached this far is far more likely to be an unset value than a
// request to erase every release on disk.
func Prune(releasesDir string, keep int, activeDir string) ([]string, error) {
	if keep <= 0 {
		return nil, nil
	}

	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", releasesDir, err)
	}

	type release struct {
		path  string
		mtime int64
	}
	var releases []release
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", filepath.Join(releasesDir, entry.Name()), err)
		}
		releases = append(releases, release{
			path:  filepath.Join(releasesDir, entry.Name()),
			mtime: info.ModTime().UnixNano(),
		})
	}

	sort.Slice(releases, func(i, j int) bool { return releases[i].mtime > releases[j].mtime })

	active := filepath.Clean(activeDir)
	var removed []string
	for _, r := range releases[min(keep, len(releases)):] {
		if filepath.Clean(r.path) == active {
			continue
		}
		if err := os.RemoveAll(r.path); err != nil {
			return removed, fmt.Errorf("pruning %s: %w", r.path, err)
		}
		removed = append(removed, r.path)
	}
	return removed, nil
}
