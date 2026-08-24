//go:build windows

package deploy

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// tryLockFile takes a non-blocking exclusive LockFileEx lock on the first
// byte of f, returning errLockHeld when another process holds it. Like the
// unix flock, the lock is released by the kernel when the handle is closed
// or the process exits, so a crashed deploy cannot wedge future deploys.
//
// What this path does NOT guarantee: LockFileEx is a byte-range lock, not a
// whole-file advisory lock - a process that never calls AcquireLock is not
// excluded from the site root in any way, and unlike flock the locked byte
// range is mandatory, so an unrelated reader of the lock file's first byte
// would be refused while a deploy runs. Neither difference matters for the
// lock's actual job: serialising cooperating deploy processes, which all
// come through AcquireLock.
func tryLockFile(f *os.File) error {
	ol := new(windows.Overlapped)
	err := windows.LockFileEx(windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, ol)
	if err == nil {
		return nil
	}
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return errLockHeld
	}
	return err
}

// unlockFile drops the byte-range lock.
func unlockFile(f *os.File) error {
	ol := new(windows.Overlapped)
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, ol)
}
