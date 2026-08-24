// Package deploy implements the Basil deploy engine (FEAT-153): release
// materialisation, activation, pruning, the deploy record, and the per-site
// lock that serialises deploys.
package deploy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ErrLocked is returned when another deploy holds the site's lock and the
// caller asked not to wait (or the wait ran out).
var ErrLocked = errors.New("another deploy is in progress")

// errLockHeld is the platform-neutral signal from tryLockFile that another
// process holds the lock right now, as opposed to the lock attempt itself
// failing.
var errLockHeld = errors.New("deploy lock held by another process")

// LockFileName is the deploy lock, kept in the site root next to releases/
// and current: the lock guards the site, so it lives with the site, not
// inside any release.
const LockFileName = ".deploy.lock"

// lockPollInterval is how often a waiting AcquireLock retries.
const lockPollInterval = 25 * time.Millisecond

// Lock is an exclusive per-site deploy lock. It is advisory, which is
// enough: every writer to releases/ and current goes through this package.
// The platform mechanics live in tryLockFile/unlockFile (lock_unix.go,
// lock_windows.go).
type Lock struct {
	f *os.File
}

// AcquireLock takes the exclusive deploy lock for a site. With wait=0 it
// refuses immediately with ErrLocked when the lock is held; with wait>0 it
// retries until the deadline and then fails with an error wrapping ErrLocked.
//
// The lock is a kernel file lock, so a deploy that crashes - or is
// kill -9ed - releases it on process exit and can never wedge future
// deploys. The lock file itself is never deleted; only the lock on it
// matters.
func AcquireLock(siteRoot string, wait time.Duration) (*Lock, error) {
	path := filepath.Join(siteRoot, LockFileName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening deploy lock: %w", err)
	}

	deadline := time.Now().Add(wait)
	for {
		err := tryLockFile(f)
		if err == nil {
			return &Lock{f: f}, nil
		}
		if !errors.Is(err, errLockHeld) {
			f.Close()
			return nil, fmt.Errorf("locking %s: %w", path, err)
		}
		if wait <= 0 {
			f.Close()
			return nil, ErrLocked
		}
		if time.Now().After(deadline) {
			f.Close()
			return nil, fmt.Errorf("%w (gave up after %s)", ErrLocked, wait)
		}
		time.Sleep(lockPollInterval)
	}
}

// Release drops the lock. Safe to call more than once.
func (l *Lock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	f := l.f
	l.f = nil
	if err := unlockFile(f); err != nil {
		f.Close()
		return fmt.Errorf("unlocking deploy lock: %w", err)
	}
	return f.Close()
}
